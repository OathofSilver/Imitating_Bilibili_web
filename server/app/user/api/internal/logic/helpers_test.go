package logic

import (
	"fmt"
	"strconv"
	"strings"

	"bilibili-web/server/app/user/api/internal/password"

	"github.com/go-sql-driver/mysql"
)

// passwordHash 用生产实现生成哈希，供测试复用。
//
// 刻意不返回固定假哈希：若测试用固定串，则密码校验逻辑被改坏时测试仍会通过。
func passwordHash(plain string) (string, error) {
	return password.Hash(plain)
}

// uidVerToken 生成测试用的「凭证」串。
//
// 真实的 JWT 由 common/token 签发（其正确性已在该包的测试中覆盖），
// 这里的替身只需要能原样携带 uid 与 ver。
func uidVerToken(uid, ver int64) string {
	return fmt.Sprintf("%d:%d", uid, ver)
}

// parseUIDVer 解析测试凭证。
//
// 允许 out 参数为 nil 以表示「不关心该值」，这样调用处可以写成
// parseUIDVer(raw, nil, nil) 而不产生未使用变量的噪音。
func parseUIDVer(raw string, outUID, outVer *int64) (int64, int64, error) {
	parts := strings.Split(raw, ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("凭证据格式非法: %q", raw)
	}

	uid, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("uid 解析失败: %w", err)
	}

	ver, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("ver 解析失败: %w", err)
	}

	if outUID != nil {
		*outUID = uid
	}
	if outVer != nil {
		*outVer = ver
	}

	return uid, ver, nil
}

// mysqlError1062 用真实驱动的错误类型构造「唯一键冲突」错误。
//
// 用真实类型而非自定义错误，是为了让 model.IsDuplicateUsername 的
// errors.As 判定路径被真正走到——若测试用自定义类型，
// 该判定逻辑就永远没被验证过。
type mysqlError1062 struct{}

func (e *mysqlError1062) Error() string {
	return "Error 1062 (23000): Duplicate entry 'x' for key 'users.uk_users_username'"
}

func (e *mysqlError1062) Is(target error) bool {
	_, ok := target.(*mysql.MySQLError)
	return ok
}

// asMySQLDuplicate 让 fakeUsers 能返回一个 errors.As 可识别的 *mysql.MySQLError。
//
// 之所以不直接把 &mysql.MySQLError 作为 errDuplicate 的值，
// 是因为它需要同时满足「Error() 有内容」与「errors.As 能被 Is 命中」，
// 这里用包装方式一次性满足。
func newDuplicateError() error {
	return &mysql.MySQLError{
		Number:  1062,
		Message: "Duplicate entry 'someone' for key 'users.uk_users_username'",
	}
}

// stubResult 是 sql.Result 的最小实现，供 Insert 替身返回。
type stubResult int64

func (r stubResult) LastInsertId() (int64, error) {
	return 0, nil
}

func (r stubResult) RowsAffected() (int64, error) {
	return int64(r), nil
}
