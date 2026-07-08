package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// 消息发送总量，区分single/room/broadcast
var MsgSendTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "ws_gateway_msg_send_total",
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

// 当前在线连接数
var OnlineConnGauge = promauto.NewGauge(
	prometheus.GaugeOpts{
		Name: "ws_gateway_online_conn",
		Help: "当前在线WebSocket长连接数量",
	},
)