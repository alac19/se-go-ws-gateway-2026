// Package handler provides HTTP and WebSocket request handlers for the gateway.
// It contains handlers for WebSocket connection management, message broadcasting,
// room-based messaging, unicast messaging, and system statistics.
package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	model "github.com/alac/se-go-ws-gateway-2026/internal/model"
	service "github.com/alac/se-go-ws-gateway-2026/internal/service"
)

// HandleBroadcast 全服广播：读取请求体 JSON 并推送给所有在线客户端。
// 请求体必须为合法的 JSON 对象或数组, 否则返回 400 错误。
// 推送成功后返回 200 状态码, 业务码为 BizCodeSuccess（0）。
func HandleBroadcast(router *service.MessageRouter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 读取请求体原始数据
		body, err := c.GetRawData()

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": model.BizCodeBadRequest, "status": "error", "error": "无效的请求体"})
			return
		}

		if len(body) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"code": model.BizCodeBadRequest, "status": "error", "error": "请求体不能为空"})
			return
		}

		if !json.Valid(body) || !(body[0] == '{' || body[0] == '[') {
			c.JSON(http.StatusBadRequest, gin.H{"code": model.BizCodeBadRequest, "status": "error", "error": "无效的 JSON 格式"})
			return
		}

		msg := &model.Message{Payload: body}
		router.SendBroadcast(msg)

		c.JSON(http.StatusOK, gin.H{"code": model.BizCodeSuccess, "status": "success", "data": nil})
	}
}

// HandleRoomBroadcast 房间广播：读取请求体 JSON 并推送给指定房间的所有客户端。
// 若 roomId 为空或房间不存在, 返回对应的 400 或 404 错误。
// 推送成功后返回 200 状态码, 业务码为 BizCodeSuccess（0）。
func HandleRoomBroadcast(router *service.MessageRouter, roomMgr *service.RoomManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 读取请求原始数据
		body, err := c.GetRawData()

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": model.BizCodeBadRequest, "status": "error", "error": "无效的请求体"})
			return
		}

		if len(body) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"code": model.BizCodeBadRequest, "status": "error", "error": "请求体不能为空"})
			return
		}

		if !json.Valid(body) || !(body[0] == '{' || body[0] == '[') {
			c.JSON(http.StatusBadRequest, gin.H{"code": model.BizCodeBadRequest, "status": "error", "error": "无效的 JSON 格式"})
			return
		}

		roomId := c.Param("roomId")

		if roomId == "" {
			c.JSON(http.StatusBadRequest, gin.H{"code": model.BizCodeBadRequest, "status": "error", "error": "roomId is required"})
			return
		}

		ok := roomMgr.HasRoom(roomId)

		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"code": model.BizCodeNotFound, "status": "error", "error": "目标房间不存在"})
			return
		}

		msg := &model.Message{Payload: body}
		router.SendRoom(roomId, msg)

		c.JSON(http.StatusOK, gin.H{"code": model.BizCodeSuccess, "status": "success", "data": nil})
	}
}

// HandleClientSend 单播推送：读取请求体 JSON 并推送给指定客户端。
// 若 clientId 为空或目标客户端离线, 返回对应的 400 或 404 错误。
// 推送成功后返回 200 状态码, 业务码为 BizCodeSuccess（0）。
func HandleClientSend(router *service.MessageRouter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 读取请求体原始数据
		body, err := c.GetRawData()

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": model.BizCodeBadRequest, "status": "error", "error": "无效的请求体"})
			return
		}

		if len(body) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"code": model.BizCodeBadRequest, "status": "error", "error": "请求体不能为空"})
			return
		}

		if !json.Valid(body) || !(body[0] == '{' || body[0] == '[') {
			c.JSON(http.StatusBadRequest, gin.H{"code": model.BizCodeBadRequest, "status": "error", "error": "无效的 JSON 格式"})
			return
		}

		clientId := c.Param("clientId")

		if clientId == "" {
			c.JSON(http.StatusBadRequest, gin.H{"code": model.BizCodeBadRequest, "status": "error", "error": "clientId is required"})
			return
		}

		msg := &model.Message{Payload: body}
		ok := router.SendSingle(clientId, msg)

		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"code": model.BizCodeNotFound, "status": "error", "error": "目标客户端离线或不存在"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"code": model.BizCodeSuccess, "status": "success", "data": nil})
	}
}

// HandleStats 连接统计：返回当前网关的在线连接数和各房间连接分布。
// 响应体包含：总在线连接数、各房间连接数分布、服务运行时长（秒）。
// 始终返回 200 状态码, 业务码为 BizCodeSuccess（0）。
func HandleStats(clientMgr *service.ClientManager, roomMgr *service.RoomManager, svrInitTime time.Time) gin.HandlerFunc {
	return func(c *gin.Context) {
		res1 := clientMgr.GetOnlineCount()
		res2 := roomMgr.GetAllRoomsConnStats()

		c.JSON(http.StatusOK, gin.H{
			"code":   model.BizCodeSuccess,
			"status": "success",
			"data": gin.H{
				"online_connections":          res1,
				"all_rooms_connections_stats": res2,
				"gateway_server_initial_time": int(time.Since(svrInitTime).Seconds()),
			},
		})
	}
}
