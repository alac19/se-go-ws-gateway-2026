// Package service provides the core business logic for the WebSocket gateway.
// It includes connection management, room management, and message routing.
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

// ClientManager 连接管理器，采用 CSP（Communicating Sequential Processes）模型，
// 通过 channel 进行协程间通信，使用 sync.Map 保证并发安全。
// 负责客户端的注册、注销、查询和优雅关闭。
type ClientManager struct {
	clients      sync.Map
	register     chan *model.Client
	unregister   chan string
	roomMgr      *RoomManager
	shuttingDown bool
}

// NewClientManager 创建连接管理器实例。
// 参数:
//   - roomMgr: 房间管理器实例
//   - registerBufferSize: register 通道的缓冲区大小
//   - unregisterBufferSize: unregister 通道的缓冲区大小
func NewClientManager(roomMgr *RoomManager, registerBufferSize, unregisterBufferSize int) *ClientManager {
	return &ClientManager{
		clients:    sync.Map{},
		register:   make(chan *model.Client, registerBufferSize),
		unregister: make(chan string, unregisterBufferSize),
		roomMgr:    roomMgr,
	}
}

// IsShuttingDown 返回当前连接管理器是否正在优雅关闭。
// 外部可根据此标志拒绝新连接。
func (cm *ClientManager) IsShuttingDown() bool {
	return cm.shuttingDown
}

// Register 向 register 通道发送客户端注册事件（非阻塞）。
// 若通道已满, 调用方会阻塞直到有空间。
func (cm *ClientManager) Register(client *model.Client) {
	cm.register <- client
}

// Unregister 向 unregister 通道发送客户端注销事件（非阻塞）。
// 若通道已满, 调用方会阻塞直到有空间。
func (cm *ClientManager) Unregister(clientID string) {
	cm.unregister <- clientID
}

// Init 启动 ClientManager 的主循环, 持续处理 register 和 unregister 事件。
// 该函数应在独立的 goroutine 中运行, 监听 ctx.Done() 信号退出。
// 参数:
//   - ctx: 上下文, 用于控制协程的生命周期
//   - controlWriteTimeout: 发送关闭帧的超时时间
func (cm *ClientManager) Init(ctx context.Context, controlWriteTimeout time.Duration) {
	for {
		select {
		case client := <-cm.register:
			_, loaded := cm.clients.LoadOrStore(client.ClientID, client)

			if loaded {
				if client.Conn != nil {
					err := client.Conn.WriteControl(websocket.CloseMessage,
						websocket.FormatCloseMessage(model.CloseCodeDuplicateID, "clientId already exists"),
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

// Get 根据 clientID 查询客户端。
// 返回客户端指针和是否存在标志。
func (cm *ClientManager) Get(clientID string) (*model.Client, bool) {
	val, ok := cm.clients.Load(clientID)
	if !ok {
		return nil, false
	}
	return val.(*model.Client), true
}

// Range 遍历所有在线客户端, 对每个客户端执行回调函数 fn。
// 若 fn 返回 false, 则停止遍历。
func (cm *ClientManager) Range(fn func(key any, val any) bool) {
	cm.clients.Range(fn)
}

// GetOnlineCount 获取当前在线客户端总数。
// 通过遍历 sync.Map 计算, 时间复杂度 O(n)。
func (cm *ClientManager) GetOnlineCount() int {
	count := 0
	cm.Range(func(_, _ any) bool {
		count++
		return true
	})
	return count
}

// Shutdown 优雅关闭连接管理器。
// 执行流程：
//  1. 设置 shuttingDown = true, 拒绝新连接
//  2. 向所有在线客户端发送 WebSocket 关闭帧（状态码 1000）
//  3. 等待宽限期（gracePeriod）结束
//  4. 强制清理所有客户端（关闭通道、移除房间、关闭连接）
//
// 参数:
//   - gracePeriod: 宽限期, 等待客户端主动断开
//   - controlWriteTimeout: 发送关闭帧的超时时间
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
