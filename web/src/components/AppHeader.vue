<script setup>
import { useRouter } from 'vue-router'

defineProps({
  title: { type: String, required: true },
})

const router = useRouter()

function handleLogout() {
  localStorage.removeItem('token')
  router.push({ name: 'login' })
}
</script>

<template>
  <header class="topbar">
    <span class="title">{{ title }}</span>
    <nav>
      <router-link to="/dashboard">看板</router-link>
      <router-link to="/broadcast">消息推送</router-link>
      <router-link to="/architecture">架构图</router-link>
    </nav>
    <!-- 右侧自定义区域：各页面可插入自己的控件（如看板的刷新按钮） -->
    <slot />
    <button class="logout" @click="handleLogout">退出登录</button>
  </header>
</template>

<style scoped>
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
</style>
