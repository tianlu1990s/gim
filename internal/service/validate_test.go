package service

import (
	"testing"

	"github.com/tianlu1990s/gim/internal/model"
)

func TestValidateRegisterReq(t *testing.T) {
	tests := []struct {
		name    string
		req     *model.RegisterReq
		wantErr bool
	}{
		{
			name: "valid request",
			req: &model.RegisterReq{
				UserID:   "alice123",
				Password: "Pass1234",
				Phone:    "13800138000",
				Email:    "alice@example.com",
			},
			wantErr: false,
		},
		{
			name: "userId too short",
			req: &model.RegisterReq{
				UserID:   "ab",
				Password: "Pass1234",
			},
			wantErr: true,
		},
		{
			name: "userId starts with digit",
			req: &model.RegisterReq{
				UserID:   "1alice",
				Password: "Pass1234",
			},
			wantErr: true,
		},
		{
			name: "password too short",
			req: &model.RegisterReq{
				UserID:   "alice123",
				Password: "Ab1",
			},
			wantErr: true,
		},
		{
			name: "password missing uppercase",
			req: &model.RegisterReq{
				UserID:   "alice123",
				Password: "abcdef123",
			},
			wantErr: true,
		},
		{
			name: "password missing lowercase",
			req: &model.RegisterReq{
				UserID:   "alice123",
				Password: "ABCDEF123",
			},
			wantErr: true,
		},
		{
			name: "password missing digit",
			req: &model.RegisterReq{
				UserID:   "alice123",
				Password: "Abcdefgh",
			},
			wantErr: true,
		},
		{
			name: "invalid phone format",
			req: &model.RegisterReq{
				UserID:   "alice123",
				Password: "Pass1234",
				Phone:    "12345",
			},
			wantErr: true,
		},
		{
			name: "invalid email format",
			req: &model.RegisterReq{
				UserID:   "alice123",
				Password: "Pass1234",
				Email:    "not-an-email",
			},
			wantErr: true,
		},
		{
			name: "empty phone and email (optional)",
			req: &model.RegisterReq{
				UserID:   "alice123",
				Password: "Pass1234",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRegisterReq(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRegisterReq() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsStrongPassword(t *testing.T) {
	tests := []struct {
		pwd     string
		isStrong bool
	}{
		{"", false},
		{"abc", false},
		{"abcdefgh", false},   // 无大写、无数字
		{"ABCDEFGH", false},   // 无小写、无数字
		{"12345678", false},   // 无字母
		{"Abcdefgh", false},   // 无数字
		{"Abc12345", true},    // 含大小写+数字
		{"Pass1234", true},    // 标准强密码
		{"MyP@ssw0rd", true},  // 含特殊字符
		{"Ab1", false},        // 太短
	}

	for _, tt := range tests {
		t.Run(tt.pwd, func(t *testing.T) {
			got := isStrongPassword(tt.pwd)
			if got != tt.isStrong {
				t.Errorf("isStrongPassword(%q) = %v, want %v", tt.pwd, got, tt.isStrong)
			}
		})
	}
}

func TestIsDigit(t *testing.T) {
	tests := []struct {
		s     string
		isDigits bool
	}{
		{"", false},
		{"123456", true},
		{"012", true},
		{"abc123", false},
		{"12.34", false},
		{"1a2b", false},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			got := isDigit(tt.s)
			if got != tt.isDigits {
				t.Errorf("isDigit(%q) = %v, want %v", tt.s, got, tt.isDigits)
			}
		})
	}
}
