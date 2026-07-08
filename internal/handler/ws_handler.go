package handler

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	service "github.com/alac/se-go-ws-gateway-2026/internal/service"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// 入参改为 *service.ClientManager
func HandlerConnManagement(clientMgr *service.ClientManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		fmt.Println("进入 handler 层 —— WebSocket 接入...")

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)

		if err != nil {
			log.Println(err)
			return
		}

		// 交给连接管理器托管，移除内部读写循环
		clientId := c.Query("clientId")
		roomId := c.Query("roomId")
		clientMgr.AddClient(clientId, roomId, conn)
	}
}