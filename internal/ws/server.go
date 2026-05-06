package ws

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/tianlu1990s/gim/internal/config"
	"github.com/tianlu1990s/gim/pkg/jwt"
	"github.com/tianlu1990s/gim/pkg/rediskey"
)

// Server WebSocket HTTP 升级服务器。监听独立端口，处理 HTTP → WS 协议升级，
// 验证 Token，检查连接数限制，创建 Client 并注册到 Hub。
type Server struct {
	cfg    config.WebSocketConfig
	hub    *Hub
	jwtMgr *jwt.JWTManager
}

// upgrader HTTP → WebSocket 协议升级器。
// CheckOrigin 允许所有来源（开发阶段），生产环境需限制具体域名。
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// NewServer 创建 WebSocket 服务器。
func NewServer(cfg config.WebSocketConfig, hub *Hub, jwtMgr *jwt.JWTManager) *Server {
	return &Server{cfg: cfg, hub: hub, jwtMgr: jwtMgr}
}

// Start 启动 WebSocket 监听。
func (s *Server) Start() error {
	http.HandleFunc("/ws", s.handleWebSocket)
	return http.ListenAndServe(fmt.Sprintf(":%d", s.cfg.Port), nil)
}

// handleWebSocket 处理 WebSocket 升级请求。执行流程：
// 1. 从查询参数获取 token 和 platform
// 2. 验证 JWT Token（签名 + 有效期）
// 3. 检查 Token 是否在黑名单中（登出后不可用）
// 4. 检查同用户连接数限制（max 5）
// 5. HTTP → WS 协议升级
// 6. 创建 Client 并注册到 Hub
// 7. 启动读写协程
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	platform := r.URL.Query().Get("platform")
	if platform == "" {
		platform = "web"
	}

	// 验证 Token
	claims, err := s.jwtMgr.ParseToken(token)
	if err != nil {
		http.Error(w, "Unauthorized: invalid or expired token", http.StatusUnauthorized)
		return
	}

	// 检查 Token 黑名单
	ctx := r.Context()
	exists, _ := s.hub.rdb.Exists(ctx, rediskey.BlacklistTokenKey(claims.ID)).Result()
	if exists > 0 {
		http.Error(w, "Unauthorized: token has been revoked", http.StatusUnauthorized)
		return
	}

	// 检查连接数限制 — 同用户最多 MaxConnPerUser 个连接
	s.hub.mu.RLock()
	connCount := len(s.hub.clients[claims.UserID])
	s.hub.mu.RUnlock()
	if connCount >= s.cfg.MaxConnPerUser {
		http.Error(w, "Too many connections", http.StatusTooManyRequests)
		return
	}

	// 升级 HTTP → WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	// 创建 Client 并注册到 Hub
	client := NewClient(s.hub, conn, claims.UserID, platform)
	s.hub.register <- client

	// 启动读写协程
	go client.WritePump()
	go client.ReadPump()
}
