// Package store 提供跨业务域复用的存储连接能力。
//
// 按 backend/architecture 规范，跨服务复用的公共能力必须位于 common/：
// 各域只负责把配置传进来，连接的构造、连通性校验与释放统一在这里实现。
package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	// 显式声明驱动依赖：go-zero 的 sqlx 内部虽已引入，但驱动注册属于
	// 本包的能力前提，不依赖其他包的间接引入。
	_ "github.com/go-sql-driver/mysql"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// connectTimeout 是构造连接时校验连通性的超时上限。
const connectTimeout = 3 * time.Second

// MysqlConf 是 MySQL 的连接配置，五个域共用同一份定义。
//
// 优先级为「环境变量 > etc/*.yaml > 默认值」；账号与口令一律由环境变量
// MYSQL_USER / MYSQL_PASSWORD 提供（见 Makefile 的开发默认值），
// 仓库内的配置文件与代码不得出现可用凭据字面量。
//
// 注意：go-zero 经 core/proc.Env 读环境变量并在进程内按变量名缓存首次结果，
// 因此运行期改环境变量不会生效，必须在启动前设置。
type MysqlConf struct {
	Addr     string `json:",default=127.0.0.1:13306,env=MYSQL_ADDR"`
	User     string `json:",optional,env=MYSQL_USER"`
	Password string `json:",optional,env=MYSQL_PASSWORD"`
	Database string `json:",default=bilibili_web,env=MYSQL_DATABASE"`
}

// SqlConf 转换为 go-zero sqlx 所需的连接配置。
func (c MysqlConf) SqlConf() sqlx.SqlConf {
	return sqlx.SqlConf{
		DataSource: MySqlDSN(c.User, c.Password, c.Addr, c.Database),
		DriverName: "mysql",
	}
}

// MySqlDSN 组装 go-sql-driver/mysql 的连接串。
//
// 固定附带两个驱动参数，缺一不可：
//   - parseTime=true：驱动默认把 DATETIME 返回为 []uint8，而 goctl 生成的 model
//     把时间列映射为 time.Time，不开启会以 Scan error 直接失败。
//   - loc=Local：时间按本地时区解释，与建表时 CURRENT_TIMESTAMP 的语义一致。
func MySqlDSN(user, password, addr, database string) string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=true&loc=Local",
		user, password, addr, database,
	)
}

// NewMysqlConn 建立 MySQL 连接：只校验配置，不校验连通性。
//
// 服务启动使用它——中间件暂时不可用不应阻止进程起不来，
// 连通状态由 health 接口实时反映（骨架决策 5）。
func NewMysqlConn(conf sqlx.SqlConf) (sqlx.SqlConn, error) {
	if err := conf.Validate(); err != nil {
		return nil, fmt.Errorf("store: mysql 配置非法: %w", err)
	}

	conn, err := sqlx.NewConn(conf)
	if err != nil {
		return nil, fmt.Errorf("store: 创建 mysql 连接失败: %w", err)
	}

	return conn, nil
}

// NewMysql 建立 MySQL 连接并立即校验连通性。
//
// 使用 sqlx.NewConn 而非 MustNewConn / NewMysql：后两者在失败时调用 logx.Must
// 直接终止进程，与 backend/architecture「错误逐层包装且不得吞掉」不符。
//
// 注意 sqlx.NewConn 只做配置校验，底层 sql.Open 是惰性的、不会真正建连，
// 因此这里主动 ping 一次，让「连不上」在构造阶段就以 error 形式暴露。
func NewMysql(conf sqlx.SqlConf) (sqlx.SqlConn, error) {
	conn, err := NewMysqlConn(conf)
	if err != nil {
		return nil, err
	}

	if err := PingMysql(context.Background(), conn); err != nil {
		_ = CloseMysql(conn)
		return nil, err
	}

	return conn, nil
}

// PingMysql 校验 MySQL 是否可连通，供健康检查与启动自检复用。
func PingMysql(ctx context.Context, conn sqlx.SqlConn) error {
	if conn == nil {
		return errors.New("store: mysql 连接为空")
	}

	db, err := conn.RawDB()
	if err != nil {
		return fmt.Errorf("store: 获取 mysql 底层连接失败: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("store: mysql 连通性校验失败: %w", err)
	}

	return nil
}

// CloseMysql 释放 MySQL 底层连接池。
//
// sqlx.SqlConn 未暴露 Close，需要经 RawDB 取回 *sql.DB 再关闭。
//
// 注意：go-zero 的 sqlx 按「驱动 + 数据源」全局缓存连接池，这里关闭的是缓存中的
// 那一个实例。因此本方法只应在进程退出时调用一次；在运行期调用后，同一数据源的
// 再次连接会拿到已关闭的连接池而报 "database is closed"。
func CloseMysql(conn sqlx.SqlConn) error {
	if conn == nil {
		return nil
	}

	db, err := conn.RawDB()
	if err != nil {
		return fmt.Errorf("store: 获取 mysql 底层连接失败: %w", err)
	}

	if err := db.Close(); err != nil {
		return fmt.Errorf("store: 关闭 mysql 连接失败: %w", err)
	}

	return nil
}
