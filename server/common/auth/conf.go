// Package auth 提供登录态签发与校验的公共能力。
//
// 按 backend/architecture 与 identity/auth 的约束，凭证校验必须由每个业务域
// 独立完成，因此签名密钥与有效期属于跨域共享配置，统一定义在这里。
package auth

import (
	"errors"
	"fmt"
	"time"
)

// minSecretLen 是签名密钥的最小长度，低于此值视为配置不可用。
const minSecretLen = 16

// MinSecretLen 对外暴露密钥长度下限，供 token.NewIssuer 复用同一标准，
// 避免两处各写一个数字后逐渐不一致。
func MinSecretLen() int {
	return minSecretLen
}

// AuthConf 是登录态签发与校验的配置。
//
// JwtSecret 不设默认值：密钥必须由环境变量 JWT_SECRET 注入，
// 不得以字面量形式出现在配置文件或代码中。
type AuthConf struct {
	JwtSecret     string `json:",optional,env=JWT_SECRET"`
	TokenTTLHours int    `json:",default=168,env=JWT_TTL_HOURS"`
}

// TTL 返回凭证有效期。默认 168 小时（7 天）。
func (c AuthConf) TTL() time.Duration {
	if c.TokenTTLHours <= 0 {
		return defaultTokenTTL
	}

	return time.Duration(c.TokenTTLHours) * time.Hour
}

// Validate 校验配置是否可用于签发与校验凭证。
func (c AuthConf) Validate() error {
	if len(c.JwtSecret) < minSecretLen {
		return fmt.Errorf("auth: JwtSecret 未配置或长度不足 %d 个字符", minSecretLen)
	}

	if c.TokenTTLHours <= 0 {
		return errors.New("auth: TokenTTLHours 必须为正数")
	}

	return nil
}

// defaultTokenTTL 与 AuthConf.TokenTTLHours 的默认值保持一致（7 天）。
const defaultTokenTTL = 7 * 24 * time.Hour
