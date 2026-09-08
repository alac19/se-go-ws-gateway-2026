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
	}
}

func TestApplyEnvOverrides(t *testing.T) {
	t.Run("环境变量不存在", func(t *testing.T) {
		config := defaultConfig()

		config.ApplyEnvOverrides()

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
	})

	t.Run("端口覆盖", func(t *testing.T) {
		config := defaultConfig()

		os.Setenv("WS_PORT", "9090")
		defer os.Unsetenv("WS_PORT")

		config.ApplyEnvOverrides()

		if config.Server.Port != 9090 {
			t.Errorf("Port 环境变量设置失败, 实际得到 %d", config.Server.Port)
		}
	})

	t.Run("心跳间隔覆盖", func(t *testing.T) {
		config := defaultConfig()

		os.Setenv("WS_PING_INTERVAL", "15")
		defer os.Unsetenv("WS_PING_INTERVAL")

		config.ApplyEnvOverrides()

		if config.Heartbeat.PingIntervalSeconds != 15 {
			t.Errorf("PingIntervalSeconds 环境变量设置失败, 实际得到 %d", config.Heartbeat.PingIntervalSeconds)
		}
	})

	t.Run("Pong 超时覆盖", func(t *testing.T) {
		config := defaultConfig()

		os.Setenv("WS_PONG_WAIT", "45")
		defer os.Unsetenv("WS_PONG_WAIT")

		config.ApplyEnvOverrides()

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

		config.ApplyEnvOverrides()

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

		config.ApplyEnvOverrides()

		if config.GracefulShutdown.TimeoutSeconds != 3 {
			t.Errorf("TimeoutSeconds 环境变量设置失败, 实际得到 %d", config.GracefulShutdown.TimeoutSeconds)
		}
	})

	t.Run("环境变量为空", func(t *testing.T) {
		config := defaultConfig()

		os.Setenv("WS_PORT", "")
		defer os.Unsetenv("WS_PORT")

		config.ApplyEnvOverrides()

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

		config.ApplyEnvOverrides()

		if config.Server.Port != 9090 {
			t.Errorf("Port 环境变量设置失败, 实际得到 %d", config.Server.Port)
		}
		if config.Heartbeat.PingIntervalSeconds != 15 {
			t.Errorf("PingIntervalSeconds 环境变量设置失败, 实际得到 %d", config.Heartbeat.PingIntervalSeconds)
		}
	})

	t.Run("环境变量为非法值", func(t *testing.T) {
		config := defaultConfig()

		os.Setenv("WS_PORT", "abc")
		defer os.Unsetenv("WS_PORT")

		config.ApplyEnvOverrides()

		if config.Server.Port != 8080 {
			t.Errorf("Port 不为默认值, 实际得到 %d", config.Server.Port)
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
		wantErr                    error
	}{
		{"配置合法", 8080, 1024, 1024, 60, 10, 1, 30, 60, 10, 255, 255, 255, 12, 5, 5, nil},
		{"端口为 0", 0, 1024, 1024, 60, 10, 1, 30, 60, 10, 255, 255, 255, 12, 5, 5, errors.New("server.port 必须在 1-65535 之间, 当前值: 0")},
		{"端口为 65536", 65536, 1024, 1024, 60, 10, 1, 30, 60, 10, 255, 255, 255, 12, 5, 5, errors.New("server.port 必须在 1-65535 之间, 当前值: 65536")},
		{"读缓冲区间为 0", 8080, 0, 1024, 60, 10, 1, 30, 60, 10, 255, 255, 255, 12, 5, 5, errors.New("websocket.read_buffer_size 必须 > 0, 当前值: 0")},
		{"写缓冲区间为 0", 8080, 1024, 0, 60, 10, 1, 30, 60, 10, 255, 255, 255, 12, 5, 5, errors.New("websocket.write_buffer_size 必须 > 0, 当前值: 0")},
		{"读超时为 0", 8080, 1024, 1024, 0, 10, 1, 30, 60, 10, 255, 255, 255, 12, 5, 5, errors.New("websocket.read_deadline_seconds 必须 > 0, 当前值: 0")},
		{"写超时为 0", 8080, 1024, 1024, 60, 0, 1, 30, 60, 10, 255, 255, 255, 12, 5, 5, errors.New("websocket.write_deadline_seconds 必须 > 0, 当前值: 0")},
		{"控制帧发送超时为 0", 8080, 1024, 1024, 60, 10, 0, 30, 60, 10, 255, 255, 255, 12, 5, 5, errors.New("websocket.control_write_timeout_seconds 必须 > 0, 当前值: 0")},
		{"心跳间隔为 0", 8080, 1024, 1024, 60, 10, 1, 0, 60, 10, 255, 255, 255, 12, 5, 5, errors.New("heartbeat.ping_interval_seconds 必须 > 0, 当前值: 0")},
		{"pong 超时为 0", 8080, 1024, 1024, 60, 10, 1, 30, 0, 10, 255, 255, 255, 12, 5, 5, errors.New("heartbeat.pong_wait_seconds 必须 > 0, 当前值: 0")},
		{"pong 超时小于心跳间隔", 8080, 1024, 1024, 60, 10, 1, 30, 20, 10, 255, 255, 255, 12, 5, 5, errors.New("heartbeat.pong_wait_seconds 必须 > 30, 当前值: 20")},
		{"pong 超时等于心跳间隔", 8080, 1024, 1024, 60, 10, 1, 30, 30, 10, 255, 255, 255, 12, 5, 5, errors.New("heartbeat.pong_wait_seconds 必须 > 30, 当前值: 30")},
		{"ping 帧发送超时为 0", 8080, 1024, 1024, 60, 10, 1, 30, 60, 0, 255, 255, 255, 12, 5, 5, errors.New("heartbeat.ping_write_timeout_seconds 必须 > 0, 当前值: 0")},
		{"send 缓冲大小为 0", 8080, 1024, 1024, 60, 10, 1, 30, 60, 10, 0, 255, 255, 12, 5, 5, errors.New("channel.send_buffer_size 必须 > 0, 当前值: 0")},
		{"register 缓冲大小为 0", 8080, 1024, 1024, 60, 10, 1, 30, 60, 10, 255, 0, 255, 12, 5, 5, errors.New("channel.register_buffer_size 必须 > 0, 当前值: 0")},
		{"unregister 缓冲大小为 0", 8080, 1024, 1024, 60, 10, 1, 30, 60, 10, 255, 255, 0, 12, 5, 5, errors.New("channel.unregister_buffer_size 必须 > 0, 当前值: 0")},
		{"限流速率为 0", 8080, 1024, 1024, 60, 10, 1, 30, 60, 10, 255, 255, 255, 0, 5, 5, errors.New("ratelimit.every_seconds 必须 > 0, 当前值: 0")},
		{"限流桶大小为 0", 8080, 1024, 1024, 60, 10, 1, 30, 60, 10, 255, 255, 255, 12, 0, 5, errors.New("ratelimit.burst 必须 > 0, 当前值: 0")},
		{"优雅退出宽限期为 0", 8080, 1024, 1024, 60, 10, 1, 30, 60, 10, 255, 255, 255, 12, 5, 0, errors.New("graceful_shutdown.timeout_seconds 必须 > 0, 当前值: 0")},
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
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.got != test.want {
				t.Errorf("%s() = %v, want %v", test.name, test.got, test.want)
			}
		})
	}
}
