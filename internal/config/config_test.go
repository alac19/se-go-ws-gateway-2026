package config

import (
	"errors"
	"os"
	"testing"
	"time"
)

func defaultConfig() Config {
	return Config{
		Server:           Server{Port: 8080},
		Websocket:        Websocket{WriteBufferSize: 1024, ReadBufferSize: 1024, ReadDeadlineSeconds: 60, WriteDeadlineSeconds: 10, ControlWriteTimeoutSeconds: 1},
		Heartbeat:        Heartbeat{PingIntervalSeconds: 30, PongWaitSeconds: 60, PingWriteTimeoutSeconds: 10},
		Channel:          Channel{SendBufferSize: 255, RegisterBufferSize: 255, UnregisterBufferSize: 255},
		Ratelimit:        Ratelimit{EverySeconds: 12, Burst: 5},
		GracefulShutdown: GracefulShutdown{TimeoutSeconds: 5},
		Log:              Log{Level: "info", FilePath: "logs/gateway.log"},
		Jwt:              JWT{Secret: "test-secret-change-me", TTLHours: 24},
		Auth:             Auth{UserName: "zhangsan", PasswordHash: "$2a$10$1.u3pTISj0QHmvquKGDKOO8kxXVhSmCcqbdkN4HHLauKUsO8yl3U."},
		Redis:            Redis{Enabled: false, Addr: "localhost:6379", DB: 0},
	}
}

func TestApplyEnvOverrides(t *testing.T) {
	t.Run("环境变量不存在", func(t *testing.T) {
		config := defaultConfig()

		if err := config.ApplyEnvOverrides(); err != nil {
			t.Fatalf("ApplyEnvOverrides() 期望 nil, 得到 %v", err)
		}

		if config.Server.Port != 8080 {
			t.Errorf("Port 不为默认值, 实际得到 %d", config.Server.Port)
		}
		if config.Heartbeat.PingIntervalSeconds != 30 {
			t.Errorf("Port 不为默认值, 实际得到 %d", config.Heartbeat.PingIntervalSeconds)
		}
		if config.Heartbeat.PongWaitSeconds != 60 {
			t.Errorf("PongWaitSeconds 不为默认值, 实际得到 %d", config.Heartbeat.PongWaitSeconds)
		}
		if config.Ratelimit.EverySeconds != 12 {
			t.Errorf("EverySeconds 不为默认值, 实际得到 %d", config.Ratelimit.EverySeconds)
		}
		if config.Ratelimit.Burst != 5 {
			t.Errorf("Burst 不为默认值, 实际得到 %d", config.Ratelimit.Burst)
		}
		if config.GracefulShutdown.TimeoutSeconds != 5 {
			t.Errorf("TimeoutSeconds 不为默认值, 实际得到 %d", config.GracefulShutdown.TimeoutSeconds)
		}
		if config.Log.Level != "info" {
			t.Errorf("Level 不为默认值, 实际得到 %v", config.Log.Level)
		}
		if config.Log.FilePath != "logs/gateway.log" {
			t.Errorf("FilePath 不为默认值, 实际得到 %v", config.Log.FilePath)
		}
		if config.Jwt.Secret != "test-secret-change-me" {
			t.Errorf("Secret 不为默认值, 实际得到 %v", config.Jwt.Secret)
		}
		if config.Jwt.TTLHours != 24 {
			t.Errorf("TTLHours 不为默认值, 实际得到 %d", config.Jwt.TTLHours)
		}
		if config.Auth.UserName != "zhangsan" {
			t.Errorf("UserName 不为默认值，实际得到 %v", config.Auth.UserName)
		}
		if config.Auth.PasswordHash != "$2a$10$1.u3pTISj0QHmvquKGDKOO8kxXVhSmCcqbdkN4HHLauKUsO8yl3U." {
			t.Errorf("PasswordHash 不为默认值，实际得到 %v", config.Auth.PasswordHash)
		}
	})

	t.Run("端口覆盖", func(t *testing.T) {
		config := defaultConfig()

		os.Setenv("WS_PORT", "9090")
		defer os.Unsetenv("WS_PORT")

		if err := config.ApplyEnvOverrides(); err != nil {
			t.Fatalf("ApplyEnvOverrides() 期望 nil, 得到 %v", err)
		}

		if config.Server.Port != 9090 {
			t.Errorf("Port 环境变量设置失败, 实际得到 %d", config.Server.Port)
		}
	})

	t.Run("心跳间隔覆盖", func(t *testing.T) {
		config := defaultConfig()

		os.Setenv("WS_PING_INTERVAL", "15")
		defer os.Unsetenv("WS_PING_INTERVAL")

		if err := config.ApplyEnvOverrides(); err != nil {
			t.Fatalf("ApplyEnvOverrides() 期望 nil, 得到 %v", err)
		}

		if config.Heartbeat.PingIntervalSeconds != 15 {
			t.Errorf("PingIntervalSeconds 环境变量设置失败, 实际得到 %d", config.Heartbeat.PingIntervalSeconds)
		}
	})

	t.Run("Pong 超时覆盖", func(t *testing.T) {
		config := defaultConfig()

		os.Setenv("WS_PONG_WAIT", "45")
		defer os.Unsetenv("WS_PONG_WAIT")

		if err := config.ApplyEnvOverrides(); err != nil {
			t.Fatalf("ApplyEnvOverrides() 期望 nil, 得到 %v", err)
		}

		if config.Heartbeat.PongWaitSeconds != 45 {
			t.Errorf("PongWaitSeconds 环境变量设置失败, 实际得到 %d", config.Heartbeat.PongWaitSeconds)
		}
	})

	t.Run("限流参数覆盖", func(t *testing.T) {
		config := defaultConfig()

		os.Setenv("WS_RATELIMIT_INTERVAL", "5")
		os.Setenv("WS_BURST", "2")
		defer os.Unsetenv("WS_BURST")
		defer os.Unsetenv("WS_RATELIMIT_INTERVAL")

		if err := config.ApplyEnvOverrides(); err != nil {
			t.Fatalf("ApplyEnvOverrides() 期望 nil, 得到 %v", err)
		}

		if config.Ratelimit.EverySeconds != 5 {
			t.Errorf("EverySeconds 环境变量设置失败, 实际得到 %d", config.Ratelimit.EverySeconds)
		}
		if config.Ratelimit.Burst != 2 {
			t.Errorf("Burst 环境变量设置失败, 实际得到 %d", config.Ratelimit.Burst)
		}
	})

	t.Run("优雅退出宽限期覆盖", func(t *testing.T) {
		config := defaultConfig()

		os.Setenv("WS_SHUTDOWN_TIMEOUT", "3")
		defer os.Unsetenv("WS_SHUTDOWN_TIMEOUT")

		if err := config.ApplyEnvOverrides(); err != nil {
			t.Fatalf("ApplyEnvOverrides() 期望 nil, 得到 %v", err)
		}

		if config.GracefulShutdown.TimeoutSeconds != 3 {
			t.Errorf("TimeoutSeconds 环境变量设置失败, 实际得到 %d", config.GracefulShutdown.TimeoutSeconds)
		}
	})

	t.Run("日志参数覆盖", func(t *testing.T) {
		config := defaultConfig()

		os.Setenv("WS_LOG_LEVEL", "error")
		os.Setenv("WS_LOG_FILE", "logs/gateway2.log")
		defer os.Unsetenv("WS_LOG_FILE")
		defer os.Unsetenv("WS_LOG_LEVEL")

		if err := config.ApplyEnvOverrides(); err != nil {
			t.Fatalf("ApplyEnvOverrides() 期望 nil, 得到 %v", err)
		}

		if config.Log.Level != "error" {
			t.Errorf("Level 环境变量设置失败, 实际得到 %v", config.Log.Level)
		}
		if config.Log.FilePath != "logs/gateway2.log" {
			t.Errorf("FilePath 环境变量设置失败, 实际得到 %v", config.Log.FilePath)
		}
	})

	t.Run("鉴权参数覆盖", func(t *testing.T) {
		config := defaultConfig()

		os.Setenv("WS_JWT_SECRET", "test2-secret-change-me")
		os.Setenv("WS_JWT_TTL_HOURS", "10")
		defer os.Unsetenv("WS_JWT_TTL_HOURS")
		defer os.Unsetenv("WS_JWT_SECRET")

		if err := config.ApplyEnvOverrides(); err != nil {
			t.Fatalf("ApplyEnvOverrides() 期望 nil, 得到 %v", err)
		}

		if config.Jwt.Secret != "test2-secret-change-me" {
			t.Errorf("Secret 环境变量设置失败, 实际得到 %v", config.Jwt.Secret)
		}
		if config.Jwt.TTLHours != 10 {
			t.Errorf("TTLHours 环境变量设置失败, 实际得到 %d", config.Jwt.TTLHours)
		}
	})

	t.Run("身份凭证参数覆盖", func(t *testing.T) {
		config := defaultConfig()

		os.Setenv("WS_AUTH_USERNAME", "lisi")
		os.Setenv("WS_AUTH_PASSWORD_HASH", "$2a$10$tRUpNnJ5F9Z6SV0iqLjin.Wf2h9pl/rx6T4gNjqNDw3GApUgmriG.")
		defer os.Unsetenv("WS_AUTH_PASSWORD_HASH")
		defer os.Unsetenv("WS_AUTH_USERNAME")

		if err := config.ApplyEnvOverrides(); err != nil {
			t.Fatalf("ApplyEnvOverrides() 期望 nil, 得到 %v", err)
		}

		if config.Auth.UserName != "lisi" {
			t.Errorf("UserName 环境变量设置失败, 实际得到 %v", config.Auth.UserName)
		}
		if config.Auth.PasswordHash != "$2a$10$tRUpNnJ5F9Z6SV0iqLjin.Wf2h9pl/rx6T4gNjqNDw3GApUgmriG." {
			t.Errorf("PasswordHash 环境变量设置失败, 实际得到 %v", config.Auth.PasswordHash)
		}
	})

	t.Run("Redis 参数覆盖", func(t *testing.T) {
		config := defaultConfig()

		os.Setenv("WS_REDIS_ENABLED", "true")
		os.Setenv("WS_REDIS_ADDR", "redis.example.com:6380")
		os.Setenv("WS_REDIS_PASSWORD", "secret")
		os.Setenv("WS_REDIS_DB", "2")
		defer os.Unsetenv("WS_REDIS_DB")
		defer os.Unsetenv("WS_REDIS_PASSWORD")
		defer os.Unsetenv("WS_REDIS_ADDR")
		defer os.Unsetenv("WS_REDIS_ENABLED")

		if err := config.ApplyEnvOverrides(); err != nil {
			t.Fatalf("ApplyEnvOverrides() 期望 nil, 得到 %v", err)
		}

		if !config.Redis.Enabled {
			t.Errorf("Enabled 环境变量设置失败, 实际得到 %v", config.Redis.Enabled)
		}
		if config.Redis.Addr != "redis.example.com:6380" {
			t.Errorf("Addr 环境变量设置失败, 实际得到 %v", config.Redis.Addr)
		}
		if config.Redis.Password != "secret" {
			t.Errorf("Password 环境变量设置失败, 实际得到 %v", config.Redis.Password)
		}
		if config.Redis.DB != 2 {
			t.Errorf("DB 环境变量设置失败, 实际得到 %d", config.Redis.DB)
		}
	})

	t.Run("环境变量为空", func(t *testing.T) {
		config := defaultConfig()

		os.Setenv("WS_PORT", "")
		defer os.Unsetenv("WS_PORT")

		if err := config.ApplyEnvOverrides(); err != nil {
			t.Fatalf("ApplyEnvOverrides() 期望 nil, 得到 %v", err)
		}

		if config.Server.Port != 8080 {
			t.Errorf("Port 不为默认值, 实际得到 %d", config.Server.Port)
		}
	})

	t.Run("多个环境变量覆盖", func(t *testing.T) {
		config := defaultConfig()

		os.Setenv("WS_PORT", "9090")
		os.Setenv("WS_PING_INTERVAL", "15")
		defer os.Unsetenv("WS_PING_INTERVAL")
		defer os.Unsetenv("WS_PORT")

		if err := config.ApplyEnvOverrides(); err != nil {
			t.Fatalf("ApplyEnvOverrides() 期望 nil, 得到 %v", err)
		}

		if config.Server.Port != 9090 {
			t.Errorf("Port 环境变量设置失败, 实际得到 %d", config.Server.Port)
		}
		if config.Heartbeat.PingIntervalSeconds != 15 {
			t.Errorf("PingIntervalSeconds 环境变量设置失败, 实际得到 %d", config.Heartbeat.PingIntervalSeconds)
		}
	})

	t.Run("环境变量为非法值", func(t *testing.T) {
		tests := []struct {
			name    string
			key     string
			value   string
			wantErr string
		}{
			{"端口非数字", "WS_PORT", "abc", `环境变量 WS_PORT 的值 "abc" 无法解析为整数`},
			{"心跳间隔为小数", "WS_PING_INTERVAL", "12.5", `环境变量 WS_PING_INTERVAL 的值 "12.5" 无法解析为整数`},
			{"Pong 超时非数字", "WS_PONG_WAIT", "abc", `环境变量 WS_PONG_WAIT 的值 "abc" 无法解析为整数`},
			{"限流速率非数字", "WS_RATELIMIT_INTERVAL", "abc", `环境变量 WS_RATELIMIT_INTERVAL 的值 "abc" 无法解析为整数`},
			{"限流桶大小非数字", "WS_BURST", "abc", `环境变量 WS_BURST 的值 "abc" 无法解析为整数`},
			{"优雅退出宽限期含前导空格", "WS_SHUTDOWN_TIMEOUT", " 3", `环境变量 WS_SHUTDOWN_TIMEOUT 的值 " 3" 无法解析为整数`},
			{"鉴权过期时间非数字", "WS_JWT_TTL_HOURS", "abc", `环境变量 WS_JWT_TTL_HOURS 的值 "abc" 无法解析为整数`},
			{"Redis 开关非布尔值", "WS_REDIS_ENABLED", "yes", `环境变量 WS_REDIS_ENABLED 的值 "yes" 无法解析为布尔值`},
			{"Redis 库编号非数字", "WS_REDIS_DB", "abc", `环境变量 WS_REDIS_DB 的值 "abc" 无法解析为整数`},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				config := defaultConfig()

				os.Setenv(test.key, test.value)
				defer os.Unsetenv(test.key)

				err := config.ApplyEnvOverrides()

				if err == nil {
					t.Fatalf("ApplyEnvOverrides() 期望错误 %q, 得到 nil", test.wantErr)
				}
				if err.Error() != test.wantErr {
					t.Errorf("错误消息不匹配: got %q, want %q", err.Error(), test.wantErr)
				}
			})
		}
	})
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name                       string
		port                       int
		readBufferSize             int
		writeBufferSize            int
		readDeadlineSeconds        int
		writeDeadlineSeconds       int
		controlWriteTimeoutSeconds int
		pingIntervalSeconds        int
		pongWaitSeconds            int
		pingWriteTimeoutSeconds    int
		sendBufferSize             int
		registerBufferSize         int
		unregisterBufferSize       int
		everySeconds               int
		burst                      int
		timeoutSeconds             int
		level                      string
		secret                     string
		ttlHours                   int
		userName                   string
		passwordHash               string
		redisEnabled               bool
		redisAddr                  string
		redisDB                    int
		wantErr                    error
	}{
		{"配置合法", 8080, 1024, 1024, 60, 10, 1, 30, 60, 10, 255, 255, 255, 12, 5, 5, "info", "test-secret-change-me", 24, "zhangsan", "$2a$10$1.u3pTISj0QHmvquKGDKOO8kxXVhSmCcqbdkN4HHLauKUsO8yl3U.", false, "localhost:6379", 0, nil},
		{"端口为 0", 0, 1024, 1024, 60, 10, 1, 30, 60, 10, 255, 255, 255, 12, 5, 5, "info", "test-secret-change-me", 24, "zhangsan", "$2a$10$1.u3pTISj0QHmvquKGDKOO8kxXVhSmCcqbdkN4HHLauKUsO8yl3U.", false, "localhost:6379", 0, errors.New("server.port 必须在 1-65535 之间, 当前值: 0")},
		{"端口为 65536", 65536, 1024, 1024, 60, 10, 1, 30, 60, 10, 255, 255, 255, 12, 5, 5, "info", "test-secret-change-me", 24, "zhangsan", "$2a$10$1.u3pTISj0QHmvquKGDKOO8kxXVhSmCcqbdkN4HHLauKUsO8yl3U.", false, "localhost:6379", 0, errors.New("server.port 必须在 1-65535 之间, 当前值: 65536")},
		{"读缓冲区间为 0", 8080, 0, 1024, 60, 10, 1, 30, 60, 10, 255, 255, 255, 12, 5, 5, "info", "test-secret-change-me", 24, "zhangsan", "$2a$10$1.u3pTISj0QHmvquKGDKOO8kxXVhSmCcqbdkN4HHLauKUsO8yl3U.", false, "localhost:6379", 0, errors.New("websocket.read_buffer_size 必须 > 0, 当前值: 0")},
		{"写缓冲区间为 0", 8080, 1024, 0, 60, 10, 1, 30, 60, 10, 255, 255, 255, 12, 5, 5, "info", "test-secret-change-me", 24, "zhangsan", "$2a$10$1.u3pTISj0QHmvquKGDKOO8kxXVhSmCcqbdkN4HHLauKUsO8yl3U.", false, "localhost:6379", 0, errors.New("websocket.write_buffer_size 必须 > 0, 当前值: 0")},
		{"读超时为 0", 8080, 1024, 1024, 0, 10, 1, 30, 60, 10, 255, 255, 255, 12, 5, 5, "info", "test-secret-change-me", 24, "zhangsan", "$2a$10$1.u3pTISj0QHmvquKGDKOO8kxXVhSmCcqbdkN4HHLauKUsO8yl3U.", false, "localhost:6379", 0, errors.New("websocket.read_deadline_seconds 必须 > 0, 当前值: 0")},
		{"写超时为 0", 8080, 1024, 1024, 60, 0, 1, 30, 60, 10, 255, 255, 255, 12, 5, 5, "info", "test-secret-change-me", 24, "zhangsan", "$2a$10$1.u3pTISj0QHmvquKGDKOO8kxXVhSmCcqbdkN4HHLauKUsO8yl3U.", false, "localhost:6379", 0, errors.New("websocket.write_deadline_seconds 必须 > 0, 当前值: 0")},
		{"控制帧发送超时为 0", 8080, 1024, 1024, 60, 10, 0, 30, 60, 10, 255, 255, 255, 12, 5, 5, "info", "test-secret-change-me", 24, "zhangsan", "$2a$10$1.u3pTISj0QHmvquKGDKOO8kxXVhSmCcqbdkN4HHLauKUsO8yl3U.", false, "localhost:6379", 0, errors.New("websocket.control_write_timeout_seconds 必须 > 0, 当前值: 0")},
		{"心跳间隔为 0", 8080, 1024, 1024, 60, 10, 1, 0, 60, 10, 255, 255, 255, 12, 5, 5, "info", "test-secret-change-me", 24, "zhangsan", "$2a$10$1.u3pTISj0QHmvquKGDKOO8kxXVhSmCcqbdkN4HHLauKUsO8yl3U.", false, "localhost:6379", 0, errors.New("heartbeat.ping_interval_seconds 必须 > 0, 当前值: 0")},
		{"pong 超时为 0", 8080, 1024, 1024, 60, 10, 1, 30, 0, 10, 255, 255, 255, 12, 5, 5, "info", "test-secret-change-me", 24, "zhangsan", "$2a$10$1.u3pTISj0QHmvquKGDKOO8kxXVhSmCcqbdkN4HHLauKUsO8yl3U.", false, "localhost:6379", 0, errors.New("heartbeat.pong_wait_seconds 必须 > 0, 当前值: 0")},
		{"pong 超时小于心跳间隔", 8080, 1024, 1024, 60, 10, 1, 30, 20, 10, 255, 255, 255, 12, 5, 5, "info", "test-secret-change-me", 24, "zhangsan", "$2a$10$1.u3pTISj0QHmvquKGDKOO8kxXVhSmCcqbdkN4HHLauKUsO8yl3U.", false, "localhost:6379", 0, errors.New("heartbeat.pong_wait_seconds 必须 > 30, 当前值: 20")},
		{"pong 超时等于心跳间隔", 8080, 1024, 1024, 60, 10, 1, 30, 30, 10, 255, 255, 255, 12, 5, 5, "info", "detestv-secret-change-me", 24, "zhangsan", "$2a$10$1.u3pTISj0QHmvquKGDKOO8kxXVhSmCcqbdkN4HHLauKUsO8yl3U.", false, "localhost:6379", 0, errors.New("heartbeat.pong_wait_seconds 必须 > 30, 当前值: 30")},
		{"ping 帧发送超时为 0", 8080, 1024, 1024, 60, 10, 1, 30, 60, 0, 255, 255, 255, 12, 5, 5, "info", "test-secret-change-me", 24, "zhangsan", "$2a$10$1.u3pTISj0QHmvquKGDKOO8kxXVhSmCcqbdkN4HHLauKUsO8yl3U.", false, "localhost:6379", 0, errors.New("heartbeat.ping_write_timeout_seconds 必须 > 0, 当前值: 0")},
		{"send 缓冲大小为 0", 8080, 1024, 1024, 60, 10, 1, 30, 60, 10, 0, 255, 255, 12, 5, 5, "info", "test-secret-change-me", 24, "zhangsan", "$2a$10$1.u3pTISj0QHmvquKGDKOO8kxXVhSmCcqbdkN4HHLauKUsO8yl3U.", false, "localhost:6379", 0, errors.New("channel.send_buffer_size 必须 > 0, 当前值: 0")},
		{"register 缓冲大小为 0", 8080, 1024, 1024, 60, 10, 1, 30, 60, 10, 255, 0, 255, 12, 5, 5, "info", "test-secret-change-me", 24, "zhangsan", "$2a$10$1.u3pTISj0QHmvquKGDKOO8kxXVhSmCcqbdkN4HHLauKUsO8yl3U.", false, "localhost:6379", 0, errors.New("channel.register_buffer_size 必须 > 0, 当前值: 0")},
		{"unregister 缓冲大小为 0", 8080, 1024, 1024, 60, 10, 1, 30, 60, 10, 255, 255, 0, 12, 5, 5, "info", "test-secret-change-me", 24, "zhangsan", "$2a$10$1.u3pTISj0QHmvquKGDKOO8kxXVhSmCcqbdkN4HHLauKUsO8yl3U.", false, "localhost:6379", 0, errors.New("channel.unregister_buffer_size 必须 > 0, 当前值: 0")},
		{"限流速率为 0", 8080, 1024, 1024, 60, 10, 1, 30, 60, 10, 255, 255, 255, 0, 5, 5, "info", "test-secret-change-me", 24, "zhangsan", "$2a$10$1.u3pTISj0QHmvquKGDKOO8kxXVhSmCcqbdkN4HHLauKUsO8yl3U.", false, "localhost:6379", 0, errors.New("ratelimit.every_seconds 必须 > 0, 当前值: 0")},
		{"限流桶大小为 0", 8080, 1024, 1024, 60, 10, 1, 30, 60, 10, 255, 255, 255, 12, 0, 5, "info", "test-secret-change-me", 24, "zhangsan", "$2a$10$1.u3pTISj0QHmvquKGDKOO8kxXVhSmCcqbdkN4HHLauKUsO8yl3U.", false, "localhost:6379", 0, errors.New("ratelimit.burst 必须 > 0, 当前值: 0")},
		{"优雅退出宽限期为 0", 8080, 1024, 1024, 60, 10, 1, 30, 60, 10, 255, 255, 255, 12, 5, 0, "info", "test-secret-change-me", 24, "zhangsan", "$2a$10$1.u3pTISj0QHmvquKGDKOO8kxXVhSmCcqbdkN4HHLauKUsO8yl3U.", false, "localhost:6379", 0, errors.New("graceful_shutdown.timeout_seconds 必须 > 0, 当前值: 0")},
		{"日志级别为空", 8080, 1024, 1024, 60, 10, 1, 30, 60, 10, 255, 255, 255, 12, 5, 5, "", "test-secret-change-me", 24, "zhangsan", "$2a$10$1.u3pTISj0QHmvquKGDKOO8kxXVhSmCcqbdkN4HHLauKUsO8yl3U.", false, "localhost:6379", 0, errors.New("log.level 必须为 debug/info/warn/error 其中一个, 当前值: ")},
		{"密钥为空", 8080, 1024, 1024, 60, 10, 1, 30, 60, 10, 255, 255, 255, 12, 5, 5, "info", "", 24, "zhangsan", "$2a$10$1.u3pTISj0QHmvquKGDKOO8kxXVhSmCcqbdkN4HHLauKUsO8yl3U.", false, "localhost:6379", 0, errors.New("jwt.secret 不能为空, 当前值: ")},
		{"密钥长度小于 16", 8080, 1024, 1024, 60, 10, 1, 30, 60, 10, 255, 255, 255, 12, 5, 5, "info", "test3-secret", 24, "zhangsan", "$2a$10$1.u3pTISj0QHmvquKGDKOO8kxXVhSmCcqbdkN4HHLauKUsO8yl3U.", false, "localhost:6379", 0, errors.New("jwt.secret 长度不能少于 16 个字符(建议 32 以上), 当前长度: 12")},
		{"鉴权过期时间为 0", 8080, 1024, 1024, 60, 10, 1, 30, 60, 10, 255, 255, 255, 12, 5, 5, "info", "test-secret-change-me", 0, "zhangsan", "$2a$10$1.u3pTISj0QHmvquKGDKOO8kxXVhSmCcqbdkN4HHLauKUsO8yl3U.", false, "localhost:6379", 0, errors.New("jwt.ttl_hours 必须 > 0, 当前值: 0")},
		{"预置账号名为空", 8080, 1024, 1024, 60, 10, 1, 30, 60, 10, 255, 255, 255, 12, 5, 5, "info", "test-secret-change-me", 24, "", "$2a$10$1.u3pTISj0QHmvquKGDKOO8kxXVhSmCcqbdkN4HHLauKUsO8yl3U.", false, "localhost:6379", 0, errors.New("auth.username 不能为空, 当前值: ")},
		{"预置哈希值为空", 8080, 1024, 1024, 60, 10, 1, 30, 60, 10, 255, 255, 255, 12, 5, 5, "info", "test-secret-change-me", 24, "zhangsan", "", false, "localhost:6379", 0, errors.New("auth.password_hash 不能为空, 当前值: ")},
		{"预置哈希值格式错误", 8080, 1024, 1024, 60, 10, 1, 30, 60, 10, 255, 255, 255, 12, 5, 5, "info", "test-secret-change-me", 24, "zhangsan", "default", false, "localhost:6379", 0, errors.New("auth.password_hash 必须为合法的 bcrypt 哈希(以 $2a$/$2b$/$2y$ 开头, 长度 60), 当前值: default")},
		{"启用 Redis 但地址为空", 8080, 1024, 1024, 60, 10, 1, 30, 60, 10, 255, 255, 255, 12, 5, 5, "info", "test-secret-change-me", 24, "zhangsan", "$2a$10$1.u3pTISj0QHmvquKGDKOO8kxXVhSmCcqbdkN4HHLauKUsO8yl3U.", true, "", 0, errors.New("redis.addr 在启用 Redis 时不能为空")},
		{"Redis 数据库编号为负数", 8080, 1024, 1024, 60, 10, 1, 30, 60, 10, 255, 255, 255, 12, 5, 5, "info", "test-secret-change-me", 24, "zhangsan", "$2a$10$1.u3pTISj0QHmvquKGDKOO8kxXVhSmCcqbdkN4HHLauKUsO8yl3U.", false, "localhost:6379", -1, errors.New("redis.db 不能为负数, 当前值: -1")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := defaultConfig()

			config.Server.Port = test.port
			config.Websocket.ReadBufferSize = test.readBufferSize
			config.Websocket.WriteBufferSize = test.writeBufferSize
			config.Websocket.ReadDeadlineSeconds = test.readDeadlineSeconds
			config.Websocket.WriteDeadlineSeconds = test.writeDeadlineSeconds
			config.Websocket.ControlWriteTimeoutSeconds = test.controlWriteTimeoutSeconds
			config.Heartbeat.PingIntervalSeconds = test.pingIntervalSeconds
			config.Heartbeat.PongWaitSeconds = test.pongWaitSeconds
			config.Heartbeat.PingWriteTimeoutSeconds = test.pingWriteTimeoutSeconds
			config.Channel.SendBufferSize = test.sendBufferSize
			config.Channel.RegisterBufferSize = test.registerBufferSize
			config.Channel.UnregisterBufferSize = test.unregisterBufferSize
			config.Ratelimit.EverySeconds = test.everySeconds
			config.Ratelimit.Burst = test.burst
			config.GracefulShutdown.TimeoutSeconds = test.timeoutSeconds
			config.Log.Level = test.level
			config.Jwt.Secret = test.secret
			config.Jwt.TTLHours = test.ttlHours
			config.Auth.UserName = test.userName
			config.Auth.PasswordHash = test.passwordHash
			config.Redis.Enabled = test.redisEnabled
			config.Redis.Addr = test.redisAddr
			config.Redis.DB = test.redisDB

			got := config.Validate()

			if test.wantErr == nil {
				if got != nil {
					t.Errorf("Validate() 期望 nil, 得到 %v", got)
				}
			} else {
				if got == nil {
					t.Errorf("Validate() 期望错误 %q, 得到 nil", test.wantErr.Error())
				} else if got.Error() != test.wantErr.Error() {
					t.Errorf("Validate() 错误消息不匹配: got %q, want %q", got.Error(), test.wantErr.Error())
				}
			}
		})
	}
}

func TestConfigHelpers(t *testing.T) {
	cfg := defaultConfig()

	tests := []struct {
		name string
		got  time.Duration
		want time.Duration
	}{
		{"PingInterval", cfg.PingInterval(), 30 * time.Second},
		{"PongWait", cfg.PongWait(), 60 * time.Second},
		{"PingWriteTimeout", cfg.PingWriteTimeout(), 10 * time.Second},
		{"ReadDeadline", cfg.ReadDeadline(), 60 * time.Second},
		{"WriteDeadline", cfg.WriteDeadline(), 10 * time.Second},
		{"ControlWriteTimeout", cfg.ControlWriteTimeout(), 1 * time.Second},
		{"ShutdownTimeout", cfg.ShutdownTimeout(), 5 * time.Second},
		{"RateLimitInterval", cfg.RateLimitInterval(), 12 * time.Second},
		{"TokenTTL", cfg.TokenTTL(), 24 * time.Hour},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.got != test.want {
				t.Errorf("%s() = %v, want %v", test.name, test.got, test.want)
			}
		})
	}
}

func TestIsBcryptHash(t *testing.T) {
	tests := []struct {
		name string
		hash string
		want bool
	}{
		{"合法的 $2a$ 哈希", "$2a$10$MqkF//ZWuwkD2z1gJdPYZuURfCrSs4VLa7.dNX1VnBTYg28.lrCka", true},
		{"合法的 $2b$ 哈希", "$2b$10$MqkF//ZWuwkD2z1gJdPYZuURfCrSs4VLa7.dNX1VnBTYg28.lrCka", true},
		{"合法的 $2y$ 哈希", "$2y$10$MqkF//ZWuwkD2z1gJdPYZuURfCrSs4VLa7.dNX1VnBTYg28.lrCka", true},
		{"空字符串", "", false},
		{"非 bcrypt 占位符", "default", false},
		{"长度不足 60 位", "$2a$10$MqkF//ZWuwkD2z1gJdPYZuURfCrSs4VLa7.dNX1VnBTYg28.lrCk", false},
		{"长度超过 60 位", "$2a$10$MqkF//ZWuwkD2z1gJdPYZuURfCrSs4VLa7.dNX1VnBTYg28.lrCkaa", false},
		{"版本前缀不支持", "$2x$10$MqkF//ZWuwkD2z1gJdPYZuURfCrSs4VLa7.dNX1VnBTYg28.lrCka", false},
		{"缺少 $ 前缀", "x2a$10$MqkF//ZWuwkD2z1gJdPYZuURfCrSs4VLa7.dNX1VnBTYg28.lrCka", false},
		{"成本因子非数字", "$2a$ab$MqkF//ZWuwkD2z1gJdPYZuURfCrSs4VLa7.dNX1VnBTYg28.lrCka", false},
		{"盐摘要含标准 base64 加号", "$2a$10$MqkF+/ZWuwkD2z1gJdPYZuURfCrSs4VLa7.dNX1VnBTYg28.lrCka", false},
		{"盐摘要含标准 base64 等号", "$2a$10$MqkF/=ZWuwkD2z1gJdPYZuURfCrSs4VLa7.dNX1VnBTYg28.lrCka", false},
		{"盐摘要含空格", "$2a$10$MqkF/ ZWuwkD2z1gJdPYZuURfCrSs4VLa7.dNX1VnBTYg28.lrCka", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isBcryptHash(test.hash); got != test.want {
				t.Errorf("isBcryptHash(%q) 期望 %v, 实际得到 %v", test.hash, test.want, got)
			}
		})
	}
}
