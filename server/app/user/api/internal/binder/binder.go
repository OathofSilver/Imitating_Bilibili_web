// Package binder 负责请求体的解码与参数校验。
//
// 按 backend/architecture，参数校验属于 api 层职责；按 api/contract，
// 校验失败必须返回 40001 且在 message 中指明字段名。
package binder

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// maxBodyBytes 限制请求体大小，避免超大载荷占用内存。
const maxBodyBytes = 64 << 10

// ErrEmptyBody 表示请求体为空。
var ErrEmptyBody = errors.New("请求体为空")

// ValidationError 是参数校验失败，携带需要告知用户的字段名。
type ValidationError struct {
	Field  string
	Reason string
}

func (e *ValidationError) Error() string {
	if e.Field == "" {
		return e.Reason
	}

	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}

// NewValidationError 构造一个字段级校验错误。
func NewValidationError(field, reason string) *ValidationError {
	return &ValidationError{Field: field, Reason: reason}
}

// Decode 把请求体解码到 dst。
//
// 使用 DisallowUnknownFields：这是 identity/profile「请求体中出现白名单外字段时
// MUST 整体拒绝」的实现基础。JSON 解码遇到未知字段会直接报错，
// 且此时**没有任何字段被写入 dst**，天然满足「不得应用部分更新」。
//
// 返回的错误若为 *ValidationError，说明是「字段问题」，
// 其余情况（IO 错误、JSON 语法错误）由调用方归为通用参数非法。
func Decode(r *http.Request, dst any) error {
	if r.Body == nil {
		return ErrEmptyBody
	}

	limited := io.LimitReader(r.Body, maxBodyBytes)

	decoder := json.NewDecoder(limited)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return translateDecodeError(err, dst)
	}

	// 请求体中存在第二个 JSON 值，说明载荷不是单一对象。
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return &ValidationError{Reason: "请求体必须是单个 JSON 对象"}
	}

	return nil
}

// RawFields 读出请求体的顶层字段名，用于未知字段的诊断与拒绝。
//
// 单独走一遍解码而非复用 Decode 的报错信息，是为了拿到完整字段列表：
// encoding/json 只报告第一个未知字段，若用户同时提交了 username 与 role，
// 我们希望一次性告知全部越界字段，而不是让用户改一个报一个。
func RawFields(r *http.Request) (map[string]any, error) {
	if r.Body == nil {
		return nil, ErrEmptyBody
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("读取请求体失败: %w", err)
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		return nil, ErrEmptyBody
	}

	var fields map[string]any
	if err := json.Unmarshal(body, &fields); err != nil {
		return nil, fmt.Errorf("请求体不是合法的 JSON 对象: %w", err)
	}

	return fields, nil
}

// RejectUnknownFields 检查 fields 中是否出现 allowed 之外的字段，
// 有则返回一个列出**全部**越界字段的校验错误。
func RejectUnknownFields(fields map[string]any, allowed ...string) error {
	allowSet := make(map[string]struct{}, len(allowed))
	for _, name := range allowed {
		allowSet[name] = struct{}{}
	}

	// 排序输出，让报错信息稳定可测。
	var unknown []string
	for name := range fields {
		if _, ok := allowSet[name]; !ok {
			unknown = append(unknown, name)
		}
	}
	if len(unknown) == 0 {
		return nil
	}
	sortStrings(unknown)

	return &ValidationError{
		Reason: fmt.Sprintf("请求体包含不可修改的字段: %s", strings.Join(unknown, ", ")),
	}
}

// translateDecodeError 把 encoding/json 的错误转成字段级校验错误。
func translateDecodeError(err error, dst any) error {
	if errors.Is(err, io.EOF) {
		return ErrEmptyBody
	}
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return &ValidationError{Reason: "请求体不是完整的 JSON"}
	}

	msg := err.Error()

	// encoding/json 的未知字段报错形如：
	//   json: unknown field "username"
	if field, ok := strings.CutPrefix(msg, `json: unknown field "`); ok {
		field = strings.TrimSuffix(field, `"`)

		return &ValidationError{
			Field:  field,
			Reason: "该字段不可通过本接口修改",
		}
	}

	// 类型不匹配的报错形如：
	//   json: cannot unmarshal string into Go struct field X.gender of type int64
	if idx := strings.Index(msg, "Go struct field "); idx >= 0 {
		rest := msg[idx+len("Go struct field "):]
		// 取 "结构体名.字段名 " 中的字段名部分。
		if dot := strings.Index(rest, "."); dot >= 0 {
			rest = rest[dot+1:]
		}
		if space := strings.IndexAny(rest, " ,"); space > 0 {
			rest = rest[:space]
		}

		return &ValidationError{
			Field:  rest,
			Reason: "字段类型不合法",
		}
	}

	return &ValidationError{Reason: "请求体解析失败"}
}

// sortStrings 是一个不引入 sort 依赖的最小插入排序：
// 待排序的字段名数量等于越界字段数，通常个位数。
func sortStrings(items []string) {
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && items[j] < items[j-1]; j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
}
