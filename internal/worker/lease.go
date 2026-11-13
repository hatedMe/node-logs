package worker

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// LeaseClient 是租约依赖的 Redis 子集；*redis.Client 已实现。
type LeaseClient interface {
	Eval(ctx context.Context, script string, keys []string, args ...any) *redis.Cmd
}

var _ LeaseClient = (*redis.Client)(nil)

// defaultLeaseTTL 是租约默认存活时间。
const defaultLeaseTTL = 30 * time.Second

// Leases 管理处理中消息的租约：持有者存活则消息不被收割，
// 持有者崩溃后租约过期，消息可被认领或收割。
type Leases struct {
	Client LeaseClient
	Stream string
	Owner  string
	TTL    time.Duration
}

func (l *Leases) ttl() time.Duration {
	if l.TTL <= 0 {
		return defaultLeaseTTL
	}
	return l.TTL
}

// LeaseKey 返回消息租约键。
func LeaseKey(stream, id string) string {
	return "lease:" + stream + ":" + id
}

// leaseAcquireScript 无条件获取一批租约（投递即属主转移）。
const leaseAcquireScript = `
for _, k in ipairs(KEYS) do
  redis.call('SET', k, ARGV[1], 'EX', ARGV[2])
end
return #KEYS
`

// leaseRenewScript 仅当属主匹配时续约，返回续约个数。
const leaseRenewScript = `
local n = 0
for _, k in ipairs(KEYS) do
  if redis.call('GET', k) == ARGV[1] then
    redis.call('SET', k, ARGV[1], 'EX', ARGV[2])
    n = n + 1
  end
end
return n
`

// Acquire 获取一批消息的租约。
func (l *Leases) Acquire(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	keys := make([]string, 0, len(ids))
	for _, id := range ids {
		keys = append(keys, LeaseKey(l.Stream, id))
	}
	return l.Client.Eval(ctx, leaseAcquireScript, keys,
		l.Owner, strconv.FormatInt(int64(l.ttl().Seconds()), 10)).Err()
}

// Renew 续约属主为自己的租约，返回续约个数。
func (l *Leases) Renew(ctx context.Context, ids []string) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	keys := make([]string, 0, len(ids))
	for _, id := range ids {
		keys = append(keys, LeaseKey(l.Stream, id))
	}
	n, err := l.Client.Eval(ctx, leaseRenewScript, keys,
		l.Owner, strconv.FormatInt(int64(l.ttl().Seconds()), 10)).Result()
	if err != nil {
		return 0, err
	}
	switch v := n.(type) {
	case int64:
		return int(v), nil
	default:
		return 0, nil
	}
}
