package handler

import (
	"fmt"
	"log"
	"net/http"
	"time"

	client "github.com/alac/se-go-ws-gateway-2026/internal/model"
	clientManager "github.com/alac/se-go-ws-gateway-2026/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func HandlerConnManagement(cm *clientManager.ClientManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		fmt.Println("进入 handler 层 —— WebSocket 接入...")

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)

		if err != nil {
			log.Println(err)
			return
		}

		// 连接管理业务逻辑...
		// 加入连接池...
		clientID := c.Query("clientId")
		roomID := c.Query("roomId")

		if clientID == "" || roomID == "" {
			conn.Close()
			return
		}

		currentTime := time.Now().UTC()
		client := client.NewClient(clientID, roomID, conn, currentTime)
		cm.Register(client)
		defer conn.Close()
		defer cm.Unregister(clientID)

		for {
			messageType, p, err := conn.ReadMessage()

			if err != nil {
				log.Println(err)
				return
			}

			if err := conn.WriteMessage(messageType, p); err != nil {
				log.Println(err)
				return
			}
		}
	}
}
