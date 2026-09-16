package handler

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/alac/se-go-ws-gateway-2026/pkg/auth"
)

func HandleLoginAuth(user *auth.User, a *auth.Authenticator) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		slog.Debug("登录路由处理...")
		// 获取密码
		name := "alac"
		password := "123456"

		// 1. 验证密码
		user.VerifyPassword(name, password)

		// 2. 返回 token
		a.GenerateToken(name)
	}
}
