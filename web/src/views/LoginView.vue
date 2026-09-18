<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { login } from '../api/auth'

const router = useRouter()
const username = ref('zhangsan')
const password = ref('123456')
const errorMsg = ref('')
const loading = ref(false)

async function handleLogin() {
  if (!username.value.trim() || !password.value) {
    errorMsg.value = '请输入账号和密码'
    return
  }
  loading.value = true
  errorMsg.value = ''
  try {
    const res = await login(username.value.trim(), password.value)
    localStorage.setItem('token', res.data.token)
    router.push({ name: 'dashboard' })
  } catch (err) {
    errorMsg.value = err.message || '登录失败'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-page">
    <div class="login-card">
      <h1>WebSocket 网关监控看板</h1>
      <p class="subtitle">请使用管理员账号登录</p>
      <form @submit.prevent="handleLogin">
        <div class="field">
          <label for="username">账号</label>
          <input id="username" v-model="username" type="text" placeholder="请输入账号" autocomplete="username" />
        </div>
        <div class="field">
          <label for="password">密码</label>
          <input id="password" v-model="password" type="password" placeholder="请输入密码" autocomplete="current-password" />
        </div>
        <p v-if="errorMsg" class="error">{{ errorMsg }}</p>
        <button type="submit" :disabled="loading">
          {{ loading ? '登录中…' : '登 录' }}
        </button>
      </form>
    </div>
  </div>
</template>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #1f2d3d 0%, #2d3a4b 100%);
}

.login-card {
  width: 360px;
  padding: 40px 36px;
  background: #fff;
  border-radius: 10px;
  box-shadow: 0 12px 32px rgba(0, 0, 0, 0.25);
}

h1 {
  font-size: 20px;
  text-align: center;
  color: #1f2d3d;
}

.subtitle {
  margin: 8px 0 28px;
  text-align: center;
  font-size: 13px;
  color: #909399;
}

.field {
  margin-bottom: 18px;
}

label {
  display: block;
  margin-bottom: 6px;
  font-size: 14px;
  color: #606266;
}

input {
  width: 100%;
  height: 40px;
  padding: 0 12px;
  border: 1px solid #dcdfe6;
  border-radius: 6px;
  font-size: 14px;
  outline: none;
  transition: border-color 0.2s;
}

input:focus {
  border-color: #409eff;
}

.error {
  margin-bottom: 14px;
  font-size: 13px;
  color: #f56c6c;
}

button {
  width: 100%;
  height: 42px;
  border: none;
  border-radius: 6px;
  background: #409eff;
  color: #fff;
  font-size: 15px;
  cursor: pointer;
  transition: background 0.2s;
}

button:hover {
  background: #66b1ff;
}

button:disabled {
  background: #a0cfff;
  cursor: not-allowed;
}
</style>
