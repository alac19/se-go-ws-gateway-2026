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

	cm := &service.ClientManager{}

	go cm.Init()

	hd := handler.HandlerConnManagement(cm)

	r.GET("/ws", hd)

	fmt.Println("路由注册成功！")

	service := service.NewService()

	hd1 := handler.HandleBroadcast(service)

	r.POST("/api/broadcast", hd1)

	fmt.Println("路由注册成功！")

	hd2 := handler.HandleRoomBroadcast(service)

	r.POST("/api/room/:roomId/broadcast", hd2)

	fmt.Println("路由注册成功！")

	hd3 := handler.HandleClientSend(service)

	r.POST("/api/client/:clientId/send", hd3)

	fmt.Println("路由注册成功！")

	hd4 := handler.HandleStats(service)

	r.GET("/api/stats", hd4)

	fmt.Println("路由注册成功！")

	r.Run()
}
