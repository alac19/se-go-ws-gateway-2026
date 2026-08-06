// Package service provides the core business logic for the WebSocket gateway.
// It includes connection management, room management, and message routing.
package service

import (
	"sync"
)

// RoomManager 房间管理器，读写锁保证并发安全
type RoomManager struct {
	mu    sync.RWMutex
	rooms map[string]map[string]bool // roomId -> clientId集合
}

// NewRoomManager 创建房间管理器实例。
func NewRoomManager() *RoomManager {
	return &RoomManager{
		rooms: make(map[string]map[string]bool),
	}
}

// Join 客户端加入房间
func (rm *RoomManager) Join(roomID, clientID string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	// 房间不存在则初始化
	if _, ok := rm.rooms[roomID]; !ok {
		rm.rooms[roomID] = make(map[string]bool)
	}
	rm.rooms[roomID][clientID] = true
}

// Leave 客户端离开单个房间
func (rm *RoomManager) Leave(roomID, clientID string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	members, ok := rm.rooms[roomID]
	if !ok {
		return
	}
	delete(members, clientID)
	// 房间无人则删除房间key
	if len(members) == 0 {
		delete(rm.rooms, roomID)
	}
}

// GetClients 获取房间内全部客户端ID列表
func (rm *RoomManager) GetClients(roomID string) []string {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	members, ok := rm.rooms[roomID]
	if !ok {
		return nil
	}
	list := make([]string, 0, len(members))
	for cid := range members {
		list = append(list, cid)
	}
	return list
}

// RemoveClientFromAllRooms 从所有房间移除指定客户端
func (rm *RoomManager) RemoveClientFromAllRooms(clientID string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	for roomID, members := range rm.rooms {
		delete(members, clientID)
		if len(members) == 0 {
			delete(rm.rooms, roomID)
		}
	}
}

// GetAllRoomsConnStats 获取所有房间的连接统计。
// 返回 map[string]int, key 为 roomId, value 为该房间的在线客户端数。
func (rm *RoomManager) GetAllRoomsConnStats() map[string]int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	connStats := make(map[string]int)

	for roomID, members := range rm.rooms {
		connStats[roomID] = len(members)
	}

	return connStats
}

// HasRoom 检查指定房间是否存在（即是否有客户端在线）。
// 返回 true 表示房间存在且至少有一个客户端。
func (rm *RoomManager) HasRoom(roomID string) bool {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	_, ok := rm.rooms[roomID]

	return ok
}
