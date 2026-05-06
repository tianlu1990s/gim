package ws

import (
	"context"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/tianlu1990s/gim/internal/config"
	"github.com/tianlu1990s/gim/pkg/rediskey"
)

// Hub WebSocket 连接管理中心。类比电话总机：维护一张"谁在线"的映射表，
// 收到推送请求时查找目标用户的连接并发送，无连接的用户走离线逻辑。
// 通过 channel 接收 register/unregister/push 事件，保证 map 操作只在 Run goroutine 中进行。
type Hub struct {
	clients    map[string]map[*Client]struct{} // userID → 该用户的所有连接
	register   chan *Client                    // 注册新连接
	unregister chan *Client                    // 注销连接
	push       chan *pushTask                  // 推送消息队列，缓冲 1024

	rdb *redis.Client            // Redis，用于在线状态管理
	cfg config.WebSocketConfig   // WebSocket 配置（连接数限制、超时等）
	mu  sync.RWMutex             // 保护 clients map 的并发读写（仅在需要直接读时使用）
}

// pushTask 单次推送任务，包含目标用户 ID 和消息体。
type pushTask struct {
	userID  string
	message *WSMessage
}

// NewHub 创建 Hub。Phase 1 中 Hub 仅负责连接管理和消息推送，
// 不直接调用 Service 层（消息通过 HTTP API 发送，WS 仅用于推送通知）。
func NewHub(rdb *redis.Client, cfg config.WebSocketConfig) *Hub {
	return &Hub{
		clients:    make(map[string]map[*Client]struct{}),
		register:   make(chan *Client, 256),
		unregister: make(chan *Client, 256),
		push:       make(chan *pushTask, 1024),
		rdb:        rdb,
		cfg:        cfg,
	}
}

// Run 启动 Hub 的主事件循环，阻塞运行，需在 goroutine 中调用。
// 所有对 clients map 的修改都在此循环中进行，保证线程安全。
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.userID] == nil {
				h.clients[client.userID] = make(map[*Client]struct{})
			}
			h.clients[client.userID][client] = struct{}{}
			h.mu.Unlock()
			// 上线：在 Redis 中记录在线状态
			h.setOnline(client.userID, client.connID, client.platform)

		case client := <-h.unregister:
			h.mu.Lock()
			if conns, ok := h.clients[client.userID]; ok {
				delete(conns, client)
				if len(conns) == 0 {
					delete(h.clients, client.userID)
				}
			}
			h.mu.Unlock()
			// 下线：从 Redis 中清除连接信息
			h.setOffline(client.userID, client.connID, client.platform)

		case task := <-h.push:
			// 查找目标用户的所有连接并发送
			h.mu.RLock()
			conns := h.clients[task.userID]
			h.mu.RUnlock()
			for client := range conns {
				client.Send(task.message)
			}
		}
	}
}

// PushToUser 向指定用户推送 WebSocket 消息（非阻塞）。
// 消息写入 push channel，由 Run 循环消费并发送到所有连接。
// 如果用户不在线，消息将被丢弃（离线消息在消息发送时已持久化到 MySQL）。
func (h *Hub) PushToUser(userID string, msg *WSMessage) {
	select {
	case h.push <- &pushTask{userID: userID, message: msg}:
	default:
		// push channel 满时丢弃，避免阻塞上游 Service 层
	}
}

// IsOnline 判断用户是否在线（有活跃的 WebSocket 连接）。
func (h *Hub) IsOnline(userID string) bool {
	if h.rdb == nil {
		return false // 无 Redis 时返回 false
	}
	ctx := context.Background()
	exists, _ := h.rdb.Exists(ctx, rediskey.OnlineKey(userID)).Result()
	return exists > 0
}

// OnlineConnCount 获取用户当前连接数。
func (h *Hub) OnlineConnCount(userID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID])
}

// -------- 在线状态管理（Redis）--------

// setOnline 记录用户上线状态到 Redis。
// Key: online:{userId} (Hash), conn_map:{userId} (Set)，TTL=60s，心跳续期。
func (h *Hub) setOnline(userID, connID, platform string) {
	ctx := context.Background()
	onlineKey := rediskey.OnlineKey(userID)
	mapKey := rediskey.ConnMapKey(userID)
	ttl := 60 * time.Second

	h.rdb.SAdd(ctx, mapKey, connID)
	h.rdb.Expire(ctx, mapKey, ttl)
	h.rdb.HSet(ctx, onlineKey, "platform", platform, "lastActive", time.Now().Unix())
	h.rdb.Expire(ctx, onlineKey, ttl)
}

// setOffline 清除用户下线时的 Redis 在线信息。
// 移除 connID，如果所有连接都已下线则删除整个 key。
func (h *Hub) setOffline(userID, connID, _ string) {
	ctx := context.Background()
	mapKey := rediskey.ConnMapKey(userID)

	h.rdb.SRem(ctx, mapKey, connID)
	count := h.rdb.SCard(ctx, mapKey).Val()
	if count == 0 {
		h.rdb.Del(ctx, mapKey)
		h.rdb.Del(ctx, rediskey.OnlineKey(userID))
	}
}

// RefreshOnline 刷新用户在线状态 TTL（心跳续期）。
func (h *Hub) RefreshOnline(userID, connID string) {
	ctx := context.Background()
	onlineKey := rediskey.OnlineKey(userID)
	mapKey := rediskey.ConnMapKey(userID)
	ttl := 60 * time.Second

	h.rdb.Expire(ctx, onlineKey, ttl)
	h.rdb.Expire(ctx, mapKey, ttl)
	h.rdb.HSet(ctx, onlineKey, "lastActive", time.Now().Unix())
}
