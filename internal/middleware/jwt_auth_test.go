package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/alac/se-go-ws-gateway-2026/pkg/auth"
)

func TestExtractToken(t *testing.T) {
	tests := []struct {
		name   string
		header string
		query  string
		want   string
	}{
		{"请求头携带 Bearer 令牌", "Bearer abc.def.ghi", "", "abc.def.ghi"},
		{"方案名小写", "bearer abc.def.ghi", "", "abc.def.ghi"},
		{"方案名大写", "BEARER abc.def.ghi", "", "abc.def.ghi"},
		{"令牌前后有多余空格", "Bearer   abc.def.ghi  ", "", "abc.def.ghi"},
		{"方案名不正确时回退到 query", "Token abc.def.ghi", "query-token", "query-token"},
		{"Bearer 后为空时回退到 query", "Bearer   ", "query-token", "query-token"},
		{"仅 query 携带令牌", "", "query-token", "query-token"},
		{"query 令牌前后有空格", "", "  query-token  ", "query-token"},
		{"两者都有时优先使用请求头", "Bearer header-token", "query-token", "header-token"},
		{"两者都没有", "", "", ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest(http.MethodGet, "/ws?token="+url.QueryEscape(test.query), nil)

			if test.header != "" {
				ctx.Request.Header.Set("Authorization", test.header)
			}

			if got := extractToken(ctx); got != test.want {
				t.Errorf("extractToken() 期望 %q, 实际得到 %q", test.want, got)
			}
		})
	}
}

func TestHandleJWTAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	authenticator := auth.NewAuthenticator("test-secret-for-unit-test", time.Hour)

	validToken, err := authenticator.GenerateToken("zhangsan")

	if err != nil {
		t.Fatalf("生成测试令牌失败: %v", err)
	}

	tests := []struct {
		name        string
		header      string
		query       string
		wantStatus  int
		wantCode    int
		wantSubject string
		wantErrMsg  string
	}{
		{"未携带令牌", "", "", http.StatusUnauthorized, 401, "", "缺少 token"},
		{"令牌格式非法", "Bearer not-a-jwt", "", http.StatusUnauthorized, 401, "", "token 无效或已过期"},
		{"请求头携带有效令牌", "Bearer " + validToken, "", http.StatusOK, 0, "zhangsan", ""},
		{"query 携带有效令牌", "", validToken, http.StatusOK, 0, "zhangsan", ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := gin.New()

			// 末尾的处理函数用于观察中间件是否放行, 以及上下文中的身份是否写入成功
			r.GET("/test", HandleJWTAuth(authenticator), func(ctx *gin.Context) {
				value, exists := ctx.Get(auth.ContextKeyClaims)

				if !exists {
					ctx.String(http.StatusOK, "缺少上下文身份")

					return
				}

				claims, ok := value.(auth.Claims)

				if !ok {
					ctx.String(http.StatusOK, "上下文身份类型异常")

					return
				}

				ctx.String(http.StatusOK, claims.Subject)
			})

			req := httptest.NewRequest(http.MethodGet, "/test?token="+url.QueryEscape(test.query), nil)

			if test.header != "" {
				req.Header.Set("Authorization", test.header)
			}

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != test.wantStatus {
				t.Errorf("状态码错: got %d, want %d", rec.Code, test.wantStatus)
			}

			if test.wantStatus == http.StatusOK {
				if got := rec.Body.String(); got != test.wantSubject {
					t.Errorf("上下文中的账号名期望 %q, 实际得到 %q", test.wantSubject, got)
				}

				return
			}

			var body struct {
				Code   int    `json:"code"`
				Status string `json:"status"`
				Error  string `json:"error"`
			}

			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("解析响应体失败: %v, 原始内容: %s", err, rec.Body.String())
			}

			if body.Code != test.wantCode {
				t.Errorf("业务码期望 %d, 实际得到 %d", test.wantCode, body.Code)
			}

			if body.Status != "error" {
				t.Errorf("响应状态期望 %q, 实际得到 %q", "error", body.Status)
			}

			if body.Error != test.wantErrMsg {
				t.Errorf("错误信息期望 %q, 实际得到 %q", test.wantErrMsg, body.Error)
			}
		})
	}
}
