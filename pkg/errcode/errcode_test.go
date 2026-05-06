package errcode

import (
	"testing"
)

func TestError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		expected string
	}{
		{"no detail", ErrInvalidParam, "[40001] 参数错误"},
		{"with detail", ErrUserAlreadyExists.WithDetail("手机号已注册"), "[40010] 用户已存在: 手机号已注册"},
		{"internal error", ErrInternal, "[50001] 服务内部错误"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("Error() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestError_WithDetail(t *testing.T) {
	// WithDetail 返回新实例，不修改原始错误（值语义复制，并发安全）
	original := ErrUserNotFound
	detailed := original.WithDetail("用户ID: alice")

	if original.Detail != "" {
		t.Error("original should not be modified by WithDetail")
	}
	if detailed.Code != original.Code {
		t.Errorf("Code should be preserved: %d vs %d", detailed.Code, original.Code)
	}
	if detailed.Message != original.Message {
		t.Errorf("Message should be preserved: %s vs %s", detailed.Message, original.Message)
	}
	if detailed.Detail != "用户ID: alice" {
		t.Errorf("Detail should be set: got %s", detailed.Detail)
	}
}

func TestErrorCodesUnique(t *testing.T) {
	// 所有预定义错误码应该唯一
	codes := map[int]bool{}
	all := []*Error{
		ErrInvalidParam, ErrUnauthorized, ErrForbidden, ErrResourceNotFound,
		ErrAlreadyProcessed, ErrInternal,
		ErrUserAlreadyExists, ErrUserNotFound, ErrUserOrPasswordWrong, ErrUserDisabled,
		ErrAlreadyFriend, ErrNotFriend, ErrFriendRequestExists, ErrCannotFriendSelf,
		ErrMsgContentTooLong, ErrMsgNotFound, ErrMsgRevokeTimeout, ErrMsgSeqConflict,
		ErrConversationExists, ErrConversationNotFound,
		ErrRateLimitExceeded,
	}
	for _, e := range all {
		if codes[e.Code] {
			t.Errorf("duplicate error code: %d (%s)", e.Code, e.Message)
		}
		codes[e.Code] = true
	}
}
