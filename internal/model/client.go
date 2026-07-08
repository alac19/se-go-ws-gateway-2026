package model

import (
	"context"
	"github.com/gorilla/websocket"
)

type Client struct {
    ID       string          // 客户端唯一标识
    RoomID   string          // 所属房间 ID
    Conn     *websocket.Conn // WebSocket 连接对象
    Send     chan []byte     // 消息发送通道（缓冲区大小可配置）
    LastPong time.Time       // 最后一次收到 Pong 的时间
    mu       sync.Mutex      // 保护 Conn 的写操作
}