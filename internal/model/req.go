package model

// 请求结构体统一在此定义，使用 Gin binding 标签做参数校验。
// 每个请求字段都有明确的校验规则，减少 Service 层的手动校验代码。

// --- 认证请求 ---

type RegisterReq struct {
	UserID   string `json:"userId" binding:"required,alphanum,min=4,max=32"`
	Password string `json:"password" binding:"required,min=8,max=64"`
	Nickname string `json:"nickname" binding:"omitempty,min=1,max=64"`
	Phone    string `json:"phone" binding:"omitempty,len=11"`
	Email    string `json:"email" binding:"omitempty,email"`
}

type LoginReq struct {
	UserID   string `json:"userId" binding:"required"`
	Password string `json:"password" binding:"required"`
	Platform string `json:"platform" binding:"required,oneof=web ios android"`
}

type RefreshReq struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
	Platform     string `json:"platform" binding:"required,oneof=web ios android"`
}

// --- 用户请求 ---

type UpdateProfileReq struct {
	Nickname  string `json:"nickname" binding:"omitempty,min=1,max=64"`
	AvatarURL string `json:"avatarUrl" binding:"omitempty,url,max=512"`
	Phone     string `json:"phone" binding:"omitempty,len=11"`
	Email     string `json:"email" binding:"omitempty,email"`
}

type SearchReq struct {
	Keyword  string `json:"keyword" binding:"required,min=1,max=64"`
	Page     int    `json:"page" binding:"omitempty,min=1"`
	PageSize int    `json:"pageSize" binding:"omitempty,min=1,max=50"`
}

// --- 好友请求 ---

type SendFriendRequestReq struct {
	ToUserID string `json:"toUserId" binding:"required"`
	Message  string `json:"message" binding:"omitempty,max=256"`
}

type HandleFriendRequestReq struct {
	ID int64 `uri:"id" binding:"required"`
}

type SetRemarkReq struct {
	Remark string `json:"remark" binding:"required,min=1,max=64"`
}

type ListFriendRequestsReq struct {
	Page     int `json:"page" binding:"omitempty,min=1"`
	PageSize int `json:"pageSize" binding:"omitempty,min=1,max=50"`
}

type DeleteFriendReq struct {
	UserID string `uri:"userId" binding:"required"`
}

// --- 消息请求 ---

type SendMsgReq struct {
	ConversationID string `json:"conversationId" binding:"required"`
	ClientMsgID    string `json:"clientMsgId" binding:"required"`                            // 客户端生成，全局唯一去重
	ContentType    int    `json:"contentType" binding:"required,oneof=1 2 3 4"`              // 1=文本 2=图片 3=文件 4=系统消息
	Content        string `json:"content" binding:"required,max=4096"`
}

type HistoryReq struct {
	ConversationID string `json:"conversationId" form:"conversationId" binding:"required"`
	StartSeq       int64  `json:"startSeq" form:"startSeq"` // 起始 Seq，0 表示从最新开始
	Count          int    `json:"count" form:"count" binding:"required,min=1,max=50"`
}

type MarkReadReq struct {
	ConversationID string `json:"conversationId" binding:"required"`
	ReadSeq        int64  `json:"readSeq" binding:"required,min=1"`
}

type RevokeMsgReq struct {
	ConversationID string `json:"conversationId" binding:"required"`
	ClientMsgID    string `json:"clientMsgId" binding:"required"`
}

// --- 会话请求 ---

type PinConversationReq struct {
	IsPinned bool `json:"isPinned"`
}

type DeleteConversationReq struct {
	ID string `uri:"id" binding:"required"`
}

// --- 分页 ---

type PageReq struct {
	Page     int `json:"page" form:"page" binding:"omitempty,min=1"`
	PageSize int `json:"pageSize" form:"pageSize" binding:"omitempty,min=1,max=50"`
}

// GetPage 返回安全的页码，<=0 时默认第 1 页。
func (r *PageReq) GetPage() int {
	if r.Page <= 0 {
		return 1
	}
	return r.Page
}

// GetPageSize 返回安全的页大小，<=0 时默认 20，防止请求全部数据。
func (r *PageReq) GetPageSize() int {
	if r.PageSize <= 0 {
		return 20
	}
	return r.PageSize
}
