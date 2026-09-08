package service

import (
	"testing"
	"time"
)

const testTimeout = 1 * time.Second
const waitTimeout = 200 * time.Millisecond

func waitForClients(t *testing.T, cm *ClientManager, expected int, timeout time.Duration) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if cm.GetOnlineCount() == expected {
			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Errorf("等待 %d 个客户端注册超时", expected)
}

func waitForClientRemoved(t *testing.T, cm *ClientManager, clientID string, timeout time.Duration) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if _, ok := cm.Get(clientID); !ok {
			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Errorf("等待客户端 %s 被移除超时", clientID)
}
