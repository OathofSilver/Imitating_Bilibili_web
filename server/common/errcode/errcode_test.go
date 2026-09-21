package errcode

import (
	"strconv"
	"strings"
	"testing"
)

// TestMessageForConflict 覆盖任务 2.1 的验收：40901 有独立且非空的文案。
func TestMessageForConflict(t *testing.T) {
	got := Message(ErrConflict)
	if got == "" {
		t.Fatalf("Message(%d) 不应为空", ErrConflict)
	}
	if got == Message(ErrParam) {
		t.Fatalf("Message(%d) 不应与 ErrParam 的文案相同，实际 %q", ErrConflict, got)
	}
}

// TestAllCodesHaveMessage 兜住「新增了常量却忘记登记文案」这类疏漏。
func TestAllCodesHaveMessage(t *testing.T) {
	codes := []int{
		OK,
		ErrParam,
		ErrUnauthorized,
		ErrForbidden,
		ErrNotFound,
		ErrConflict,
		ErrInternal,
		ErrDependency,
	}

	for _, code := range codes {
		msg, ok := messages[code]
		if !ok {
			t.Fatalf("错误码 %d 未登记文案", code)
		}
		if msg == "" {
			t.Fatalf("错误码 %d 的文案为空", code)
		}
	}
}

// TestCodeSegments 保证区段约定不被后续新增破坏：
// 4xxxx 为客户端侧错误，5xxxx 为服务端侧错误。
func TestCodeSegments(t *testing.T) {
	tests := []struct {
		name string
		code int
		want string
	}{
		{"参数非法属客户端侧", ErrParam, "4"},
		{"未认证属客户端侧", ErrUnauthorized, "4"},
		{"无权限属客户端侧", ErrForbidden, "4"},
		{"资源不存在属客户端侧", ErrNotFound, "4"},
		{"用户名冲突属客户端侧", ErrConflict, "4"},
		{"内部错误属服务端侧", ErrInternal, "5"},
		{"依赖不可用属服务端侧", ErrDependency, "5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := strconv.Itoa(tt.code)[:1]
			if got != tt.want {
				t.Fatalf("错误码 %d 应以 %s 开头表达区段", tt.code, tt.want)
			}
		})
	}
}

// TestUnknownCodeFallsBack 未知错误码必须回退为内部错误文案，不得返回空串。
func TestUnknownCodeFallsBack(t *testing.T) {
	const unknown = 49999

	got := Message(unknown)
	if got != messages[ErrInternal] {
		t.Fatalf("未知错误码应回退为内部错误文案，实际 %q", got)
	}
	if strings.TrimSpace(got) == "" {
		t.Fatalf("回退文案不应为空")
	}
}
