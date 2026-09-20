package response

import (
	"encoding/json"
	"net/http"

	"bilibili-web/server/common/errcode"
)

// Body 是全部 HTTP 接口统一使用的响应结构。
type Body struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func OK(data any) Body {
	return Body{Code: errcode.OK, Message: errcode.Message(errcode.OK), Data: data}
}

func Fail(code int, message string) Body {
	if message == "" {
		message = errcode.Message(code)
	}
	return Body{Code: code, Message: message, Data: nil}
}

// Write 以 JSON 输出统一响应；HTTP 状态码仅表达传输层语义。
func Write(w http.ResponseWriter, status int, body Body) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func WriteOK(w http.ResponseWriter, data any) {
	Write(w, http.StatusOK, OK(data))
}

func WriteFail(w http.ResponseWriter, status int, code int, message string) {
	Write(w, status, Fail(code, message))
}
