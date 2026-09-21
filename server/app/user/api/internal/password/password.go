// Package password 提供密码的哈希与校验。
//
// 只暴露「生成哈希」与「校验明文」两个操作，不提供任何可逆能力：
// identity/auth 要求密码 MUST 以不可逆哈希形式存储，MUST NOT 以明文或可逆加密落库。
//
// 采用 bcrypt：它自带 salt（无需另存盐字段）且计算成本可调。
// 注册与登录都是低频接口，bcrypt 的 CPU 开销在此可接受（design Risks）。
package password

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// minPasswordLen 是密码的最小长度。
//
// 取 8 是一个明确的取舍：更长更安全，但会提高注册门槛。
// 上限 72 来自 bcrypt 本身的限制——超过 72 字节的部分会被静默丢弃，
// 若不校验，用户会看到一个「很长的密码」实际只有前 72 字节生效。
const (
	minPasswordLen = 8
	maxPasswordLen = 72
)

// ErrTooShort 与 ErrTooLong 供 api 层映射为参数非法错误码并指明字段。
var (
	ErrTooShort = fmt.Errorf("密码长度不得少于 %d 个字符", minPasswordLen)
	ErrTooLong  = fmt.Errorf("密码长度不得超过 %d 个字节", maxPasswordLen)
)

// Validate 校验密码是否满足强度要求。
//
// 刻意只校验长度，不强制「必须含大小写与数字」：那类规则会把用户推向
// `Passw0rd!` 这种可预测的变形，且常需要前端同步一套正则。
// 真正的强度保障来自 bcrypt 的计算成本与后续可能引入的撞库检测。
func Validate(plain string) error {
	if len(plain) < minPasswordLen {
		return ErrTooShort
	}
	if len(plain) > maxPasswordLen {
		return ErrTooLong
	}

	return nil
}

// Hash 生成密码哈希。
func Hash(plain string) (string, error) {
	if err := Validate(plain); err != nil {
		return "", err
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("password: 生成哈希失败: %w", err)
	}

	return string(hashed), nil
}

// Verify 校验明文密码是否与哈希匹配。
//
// 任何不匹配的情形都返回同一个 false，不区分「哈希格式非法」与「密码错误」——
// 调用方按 identity/auth 的要求，对外只返回统一的未认证错误码。
func Verify(hashed, plain string) bool {
	if hashed == "" || plain == "" {
		return false
	}

	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain)) == nil
}

// IsHashFormat 粗略判断一个字符串是否像 bcrypt 哈希。
//
// 用于 model 层测试与数据体检，不参与校验流程。
func IsHashFormat(s string) bool {
	_, err := bcrypt.Cost([]byte(s))
	return err == nil && !errors.Is(err, bcrypt.ErrHashTooShort)
}
