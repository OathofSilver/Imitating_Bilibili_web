package handler

import (
	"encoding/json"
	"strings"
	"testing"

	"bilibili-web/server/app/user/api/internal/binder"
	"bilibili-web/server/app/user/api/internal/types"
)

// profileWritableFields 必须与 types.UpdateProfileReq 的 json 标签完全一致。
//
// 两处各写一份是刻意的（详见 request.go 的注释），代价是可能漂移，
// 因此用这条测试把它们钉在一起：结构体新增了字段却忘了放行，
// 或放行了一个结构体上没有的字段，都会在这里失败。
func TestProfileWritableFieldsMatchRequestStruct(t *testing.T) {
	// 用实际序列化一个填满全部字段的请求体，读出其 json 键名。
	nickname := "n"
	avatar := "https://example.com/a.png"
	signature := "s"
	gender := int64(1)
	birthday := "2000-01-01"

	raw, err := json.Marshal(&types.UpdateProfileReq{
		Nickname:  &nickname,
		AvatarURL: &avatar,
		Signature: &signature,
		Gender:    &gender,
		Birthday:  &birthday,
	})
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}

	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatalf("反序列化失败: %v", err)
	}

	allowed := map[string]struct{}{}
	for _, name := range profileWritableFields {
		allowed[name] = struct{}{}
	}

	// 结构体上有的字段必须都在白名单里。
	for name := range fields {
		if _, ok := allowed[name]; !ok {
			t.Fatalf("types.UpdateProfileReq 的字段 %q 未出现在白名单 %v 中", name, profileWritableFields)
		}
		delete(allowed, name)
	}

	// 白名单里的字段必须都存在于结构体上。
	if len(allowed) > 0 {
		missing := make([]string, 0, len(allowed))
		for name := range allowed {
			missing = append(missing, name)
		}
		t.Fatalf("白名单中的字段 %v 在 types.UpdateProfileReq 上不存在", missing)
	}
}

// 不可修改的字段必须被整体拒绝，且一次性列出全部越界项。
func TestRejectUnknownFields(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantErr   bool
		wantParts []string
	}{
		{
			name:    "全部字段合法",
			body:    `{"nickname":"a","avatar_url":"https://e.com/a.png","signature":"s","gender":1,"birthday":"2000-01-01"}`,
			wantErr: false,
		},
		{
			name:    "空对象合法",
			body:    `{}`,
			wantErr: false,
		},
		{
			name:      "包含 username",
			body:      `{"nickname":"a","username":"hacked"}`,
			wantErr:   true,
			wantParts: []string{"username"},
		},
		{
			name:      "包含 id",
			body:      `{"id":1}`,
			wantErr:   true,
			wantParts: []string{"id"},
		},
		{
			name:      "包含 level",
			body:      `{"level":99}`,
			wantErr:   true,
			wantParts: []string{"level"},
		},
		{
			name:      "包含 role",
			body:      `{"role":1}`,
			wantErr:   true,
			wantParts: []string{"role"},
		},
		{
			name:      "包含 password_hash",
			body:      `{"password_hash":"x"}`,
			wantErr:   true,
			wantParts: []string{"password_hash"},
		},
		{
			name:      "多个越界字段全部列出",
			body:      `{"role":1,"level":99,"username":"x"}`,
			wantErr:   true,
			wantParts: []string{"role", "level", "username"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fields map[string]any
			if err := json.Unmarshal([]byte(tt.body), &fields); err != nil {
				t.Fatalf("测试数据不是合法 JSON: %v", err)
			}

			err := binder.RejectUnknownFields(fields, profileWritableFields...)

			if !tt.wantErr {
				if err != nil {
					t.Fatalf("期望通过，实际错误: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("期望被拒绝，实际通过")
			}

			msg := err.Error()
			for _, part := range tt.wantParts {
				if !strings.Contains(msg, part) {
					t.Fatalf("错误信息应包含 %q，实际 %q", part, msg)
				}
			}
		})
	}
}
