package ws

// WSMessage WebSocket 推送消息结构。
// Type 定义消息类型（见 API.md），ReqID 用于请求-响应匹配，Data 为具体内容。
type WSMessage struct {
	Type  int    `json:"type"`
	ReqID string `json:"reqId,omitempty"` // 请求ID，用于请求-响应匹配
	Data  any    `json:"data"`
}
