package worker

import (
	"context"
	"maps"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/hatedMe/node-logs/internal/metrics"
)

// DLQClient 是死信写入依赖的 Redis 子集；*redis.Client 已实现。
type DLQClient interface {
	Eval(ctx context.Context, script string, keys []string, args ...any) *redis.Cmd
}

var _ DLQClient = (*redis.Client)(nil)

// defaultIdemTTL 是死信幂等键的默认存活时间（30 天，与原始事件保留周期对齐）。
// 重试发生在秒级，30 天窗口远超任何现实重试周期；极晚重试的重复可用 source_id 去重。
const defaultIdemTTL = 30 * 24 * time.Hour

// DeadLetter 将无法处理的消息写入死信流，保留原始字段与失败原因。
// 同一 source_id 重复写入只产生一条死信（Lua 原子保证并发安全）。
// 幂等键按源 Stream 作用域划分，多源共享 DLQ 时不会碰撞。
type DeadLetter struct {
	Client DLQClient
	// Stream 为死信流；SourceStream 为来源流（用于幂等键作用域）。
	Stream       string
	SourceStream string
	// MaxLen 为死信流近似上限；<=0 表示不限制。
	MaxLen int64
	// IdemTTL 为幂等键存活时间；<=0 时默认为 30 天。
	IdemTTL time.Duration
	// Metrics 为空时不记录指标。
	Metrics *metrics.Registry
}

func (d *DeadLetter) idemTTL() time.Duration {
	if d.IdemTTL <= 0 {
		return defaultIdemTTL
	}
	return d.IdemTTL
}

// maxLenArg 将 MaxLen 转为 Lua 参数；<=0 表示不限长，绝不传递负值。
func maxLenArg(m int64) string {
	if m <= 0 {
		return "0"
	}
	return strconv.FormatInt(m, 10)
}

// idemKey 返回死信幂等键（按源流作用域）。
func idemKey(sourceStream, sourceID string) string {
	return "dlq:idem:" + sourceStream + ":" + sourceID
}

// idemWriteScript 原子执行“判重-XADD-记键”：
// KEYS[1]=幂等键；ARGV[1]=死信流，ARGV[2]=键 TTL 秒，ARGV[3]=流上限（"0" 不限），ARGV[4+]=字段。
// 已存在返回旧记录 ID，否则写入并返回新记录 ID。
const idemWriteScript = `
local existing = redis.call('GET', KEYS[1])
if existing then
  return existing
end
local id
if ARGV[3] == '0' then
  id = redis.call('XADD', ARGV[1], '*', unpack(ARGV, 4))
else
  id = redis.call('XADD', ARGV[1], 'MAXLEN', '~', ARGV[3], '*', unpack(ARGV, 4))
end
redis.call('SET', KEYS[1], id, 'EX', ARGV[2])
return id
`

// WriteRaw 写入一条死信；同一 sourceID 重复调用只保留首次写入。
// sourceID 为源 Stream 记录 ID。失败只返回错误从不静默丢弃，调用方重试安全。
func (d *DeadLetter) WriteRaw(ctx context.Context, values map[string]any, sourceID, reason string) error {
	fields := make(map[string]any, len(values)+3)
	maps.Copy(fields, values)
	fields["source_id"] = sourceID
	fields["reason"] = reason
	fields["failed_at"] = time.Now().UTC().Format(time.RFC3339)

	args := make([]any, 0, 3+2*len(fields))
	args = append(args, d.Stream, strconv.FormatInt(int64(d.idemTTL().Seconds()), 10), maxLenArg(d.MaxLen))
	for k, v := range fields {
		args = append(args, k, v)
	}
	key := idemKey(d.SourceStream, sourceID)
	if err := d.Client.Eval(ctx, idemWriteScript, []string{key}, args...).Err(); err != nil {
		return err
	}
	if d.Metrics != nil {
		d.Metrics.ObserveDLQ()
	}
	return nil
}
