<script setup>
import { useRouter } from 'vue-router'

const router = useRouter()

function handleLogout() {
  localStorage.removeItem('token')
  router.push({ name: 'login' })
}
</script>

<template>
  <div class="page">
    <header class="topbar">
      <span class="title">监控看板</span>
      <nav>
        <router-link to="/dashboard">看板</router-link>
        <router-link to="/broadcast">消息推送</router-link>
      </nav>
      <button class="logout" @click="handleLogout">退出登录</button>
    </header>
    <main class="content">
      <div class="placeholder">
        <h2>性能数据看板（占位页）</h2>
        <p>在线连接数、房间分布、服务运行时长将在分支 <code>dashboard</code> 中实现。</p>
        <p>数据来源：<code>GET /api/stats</code>（已可调用，注意网关限流：12 秒约 1 次）。</p>
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

.logout {
  height: 32px;
  padding: 0 14px;
  border: 1px solid #dcdfe6;
  border-radius: 6px;
  background: #fff;
  font-size: 13px;
  color: #606266;
  cursor: pointer;
}

.logout:hover {
  color: #409eff;
  border-color: #409eff;
}

.content {
  padding: 24px;
}

.placeholder {
  padding: 60px;
  background: #fff;
  border-radius: 10px;
  text-align: center;
  color: #909399;
}

.placeholder h2 {
  margin-bottom: 16px;
  color: #303133;
}

.placeholder p {
  margin: 8px 0;
  font-size: 14px;
}

code {
  padding: 2px 6px;
  background: #f0f2f5;
  border-radius: 4px;
  font-size: 13px;
  color: #476582;
}
</style>
