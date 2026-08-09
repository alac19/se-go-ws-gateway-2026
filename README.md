# **Go轻量级WebSocket网关**

> 基于 Go 语言的高并发 WebSocket 网关，支持连接管理、消息路由、心跳保活、优雅启停、监控指标与容器化部署。

---

## 📌 项目简介

本项目是一个从零构建的轻量级 WebSocket 网关，作为业务后端与前端长连接之间的统一流量入口。它沉淀了实时推送场景下的通用能力——连接管理、房间隔离、消息路由、心跳保活、优雅关闭，并提供了完善的监控指标和容器化部署方案。

---

## ✨ 功能特性

| 类别 | 功能 | 说明 |
|------|------|------|
| **WebSocket 接入** | 协议升级与连接管理 | 支持 `clientId` 和 `roomId` 参数，参数校验与重复连接拒绝 |
| **连接管理** | 注册/注销/查询/遍历 | 基于 `sync.Map` 的连接池，并发安全 |
| **房间管理** | 加入/离开/查询成员 | 读写锁保证并发安全，房间无人时自动回收 |
| **消息路由** | 全服广播 / 房间广播 / 单播 | 非阻塞 `select` 发送，慢客户端自动下线 |
| **心跳保活** | Ping/Pong 机制 | 支持配置 Ping 间隔、Pong 超时和 Ping 写入超时 |
| **优雅启停** | 信号捕获 + 宽限期 + 连接清理 | 收到 SIGINT/SIGTERM 后平滑关闭所有连接 |
| **HTTP REST API** | 广播/房间广播/单播/统计 | 标准 JSON 请求与响应 |
| **限流防护** | 基于 IP 的令牌桶限流 | 挂载到 `/api` 路由组，可配置速率与突发容量 |
| **监控指标** | Prometheus 指标暴露 | 连接数、消息收发量、消息发送失败数等 |
| **结构化日志** | `slog` JSON 格式日志 | 支持控制台输出和文件输出，可配置日志级别 |
| **配置管理** | TOML + 环境变量覆盖 | 所有关键参数支持环境变量动态覆盖 |
| **健康检查** | `/health` 端点 | 用于容器编排的健康探针 |
| **性能分析** | pprof 集成 | 支持 CPU 和内存 Profile 采集，便于性能调优 |
| **容器化部署** | Docker + Docker Compose | 多阶段构建镜像仅 34.9MB，支持 Prometheus 监控 |

---

## 📊 性能指标

| 指标 | 实测值 | 说明 |
|------|--------|------|
| 并发连接数 | **5,000** | 连接建立成功率 100% |
| 广播 P99 延迟（2,000 并发） | **39.01 ms** | 满足 < 50ms 目标 |
| 广播 P99 延迟（5,000 并发） | **59.38 ms** | 极限验证，远低于 80ms 标准 |
| HTTP 接口 QPS | **50,260** | 200 并发下吞吐量 |
| 混合负载 P99 延迟 | **31.41 ms** | 上行消息 + 下行广播 |
| 长稳测试（30 分钟） | **无内存泄漏** | 60 次广播全部成功 |
| 镜像大小 | **34.9 MB** | 基于 Alpine 多阶段构建 |

> 性能测试在 Windows 11（8核12线程，16GB DDR5）环境下完成。

---

## 🛠 技术栈

| 组件 | 选型 | 说明 |
|------|------|------|
| 语言 | Go 1.26.4 | 高并发、内存安全 |
| HTTP 框架 | Gin | 轻量级 Web 框架 |
| WebSocket | gorilla/websocket | WebSocket 协议实现 |
| 配置管理 | BurntSushi/toml + 环境变量 | 配置文件与环境变量双支持 |
| 限流 | golang.org/x/time/rate | 令牌桶算法 |
| 监控 | Prometheus | 指标暴露与采集 |
| 日志 | slog（标准库） | JSON 格式结构化日志 |
| 性能分析 | net/http/pprof | CPU 和内存 Profile |
| 容器化 | Docker + Docker Compose | 多阶段构建，镜像 34.9MB |

---

## 🚀 快速启动

### 环境要求

| 组件 | 版本要求 |
|------|----------|
| Go | 1.26.4+ |
| Docker | 24.0+（可选，用于容器化部署） |
| Docker Compose | 2.20+（可选） |

### 本地运行

```bash
# 克隆项目
git clone https://github.com/alac19/se-go-ws-gateway-2026.git
cd se-go-ws-gateway-2026

# 安装依赖
go mod download

# 启动服务
go run cmd/gateway/main.go
```

服务默认监听 `8080` 端口，pprof 调试端口为 `6060`。

### 容器化部署

```bash
# 构建镜像
docker build -t se-go-ws-gateway:v1.2 .

# 启动服务（含 Prometheus 监控）
docker-compose up -d

# 查看状态
docker-compose ps

# 查看日志
docker-compose logs -f gateway
```

### 验证服务

```bash
# 健康检查
curl http://localhost:8080/health
# 输出：{"status":"ok"}

# 建立 WebSocket 连接（Linux）
wscat -c "ws://localhost:8080/ws?clientId=test&roomId=room1"
# 输出：Connected (press CTRL+C to quit)

# 触发全服广播
curl -X POST http://localhost:8080/api/broadcast \
  -H "Content-Type: application/json" \
  -d '{"type":"text","data":"hello all"}'
```

---

## ⚙️ 配置说明

### 配置文件（configs/config.toml）

```toml
[server]
port = 8080

[websocket]
read_buffer_size = 1024
write_buffer_size = 1024
read_deadline_seconds = 60
write_deadline_seconds = 10
control_write_timeout_seconds = 1

[heartbeat]
ping_interval_seconds = 30
pong_wait_seconds = 60
ping_write_timeout_seconds = 10

[channel]
send_buffer_size = 256
register_buffer_size = 256
unregister_buffer_size = 256

[ratelimit]
every_seconds = 12
burst = 5

[graceful_shutdown]
timeout_seconds = 5

[log]
level = "error"
file_path = "logs/gateway.log"
```

### 环境变量覆盖

以下环境变量可覆盖配置文件中的对应值（**优先级更高**）：

| 环境变量 | 说明 | 默认值 |
|----------|------|--------|
| `WS_PORT` | 服务监听端口 | 8080 |
| `WS_PING_INTERVAL` | Ping 帧发送间隔（秒） | 30 |
| `WS_PONG_WAIT` | Pong 响应超时（秒） | 60 |
| `WS_RATELIMIT_INTERVAL` | 限流令牌生成间隔（秒） | 12 |
| `WS_BURST` | 限流桶容量 | 5 |
| `WS_SHUTDOWN_TIMEOUT` | 优雅退出宽限期（秒） | 5 |

```bash
# 示例：通过环境变量覆盖端口和日志级别
export WS_PORT=8080
go run cmd/gateway/main.go
```

---

## 📮 API 接口文档

### WebSocket 接入

**端点**：`ws://localhost:8080/ws?clientId={clientId}&roomId={roomId}`

**参数说明**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `clientId` | string | 是 | 客户端唯一标识，正则 `^[a-zA-Z0-9_-]+$` |
| `roomId` | string | 是 | 房间 ID，正则 `^[a-zA-Z0-9_-]+$` |

**WebSocket 关闭状态码**：

| 状态码 | 含义 |
|--------|------|
| `4000` | `clientId` 或 `roomId` 缺失 |
| `4001` | 参数格式无效（含特殊字符） |
| `4002` | `clientId` 已存在（重复连接拒绝） |
| `1000` | 正常关闭 |
| `1001` | 服务正在关闭（优雅退出时） |

---

### HTTP REST API

| 方法 | 路径 | 功能 | 请求体 |
|------|------|------|--------|
| POST | `/api/broadcast` | 全服广播 | JSON 对象或数组 |
| POST | `/api/room/:roomId/broadcast` | 房间广播 | JSON 对象或数组 |
| POST | `/api/client/:clientId/send` | 单播 | JSON 对象或数组 |
| GET | `/api/stats` | 连接统计 | 无 |
| GET | `/health` | 健康检查 | 无 |
| GET | `/metrics` | Prometheus 指标 | 无 |

#### 全服广播

```bash
curl -X POST http://localhost:8080/api/broadcast \
  -H "Content-Type: application/json" \
  -d '{"type":"text","data":"hello all"}'
```

**响应**：
```json
{"code":0,"status":"success","data":null}
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
  -H "Content-Type": "application/json" \
  -d '{"type":"text","data":"private msg"}'
```

#### 连接统计

```bash
curl http://localhost:8080/api/stats
```

**响应示例**：
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

**响应**：
```json
{"status":"ok"}
```

#### Prometheus 指标

访问 `http://localhost:8080/metrics` 可获取以下关键指标：

| 指标名称 | 类型 | 说明 |
|----------|------|------|
| `ws_active_connections` | Gauge | 当前在线连接数 |
| `ws_gateway_msg_sent_total` | Counter | 消息发送总量（按类型：single/room/broadcast） |
| `ws_gateway_msg_send_fail_total` | Counter | 消息发送失败数（按原因：offline/block） |
| `ws_connection_events_total` | Counter | 连接建立/关闭事件总数 |
| `ws_messages_received_total` | Counter | 消息接收总数 |

---

## 📁 项目结构

```
.
├── cmd/
│   ├── gateway/
│   │   └── main.go              # 程序入口，依赖组装
│   └── loadtest/
│       └── main.go              # 压测工具（自定义 WebSocket 负载测试）
├── internal/
│   ├── config/
│   │   └── config.go            # 配置加载与校验（TOML + 环境变量）
│   ├── handler/
│   │   ├── api_handler.go       # HTTP REST API 处理
│   │   └── ws_handler.go        # WebSocket 接入与心跳保活
│   ├── service/
│   │   ├── client_manager.go    # 连接管理器（CSP 模型）
│   │   ├── room_manager.go      # 房间管理器（读写锁）
│   │   └── message_router.go    # 消息路由（广播/单播）
│   ├── model/
│   │   ├── client.go            # Client 结构体
│   │   ├── constants.go         # WebSocket 关闭码与业务响应码常量
│   │   └── message.go           # Message 结构体
│   └── middleware/
│       └── ratelimit.go         # 限流中间件（令牌桶）
├── pkg/
│   ├── limiter/                 # 令牌桶限流器（IP 隔离）
│   ├── logger/                  # slog 日志初始化
│   └── metrics/                 # Prometheus 指标注册
├── configs/
│   └── config.toml              # 配置文件
├── docs/                        # 项目文档（需求、设计、测试、部署）
├── logs/                        # 日志文件目录
├── test_results/                # 压测结果输出
├── Dockerfile                   # 多阶段构建镜像
├── docker-compose.yml           # 容器编排（网关 + Prometheus）
├── prometheus.yml               # Prometheus 采集配置
├── go.mod
├── go.sum
├── README.md
└── LICENSE
```

---

## 🧪 测试

### 单元测试

```bash
# 运行所有单元测试
go test -v -cover ./...

# 生成覆盖率报告
go test -cover -coverprofile=c.out ./...
go tool cover -html=c.out
```

**测试结果**：核心模块覆盖率 82.9%，所有用例通过。

### 性能测试

项目提供了自定义压测工具 `cmd/loadtest/main.go`：

```bash
# 连接建立性能（1000 并发）
go run cmd/loadtest/main.go -num 1000

# 广播延迟（2000 并发）
go run cmd/loadtest/main.go -num 2000

# 极限验证（5000 并发）
go run cmd/loadtest/main.go -num 5000

# HTTP 接口吞吐量（需关闭限流）
hey -n 100000 -c 200 -m POST \
  -H "Content-Type: application/json" \
  -d '{"type":"text","data":"load"}' \
  http://localhost:8080/api/broadcast
```

> 详细性能测试数据请参阅 [测试报告](docs/8-测试报告.md)。

---

## 🐳 部署

### Docker 镜像构建

```bash
docker build -t se-go-ws-gateway:v1.2 .
```

### Docker Compose 启动

```bash
docker-compose up -d
```

### 服务验证

```bash
# 健康检查
curl http://localhost:8080/health

# Prometheus 监控
# 浏览器访问 http://localhost:9090

# pprof 性能分析
go tool pprof -http=:8081 http://localhost:6060/debug/pprof/profile?seconds=30
```

> 详细部署步骤请参阅 [部署计划](docs/13-部署计划.md) 和 [部署报告](docs/14-部署报告.md)。

---

## 📚 文档

| 文档 | 说明 |
|------|------|
| [软件需求规格说明书](docs/2-需求规格说明书/2-1软件需求规格说明书.md) | 功能需求与非功能需求定义 |
| [接口设计说明书](docs/2-需求规格说明书/2-2接口设计说明书.md) | HTTP API 与 WebSocket 接口定义 |
| [系统设计文档](docs/4-系统设计文档.md) | 架构设计、模块划分、数据流 |
| [测试计划](docs/7-测试计划.md) | 单元测试、集成测试、性能测试计划 |
| [测试报告](docs/8-测试报告.md) | 测试执行结果与覆盖率报告 |
| [用户验收测试计划](docs/11-用户验收测试计划.md) | UAT 测试用例与通过标准 |
| [用户验收测试报告](docs/12-用户验收测试报告.md) | UAT 执行结果与签署确认 |
| [部署计划](docs/13-部署计划.md) | 部署架构、环境准备、容器化部署流程 |
| [部署报告](docs/14-部署报告.md) | 部署执行过程、验证结果、回滚测试 |

---

## 📄 许可证

本项目采用 MIT 许可证。

---

**版本记录**

| 版本 | 日期 | 说明 |
|------|------|------|
| v1.0 | 2026-07-06 | MVP 框架|
| v1.1 | 2026-07-28 | 完整功能实现 |
| v1.2 | 2026-08-09 | 测试调优、容器化部署、监控集成、文档完善 |