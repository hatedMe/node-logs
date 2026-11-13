package worker

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

// 收割结果。
const (
	OutcomeReaped = "reaped"
	OutcomeLeased = "leased"
	OutcomeDup    = "dup"
	OutcomeGone   = "gone"
)

// Reaper 用 Lua 原子收割毒丸：租约检查 → 读正文 → 幂等判重 → 写死信 → 确认。
// 原子性保证并发收割者只有一个成功写死信，且处理中的消息不被误转。
type Reaper struct {
	Client LeaseClient
	DLQ    *DeadLetter
	// MainStream 为源流，Group 为消费组，Owner 为本 worker 名。
	MainStream string
	Group      string
	Owner      string
}

// reapScript 原子收割单条 pending：
// KEYS[1]=租约键，KEYS[2]=源流，KEYS[3]=幂等键；
// ARGV[1]=记录 ID，ARGV[2]=消费组，ARGV[3]=死信流，ARGV[4]=TTL 秒，
// ARGV[5]=流上限（"0" 不限），ARGV[6]=原因，ARGV[7]=投递次数，
// ARGV[8]=失败时间，ARGV[9]=本 worker 名。
const reapScript = `
local owner = redis.call('GET', KEYS[1])
if owner and owner ~= ARGV[9] then
  return 'leased'
end
local msgs = redis.call('XRANGE', KEYS[2], ARGV[1], ARGV[1])
if #msgs == 0 then
  redis.call('XACK', KEYS[2], ARGV[2], ARGV[1])
  return 'gone'
end
if redis.call('GET', KEYS[3]) then
  redis.call('XACK', KEYS[2], ARGV[2], ARGV[1])
  return 'dup'
end
local fields = msgs[1][2]
table.insert(fields, 'source_id')
table.insert(fields, ARGV[1])
table.insert(fields, 'attempts')
table.insert(fields, ARGV[7])
table.insert(fields, 'reason')
table.insert(fields, ARGV[6])
table.insert(fields, 'failed_at')
table.insert(fields, ARGV[8])
local id
if ARGV[5] == '0' then
  id = redis.call('XADD', ARGV[3], '*', unpack(fields))
else
  id = redis.call('XADD', ARGV[3], 'MAXLEN', '~', ARGV[5], '*', unpack(fields))
end
redis.call('SET', KEYS[3], id, 'EX', ARGV[4])
redis.call('XACK', KEYS[2], ARGV[2], ARGV[1])
return 'reaped'
`

// Reap 收割单条 pending，返回收割结果；Redis 错误返回 error。
func (r *Reaper) Reap(ctx context.Context, id string, attempts int64, reason string) (string, error) {
	ttl := r.DLQ.IdemTTL
	if ttl <= 0 {
		ttl = defaultIdemTTL
	}
	out, err := r.Client.Eval(ctx, reapScript,
		[]string{
			LeaseKey(r.MainStream, id),
			r.MainStream,
			idemKey(r.DLQ.SourceStream, id),
		},
		id,
		r.Group,
		r.DLQ.Stream,
		strconv.FormatInt(int64(ttl.Seconds()), 10),
		maxLenArg(r.DLQ.MaxLen),
		reason,
		strconv.FormatInt(attempts, 10),
		time.Now().UTC().Format(time.RFC3339),
		r.Owner,
	).Result()
	if err != nil {
		return "", fmt.Errorf("worker: 收割脚本失败: %w", err)
	}
	s, _ := out.(string)
	return s, nil
}
