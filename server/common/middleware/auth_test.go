package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bilibili-web/server/common/errcode"
	"bilibili-web/server/common/response"
	"bilibili-web/server/common/session"
	"bilibili-web/server/common/token"
)

const (
	testSecret = "test-secret-at-least-16-chars"
	testTTL    = time.Hour
	minSecret  = 16
)

// fakeChecker 是确定性的撤销检查器：只有 recorded 版本被视为有效。
type fakeChecker struct {
	recorded  int64
	err       error
	callsSeen int
}

func (c *fakeChecker) Matches(_ context.Context, _, ver int64) (bool, error) {
	c.callsSeen++
	if c.err != nil {
		return false, c.err
	}

	return ver == c.recorded, nil
}

func newTestIssuer(t *testing.T) *token.Issuer {
	t.Helper()

	issuer, err := token.NewIssuer(testSecret, testTTL, minSecret)
	if err != nil {
		t.Fatalf("创建签发器失败: %v", err)
	}

	return issuer
}

// callAuth 用一个请求驱动中间件，返回响应与「业务处理是否被放行」。
func callAuth(t *testing.T, parser TokenParser, checker RevocationChecker, header string) (*httptest.ResponseRecorder, bool, context.Context) {
	t.Helper()

	var (
		reached bool
		gotCtx  context.Context
	)

	handler := Auth(parser, checker)(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		gotCtx = r.Context()
		response.WriteOK(w, nil)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/me", nil)
	if header != "" {
		req.Header.Set("Authorization", header)
	}

	rec := httptest.NewRecorder()
	handler(rec, req)

	return rec, reached, gotCtx
}

func decodeCode(t *testing.T, rec *httptest.ResponseRecorder) int {
	t.Helper()

	var body response.Body
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("解析响应失败: %v，原文 %q", err, rec.Body.String())
	}

	return body.Code
}

// 任务 2.2 的五类失败 + 一类成功。
func TestAuth(t *testing.T) {
	issuer := newTestIssuer(t)
	now := time.Now()

	const uid = int64(88888)

	validToken, err := issuer.Issue(uid, 5, now)
	if err != nil {
		t.Fatalf("签发失败: %v", err)
	}

	expiredToken, err := issuer.Issue(uid, 5, now.Add(-2*testTTL))
	if err != nil {
		t.Fatalf("签发过期凭证失败: %v", err)
	}

	// 用另一把密钥签发的凭证，模拟签名被篡改。
	otherIssuer, err := token.NewIssuer("other-secret-value-16", testTTL, minSecret)
	if err != nil {
		t.Fatalf("创建另一个签发器失败: %v", err)
	}
	forgedToken, err := otherIssuer.Issue(uid, 5, now)
	if err != nil {
		t.Fatalf("签发伪造凭证失败: %v", err)
	}

	// 版本 4 而非 5：模拟凭证已被撤销。
	staleToken, err := issuer.Issue(uid, 4, now)
	if err != nil {
		t.Fatalf("签发旧版本凭证失败: %v", err)
	}

	tests := []struct {
		name      string
		header    string
		wantCode  int
		wantPass  bool
		wantCalls int
	}{
		{"缺少 Authorization 头", "", errcode.ErrUnauthorized, false, 0},
		{"格式错误：非 Bearer", "Token " + validToken, errcode.ErrUnauthorized, false, 0},
		{"格式错误：Bearer 后为空", "Bearer ", errcode.ErrUnauthorized, false, 0},
		{"签名被篡改", "Bearer " + forgedToken, errcode.ErrUnauthorized, false, 0},
		{"已过期", "Bearer " + expiredToken, errcode.ErrUnauthorized, false, 0},
		{"版本不匹配（已撤销）", "Bearer " + staleToken, errcode.ErrUnauthorized, false, 1},
		{"有效凭证", "Bearer " + validToken, errcode.OK, true, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := &fakeChecker{recorded: 5}

			rec, reached, _ := callAuth(t, issuer, checker, tt.header)

			if reached != tt.wantPass {
				t.Fatalf("业务处理放行情况期望 %v，实际 %v", tt.wantPass, reached)
			}
			if got := decodeCode(t, rec); got != tt.wantCode {
				t.Fatalf("错误码期望 %d，实际 %d", tt.wantCode, got)
			}
			// 本地验签失败的请求不应触发会话检查。
			if checker.callsSeen != tt.wantCalls {
				t.Fatalf("会话检查调用次数期望 %d，实际 %d", tt.wantCalls, checker.callsSeen)
			}
		})
	}
}

// 大小写不敏感地接受 Bearer 前缀。
func TestAuthAcceptsCaseInsensitiveBearer(t *testing.T) {
	issuer := newTestIssuer(t)
	checker := &fakeChecker{recorded: 0}

	raw, err := issuer.Issue(1, 0, time.Now())
	if err != nil {
		t.Fatalf("签发失败: %v", err)
	}

	for _, prefix := range []string{"Bearer ", "bearer ", "BEARER "} {
		t.Run(prefix, func(t *testing.T) {
			checker.callsSeen = 0

			_, reached, _ := callAuth(t, issuer, checker, prefix+raw)
			if !reached {
				t.Fatalf("前缀 %q 应被接受", prefix)
			}
		})
	}
}

// 校验通过时上下文必须携带正确身份——否则 logic 层取不到当前用户。
func TestAuthInjectsIdentity(t *testing.T) {
	issuer := newTestIssuer(t)
	checker := &fakeChecker{recorded: 9}

	const uid = int64(424242)

	raw, err := issuer.Issue(uid, 9, time.Now())
	if err != nil {
		t.Fatalf("签发失败: %v", err)
	}

	_, reached, ctx := callAuth(t, issuer, checker, "Bearer "+raw)
	if !reached {
		t.Fatalf("有效凭证应放行")
	}

	id, ok := IdentityFrom(ctx)
	if !ok {
		t.Fatalf("上下文应携带身份")
	}
	if id.UserID != uid {
		t.Fatalf("身份 uid 期望 %d，实际 %d", uid, id.UserID)
	}

	got, err := MustIdentity(ctx)
	if err != nil {
		t.Fatalf("MustIdentity 期望成功，实际: %v", err)
	}
	if got.UserID != uid {
		t.Fatalf("MustIdentity 的 uid 期望 %d，实际 %d", uid, got.UserID)
	}
}

// 依赖缺失时必须拒绝，绝不能放行。
func TestAuthRejectsWhenDependenciesMissing(t *testing.T) {
	tests := []struct {
		name    string
		parser  TokenParser
		checker RevocationChecker
	}{
		{"parser 为 nil", nil, &fakeChecker{}},
		{"checker 为 nil", newTestIssuer(t), nil},
		{"两者皆为 nil", nil, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec, reached, _ := callAuth(t, tt.parser, tt.checker, "Bearer whatever")
			if reached {
				t.Fatalf("依赖缺失时不得放行")
			}
			if got := decodeCode(t, rec); got != errcode.ErrUnauthorized {
				t.Fatalf("错误码期望 %d，实际 %d", errcode.ErrUnauthorized, got)
			}
		})
	}
}

// 会话存储故障映射为依赖不可用（50001），与「凭证已撤销」（40100）区分。
func TestAuthMapsStorageFailureToDependencyError(t *testing.T) {
	issuer := newTestIssuer(t)
	checker := &fakeChecker{err: session.ErrUnavailable}

	raw, err := issuer.Issue(1, 0, time.Now())
	if err != nil {
		t.Fatalf("签发失败: %v", err)
	}

	rec, reached, _ := callAuth(t, issuer, checker, "Bearer "+raw)

	if reached {
		t.Fatalf("存储故障时不得放行")
	}
	if got := decodeCode(t, rec); got != errcode.ErrDependency {
		t.Fatalf("错误码期望 %d，实际 %d", errcode.ErrDependency, got)
	}
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("HTTP 状态期望 %d，实际 %d", http.StatusServiceUnavailable, rec.Code)
	}
}

// 未挂载中间件的请求不得从上下文里读出身份。
func TestIdentityFromEmptyContext(t *testing.T) {
	if _, ok := IdentityFrom(context.Background()); ok {
		t.Fatalf("空上下文不应有身份")
	}
	if _, err := MustIdentity(context.Background()); err == nil {
		t.Fatalf("空上下文调用 MustIdentity 应返回错误")
	}
}

// RedisChecker 在 Store 为 nil 时返回 ErrUnavailable，而不是 panic。
func TestRedisCheckerWithoutStore(t *testing.T) {
	var checker RedisChecker

	if _, err := checker.Matches(context.Background(), 1, 0); !errors.Is(err, session.ErrUnavailable) {
		t.Fatalf("期望 session.ErrUnavailable，实际: %v", err)
	}
}

// 细节：Bearer 前缀后带多余空格仍应被接受（TrimSpace 的语义）。
func TestAuthTrimsTokenWhitespace(t *testing.T) {
	issuer := newTestIssuer(t)
	checker := &fakeChecker{recorded: 0}

	raw, err := issuer.Issue(1, 0, time.Now())
	if err != nil {
		t.Fatalf("签发失败: %v", err)
	}

	_, reached, _ := callAuth(t, issuer, checker, "Bearer   "+strings.TrimSpace(raw))
	if !reached {
		t.Fatalf("前缀后的多余空格应被裁剪后接受")
	}
}
