// Package handler provides HTTP and WebSocket request handlers for the gateway.
// It contains handlers for WebSocket connection management, message broadcasting,
// room-based messaging, unicast messaging, and system statistics.
package handler

import (
	"context"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	config "github.com/alac/se-go-ws-gateway-2026/internal/config"
	model "github.com/alac/se-go-ws-gateway-2026/internal/model"
	service "github.com/alac/se-go-ws-gateway-2026/internal/service"
	auth "github.com/alac/se-go-ws-gateway-2026/pkg/auth"
	metrics "github.com/alac/se-go-ws-gateway-2026/pkg/metrics"
)

// validIDPattern 用于校验 clientId 和 roomId 的合法字符集。
// 仅允许字母、数字、下划线和短横线, 防止注入攻击。
var validIDPattern = regexp.MustCompile("^[a-zA-Z0-9_-]+$")

// HandlerConnManagement WebSocket 接入处理器。
// 负责将 HTTP 请求升级为 WebSocket 连接, 并进行参数校验、重复连接检查、客户端注册，
// 身份绑定校验（clientId 必须与 token 中的账号一致），
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
			// 升级失败绝大多数是客户端未按 WebSocket 协议发起连接(如普通 HTTP 请求),
			// 属于客户端问题, 记为 Warn
			slog.Warn("WebSocket 升级失败", "error", err)

			c.JSON(http.StatusInternalServerError, gin.H{
				"code":   model.BizCodeInternalError,
				"status": "error",
				"error":  "WebSocket 协议升级失败",
			})

			return
		}

		// 若网关正在优雅退出, 拒绝新连接并返回 1001 状态码（Going Away）
		if clientMgr.IsShuttingDown() {
			closeWithReason(conn, cfg.ControlWriteTimeout(), websocket.CloseGoingAway, "service is shutting down")

			return
		}

		// 校验查询参数
		clientID := c.Query("clientId")
		roomID := c.Query("roomId")

		if clientID == "" || roomID == "" {
			slog.Warn("参数缺失", "clientId", clientID, "roomId", roomID)

			closeWithReason(conn, cfg.ControlWriteTimeout(), model.CloseCodeMissingParam, "clientId and roomId are required")

			return
		}

		if !validIDPattern.MatchString(clientID) {
			slog.Warn("clientId 包含非法字符", "clientId", clientID)

			closeWithReason(conn, cfg.ControlWriteTimeout(), model.CloseCodeInvalidFormat, "invalid clientId format")

			return
		}

		if !validIDPattern.MatchString(roomID) {
			slog.Warn("roomID 包含非法字符", "roomID", roomID)

			closeWithReason(conn, cfg.ControlWriteTimeout(), model.CloseCodeInvalidFormat, "invalid roomID format")

			return
		}

		// 身份绑定: 校验 token 中的账号与 clientId 是否匹配。
		// 鉴权中间件(HandleJWTAuth)已把解析出的声明写入 gin.Context, 这里取出本次连接的身份。
		claimsValue, exists := c.Get(auth.ContextKeyClaims)

		if !exists {
			// 正常情况下中间件已经拦截, 走到这里说明该路由没有挂载鉴权中间件
			slog.Error("上下文缺少鉴权信息", "clientId", clientID)

			closeWithReason(conn, cfg.ControlWriteTimeout(), model.CloseCodeUnauthorized, "missing auth context")

			return
		}

		claims, ok := claimsValue.(auth.Claims)

		if !ok {
			slog.Error("上下文中的鉴权信息类型异常", "clientId", clientID)

			closeWithReason(conn, cfg.ControlWriteTimeout(), model.CloseCodeUnauthorized, "invalid auth context")

			return
		}

		// clientId 约定为 "<账号名>" 或 "<账号名>-<后缀>"。
		// 同一账号打开多个页面时用后缀区分, 既满足重复连接检查又能绑定身份;
		// 若既不等于账号名也不以其为前缀, 说明客户端在冒用他人的连接标识, 直接拒绝。
		if clientID != claims.Subject && !strings.HasPrefix(clientID, claims.Subject+"-") {
			slog.Warn("clientId 与登录身份不匹配", "clientId", clientID, "user", claims.Subject)

			closeWithReason(conn, cfg.ControlWriteTimeout(), model.CloseCodeIdentityMismatch, "clientId does not match the authenticated user")

			return
		}

		if _, res := clientMgr.Get(clientID); res {
			slog.Warn("clientId 已存在, 拒绝连接", "clientId", clientID)

			closeWithReason(conn, cfg.ControlWriteTimeout(), model.CloseCodeDuplicateID, "clientId already exists")

			return
		}

		// 设置初始读超时（Pong 响应窗口 60s）
		_ = conn.SetReadDeadline(time.Now().Add(cfg.ReadDeadline()))

		// 创建客户端并注册到连接管理器
		client := model.NewClient(clientID, roomID, conn, cfg.Channel.SendBufferSize, time.Now())
		clientMgr.Register(client)

		// 连接建立日志放在 Debug 级别: 压测时成千上万条连接日志会刷屏并拖慢终端输出,
		// 默认 info 级别下不会输出; 需要观察时用 WS_LOG_LEVEL=debug 启动。
		// 压测期间的连接数等统计数据应看 /metrics, 而不是翻日志。
		slog.Debug("WebSocket 连接建立", "clientId", clientID, "roomId", roomID, "user", claims.Subject)

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

// closeWithReason 向客户端发送关闭帧说明拒绝原因, 随后关闭连接。
// 关闭帧发送失败只记录日志而不中断流程: 连接马上就要关闭,
// 这里失败不影响后续清理, 但记下来有助于判断客户端是否收到了拒绝原因。
//
// 参数:
//   - conn: 待关闭的连接
//   - timeout: 发送关闭帧的写入超时
//   - code: 关闭码, 4000-4999 为应用自定义范围, 见 model 包中的 CloseCode 常量
//   - reason: 关闭原因, 会随关闭帧一起发给客户端
func closeWithReason(conn *websocket.Conn, timeout time.Duration, code int, reason string) {
	if err := conn.WriteControl(websocket.CloseMessage,
		websocket.FormatCloseMessage(code, reason),
		time.Now().Add(timeout)); err != nil {
		// 关闭帧发送失败只影响客户端能否收到拒绝原因, 连接随后仍会被关闭
		slog.Warn("发送关闭帧失败", "code", code, "error", err)
	}

	_ = conn.Close()
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
				// 写入失败通常意味着客户端已断开或网络异常, 属于连接级问题
				slog.Warn("writePump 写入失败", "clientId", client.ClientID, "error", err)
				return
			}
		case <-ticker.C:
			_ = client.Conn.SetWriteDeadline(time.Now().Add(writeDeadline))
			err := client.Conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(pingWriteTimeout))

			if err != nil {
				// Ping 发送失败说明连接已不可用, 属于连接级问题
				slog.Warn("发送 ping 帧失败", "clientId", client.ClientID, "error", err)
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
				// 走到这里说明读取已经失败(客户端多半已断开), 再补发关闭帧失败属于预期结果,
				// 因此记为 Debug, 避免每个正常断开的客户端都刷一条 Warn
				slog.Debug("发送关闭帧失败", "clientId", client.ClientID, "error", err)
			}

			// 连接断开时, 通知连接池注销
			clientMgr.Unregister(client.ClientID)
		}

		wg.Done()
	}()

	for {
		_, _, err := client.Conn.ReadMessage()

		if err != nil {
			// 客户端正常关闭(1000)或服务端优雅退出主动关闭(1001)属于预期行为, 记 Info;
			// 其余(连接重置、超时、异常关闭码)才记 Warn, 避免正常断开刷出 ERROR 日志
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				slog.Info("客户端连接已关闭", "clientId", client.ClientID)
			} else {
				slog.Warn("readPump 读取失败", "clientId", client.ClientID, "error", err)
			}

			return
		}

		metrics.MsgRecvTotal.Inc()

		if time.Since(client.LastPong) > pongWait {
			return
		}
	}
}
