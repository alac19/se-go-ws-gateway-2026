<script setup>
import { onBeforeUnmount, onMounted, ref } from 'vue'
import * as echarts from 'echarts'
import AppHeader from '../components/AppHeader.vue'
import { getMetricsSummary, getStats } from '../api/stats'

// /api/stats 受网关限流（12 秒 1 个令牌），轮询间隔取 20 秒留出余量；
// /metrics 不参与限流，可以更勤快一些
const STATS_INTERVAL = 20000
const METRICS_INTERVAL = 10000

const onlineConnections = ref(0)
const rooms = ref([])
const uptimeSeconds = ref(0)
const metrics = ref({ msgSent: 0, msgSendFail: 0, connEvents: 0, msgReceived: 0 })
const lastUpdated = ref('')
const statsError = ref('')
const refreshing = ref(false)

let statsTimer = null
let metricsTimer = null
let barChart = null
let pieChart = null

function formatUptime(seconds) {
  const d = Math.floor(seconds / 86400)
  const h = Math.floor((seconds % 86400) / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  const parts = []
  if (d) parts.push(`${d} 天`)
  if (d || h) parts.push(`${h} 小时`)
  parts.push(`${m} 分钟`)
  return parts.join(' ')
}

function formatCount(n) {
  return Math.round(n).toLocaleString('zh-CN')
}

async function refreshStats() {
  refreshing.value = true
  try {
    const res = await getStats()
    onlineConnections.value = res.data.online_connections
    rooms.value = Object.entries(res.data.all_rooms_connections_stats).map(([name, count]) => ({
      name,
      count,
    }))
    uptimeSeconds.value = res.data.gateway_server_initial_time
    statsError.value = ''
    lastUpdated.value = new Date().toLocaleTimeString('zh-CN')
    renderCharts()
  } catch (err) {
    statsError.value = err.message || '获取统计数据失败'
  } finally {
    refreshing.value = false
  }
}

async function refreshMetrics() {
  try {
    metrics.value = await getMetricsSummary()
  } catch {
    // 指标获取失败不打扰用户，下次轮询会重试
  }
}

function renderCharts() {
  barChart?.setOption({
    tooltip: { trigger: 'axis' },
    grid: { left: 48, right: 24, top: 40, bottom: 32 },
    xAxis: { type: 'category', data: rooms.value.map((r) => r.name) },
    yAxis: { type: 'value', minInterval: 1 },
    series: [
      {
        type: 'bar',
        data: rooms.value.map((r) => r.count),
        barMaxWidth: 48,
        itemStyle: { color: '#409eff', borderRadius: [4, 4, 0, 0] },
      },
    ],
  })
  pieChart?.setOption({
    tooltip: { trigger: 'item', formatter: '{b}: {c} 个连接（{d}%）' },
    legend: { bottom: 0 },
    series: [
      {
        type: 'pie',
        radius: ['40%', '65%'],
        center: ['50%', '44%'],
        data: rooms.value.map((r) => ({ name: r.name, value: r.count })),
        label: { formatter: '{b}\n{c} 连接' },
      },
    ],
  })
}

function handleResize() {
  barChart?.resize()
  pieChart?.resize()
}

onMounted(() => {
  barChart = echarts.init(document.getElementById('room-bar-chart'))
  pieChart = echarts.init(document.getElementById('room-pie-chart'))
  window.addEventListener('resize', handleResize)
  refreshStats()
  refreshMetrics()
  statsTimer = setInterval(refreshStats, STATS_INTERVAL)
  metricsTimer = setInterval(refreshMetrics, METRICS_INTERVAL)
})

onBeforeUnmount(() => {
  clearInterval(statsTimer)
  clearInterval(metricsTimer)
  window.removeEventListener('resize', handleResize)
  barChart?.dispose()
  pieChart?.dispose()
})
</script>

<template>
  <div class="page">
    <AppHeader title="监控看板">
      <span class="updated">更新于 {{ lastUpdated || '—' }}</span>
      <button class="refresh" :disabled="refreshing" @click="refreshStats(); refreshMetrics()">
        {{ refreshing ? '刷新中…' : '立即刷新' }}
      </button>
    </AppHeader>

    <main class="content">
      <p v-if="statsError" class="error-banner">⚠ {{ statsError }}（数据展示的是最近一次成功结果）</p>

      <section class="cards">
        <div class="card highlight">
          <span class="card-label">在线连接数</span>
          <span class="card-value">{{ formatCount(onlineConnections) }}</span>
        </div>
        <div class="card">
          <span class="card-label">在线房间数</span>
          <span class="card-value">{{ rooms.length }}</span>
        </div>
        <div class="card">
          <span class="card-label">服务运行时长</span>
          <span class="card-value card-value--sm">{{ formatUptime(uptimeSeconds) }}</span>
        </div>
      </section>

      <section class="cards cards--metrics">
        <div class="card card--mini">
          <span class="card-label">消息发送总数</span>
          <span class="card-value card-value--mini">{{ formatCount(metrics.msgSent) }}</span>
        </div>
        <div class="card card--mini">
          <span class="card-label">消息发送失败</span>
          <span class="card-value card-value--mini">{{ formatCount(metrics.msgSendFail) }}</span>
        </div>
        <div class="card card--mini">
          <span class="card-label">连接事件总数</span>
          <span class="card-value card-value--mini">{{ formatCount(metrics.connEvents) }}</span>
        </div>
        <div class="card card--mini">
          <span class="card-label">上行消息总数</span>
          <span class="card-value card-value--mini">{{ formatCount(metrics.msgReceived) }}</span>
        </div>
      </section>

      <section class="charts">
        <div class="chart-box">
          <h3>各房间连接数分布</h3>
          <div id="room-bar-chart" class="chart" />
        </div>
        <div class="chart-box">
          <h3>房间连接占比</h3>
          <div id="room-pie-chart" class="chart" />
        </div>
      </section>
      <p v-if="!rooms.length && !statsError" class="empty-tip">
        当前没有客户端在线。可以打开广播页建立 WebSocket 连接，或用多个浏览器标签页连上来观察数据变化。
      </p>
    </main>
  </div>
</template>

<style scoped>
.page {
  min-height: 100vh;
}

.updated {
  font-size: 12px;
  color: #909399;
}

.refresh {
  height: 32px;
  padding: 0 14px;
  margin-right: -8px;
  border: 1px solid #b3d8ff;
  border-radius: 6px;
  background: #fff;
  font-size: 13px;
  color: #409eff;
  cursor: pointer;
}

.refresh:hover {
  border-color: #409eff;
}

.content {
  padding: 24px;
}

.error-banner {
  margin-bottom: 16px;
  padding: 10px 16px;
  border-radius: 6px;
  background: #fef0f0;
  color: #f56c6c;
  font-size: 13px;
}

.cards {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
  margin-bottom: 16px;
}

.cards--metrics {
  grid-template-columns: repeat(4, 1fr);
}

.card {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 20px 24px;
  background: #fff;
  border-radius: 10px;
  box-shadow: 0 1px 4px rgba(0, 21, 41, 0.06);
}

.card-label {
  font-size: 13px;
  color: #909399;
}

.card-value {
  font-size: 34px;
  font-weight: 700;
  color: #303133;
}

.highlight .card-value {
  color: #409eff;
}

.card-value--sm {
  font-size: 24px;
}

.card-value--mini {
  font-size: 22px;
}

.charts {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

.chart-box {
  padding: 20px;
  background: #fff;
  border-radius: 10px;
  box-shadow: 0 1px 4px rgba(0, 21, 41, 0.06);
}

.chart-box h3 {
  margin-bottom: 8px;
  font-size: 15px;
  color: #303133;
}

.chart {
  width: 100%;
  height: 300px;
}

.empty-tip {
  margin-top: 16px;
  text-align: center;
  font-size: 13px;
  color: #909399;
}
</style>
