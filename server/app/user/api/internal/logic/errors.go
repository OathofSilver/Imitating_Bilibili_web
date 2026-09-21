package logic

import (
	"errors"
	"time"
)

// nowFunc 是签发时间的取用点。
//
// 抽成变量而非各处直接 time.Now()，是为了让测试能固定时间、
// 从而在不等待的前提下断言有效期边界。
var nowFunc = time.Now

// FieldError 把业务校验失败绑定到具体字段。
//
// identity/profile 与 api/contract 都要求校验失败时在 message 中指明字段名，
// 因此错误必须携带字段信息，而不是一段无法定位的文案。
type FieldError struct {
	Field string
	Err   error
}

func (e *FieldError) Error() string {
	if e.Field == "" {
		return e.Err.Error()
	}

	return e.Field + ": " + e.Err.Error()
}

func (e *FieldError) Unwrap() error {
	return e.Err
}

// Message 返回供响应体使用的文案，格式为「字段名：原因」。
func (e *FieldError) Message() string {
	return e.Error()
}

// asFieldError 尝试把错误提取为字段级错误。
func asFieldError(err error) (*FieldError, bool) {
	var fe *FieldError
	if errors.As(err, &fe) {
		return fe, true
	}

	return nil, false
}
