package middleware

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/tianlu1990s/gim/pkg/errcode"
	"github.com/tianlu1990s/gim/pkg/rediskey"
	"github.com/tianlu1990s/gim/pkg/resp"
)

// RateLimit 限流中间件。使用 Redis 滑动窗口计数，同一用户/IP 在窗口内超过限制则拒绝请求。
//
// 限流策略：
//   - 优先使用登录用户 ID 作为限流 Key（认证后）
//   - 未登录用户使用客户端 IP
//   - Redis Key: ratelimit:{userId} 或 ratelimit:{ip}
//   - 首次请求时设置 TTL（窗口过期后自动重置计数）
func RateLimit(rdb *redis.Client, rate int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 优先使用登录用户 ID，未登录使用 IP
		userID := c.GetString("userID")
		if userID == "" {
			userID = c.ClientIP()
		}

		ctx := context.Background()
		key := rediskey.RateLimitKey(userID)

		count, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			// Redis 不可用时限流降级为放行，避免影响正常业务
			c.Next()
			return
		}
		// 首次请求设置窗口过期时间
		if count == 1 {
			rdb.Expire(ctx, key, window)
		}

		if count > int64(rate) {
			resp.Fail(c, errcode.ErrRateLimitExceeded.WithDetail("请求过于频繁，请稍后再试"))
			c.Abort()
			return
		}
		c.Next()
	}
}
