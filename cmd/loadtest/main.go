// cmd/loadtest/main.go
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	addr     = flag.String("addr", "ws://localhost:8080/ws", "网关 WebSocket 地址")
	numConns = flag.Int("num", 50, "并发连接数")
	duration = flag.Int("duration", 10, "等待稳定时间（秒）")
	user     = flag.String("user", "zhangsan", "登录账号, 用于换取 JWT")
	pass     = flag.String("pass", "123456", "登录口令")
	token    = flag.String("token", "", "直接指定 JWT, 留空则自动调用登录接口获取")
)

type result struct {
	receivedAt time.Time
}

// httpBaseFromWS 由 WebSocket 地址推导 HTTP 基地址: ws://host:port/ws → http://host:port
func httpBaseFromWS(wsAddr string) string {
	base := strings.TrimSuffix(wsAddr, "/ws")
	base = strings.Replace(base, "ws://", "http://", 1)

	return strings.Replace(base, "wss://", "https://", 1)
}

// loginAndGetToken 调用登录接口换取 JWT, 网关的 /ws 与 /api 接口均需携带该令牌
func loginAndGetToken(base, userName, password string) (string, error) {
	body, err := json.Marshal(map[string]string{"username": userName, "password": password})
	if err != nil {
		return "", err
	}

	resp, err := http.Post(base+"/api/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var loginResp struct {
		Code int `json:"code"`
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return "", err
	}

	if loginResp.Code != 0 || loginResp.Data.Token == "" {
		return "", fmt.Errorf("登录失败, 状态码: %d", resp.StatusCode)
	}

	return loginResp.Data.Token, nil
}

func main() {
	flag.Parse()

	fmt.Printf("开始压测: 目标=%s, 并发数=%d\n", *addr, *numConns)

	base := httpBaseFromWS(*addr)

	// 压测连接需要携带令牌: 未指定 -token 时自动登录获取
	jwtToken := *token

	if jwtToken == "" {
		got, err := loginAndGetToken(base, *user, *pass)
		if err != nil {
			log.Fatalf("获取令牌失败: %v (可用 -token 直接指定)", err)
		}

		jwtToken = got

		fmt.Printf("已获取 JWT, 长度=%d\n", len(jwtToken))
	}

	var wg sync.WaitGroup
	results := make(chan result, *numConns*2)

	// 1. 建立连接并计时
	startTime := time.Now()

	for i := 0; i < *numConns; i++ {
		wg.Add(1)
		// 人为延迟 1 ms 避免瞬间连接数过大导致 Windows Accept 队列溢出
		time.Sleep(1 * time.Millisecond)

		go func(id int) {
			// clientId 需满足身份绑定规则(账号名 或 账号名-后缀), 这里用账号名加独立后缀
			wsURL := fmt.Sprintf("%s?clientId=%s-loadtest-%d&roomId=loadtest-room&token=%s",
				*addr, *user, id, url.QueryEscape(jwtToken))
			conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)

			// 等待连接建立成功
			wg.Done()

			if err != nil {
				log.Printf("连接 %d 失败: %v", id, err)
				return
			}

			defer conn.Close()

			// ticker := time.NewTicker(2 * time.Second)

			// go func() {
			// 	for {
			// 		select {
			// 		case <-ticker.C:
			// 			err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"ping"}`))

			// 			if err != nil {
			// 				return
			// 			}
			// 		}
			// 	}
			// }()

			// 读取服务端推送的消息（广播消息会通过这个通道到达）
			for {
				_, msg, err := conn.ReadMessage()

				if err != nil {
					// 连接正常关闭或出错
					return
				}

				// 收到消息，记录时间
				if bytes.Contains(msg, []byte("hello all")) {
					results <- result{receivedAt: time.Now()}
				}
			}
		}(i)
	}

	wg.Wait()
	endtime := time.Now()

	// 2. 等待连接稳定
	fmt.Printf("等待 %d 秒，让连接稳定...\n", *duration)

	time.Sleep(time.Duration(*duration) * time.Second)

	// 3. 触发广播
	fmt.Println("触发全服广播...")

	broadcastStart := time.Now()
	req, err := http.NewRequest(http.MethodPost, base+"/api/broadcast",
		bytes.NewReader([]byte(`{"type":"text","data":"hello all"}`)))

	if err != nil {
		log.Fatalf("构造广播请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+jwtToken)

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		log.Fatalf("广播请求失败: %v", err)
	}

	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("广播请求返回非 200: %d (令牌是否有效, 或触发了 IP 限流)", resp.StatusCode)
	}

	// 4. 等待所有连接处理完消息（给它们 3 秒时间）
	fmt.Println("等待消息到达...")

	time.Sleep(3 * time.Second)

	// 5. 收集所有结果
	close(results) // 关闭通道，停止接收

	var latencies []float64

	for r := range results {
		lat := r.receivedAt.Sub(broadcastStart).Seconds() * 1000 // 毫秒
		latencies = append(latencies, lat)
	}

	// 6. 统计并输出压测结果
	successCount := len(latencies)

	fmt.Println("\n=== 压测结果 ===")
	fmt.Printf("总连接数: %d\n", *numConns)
	fmt.Printf("收到广播的客户端: %d\n", successCount)
	fmt.Printf("连接建立时间: %.2f s\n", endtime.Sub(startTime).Seconds()-float64(*numConns)*0.001)
	fmt.Printf("消息到达率: %.2f%%\n", float64(successCount)/float64(*numConns)*100)

	if successCount > 0 {
		sort.Float64s(latencies)

		p99Index := int(float64(successCount) * 0.99)

		if p99Index >= successCount {
			p99Index = successCount - 1
		}

		var sum float64

		for _, v := range latencies {
			sum += v
		}

		avg := sum / float64(successCount)

		fmt.Printf("P99 延迟: %.2f ms\n", latencies[p99Index])
		fmt.Printf("最大延迟: %.2f ms\n", latencies[successCount-1])
		fmt.Printf("平均延迟: %.2f ms\n", avg)
	}

	// 永久阻塞，保持连接活跃用于长稳测试或 pprof 采集
	fmt.Println("\n连接保持活跃(用于长稳测试或 pprof 采样), 按 Ctrl+C 结束")

	select {}
}
