// Package handler provides HTTP and WebSocket request handlers for the gateway.
// It contains handlers for WebSocket connection management, message broadcasting,
// room-based messaging, unicast messaging, and system statistics.
package handler

import (
	"context"
	"log/slog"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	config "github.com/alac/se-go-ws-gateway-2026/internal/config"
	model "github.com/alac/se-go-ws-gateway-2026/internal/model"
	service "github.com/alac/se-go-ws-gateway-2026/internal/service"
	metrics "github.com/alac/se-go-ws-gateway-2026/pkg/metrics"
)

// validIDPattern 用于校验 clientId 和 roomId 的合法字符集。
// 仅允许字母、数字、下划线和短横线, 防止注入攻击。
var validIDPattern = regexp.MustCompile("^[a-zA-Z0-9_-]+$")

// HandlerConnManagement WebSocket 接入处理器。
// 负责将 HTTP 请求升级为 WebSocket 连接, 并进行参数校验、重复连接检查、客户端注册，
// 以及启动读写协程（writePump / readPump）进行消息处理和心跳保活。
//
// 参数:
//   - clientMgr: 连接管理器, 用于注册/注销客户端
//   - ctx: 上下文, 用于通知协程退出
//   - wg: 等待组, 用于追踪读写协程的生命周期
//   - cfg: 配置对象, 提供 WebSocket 超时、缓冲区大小等参数
func HandlerConnManagement(clientMgr *service.ClientManager, ctx context.Context, wg *sync.WaitGroup, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		upgrader := websocket.Upgrader{
			CheckOrigin:     func(r *http.Request) bool { return true }, // 允许跨域请求（生产环境可根据需要限制）
			ReadBufferSize:  cfg.Websocket.ReadBufferSize,
			WriteBufferSize: cfg.Websocket.WriteBufferSize,
		}

		// 升级 HTTP 为 WebSocket 连接
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)

		if err != nil {
			slog.Error("WebSocket 升级失败", "error", err)

			c.JSON(http.StatusInternalServerError, gin.H{
				"code":   model.BizCodeInternalError,
				"status": "error",
				"error":  "WebSocket 协议升级失败",
			})

			return
		}

		// 若网关正在优雅退出, 拒绝新连接并返回 1001 状态码（Going Away）
		if clientMgr.IsShuttingDown() {
			err := conn.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseGoingAway, "service is shutting down"),
				time.Now().Add(cfg.ControlWriteTimeout()))

			if err != nil {
				slog.Error("发送关闭帧失败", "error", err)
			}

			_ = conn.Close()

			return
		}

		// 校验查询参数
		clientID := c.Query("clientId")
		roomID := c.Query("roomId")

		if clientID == "" || roomID == "" {
			slog.Warn("参数缺失", "clientId", clientID, "roomId", roomID)

			err := conn.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(model.CloseCodeMissingParam, "clientId and roomId are required"),
				time.Now().Add(cfg.ControlWriteTimeout()))

			if err != nil {
				slog.Error("发送关闭帧失败", "error", err)
			}

			_ = conn.Close()

			return
		}

		if !validIDPattern.MatchString(clientID) {
			slog.Warn("clientId 包含非法字符", "clientId", clientID)

			err := conn.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(model.CloseCodeInvalidFormat, "invalid clientId format"),
				time.Now().Add(cfg.ControlWriteTimeout()))

			if err != nil {
				slog.Error("发送关闭帧失败", "error", err)
			}

			_ = conn.Close()

			return
		}

		if !validIDPattern.MatchString(roomID) {
			slog.Warn("roomID 包含非法字符", "roomID", roomID)

			err := conn.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(model.CloseCodeInvalidFormat, "invalid roomID format"),
				time.Now().Add(cfg.ControlWriteTimeout()))

			if err != nil {
				slog.Error("发送关闭帧失败", "error", err)
			}

			_ = conn.Close()

			return
		}

		if _, res := clientMgr.Get(clientID); res {
			slog.Warn("clientId 已存在, 拒绝连接", "clientId", clientID)

			err := conn.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(model.CloseCodeDuplicateID, "clientId already exists"),
				time.Now().Add(cfg.ControlWriteTimeout()))

			if err != nil {
				slog.Error("发送关闭帧失败", "error", err)
			}

			_ = conn.Close()

			return
		}

		// 设置初始读超时（Pong 响应窗口 60s）
		_ = conn.SetReadDeadline(time.Now().Add(cfg.ReadDeadline()))

		// 创建客户端并注册到连接管理器
		client := model.NewClient(clientID, roomID, conn, cfg.Channel.SendBufferSize, time.Now())
		clientMgr.Register(client)

		// 设置 Pong 处理器, 收到 Pong 时延长读超时并更新 LastPong
		conn.SetPongHandler(func(appData string) error {
			client.LastPong = time.Now()

			return conn.SetReadDeadline(time.Now().Add(cfg.ReadDeadline()))
		})

		// 启动读写协程（每个连接独享）
		wg.Add(2)
		go writePump(client, ctx, wg, cfg.PingInterval(), cfg.WriteDeadline(), cfg.PingWriteTimeout())
		go readPump(client, clientMgr, wg, cfg.PongWait(), cfg.ControlWriteTimeout())
	}
}

// writePump 负责从客户端的 SendChan 通道读取消息并写入 WebSocket 连接。
// 同时定期发送 Ping 帧以维持心跳保活。
//
// 退出条件：
//   - SendChan 被关闭（客户端已注销）
//   - 写入失败（连接已断开）
//   - 收到 ctx.Done() 信号（服务正在关闭）
//
// 参数:
//   - client: 客户端对象
//   - ctx: 上下文, 用于协程退出
//   - wg: 等待组, 退出时调用 Done()
//   - pingInterval: Ping 帧发送间隔
//   - writeDeadline: 写入超时
//   - pingWriteTimeout: Ping 帧写入超时
func writePump(client *model.Client, ctx context.Context, wg *sync.WaitGroup, pingInterval, writeDeadline, pingWriteTimeout time.Duration) {
	ticker := time.NewTicker(pingInterval)
	defer func() {
		if p := recover(); p != nil {
			slog.Error("panic 恢复", "error", p)
		}

		wg.Done()
		ticker.Stop()
	}()

	for {
		select {
		case msg, ok := <-client.SendChan:
			if !ok {
				return
			}

			_ = client.Conn.SetWriteDeadline(time.Now().Add(writeDeadline))
			client.Lock()
			err := client.Conn.WriteMessage(websocket.TextMessage, msg)
			client.Unlock()

			if err != nil {
				slog.Error("writePump 写入失败", "clientId", client.ClientID, "error", err)
				return
			}
		case <-ticker.C:
			_ = client.Conn.SetWriteDeadline(time.Now().Add(writeDeadline))
			err := client.Conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(pingWriteTimeout))

			if err != nil {
				slog.Error("发送 ping 帧失败", "error", err)
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

// readPump 负责从 WebSocket 连接读取消息, 并检测连接状态和心跳超时。
//
// 职责：
//   - 持续调用 ReadMessage() 读取客户端消息（用于维持连接活跃）
//   - 监测 Pong 超时：若 LastPong 距离当前时间超过 pongWait, 则认为连接失活
//   - 连接异常断开时，通过 ClientManager.Unregister() 通知注销
//   - 若网关正在优雅退出（IsShuttingDown() == true）, 则不发送关闭帧，直接注销
//
// 参数:
//   - client: 客户端对象
//   - clientMgr: 连接管理器
//   - wg: 等待组，退出时调用 Done()
//   - pongWait: Pong 响应等待超时
//   - controlWriteTimeout: 发送关闭帧的超时
func readPump(client *model.Client, clientMgr *service.ClientManager, wg *sync.WaitGroup, pongWait, controlWriteTimeout time.Duration) {
	defer func() {
		if p := recover(); p != nil {
			slog.Error("panic 恢复", "error", p)
		}
		if !clientMgr.IsShuttingDown() {
			// 发送关闭帧通知客户端
			err := client.Conn.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(controlWriteTimeout))

			if err != nil {
				slog.Error("发送关闭帧失败", "error", err)
			}

			// 连接断开时, 通知连接池注销
			clientMgr.Unregister(client.ClientID)
		}

		wg.Done()
	}()

	for {
		_, _, err := client.Conn.ReadMessage()

		if err != nil {
			slog.Error("readPump 读取失败", "clientId", client.ClientID, "error", err)
			return
		}

		metrics.MsgRecvTotal.Inc()

		if time.Since(client.LastPong) > pongWait {
			return
		}
	}
}
