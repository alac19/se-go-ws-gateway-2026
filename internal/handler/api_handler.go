package handler

import (
	"fmt"

	"github.com/gin-gonic/gin"

	service "github.com/alac/se-go-ws-gateway-2026/internal/service"
	"github.com/alac/se-go-ws-gateway-2026/internal/model"
)

// 入参从 service.Service 改为 *service.MessageRouter
func HandleBroadcast(router *service.MessageRouter) gin.HandlerFunc {
	return func(c *gin.Context) {
		fmt.Println("进入 handler 层 —— 全服广播...")
		// 构造空消息兼容原有无参调用，如需传参后续再加
		msg := &model.Message{Payload: []byte("")}
		router.SendBroadcast(msg)
	}
}

func HandleRoomBroadcast(router *service.MessageRouter) gin.HandlerFunc {
	return func(c *gin.Context) {
		fmt.Println("进入 handler 层 —— 房间广播...")
		msg := &model.Message{Payload: []byte("")}
		roomId := c.Param("roomId")
		router.SendRoom(roomId, msg)
	}
}

func HandleClientSend(router *service.MessageRouter) gin.HandlerFunc {
	return func(c *gin.Context) {
		fmt.Println("进入 handler 层 —— 单播...")
		msg := &model.Message{Payload: []byte("")}
		clientId := c.Param("clientId")
		router.SendSingle(clientId, msg)
	}
}

// 统计接口依赖 ClientManager
func HandleStats(clientMgr *service.ClientManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		fmt.Println("进入 handler 层 —— 连接管理统计信息...")
		res := clientMgr.GetOnlineCount()
		c.JSON(200, gin.H{"连接数: ": res})
	}
}