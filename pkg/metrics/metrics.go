package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// 消息发送总量，区分single/room/broadcast
var MsgSendTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "ws_gateway_msg_sent_total",
		Help: "网关推送消息总次数",
	},
	[]string{"msg_type"},
)

// 消息发送失败计数器
var MsgSendFail = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "ws_gateway_msg_send_fail_total",
		Help: "网关推送消息失败次数",
	},
	[]string{"reason"},
)

// 消息接收总量
var MsgRecvTotal = promauto.NewCounter(
	prometheus.CounterOpts{
		Name: "ws_messages_received_total",
		Help: "网关接收消息总次数",
	},
)

// 当前在线连接数
var OnlineConnGauge = promauto.NewGauge(
	prometheus.GaugeOpts{
		Name: "ws_active_connections",
		Help: "当前在线WebSocket长连接数量",
	},
)

// 连接建立/关闭事件计数
var ConnEventTotal = promauto.NewCounter(
	prometheus.CounterOpts{
		Name: "ws_connection_events_total",
		Help: "网关Websocket长连接建立/关闭事件总数",
	},
)
