# se-go-ws-gateway-web

WebSocket 网关监控看板前端。

## 技术栈

- Vue 3 + Vite
- vue-router（登录守卫）
- axios（统一请求封装：自动携带 token、401 跳登录、429 限流提示）
- ECharts（房间分布图表，后续分支使用）

## 本地运行

```bash
cd web
npm install        # 首次运行，已配置 npmmirror 镜像源
npm run dev        # 开发服务器 http://localhost:5173
```

后端网关需在本机 8080 端口运行（仓库根目录执行 `go run cmd/gateway/main.go`）。后端已开启 CORS，前端无需配置代理。

演示账号：`zhangsan` / `123456`

## 页面

| 路由 | 说明 |
|------|------|
| `/login` | 登录页（`POST /api/auth/login`，token 存 localStorage） |
| `/dashboard` | 监控看板（占位，`GET /api/stats`） |
| `/broadcast` | 广播消息操作区（占位，三种推送接口） |
