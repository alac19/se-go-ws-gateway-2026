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
// clientId: 目标客户端ID
// msg: 待推送消息实体
func (r *MessageRouter) SendSingle(clientId string, msg *model.Message) bool {
	// 指标计数：单播消息总量
	metrics.MsgSendTotal.WithLabelValues("single").Inc()

	// cli, ok := r.clientMgr.GetClient(clientId)
	// if !ok {
	// 	metrics.MsgSendFail.WithLabelValues("single_offline").Inc()
	// 	return false
	// }
	// // 非阻塞写入发送通道，慢客户端自动剔除
	// select {
	// case cli.SendChan <- msg.Payload:
	// default:
	// 	metrics.MsgSendFail.WithLabelValues("single_block").Inc()
	// 	r.clientMgr.RemoveClient(clientId)
	// 	return false
	// }
	return true
}

// SendRoom 房间全员广播
func (r *MessageRouter) SendRoom(roomId string, msg *model.Message) {
	metrics.MsgSendTotal.WithLabelValues("room").Inc()
	// clients := r.roomMgr.GetRoomClients(roomId)
	// for _, cli := range clients {
	// 	select {
	// 	case cli.SendChan <- msg.Payload:
	// 	default:
	// 		metrics.MsgSendFail.WithLabelValues("room_block").Inc()
	// 		r.clientMgr.RemoveClient(cli.ClientID)
	// 	}
	// }

	// 分布式同步：推送至Redis Pub/Sub，其他网关同步推送
	if r.redisCli != nil {
		_ = r.redisCli.Publish(r.ctx, "ws:room:"+roomId, msg.Payload).Err()
	}
}

// SendBroadcast 全局全服广播
func (r *MessageRouter) SendBroadcast(msg *model.Message) {
	metrics.MsgSendTotal.WithLabelValues("broadcast").Inc()
	// allClients := r.clientMgr.GetAllClients()
	// for _, cli := range allClients {
	// 	select {
	// 	case cli.SendChan <- msg.Payload:
	// 	default:
	// 		metrics.MsgSendFail.WithLabelValues("broadcast_block").Inc()
	// 		r.clientMgr.RemoveClient(cli.ClientID)
	// 	}
	// }

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
