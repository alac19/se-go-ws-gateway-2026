// Package middleware provides HTTP middleware based on the Gin framework.
package middleware

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/alac/se-go-ws-gateway-2026/internal/model"
	"github.com/alac/se-go-ws-gateway-2026/pkg/limiter"
)

// HandleRateLimit 返回一个 Gin 中间件处理函数, 用于对请求进行限流。
// 参数 lm 是实现了 limiter.Limiter 接口的限流器实例。
// 如果 lm.Allow(ip) 返回 true（表示应限流）, 则中间件返回 429 状态码并终止请求；
// 否则调用 ctx.Next() 继续处理后续的 HTTP 处理函数。
// 触发限流时的响应体与其他接口保持一致, 统一使用 code/status/error 三个字段。
func HandleRateLimit(lm limiter.Limiter) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ip := ctx.ClientIP()

		if lm.Allow(ip) {
			slog.Warn("请求触发限流", "ip", ip, "path", ctx.Request.URL.Path)

			ctx.JSON(http.StatusTooManyRequests, gin.H{
				"code":   model.BizCodeTooManyRequests,
				"status": "error",
				"error":  "请求过于频繁, 请稍后重试",
			})

			ctx.Abort()

			return
		}

		ctx.Next()
	}
}
