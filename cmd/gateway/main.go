package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/time/rate"

	config "github.com/alac/se-go-ws-gateway-2026/internal/config"
	handler "github.com/alac/se-go-ws-gateway-2026/internal/handler"
	ratelimit "github.com/alac/se-go-ws-gateway-2026/internal/middleware"
	service "github.com/alac/se-go-ws-gateway-2026/internal/service"
	limiter "github.com/alac/se-go-ws-gateway-2026/pkg/limiter"
	logger "github.com/alac/se-go-ws-gateway-2026/pkg/logger"
)

func main() {
	serverInitTime := time.Now()

	cfg, err := config.LoadConfig("configs/config.toml")

	if err != nil {
		slog.Error("加载配置失败", "error", err)
		os.Exit(1)
	}

	cfg.ApplyEnvOverrides()

	if err := logger.Init(cfg.Log.Level, cfg.Log.FilePath); err != nil {
		slog.Error("初始化日志失败", "error", err)
		os.Exit(1)
	}

	slog.Info("日志系统初始化成功", "level", cfg.Log.Level, "file", cfg.Log.FilePath)

	r := gin.Default()

	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())
	quit := make(chan os.Signal, 2)

	signal.Notify(quit, os.Interrupt, syscall.SIGTERM) // 监听信号

	// 1. 初始化核心层（顺序：RoomManager -> ClientManager -> MessageRouter）
	roomMgr := service.NewRoomManager()
	clientMgr := service.NewClientManager(roomMgr, cfg.Channel.RegisterBufferSize, cfg.Channel.UnregisterBufferSize)
	router := service.NewMessageRouter(clientMgr, roomMgr, nil)
	lm := limiter.NewLimiterMap(rate.Every(cfg.RateLimitInterval()), cfg.Ratelimit.Burst)
	md1 := ratelimit.HandleRateLimit(lm)

	api := r.Group("/api", md1)

	// 2. 启动 ClientManager 后台循环（处理 register/unregister 事件）
	go clientMgr.Init(cfg.ControlWriteTimeout())

	// 2. WebSocket 路由，传入 clientMgr
	hd := handler.HandlerConnManagement(clientMgr, ctx, &wg, cfg)
	r.GET("/ws", hd)
	fmt.Println("路由注册成功！")

	// 3. 推送类接口，传入 messageRouter
	hd1 := handler.HandleBroadcast(router)
	api.POST("/broadcast", hd1)
	fmt.Println("路由注册成功！")

	hd2 := handler.HandleRoomBroadcast(router, roomMgr)
	api.POST("/room/:roomId/broadcast", hd2)
	fmt.Println("路由注册成功！")

	hd3 := handler.HandleClientSend(router)
	api.POST("/client/:clientId/send", hd3)
	fmt.Println("路由注册成功！")

	// 4. 统计接口，传入 clientMgr
	hd4 := handler.HandleStats(clientMgr, roomMgr, serverInitTime)
	api.GET("/stats", hd4)
	fmt.Println("路由注册成功！")

	// 5. /metrics 端点
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	fmt.Println("路由注册成功！")

	// 6. 健康检查端点
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	httpSvr := http.Server{Addr: fmt.Sprintf(":%d", cfg.Server.Port), Handler: r}

	go httpSvr.ListenAndServe()

	if s := <-quit; s != nil { // 收到信号
		slog.Info("收到终止信号，开始优雅退出")

		cancel() // 传递上下文

		if err := httpSvr.Shutdown(ctx); err != nil { // 关闭 HTTP Server
			slog.Error("HTTP 服务器关闭失败", "error", err)
		}

		log.Printf("优雅退出，宽限期: %v", cfg.ShutdownTimeout())
		clientMgr.Shutdown(cfg.ShutdownTimeout(), cfg.ControlWriteTimeout())
		wg.Wait()
	}
}
