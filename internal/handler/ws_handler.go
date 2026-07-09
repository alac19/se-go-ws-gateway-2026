package handler

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	model "github.com/alac/se-go-ws-gateway-2026/internal/model"
	service "github.com/alac/se-go-ws-gateway-2026/internal/service"
)

var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// HandlerConnManagement WebSocket 接入：协议升级、连接注册、心跳保活
func HandlerConnManagement(clientMgr *service.ClientManager) gin.HandlerFunc {
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

			_ = conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(4000, "clientId and roomId are required"))
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

			return conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		})

		// 启动读写协程（每个连接独享）
		go writePump(client)
		go readPump(client, clientMgr)
	}
}

// writePump 从 SendChan 读取消息并写入 WebSocket, 同时定时发送 Ping 帧
func writePump(client *model.Client) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-client.SendChan:
			if !ok {
				// SendChan 已关闭，退出
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
			_ = client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			err := client.Conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second))

			if err != nil {
				log.Printf("writePump ping error: %v", err)
				return
			}
		}
	}
}

// readPump 读取 WebSocket 消息, 检测连接断开和 Pong 超时
func readPump(client *model.Client, clientMgr *service.ClientManager) {
	defer func() {
		// 发送关闭帧通知客户端
		_ = client.Conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		// 连接断开时，通知连接池注销
		clientMgr.Unregister(client.ClientID)
	}()

	for {
		_, _, err := client.Conn.ReadMessage()

		if err != nil {
			log.Printf("readPump error: %v", err)
			return
		}

		if time.Since(client.LastPong) > 60*time.Second {
			fmt.Println("Pong 超时, 失活!")
			return
		}
	}
}
