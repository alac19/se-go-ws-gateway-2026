// Package metrics provides Prometheus metrics collection for the gateway service.
// It defines and registers metrics for connection counts, message throughput, and errors.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// MsgSendTotal 消息发送总量, 按消息类型（single/room/broadcast）区分标签。
	MsgSendTotal *prometheus.CounterVec
	// MsgSendFail 消息发送失败计数器, 按失败原因（offline/block）区分标签。
	MsgSendFail *prometheus.CounterVec
	// MsgRecvTotal 消息接收总量（客户端发送到网关的消息）。
	MsgRecvTotal prometheus.Counter
	// OnlineConnGauge 当前在线 WebSocket 长连接数（瞬时值）。
	OnlineConnGauge prometheus.Gauge
	// ConnEventTotal 连接建立/关闭事件总计数。
	ConnEventTotal prometheus.Counter
)

// Init 初始化所有 Prometheus 指标, 并注册到指定的注册器。
// 如果 reg 为 nil, 则使用 prometheus.DefaultRegisterer 默认注册器。
// 该函数应在程序启动时调用一次, 重复调用会导致注册冲突。
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
