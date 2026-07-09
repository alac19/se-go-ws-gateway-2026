package model

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// 连接结构体
type Client struct {
	ClientID string          // 客户端唯一标识
	RoomID   string          // 所属房间 ID
	Conn     *websocket.Conn // WebSocket 连接对象
	SendChan chan []byte     // 消息发送通道（缓冲区大小可配置）
	LastPong time.Time       // 最后一次收到 Pong 的时间
	mu       sync.Mutex      // 保护 Conn 的写操作
}

func NewClient(clientID, roomID string, conn *websocket.Conn, lastPong time.Time) *Client {
	return &Client{
		ClientID: clientID,
		RoomID:   roomID,
		Conn:     conn,
		SendChan: make(chan []byte, 256),
		LastPong: lastPong,
	}
}
