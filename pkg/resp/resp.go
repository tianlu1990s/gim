package resp

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tianlu1990s/gim/pkg/errcode"
)

// Response 统一 HTTP 响应格式。所有接口返回此结构，HTTP 状态码恒为 200，
// 真正的业务状态由 body.code 承载（0=成功，非0=错误码）。
type Response struct {
	Code   int    `json:"code"`
	Msg    string `json:"msg"`
	Data   any    `json:"data,omitempty"`
	Detail string `json:"detail,omitempty"` // 调试详情，生产环境可通过中间件清空
}

func Success(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response{
		Code: 0,
		Msg:  "success",
		Data: data,
	})
}

// Fail 统一失败响应。自动识别 *errcode.Error 提取业务错误码和消息；
// 普通 error 则返回通用 code=50000。
func Fail(c *gin.Context, err error) {
	if e, ok := err.(*errcode.Error); ok {
		c.JSON(http.StatusOK, Response{
			Code:   e.Code,
			Msg:    e.Message,
			Detail: e.Detail,
		})
		return
	}
	c.JSON(http.StatusOK, Response{
		Code: 50000,
		Msg:  err.Error(),
	})
}

// FailWithStatus 返回带指定 HTTP 状态码的失败响应。
// 用于中间件等需要返回非 200 状态码的场景（如 401 Unauthorized、429 Too Many Requests）。
func FailWithStatus(c *gin.Context, httpStatus int, err error) {
	if e, ok := err.(*errcode.Error); ok {
		c.JSON(httpStatus, Response{
			Code:   e.Code,
			Msg:    e.Message,
			Detail: e.Detail,
		})
		return
	}
	c.JSON(httpStatus, Response{
		Code: 50000,
		Msg:  err.Error(),
	})
}
