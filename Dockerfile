# 第一阶段：编译
FROM golang:1.26.4-alpine AS builder

WORKDIR /app

# 复制依赖文件并下载
COPY go.mod go.sum ./

RUN go env -w GOPROXY=https://goproxy.cn,direct && \
	go env -w GOSUMDB=sum.golang.google.cn

RUN go mod download

# 复制源代码并编译
COPY . ./

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o gateway ./cmd/gateway

# 第二阶段：运行
FROM alpine:latest

# 安装根证书（用于 HTTPS 请求）
RUN apk add --no-cache ca-certificates

WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/gateway ./

# 复制配置文件（使用默认配置）
COPY ./configs/config.toml ./configs/

# 创建日志目录
RUN mkdir -p ./logs

# 暴露端口（业务 + pprof）
EXPOSE 8080 6060

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

CMD ["./gateway"]