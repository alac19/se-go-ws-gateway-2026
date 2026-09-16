package middleware

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/alac/se-go-ws-gateway-2026/pkg/auth"
)

func HandleJWTAuth(a *auth.Authenticator) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		slog.Debug("鉴权中间件")

		// 验 token
		a.ParseToken("token")
	}
}
