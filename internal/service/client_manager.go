package service

import (
	"fmt"
	"sync"

	model "github.com/alac/se-go-ws-gateway-2026/internal/model"
)

// 连接管理器
type ClientManager struct {
	clients    sync.Map
	register   chan *model.Client
	unregister chan string
}

func (cm *ClientManager) Register(client *model.Client) {
	cm.register <- client
}

func (cm *ClientManager) Unregister(ID string) {
	cm.unregister <- ID
}

func (cm *ClientManager) Init() {
	cm.register = make(chan *model.Client, 256)
	cm.unregister = make(chan string, 256)

	for {
		select {
		case client := <-cm.register:
			cm.clients.Store(client.ID, client)

			fmt.Println("注册成功！")
		case ID := <-cm.unregister:
			cm.clients.Delete(ID)

			fmt.Println("注销成功！")
		}
	}
}
