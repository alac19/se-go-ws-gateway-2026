import request from './request'

// 连接统计。注意：/api/stats 受网关 IP 限流（令牌桶：12 秒生成 1 个、桶容量 5），
// 轮询间隔不要低于 12 秒，否则连续第 6 次请求会返回 429
export function getStats() {
  return request.get('/api/stats')
}

// Prometheus 监控指标：/metrics 不在 /api 组内，匿名可访问、不参与限流，
// 因此可以用更短的间隔轮询。返回看板需要的几个计数值
export async function getMetricsSummary() {
  const res = await fetch('http://localhost:8080/metrics')
  if (!res.ok) {
    throw new Error(`监控指标请求失败（${res.status}）`)
  }
  return parseMetrics(await res.text())
}

// Prometheus 文本格式每行形如：ws_gateway_msg_sent_total{msg_type="room"} 5
// 同名指标可能带不同标签出现多行，这里按前缀匹配后求和
function parseMetrics(text) {
  const sumByPrefix = (prefix) => {
    let total = 0
    for (const line of text.split('\n')) {
      if (line.startsWith(prefix)) {
        const value = Number.parseFloat(line.slice(line.lastIndexOf(' ') + 1))
        if (Number.isFinite(value)) {
          total += value
        }
      }
    }
    return total
  }
  return {
    msgSent: sumByPrefix('ws_gateway_msg_sent_total'),
    msgSendFail: sumByPrefix('ws_gateway_msg_send_fail_total'),
    connEvents: sumByPrefix('ws_connection_events_total'),
    msgReceived: sumByPrefix('ws_messages_received_total'),
  }
}
