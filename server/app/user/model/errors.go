package model

import (
	"errors"

	"github.com/go-sql-driver/mysql"
)

// mysqlErrDuplicateEntry 是 MySQL 违反唯一约束的错误码。
//
// go-zero v1.10.3 的 sqlx 未提供重复键判定辅助函数（全仓检索无 IsDuplicate），
// 因此这里自行判定驱动返回的错误码（design 决策 4）。
// 好消息是 sqlx 内置熔断器已把 1062 登记为 acceptable，重复键不会触发熔断。
const mysqlErrDuplicateEntry = 1062

// IsDuplicateUsername 判断错误是否为 username 唯一索引冲突。
//
// 由唯一索引兜底而非「先查后插」，是因为后者在并发下存在竞态——两个请求可能
// 同时通过「用户名未被占用」的检查。唯一索引是最终防线，因此注册流程必须
// 能识别它抛出的错误码。
//
// 只判定错误码 1062，不去匹配错误信息中的索引名：当前 users 表只有 username
// 一个唯一索引，1062 只可能来自它；匹配索引名会让判定依赖驱动的文案格式，
// 反而更脆弱。后续若新增其他唯一索引，此处需要一并调整。
func IsDuplicateUsername(err error) bool {
	if err == nil {
		return false
	}

	var mysqlErr *mysql.MySQLError
	if !errors.As(err, &mysqlErr) {
		return false
	}

	return mysqlErr.Number == mysqlErrDuplicateEntry
}
