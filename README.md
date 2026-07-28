# **Go轻量级WebSocket网关**

> 基于 Go 语言的高并发 WebSocket 网关，支持连接管理、消息广播与单播、心跳保活、优雅启停、配置管理与结构化日志。

## 功能特性

- ✅ **WebSocket 接入**：支持 `clientId` 和 `roomId` 参数，协议升级与连接管理
- ✅ **连接管理器**：基于 `sync.Map` 的连接池，支持注册/注销/查询/遍历
- ✅ **房间管理器**：支持客户端加入/离开房间，房间广播隔离
- ✅ **消息路由**：全服广播、房间广播、单播（非阻塞 `select` 发送）
- ✅ **心跳保活**：Ping/Pong 机制，支持配置间隔和超时
- ✅ **优雅启停**：信号捕获 + 宽限期 + 连接清理
- ✅ **HTTP REST API**：广播/房间广播/单播/统计接口
- ✅ **限流防护**：基于 IP 的令牌桶限流，挂载到 `/api` 路由组
- ✅ **监控指标**：Prometheus 指标暴露（连接数、消息数等）
- ✅ **结构化日志**：`slog` JSON 格式日志，支持文件输出
- ✅ **配置管理**：`config.toml` + 环境变量覆盖
- ✅ **健康检查**：`/health` 端点
- ✅ **panic 恢复**：单个连接异常不影响整体服务

## 技术栈

| 组件 | 选型 |
|------|------|
| 语言 | Go 1.26+ |
| HTTP 框架 | Gin |
| WebSocket | gorilla/websocket |
| 配置管理 | TOML + 环境变量覆盖 |
| 限流 | golang.org/x/time/rate（令牌桶） |
| 监控 | Prometheus |
| 日志 | slog（标准库，JSON 格式） |
| 依赖注入 | 手动注入，分层清晰 |

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

### 配置方式

服务支持两种配置方式，**环境变量优先级高于配置文件**：

1. **配置文件**：`configs/config.toml`
2. **环境变量**（可选覆盖）：

| 环境变量 | 说明 | 默认值 |
|----------|------|--------|
| `WS_PORT` | 服务监听端口 | 8080 |
| `WS_PING_INTERVAL` | Ping 帧发送间隔（秒） | 30 |
| `WS_PONG_WAIT` | Pong 响应超时（秒） | 60 |
| `WS_RATELIMIT_INTERVAL` | 限流令牌生成间隔（秒） | 12 |
| `WS_BURST` | 限流桶容量 | 5 |
| `WS_SHUTDOWN_TIMEOUT` | 优雅退出宽限期（秒） | 5 |

```bash
# 示例：通过环境变量覆盖端口
export WS_PORT=9090
go run cmd/gateway/main.go
```

### 日志输出

- 控制台实时输出
- 日志文件：`logs/gateway.log`（JSON 格式）

## API 接口

### WebSocket 接入

**连接端点**：`ws://localhost:8080/ws?clientId={clientId}&roomId={roomId}`

**参数说明**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `clientId` | string | 是 | 客户端唯一标识，仅允许字母、数字、下划线、中划线 |
| `roomId` | string | 是 | 房间 ID，仅允许字母、数字、下划线、中划线 |

**关闭状态码**：

| 状态码 | 含义 |
|--------|------|
| `4000` | `clientId` 或 `roomId` 缺失 |
| `4001` | `clientId` 或 `roomId` 格式无效（含特殊字符） |
| `4002` | `clientId` 已存在 |

**心跳保活**：
- 服务端每 30 秒（可配置）发送 Ping 帧
- 客户端须在 60 秒（可配置）内回复 Pong 帧
- 超时未响应则服务端主动关闭连接

### REST API

| 方法 | 路径 | 功能 | 请求体 |
|------|------|------|--------|
| POST | `/api/broadcast` | 全服广播 | JSON 对象 |
| POST | `/api/room/:roomId/broadcast` | 房间广播 | JSON 对象 |
| POST | `/api/client/:clientId/send` | 单播 | JSON 对象 |
| GET | `/api/stats` | 连接统计 | 无 |
| GET | `/health` | 健康检查 | 无 |
| GET | `/metrics` | Prometheus 指标 | 无 |

#### 全服广播

```bash
curl -X POST http://localhost:8080/api/broadcast \
  -H "Content-Type: application/json" \
  -d '{"type":"text","data":"hello all"}'
```

#### 房间广播

```bash
curl -X POST "http://localhost:8080/api/room/room1/broadcast" \
  -H "Content-Type: application/json" \
  -d '{"type":"text","data":"Hello room1!"}'
```

#### 单播

```bash
curl -X POST http://localhost:8080/api/client/test1/send \
  -H "Content-Type: application/json" \
  -d '{"type":"text","data":"private msg"}'
```

#### 连接统计

```bash
curl http://localhost:8080/api/stats
```

响应示例：
```json
{
  "code": 0,
  "status": "success",
  "data": {
    "online_connections": 3,
    "all_rooms_connections_stats": {"room1": 2, "room2": 1},
    "gateway_server_initial_time": 3600
  }
}
```

#### 健康检查

```bash
curl http://localhost:8080/health
```

响应示例：
```json
{"status":"ok"}
```

## 项目结构

```
.
├── cmd/gateway/
│   └── main.go                 # 程序入口，依赖组装
├── internal/
│   ├── config/
│   │   └── config.go           # 配置加载与校验
│   ├── handler/
│   │   ├── api_handler.go      # REST API 处理
│   │   └── ws_handler.go       # WebSocket 接入与心跳
│   ├── service/
│   │   ├── client_manager.go   # 连接管理器
│   │   ├── room_manager.go     # 房间管理器
│   │   └── message_router.go   # 消息路由
│   ├── model/
│   │   ├── client.go           # Client 结构体
│   │   └── message.go          # Message 结构体
│   └── middleware/
│       └── ratelimit.go        # 限流中间件
├── pkg/
│   ├── limiter/                # 令牌桶限流器
│   ├── logger/                 # 日志初始化
│   └── metrics/                # Prometheus 指标
├── configs/
│   └── config.toml             # 配置文件
├── docs/                       # 项目文档
├── logs/                       # 日志文件目录
├── go.mod
├── go.sum
├── README.md
└── LICENSE
```

## 版本记录

| 版本 | 日期 | 说明 |
|------|------|------|
| v1.0 | 2026-07-28 | 完整实现：连接管理、消息路由、心跳保活、优雅启停、配置管理、限流、监控、日志 |

---