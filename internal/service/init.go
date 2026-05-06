package service

import (
	"github.com/redis/go-redis/v9"

	"github.com/tianlu1990s/gim/internal/config"
	"github.com/tianlu1990s/gim/internal/repository"
	"github.com/tianlu1990s/gim/internal/ws"
	"github.com/tianlu1990s/gim/pkg/jwt"
)

// Services 聚合所有业务 Service，Handler 通过它获取所需的 Service。
// 每个 Service 持有自己依赖的 Repo/外部组件，而非持有整个 Repositories。
type Services struct {
	Auth         AuthService
	User         UserService
	Friend       FriendService
	Message      MessageService
	Conversation ConversationService
}

// NewServices 创建所有 Service，注入各自依赖。
// repos 为各 Service 提供数据访问，jwtMgr 用于 Token 生成/验证，
// rdb 用于缓存/黑名单，hub 用于 WS 消息推送。
func NewServices(repos *repository.Repositories, cfg *config.Config, jwtMgr *jwt.JWTManager, rdb *redis.Client, hub *ws.Hub) *Services {
	return &Services{
		Auth:         newAuthService(repos, jwtMgr, rdb, hub, cfg),
		User:         newUserService(repos),
		Friend:       newFriendService(repos, hub, rdb),
		Message:      newMessageService(repos, hub, rdb),
		Conversation: newConversationService(repos),
	}
}
