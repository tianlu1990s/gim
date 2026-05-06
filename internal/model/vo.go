package model

import "time"

// VO（View Object）定义返回给前端的结构。与 Model 分离的原因：
// 1. 控制返回字段，避免泄露敏感信息（如密码、内部 ID）
// 2. 不同接口返回同一个 Model 的不同字段子集
// 3. 可以聚合多个 Model 的数据（如会话列表 JOIN 用户信息）

// --- 认证响应 ---

type TokenPair struct {
	AccessToken     string `json:"accessToken"`
	RefreshToken    string `json:"refreshToken"`
	AccessExpireAt  int64  `json:"accessExpireAt"`  // Unix 时间戳，前端据此判断是否需要刷新
	RefreshExpireAt int64  `json:"refreshExpireAt"`
	UserID          string `json:"userId"`
}

// --- 用户视图 ---

type UserVO struct {
	UserID    string `json:"userId"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatarUrl"`
	Phone     string `json:"phone"`
	Email     string `json:"email"`
	Status    int8   `json:"status"`
	CreatedAt string `json:"createdAt"`
}

func (u *User) ToVO() *UserVO {
	return &UserVO{
		UserID:    u.UserID,
		Nickname:  u.Nickname,
		AvatarURL: u.AvatarURL,
		Phone:     u.Phone,
		Email:     u.Email,
		Status:    u.Status,
		CreatedAt: u.CreatedAt.Format(time.RFC3339),
	}
}

// OtherUserVO 查看他人资料时返回，额外包含好友关系和备注。
type OtherUserVO struct {
	UserID    string `json:"userId"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatarUrl"`
	IsFriend  bool   `json:"isFriend"`
	Remark    string `json:"remark"` // 仅好友可见
}

type SearchUserVO struct {
	UserID    string `json:"userId"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatarUrl"`
}

// --- 好友视图 ---

type FriendVO struct {
	FriendID  string `json:"friendId"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatarUrl"`
	Remark    string `json:"remark"`
	CreatedAt string `json:"createdAt"`
}

type FriendRequestVO struct {
	ID         int64  `json:"id"`
	FromUserID string `json:"fromUserId"`
	Nickname   string `json:"nickname"`
	AvatarURL  string `json:"avatarUrl"`
	Message    string `json:"message"`
	Status     int8   `json:"status"` // 0=待处理 1=已同意 2=已拒绝
	CreatedAt  string `json:"createdAt"`
}

// --- 消息视图 ---

type MessageVO struct {
	ConversationID string `json:"conversationId"`
	Seq            int64  `json:"seq"`
	SenderID       string `json:"senderId"`
	MsgType        int    `json:"msgType"`
	Content        string `json:"content"`
	ClientMsgID    string `json:"clientMsgId"`
	ServerMsgID    string `json:"serverMsgId"`
	IsRevoked      bool   `json:"isRevoked"`
	CreatedAt      string `json:"createdAt"`
}

type SendMsgResp struct {
	Seq         int64  `json:"seq"`
	ServerMsgID string `json:"serverMsgId"`
	SendTime    int64  `json:"sendTime"` // 服务端接收时间，毫秒时间戳
}

// --- 会话视图 ---

type ConversationVO struct {
	ConversationID string    `json:"conversationId"`
	ConvType       int       `json:"convType"`
	TargetID       string    `json:"targetId"`
	TargetName     string    `json:"targetName"`   // JOIN users 获取
	TargetAvatar   string    `json:"targetAvatar"` // JOIN users 获取
	MaxSeq         int64     `json:"maxSeq"`
	ReadSeq        int64     `json:"readSeq"`
	UnreadCount    int64     `json:"unreadCount"`    // MaxSeq - ReadSeq
	LastMsgContent string    `json:"lastMsgContent"` // 最后一条消息预览
	LastMsgTime    string    `json:"lastMsgTime"`
	IsPinned       bool      `json:"isPinned"`
	LastMsg        *Message  `json:"-"` // 内部使用，不序列化
	UpdatedAt      time.Time `json:"-"` // 内部排序用
}

// --- 分页 ---

// PageResult 泛型分页结果，用于所有列表接口。
type PageResult[T any] struct {
	List     []T   `json:"list"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
}
