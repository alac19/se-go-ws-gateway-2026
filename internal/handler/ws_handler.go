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
		defer conn.Close()
		defer clientMgr.Unregister(clientID)

		// 读写逻辑...
		// for {
		// 	messageType, p, err := conn.ReadMessage()

		// 	if err != nil {
		// 		log.Println(err)
		// 		return
		// 	}

		// 	if err := conn.WriteMessage(messageType, p); err != nil {
		// 		log.Println(err)
		// 		return
		// 	}
		// }

		// go func() {
		// 	for {
		// 		messageType := <-client.Send

		// 		if err := conn.WriteMessage(messageType, p); err != nil {
		// 			log.Println(err)
		// 		}
		// 	}
		// }()
	}
}
