import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// 后端已开启 CORS，开发阶段无需配置代理，直接请求 http://localhost:8080
export default defineConfig({
  plugins: [vue()],
  server: {
    port: 5173,
  },
})
