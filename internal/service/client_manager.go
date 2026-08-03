package service

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	model "github.com/alac/se-go-ws-gateway-2026/internal/model"
	metrics "github.com/alac/se-go-ws-gateway-2026/pkg/metrics"
)

// ClientManager 连接管理器（CSP 模型：通过 channel 通信），并发安全使用sync.Map
type ClientManager struct {
	clients      sync.Map
	register     chan *model.Client
	unregister   chan string
	roomMgr      *RoomManager
	shuttingDown bool
}

// NewClientManager 创建连接管理器实例，注入房间管理器
func NewClientManager(roomMgr *RoomManager, registerBufferSize, unregisterBufferSize int) *ClientManager {
	return &ClientManager{
		clients:    sync.Map{},
		register:   make(chan *model.Client, registerBufferSize),
		unregister: make(chan string, unregisterBufferSize),
		roomMgr:    roomMgr,
	}
}

// 提供导出方法供外部读取
func (cm *ClientManager) IsShuttingDown() bool {
	return cm.shuttingDown
}

// Register 向 register 通道发送注册事件（非阻塞）
func (cm *ClientManager) Register(client *model.Client) {
	cm.register <- client
}

// Unregister 向 unregister 通道发送注销事件（非阻塞）
func (cm *ClientManager) Unregister(clientID string) {
	cm.unregister <- clientID
}

func (cm *ClientManager) Init(ctx context.Context, controlWriteTimeout time.Duration) {
	for {
		select {
		case client := <-cm.register:
			_, loaded := cm.clients.LoadOrStore(client.ClientID, client)

			if loaded {
				if client.Conn != nil {
					err := client.Conn.WriteControl(websocket.CloseMessage,
						websocket.FormatCloseMessage(4002, "clientId already exists"),
						time.Now().Add(controlWriteTimeout))

					if err != nil {
						slog.Error("发送关闭帧失败", "error", err)
					}

					_ = client.Conn.Close()
				}

				continue
			}

			metrics.OnlineConnGauge.Inc()
			metrics.ConnEventTotal.Inc()

			if client.RoomID != "" && cm.roomMgr != nil {
				cm.roomMgr.Join(client.RoomID, client.ClientID)
			}
		case clientID := <-cm.unregister:
			val, ok := cm.clients.LoadAndDelete(clientID)

			if !ok {
				continue
			}

			metrics.OnlineConnGauge.Dec()
			metrics.ConnEventTotal.Inc()

			client := val.(*model.Client)

			// 关闭发送通道
			close(client.SendChan)

			// 从所有房间移除
			if cm.roomMgr != nil {
				cm.roomMgr.RemoveClientFromAllRooms(clientID)
			}

			if client.Conn != nil {
				// 关闭 WebSocket 连接
				_ = client.Conn.Close()
			}

		case <-ctx.Done():
			return
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

func (cm *ClientManager) Shutdown(gracePeriod, controlWriteTimeout time.Duration) {
	cm.shuttingDown = true
	clients := make([]*model.Client, 0, cm.GetOnlineCount())
	forceCloseTicker := time.After(gracePeriod)

	cm.Range(func(key, value any) bool {
		client := value.(*model.Client)
		client.Lock()

		if client.Conn != nil {
			err := client.Conn.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
				time.Now().Add(controlWriteTimeout))
			if err != nil {
				slog.Error("发送关闭帧失败", "error", err)
			}
		}

		client.Unlock()
		clients = append(clients, client)

		return true
	})

	<-forceCloseTicker

	for _, client := range clients {
		val, ok := cm.clients.LoadAndDelete(client.ClientID)

		if !ok {
			continue
		}

		metrics.OnlineConnGauge.Dec()
		metrics.ConnEventTotal.Inc()

		client := val.(*model.Client)

		// 关闭发送通道
		close(client.SendChan)

		// 从所有房间移除
		if cm.roomMgr != nil {
			cm.roomMgr.RemoveClientFromAllRooms(client.ClientID)
		}

		if client.Conn != nil {
			// 关闭 WebSocket 连接
			_ = client.Conn.Close()
		}
	}
}
