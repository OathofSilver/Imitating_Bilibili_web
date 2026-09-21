package auth

import (
	"testing"
	"time"
)

func TestAuthConfTTL(t *testing.T) {
	tests := []struct {
		name string
		conf AuthConf
		want time.Duration
	}{
		{
			name: "显式配置的小时数生效",
			conf: AuthConf{JwtSecret: "0123456789abcdef", TokenTTLHours: 24},
			want: 24 * time.Hour,
		},
		{
			name: "未配置或非法时回退为 7 天",
			conf: AuthConf{JwtSecret: "0123456789abcdef", TokenTTLHours: 0},
			want: 7 * 24 * time.Hour,
		},
		{
			name: "负数同样回退为 7 天",
			conf: AuthConf{JwtSecret: "0123456789abcdef", TokenTTLHours: -1},
			want: 7 * 24 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.conf.TTL(); got != tt.want {
				t.Fatalf("TTL() 期望 %v，实际 %v", tt.want, got)
			}
		})
	}
}

func TestAuthConfValidate(t *testing.T) {
	tests := []struct {
		name    string
		conf    AuthConf
		wantErr bool
	}{
		{
			name:    "密钥为空时不可用",
			conf:    AuthConf{TokenTTLHours: 168},
			wantErr: true,
		},
		{
			name:    "密钥过短时不可用",
			conf:    AuthConf{JwtSecret: "short", TokenTTLHours: 168},
			wantErr: true,
		},
		{
			name:    "有效期非正时不可用",
			conf:    AuthConf{JwtSecret: "0123456789abcdef", TokenTTLHours: 0},
			wantErr: true,
		},
		{
			name:    "完整配置可用",
			conf:    AuthConf{JwtSecret: "0123456789abcdef", TokenTTLHours: 168},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.conf.Validate()
			if tt.wantErr && err == nil {
				t.Fatalf("Validate() 期望返回错误，实际为 nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("Validate() 期望通过，实际错误: %v", err)
			}
		})
	}
}
