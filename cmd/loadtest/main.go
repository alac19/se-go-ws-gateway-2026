// cmd/loadtest/main.go
package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	addr     = flag.String("addr", "ws://localhost:8080/ws", "网关 WebSocket 地址")
	numConns = flag.Int("num", 50, "并发连接数")
	duration = flag.Int("duration", 10, "等待稳定时间（秒）")
)

type result struct {
	receivedAt time.Time
}

func main() {
	flag.Parse()

	fmt.Printf("开始压测: 目标=%s, 并发数=%d\n", *addr, *numConns)

	var wg sync.WaitGroup
	results := make(chan result, *numConns*2)

	// 1. 建立连接并计时
	startTime := time.Now()

	for i := 0; i < *numConns; i++ {
		wg.Add(1)
		time.Sleep(1 * time.Millisecond)

		go func(id int) {
			// 每个客户端使用不同的 clientId
			url := fmt.Sprintf("%s?clientId=loadtest-%d&roomId=loadtest-room", *addr, id)
			conn, _, err := websocket.DefaultDialer.Dial(url, nil)

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
	resp, err := http.Post("http://localhost:8080/api/broadcast",
		"application/json",
		bytes.NewReader([]byte(`{"type":"text","data":"hello all"}`)))

	if err != nil {
		log.Fatalf("广播请求失败: %v", err)
	}

	resp.Body.Close()

	// 4. 等待所有连接处理完消息（给它们 3 秒时间）
	fmt.Println("等待消息到达...")

	time.Sleep(3 * time.Second)

	// 5. 关闭所有连接（通过 close 所有客户端的 conn，这里我们通过程序结束来清理）
	// 但为了统计完整，我们不主动关闭，让程序自然结束。
	// 实际我们通过等待时间确保了所有消息都已到达。

	// 收集所有结果
	close(results) // 关闭通道，停止接收

	var latencies []float64

	for r := range results {
		lat := r.receivedAt.Sub(broadcastStart).Seconds() * 1000 // 毫秒
		latencies = append(latencies, lat)
	}

	// 6. 统计结果
	successCount := len(latencies)

	fmt.Println("\n=== 压测结果 ===")
	fmt.Printf("总连接数: %d\n", *numConns)
	fmt.Printf("成功建立连接: %d\n", successCount)
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

	select {}
}
