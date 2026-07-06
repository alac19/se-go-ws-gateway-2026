package service

import "fmt"

type Service struct {
}

func NewService() Service {
	return Service{}
}

func (s Service) Broadcast() {
	fmt.Println("全服广播业务逻辑...")
}

func (s Service) RoomBroadcast() {
	fmt.Println("房间广播业务逻辑...")
}

func (s Service) ClientSend() {
	fmt.Println("单播业务逻辑...")
}

func (s Service) Stats() int {
	fmt.Println("连接管理统计信息业务逻辑...")

	return 1
}
