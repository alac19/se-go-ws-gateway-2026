package handler

import (
	"fmt"

	"github.com/gin-gonic/gin"

	service "github.com/alac/se-go-ws-gateway-2026/internal/service"
	"github.com/alac/se-go-ws-gateway-2026/internal/model"
)

// HandleBroadcast 全服广播：读取请求体 JSON 并推送给所有在线客户端
func HandleBroadcast(router *service.MessageRouter) gin.HandlerFunc {
	return func(c *gin.Context) {
		fmt.Println("进入 handler 层 —— 全服广播...")

		// 读取请求体 JSON
		body, err := c.GetRawData()
		if err != nil {
			c.JSON(400, gin.H{"code": 400, "status": "error", "error": "无效的请求体"})
			return
		}

		if len(body) == 0 {
			c.JSON(400, gin.H{"code": 400, "status": "error", "error": "请求体不能为空"})
			return
		}

		msg := &model.Message{Payload: body}
		router.SendBroadcast(msg)

		fmt.Printf("全服广播完成，消息大小: %d bytes\n", len(body))
		c.JSON(200, gin.H{"code": 0, "status": "success", "data": nil})
	}
}

// HandleRoomBroadcast 房间广播：读取请求体 JSON 并推送给指定房间的所有客户端
func HandleRoomBroadcast(router *service.MessageRouter) gin.HandlerFunc {
	return func(c *gin.Context) {
		fmt.Println("进入 handler 层 —— 房间广播...")

		// 读取请求体 JSON
		body, err := c.GetRawData()
		if err != nil {
			c.JSON(400, gin.H{"code": 400, "status": "error", "error": "无效的请求体"})
			return
		}

		if len(body) == 0 {
			c.JSON(400, gin.H{"code": 400, "status": "error", "error": "请求体不能为空"})
			return
		}

		roomId := c.Param("roomId")
		msg := &model.Message{Payload: body}
		router.SendRoom(roomId, msg)

		fmt.Printf("房间广播完成，roomId=%s，消息大小: %d bytes\n", roomId, len(body))
		c.JSON(200, gin.H{"code": 0, "status": "success", "data": nil})
	}
}

// HandleClientSend 单播推送：读取请求体 JSON 并推送给指定客户端
func HandleClientSend(router *service.MessageRouter) gin.HandlerFunc {
	return func(c *gin.Context) {
		fmt.Println("进入 handler 层 —— 单播...")

		// 读取请求体 JSON
		body, err := c.GetRawData()
		if err != nil {
			c.JSON(400, gin.H{"code": 400, "status": "error", "error": "无效的请求体"})
			return
		}

		if len(body) == 0 {
			c.JSON(400, gin.H{"code": 400, "status": "error", "error": "请求体不能为空"})
			return
		}

		clientId := c.Param("clientId")
		msg := &model.Message{Payload: body}
		ok := router.SendSingle(clientId, msg)

		if !ok {
			c.JSON(404, gin.H{"code": 404, "status": "error", "error": "目标客户端离线或不存在"})
			return
		}

		fmt.Printf("单播完成，clientId=%s，消息大小: %d bytes\n", clientId, len(body))
		c.JSON(200, gin.H{"code": 0, "status": "success", "data": nil})
	}
}

// HandleStats 连接统计：返回当前网关在线连接数
func HandleStats(clientMgr *service.ClientManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		fmt.Println("进入 handler 层 —— 连接管理统计信息...")
		res := clientMgr.GetOnlineCount()
		c.JSON(200, gin.H{
			"code":   0,
			"status": "success",
			"data": gin.H{
				"online_connections": res,
			},
		})
	}
}