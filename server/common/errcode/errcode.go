package errcode

// 错误码区段：0 成功，4xxxx 客户端侧错误，5xxxx 服务端侧错误。
const (
	OK              = 0
	ErrParam        = 40001
	ErrUnauthorized = 40100
	ErrForbidden    = 40300
	ErrNotFound     = 40400
	ErrInternal     = 50000
	ErrDependency   = 50001
)

var messages = map[int]string{
	OK:              "ok",
	ErrParam:        "参数非法",
	ErrUnauthorized: "未认证",
	ErrForbidden:    "无权限",
	ErrNotFound:     "资源不存在",
	ErrInternal:     "服务内部错误",
	ErrDependency:   "依赖不可用",
}

// Message 返回错误码对应的默认文案，未知错误码回退为内部错误文案。
func Message(code int) string {
	if msg, ok := messages[code]; ok {
		return msg
	}
	return messages[ErrInternal]
}
