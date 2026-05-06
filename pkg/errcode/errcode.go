package errcode

import "fmt"

// Error 统一错误类型，兼顾机器可读（Code）和人类可读（Message + Detail）。
// 所有业务错误通过它返回给前端，前端根据 Code 判断而非解析 Message。
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"msg"`
	Detail  string `json:"detail,omitempty"` // 调试详情，仅开发环境可见，生产环境清空
}

func (e *Error) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("[%d] %s: %s", e.Code, e.Message, e.Detail)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// WithDetail 复制一份错误并追加调试信息。值语义复制避免修改原始错误，并发安全。
func (e *Error) WithDetail(detail string) *Error {
	clone := *e
	clone.Detail = detail
	return &clone
}

// 错误码分段规则：
//   40xxx — 客户端错误（参数、认证、权限、资源）
//   50xxx — 服务端内部错误
//   第3位按模块划分：0=通用 1=用户 2=好友 3=消息 4=会话 9=限流

// --- 通用错误 (x0xxx) ---
var (
	ErrInvalidParam      = &Error{Code: 40001, Message: "参数错误"}
	ErrUnauthorized       = &Error{Code: 40101, Message: "未认证"}
	ErrForbidden          = &Error{Code: 40301, Message: "无权限"}
	ErrResourceNotFound   = &Error{Code: 40401, Message: "资源不存在"}
	ErrAlreadyProcessed   = &Error{Code: 40901, Message: "已处理"}
	ErrInternal           = &Error{Code: 50001, Message: "服务内部错误"}
)

// --- 用户模块 (x1xx) ---
var (
	ErrUserAlreadyExists    = &Error{Code: 40010, Message: "用户已存在"}
	ErrUserNotFound         = &Error{Code: 40410, Message: "用户不存在"}
	ErrUserOrPasswordWrong  = &Error{Code: 40110, Message: "用户名或密码错误"}
	ErrUserDisabled         = &Error{Code: 40310, Message: "用户已被禁用"}
)

// --- 好友模块 (x2xx) ---
var (
	ErrAlreadyFriend         = &Error{Code: 40020, Message: "已是好友"}
	ErrNotFriend             = &Error{Code: 40021, Message: "不是好友"}
	ErrFriendRequestExists   = &Error{Code: 40022, Message: "已发送过好友申请"}
	ErrCannotFriendSelf      = &Error{Code: 40023, Message: "不能添加自己为好友"}
)

// --- 消息模块 (x3xx) ---
var (
	ErrMsgContentTooLong  = &Error{Code: 40030, Message: "消息内容过长"}
	ErrMsgNotFound        = &Error{Code: 40430, Message: "消息不存在"}
	ErrMsgRevokeTimeout   = &Error{Code: 40031, Message: "消息已超撤回时限"}
	ErrMsgSeqConflict     = &Error{Code: 40930, Message: "消息Seq冲突"}
)

// --- 会话模块 (x4xx) ---
var (
	ErrConversationExists   = &Error{Code: 40040, Message: "会话已存在"}
	ErrConversationNotFound = &Error{Code: 40440, Message: "会话不存在"}
)

// --- 限流 (x9xx) ---
var (
	ErrRateLimitExceeded = &Error{Code: 42901, Message: "请求过于频繁"}
)
