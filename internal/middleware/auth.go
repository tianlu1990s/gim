package middleware

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/tianlu1990s/gim/pkg/errcode"
	"github.com/tianlu1990s/gim/pkg/jwt"
	"github.com/tianlu1990s/gim/pkg/rediskey"
	"github.com/tianlu1990s/gim/pkg/resp"
)

// JWTAuth JWT 鉴权中间件。处理流程：
// 1. 从 Authorization header 提取 Bearer token
// 2. 用公钥验证 RS256 签名和有效期
// 3. 检查 token JTI 是否在 Redis 黑名单中（登出后立即失效）
// 4. 验证通过后将 userId 和 platform 注入 gin.Context
func JWTAuth(jwtMgr *jwt.JWTManager, rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			resp.Fail(c, errcode.ErrUnauthorized.WithDetail("缺少认证信息"))
			c.Abort()
			return
		}
		tokenStr := authHeader[7:] // 去掉 "Bearer " 前缀

		// 解析并验证 token（签名 + 有效期），RS256 非对称加密
		claims, err := jwtMgr.ParseToken(tokenStr)
		if err != nil {
			resp.Fail(c, errcode.ErrUnauthorized.WithDetail("无效或过期的 Token"))
			c.Abort()
			return
		}

		// 检查 token JTI 是否在黑名单中（logout 后加入，TTL = 剩余有效期）
		if rdb != nil {
			blacklisted, err := rdb.Exists(context.Background(), rediskey.BlacklistTokenKey(claims.ID)).Result()
			if err == nil && blacklisted > 0 {
				resp.Fail(c, errcode.ErrUnauthorized.WithDetail("Token 已被注销"))
				c.Abort()
				return
			}
		}

		// 注入用户信息到 Context，下游 Handler 通过 c.GetString 获取
		c.Set("userID", claims.UserID)
		c.Set("platform", claims.Platform)
		c.Next()
	}
}
