// Package ws 提供 WebSocket 连接管理和消息推送能力。
//
// 核心组件：
//   - Hub: 连接中心，维护用户→连接映射，负责推送路由和在线状态管理
//   - Client: 单个 WebSocket 连接的抽象，包含读写协程
//   - Server: HTTP → WebSocket 升级服务器，处理鉴权和连接数限制
//
// 消息流：
//   Service (推送) → Hub.PushToUser → Client.Send → Client.WritePump → WebSocket → 客户端
//   客户端 → WebSocket → Client.ReadPump → handleMessage (心跳/输入状态)
//
// 并发模型：
//   所有 clients map 的修改都通过 channel 提交到 Hub.Run() 的 goroutine 中处理，
//   遵循 Go 的 CSP 模式——"通过通信来共享内存，而非通过共享内存来通信"。
package ws
