package handler

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/tianlu1990s/gim/internal/service"
)

// Handlers 聚合所有 HTTP Handler，由 main.go 注册到 Gin 路由。
type Handlers struct {
	Auth         *AuthHandler
	User         *UserHandler
	Friend       *FriendHandler
	Message      *MessageHandler
	Conversation *ConversationHandler
}

// NewHandlers 创建所有 Handler，注入对应的 Service 依赖。
func NewHandlers(svc *service.Services) *Handlers {
	return &Handlers{
		Auth:         NewAuthHandler(svc.Auth),
		User:         NewUserHandler(svc.User),
		Friend:       NewFriendHandler(svc.Friend),
		Message:      NewMessageHandler(svc.Message),
		Conversation: NewConversationHandler(svc.Conversation),
	}
}

// extractBearerToken 从 Authorization header 提取 Bearer token。
// "Bearer <token>" → "<token>"；非 Bearer 格式返回空字符串。
func extractBearerToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if len(authHeader) > 7 && strings.EqualFold(authHeader[:7], "Bearer ") {
		return authHeader[7:]
	}
	return ""
}
