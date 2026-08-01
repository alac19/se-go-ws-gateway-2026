package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	model "github.com/alac/se-go-ws-gateway-2026/internal/model"
	metrics "github.com/alac/se-go-ws-gateway-2026/pkg/metrics"
)

func TestRegister(t *testing.T) {
	cm := NewClientManager(nil, 10, 10)
	client := model.NewClient("test1", "room1", nil, 5, time.Now())

	cm.Register(client)

	select {
	case received := <-cm.register:
		if received != client {
			t.Errorf("期望 %v, 得到 %v", client, received)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("register 通道没有收到事件")
	}
}

func TestUnRegister(t *testing.T) {
	cm := NewClientManager(nil, 10, 10)
	client := model.NewClient("test1", "room1", nil, 5, time.Now())

	cm.Unregister(client.ClientID)

	select {
	case received := <-cm.unregister:
		if received != client.ClientID {
			t.Errorf("期望 %v, 得到 %v", client.ClientID, received)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("unregister 通道没有收到事件")
	}
}

func TestGet(t *testing.T) {
	cm := NewClientManager(nil, 10, 10)
	client := model.NewClient("test1", "room1", nil, 5, time.Now())

	cm.clients.Store(client.ClientID, client)

	tests := []struct {
		name       string
		client     *model.Client
		wantClient *model.Client
		wantBool   bool
	}{
		{"客户端存在", client, model.NewClient("test1", "room1", nil, 5, time.Now()), true},
		{"客户端不存在", model.NewClient("test2", "room1", nil, 5, time.Now()), nil, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := cm.Get(test.client.ClientID)

			if ok != test.wantBool {
				t.Errorf("期望 %v, 得到 %v", test.wantBool, ok)
			}
			if test.wantBool {
				if got != nil && got.ClientID != test.wantClient.ClientID {
					t.Errorf("期望 %v, 得到 %v", test.wantClient.ClientID, got.ClientID)
				}
			} else {
				if got != nil {
					t.Errorf("期望 nil, 得到 %v", got)
				}
			}
		})
	}
}

func TestGetOnlineCount(t *testing.T) {
	t.Run("在线连接数为 0", func(t *testing.T) {
		cm := NewClientManager(nil, 10, 10)

		got := cm.GetOnlineCount()

		if got != 0 {
			t.Errorf("期望 0, 得到 %v", got)
		}
	})

	t.Run("在线连接数为 2", func(t *testing.T) {
		cm := NewClientManager(nil, 10, 10)
		client1 := model.NewClient("test1", "room1", nil, 5, time.Now())
		client2 := model.NewClient("test2", "room1", nil, 5, time.Now())

		cm.clients.Store(client1.ClientID, client1)
		cm.clients.Store(client2.ClientID, client2)
		got := cm.GetOnlineCount()

		if got != 2 {
			t.Errorf("期望 2, 得到 %v", got)
		}
	})
}

// getGaugeValue 从注册器中获取指定名称的 Gauge 指标值。
func getGaugeValue(reg prometheus.Gatherer, name string) (float64, error) {
	mfs, err := reg.Gather()

	if err != nil {
		return 0, err
	}

	for _, mf := range mfs {
		if mf.GetName() == name {
			if mf.GetType() != dto.MetricType_GAUGE {
				return 0, fmt.Errorf("metric %s is not a Gauge", name)
			}

			metrics := mf.GetMetric()

			if len(metrics) == 0 {
				return 0, nil
			}

			return metrics[0].GetGauge().GetValue(), nil
		}
	}

	return 0, fmt.Errorf("metric %s not found", name)
}

func TestInitAndGaugeValue(t *testing.T) {
	reg := prometheus.NewRegistry()

	metrics.Init(reg)

	val, err := getGaugeValue(reg, "ws_active_connections")

	if err != nil {
		t.Fatal(err)
	}
	if val != 0 {
		t.Errorf("期望初始值为 0, 得到 %v", val)
	}

	metrics.OnlineConnGauge.Set(5)
	val, err = getGaugeValue(reg, "ws_active_connections")

	if err != nil {
		t.Fatal(err)
	}
	if val != 5 {
		t.Errorf("期望值为 5, 得到 %v", val)
	}
}

// getCounterValue 从注册器中获取指定名称的 Counter 指标值。
func getCounterValue(reg prometheus.Gatherer, name string) (float64, error) {
	mfs, err := reg.Gather()

	if err != nil {
		return 0, err
	}

	for _, mf := range mfs {
		if mf.GetName() == name {
			if mf.GetType() != dto.MetricType_COUNTER {
				return 0, fmt.Errorf("metric %s is not a Counter", name)
			}

			metrics := mf.GetMetric()

			if len(metrics) == 0 {
				return 0, nil
			}

			return metrics[0].GetCounter().GetValue(), nil
		}
	}

	return 0, fmt.Errorf("metric %s not found", name)
}

func TestInitAndCounterValue(t *testing.T) {
	reg := prometheus.NewRegistry()

	metrics.Init(reg)

	val, err := getCounterValue(reg, "ws_connection_events_total")

	if err != nil {
		t.Fatal(err)
	}
	if val != 0 {
		t.Errorf("期望初始值为 0, 得到 %v", val)
	}

	metrics.ConnEventTotal.Inc()
	val, err = getCounterValue(reg, "ws_connection_events_total")

	if err != nil {
		t.Fatal(err)
	}
	if val != 1 {
		t.Errorf("期望值为 1, 得到 %v", val)
	}
}

func TestInit(t *testing.T) {
	t.Run("注册成功", func(t *testing.T) {
		reg := prometheus.NewRegistry()

		metrics.Init(reg)

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		roomMgr := NewRoomManager()
		cm := NewClientManager(roomMgr, 10, 10)

		go cm.Init(ctx, 1*time.Second)

		client := model.NewClient("test1", "room1", nil, 5, time.Now())

		cm.Register(client)
		waitForClients(t, cm, 1, waitTimeout)

		got, ok := cm.Get("test1")

		if !ok {
			t.Error("期望客户端存在, 但 Get 返回 false")
		}
		if got != nil && got.ClientID != "test1" {
			t.Errorf("期望 clientID 为 test1, 得到 %v", got.ClientID)
		}

		if cm.GetOnlineCount() != 1 {
			t.Errorf("期望在线连接数为 1, 得到 %d", cm.GetOnlineCount())
		}

		val, err := getGaugeValue(reg, "ws_active_connections")

		if err != nil {
			t.Fatal(err)
		}
		if val != 1 {
			t.Errorf("期望 Gauge 值为 1, 得到 %v", val)
		}

		val, err = getCounterValue(reg, "ws_connection_events_total")

		if err != nil {
			t.Fatal(err)
		}
		if val != 1 {
			t.Errorf("期望 Counter 值为 1, 得到 %v", val)
		}

		got2 := cm.roomMgr.GetClients("room1")

		var flag bool
		for _, g := range got2 {
			if g == "test1" {
				flag = true
				break
			}
		}
		if !flag {
			t.Errorf("期望 test1 存在, 但 GetClients 中未找到")
		}
	})

	t.Run("重复注册被拒绝", func(t *testing.T) {
		reg := prometheus.NewRegistry()

		metrics.Init(reg)

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		roomMgr := NewRoomManager()
		cm := NewClientManager(roomMgr, 10, 10)

		go cm.Init(ctx, 1*time.Second)

		client1 := model.NewClient("test2", "room2", nil, 5, time.Now())

		cm.Register(client1)
		waitForClients(t, cm, 1, waitTimeout)

		val, err := getCounterValue(reg, "ws_connection_events_total")

		if err != nil {
			t.Fatal(err)
		}
		if val != 1 {
			t.Errorf("期望 Counter 值为 1, 得到 %v", val)
		}

		client2 := model.NewClient("test2", "room2", nil, 5, time.Now())
		cm.Register(client2)
		waitForClients(t, cm, 1, waitTimeout)

		if cm.GetOnlineCount() != 1 {
			t.Errorf("期望在线连接数仍为 1, 得到 %d", cm.GetOnlineCount())
		}

		val, err = getGaugeValue(reg, "ws_active_connections")

		if err != nil {
			t.Fatal(err)
		}
		if val != 1 {
			t.Errorf("期望 Gauge 值仍为 1, 得到 %v", val)
		}

		val, err = getCounterValue(reg, "ws_connection_events_total")

		if err != nil {
			t.Fatal(err)
		}
		if val != 1 {
			t.Errorf("期望 Counter 值仍为 1, 得到 %v", val)
		}
	})

	t.Run("注销成功", func(t *testing.T) {
		reg := prometheus.NewRegistry()

		metrics.Init(reg)

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		roomMgr := NewRoomManager()
		cm := NewClientManager(roomMgr, 10, 10)

		go cm.Init(ctx, 1*time.Second)

		client := model.NewClient("test3", "room1", nil, 5, time.Now())

		cm.Register(client)
		waitForClients(t, cm, 1, waitTimeout)
		cm.Unregister(client.ClientID)
		waitForClientRemoved(t, cm, "test3", waitTimeout)

		val, err := getGaugeValue(reg, "ws_active_connections")

		if err != nil {
			t.Fatal(err)
		}
		if val != 0 {
			t.Errorf("期望 Gauge 值为 0, 得到 %v", val)
		}

		val, err = getCounterValue(reg, "ws_connection_events_total")

		if err != nil {
			t.Fatal(err)
		}
		if val != 2 {
			t.Errorf("期望 Counter 值为 2, 得到 %v", val)
		}

		got2 := cm.roomMgr.GetClients("room1")

		var flag bool
		for _, g := range got2 {
			if g == "test3" {
				flag = true
				break
			}
		}
		if flag {
			t.Errorf("期望 test3 不存在, 但 GetClients 中却被找到")
		}
	})

	t.Run("clientID 不存在, 注销失败", func(t *testing.T) {
		reg := prometheus.NewRegistry()

		metrics.Init(reg)

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		roomMgr := NewRoomManager()
		cm := NewClientManager(roomMgr, 10, 10)

		go cm.Init(ctx, 1*time.Second)

		client := model.NewClient("test4", "room4", nil, 5, time.Now())

		cm.Register(client)
		waitForClients(t, cm, 1, waitTimeout)
		cm.Unregister("test999")
		waitForClients(t, cm, 1, waitTimeout)

		val, err := getGaugeValue(reg, "ws_active_connections")

		if err != nil {
			t.Fatal(err)
		}
		if val != 1 {
			t.Errorf("期望 Gauge 值为 1, 得到 %v", val)
		}

		val, err = getCounterValue(reg, "ws_connection_events_total")

		if err != nil {
			t.Fatal(err)
		}
		if val != 1 {
			t.Errorf("期望 Counter 值为 1, 得到 %v", val)
		}

		got2 := cm.roomMgr.GetClients("room4")

		var flag bool
		for _, g := range got2 {
			if g == "test4" {
				flag = true
				break
			}
		}
		if !flag {
			t.Errorf("期望 test4 存在, 但 GetClients 中未找到")
		}
	})
}

func TestShutdown(t *testing.T) {
	roomMgr := NewRoomManager()
	cm := NewClientManager(roomMgr, 10, 10)

	client1 := model.NewClient("test1", "room1", nil, 5, time.Now())
	client2 := model.NewClient("test2", "room2", nil, 5, time.Now())
	cm.clients.Store(client1.ClientID, client1)
	cm.clients.Store(client2.ClientID, client2)

	cm.Shutdown(10*time.Millisecond, 1*time.Second)

	if !cm.shuttingDown {
		t.Error("期望 shuttingDown = true")
	}

	if _, ok := cm.Get("test1"); ok {
		t.Error("test1 应该已被删除")
	}
	if _, ok := cm.Get("test2"); ok {
		t.Error("test2 应该已被删除")
	}
}
