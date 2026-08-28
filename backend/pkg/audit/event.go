package audit

import "context"

// AuditEvent 是单条 SQL 执行事件的采集快照，由 driver wrapper 产生、
// transport 中间件 post-handler 取出并落库。wrapper 自身不落库，避免
// 构造期循环依赖（wrapper 在 ent_client.go 构造时落库回调尚不可用）。
type AuditEvent struct {
	SqlText      string
	SqlDigest    string
	Latency      int64
	Dialect      string
	IsWrite      bool
	AffectedRows int64
	DataMasked   bool
	MaskingRules string
}

type ctxKey int

const (
	accumulatorKey ctxKey = iota
	sinkKey
)

// AccumulatorKey 返回 accumulator 的 context key，供 transport 中间件植入与读取。
func AccumulatorKey() any { return accumulatorKey }

// SinkKey 返回防递归标记的 context key，供 transport 中间件落库时植入。
func SinkKey() any { return sinkKey }

// FromContext 取出 ctx 中的 accumulator（若存在）。
func FromContext(ctx context.Context) (*[]AuditEvent, bool) {
	acc, ok := ctx.Value(accumulatorKey).(*[]AuditEvent)
	return acc, ok
}

// IsSinking 报告 ctx 是否处于审计落库阶段（防递归标记）。
func IsSinking(ctx context.Context) bool {
	v, _ := ctx.Value(sinkKey).(bool)
	return v
}
