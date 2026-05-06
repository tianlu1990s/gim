package ws

import "encoding/json"

// toJSON 将任意值序列化为 JSON 字节，序列化失败返回 nil。
// 仅用于内部消息格式转换，输入保证可序列化。
func toJSON(v any) []byte {
	data, _ := json.Marshal(v)
	return data
}
