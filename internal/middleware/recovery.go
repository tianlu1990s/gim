package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"

	"github.com/tianlu1990s/gim/pkg/errcode"
	"github.com/tianlu1990s/gim/pkg/resp"
)

// Recovery panic 恢复中间件。捕获 Handler 中发生的 panic，
// 打印堆栈信息，返回统一错误响应，防止单个请求的 panic 导致整个进程崩溃。
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				// 打印完整堆栈信息，便于排查问题
				debug.PrintStack()
				// 返回统一错误响应（HTTP 500），防止前端收到空响应
				resp.FailWithStatus(c, http.StatusInternalServerError, errcode.ErrInternal.WithDetail("服务器内部错误"))
				c.Abort()
			}
		}()
		c.Next()
	}
}
