// Package config provides configuration management for the gateway service.
// It supports loading from TOML files and overriding via environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

// Config 总配置结构, 包含所有子配置项。
type Config struct {
	Server           Server           `toml:"server"`
	Websocket        Websocket        `toml:"websocket"`
	Heartbeat        Heartbeat        `toml:"heartbeat"`
	Channel          Channel          `toml:"channel"`
	Ratelimit        Ratelimit        `toml:"ratelimit"`
	GracefulShutdown GracefulShutdown `toml:"graceful_shutdown"`
	Log              Log              `toml:"log"`
	Jwt              JWT              `toml:"jwt"`
	Auth             Auth             `toml:"auth"`
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

// Log 定义日志配置。
type Log struct {
	Level    string `toml:"level"`     // 日志级别（debug, info, warn, error）
	FilePath string `toml:"file_path"` // 日志文件路径，为空则只输出到控制台
}

// JWT 定义鉴权配置。
type JWT struct {
	Secret   string `toml:"secret"`    // 密钥
	TTLHours int    `toml:"ttl_hours"` // 过期时间
}

// Auth 定义身份凭证配置。
type Auth struct {
	UserName     string `toml:"username"`      // 账号名
	PasswordHash string `toml:"password_hash"` // 哈希码
}

// LoadConfig 从指定路径加载 TOML 配置文件, 并校验配置合法性。
func LoadConfig(path string) (*Config, error) {
	var config Config

	if _, err := toml.DecodeFile(path, &config); err != nil {
		return nil, fmt.Errorf("decode config failed: %w", err)
	}

	config.ApplyEnvOverrides()

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}

// ApplyEnvOverrides 使用环境变量覆盖配置中的对应字段。
// 支持的环境变量：WS_PORT, WS_PING_INTERVAL, WS_PONG_WAIT,
// WS_RATELIMIT_INTERVAL, WS_BURST, WS_SHUTDOWN_TIMEOUT,
// WS_LOG_LEVEL, WS_LOG_FILE, WS_JWT_SECRET,
// WS_JWT_TTL_HOURS, WS_AUTH_USERNAME, WS_AUTH_PASSWORD_HASH。
// 仅当环境变量非空时才会覆盖, 数值型还需能正确解析。
func (c *Config) ApplyEnvOverrides() {
	if v := os.Getenv("WS_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.Server.Port = port
		}
	}
	if v := os.Getenv("WS_PING_INTERVAL"); v != "" {
		if pingInternal, err := strconv.Atoi(v); err == nil {
			c.Heartbeat.PingIntervalSeconds = pingInternal
		}
	}
	if v := os.Getenv("WS_PONG_WAIT"); v != "" {
		if pongWait, err := strconv.Atoi(v); err == nil {
			c.Heartbeat.PongWaitSeconds = pongWait
		}
	}
	if v := os.Getenv("WS_RATELIMIT_INTERVAL"); v != "" {
		if ratelimitInternal, err := strconv.Atoi(v); err == nil {
			c.Ratelimit.EverySeconds = ratelimitInternal
		}
	}
	if v := os.Getenv("WS_BURST"); v != "" {
		if burst, err := strconv.Atoi(v); err == nil {
			c.Ratelimit.Burst = burst
		}
	}
	if v := os.Getenv("WS_SHUTDOWN_TIMEOUT"); v != "" {
		if shutdownTimeout, err := strconv.Atoi(v); err == nil {
			c.GracefulShutdown.TimeoutSeconds = shutdownTimeout
		}
	}
	if v := os.Getenv("WS_LOG_LEVEL"); v != "" {
		c.Log.Level = v
	}
	if v := os.Getenv("WS_LOG_FILE"); v != "" {
		c.Log.FilePath = v
	}
	if v := os.Getenv("WS_JWT_SECRET"); v != "" {
		c.Jwt.Secret = v
	}
	if v := os.Getenv("WS_JWT_TTL_HOURS"); v != "" {
		if ttlHours, err := strconv.Atoi(v); err == nil {
			c.Jwt.TTLHours = ttlHours
		}
	}
	if v := os.Getenv("WS_AUTH_USERNAME"); v != "" {
		c.Auth.UserName = v
	}
	if v := os.Getenv("WS_AUTH_PASSWORD_HASH"); v != "" {
		c.Auth.PasswordHash = v
	}
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

	// Log
	if strings.ToLower(c.Log.Level) != "debug" && strings.ToLower(c.Log.Level) != "info" && strings.ToLower(c.Log.Level) != "warn" && strings.ToLower(c.Log.Level) != "error" {
		return fmt.Errorf("log.level 必须为 debug/info/warn/error 其中一个, 当前值: %v", c.Log.Level)
	}

	// JWT
	if c.Jwt.Secret == "" {
		return fmt.Errorf("jwt.secret 不能为空, 当前值: %v", c.Jwt.Secret)
	}
	if len(c.Jwt.Secret) < 16 {
		return fmt.Errorf("jwt.secret 长度不能少于 16 个字符(建议 32 以上), 当前长度: %d", len(c.Jwt.Secret))
	}
	if c.Jwt.TTLHours <= 0 {
		return fmt.Errorf("jwt.ttl_hours 必须 > 0, 当前值: %d", c.Jwt.TTLHours)
	}

	// Auth
	if c.Auth.UserName == "" {
		return fmt.Errorf("auth.username 不能为空, 当前值: %v", c.Auth.UserName)
	}
	if c.Auth.PasswordHash == "" {
		return fmt.Errorf("auth.password_hash 不能为空, 当前值: %v", c.Auth.PasswordHash)
	}
	if !isBcryptHash(c.Auth.PasswordHash) {
		return fmt.Errorf("auth.password_hash 必须为合法的 bcrypt 哈希(以 $2a$/$2b$/$2y$ 开头, 长度 60), 当前值: %s", c.Auth.PasswordHash)
	}

	return nil
}

// PingInterval 返回 Ping 帧发送间隔的 time.Duration。
func (c *Config) PingInterval() time.Duration {
	return time.Duration(c.Heartbeat.PingIntervalSeconds) * time.Second
}

// PongWait 返回等待 Pong 响应的超时 time.Duration。
func (c *Config) PongWait() time.Duration {
	return time.Duration(c.Heartbeat.PongWaitSeconds) * time.Second
}

// PingWriteTimeout 返回发送 Ping 帧的写入超时 time.Duration。
func (c *Config) PingWriteTimeout() time.Duration {
	return time.Duration(c.Heartbeat.PingWriteTimeoutSeconds) * time.Second
}

// ReadDeadline 返回读超时的 time.Duration。
func (c *Config) ReadDeadline() time.Duration {
	return time.Duration(c.Websocket.ReadDeadlineSeconds) * time.Second
}

// WriteDeadline 返回写超时的 time.Duration。
func (c *Config) WriteDeadline() time.Duration {
	return time.Duration(c.Websocket.WriteDeadlineSeconds) * time.Second
}

// ControlWriteTimeout 返回控制帧（关闭/Ping）写入超时的 time.Duration。
func (c *Config) ControlWriteTimeout() time.Duration {
	return time.Duration(c.Websocket.ControlWriteTimeoutSeconds) * time.Second
}

// ShutdownTimeout 返回优雅退出宽限期的 time.Duration。
func (c *Config) ShutdownTimeout() time.Duration {
	return time.Duration(c.GracefulShutdown.TimeoutSeconds) * time.Second
}

// RateLimitInterval 返回限流器令牌生成间隔的 time.Duration。
func (c *Config) RateLimitInterval() time.Duration {
	return time.Duration(c.Ratelimit.EverySeconds) * time.Second
}

// TokenTTL 返回 token 验证器过期时间的 time.Duration。
func (c *Config) TokenTTL() time.Duration {
	return time.Duration(c.Jwt.TTLHours) * time.Hour
}

// isBcryptHash 校验字符串是否为合法的 bcrypt 哈希格式:
// "$2a$" 或 "$2b$" 或 "$2y$" + 两位成本因子 + 22 位盐 + 31 位摘要, 共 60 个字符。
func isBcryptHash(hash string) bool {
	if len(hash) != 60 {
		return false
	}

	if !strings.HasPrefix(hash, "$2a$") && !strings.HasPrefix(hash, "$2b$") && !strings.HasPrefix(hash, "$2y$") {
		return false
	}

	// 第 5、6 位是成本因子, 必须是两位数字
	if _, err := strconv.Atoi(hash[4:6]); err != nil {
		return false
	}

	// 第 8 位起是盐和摘要, 只允许 bcrypt 的 base64 字母表
	for i := 7; i < len(hash); i++ {
		ch := hash[i]
		isAlnum := (ch >= '0' && ch <= '9') || (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z')

		if !isAlnum && ch != '.' && ch != '/' {
			return false
		}
	}

	return true
}
