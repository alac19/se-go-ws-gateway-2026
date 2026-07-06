package handler

import (
	"fmt"

	"github.com/gin-gonic/gin"

	service "github.com/alac/se-go-ws-gateway-2026/internal/service"
)

func HandleBroadcast(s service.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		fmt.Println("进入 handler 层 —— 全服广播...")

		s.Broadcast()
	}
}

func HandleRoomBroadcast(s service.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		fmt.Println("进入 handler 层 —— 房间广播...")

		s.RoomBroadcast()
	}
}

func HandleClientSend(s service.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		fmt.Println("进入 handler 层 —— 单播...")

		s.ClientSend()
	}
}

func HandleStats(s service.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		fmt.Println("进入 handler 层 —— 连接管理统计信息...")

		res := s.Stats()

		c.JSON(200, gin.H{"连接数: ": res})
	}
}
