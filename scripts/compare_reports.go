//go:build ignore

// 对比 Go 报表与基线，用于双写核对与切流验收。
//
// Node 侧没有报表 HTTP 接口：从 Mongo 按应用+日期汇总后存为 JSON 固件，
// 格式 {"pv":N,"uv":N,"ip_count":N,"sessions":N}（缺失的指标跳过对比）。
// 也支持 Go-vs-Go 双基线模式（-baseline-base）与 Go 侧离线模式（-go-json）。
//
// 用法：
//
//	go run ./scripts/compare_reports.go -go-base http://127.0.0.1:8080 -app 800800 \
//	  -from 2026-11-12 -to 2026-11-12 -go-token $JWT -node-json baseline.json
//	go run ./scripts/compare_reports.go -go-json go.json -node-json node.json
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type numbers struct {
	PV       *int64 `json:"pv"`
	UV       *int64 `json:"uv"`
	IPCount  *int64 `json:"ip_count"`
	Sessions *int64 `json:"sessions"`
}

type envelope struct {
	Data numbers `json:"data"`
}

func main() {
	goBase := flag.String("go-base", "", "Go 服务地址（与 -go-json 二选一）")
	goToken := flag.String("go-token", "", "Go 报表 JWT（-go-base 模式必填）")
	goJSON := flag.String("go-json", "", "Go 侧离线响应文件（包络或裸 data 均可）")
	app := flag.String("app", "", "应用 ID（-go-base/-baseline-base 模式必填）")
	from := flag.String("from", "", "开始日期 YYYY-MM-DD")
	to := flag.String("to", "", "结束日期 YYYY-MM-DD")
	nodeJSON := flag.String("node-json", "", "Node 基线 JSON 文件（与 -baseline-base 二选一）")
	baselineBase := flag.String("baseline-base", "", "基线 Go 服务地址（Go-vs-Go 模式）")
	baselineToken := flag.String("baseline-token", "", "基线服务 JWT")
	tolerance := flag.Float64("tolerance", 0, "相对容差，如 0.01 表示 1%")
	timeout := flag.Duration("timeout", 10*time.Second, "单请求超时")
	flag.Parse()

	if err := run(config{
		goBase: *goBase, goToken: *goToken, goJSON: *goJSON,
		app: *app, from: *from, to: *to,
		nodeJSON: *nodeJSON, baselineBase: *baselineBase, baselineToken: *baselineToken,
		tolerance: *tolerance, timeout: *timeout,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "compare: %v\n", err)
		os.Exit(exitCode(err))
	}
}

type config struct {
	goBase, goToken, goJSON     string
	app, from, to               string
	nodeJSON                    string
	baselineBase, baselineToken string
	tolerance                   float64
	timeout                     time.Duration
}

type mismatchError struct{ msg string }

func (e *mismatchError) Error() string { return e.msg }

func exitCode(err error) int {
	var mm *mismatchError
	if errors.As(err, &mm) {
		return 1
	}
	return 2
}

func run(c config) error {
	got, err := loadGo(c)
	if err != nil {
		return err
	}
	want, err := loadWant(c)
	if err != nil {
		return err
	}
	rows := [][3]string{{"指标", "Go", "基线"}}
	mismatched := false
	compare := func(name string, g, w *int64) {
		gs, ws, mark := "-", "-", "skip"
		if g != nil {
			gs = fmt.Sprint(*g)
		}
		if w == nil {
			rows = append(rows, [3]string{name, gs, ws + " (" + mark + ")"})
			return
		}
		ws = fmt.Sprint(*w)
		switch {
		case g == nil:
			mark = "MISMATCH(go 缺失)"
			mismatched = true
		case diffOK(*g, *w, c.tolerance):
			mark = "ok"
		default:
			mark = "MISMATCH"
			mismatched = true
		}
		rows = append(rows, [3]string{name, gs, ws + " (" + mark + ")"})
	}
	compare("pv", got.PV, want.PV)
	compare("uv", got.UV, want.UV)
	compare("ip_count", got.IPCount, want.IPCount)
	compare("sessions", got.Sessions, want.Sessions)
	for _, r := range rows {
		fmt.Printf("%-10s %-12s %s\n", r[0], r[1], r[2])
	}
	if mismatched {
		return &mismatchError{msg: "指标不一致"}
	}
	fmt.Println("一致。")
	return nil
}

func diffOK(got, want int64, tolerance float64) bool {
	diff := got - want
	if diff < 0 {
		diff = -diff
	}
	allowed := int64(tolerance * float64(max(want, 1)))
	return diff <= allowed
}

func loadGo(c config) (numbers, error) {
	if c.goJSON != "" {
		return readNumbersFile(c.goJSON)
	}
	if c.goBase == "" {
		return numbers{}, fmt.Errorf("缺少 Go 侧来源：-go-base 或 -go-json 二选一")
	}
	if c.goToken == "" || c.app == "" || c.from == "" || c.to == "" {
		return numbers{}, fmt.Errorf("-go-base 模式需要 -go-token/-app/-from/-to")
	}
	return fetchOverview(c.goBase, c.goToken, c.app, c.from, c.to, c.timeout)
}

func loadWant(c config) (numbers, error) {
	if c.nodeJSON != "" {
		raw, err := os.ReadFile(c.nodeJSON)
		if err != nil {
			return numbers{}, fmt.Errorf("读取基线文件失败: %w", err)
		}
		var n numbers
		if err := json.Unmarshal(raw, &n); err != nil {
			return numbers{}, fmt.Errorf("基线文件不是合法 JSON: %w", err)
		}
		return n, nil
	}
	if c.baselineBase == "" {
		return numbers{}, fmt.Errorf("缺少基线来源：-node-json 或 -baseline-base 二选一")
	}
	if c.app == "" || c.from == "" || c.to == "" {
		return numbers{}, fmt.Errorf("-baseline-base 模式需要 -app/-from/-to")
	}
	return fetchOverview(c.baselineBase, c.baselineToken, c.app, c.from, c.to, c.timeout)
}

func readNumbersFile(path string) (numbers, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return numbers{}, fmt.Errorf("读取 Go 侧文件失败: %w", err)
	}
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return numbers{}, fmt.Errorf("Go 侧文件不是合法 JSON: %w", err)
	}
	// 显式 "data":null 必须报错，不能当成"全部跳过"误报一致。
	var probe map[string]any
	if err := json.Unmarshal(raw, &probe); err == nil {
		if v, hasData := probe["data"]; hasData && v == nil {
			return numbers{}, fmt.Errorf("Go 侧文件 data 为 null")
		}
	}
	// 裸 data 对象同样可解析（envelope.Data 为零值时再试一次裸解析）。
	if env.Data.PV == nil && env.Data.UV == nil && env.Data.IPCount == nil && env.Data.Sessions == nil {
		var n numbers
		if err := json.Unmarshal(raw, &n); err != nil {
			return numbers{}, fmt.Errorf("Go 侧文件不是合法 JSON: %w", err)
		}
		return n, nil
	}
	return env.Data, nil
}

func fetchOverview(base, token, app, from, to string, timeout time.Duration) (numbers, error) {
	u, err := url.Parse(strings.TrimRight(base, "/") + "/api/v1/reports/overview")
	if err != nil {
		return numbers{}, fmt.Errorf("非法服务地址: %w", err)
	}
	q := u.Query()
	q.Set("app_id", app)
	q.Set("from", from)
	q.Set("to", to)
	u.RawQuery = q.Encode()
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return numbers{}, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return numbers{}, fmt.Errorf("请求报表失败: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return numbers{}, fmt.Errorf("读取报表失败: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return numbers{}, fmt.Errorf("报表接口返回 %d：%.200s", resp.StatusCode, body)
	}
	var env envelope
	// Go 报表包络为 {code,message,data}；data 缺失则报错而非静默零值。
	var raw struct {
		Data *numbers `json:"data"`
	}
	if err := json.Unmarshal(body, &raw); err != nil || raw.Data == nil {
		return numbers{}, fmt.Errorf("报表响应无 data：%.200s", body)
	}
	env.Data = *raw.Data
	return env.Data, nil
}
