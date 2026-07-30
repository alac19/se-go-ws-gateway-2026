package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// 消息发送总量，区分single/room/broadcast
	MsgSendTotal *prometheus.CounterVec
	// 消息发送失败计数器
	MsgSendFail *prometheus.CounterVec
	// 消息接收总量
	MsgRecvTotal prometheus.Counter
	// 当前在线连接数
	OnlineConnGauge prometheus.Gauge
	// 连接建立/关闭事件计数
	ConnEventTotal prometheus.Counter
)

// Init 初始化所有指标，并注册到指定的注册器。
// 如果 reg 为 nil，则使用默认注册器（prometheus.DefaultRegisterer）。
func Init(reg prometheus.Registerer) {
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}

	MsgSendTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ws_gateway_msg_sent_total",
			Help: "网关推送消息总次数",
		},
		[]string{"msg_type"},
	)
	reg.MustRegister(MsgSendTotal)

	MsgSendFail = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ws_gateway_msg_send_fail_total",
			Help: "网关推送消息失败次数",
		},
		[]string{"reason"},
	)
	reg.MustRegister(MsgSendFail)

	MsgRecvTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "ws_messages_received_total",
			Help: "网关接收消息总次数",
		},
	)
	reg.MustRegister(MsgRecvTotal)

	OnlineConnGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "ws_active_connections",
			Help: "当前在线WebSocket长连接数量",
		},
	)
	reg.MustRegister(OnlineConnGauge)

	ConnEventTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "ws_connection_events_total",
			Help: "网关Websocket长连接建立/关闭事件总数",
		},
	)
	reg.MustRegister(ConnEventTotal)
}
