package service

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	model "github.com/alac/se-go-ws-gateway-2026/internal/model"
	metrics "github.com/alac/se-go-ws-gateway-2026/pkg/metrics"
)

// getCounterVecValue 从注册器中获取指定名称的 CounterVec 指标值。
func getCounterVecValue(reg prometheus.Gatherer, name string, labelName string, labelValue string) (float64, error) {
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

			for i, ms := range metrics {
				for _, lps := range ms.GetLabel() {
					if lps.GetName() == labelName && lps.GetValue() == labelValue {
						return metrics[i].GetCounter().GetValue(), nil
					}
				}
			}
		}
	}

	return 0, fmt.Errorf("metric %s not found", name)
}

func TestInitAndCounterVecValue(t *testing.T) {
	reg := prometheus.NewRegistry()

	metrics.Init(reg)

	metrics.MsgSendTotal.WithLabelValues("single").Inc()
	val, err := getCounterVecValue(reg, "ws_gateway_msg_sent_total", "msg_type", "single")

	if err != nil {
		t.Fatal(err)
	}
	if val != 1 {
		t.Errorf("期望值为 1, 得到 %v", val)
	}
}

func resetRouter() {
	globalRouter = nil
	routerOnce = sync.Once{}
}

func TestSendSingle(t *testing.T) {
	t.Run("目标在线", func(t *testing.T) {
		reg := prometheus.NewRegistry()

		metrics.Init(reg)

		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()
		resetRouter()

		roomMgr := NewRoomManager()
		clientMgr := NewClientManager(roomMgr, 10, 10)
		router := NewMessageRouter(clientMgr, roomMgr, nil)

		go clientMgr.Init(ctx, 1*time.Second)

		client := model.NewClient("test1", "room1", nil, 5, time.Now())

		clientMgr.Register(client)
		waitForClients(t, clientMgr, 1, waitTimeout)

		msg := &model.Message{Payload: []byte("hello")}
		ok := router.SendSingle("test1", msg)

		if !ok {
			t.Error("期望返回 true, 结果为 false")
		} else {
			select {
			case received := <-client.SendChan:
				if !bytes.Equal(received, msg.Payload) {
					t.Errorf("期望 %v, 得到 %v", msg.Payload, received)
				}
			case <-time.After(100 * time.Millisecond):
				t.Error("SendChan 没有收到事件")
			}

			val, err := getCounterVecValue(reg, "ws_gateway_msg_sent_total", "msg_type", "single")

			if err != nil {
				t.Fatal(err)
			}
			if val != 1 {
				t.Errorf("期望 Counter 值为 1, 得到 %v", val)
			}
		}
	})

	t.Run("目标离线", func(t *testing.T) {
		reg := prometheus.NewRegistry()

		metrics.Init(reg)

		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()
		resetRouter()

		roomMgr := NewRoomManager()
		clientMgr := NewClientManager(roomMgr, 10, 10)
		router := NewMessageRouter(clientMgr, roomMgr, nil)

		go clientMgr.Init(ctx, 1*time.Second)

		client := model.NewClient("test1", "room1", nil, 5, time.Now())

		clientMgr.Register(client)
		waitForClients(t, clientMgr, 1, waitTimeout)

		msg := &model.Message{Payload: []byte("hello")}
		ok := router.SendSingle("test2", msg)

		if ok {
			t.Error("期望返回 false, 结果为 true")
		} else {
			val, err := getCounterVecValue(reg, "ws_gateway_msg_sent_total", "msg_type", "single")

			if err != nil {
				t.Fatal(err)
			}
			if val != 1 {
				t.Errorf("期望 Counter 值为 1, 得到 %v", val)
			}

			val, err = getCounterVecValue(reg, "ws_gateway_msg_send_fail_total", "reason", "single_offline")

			if err != nil {
				t.Fatal(err)
			}
			if val != 1 {
				t.Errorf("期望 Counter 值为 1, 得到 %v", val)
			}
		}
	})

	t.Run("目标通道满", func(t *testing.T) {
		reg := prometheus.NewRegistry()

		metrics.Init(reg)

		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()
		resetRouter()

		roomMgr := NewRoomManager()
		clientMgr := NewClientManager(roomMgr, 10, 10)
		router := NewMessageRouter(clientMgr, roomMgr, nil)

		go clientMgr.Init(ctx, 1*time.Second)

		client := model.NewClient("test1", "room1", nil, 1, time.Now())

		clientMgr.Register(client)
		waitForClients(t, clientMgr, 1, waitTimeout)

		msg := &model.Message{Payload: []byte("hello")}

		client.SendChan <- msg.Payload

		ok := router.SendSingle("test1", msg)

		if ok {
			t.Error("期望返回 false, 结果为 true")
		} else {
			val, err := getCounterVecValue(reg, "ws_gateway_msg_sent_total", "msg_type", "single")

			if err != nil {
				t.Fatal(err)
			}
			if val != 1 {
				t.Errorf("期望 Counter 值为 1, 得到 %v", val)
			}

			val, err = getCounterVecValue(reg, "ws_gateway_msg_send_fail_total", "reason", "single_block")

			if err != nil {
				t.Fatal(err)
			}
			if val != 1 {
				t.Errorf("期望 Counter 值为 1, 得到 %v", val)
			}

			waitForClientRemoved(t, clientMgr, "test1", waitTimeout)
		}
	})
}

func TestSendRoom(t *testing.T) {
	t.Run("房间存在且有成员", func(t *testing.T) {
		reg := prometheus.NewRegistry()

		metrics.Init(reg)

		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()
		resetRouter()

		roomMgr := NewRoomManager()
		clientMgr := NewClientManager(roomMgr, 10, 10)
		router := NewMessageRouter(clientMgr, roomMgr, nil)

		go clientMgr.Init(ctx, 1*time.Second)

		client := model.NewClient("test1", "room1", nil, 5, time.Now())

		clientMgr.Register(client)
		waitForClients(t, clientMgr, 1, waitTimeout)

		msg := &model.Message{Payload: []byte("hello room1")}
		router.SendRoom("room1", msg)

		val, err := getCounterVecValue(reg, "ws_gateway_msg_sent_total", "msg_type", "room")

		if err != nil {
			t.Fatal(err)
		}
		if val != 1 {
			t.Errorf("期望 Counter 值为 1, 得到 %v", val)
		}

		val, err = getCounterVecValue(reg, "ws_gateway_msg_sent_total", "msg_type", "single")

		if err != nil {
			t.Fatal(err)
		}
		if val != 1 {
			t.Errorf("期望 Counter 值为 1, 得到 %v", val)
		}
	})

	t.Run("房间不存在", func(t *testing.T) {
		reg := prometheus.NewRegistry()

		metrics.Init(reg)

		resetRouter()

		roomMgr := NewRoomManager()
		clientMgr := NewClientManager(roomMgr, 10, 10)
		router := NewMessageRouter(clientMgr, roomMgr, nil)

		msg := &model.Message{Payload: []byte("hello room2")}
		router.SendRoom("room2", msg)

		val, err := getCounterVecValue(reg, "ws_gateway_msg_sent_total", "msg_type", "room")

		if err != nil {
			t.Fatal(err)
		}
		if val != 1 {
			t.Errorf("期望 Counter 值为 1, 得到 %v", val)
		}
	})
}

func TestSendBroadcast(t *testing.T) {
	t.Run("有在线客户端", func(t *testing.T) {
		reg := prometheus.NewRegistry()

		metrics.Init(reg)

		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()
		resetRouter()

		roomMgr := NewRoomManager()
		clientMgr := NewClientManager(roomMgr, 10, 10)
		router := NewMessageRouter(clientMgr, roomMgr, nil)

		go clientMgr.Init(ctx, 1*time.Second)

		client1 := model.NewClient("test1", "room1", nil, 5, time.Now())
		client2 := model.NewClient("test2", "room2", nil, 5, time.Now())

		clientMgr.Register(client1)
		clientMgr.Register(client2)

		waitForClients(t, clientMgr, 2, waitTimeout)

		msg := &model.Message{Payload: []byte("hello all")}
		router.SendBroadcast(msg)

		select {
		case received := <-client1.SendChan:
			if !bytes.Equal(received, msg.Payload) {
				t.Errorf("期望 %v, 得到 %v", msg.Payload, received)
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("client1 的 SendChan 都没有收到事件")
		}

		select {
		case received := <-client2.SendChan:
			if !bytes.Equal(received, msg.Payload) {
				t.Errorf("期望 %v, 得到 %v", msg.Payload, received)
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("client2 的 SendChan 都没有收到事件")
		}

		val, err := getCounterVecValue(reg, "ws_gateway_msg_sent_total", "msg_type", "broadcast")

		if err != nil {
			t.Fatal(err)
		}
		if val != 1 {
			t.Errorf("期望 Counter 值为 1, 得到 %v", val)
		}
	})

	t.Run("无在线客户端", func(t *testing.T) {
		reg := prometheus.NewRegistry()

		metrics.Init(reg)

		resetRouter()

		roomMgr := NewRoomManager()
		clientMgr := NewClientManager(roomMgr, 10, 10)
		router := NewMessageRouter(clientMgr, roomMgr, nil)

		msg := &model.Message{Payload: []byte("hello all")}
		router.SendBroadcast(msg)

		val, err := getCounterVecValue(reg, "ws_gateway_msg_sent_total", "msg_type", "broadcast")

		if err != nil {
			t.Fatal(err)
		}
		if val != 1 {
			t.Errorf("期望 Counter 值为 1, 得到 %v", val)
		}
	})

	t.Run("存在某个在线客户端通道满", func(t *testing.T) {
		reg := prometheus.NewRegistry()

		metrics.Init(reg)

		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()
		resetRouter()

		roomMgr := NewRoomManager()
		clientMgr := NewClientManager(roomMgr, 10, 10)
		router := NewMessageRouter(clientMgr, roomMgr, nil)

		go clientMgr.Init(ctx, 1*time.Second)

		client1 := model.NewClient("test1", "room1", nil, 5, time.Now())
		client2 := model.NewClient("test2", "room2", nil, 1, time.Now())

		clientMgr.Register(client1)
		clientMgr.Register(client2)

		waitForClients(t, clientMgr, 2, waitTimeout)

		msg := &model.Message{Payload: []byte("hello all")}

		client2.SendChan <- msg.Payload
		router.SendBroadcast(msg)

		select {
		case received := <-client1.SendChan:
			if !bytes.Equal(received, msg.Payload) {
				t.Errorf("期望 %v, 得到 %v", msg.Payload, received)
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("client1 的 SendChan 没有收到事件")
		}

		val, err := getCounterVecValue(reg, "ws_gateway_msg_sent_total", "msg_type", "broadcast")

		if err != nil {
			t.Fatal(err)
		}
		if val != 1 {
			t.Errorf("期望 Counter 值为 1, 得到 %v", val)
		}

		val, err = getCounterVecValue(reg, "ws_gateway_msg_send_fail_total", "reason", "broadcast_block")

		if err != nil {
			t.Fatal(err)
		}
		if val != 1 {
			t.Errorf("期望 Counter 值为 1, 得到 %v", val)
		}

		waitForClientRemoved(t, clientMgr, "test2", waitTimeout)
	})
}
