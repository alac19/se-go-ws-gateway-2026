package main

import (
	"fmt"

	"github.com/gin-gonic/gin"

	handler "github.com/alac/se-go-ws-gateway-2026/internal/handler"
	service "github.com/alac/se-go-ws-gateway-2026/internal/service"
)

func main() {
	fmt.Println("MVP 框架搭建...")
	r := gin.Default()

	// 1. 初始化核心层（顺序：RoomManager -> ClientManager -> MessageRouter）
	roomMgr := service.NewRoomManager()
	clientMgr := service.NewClientManager(roomMgr)
	router := service.NewMessageRouter(clientMgr, roomMgr, nil)

	// 2. 启动 ClientManager 后台循环（处理 register/unregister 事件）
	go clientMgr.Init()

	// 2. WebSocket 路由，传入 clientMgr
	hd := handler.HandlerConnManagement(clientMgr)
	r.GET("/ws", hd)
	fmt.Println("路由注册成功！")

	// 3. 推送类接口，传入 messageRouter
	hd1 := handler.HandleBroadcast(router)
	r.POST("/api/broadcast", hd1)
	fmt.Println("路由注册成功！")

	hd2 := handler.HandleRoomBroadcast(router)
	r.POST("/api/room/:roomId/broadcast", hd2)
	fmt.Println("路由注册成功！")

	hd3 := handler.HandleClientSend(router)
	r.POST("/api/client/:clientId/send", hd3)
	fmt.Println("路由注册成功！")

	// 4. 统计接口，传入 clientMgr
	hd4 := handler.HandleStats(clientMgr)
	r.GET("/api/stats", hd4)
	fmt.Println("路由注册成功！")

	r.Run()
}
