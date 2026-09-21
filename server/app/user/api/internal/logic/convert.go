// Package logic 承载 user 域的业务规则。
//
// 分层约定（backend/architecture）：SQL 只在 model 层，业务规则在 logic 层，
// handler 只做绑定与响应。因此本包不出现任何 SQL 字符串，也不直接写 HTTP 响应。
package logic

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"bilibili-web/server/app/user/api/internal/types"
	"bilibili-web/server/app/user/model"
)

// sqlNullTime 是 database/sql.NullTime 的本地别名，
// 让下面的签名短一些而不影响语义。
type sqlNullTime = sql.NullTime

// 资料字段的长度上限，与建表语句的列宽保持一致。
const (
	maxUsernameLen  = 32
	minUsernameLen  = 3
	maxNicknameLen  = 32
	maxSignatureLen = 200
	maxAvatarURLLen = 512
)

// 性别取值。0 未知（默认），1 男，2 女。
const (
	genderUnknown = 0
	genderMale    = 1
	genderFemale  = 2
)

// 用户等级与角色的注册默认值。
//
// 二者的成长与权限规则尚未定义（design Open Questions），本变更只落字段与默认值。
const (
	defaultLevel = 0
	defaultRole  = 0
)

// 时间字段的对外格式。
//
// 统一为 RFC3339 而非依赖各端的默认序列化，避免前端拿到时间后
// 还要猜时区与精度。
const timeLayout = time.RFC3339

// userToDTO 把 model 结构转为本人资料的响应 DTO。
//
// 这是「响应不含密码哈希」的唯一收敛点：转换只取白名单字段，
// 新增列不会自动出现在响应里。
func userToDTO(u *model.Users) *types.UserDTO {
	if u == nil {
		return nil
	}

	return &types.UserDTO{
		ID:        u.Id,
		Username:  u.Username,
		Nickname:  u.Nickname,
		AvatarURL: u.AvatarUrl,
		Signature: u.Signature,
		Gender:    u.Gender,
		Birthday:  formatNullTime(u.Birthday),
		Level:     u.Level,
		Role:      u.Role,
		CreatedAt: u.CreatedAt.Format(timeLayout),
	}
}

// userToPublicDTO 把 model 结构转为他人公开资料的响应 DTO。
//
// 与 userToDTO 分开定义而不是复用后删字段：可见性差异是业务规则，
// 写成两份显式白名单，评审时一眼能看出「他人能看到什么」。
func userToPublicDTO(u *model.Users) *types.PublicUserDTO {
	if u == nil {
		return nil
	}

	return &types.PublicUserDTO{
		ID:        u.Id,
		Nickname:  u.Nickname,
		AvatarURL: u.AvatarUrl,
		Signature: u.Signature,
		Gender:    u.Gender,
		Level:     u.Level,
	}
}

// formatNullTime 把可空时间转为字符串，无效值输出空串。
func formatNullTime(t sqlNullTime) string {
	if !t.Valid {
		return ""
	}

	return t.Time.Format(timeLayout)
}

// validateUsername 校验用户名。
//
// 只允许字母、数字与下划线：用户名会出现在 URL 与日志中，
// 放开任意字符会让「需要转义的地方」变得难以穷举。
func validateUsername(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return fmt.Errorf("用户名不得为空")
	}
	// 长度按字符数而非字节数判定，否则中文用户名会被误判超长。
	if n := utf8.RuneCountInString(trimmed); n < minUsernameLen || n > maxUsernameLen {
		return fmt.Errorf("用户名长度需在 %d-%d 个字符之间", minUsernameLen, maxUsernameLen)
	}
	for _, r := range trimmed {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '_':
		default:
			return fmt.Errorf("用户名只能包含字母、数字与下划线")
		}
	}

	return nil
}

// normalizeUsername 归一化用户名。
//
// 归一化到小写，使「大小写不敏感地判定重复」在应用层就成立。
// 注意这只是第一道：并发下仍可能两个请求同时通过，最终由
// username 唯一索引兜底（identity/auth 的注册场景要求）。
//
// 建表时选了 utf8mb4_0900_ai_ci 排序规则，MySQL 侧本身已大小写不敏感，
// 这里再做一次是为了让「存进去的值」本身也是规范形态。
func normalizeUsername(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// validateNickname 校验昵称。
func validateNickname(nickname string) error {
	trimmed := strings.TrimSpace(nickname)
	if trimmed == "" {
		return fmt.Errorf("昵称不得为空")
	}
	if n := utf8.RuneCountInString(trimmed); n > maxNicknameLen {
		return fmt.Errorf("昵称不得超过 %d 个字符", maxNicknameLen)
	}

	return nil
}

// validateSignature 校验个性签名。允许为空。
func validateSignature(signature string) error {
	if n := utf8.RuneCountInString(signature); n > maxSignatureLen {
		return fmt.Errorf("个性签名不得超过 %d 个字符", maxSignatureLen)
	}

	return nil
}

// validateAvatarURL 校验头像地址。允许为空。
//
// 本变更不承担文件上传与存储（proposal Out of Scope），头像只接受外部 URL。
// 因此这里只做基本形态校验：必须是 http/https 且长度合理。
// 不在这里探测 URL 可达性——那会把接口的响应时间绑到第三方站点上。
func validateAvatarURL(url string) error {
	if url == "" {
		return nil
	}
	if len(url) > maxAvatarURLLen {
		return fmt.Errorf("头像地址不得超过 %d 个字节", maxAvatarURLLen)
	}
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("头像地址必须以 http:// 或 https:// 开头")
	}

	return nil
}

// validateGender 校验性别取值。
func validateGender(gender int64) error {
	switch gender {
	case genderUnknown, genderMale, genderFemale:
		return nil
	default:
		return fmt.Errorf("性别取值非法")
	}
}

// parseBirthday 解析生日字符串，空串表示清除该字段。
//
// 只接受 YYYY-MM-DD 这一种格式，不做多格式猜测：
// 「03/04/2024」在不同地区含义相反，猜测会引入难以复现的数据错误。
func parseBirthday(raw string) (sqlNullTime, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return sqlNullTime{}, nil
	}

	t, err := time.ParseInLocation("2006-01-02", trimmed, time.Local)
	if err != nil {
		return sqlNullTime{}, fmt.Errorf("生日格式需为 YYYY-MM-DD")
	}
	// 拒绝未来日期与过于久远的日期：前者必然错误，后者通常是录入失误。
	now := time.Now()
	if t.After(now) {
		return sqlNullTime{}, fmt.Errorf("生日不得晚于当前日期")
	}
	if t.Year() < 1900 {
		return sqlNullTime{}, fmt.Errorf("生日年份不得早于 1900")
	}

	return sqlNullTime{Time: t, Valid: true}, nil
}

// isNotFound 包装 model 层的未找到判定，避免 logic 层各处重复 import errors。
func isNotFound(err error) bool {
	return errors.Is(err, model.ErrNotFound)
}

// trimSpace 去除首尾空白。
//
// 抽成函数而非各处调用 strings.TrimSpace，是为了让「入库前统一去空白」
// 这条规则只有一个执行点：昵称与签名两端的空白会让前端的等值判断出错。
func trimSpace(s string) string {
	return strings.TrimSpace(s)
}
