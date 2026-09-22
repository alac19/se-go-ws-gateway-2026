// Package main is the entry point for the WebSocket gateway service.
// It initializes configuration, logging, metrics, and starts both HTTP and WebSocket servers.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/time/rate"

	"github.com/alac/se-go-ws-gateway-2026/internal/config"
	"github.com/alac/se-go-ws-gateway-2026/internal/handler"
	"github.com/alac/se-go-ws-gateway-2026/internal/middleware"
	"github.com/alac/se-go-ws-gateway-2026/internal/service"
	"github.com/alac/se-go-ws-gateway-2026/pkg/auth"
	"github.com/alac/se-go-ws-gateway-2026/pkg/limiter"
	"github.com/alac/se-go-ws-gateway-2026/pkg/logger"
	"github.com/alac/se-go-ws-gateway-2026/pkg/metrics"
)

func main() {
	// 记录服务启动时间，用于统计接口计算运行时长
	serverInitTime := time.Now()

	cfg, err := config.LoadConfig("configs/config.toml")

	if err != nil {
		slog.Error("加载配置失败", "error", err)
		os.Exit(1)
	}

	if err := logger.Init(cfg.Log.Level, cfg.Log.FilePath); err != nil {
		slog.Error("初始化日志失败", "error", err)
		os.Exit(1)
	}

	slog.Info("日志系统初始化成功", "log_level", cfg.Log.Level, "file", cfg.Log.FilePath)

	r := gin.New()
	r.Use(gin.Recovery())

	// 跨域处理放在最前面: 浏览器的 OPTIONS 预检请求不带 Authorization 头,
	// 必须在限流与鉴权中间件之前直接返回, 否则前端会被判定为跨域失败。
	r.Use(middleware.HandleCORS())

	// 不信任任何代理: 网关直接对外暴露时, ClientIP() 只取 TCP 连接的对端地址,
	// 从而忽略客户端自行伪造的 X-Forwarded-For / X-Real-IP, 避免限流被绕过。
	// 若将来在网关前部署反向代理, 这里应改为代理的 IP 或网段, 再由代理传递真实 IP。
	if err := r.SetTrustedProxies(nil); err != nil {
		slog.Error("设置可信代理失败", "error", err)
		os.Exit(1)
	}

	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())
	quit := make(chan os.Signal, 2)

	signal.Notify(quit, os.Interrupt, syscall.SIGTERM) // 监听信号

	metrics.Init(nil)

	// 1. 初始化核心层（顺序：RoomManager -> ClientManager -> MessageRouter）
	roomMgr := service.NewRoomManager()
	clientMgr := service.NewClientManager(roomMgr, cfg.Channel.RegisterBufferSize, cfg.Channel.UnregisterBufferSize)
	router := service.NewMessageRouter(clientMgr, roomMgr, nil)
	lm := limiter.NewLimiterMap(rate.Every(cfg.RateLimitInterval()), cfg.Ratelimit.Burst)
	md1 := middleware.HandleRateLimit(lm)
	a := auth.NewAuthenticator(cfg.Jwt.Secret, cfg.TokenTTL())
	user := auth.NewUser(cfg.Auth.UserName, cfg.Auth.PasswordHash)
	md2 := middleware.HandleJWTAuth(a)

	// 2. 登录接口
	public := r.Group("/api", md1)
	hd5 := handler.HandleLoginAuth(user, a)
	public.POST("/auth/login", hd5)

	api := r.Group("/api", md1, md2)

	// 3. 启动 ClientManager 后台循环（处理 register/unregister 事件）
	go clientMgr.Init(ctx, cfg.ControlWriteTimeout())

	// 4. WebSocket 路由，传入 clientMgr
	hd := handler.HandlerConnManagement(clientMgr, ctx, &wg, cfg)
	r.GET("/ws", md2, hd)

	// 5. 推送类接口，传入 messageRouter
	hd1 := handler.HandleBroadcast(router)
	api.POST("/broadcast", hd1)

	hd2 := handler.HandleRoomBroadcast(router, roomMgr)
	api.POST("/room/:roomId/broadcast", hd2)

	hd3 := handler.HandleClientSend(router)
	api.POST("/client/:clientId/send", hd3)

	// 6. 统计接口，传入 clientMgr
	hd4 := handler.HandleStats(clientMgr, roomMgr, serverInitTime)
	api.GET("/stats", hd4)

	// 7. /metrics 端点
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// 8. 健康检查端点
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	go func() {
		if err := http.ListenAndServe(":6060", nil); err != nil {
			// pprof 仅用于调试, 启动失败不影响网关对外提供服务, 因此降级为 Warn
			slog.Warn("pprof 服务启动失败, 网关主服务不受影响", "error", err)
		}
	}()

	httpSvr := http.Server{Addr: fmt.Sprintf(":%d", cfg.Server.Port), Handler: r}

	// 先同步监听端口, 让"端口被占用"这类错误在启动阶段就以 Error 暴露并退出,
	// 否则监听失败只会在后台静默发生, 进程照常运行却无法对外服务。
	ln, err := net.Listen("tcp", httpSvr.Addr)

	if err != nil {
		slog.Error("HTTP 服务监听失败", "addr", httpSvr.Addr, "error", err)
		os.Exit(1)
	}

	slog.Info("网关服务启动完成", "addr", httpSvr.Addr, "pprof", ":6060")

	go func() {
		// 关闭时的 ErrServerClosed 属于正常退出, 不作为错误记录
		if err := httpSvr.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP 服务异常退出", "error", err)
			os.Exit(1)
		}
	}()

	if s := <-quit; s != nil { // 收到信号
		slog.Info("收到终止信号，开始优雅退出")

		cancel() // 传递上下文

		if err := httpSvr.Shutdown(ctx); err != nil { // 关闭 HTTP Server
			slog.Error("HTTP 服务器关闭失败", "error", err)
		}

		clientMgr.Shutdown(cfg.ShutdownTimeout(), cfg.ControlWriteTimeout())
		wg.Wait()

		slog.Info("网关已退出")
	}
}
