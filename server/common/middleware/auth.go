package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"bilibili-web/server/common/errcode"
	"bilibili-web/server/common/response"
	"bilibili-web/server/common/session"
	"bilibili-web/server/common/token"

	"github.com/zeromicro/go-zero/core/logx"
)

// bearerPrefix 是凭证在请求头中的固定前缀（大小写不敏感地匹配）。
const bearerPrefix = "bearer "

// Identity 是校验通过的请求身份，由 Auth 注入到请求上下文中。
type Identity struct {
	UserID int64
}

// ctxKey 是本包私有的上下文键类型。
//
// 刻意不用字符串作键：字符串键存在跨包碰撞的风险，
// 自定义类型可以保证只有本包能写入与读取该值。
type ctxKey struct{}

var identityKey ctxKey

// RevocationChecker 判定某个会话版本是否仍然有效。
//
// 抽成接口而非直接依赖 session.Store，是为了让鉴权中间件不绑定具体存储，
// 同时让测试可以注入确定性的实现（design 决策 3）。
type RevocationChecker interface {
	Matches(ctx context.Context, userID, ver int64) (bool, error)
}

// TokenParser 解析凭证并返回其载荷。
type TokenParser interface {
	Parse(raw string) (*token.Claims, error)
}

// Auth 校验登录态，失败时直接写出统一的未认证响应。
//
// 校验顺序（任一环节失败都不继续执行后续逻辑）：
//  1. 请求头存在且格式为 `Bearer <token>`
//  2. 本地验签与有效期校验（纯计算，不访问外部依赖）
//  3. 会话版本号比对，判断是否已被撤销（一次 Redis 读取）
//
// 把本地验签放在前面是有意的：绝大多数无效请求止步于此，
// 不必为它们访问 Redis。
//
// nil 的 parser 或 checker 一律视为「校验不可能通过」并返回未认证，
// 而不是放行——鉴权中间件的失败方向必须是拒绝。
func Auth(parser TokenParser, checker RevocationChecker) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if parser == nil || checker == nil {
				logx.Error("[middleware] Auth 依赖未注入，拒绝请求")
				response.WriteFail(w, http.StatusUnauthorized, errcode.ErrUnauthorized, "")
				return
			}

			raw, ok := bearerToken(r)
			if !ok {
				response.WriteFail(w, http.StatusUnauthorized, errcode.ErrUnauthorized, "")
				return
			}

			claims, err := parser.Parse(raw)
			if err != nil {
				// 签名错误、已过期、格式非法在这里被统一吸收为未认证，
				// 不向外区分具体原因（identity/auth「凭证过期统一处理」）。
				response.WriteFail(w, http.StatusUnauthorized, errcode.ErrUnauthorized, "")
				return
			}

			matched, err := checker.Matches(r.Context(), claims.UserID, claims.Version)
			if err != nil {
				// 会话存储故障与「凭证已撤销」必须区分：前者是依赖不可用，
				// 让客户端重试是有意义的；后者重试永远不会成功。
				logx.Errorf("[middleware] 校验会话版本失败 uid=%d: %v", claims.UserID, err)
				response.WriteFail(w, http.StatusServiceUnavailable, errcode.ErrDependency, "")
				return
			}
			if !matched {
				response.WriteFail(w, http.StatusUnauthorized, errcode.ErrUnauthorized, "")
				return
			}

			next(w, r.WithContext(WithIdentity(r.Context(), Identity{UserID: claims.UserID})))
		}
	}
}

// bearerToken 从 Authorization 头中取出凭证主体。
func bearerToken(r *http.Request) (string, bool) {
	header := r.Header.Get("Authorization")
	if len(header) <= len(bearerPrefix) {
		return "", false
	}
	if !strings.EqualFold(header[:len(bearerPrefix)], bearerPrefix) {
		return "", false
	}

	token := strings.TrimSpace(header[len(bearerPrefix):])
	if token == "" {
		return "", false
	}

	return token, true
}

// WithIdentity 把请求身份写入上下文，供 logic 层读取。
func WithIdentity(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, identityKey, id)
}

// IdentityFrom 读取上下文中的请求身份。
//
// 第二个返回值为 false 表示该请求未经 Auth 校验。受保护接口的 logic 层
// 必须检查它，不得假设中间件一定挂载过。
func IdentityFrom(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(identityKey).(Identity)
	return id, ok
}

// MustIdentity 读取请求身份，缺失时返回一个明确的错误。
//
// 供 logic 层在参数校验阶段使用：缺身份属于编程错误（路由漏挂中间件），
// 返回错误比返回零值 ID 更安全——零值会让查询落到一个不存在的用户上。
func MustIdentity(ctx context.Context) (Identity, error) {
	id, ok := IdentityFrom(ctx)
	if !ok || id.UserID <= 0 {
		return Identity{}, errors.New("middleware: 请求上下文缺少有效身份")
	}

	return id, nil
}

// RedisChecker 是基于会话存储的撤销检查器实现。
//
// Store 声明为接口而非 *session.Store：svc 层已把会话存储抽象为接口
// 以便注入测试替身，这里保持同一形状可避免多一次类型断言。
type RedisChecker struct {
	Store interface {
		Matches(ctx context.Context, userID, ver int64) (bool, error)
	}
}

// Matches 委托给会话存储。
func (c RedisChecker) Matches(ctx context.Context, userID, ver int64) (bool, error) {
	if c.Store == nil {
		return false, session.ErrUnavailable
	}

	return c.Store.Matches(ctx, userID, ver)
}
