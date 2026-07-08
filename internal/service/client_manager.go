package service

import (
	"sync"

	"github.com/alac/se-go-ws-gateway-2026/internal/model"
)

// ClientManager 连接管理器，并发安全使用sync.Map
type ClientManager struct {
	onlineClients *sync.Map // key: clientID, value: *model.Client
	roomMgr       *RoomManager
}

// NewClientManager 创建连接管理器实例，注入房间管理器
func NewClientManager(rm *RoomManager) *ClientManager {
	return &ClientManager{
		onlineClients: &sync.Map{},
		roomMgr:       rm,
	}
}

// Register 注册客户端，加入连接池并加入对应房间
func (cm *ClientManager) Register(client *model.Client) {
	cm.onlineClients.Store(client.ClientID, client)
	if client.RoomID != "" {
		cm.roomMgr.Join(client.RoomID, client.ClientID)
	}
}

// Unregister 注销客户端，移除连接并退出所有房间
func (cm *ClientManager) Unregister(clientID string) {
	val, ok := cm.onlineClients.LoadAndDelete(clientID)
	if !ok {
		return
	}
	cli := val.(*model.Client)
	// 关闭发送通道
	close(cli.SendChan)
	// 从全部房间移除该客户端
	cm.roomMgr.RemoveClientFromAllRooms(clientID)
	// 关闭ws连接
	_ = cli.Conn.Close()
}

// Get 根据clientID查询客户端
func (cm *ClientManager) Get(clientID string) (*model.Client, bool) {
	val, ok := cm.onlineClients.Load(clientID)
	if !ok {
		return nil, false
	}
	return val.(*model.Client), true
}

// Range 遍历所有在线客户端，回调处理
func (cm *ClientManager) Range(fn func(key any, val any) bool) {
	cm.onlineClients.Range(fn)
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