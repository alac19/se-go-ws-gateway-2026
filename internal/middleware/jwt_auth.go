package middleware

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/alac/se-go-ws-gateway-2026/internal/model"
	"github.com/alac/se-go-ws-gateway-2026/pkg/auth"
)

// HandleJWTAuth 返回校验 token 的 Gin 中间件。
//
// 处理流程:
//  1. 提取 token: 优先 Authorization 请求头("Bearer <token>"), 其次 query 参数 token
//  2. 调用 Authenticator.ParseToken 校验签名、算法(仅 HS256)与有效期
//  3. 失败时返回 401 并中止请求; 成功时把声明写入 gin.Context, 供后续 handler 使用
//
// 之所以同时支持两种来源, 是因为浏览器的 WebSocket 连接无法自定义请求头,
// /ws 只能把 token 放在 URL 参数中, 而普通 HTTP 接口用请求头更规范。
//
// 参数 a 由 main 在启动时构造, 被所有请求共享, 因此中间件内只读取它。
func HandleJWTAuth(a *auth.Authenticator) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		raw := extractToken(ctx)

		if raw == "" {
			slog.Warn("请求缺少 token", "path", ctx.Request.URL.Path)

			abortUnauthorized(ctx, "缺少 token")

			return
		}

		claims, err := a.ParseToken(raw)

		if err != nil {
			slog.Warn("token 校验失败", "path", ctx.Request.URL.Path, "error", err)

			abortUnauthorized(ctx, "token 无效或已过期")

			return
		}

		// 身份写入上下文
		ctx.Set(auth.ContextKeyClaims, claims)

		slog.Debug("鉴权通过", "path", ctx.Request.URL.Path, "user", claims.Subject)

		ctx.Next()
	}
}

// extractToken 从请求中提取 token 字符串, 取不到时返回空字符串。
// 依次尝试两种来源:
//  1. Authorization 请求头, 期望格式为 "Bearer <token>", 方案名大小写不敏感
//  2. query 参数 token, 形如 /ws?clientId=xxx&roomId=xxx&token=xxx
func extractToken(ctx *gin.Context) string {
	if header := ctx.GetHeader("Authorization"); header != "" {
		// 只切一刀, 避免 token 中本身含有空格时被破坏
		parts := strings.SplitN(header, " ", 2)

		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			if token := strings.TrimSpace(parts[1]); token != "" {
				return token
			}
		}
	}

	return strings.TrimSpace(ctx.Query("token"))
}

// abortUnauthorized 中止请求并返回统一的 401 响应体。
// 直接使用 AbortWithStatusJSON, 一步完成"写响应"和"中止后续 handler",
// 避免先 ctx.JSON 再 ctx.Abort 时漏掉其中一个。
func abortUnauthorized(ctx *gin.Context, message string) {
	ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"code":   model.BizCodeUnauthorized,
		"status": "error",
		"error":  message,
	})
}
