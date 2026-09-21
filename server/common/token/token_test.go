package token

import (
	"errors"
	"strings"
	"testing"
	"time"
)

const (
	testSecret    = "test-secret-at-least-16-chars"
	testMinSecret = 16
	testTTL       = time.Hour
)

func newTestIssuer(t *testing.T) *Issuer {
	t.Helper()

	issuer, err := NewIssuer(testSecret, testTTL, testMinSecret)
	if err != nil {
		t.Fatalf("创建签发器失败: %v", err)
	}

	return issuer
}

// 任务 2.3 验收一：签发后可解析。
func TestIssueThenParse(t *testing.T) {
	issuer := newTestIssuer(t)
	now := time.Now()

	const (
		uid = int64(1234567890)
		ver = int64(3)
	)

	raw, err := issuer.Issue(uid, ver, now)
	if err != nil {
		t.Fatalf("签发失败: %v", err)
	}
	if raw == "" {
		t.Fatalf("签发的凭证不应为空")
	}

	claims, err := issuer.Parse(raw)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if claims.UserID != uid {
		t.Fatalf("uid 期望 %d，实际 %d", uid, claims.UserID)
	}
	if claims.Version != ver {
		t.Fatalf("ver 期望 %d，实际 %d", ver, claims.Version)
	}
	if claims.IssuedAt == nil || claims.ExpiresAt == nil {
		t.Fatalf("载荷必须同时含 iat 与 exp")
	}
	if got := claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time); got != testTTL {
		t.Fatalf("有效期期望 %v，实际 %v", testTTL, got)
	}
}

// 任务 2.3 验收二：篡改签名被拒。
func TestParseRejectsTamperedSignature(t *testing.T) {
	issuer := newTestIssuer(t)

	raw, err := issuer.Issue(1, 0, time.Now())
	if err != nil {
		t.Fatalf("签发失败: %v", err)
	}

	tests := []struct {
		name string
		raw  string
	}{
		{"签名段被替换", raw[:strings.LastIndex(raw, ".")+1] + "AAAAdeadbeef"},
		{"签名段被截断", raw[:len(raw)-4]},
		{"载荷段被替换", strings.Split(raw, ".")[0] + ".AAAA." + strings.Split(raw, ".")[2]},
		{"主体被整体替换", raw + "x"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := issuer.Parse(tt.raw); !errors.Is(err, ErrInvalidToken) {
				t.Fatalf("期望 ErrInvalidToken，实际: %v", err)
			}
		})
	}
}

// 任务 2.3 验收三：超过有效期被拒。
func TestParseRejectsExpiredToken(t *testing.T) {
	issuer := newTestIssuer(t)

	// 签发时间回拨到有效期之外。
	raw, err := issuer.Issue(1, 0, time.Now().Add(-2*testTTL))
	if err != nil {
		t.Fatalf("签发失败: %v", err)
	}

	if _, err := issuer.Parse(raw); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("过期凭证期望 ErrInvalidToken，实际: %v", err)
	}
}

// 换一把密钥必须无法通过校验——这保证密钥确实参与了签名，而非仅走过场。
func TestParseRejectsForeignSecret(t *testing.T) {
	issuer := newTestIssuer(t)

	raw, err := issuer.Issue(1, 0, time.Now())
	if err != nil {
		t.Fatalf("签发失败: %v", err)
	}

	other, err := NewIssuer("another-secret-value-16", testTTL, testMinSecret)
	if err != nil {
		t.Fatalf("创建另一个签发器失败: %v", err)
	}

	if _, err := other.Parse(raw); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("异密钥解析期望 ErrInvalidToken，实际: %v", err)
	}
}

// 不能锁定算法：头部改成 none 必须被拒。
func TestParseRejectsNoneAlgorithm(t *testing.T) {
	issuer := newTestIssuer(t)

	// {"alg":"none","typ":"JWT"} 的 base64url 编码，配合空签名段。
	const noneToken = "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJ1aWQiOjEsInZlciI6MH0."

	if _, err := issuer.Parse(noneToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("算法为 none 的凭证必须被拒，实际: %v", err)
	}
}

// 空串与垃圾输入必须归入同一个哨兵错误。
func TestParseRejectsMalformedInput(t *testing.T) {
	issuer := newTestIssuer(t)

	for _, raw := range []string{"", "   ", "abc", "a.b", "a.b.c.d"} {
		if _, err := issuer.Parse(raw); !errors.Is(err, ErrInvalidToken) {
			t.Fatalf("输入 %q 期望 ErrInvalidToken，实际: %v", raw, err)
		}
	}
}

// 构造签发器时的配置校验：密钥过短、有效期为非正数都必须显式失败。
func TestNewIssuerValidation(t *testing.T) {
	tests := []struct {
		name    string
		secret  string
		ttl     time.Duration
		wantErr bool
	}{
		{"密钥为空", "", testTTL, true},
		{"密钥过短", "short", testTTL, true},
		{"有效期为零", testSecret, 0, true},
		{"有效期非正", testSecret, -time.Hour, true},
		{"配置完整", testSecret, testTTL, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issuer, err := NewIssuer(tt.secret, tt.ttl, testMinSecret)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("期望返回错误，实际为 nil")
				}
				if issuer != nil {
					t.Fatalf("出错时不应返回可用签发器")
				}
				return
			}
			if err != nil {
				t.Fatalf("期望成功，实际错误: %v", err)
			}
		})
	}
}

// 非正数 uid 不应被签发：那会产生一个无法归属身份的凭证。
func TestIssueRejectsNonPositiveUserID(t *testing.T) {
	issuer := newTestIssuer(t)

	for _, uid := range []int64{0, -1} {
		if _, err := issuer.Issue(uid, 0, time.Now()); err == nil {
			t.Fatalf("uid=%d 期望返回错误，实际为 nil", uid)
		}
	}
}
