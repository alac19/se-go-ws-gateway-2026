// Package config 提供网关服务的配置管理。
package config

import (
	"fmt"
	"time"

	"github.com/BurntSushi/toml"
)

// Config 总配置结构。
type Config struct {
	Server           Server           `toml:"server"`
	Websocket        Websocket        `toml:"websocket"`
	Heartbeat        Heartbeat        `toml:"heartbeat"`
	Channel          Channel          `toml:"channel"`
	Ratelimit        Ratelimit        `toml:"ratelimit"`
	GracefulShutdown GracefulShutdown `toml:"graceful_shutdown"`
}

// Server 定义 HTTP 服务器配置。
type Server struct {
	Port int `toml:"port"` // 监听端口
}

// Websocket 定义 WebSocket 连接相关配置。
type Websocket struct {
	ReadBufferSize             int `toml:"read_buffer_size"`              // 读缓冲区大小（字节）
	WriteBufferSize            int `toml:"write_buffer_size"`             // 写缓冲区大小（字节）
	ReadDeadlineSeconds        int `toml:"read_deadline_seconds"`         // 读超时（秒）
	WriteDeadlineSeconds       int `toml:"write_deadline_seconds"`        // 写超时（秒）
	ControlWriteTimeoutSeconds int `toml:"control_write_timeout_seconds"` // 控制帧（关闭/Ping）写入超时（秒）
}

// Heartbeat 定义心跳保活配置。
type Heartbeat struct {
	PingIntervalSeconds     int `toml:"ping_interval_seconds"`      // Ping 帧发送间隔（秒）
	PongWaitSeconds         int `toml:"pong_wait_seconds"`          // 等待 Pong 响应的超时（秒）
	PingWriteTimeoutSeconds int `toml:"ping_write_timeout_seconds"` // 发送 Ping 帧的写入超时（秒）
}

// Channel 定义内部 channel 缓冲区大小配置。
type Channel struct {
	SendBufferSize       int `toml:"send_buffer_size"`       // 每个 Client 的 SendChan 缓冲区大小
	RegisterBufferSize   int `toml:"register_buffer_size"`   // ClientManager.register 通道缓冲区大小
	UnregisterBufferSize int `toml:"unregister_buffer_size"` // ClientManager.unregister 通道缓冲区大小
}

// Ratelimit 定义限流器配置。
type Ratelimit struct {
	EverySeconds int `toml:"every_seconds"` // 生成一个令牌的时间间隔（秒）
	Burst        int `toml:"burst"`         // 令牌桶容量
}

// GracefulShutdown 定义优雅退出配置。
type GracefulShutdown struct {
	TimeoutSeconds int `toml:"timeout_seconds"` // 关闭等待宽限期（秒）
}

// LoadConfig 从指定路径加载 TOML 配置文件，并校验配置合法性。
func LoadConfig(path string) (*Config, error) {
	var config Config

	if _, err := toml.DecodeFile(path, &config); err != nil {
		return nil, fmt.Errorf("decode config failed: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}

// Validate 校验配置项的合法性。
func (c *Config) Validate() error {
	// Server
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port 必须在 1-65535 之间, 当前值: %d", c.Server.Port)
	}

	// Websocket
	if c.Websocket.ReadBufferSize <= 0 {
		return fmt.Errorf("websocket.read_buffer_size 必须 > 0, 当前值: %d", c.Websocket.ReadBufferSize)
	}
	if c.Websocket.WriteBufferSize <= 0 {
		return fmt.Errorf("websocket.write_buffer_size 必须 > 0, 当前值: %d", c.Websocket.WriteBufferSize)
	}
	if c.Websocket.ReadDeadlineSeconds <= 0 {
		return fmt.Errorf("websocket.read_deadline_seconds 必须 > 0, 当前值: %d", c.Websocket.ReadDeadlineSeconds)
	}
	if c.Websocket.WriteDeadlineSeconds <= 0 {
		return fmt.Errorf("websocket.write_deadline_seconds 必须 > 0, 当前值: %d", c.Websocket.WriteDeadlineSeconds)
	}
	if c.Websocket.ControlWriteTimeoutSeconds <= 0 {
		return fmt.Errorf("websocket.control_write_timeout_seconds 必须 > 0, 当前值: %d", c.Websocket.ControlWriteTimeoutSeconds)
	}

	// Heartbeat
	if c.Heartbeat.PingIntervalSeconds <= 0 {
		return fmt.Errorf("heartbeat.ping_interval_seconds 必须 > 0, 当前值: %d", c.Heartbeat.PingIntervalSeconds)
	}
	if c.Heartbeat.PongWaitSeconds <= 0 {
		return fmt.Errorf("heartbeat.pong_wait_seconds 必须 > 0, 当前值: %d", c.Heartbeat.PongWaitSeconds)
	}
	if c.Heartbeat.PongWaitSeconds <= c.Heartbeat.PingIntervalSeconds {
		return fmt.Errorf("heartbeat.pong_wait_seconds 必须 > %d, 当前值: %d", c.Heartbeat.PingIntervalSeconds, c.Heartbeat.PongWaitSeconds)
	}
	if c.Heartbeat.PingWriteTimeoutSeconds <= 0 {
		return fmt.Errorf("heartbeat.ping_write_timeout_seconds 必须 > 0, 当前值: %d", c.Heartbeat.PingWriteTimeoutSeconds)
	}

	// Channel
	if c.Channel.SendBufferSize <= 0 {
		return fmt.Errorf("channel.send_buffer_size 必须 > 0, 当前值: %d", c.Channel.SendBufferSize)
	}
	if c.Channel.RegisterBufferSize <= 0 {
		return fmt.Errorf("channel.register_buffer_size 必须 > 0, 当前值: %d", c.Channel.RegisterBufferSize)
	}
	if c.Channel.UnregisterBufferSize <= 0 {
		return fmt.Errorf("channel.unregister_buffer_size 必须 > 0, 当前值: %d", c.Channel.UnregisterBufferSize)
	}

	// Ratelimit
	if c.Ratelimit.EverySeconds <= 0 {
		return fmt.Errorf("ratelimit.every_seconds 必须 > 0, 当前值: %d", c.Ratelimit.EverySeconds)
	}
	if c.Ratelimit.Burst <= 0 {
		return fmt.Errorf("ratelimit.burst 必须 > 0, 当前值: %d", c.Ratelimit.Burst)
	}

	// GracefulShutdown
	if c.GracefulShutdown.TimeoutSeconds <= 0 {
		return fmt.Errorf("graceful_shutdown.timeout_seconds 必须 > 0, 当前值: %d", c.GracefulShutdown.TimeoutSeconds)
	}

	return nil
}

// 辅助方法：将配置转换为 time.Duration（方便调用方使用）
func (c *Config) PingInterval() time.Duration {
	return time.Duration(c.Heartbeat.PingIntervalSeconds) * time.Second
}

func (c *Config) PongWait() time.Duration {
	return time.Duration(c.Heartbeat.PongWaitSeconds) * time.Second
}

func (c *Config) PingWriteTimeout() time.Duration {
	return time.Duration(c.Heartbeat.PingWriteTimeoutSeconds) * time.Second
}

func (c *Config) ReadDeadline() time.Duration {
	return time.Duration(c.Websocket.ReadDeadlineSeconds) * time.Second
}

func (c *Config) WriteDeadline() time.Duration {
	return time.Duration(c.Websocket.WriteDeadlineSeconds) * time.Second
}

func (c *Config) ControlWriteTimeout() time.Duration {
	return time.Duration(c.Websocket.ControlWriteTimeoutSeconds) * time.Second
}

func (c *Config) ShutdownTimeout() time.Duration {
	return time.Duration(c.GracefulShutdown.TimeoutSeconds) * time.Second
}

func (c *Config) RateLimitInterval() time.Duration {
	return time.Duration(c.Ratelimit.EverySeconds) * time.Second
}
