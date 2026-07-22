package handler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	model "github.com/alac/se-go-ws-gateway-2026/internal/model"
	service "github.com/alac/se-go-ws-gateway-2026/internal/service"
	metrics "github.com/alac/se-go-ws-gateway-2026/pkg/metrics"
)

var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}
var validIDPattern = regexp.MustCompile("^[a-zA-Z0-9_-]+$")

// HandlerConnManagement WebSocket 接入：协议升级、连接注册、心跳保活
func HandlerConnManagement(clientMgr *service.ClientManager, ctx context.Context, wg *sync.WaitGroup) gin.HandlerFunc {
	return func(c *gin.Context) {
		fmt.Println("进入 handler 层 —— WebSocket 接入...")

		// 升级 HTTP 为 WebSocket 连接
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)

		if err != nil {
			log.Printf("WebSocket 升级失败: %v", err)

			c.JSON(http.StatusInternalServerError, gin.H{
				"code":   500,
				"status": "error",
				"error":  "WebSocket 协议升级失败",
			})

			return
		}

		// 校验查询参数
		clientID := c.Query("clientId")
		roomID := c.Query("roomId")

		if clientID == "" || roomID == "" {
			log.Printf("参数缺失: clientId=%q, roomId=%q", clientID, roomID)

			err := conn.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(4000, "clientId and roomId are required"),
				time.Now().Add(1*time.Second))

			if err != nil {
				log.Printf("发送关闭帧失败: %v", err)
			}

			_ = conn.Close()

			return
		}

		if !validIDPattern.MatchString(clientID) {
			log.Printf("clientId 包含非法字符: %q", clientID)

			err := conn.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(4001, "invalid clientId format"),
				time.Now().Add(1*time.Second))

			if err != nil {
				log.Printf("发送关闭帧失败: %v", err)
			}

			_ = conn.Close()

			return
		}

		if !validIDPattern.MatchString(roomID) {
			log.Printf("roomID 包含非法字符: %q", roomID)

			err := conn.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(4001, "invalid roomID format"),
				time.Now().Add(1*time.Second))

			if err != nil {
				log.Printf("发送关闭帧失败: %v", err)
			}

			_ = conn.Close()

			return
		}

		if _, res := clientMgr.Get(clientID); res {
			log.Printf("clientId 已存在: clientId=%q", clientID)

			err := conn.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(4002, "clientId already exists"),
				time.Now().Add(1*time.Second))

			if err != nil {
				log.Printf("发送关闭帧失败: %v", err)
			}

			_ = conn.Close()

			return
		}

		// 设置初始读超时（Pong 响应窗口 60s）
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		// 创建客户端并注册到连接管理器
		client := model.NewClient(clientID, roomID, conn, time.Now())
		clientMgr.Register(client)

		// 设置 Pong 处理器，收到 Pong 时延长读超时并更新 LastPong
		conn.SetPongHandler(func(appData string) error {
			client.LastPong = time.Now()

			log.Printf("[心跳] 客户端 %s 收到 Pong，LastPong 更新为 %s", client.ClientID, client.LastPong.Format("15:04:05"))

			return conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		})

		// 启动读写协程（每个连接独享）
		wg.Add(2)
		go writePump(client, ctx, wg)
		go readPump(client, clientMgr, wg)
	}
}

// writePump 从 SendChan 读取消息并写入 WebSocket, 同时定时发送 Ping 帧
func writePump(client *model.Client, ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		if p := recover(); p != nil {
			log.Printf("internal error: %v", p)
		}

		wg.Done()
		ticker.Stop()
	}()

	log.Printf("[心跳] 客户端 %s writePump 启动，Ping 定时器已开启 (间隔 30s)", client.ClientID)

	for {
		select {
		case msg, ok := <-client.SendChan:
			if !ok {
				// SendChan 已关闭，退出
				log.Printf("[心跳] 客户端 %s SendChan 已关闭，writePump 退出", client.ClientID)
				return
			}

			_ = client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			client.Lock()
			err := client.Conn.WriteMessage(websocket.TextMessage, msg)
			client.Unlock()

			if err != nil {
				log.Printf("writePump error: %v", err)
				return
			}
		case <-ticker.C:
			log.Printf("[心跳] 客户端 %s 发送 Ping 帧 (时间: %s)", client.ClientID, time.Now().Format("15:04:05"))
			_ = client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			err := client.Conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second))

			if err != nil {
				log.Printf("writePump ping error: %v", err)
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

// readPump 读取 WebSocket 消息, 检测连接断开和 Pong 超时
func readPump(client *model.Client, clientMgr *service.ClientManager, wg *sync.WaitGroup) {
	defer func() {
		if p := recover(); p != nil {
			log.Printf("internal error: %v", p)
		}

		// 发送关闭帧通知客户端
		_ = client.Conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		// 连接断开时，通知连接池注销
		clientMgr.Unregister(client.ClientID)
		wg.Done()
		log.Printf("[心跳] 客户端 %s readPump 退出，连接已注销", client.ClientID)
	}()

	for {
		_, _, err := client.Conn.ReadMessage()

		if err != nil {
			log.Printf("readPump error: %v", err)
			return
		}

		metrics.MsgRecvTotal.Inc()

		if time.Since(client.LastPong) > 60*time.Second {
			log.Printf("[心跳] 客户端 %s Pong 超时 (LastPong: %s)，连接失活!", client.ClientID, client.LastPong.Format("15:04:05"))
			return
		}
	}
}
