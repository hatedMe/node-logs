// Package aggregation 实现统计语义与聚合落库。
//
// 统计规则：
//   - PV：每个唯一页面访问事件计 1。
//   - UV：同一应用 + 自然日（UTC）+ visitor_id 只计 1。
//   - 独立 IP：同一应用 + 自然日 + ip_hash 只计 1。
//   - 会话：同一访客在超时窗口（默认 30 分钟）内的连续访问归为一个会话，
//     计入会话开始日；跨天会话的跳出回扣同样归因到开始日。
//   - 跳出率：单事件会话数 / 总会话数；无会话时报表返回 null。
package aggregation

// 设备维度取值，与 daily_device_stats.dimension 一致。
const (
	DimensionBrowser = "browser"
	DimensionSystem  = "system"
	DimensionDevice  = "device"
)
