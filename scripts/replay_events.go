//go:build ignore

// 回放历史事件到 Go 采集器，用于双写核对。
//
// 输入为 JSONL：每行一个 JSON 对象（POST JSON）或原始查询串（GET）。
// JSON 行示例：{"pvid":"800800","uvmark":"u1","url":"/home","d":"1762992000000","event_id":"e1"}
//
// 用法：
//
//	go run ./scripts/replay_events.go -input events.jsonl -collect-url http://127.0.0.1:8080/pingd/front/web
//	go run ./scripts/replay_events.go -input events.jsonl -dry-run | head
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	input := flag.String("input", "", "JSONL 输入文件（- 表示 stdin），必填")
	collectURL := flag.String("collect-url", "http://127.0.0.1:8080/pingd/front/web", "采集器地址")
	workers := flag.Int("workers", 8, "并发回放数")
	limit := flag.Int("limit", 0, "最多回放行数（0 表示全部）")
	timeout := flag.Duration("timeout", 5*time.Second, "单请求超时")
	dryRun := flag.Bool("dry-run", false, "只打印将发送的请求，不发送")
	flag.Parse()

	if err := run(*input, *collectURL, *workers, *limit, *timeout, *dryRun); err != nil {
		fmt.Fprintf(os.Stderr, "replay: %v\n", err)
		os.Exit(1)
	}
}

func run(input, collectURL string, workers, limit int, timeout time.Duration, dryRun bool) error {
	if input == "" {
		return fmt.Errorf("缺少 -input（用 -h 查看用法）")
	}
	if workers < 1 {
		workers = 1
	}
	var in io.Reader = os.Stdin
	if input != "-" {
		f, err := os.Open(input)
		if err != nil {
			return fmt.Errorf("打开输入失败: %w", err)
		}
		defer f.Close()
		in = f
	}

	client := &http.Client{Timeout: timeout}
	lines := make(chan string, workers*2)
	var sent, ok, failed atomic.Int64
	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			for line := range lines {
				sent.Add(1)
				if dryRun {
					fmt.Printf("%s %s\n", methodOf(line), previewURL(collectURL, line))
					ok.Add(1)
					continue
				}
				if err := send(client, collectURL, line); err != nil {
					failed.Add(1)
					fmt.Fprintf(os.Stderr, "失败: %v\n", err)
				} else {
					ok.Add(1)
				}
			}
		})
	}

	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	n := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		lines <- line
		n++
		if limit > 0 && n >= limit {
			break
		}
	}
	close(lines)
	wg.Wait()
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("读取输入失败: %w", err)
	}
	fmt.Printf("回放完成：发送=%d 成功=%d 失败=%d\n", sent.Load(), ok.Load(), failed.Load())
	if failed.Load() > 0 {
		return fmt.Errorf("%d 条回放失败", failed.Load())
	}
	return nil
}

func isJSON(line string) bool { return strings.HasPrefix(line, "{") }

func methodOf(line string) string {
	if isJSON(line) {
		return "POST"
	}
	return "GET"
}

func previewURL(base, line string) string {
	if isJSON(line) {
		return base + " <json>"
	}
	return base + "?" + line
}

func send(client *http.Client, base, line string) error {
	var req *http.Request
	var err error
	if isJSON(line) {
		req, err = http.NewRequest(http.MethodPost, base, strings.NewReader(line))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(http.MethodGet, base+"?"+line, nil)
		if err != nil {
			return err
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("采集器返回 %d（行: %.80s）", resp.StatusCode, line)
	}
	return nil
}
