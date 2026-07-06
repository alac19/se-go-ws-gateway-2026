# **短链接服务**

> 基于 Go 语言的高并发 WebSocket 网关，支持连接管理、消息广播与单播、心跳保活、优雅启停。

## 项目状态

当前为 **MVP 阶段**，已完成：
- [x] Gin 基础路由框架
- [x] REST API 骨架（广播/房间广播/单播/统计）
- [x] WebSocket 接入与 Echo 回显验证
- [ ] 连接管理器（ClientManager）
- [ ] 消息路由（广播/房间广播/单播）
- [ ] 心跳保活（Ping/Pong）
- [ ] 优雅退出
- [ ] Prometheus 监控指标

## 技术栈

| 组件 | 选型 |
|------|------|
| 语言 | Go 1.26+ |
| HTTP 框架 | Gin |
| WebSocket | gorilla/websocket |
| 配置管理 | TOML |
| 日志 | slog（标准库） |
| 监控 | Prometheus（规划中） |

## 快速启动

### 环境要求

- Go 1.26+（推荐 64 位版本）

### 克隆与运行

```bash
# 克隆项目
git clone https://github.com/alac19/se-go-ws-gateway-2026.git
cd se-go-ws-gateway-2026

# 安装依赖
go mod download

# 启动服务
go run cmd/gateway/main.go
```

服务默认监听 `:8080`。

## API 预览

### REST API（业务后端调用）

| 方法 | 路径 | 功能 |
|------|------|------|
| POST | `/api/broadcast` | 全服广播 |
| POST | `/api/room/:roomId/broadcast` | 房间广播 |
| POST | `/api/client/:clientId/send` | 单播 |
| GET | `/api/stats` | 连接统计 |

### WebSocket 接入

**连接端点**：`ws://localhost:8080/ws?clientId={clientId}&roomId={roomId}`

**Echo 测试（当前阶段）**：连接建立后，发送任意消息，服务端原样返回。

```javascript
// 浏览器控制台测试
const ws = new WebSocket("ws://localhost:8080/ws?clientId=test&roomId=room1");
ws.onmessage = (e) => console.log("收到:", e.data);
ws.send("Hello, WebSocket!");
```

## 项目结构

```
.
├── cmd/gateway/          # 程序入口
├── internal/
│   ├── handler/          # HTTP 处理层（API + WebSocket）
│   ├── service/          # 业务逻辑层
│   ├── model/            # 数据结构
│   └── config/           # 配置管理
├── pkg/                  # 可复用公共库
├── configs/              # 配置文件
└── docs/                 # 项目文档
```

## 开发计划

1. ~~Gin 路由框架与 Echo 服务~~ ✅
2. 连接管理器（ClientManager + RoomManager）
3. 消息路由（广播/房间广播/单播）
4. 心跳保活（Ping/Pong）
5. 优雅退出（signal + WaitGroup）
6. Prometheus 指标集成
7. 性能压测与调优

## 版本记录

| 版本 | 日期 | 说明 |
|------|------|------|
| v0.1.0 | 2026-07-06 | MVP：Gin 框架 + REST 骨架 + WebSocket Echo |

---

### 几点说明

1. **当前状态**：README 第一行可以用“⚠️ 项目处于开发中”这类标识，让访问者了解这是开发中的项目。
2. **API 预览**：目前这些接口只返回空响应或模拟数据，所以建议在 WebSocket 部分标明“当前支持 Echo 回显”。
3. **项目结构**：目前还没有 `config/` 和 `pkg/` 的实际代码，可以先保留目录结构示意，为后续扩展预留位置。
4. **版本号**：`v0.1.0` 可以体现这是 MVP 阶段，后续开发完成后再提升版本号。