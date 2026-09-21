// Package token 提供登录凭证（JWT）的签发与解析。
//
// 设计见 design.md 决策 2：采用纯自校验的 JWT（HS256），使任一业务域都能在本地
// 完成验签与有效期校验，无需在受保护请求上跨域调用用户域。
//
// 凭证载荷刻意保持极简——只有 uid、ver、iat、exp：
//   - uid：用户 ID（Snowflake），受保护接口据此确定请求身份
//   - ver：会话版本号，用于服务端撤销（见 session.go）
//   - iat/exp：签发与过期时间，由 JWT 库自动写入与校验
//
// 刻意不放 username、nickname 等资料字段：JWT 是明文可读的，且资料会在凭证
// 有效期内变化，把可变数据放进凭证只会制造「凭证里的昵称与库里不一致」的问题。
package token

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims 是登录凭证的载荷。
type Claims struct {
	// UserID 是用户的 Snowflake ID。
	UserID int64 `json:"uid"`
	// Version 是签发时的会话版本号，与 Redis 中的当前版本比较以判定是否已撤销。
	Version int64 `json:"ver"`

	jwt.RegisteredClaims
}

// Issuer 按配置签发与解析凭证。
type Issuer struct {
	secret []byte
	ttl    time.Duration
}

// NewIssuer 创建签发器。
//
// secret 为空或过短时返回错误：签名密钥是安全前提，缺失时必须显式失败，
// 不得回退为某个内置默认值——那会让所有部署共享同一把可被公开推算的密钥。
func NewIssuer(secret string, ttl time.Duration, minSecretLen int) (*Issuer, error) {
	if len(secret) < minSecretLen {
		return nil, fmt.Errorf("token: 签名密钥长度不足 %d 个字符", minSecretLen)
	}
	if ttl <= 0 {
		return nil, errors.New("token: 凭证有效期必须为正数")
	}

	return &Issuer{secret: []byte(secret), ttl: ttl}, nil
}

// TTL 返回本签发器使用的凭证有效期。
func (i *Issuer) TTL() time.Duration {
	return i.ttl
}

// Issue 为指定用户签发凭证。
//
// ver 由调用方传入当前的会话版本号：登录时传递增后的值，
// 这样同一用户先前签发的凭证会因版本不匹配而全部失效。
func (i *Issuer) Issue(userID, ver int64, now time.Time) (string, error) {
	if userID <= 0 {
		return "", errors.New("token: 用户 ID 必须为正数")
	}

	claims := Claims{
		UserID:  userID,
		Version: ver,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(i.ttl)),
		},
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(i.secret)
	if err != nil {
		return "", fmt.Errorf("token: 签发凭证失败: %w", err)
	}

	return signed, nil
}

// Parse 解析并校验凭证，成功时返回其载荷。
//
// 校验内容包括签名、有效期，以及算法必须是 HS256。最后一条不可省略：
// 若只依赖密钥校验而不锁定算法，攻击者可把头部改成 `none` 或非对称算法尝试绕过。
func (i *Issuer) Parse(raw string) (*Claims, error) {
	if raw == "" {
		return nil, ErrInvalidToken
	}

	claims := &Claims{}
	parsed, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("token: 非预期的签名算法 %v", t.Header["alg"])
		}
		return i.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	if !parsed.Valid {
		return nil, ErrInvalidToken
	}
	if claims.UserID <= 0 {
		return nil, fmt.Errorf("%w: 载荷缺少有效的 uid", ErrInvalidToken)
	}

	return claims, nil
}
