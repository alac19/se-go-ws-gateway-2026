package handler

import (
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/alac/se-go-ws-gateway-2026/internal/model"
	service "github.com/alac/se-go-ws-gateway-2026/internal/service"
)

// HandleBroadcast 全服广播：读取请求体 JSON 并推送给所有在线客户端
func HandleBroadcast(router *service.MessageRouter) gin.HandlerFunc {
	return func(c *gin.Context) {
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

		if !json.Valid(body) || !(body[0] == '{' || body[0] == '[') {
			c.JSON(400, gin.H{"code": 400, "status": "error", "error": "无效的 JSON 格式"})
			return
		}

		msg := &model.Message{Payload: body}
		router.SendBroadcast(msg)

		c.JSON(200, gin.H{"code": 0, "status": "success", "data": nil})
	}
}

// HandleRoomBroadcast 房间广播：读取请求体 JSON 并推送给指定房间的所有客户端
func HandleRoomBroadcast(router *service.MessageRouter, roomMgr *service.RoomManager) gin.HandlerFunc {
	return func(c *gin.Context) {
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

		if !json.Valid(body) || !(body[0] == '{' || body[0] == '[') {
			c.JSON(400, gin.H{"code": 400, "status": "error", "error": "无效的 JSON 格式"})
			return
		}

		roomId := c.Param("roomId")

		if roomId == "" {
			c.JSON(400, gin.H{"code": 400, "status": "error", "error": "roomId is required"})
			return
		}

		ok := roomMgr.HasRoom(roomId)

		if !ok {
			c.JSON(404, gin.H{"code": 404, "status": "error", "error": "目标房间不存在"})
			return
		}

		msg := &model.Message{Payload: body}
		router.SendRoom(roomId, msg)

		c.JSON(200, gin.H{"code": 0, "status": "success", "data": nil})
	}
}

// HandleClientSend 单播推送：读取请求体 JSON 并推送给指定客户端
func HandleClientSend(router *service.MessageRouter) gin.HandlerFunc {
	return func(c *gin.Context) {
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

		if !json.Valid(body) || !(body[0] == '{' || body[0] == '[') {
			c.JSON(400, gin.H{"code": 400, "status": "error", "error": "无效的 JSON 格式"})
			return
		}

		clientId := c.Param("clientId")

		if clientId == "" {
			c.JSON(400, gin.H{"code": 400, "status": "error", "error": "clientId is required"})
			return
		}

		msg := &model.Message{Payload: body}
		ok := router.SendSingle(clientId, msg)

		if !ok {
			c.JSON(404, gin.H{"code": 404, "status": "error", "error": "目标客户端离线或不存在"})
			return
		}

		c.JSON(200, gin.H{"code": 0, "status": "success", "data": nil})
	}
}

// HandleStats 连接统计：返回当前网关在线连接数
func HandleStats(clientMgr *service.ClientManager, roomMgr *service.RoomManager, svrInitTime time.Time) gin.HandlerFunc {
	return func(c *gin.Context) {
		res1 := clientMgr.GetOnlineCount()
		res2 := roomMgr.GetAllRoomsConnStats()

		c.JSON(200, gin.H{
			"code":   0,
			"status": "success",
			"data": gin.H{
				"online_connections":          res1,
				"all_rooms_connections_stats": res2,
				"gateway_server_initial_time": int(time.Since(svrInitTime).Seconds()),
			},
		})
	}
}
