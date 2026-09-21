package svc

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"bilibili-web/server/app/user/api/internal/config"
	"bilibili-web/server/app/user/model"
	"bilibili-web/server/common/auth"
	"bilibili-web/server/common/errcode"
	"bilibili-web/server/common/health"
	"bilibili-web/server/common/middleware"
	"bilibili-web/server/common/response"
	"bilibili-web/server/common/session"
	"bilibili-web/server/common/store"
	"bilibili-web/server/common/token"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// UserRepository 是 logic 层实际需要的用户数据访问能力。
//
// 刻意不直接用 model.UsersModel：后者含未导出的 withSession 方法，
// 接口无法在本包之外实现，测试也就无法注入替身
// （quality/testing 要求逻辑以外依赖以接口注入）。
// 按「调用方定义自己需要的接口」收敛到下面四个方法，
// 使 logic 层的测试不必连数据库，也让 logic 对 model 的依赖面显式可审。
type UserRepository interface {
	Insert(ctx context.Context, data *model.Users) (sql.Result, error)
	FindOne(ctx context.Context, id int64) (*model.Users, error)
	FindOneByUsername(ctx context.Context, username string) (*model.Users, error)
	UpdateProfile(ctx context.Context, id int64, in model.ProfileUpdate) error
}

// SessionStore 是会话版本号的读写接口。
//
// 在 svc 层以接口而非具体类型暴露，是为了让 logic 层的测试能注入确定性实现
// （quality/testing 要求逻辑以外依赖以接口注入），
// 而不必为了跑单元测试拉起一个真实 Redis。
type SessionStore interface {
	Current(ctx context.Context, userID int64) (int64, error)
	Bump(ctx context.Context, userID int64) (int64, error)
	Matches(ctx context.Context, userID, ver int64) (bool, error)
}

// TokenIssuer 是凭证签发与解析接口。
type TokenIssuer interface {
	Issue(userID, ver int64, now time.Time) (string, error)
	Parse(raw string) (*token.Claims, error)
	TTL() time.Duration
}

// 编译期断言：真实实现满足上述接口。
var (
	_ UserRepository = model.UsersModel(nil)
	_ SessionStore   = (*session.Store)(nil)
	_ TokenIssuer    = (*token.Issuer)(nil)
)

// ServiceContext 持有本域服务运行期依赖。
type ServiceContext struct {
	Config config.Config
	Domain string

	// Mysql 仅用户域持有：本变更是项目首个落库功能。
	Mysql sqlx.SqlConn
	// Redis 用于登录态会话版本号，五域都要。
	Redis *redis.Client

	// Users 是 users 表的数据访问入口。
	Users UserRepository
	// Sessions 读写会话版本号，用于登录态撤销。
	Sessions SessionStore
	// Tokens 签发与解析登录凭证。
	Tokens TokenIssuer
	// Revocation 供 middleware.Auth 判定凭证是否已撤销。
	Revocation middleware.RevocationChecker
	// AuthReady 表示鉴权依赖是否齐备。
	//
	// 密钥缺失时不阻止服务启动（与中间件暂时不可用同理），
	// 但受保护接口必须返回依赖不可用而非放行——由 AuthReady 决定。
	AuthReady bool
}

// NewServiceContext 建立本域所需的连接。
//
// 刻意使用不校验连通性的构造方式：中间件暂时不可用不应阻止服务启动，
// 连通状态交由 health 接口实时反映（骨架决策 5）。
func NewServiceContext(c config.Config) *ServiceContext {
	svcCtx := &ServiceContext{Config: c, Domain: c.Domain}
	if svcCtx.Domain == "" {
		svcCtx.Domain = "user"
	}

	mysqlConn, err := store.NewMysqlConn(c.Mysql.SqlConf())
	if err != nil {
		logx.Errorf("[user] 创建 mysql 连接失败: %v", err)
	}
	svcCtx.Mysql = mysqlConn

	// 连接可用时才建立 model；连接为 nil 时保持 model 为 nil，
	// 由 logic 层识别并返回依赖不可用，而不是在此 panic。
	if svcCtx.Mysql != nil {
		svcCtx.Users = model.NewUsersModel(svcCtx.Mysql)
	}

	redisClient, err := store.NewRedisClient(c.Redis)
	if err != nil {
		logx.Errorf("[user] 创建 redis 客户端失败: %v", err)
	}
	svcCtx.Redis = redisClient

	// 会话键 TTL 取凭证有效期的 2 倍，因此先把有效期定下来。
	ttl := c.Auth.TTL()
	svcCtx.Sessions = session.NewStore(svcCtx.Redis, int(ttl.Seconds()))
	svcCtx.Revocation = middleware.RedisChecker{Store: svcCtx.Sessions}

	// 签名密钥缺失是不可恢复的配置错误：此时不签发也不校验任何凭证。
	// 调用方（handler）据此返回 50001，绝不降级为「跳过鉴权」。
	issuer, err := token.NewIssuer(c.Auth.JwtSecret, ttl, auth.MinSecretLen())
	if err != nil {
		logx.Errorf("[user] 登录凭证签发器不可用: %v", err)
	} else {
		svcCtx.Tokens = issuer
		svcCtx.AuthReady = true
	}

	return svcCtx
}

// Connections 暴露 health 检查所需的真实连接。
func (s *ServiceContext) Connections() health.Deps {
	return health.Deps{Mysql: s.Mysql, Redis: s.Redis}
}

// Protected 返回挂载了真实鉴权中间件的 handler 包装器。
//
// 当签发器不可用（密钥未配置）时，包装器直接拒绝请求：
// 鉴权依赖缺失的失败方向必须是拒绝，不能退化成直通。
func (s *ServiceContext) Protected(next http.HandlerFunc) http.HandlerFunc {
	if !s.AuthReady {
		return func(w http.ResponseWriter, r *http.Request) {
			logx.Error("[user] 鉴权依赖不可用（JWT_SECRET 未配置），拒绝受保护请求")
			response.WriteFail(w, http.StatusServiceUnavailable, errcode.ErrDependency, "")
		}
	}

	return middleware.Auth(s.Tokens, s.Revocation)(next)
}

// Close 释放本域持有的连接，应在进程退出时调用。
func (s *ServiceContext) Close() {
	if err := store.CloseMysql(s.Mysql); err != nil {
		logx.Errorf("[user] 关闭 mysql 连接失败: %v", err)
	}
	if err := store.CloseRedis(s.Redis); err != nil {
		logx.Errorf("[user] 关闭 redis 连接失败: %v", err)
	}
}
