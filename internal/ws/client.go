package ws

import (
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"

	"github.com/tianlu1990s/gim/pkg/convutil"
	"github.com/tianlu1990s/gim/pkg/snowflake"
)

// Client 代表单个 WebSocket 连接。每个连接绑定到一个用户/平台，
// 包含读写协程（ReadPump/WritePump），通过 Hub 与其他连接通信。
type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte       // 待发送消息队列，缓冲 256
	userID   string            // 连接所属用户
	platform string            // 客户端平台：web/ios/android
	connID   string            // 连接唯一标识（Snowflake ID）
}

// NewClient 创建客户端连接。connID 由 Snowflake 生成，保证唯一。
func NewClient(hub *Hub, conn *websocket.Conn, userID, platform string) *Client {
	return &Client{
		hub:      hub,
		conn:     conn,
		send:     make(chan []byte, 256),
		userID:   userID,
		platform: platform,
		connID:   snowflake.Generate().String(),
	}
}

// ReadPump 从 WebSocket 连接读取消息，在 goroutine 中运行。
// 收到消息后分发到对应的处理逻辑（心跳、输入状态等）。
// 连接断开时自动从 Hub 注销。
func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(c.hub.cfg.MaxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(c.hub.cfg.PongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(c.hub.cfg.PongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			break // 连接断开
		}
		c.handleMessage(message)
	}
}

// WritePump 向 WebSocket 连接写消息，在 goroutine 中运行。
// 同时负责定期发送 Ping 保持连接活跃。
func (c *Client) WritePump() {
	ticker := time.NewTicker(c.hub.cfg.PingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(c.hub.cfg.WriteWait))
			if !ok {
				// send channel 已关闭，发送关闭帧
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			// 定期发送 Ping 保活
			c.conn.SetWriteDeadline(time.Now().Add(c.hub.cfg.WriteWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Send 向客户端发送消息（非阻塞）。消息 JSON 序列化后写入 send channel，
// 由 WritePump 实际发送。channel 满时丢弃消息并关闭连接。
func (c *Client) Send(msg *WSMessage) {
	data, _ := json.Marshal(msg)
	select {
	case c.send <- data:
	default:
		// send channel 满，关闭连接避免内存泄漏
		close(c.send)
	}
}

// handleMessage 处理客户端发来的 WS 消息分发。
// Phase 1 支持：Type 3=心跳，Type 5=输入状态。
// 消息发送/已读/历史通过 HTTP API 完成，不通过 WS。
func (c *Client) handleMessage(raw []byte) {
	var msg WSMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		c.Send(&WSMessage{Type: -1, Data: map[string]any{"error": "invalid message format"}})
		return
	}

	switch msg.Type {
	case 3: // 心跳 — 刷新在线状态
		c.conn.SetReadDeadline(time.Now().Add(c.hub.cfg.PongWait))
		c.hub.RefreshOnline(c.userID, c.connID)
		// 回复心跳确认
		c.Send(&WSMessage{Type: 113, Data: map[string]any{}})

	case 5: // 输入状态 — 推送给对方
		data, ok := msg.Data.(map[string]any)
		if !ok {
			return
		}
		convID, _ := data["conversationId"].(string)
		isTyping, _ := data["isTyping"].(bool)
		// 单聊场景下从 convID 提取对方 userId
		targetID := convutil.ExtractTargetID(convID, c.userID)
		c.hub.PushToUser(targetID, &WSMessage{
			Type: 105,
			Data: map[string]any{
				"conversationId": convID,
				"userId":         c.userID,
				"isTyping":       isTyping,
			},
		})

	default:
		// Phase 1 仅支持以上类型，其他类型通过 HTTP API 处理
		c.Send(&WSMessage{Type: -1, Data: map[string]any{"error": "unsupported message type, use HTTP API"}})
	}
}
