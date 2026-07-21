package service

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	model "github.com/alac/se-go-ws-gateway-2026/internal/model"
	metrics "github.com/alac/se-go-ws-gateway-2026/pkg/metrics"
)

// ClientManager 连接管理器（CSP 模型：通过 channel 通信），并发安全使用sync.Map
type ClientManager struct {
	clients    sync.Map
	register   chan *model.Client
	unregister chan string
	roomMgr    *RoomManager
}

// NewClientManager 创建连接管理器实例，注入房间管理器
func NewClientManager(roomMgr *RoomManager) *ClientManager {
	return &ClientManager{
		clients:    sync.Map{},
		register:   make(chan *model.Client, 256),
		unregister: make(chan string, 256),
		roomMgr:    roomMgr,
	}
}

// Register 向 register 通道发送注册事件（非阻塞）
func (cm *ClientManager) Register(client *model.Client) {
	cm.register <- client
}

// Unregister 向 unregister 通道发送注销事件（非阻塞）
func (cm *ClientManager) Unregister(clientID string) {
	cm.unregister <- clientID
}

func (cm *ClientManager) Init() {
	for {
		select {
		case client := <-cm.register:
			_, loaded := cm.clients.LoadOrStore(client.ClientID, client)

			if loaded {
				err := client.Conn.WriteControl(websocket.CloseMessage,
					websocket.FormatCloseMessage(4002, "clientId already exists"),
					time.Now().Add(1*time.Second))

				if err != nil {
					log.Printf("发送关闭帧失败: %v", err)
				}

				_ = client.Conn.Close()

				continue
			}

			metrics.OnlineConnGauge.Inc()
			metrics.ConnEventTotal.Inc()

			if client.RoomID != "" && cm.roomMgr != nil {
				cm.roomMgr.Join(client.RoomID, client.ClientID)
			}

			fmt.Printf("注册成功: clientID=%s, roomID=%s\n", client.ClientID, client.RoomID)
		case clientID := <-cm.unregister:
			metrics.OnlineConnGauge.Dec()
			metrics.ConnEventTotal.Inc()
			val, ok := cm.clients.LoadAndDelete(clientID)

			if !ok {
				continue
			}

			client := val.(*model.Client)

			// 关闭发送通道
			close(client.SendChan)

			// 从所有房间移除
			if cm.roomMgr != nil {
				cm.roomMgr.RemoveClientFromAllRooms(clientID)
			}

			// 关闭 WebSocket 连接
			_ = client.Conn.Close()

			fmt.Printf("注销成功: clientID=%s\n", clientID)
		}
	}
}

// Get 根据clientID查询客户端

func (cm *ClientManager) Get(clientID string) (*model.Client, bool) {
	val, ok := cm.clients.Load(clientID)
	if !ok {
		return nil, false
	}
	return val.(*model.Client), true
}

// Range 遍历所有在线客户端，回调处理
func (cm *ClientManager) Range(fn func(key any, val any) bool) {
	cm.clients.Range(fn)
}

// GetOnlineCount 获取在线连接总数（统计接口使用）
func (cm *ClientManager) GetOnlineCount() int {
	count := 0
	cm.Range(func(_, _ any) bool {
		count++
		return true
	})
	return count
}

func (cm *ClientManager) Shutdown(gracePeriod int) {
	clients := make([]*model.Client, 0, cm.GetOnlineCount())
	forceCloseTicker := time.After(time.Duration(gracePeriod) * time.Second)

	cm.Range(func(key, value any) bool {
		client := value.(*model.Client)
		client.Lock()
		_ = client.Conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(1000, ""))
		client.Unlock()
		clients = append(clients, client)

		return true
	})

	<-forceCloseTicker

	for _, client := range clients {
		cm.Unregister(client.ClientID)
	}
}
