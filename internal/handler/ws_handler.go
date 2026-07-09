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
	CheckOrigin: func(r *http.Request) bool { return true },
}

// 入参改为 *service.ClientManager
func HandlerConnManagement(clientMgr *service.ClientManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		fmt.Println("进入 handler 层 —— WebSocket 接入...")

		// 升级连接
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)

		if err != nil {
			log.Println(err)
			return
		}

		// 连接生命周期管理业务逻辑...
		clientID := c.Query("clientId")
		roomID := c.Query("roomId")

		if clientID == "" || roomID == "" {
			conn.Close()
			return
		}

		client := model.NewClient(clientID, roomID, conn, time.Now())
		clientMgr.Register(client)

		// 读写逻辑...
		// 启动读写协程（每个连接独享）
		go writePump(client)
		go readPump(client, clientMgr)
	}
}

// writePump 从 SendChan 读取消息并写入 WebSocket
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
			if err := client.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				log.Printf("writePump error: %v", err)
			}
		case <-ticker.C:
			client.Conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second))
		}
	}
}

// readPump 读取 WebSocket 消息（检测连接断开）
func readPump(client *model.Client, clientMgr *service.ClientManager) {
	defer func() {
		// 连接断开时，通知连接池注销
		clientMgr.Unregister(client.ClientID)
	}()

	client.Conn.SetPongHandler(func(appData string) error {
		client.LastPong = time.Now()

		return nil
	})

	for {
		_, _, err := client.Conn.ReadMessage()

		if err != nil {
			log.Printf("readPump error: %v", err)
			return
		}
		// 这里可以处理上行消息...
		if time.Since(client.LastPong) > 60*time.Second {
			fmt.Println("Pong 超时, 失活!")
			return
		}
	}
}
