package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestLogger 请求日志中间件。记录每个请求的方法、路径、状态码、耗时和客户端 IP。
// 按状态码分级记录：2xx/3xx 用 Info，4xx 用 Warn，5xx 用 Error。
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		method := c.Request.Method
		clientIP := c.ClientIP()

		queryStr := ""
		if query != "" {
			queryStr = "?" + query
		}

		if status >= 500 {
			log.Printf("[GIN] %s | %d | %s | %s %s%s | %v",
				clientIP, status, latency, method, path, queryStr, c.Errors.String())
		} else if status >= 400 {
			log.Printf("[GIN] %s | %d | %s | %s %s%s",
				clientIP, status, latency, method, path, queryStr)
		} else {
			log.Printf("[GIN] %s | %d | %s | %s %s%s",
				clientIP, status, latency, method, path, queryStr)
		}
	}
}
