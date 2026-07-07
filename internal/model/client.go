package model

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// 连接结构体
type Client struct {
	ID       string
	RoomID   string
	Conn     *websocket.Conn
	Send     chan []byte
	LastPong time.Time
	mu       sync.Mutex
}

func NewClient(ID, roomID string, conn *websocket.Conn, lastPong time.Time) *Client {
	return &Client{ID: ID, RoomID: roomID, Conn: conn, Send: make(chan []byte, 256), LastPong: lastPong}
}
