// Package middleware provides HTTP middleware based on the Gin framework.
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HandleCORS 返回处理跨域请求的中间件, 用于前端开发服务器(如 Vite、Live Server)
// 从其他源(不同端口或 file://)调用网关接口。
//
// 处理逻辑:
//  1. 回显请求的 Origin, 并声明允许的方法与请求头(包含 Authorization)
//  2. 对浏览器的 OPTIONS 预检请求直接返回 204 并中止, 不再进入限流与鉴权中间件。
//     预检请求不会携带 Authorization 头, 若继续往下走会被鉴权中间件以 401 拒绝,
//     浏览器随即判定跨域失败, 因此这一步必须短路。
//  3. 其余请求放行, 交由后续中间件与业务处理函数继续处理
//
// 注意: 此处回显任意来源是为了便于课程开发与前后端联调, 生产环境应改为来源白名单。
// WebSocket 不受同源策略约束, 因此无需在此处理。
func HandleCORS() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		origin := ctx.GetHeader("Origin")

		if origin == "" {
			origin = "*"
		}

		header := ctx.Writer.Header()
		header.Set("Access-Control-Allow-Origin", origin)
		header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		header.Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		header.Set("Access-Control-Max-Age", "86400")
		header.Add("Vary", "Origin")

		// 浏览器预检请求: 直接返回 204, 不进入后续中间件
		if ctx.Request.Method == http.MethodOptions {
			ctx.AbortWithStatus(http.StatusNoContent)

			return
		}

		ctx.Next()
	}
}
