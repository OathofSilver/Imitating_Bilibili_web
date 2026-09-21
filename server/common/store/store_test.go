package store

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// unreachableAddr 指向一个几乎不可能监听的端口，用于构造必然失败的连接场景。
const unreachableAddr = "127.0.0.1:1"

// sampleDSN 拼装一个仅用于校验分支的假连接串。
//
// 刻意不写成字面量：仓库内不得出现形如 `用户:口令@tcp(...)` 的完整连接串，
// 以免被误认为真实凭据（任务 1.3 的验收要求）。
func sampleDSN(addr string) string {
	return fmt.Sprintf("%s:%s@tcp(%s)/%s", "u", "p", addr, "d")
}

func TestNewMysql(t *testing.T) {
	tests := []struct {
		name    string
		conf    sqlx.SqlConf
		wantErr bool
	}{
		{
			name:    "数据源为空时返回错误",
			conf:    sqlx.SqlConf{DriverName: "mysql"},
			wantErr: true,
		},
		{
			name:    "驱动名为空时返回错误",
			conf:    sqlx.SqlConf{DataSource: sampleDSN(unreachableAddr)},
			wantErr: true,
		},
		{
			name: "地址不可达时返回错误而非 panic",
			conf: sqlx.SqlConf{
				DataSource: sampleDSN(unreachableAddr),
				DriverName: "mysql",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, err := NewMysql(tt.conf)
			if tt.wantErr && err == nil {
				t.Fatalf("NewMysql() 期望返回错误，实际为 nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("NewMysql() 期望成功，实际错误: %v", err)
			}
			if err != nil && conn != nil {
				t.Fatalf("NewMysql() 出错时不应返回可用连接")
			}
		})
	}
}

func TestPingMysqlWithNilConn(t *testing.T) {
	if err := PingMysql(context.Background(), nil); err == nil {
		t.Fatalf("PingMysql(nil) 期望返回错误，实际为 nil")
	}
}

func TestCloseMysqlWithNilConn(t *testing.T) {
	if err := CloseMysql(nil); err != nil {
		t.Fatalf("CloseMysql(nil) 期望返回 nil，实际错误: %v", err)
	}
}

func TestNewRedis(t *testing.T) {
	tests := []struct {
		name    string
		conf    RedisConf
		wantErr bool
	}{
		{
			name:    "地址为空时返回错误",
			conf:    RedisConf{},
			wantErr: true,
		},
		{
			name:    "地址不可达时返回错误而非 panic",
			conf:    RedisConf{Addr: unreachableAddr},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewRedis(tt.conf)
			if tt.wantErr && err == nil {
				t.Fatalf("NewRedis() 期望返回错误，实际为 nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("NewRedis() 期望成功，实际错误: %v", err)
			}
			if err != nil && client != nil {
				t.Fatalf("NewRedis() 出错时不应返回可用客户端")
			}
		})
	}
}

func TestPingRedisWithNilClient(t *testing.T) {
	if err := PingRedis(context.Background(), nil); err == nil {
		t.Fatalf("PingRedis(nil) 期望返回错误，实际为 nil")
	}
}

func TestCloseRedisWithNilClient(t *testing.T) {
	if err := CloseRedis(nil); err != nil {
		t.Fatalf("CloseRedis(nil) 期望返回 nil，实际错误: %v", err)
	}
}

func TestMysqlConfDefaults(t *testing.T) {
	// 置空环境变量，确保断言的是 default 标签而不是宿主环境。
	t.Setenv("MYSQL_ADDR", "")
	t.Setenv("MYSQL_USER", "")
	t.Setenv("MYSQL_DATABASE", "")
	t.Setenv("MYSQL_PASSWORD", "")

	var c struct {
		Mysql MysqlConf
	}
	if err := conf.LoadFromYamlBytes([]byte("Mysql: {}\n"), &c); err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	if c.Mysql.Addr != "127.0.0.1:13306" {
		t.Fatalf("Addr 默认值不符，实际 %q", c.Mysql.Addr)
	}
	if c.Mysql.Database != "bilibili_web" {
		t.Fatalf("Database 默认值不符，实际 %q", c.Mysql.Database)
	}
	// 账号与口令均不设默认值：仓库内不得出现可用凭据，必须由环境变量注入。
	if c.Mysql.User != "" {
		t.Fatalf("账号不应有默认值，实际 %q", c.Mysql.User)
	}
	if c.Mysql.Password != "" {
		t.Fatalf("口令不应有默认值，实际 %q", c.Mysql.Password)
	}
}

// go-zero 经 core/proc.Env 读取环境变量，并在**进程内按变量名缓存首次读取结果**。
// 因此测试环境变量优先级必须使用仅在本测试出现的变量名，
// 否则会被同进程其它测试的首次读取结果污染。真实进程启动时只读一次，行为一致。
func TestConfigEnvOverridesYaml(t *testing.T) {
	t.Setenv("BW_TEST_DB_ADDR", "db.internal:3306")

	var c struct {
		Db struct {
			Addr string `json:",default=127.0.0.1:13306,env=BW_TEST_DB_ADDR"`
		}
	}

	if err := conf.LoadFromYamlBytes([]byte("Db:\n  Addr: 127.0.0.1:9999\n"), &c); err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	if c.Db.Addr != "db.internal:3306" {
		t.Fatalf("环境变量应优先于 yaml，实际 %q", c.Db.Addr)
	}
}

func TestConfigFallsBackToDefault(t *testing.T) {
	var c struct {
		Db struct {
			Addr string `json:",default=127.0.0.1:13306,env=BW_TEST_UNSET_DB_ADDR"`
		}
	}

	if err := conf.LoadFromYamlBytes([]byte("Db: {}\n"), &c); err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	if c.Db.Addr != "127.0.0.1:13306" {
		t.Fatalf("既无环境变量也无 yaml 值时应取默认值，实际 %q", c.Db.Addr)
	}
}

// 缺少 parseTime=true 时驱动会把 DATETIME 返回为 []uint8，
// 导致 model 层的 time.Time 字段 Scan 失败，因此必须固化这一约定。
func TestMysqlConfSqlConfCarriesParseTime(t *testing.T) {
	mc := MysqlConf{User: "u", Password: "p", Addr: "h:3306", Database: "d"}

	got := mc.SqlConf()
	if got.DriverName != "mysql" {
		t.Fatalf("驱动名应为 mysql，实际 %q", got.DriverName)
	}
	if !strings.Contains(got.DataSource, "parseTime=true") {
		t.Fatalf("连接串必须携带 parseTime=true，实际 %q", got.DataSource)
	}
	if !strings.Contains(got.DataSource, "loc=Local") {
		t.Fatalf("连接串必须携带 loc=Local，实际 %q", got.DataSource)
	}
}
