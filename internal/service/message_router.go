// Package service provides the core business logic for the WebSocket gateway.
// It includes connection management, room management, and message routing.
package service

import (
	"context"
	"sync"

	"github.com/alac/se-go-ws-gateway-2026/internal/model"
	"github.com/alac/se-go-ws-gateway-2026/pkg/metrics"
	"github.com/go-redis/redis/v8"
)

// MessageRouter 消息路由核心，负责各类消息分发
type MessageRouter struct {
	clientMgr *ClientManager
	roomMgr   *RoomManager
	redisCli  *redis.Client // 分布式集群同步
	ctx       context.Context
}

var routerOnce sync.Once
var globalRouter *MessageRouter

// NewMessageRouter 创建全局路由单例
func NewMessageRouter(cm *ClientManager, rm *RoomManager, redisCli *redis.Client) *MessageRouter {
	routerOnce.Do(func() {
		globalRouter = &MessageRouter{
			clientMgr: cm,
			roomMgr:   rm,
			redisCli:  redisCli,
			ctx:       context.Background(),
		}
		// 有Redis则启动订阅，接收其他网关实例推送
		if redisCli != nil {
			go globalRouter.subscribeRedis()
		}
	})
	return globalRouter
}

// GetRouter 获取全局路由实例
func GetRouter() *MessageRouter {
	return globalRouter
}

// SendSingle 点对点单播推送
func (r *MessageRouter) SendSingle(clientId string, msg *model.Message) bool {
	metrics.MsgSendTotal.WithLabelValues("single").Inc()

	cli, ok := r.clientMgr.Get(clientId)
	if !ok {
		metrics.MsgSendFail.WithLabelValues("single_offline").Inc()
		return false
	}

	// 非阻塞写入发送通道，缓冲区满判定慢客户端，自动下线
	select {
	case cli.SendChan <- msg.Payload:
		return true
	default:
		metrics.MsgSendFail.WithLabelValues("single_block").Inc()
		r.clientMgr.Unregister(clientId)
		return false
	}
}

// SendRoom 房间全员广播
func (r *MessageRouter) SendRoom(roomId string, msg *model.Message) {
	metrics.MsgSendTotal.WithLabelValues("room").Inc()

	clientIDs := r.roomMgr.GetClients(roomId)
	for _, cid := range clientIDs {
		r.SendSingle(cid, msg)
	}

	// 分布式同步：推送至Redis Pub/Sub，其他网关同步推送
	if r.redisCli != nil {
		_ = r.redisCli.Publish(r.ctx, "ws:room:"+roomId, msg.Payload).Err()
	}
}

// SendBroadcast 全局全服广播
func (r *MessageRouter) SendBroadcast(msg *model.Message) {
	metrics.MsgSendTotal.WithLabelValues("broadcast").Inc()

	// 遍历全部在线客户端
	r.clientMgr.Range(func(_, val any) bool {
		cli := val.(*model.Client)
		select {
		case cli.SendChan <- msg.Payload:
		default:
			metrics.MsgSendFail.WithLabelValues("broadcast_block").Inc()
			r.clientMgr.Unregister(cli.ClientID)
		}
		return true
	})

	// 分布式集群同步
	if r.redisCli != nil {
		_ = r.redisCli.Publish(r.ctx, "ws:broadcast", msg.Payload).Err()
	}
}

// subscribeRedis 监听Redis集群消息，同步其他网关推送内容到本地连接
func (r *MessageRouter) subscribeRedis() {
	sub := r.redisCli.Subscribe(r.ctx, "ws:broadcast", "ws:room:*")
	defer sub.Close()
	ch := sub.Channel()
	for pubMsg := range ch {
		payload := []byte(pubMsg.Payload)
		msg := &model.Message{Payload: payload}
		switch pubMsg.Channel {
		case "ws:broadcast":
			r.SendBroadcast(msg)
		default:
			// 截取roomId: ws:room:room100 → room100
			roomId := pubMsg.Channel[len("ws:room:"):]
			r.SendRoom(roomId, msg)
		}
	}
}
