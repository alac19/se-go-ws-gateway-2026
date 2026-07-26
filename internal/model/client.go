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

func NewClient(clientID, roomID string, conn *websocket.Conn, sendBufferSize int, lastPong time.Time) *Client {
	return &Client{
		ClientID: clientID,
		RoomID:   roomID,
		Conn:     conn,
		SendChan: make(chan []byte, sendBufferSize),
		LastPong: lastPong,
	}
}

// Lock 锁定客户端的 mu，用于保护 Conn 的并发写入
func (c *Client) Lock() {
	c.mu.Lock()
}

// Unlock 解锁客户端的 mu
func (c *Client) Unlock() {
	c.mu.Unlock()
}
