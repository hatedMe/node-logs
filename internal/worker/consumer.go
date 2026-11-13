package worker

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/hatedMe/node-logs/internal/metrics"
	"github.com/hatedMe/node-logs/internal/queue"
	"github.com/hatedMe/node-logs/internal/store"
)

// StreamClient 是消费依赖的 Redis Stream 子集；*redis.Client 已实现。
type StreamClient interface {
	XGroupCreateMkStream(ctx context.Context, stream, group, start string) *redis.StatusCmd
	XReadGroup(ctx context.Context, args *redis.XReadGroupArgs) *redis.XStreamSliceCmd
	XAck(ctx context.Context, stream, group string, ids ...string) *redis.IntCmd
	XAutoClaim(ctx context.Context, args *redis.XAutoClaimArgs) *redis.XAutoClaimCmd
	XPendingExt(ctx context.Context, args *redis.XPendingExtArgs) *redis.XPendingExtCmd
}

var _ StreamClient = (*redis.Client)(nil)

// Processor 处理一批已解析事件。
type Processor interface {
	Process(ctx context.Context, events []store.RawEvent) (ProcessResult, error)
}

var _ Processor = (*BatchProcessor)(nil)

// RetryPolicy 计算失败退避。
type RetryPolicy struct {
	Base time.Duration
	Max  time.Duration
}

// Next 返回第 attempt（从 1 开始）次失败后的等待时长，指数退避并封顶。
func (p RetryPolicy) Next(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	d := p.Base
	for i := 1; i < attempt; i++ {
		d *= 2
		if d >= p.Max {
			return p.Max
		}
	}
	if d > p.Max {
		return p.Max
	}
	return d
}

// Consumer 使用消费组批量读取消息，成功后 ACK，毒丸消息转入死信。
// 处理中消息持有租约并心跳续约，其他 worker 不收割；Leases/Reaper 缺失时拒绝运行。
type Consumer struct {
	Streams StreamClient
	Stream  string
	Group   string
	Name    string

	BatchSize   int
	Block       time.Duration
	ClaimIdle   time.Duration
	MaxAttempts int64
	Retry       RetryPolicy
	// ReapIdle 是收割毒丸的空闲阈值；<=0 时跟随 ClaimIdle。
	ReapIdle time.Duration

	Processor Processor
	DLQ       *DeadLetter
	Leases    *Leases
	Reaper    *Reaper
	// Metrics 为空时不记录指标。
	Metrics *metrics.Registry
	// Now 返回当前时间；测试可注入固定时钟。
	Now func() time.Time
}

func (c *Consumer) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

func (c *Consumer) normalize() {
	if c.BatchSize <= 0 {
		c.BatchSize = 100
	}
	if c.MaxAttempts <= 0 {
		c.MaxAttempts = 5
	}
	// Block 零值会导致 XREADGROUP 非阻塞空转，默认阻塞 1 秒。
	if c.Block <= 0 {
		c.Block = time.Second
	}
	// ClaimIdle 零值会认领正在处理的消息造成重复处理，默认 1 分钟。
	if c.ClaimIdle <= 0 {
		c.ClaimIdle = time.Minute
	}
	if c.ReapIdle <= 0 {
		c.ReapIdle = c.ClaimIdle
	}
	if c.Retry.Base <= 0 {
		c.Retry.Base = 100 * time.Millisecond
	}
	if c.Retry.Max <= 0 {
		c.Retry.Max = 30 * time.Second
	}
}

// Run 启动消费循环，直到 ctx 取消；ctx 取消时返回 nil。
func (c *Consumer) Run(ctx context.Context) error {
	c.normalize()
	if err := c.Streams.XGroupCreateMkStream(ctx, c.Stream, c.Group, "0").Err(); err != nil {
		if !strings.Contains(err.Error(), "BUSYGROUP") {
			return fmt.Errorf("worker: 创建消费组失败: %w", err)
		}
	}
	fails := 0
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		if err := c.Tick(ctx); err != nil {
			fails++
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(c.Retry.Next(fails)):
			}
		} else {
			fails = 0
		}
	}
}

// Tick 执行一轮消费：收割毒丸 → 认领超时 → 读取新消息 → 加锁 → 处理 → ACK。
func (c *Consumer) Tick(ctx context.Context) error {
	c.normalize()
	if c.Leases == nil || c.Reaper == nil {
		return errors.New("worker: leases/reaper 未配置")
	}
	if err := c.reapPoisoned(ctx); err != nil {
		return err
	}

	var msgs []redis.XMessage
	claimed, _, err := c.Streams.XAutoClaim(ctx, &redis.XAutoClaimArgs{
		Stream: c.Stream, Group: c.Group, Consumer: c.Name,
		MinIdle: c.ClaimIdle, Start: "0-0", Count: int64(c.BatchSize),
	}).Result()
	if err != nil {
		return fmt.Errorf("worker: 认领超时消息失败: %w", err)
	}
	msgs = append(msgs, claimed...)

	if len(msgs) < c.BatchSize {
		streams, err := c.Streams.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group: c.Group, Consumer: c.Name,
			Streams: []string{c.Stream, ">"},
			Count:   int64(c.BatchSize - len(msgs)),
			Block:   c.Block,
		}).Result()
		if err != nil && !errors.Is(err, redis.Nil) {
			return fmt.Errorf("worker: 读取新消息失败: %w", err)
		}
		for _, s := range streams {
			msgs = append(msgs, s.Messages...)
		}
	}
	if len(msgs) == 0 {
		return nil
	}

	// 获取租约并心跳：长处理中其他 worker 不收割本批消息。
	ids := make([]string, 0, len(msgs))
	for _, m := range msgs {
		ids = append(ids, m.ID)
	}
	if err := c.Leases.Acquire(ctx, ids); err != nil {
		return fmt.Errorf("worker: 获取租约失败: %w", err)
	}
	hbCtx, hbStop := context.WithCancel(ctx)
	defer hbStop()
	go c.heartbeat(hbCtx, ids)

	var events []store.RawEvent
	var eventIDs []string
	var poisonIDs []string
	for _, m := range msgs {
		e, err := queue.ParseEvent(m.Values)
		if err != nil {
			reason := fmt.Sprintf("解析失败：%v", err)
			if err := c.DLQ.WriteRaw(ctx, m.Values, m.ID, reason); err != nil {
				return fmt.Errorf("worker: 写入死信失败: %w", err)
			}
			poisonIDs = append(poisonIDs, m.ID)
			continue
		}
		events = append(events, e)
		eventIDs = append(eventIDs, m.ID)
	}
	if len(events) > 0 {
		// 处理开始时刻记录延迟：每条事件的 now - ReceivedAt；
		// 成功失败都已等待同样久，与结果无关。
		c.observeDelay(events)
		// 处理失败不 ACK 合法事件：它们保持 pending，下一轮由认领超时取回重试。
		// 解析失败者已进死信，直接 ACK，避免其阻塞后续认领。
		if _, err := c.Processor.Process(ctx, events); err != nil {
			if len(poisonIDs) > 0 {
				_ = c.Streams.XAck(ctx, c.Stream, c.Group, poisonIDs...).Err()
			}
			c.observeConsume(false)
			return fmt.Errorf("worker: 批处理失败: %w", err)
		}
		c.observeConsume(true)
	}
	ack := make([]string, 0, len(poisonIDs)+len(eventIDs))
	ack = append(ack, poisonIDs...)
	ack = append(ack, eventIDs...)
	if len(ack) > 0 {
		if err := c.Streams.XAck(ctx, c.Stream, c.Group, ack...).Err(); err != nil {
			return fmt.Errorf("worker: 确认消息失败: %w", err)
		}
	}
	return nil
}

// observeConsume 记录消费批次结果；Metrics 为空时跳过。
func (c *Consumer) observeConsume(ok bool) {
	if c.Metrics != nil {
		c.Metrics.ObserveConsume(ok)
	}
}

// observeDelay 记录本批每条事件的消费延迟（处理开始时刻 - ReceivedAt）。
// ReceivedAt 缺失的事件跳过（不污染直方图）。
func (c *Consumer) observeDelay(events []store.RawEvent) {
	if c.Metrics == nil {
		return
	}
	now := c.now()
	for _, e := range events {
		if e.ReceivedAt.IsZero() {
			continue
		}
		c.Metrics.ObserveConsumeDelay(now.Sub(e.ReceivedAt))
	}
}

// heartbeat 在处理期间按租约 1/3 周期续约；错误忽略（租约过期只放行收割）。
func (c *Consumer) heartbeat(ctx context.Context, ids []string) {
	interval := c.Leases.ttl() / 3
	if interval <= 0 {
		interval = time.Second
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			_, _ = c.Leases.Renew(ctx, ids)
		}
	}
}

// reapPoisoned 将投递次数超限且空闲超时的消息原子收割：
// 租约检查在 Lua 内完成，他人处理中的消息跳过，自身旧毒丸正常收割。
func (c *Consumer) reapPoisoned(ctx context.Context) error {
	pending, err := c.Streams.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: c.Stream, Group: c.Group, Start: "-", End: "+", Count: int64(c.BatchSize),
	}).Result()
	if err != nil {
		return fmt.Errorf("worker: 查询 pending 失败: %w", err)
	}
	for _, p := range pending {
		if p.RetryCount < c.MaxAttempts {
			continue
		}
		if p.Idle < c.ReapIdle {
			continue
		}
		reason := fmt.Sprintf("超过最大重试 %d 次", c.MaxAttempts)
		if _, err := c.Reaper.Reap(ctx, p.ID, p.RetryCount, reason); err != nil {
			return fmt.Errorf("worker: 收割死信失败: %w", err)
		}
	}
	return nil
}
