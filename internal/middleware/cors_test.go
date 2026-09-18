package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHandleCORS(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name            string
		method          string
		origin          string
		wantStatus      int
		wantAllowOrigin string
	}{
		{"普通请求回显来源", http.MethodGet, "http://localhost:5173", http.StatusOK, "http://localhost:5173"},
		{"预检请求直接放行", http.MethodOptions, "http://localhost:5173", http.StatusNoContent, "http://localhost:5173"},
		{"无来源时回退通配", http.MethodGet, "", http.StatusOK, "*"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := gin.New()

			r.Use(HandleCORS())
			r.GET("/test", func(ctx *gin.Context) { ctx.String(http.StatusOK, "ok") })
			r.OPTIONS("/test", func(ctx *gin.Context) { ctx.String(http.StatusOK, "业务处理函数不应被执行") })

			req := httptest.NewRequest(test.method, "/test", nil)

			if test.origin != "" {
				req.Header.Set("Origin", test.origin)
			}

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != test.wantStatus {
				t.Errorf("状态码错: got %d, want %d", rec.Code, test.wantStatus)
			}

			if got := rec.Header().Get("Access-Control-Allow-Origin"); got != test.wantAllowOrigin {
				t.Errorf("Allow-Origin 错误: got %q, want %q", got, test.wantAllowOrigin)
			}

			if got := rec.Header().Get("Access-Control-Allow-Headers"); got == "" {
				t.Errorf("缺少 Access-Control-Allow-Headers 响应头")
			}
		})
	}
}
