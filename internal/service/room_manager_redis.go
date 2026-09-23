package service

import (
	"context"
	"github.com/redis/go-redis/v9"
)

// RedisRoomManager Redis分布式房间管理器，实现和内存版一致的行为
// key设计:
// room:{roomId} → Set集合，存储该房间下所有clientId
type RedisRoomManager struct {
	client *redis.Client
	ctx    context.Context
}

// 编译期保证 Redis 实现满足统一的房间管理器接口
var _ IRoomManager = (*RedisRoomManager)(nil)

// NewRedisRoomManager 基于已建立的 Redis 客户端创建分布式房间管理器。
// 客户端由 main 持有并在退出时统一关闭, 与消息路由共用同一个连接池。
func NewRedisRoomManager(client *redis.Client) *RedisRoomManager {
	return &RedisRoomManager{
		client: client,
		ctx:    context.Background(),
	}
}

// Join 客户端加入房间
func (r *RedisRoomManager) Join(roomID, clientID string) error {
	key := "room:" + roomID
	_, err := r.client.SAdd(r.ctx, key, clientID).Result()
	return err
}

// Leave 客户端离开单个房间
func (r *RedisRoomManager) Leave(roomID, clientID string) error {
	key := "room:" + roomID
	_, err := r.client.SRem(r.ctx, key, clientID).Result()
	if err != nil {
		return err
	}
	// 如果集合为空，删除key
	cnt, err := r.client.SCard(r.ctx, key).Result()
	if err != nil {
		return err
	}
	if cnt == 0 {
		_, _ = r.client.Del(r.ctx, key).Result()
	}
	return nil
}

// GetClients 获取房间内全部客户端ID列表
func (r *RedisRoomManager) GetClients(roomID string) ([]string, error) {
	key := "room:" + roomID
	exist, err := r.client.Exists(r.ctx, key).Result()
	if err != nil {
		return nil, err
	}
	if exist == 0 {
		return nil, nil
	}
	members, err := r.client.SMembers(r.ctx, key).Result()
	return members, err
}

// RemoveClientFromAllRooms 将客户端从全部房间移除
// 注意：Redis没有直接遍历全部room:*的高性能接口；
// 生产建议维护client→rooms反向集合；实训场景这里做简易实现。
func (r *RedisRoomManager) RemoveClientFromAllRooms(clientID string) error {
	// 实训简化：实际项目建议维护反向索引 client:{cid} → set(roomIds)
	// 本版本为了实训演示，使用scan遍历room:* key
	var cursor uint64
	for {
		keys, nextCursor, err := r.client.Scan(r.ctx, cursor, "room:*", 100).Result()
		if err != nil {
			return err
		}
		for _, k := range keys {
			_, _ = r.client.SRem(r.ctx, k, clientID).Result()
			// 删除空房间
			cnt, _ := r.client.SCard(r.ctx, k).Result()
			if cnt == 0 {
				_, _ = r.client.Del(r.ctx, k).Result()
			}
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return nil
}

// GetAllRoomsConnStats 获取所有房间连接统计
func (r *RedisRoomManager) GetAllRoomsConnStats() (map[string]int, error) {
	stats := make(map[string]int)
	var cursor uint64
	for {
		keys, nextCursor, err := r.client.Scan(r.ctx, cursor, "room:*", 100).Result()
		if err != nil {
			return nil, err
		}
		for _, fullKey := range keys {
			roomID := fullKey[len("room:"):]
			n, _ := r.client.SCard(r.ctx, fullKey).Result()
			stats[roomID] = int(n)
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return stats, nil
}

// HasRoom 判断房间是否存在
func (r *RedisRoomManager) HasRoom(roomID string) (bool, error) {
	key := "room:" + roomID
	n, err := r.client.Exists(r.ctx, key).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
