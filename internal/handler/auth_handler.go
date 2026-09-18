package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/alac/se-go-ws-gateway-2026/internal/model"
	"github.com/alac/se-go-ws-gateway-2026/pkg/auth"
)

// LoginRequest 是登录接口的请求体。
// Password 为客户端提交的明文密码, 仅用于本次比对, 不会记录也不会上报。
type LoginRequest struct {
	UserName string `json:"username" binding:"required"` // 账号名
	Password string `json:"password" binding:"required"` // 明文密码
}

// HandleLoginAuth 返回管理员登录接口的处理器。
//
// 处理流程:
//  1. 解析请求体, JSON 格式错误或账号名/密码缺失返回 400
//  2. 校验凭证, 账号名或密码错误统一返回 401(不区分二者, 避免暴露账号是否存在)
//  3. 签发 token, 失败返回 500
//  4. 成功返回 200, token 放在 data.token 中交由前端保存
//
// 参数:
//   - user: 预置账号凭证, 提供口令校验能力
//   - a: token 签发器
func HandleLoginAuth(user *auth.User, a *auth.Authenticator) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req LoginRequest

		// 1. 解析请求体: 字段缺失或 JSON 格式非法都会在这里失败
		if err := ctx.ShouldBindJSON(&req); err != nil {
			slog.Warn("登录请求参数无效", "error", err)

			ctx.JSON(http.StatusBadRequest, gin.H{
				"code":   model.BizCodeBadRequest,
				"status": "error",
				"error":  "用户名和密码不能为空",
			})

			return
		}

		// 2. 校验凭证: 日志只记录账号名, 绝不记录明文密码
		if !user.VerifyPassword(req.UserName, req.Password) {
			slog.Warn("登录失败", "user", req.UserName, "ip", ctx.ClientIP())

			ctx.JSON(http.StatusUnauthorized, gin.H{
				"code":   model.BizCodeUnauthorized,
				"status": "error",
				"error":  "用户名或密码错误",
			})

			return
		}

		// 3. 签发 token
		token, err := a.GenerateToken(req.UserName)

		if err != nil {
			slog.Error("签发 token 失败", "user", req.UserName, "error", err)

			ctx.JSON(http.StatusInternalServerError, gin.H{
				"code":   model.BizCodeInternalError,
				"status": "error",
				"error":  "签发 token 失败",
			})

			return
		}

		// 4. 返回 token: 前端负责保存, 后续请求放在 Authorization 头中携带
		slog.Info("登录成功", "user", req.UserName, "ip", ctx.ClientIP())

		ctx.JSON(http.StatusOK, gin.H{
			"code":   model.BizCodeSuccess,
			"status": "success",
			"data":   gin.H{"token": token},
		})
	}
}
