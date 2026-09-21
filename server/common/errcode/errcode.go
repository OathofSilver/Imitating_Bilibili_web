package errcode

// 错误码区段：0 成功，4xxxx 客户端侧错误，5xxxx 服务端侧错误。
//
// 4xxxx 段内按 HTTP 语义分组，便于前端仅凭 code 判断错误类型：
//
//	400xx 参数与请求格式
//	401xx 认证
//	403xx 权限
//	404xx 资源不存在
//	409xx 状态冲突
const (
	OK              = 0
	ErrParam        = 40001
	ErrUnauthorized = 40100
	ErrForbidden    = 40300
	ErrNotFound     = 40400
	ErrConflict     = 40901
	ErrInternal     = 50000
	ErrDependency   = 50001
)

var messages = map[int]string{
	OK:              "ok",
	ErrParam:        "参数非法",
	ErrUnauthorized: "未认证",
	ErrForbidden:    "无权限",
	ErrNotFound:     "资源不存在",
	ErrConflict:     "该用户名已被占用",
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
