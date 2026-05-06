// Package middleware 提供 Gin 中间件集合。
//
// 中间件列表：
//   - auth.go: JWTAuth — JWT 鉴权（Token 验证 + 黑名单检查）
//   - cors.go: CORS — 跨域资源共享
//   - recovery.go: Recovery — Panic 恢复，防止进程崩溃
//   - logger.go: RequestLogger — 请求日志记录
//   - ratelimit.go: RateLimit — 基于 Redis 的限流
package middleware
