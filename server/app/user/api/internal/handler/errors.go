package handler

import (
	"errors"
	"net/http"

	"bilibili-web/server/app/user/api/internal/logic"
	"bilibili-web/server/common/errcode"
	"bilibili-web/server/common/response"

	"github.com/zeromicro/go-zero/core/logx"
)

// writeLogicError 把 logic 层错误映射为统一错误码。
//
// 集中在一处做映射，而不是让每个 handler 各写一遍 switch：
// 错误码与错误的对应关系是接口契约的一部分（api/contract），
// 分散写会让「同一个错误在不同接口返回不同码」这种事迟早发生。
//
// 映射规则：
//   - 字段级校验失败        → 40001，message 带字段名
//   - 用户名已被占用        → 40901
//   - 凭据无效（含不存在）  → 40100，且不区分原因
//   - 资源不存在            → 40400
//   - 依赖不可用            → 50001
//   - 其余（未预期）        → 50000，仅记日志，不把内部细节返回给客户端
func writeLogicError(w http.ResponseWriter, err error) {
	var fieldErr *logic.FieldError
	if errors.As(err, &fieldErr) {
		response.WriteFail(w, http.StatusBadRequest, errcode.ErrParam, fieldErr.Message())
		return
	}

	switch {
	case errors.Is(err, logic.ErrUsernameTaken):
		response.WriteFail(w, http.StatusConflict, errcode.ErrConflict, "")
	case errors.Is(err, logic.ErrInvalidCredentials):
		response.WriteFail(w, http.StatusUnauthorized, errcode.ErrUnauthorized, "")
	case errors.Is(err, logic.ErrUserNotFound):
		response.WriteFail(w, http.StatusNotFound, errcode.ErrNotFound, "")
	case errors.Is(err, logic.ErrDependencyUnavailable):
		logx.Errorf("[user] 依赖不可用: %v", err)
		response.WriteFail(w, http.StatusServiceUnavailable, errcode.ErrDependency, "")
	default:
		// 未预期的错误：内部细节只进日志。响应里的 message 由 errcode 提供，
		// 不含任何数据库或堆栈信息。
		logx.Errorf("[user] 未预期的业务错误: %v", err)
		response.WriteFail(w, http.StatusInternalServerError, errcode.ErrInternal, "")
	}
}

// writeBindError 把请求体解析错误映射为 40001。
func writeBindError(w http.ResponseWriter, err error) {
	response.WriteFail(w, http.StatusBadRequest, errcode.ErrParam, err.Error())
}
