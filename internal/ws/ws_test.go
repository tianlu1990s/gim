package ws

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"

	"github.com/tianlu1990s/gim/internal/config"
)

func TestNewHub(t *testing.T) {
	cfg := config.WebSocketConfig{
		Port:           8081,
		MaxConnPerUser: 5,
		MaxMessageSize: 4096,
		WriteWait:      10 * time.Second,
		PongWait:       60 * time.Second,
		PingPeriod:     30 * time.Second,
	}
	hub := NewHub(nil, cfg)

	if hub == nil {
		t.Fatal("NewHub returned nil")
	}
	if hub.clients == nil {
		t.Error("clients map should not be nil")
	}
	if hub.register == nil {
		t.Error("register channel should not be nil")
	}
	if hub.unregister == nil {
		t.Error("unregister channel should not be nil")
	}
	if hub.push == nil {
		t.Error("push channel should not be nil")
	}
}

func TestHubPushToUser(t *testing.T) {
	cfg := config.WebSocketConfig{
		Port:           8081,
		MaxConnPerUser: 5,
		MaxMessageSize: 4096,
		WriteWait:      10 * time.Second,
		PongWait:       60 * time.Second,
		PingPeriod:     30 * time.Second,
	}
	hub := NewHub(nil, cfg)
	go hub.Run()

	msg := &WSMessage{Type: 101, Data: map[string]any{"text": "hello"}}
	hub.PushToUser("alice", msg)

	// 消息应被 push channel 消费（用户不在线时丢弃）
	// 验证不阻塞、不 panic
}

func TestHubIsOnline(t *testing.T) {
	cfg := config.WebSocketConfig{
		Port:           8081,
		MaxConnPerUser: 5,
		MaxMessageSize: 4096,
		WriteWait:      10 * time.Second,
		PongWait:       60 * time.Second,
		PingPeriod:     30 * time.Second,
	}
	hub := NewHub(nil, cfg)
	// 无 Redis 时 IsOnline 返回 false
	if hub.IsOnline("unknown") {
		t.Error("should not be online without Redis")
	}
}

func TestHubOnlineConnCount(t *testing.T) {
	cfg := config.WebSocketConfig{
		Port:           8081,
		MaxConnPerUser: 5,
		MaxMessageSize: 4096,
		WriteWait:      10 * time.Second,
		PongWait:       60 * time.Second,
		PingPeriod:     30 * time.Second,
	}
	hub := NewHub(nil, cfg)
	if count := hub.OnlineConnCount("alice"); count != 0 {
		t.Errorf("conn count = %d, want 0", count)
	}
}

func TestNewServer(t *testing.T) {
	cfg := config.WebSocketConfig{
		Port:           8081,
		MaxConnPerUser: 5,
		MaxMessageSize: 4096,
		WriteWait:      10 * time.Second,
		PongWait:       60 * time.Second,
		PingPeriod:     30 * time.Second,
	}
	hub := NewHub(nil, cfg)
	srv := NewServer(cfg, hub, nil)

	if srv == nil {
		t.Fatal("NewServer returned nil")
	}
}

func TestWSMessageMarshal(t *testing.T) {
	msg := WSMessage{
		Type:   101,
		ReqID:  "req-123",
		Data:   map[string]any{"text": "hello world"},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded WSMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if decoded.Type != 101 {
		t.Errorf("Type = %d, want 101", decoded.Type)
	}
	if decoded.ReqID != "req-123" {
		t.Errorf("ReqID = %s, want req-123", decoded.ReqID)
	}
}

func TestToJSON(t *testing.T) {
	data := toJSON(map[string]string{"key": "value"})
	if data == nil {
		t.Error("toJSON returned nil")
	}
	if string(data) != `{"key":"value"}` {
		t.Errorf("toJSON = %s, want {\"key\":\"value\"}", string(data))
	}
}

func TestToJSONNil(t *testing.T) {
	data := toJSON(nil)
	// json.Marshal(nil) 返回 "null"
	if string(data) != "null" {
		t.Errorf("toJSON(nil) = %s, want null", string(data))
	}
}

// --- Hub channel operation tests ---

func TestHubRegister(t *testing.T) {
	// Use a non-nil rdb to avoid panicking in setOnline.
	// Point it at an unreachable address so the Redis calls fail silently.
	rdb := redis.NewClient(&redis.Options{
		Addr:         "127.0.0.1:1",
		DialTimeout:  10 * time.Millisecond,
		MaxRetries:   -1,
		PoolSize:     1,
		MinIdleConns: 0,
	})
	defer rdb.Close()

	cfg := config.WebSocketConfig{
		PongWait:       60 * time.Second,
		MaxMessageSize: 4096,
	}
	hub := NewHub(rdb, cfg)
	go hub.Run()

	client := NewClient(hub, nil, "alice", "web")
	hub.register <- client

	// Give the Run goroutine time to process the register
	time.Sleep(50 * time.Millisecond)

	hub.mu.RLock()
	conns, ok := hub.clients["alice"]
	hub.mu.RUnlock()

	if !ok {
		t.Fatal("alice not registered in hub.clients")
	}
	if _, exists := conns[client]; !exists {
		t.Error("client not found in alice's connections")
	}
}

func TestHubPushToNonExistentUser(t *testing.T) {
	cfg := config.WebSocketConfig{}
	hub := NewHub(nil, cfg)

	// Push to a user that does not exist should not panic
	msg := &WSMessage{Type: 1, Data: "test"}
	hub.PushToUser("nonexistent", msg)
}

func TestHubOnlineConnCountNonExistentUser(t *testing.T) {
	cfg := config.WebSocketConfig{}
	hub := NewHub(nil, cfg)

	count := hub.OnlineConnCount("nonexistent")
	if count != 0 {
		t.Errorf("OnlineConnCount = %d, want 0", count)
	}
}

// --- Client.Send tests ---

func TestClientSendFullChannel(t *testing.T) {
	// Create a client with a small send buffer
	hub := NewHub(nil, config.WebSocketConfig{})
	client := &Client{
		hub:      hub,
		conn:     nil,
		send:     make(chan []byte, 2),
		userID:   "alice",
		platform: "web",
		connID:   "conn-1",
	}

	msg := &WSMessage{Type: 1, Data: "payload"}

	// First two sends fill the buffer
	client.Send(msg)
	client.Send(msg)

	// Third send: buffer full, close(c.send) called (does not block)
	client.Send(msg)

	// Drain the remaining 2 items
	for i := 0; i < 2; i++ {
		_, ok := <-client.send
		if !ok {
			t.Fatalf("channel closed prematurely at drain %d", i)
		}
	}

	// Now the channel should be closed (all items drained)
	_, ok := <-client.send
	if ok {
		t.Error("expected channel to be closed after buffer full")
	}
}

// --- handleMessage tests ---

func TestHandleMessageInvalidJSON(t *testing.T) {
	hub := NewHub(nil, config.WebSocketConfig{})
	client := NewClient(hub, nil, "alice", "web")

	// Send invalid JSON
	client.handleMessage([]byte("{invalid json"))

	select {
	case data := <-client.send:
		var msg WSMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if msg.Type != -1 {
			t.Errorf("Type = %d, want -1 (error response)", msg.Type)
		}
	default:
		t.Error("expected error response for invalid JSON")
	}
}

func TestHandleMessageHeartbeat(t *testing.T) {
	// Set up a real WebSocket connection for the heartbeat test.
	// The server side keeps reading to avoid blocking.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// Non-functional Redis client (needed so setOnline/RefreshOnline don't nil-panic)
	rdb := redis.NewClient(&redis.Options{
		Addr:         "127.0.0.1:1",
		DialTimeout:  10 * time.Millisecond,
		MaxRetries:   -1,
		PoolSize:     1,
		MinIdleConns: 0,
	})
	defer rdb.Close()

	hub := NewHub(rdb, config.WebSocketConfig{
		PongWait:       60 * time.Second,
		MaxMessageSize: 4096,
	})
	client := NewClient(hub, conn, "alice", "web")

	// Send heartbeat (type 3)
	raw, _ := json.Marshal(WSMessage{Type: 3, Data: map[string]any{}})
	client.handleMessage(raw)

	// Expect heartbeat ack (type 113)
	select {
	case data := <-client.send:
		var msg WSMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if msg.Type != 113 {
			t.Errorf("Type = %d, want 113 (heartbeat ack)", msg.Type)
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for heartbeat ack")
	}
}

func TestHandleMessageTyping(t *testing.T) {
	hub := NewHub(nil, config.WebSocketConfig{})
	client := NewClient(hub, nil, "alice", "web")

	// Send typing indicator (type 5)
	raw, _ := json.Marshal(WSMessage{Type: 5, Data: map[string]any{
		"conversationId": "single_alice_bob",
		"isTyping":       true,
	}})

	client.handleMessage(raw)

	// Should push a typing notification to the hub's push channel
	select {
	case task := <-hub.push:
		if task.message.Type != 105 {
			t.Errorf("pushed message Type = %d, want 105 (typing notification)", task.message.Type)
		}
	default:
		t.Error("expected push to hub for typing indicator")
	}
}

func TestHandleMessageUnsupportedType(t *testing.T) {
	hub := NewHub(nil, config.WebSocketConfig{})
	client := NewClient(hub, nil, "alice", "web")

	// Send an unsupported message type
	raw, _ := json.Marshal(WSMessage{Type: 999, Data: "payload"})
	client.handleMessage(raw)

	select {
	case data := <-client.send:
		var msg WSMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if msg.Type != -1 {
			t.Errorf("Type = %d, want -1 (error response)", msg.Type)
		}
	default:
		t.Error("expected error response for unsupported type")
	}
}
