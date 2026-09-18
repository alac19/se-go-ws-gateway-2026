<script setup>
import { onBeforeUnmount, ref } from 'vue'
import { useRouter } from 'vue-router'
import { broadcastAll, broadcastRoom, sendToClient } from '../api/broadcast'
import { getStats } from '../api/stats'

const router = useRouter()

// ==================== 发送操作区 ====================
const activeTab = ref('all') // all | room | single
const message = ref('')
const roomId = ref('room1')
const clientId = ref('')
const sending = ref(false)
const sendResult = ref(null) // { ok, text }
const knownRooms = ref([])

const TABS = [
  { key: 'all', label: '全服广播' },
  { key: 'room', label: '房间广播' },
  { key: 'single', label: '单播' },
]

async function loadRoomSuggestions() {
  try {
    const res = await getStats()
    knownRooms.value = Object.keys(res.data.all_rooms_connections_stats)
  } catch {
    // 房间建议列表拉取失败不影响发送功能
  }
}

async function handleSend() {
  sendResult.value = null
  const content = message.value.trim()
  if (!content) {
    sendResult.value = { ok: false, text: '请输入消息内容' }
    return
  }
  if (activeTab.value === 'room' && !roomId.value.trim()) {
    sendResult.value = { ok: false, text: '请输入目标房间 ID' }
    return
  }
  if (activeTab.value === 'single' && !clientId.value.trim()) {
    sendResult.value = { ok: false, text: '请输入目标客户端 ID' }
    return
  }
  sending.value = true
  try {
    if (activeTab.value === 'all') {
      await broadcastAll(content)
    } else if (activeTab.value === 'room') {
      await broadcastRoom(roomId.value.trim(), content)
    } else {
      await sendToClient(clientId.value.trim(), content)
    }
    sendResult.value = { ok: true, text: '发送成功，网关已推送' }
    message.value = ''
  } catch (err) {
    sendResult.value = { ok: false, text: err.message || '发送失败' }
  } finally {
    sending.value = false
  }
}

// ==================== WebSocket 收信面板 ====================
// 浏览器端 WS 无法自定义请求头，token 走 query 参数；
// clientId 必须是「账号名」或「账号名-后缀」，这里用随机后缀避免多开冲突
const CLOSE_CODE_TEXT = {
  1000: '正常关闭',
  1001: '服务正在关闭（优雅退出）',
  4000: '参数缺失：clientId 或 roomId 为空',
  4001: '参数格式无效（仅允许字母、数字、下划线、短横线）',
  4002: 'clientId 已被占用，请重试（会自动换新后缀）',
  4004: 'clientId 与登录账号不符',
}

const wsRoom = ref('room1')
const wsStatus = ref('idle') // idle | connecting | open | closed
const wsStatusText = ref('未连接')
const myClientId = ref('')
const messages = ref([]) // { id, time, kind: 'msg'|'system', raw }
const socket = ref(null)
let msgSeq = 0

function appendMessage(kind, raw) {
  messages.value.unshift({
    id: ++msgSeq,
    time: new Date().toLocaleTimeString('zh-CN', { hour12: false }),
    kind,
    raw,
  })
  if (messages.value.length > 50) {
    messages.value.length = 50
  }
}

function connectWs() {
  disconnectWs()
  const token = localStorage.getItem('token')
  if (!token) {
    router.push({ name: 'login' })
    return
  }
  const room = wsRoom.value.trim() || 'room1'
  wsRoom.value = room
  myClientId.value = `zhangsan-ui-${Math.floor(1000 + Math.random() * 9000)}`
  wsStatus.value = 'connecting'
  wsStatusText.value = '连接中…'

  const url =
    `ws://localhost:8080/ws?clientId=${myClientId.value}` +
    `&roomId=${encodeURIComponent(room)}&token=${encodeURIComponent(token)}`
  const s = new WebSocket(url)
  socket.value = s

  s.onopen = () => {
    wsStatus.value = 'open'
    wsStatusText.value = `已连接：${myClientId.value} @ ${room}`
    appendMessage('system', `连接成功，已加入房间 ${room}`)
  }
  s.onmessage = (e) => {
    let pretty = e.data
    try {
      pretty = JSON.stringify(JSON.parse(e.data), null, 2)
    } catch {
      // 非 JSON 文本原样展示
    }
    appendMessage('msg', pretty)
  }
  s.onclose = (e) => {
    socket.value = null
    wsStatus.value = 'closed'
    if (e.code === 1006) {
      wsStatusText.value = '连接建立失败：令牌可能已过期（握手 401），请退出重新登录后再试'
    } else {
      const reason = CLOSE_CODE_TEXT[e.code] || `关闭码 ${e.code}`
      wsStatusText.value = `连接关闭：${reason}`
    }
    appendMessage('system', wsStatusText.value)
  }
  s.onerror = () => {
    // 错误详情由 onclose 的关闭码给出，这里只标记
  }
}

function disconnectWs() {
  if (socket.value) {
    const s = socket.value
    socket.value = null
    s.onclose = null
    s.close(1000)
  }
  wsStatus.value = 'idle'
  wsStatusText.value = '未连接'
}

function clearMessages() {
  messages.value = []
}

function handleLogout() {
  localStorage.removeItem('token')
  router.push({ name: 'login' })
}

onBeforeUnmount(() => {
  if (socket.value) {
    socket.value.onclose = null
    socket.value.close(1000)
    socket.value = null
  }
})

loadRoomSuggestions()
</script>

<template>
  <div class="page">
    <header class="topbar">
      <span class="title">消息推送</span>
      <nav>
        <router-link to="/dashboard">看板</router-link>
        <router-link to="/broadcast">消息推送</router-link>
      </nav>
      <button class="ghost" @click="handleLogout">
        退出登录
      </button>
    </header>

    <main class="content">
      <div class="layout">
        <!-- 左：发送操作区 -->
        <section class="panel">
          <div class="tabs">
            <button
              v-for="t in TABS"
              :key="t.key"
              class="tab"
              :class="{ active: activeTab === t.key }"
              @click="activeTab = t.key; sendResult = null"
            >
              {{ t.label }}
            </button>
          </div>

          <div class="form">
            <div v-if="activeTab === 'room'" class="field">
              <label for="room-id">目标房间 ID</label>
              <input id="room-id" v-model="roomId" list="known-rooms" placeholder="例如 room1" />
              <datalist id="known-rooms">
                <option v-for="r in knownRooms" :key="r" :value="r" />
              </datalist>
              <p class="hint">当前在线房间：{{ knownRooms.length ? knownRooms.join('、') : '无' }}</p>
            </div>

            <div v-if="activeTab === 'single'" class="field">
              <label for="client-id">目标客户端 ID</label>
              <input id="client-id" v-model="clientId" placeholder="例如 zhangsan-bot1 或 zhangsan-ui-1234" />
              <p class="hint">对方离线或不存在时网关返回 404</p>
            </div>

            <div class="field">
              <label for="msg-content">消息内容</label>
              <textarea
                id="msg-content"
                v-model="message"
                rows="5"
                :placeholder="activeTab === 'all' ? '将推送给网关全部在线客户端…' : activeTab === 'room' ? '将推送给该房间内的所有在线客户端…' : '将只推送给这一条连接…'"
              />
            </div>

            <p v-if="sendResult" class="result" :class="sendResult.ok ? 'result--ok' : 'result--err'">
              {{ sendResult.ok ? '✓ ' : '✕ ' }}{{ sendResult.text }}
            </p>

            <button class="primary" :disabled="sending" @click="handleSend">
              {{ sending ? '发送中…' : '发 送' }}
            </button>
            <p class="hint hint--rate">注意：网关按 IP 限流（12 秒约 1 个令牌），连发第 6 次请求会被 429 拒绝</p>
          </div>
        </section>

        <!-- 右：WebSocket 收信面板 -->
        <section class="panel">
          <div class="panel-head">
            <h3>收信面板（WebSocket）</h3>
            <span class="badge" :class="`badge--${wsStatus}`">
              {{ wsStatus === 'open' ? '● 已连接' : wsStatus === 'connecting' ? '● 连接中' : '○ ' + (wsStatus === 'idle' ? '未连接' : '已断开') }}
            </span>
          </div>

          <div class="connect-row">
            <input v-model="wsRoom" list="ws-rooms" placeholder="加入的房间，例如 room1" :disabled="wsStatus === 'open'" />
            <datalist id="ws-rooms">
              <option v-for="r in knownRooms" :key="r" :value="r" />
            </datalist>
            <button v-if="wsStatus !== 'open'" class="primary" :disabled="wsStatus === 'connecting'" @click="connectWs">
              {{ wsStatus === 'connecting' ? '连接中…' : '连 接' }}
            </button>
            <button v-else class="danger" @click="disconnectWs">断 开</button>
          </div>
          <p class="status-line" :class="{ 'status-line--ok': wsStatus === 'open' }">{{ wsStatusText }}</p>

          <div class="msg-list">
            <p v-if="!messages.length" class="empty-tip">
              还没有消息。连接后用左侧操作区发一条广播试试（自己也能收到全服/房间广播）。
            </p>
            <div
              v-for="m in messages"
              :key="m.id"
              class="msg"
              :class="{ 'msg--system': m.kind === 'system' }"
            >
              <span class="msg-time">{{ m.time }}</span>
              <pre class="msg-body">{{ m.raw }}</pre>
            </div>
          </div>

          <button class="ghost" :disabled="!messages.length" @click="clearMessages">清空列表</button>
        </section>
      </div>
    </main>
  </div>
</template>

<style scoped>
.page {
  min-height: 100vh;
}

.topbar {
  display: flex;
  align-items: center;
  gap: 24px;
  height: 56px;
  padding: 0 24px;
  background: #fff;
  box-shadow: 0 1px 4px rgba(0, 21, 41, 0.08);
}

.title {
  font-size: 16px;
  font-weight: 600;
  color: #1f2d3d;
}

nav {
  display: flex;
  gap: 16px;
  flex: 1;
}

nav a {
  font-size: 14px;
  color: #606266;
  text-decoration: none;
}

nav a.router-link-active {
  color: #409eff;
}

.content {
  padding: 24px;
}

.layout {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
  align-items: start;
}

.panel {
  padding: 20px;
  background: #fff;
  border-radius: 10px;
  box-shadow: 0 1px 4px rgba(0, 21, 41, 0.06);
}

.panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.panel-head h3 {
  font-size: 15px;
  color: #303133;
}

.tabs {
  display: flex;
  gap: 8px;
  margin-bottom: 18px;
}

.tab {
  flex: 1;
  height: 36px;
  border: 1px solid #dcdfe6;
  border-radius: 6px;
  background: #fff;
  font-size: 14px;
  color: #606266;
  cursor: pointer;
}

.tab.active {
  color: #409eff;
  border-color: #409eff;
  background: #ecf5ff;
}

.form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

label {
  font-size: 13px;
  color: #606266;
}

input,
textarea {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid #dcdfe6;
  border-radius: 6px;
  font-size: 14px;
  font-family: inherit;
  outline: none;
  transition: border-color 0.2s;
}

input:focus,
textarea:focus {
  border-color: #409eff;
}

textarea {
  resize: vertical;
}

.hint {
  font-size: 12px;
  color: #909399;
}

.result {
  font-size: 13px;
}

.result--ok {
  color: #67c23a;
}

.result--err {
  color: #f56c6c;
}

.primary,
.danger,
.ghost {
  height: 38px;
  border-radius: 6px;
  font-size: 14px;
  cursor: pointer;
}

.primary {
  border: none;
  background: #409eff;
  color: #fff;
}

.primary:hover {
  background: #66b1ff;
}

.primary:disabled {
  background: #a0cfff;
  cursor: not-allowed;
}

.danger {
  border: 1px solid #fbc4c4;
  background: #fef0f0;
  color: #f56c6c;
}

.ghost {
  border: 1px solid #dcdfe6;
  background: #fff;
  color: #606266;
}

.ghost:hover {
  color: #409eff;
  border-color: #409eff;
}

.connect-row {
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
}

.connect-row input {
  flex: 1;
}

.connect-row .primary,
.connect-row .danger {
  flex-shrink: 0;
  min-width: 76px;
}

.status-line {
  margin-bottom: 10px;
  font-size: 12px;
  color: #909399;
  min-height: 16px;
}

.status-line--ok {
  color: #67c23a;
}

.msg-list {
  height: 380px;
  overflow-y: auto;
  border: 1px solid #ebeef5;
  border-radius: 6px;
  padding: 10px;
  margin-bottom: 12px;
  background: #fafbfc;
}

.msg {
  margin-bottom: 10px;
}

.msg-time {
  display: block;
  font-size: 11px;
  color: #909399;
  margin-bottom: 2px;
}

.msg-body {
  margin: 0;
  padding: 8px 10px;
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 6px;
  font-size: 12px;
  font-family: Consolas, Monaco, monospace;
  white-space: pre-wrap;
  word-break: break-all;
  color: #303133;
}

.msg--system .msg-body {
  background: #f4f4f5;
  color: #909399;
  font-family: inherit;
}

.badge {
  font-size: 12px;
  color: #909399;
}

.badge--open {
  color: #67c23a;
}

.badge--connecting {
  color: #e6a23c;
}
</style>
