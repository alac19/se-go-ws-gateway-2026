package service

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// redisPingTimeout 是启动时检测 Redis 连通性的超时时间,
// 避免 Redis 不可达时网关启动被长时间阻塞。
const redisPingTimeout = 2 * time.Second

// NewRedisClient 创建 Redis 客户端并执行一次 PING 做连通性检查。
// 连接不可用时返回错误并关闭客户端, 由调用方决定是否降级为内存模式。
func NewRedisClient(addr, password string, db int) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), redisPingTimeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()

		return nil, err
	}

	return client, nil
}
