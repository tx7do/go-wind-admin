package data

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"

	"go-wind-admin/pkg/audit"
)

// auditDriver 包装 dialect.Driver，在 Exec/Query 前后采集 SQL 事件。
// 照 dialect.DebugDriver/DebugTx（entgo.io/ent/dialect/dialect.go:67-208）范式：
// 内嵌 Driver 转发 Close/Dialect/Tx，覆盖 Tx 返回包装后的 auditTx。
type auditDriver struct {
	dialect.Driver
}

// auditTx 包装 dialect.Tx，在事务内 Exec/Query 前后采集 SQL 事件。
// 照 DebugTx 范式：内嵌 Tx 转发 Commit/Rollback，覆盖 Exec/Query。
type auditTx struct {
	dialect.Tx
}

// collect 从 ctx 取 accumulator，append 一条事件。无 accumulator 或落库中则跳过。
func collect(ctx context.Context, ev audit.AuditEvent) {
	if audit.IsSinking(ctx) {
		return
	}
	acc, ok := audit.FromContext(ctx)
	if !ok || acc == nil {
		return
	}
	*acc = append(*acc, ev)
}

// collectMasked 对 SQL 做字面量脱敏后采集事件。sql_text 存脱敏文本；
// sql_digest 基于脱敏文本计算——同构不同参的 SQL 指纹一致，便于按语句分组。
func collectMasked(ctx context.Context, query string, latency int64, dialectName string, isWrite bool, rows int64) {
	masked := audit.MaskSQL(query)
	collect(ctx, audit.AuditEvent{
		SqlText:      masked,
		SqlDigest:    digest(masked),
		Latency:      latency,
		Dialect:      dialectName,
		IsWrite:      isWrite,
		AffectedRows: rows,
		DataMasked:   true,
		MaskingRules: audit.MaskingRules,
	})
}

func (d *auditDriver) Exec(ctx context.Context, query string, args, v any) error {
	start := time.Now()
	err := d.Driver.Exec(ctx, query, args, v)
	collectMasked(ctx, query, time.Since(start).Milliseconds(), d.Driver.Dialect(), true, affectedRows(v))
	return err
}

func (d *auditDriver) Query(ctx context.Context, query string, args, v any) error {
	start := time.Now()
	err := d.Driver.Query(ctx, query, args, v)
	collectMasked(ctx, query, time.Since(start).Milliseconds(), d.Driver.Dialect(), false, -1)
	return err
}

func (d *auditDriver) ExecContext(ctx context.Context, query string, args ...any) (entsql.Result, error) {
	start := time.Now()
	res, err := d.Driver.(interface {
		ExecContext(context.Context, string, ...any) (entsql.Result, error)
	}).ExecContext(ctx, query, args...)
	collectMasked(ctx, query, time.Since(start).Milliseconds(), d.Driver.Dialect(), true, -1)
	return res, err
}

func (d *auditDriver) QueryContext(ctx context.Context, query string, args ...any) (*entsql.Rows, error) {
	start := time.Now()
	rows, err := d.Driver.(interface {
		QueryContext(context.Context, string, ...any) (*entsql.Rows, error)
	}).QueryContext(ctx, query, args...)
	collectMasked(ctx, query, time.Since(start).Milliseconds(), d.Driver.Dialect(), false, -1)
	return rows, err
}

func (d *auditDriver) Tx(ctx context.Context) (dialect.Tx, error) {
	tx, err := d.Driver.Tx(ctx)
	if err != nil {
		return nil, err
	}
	return &auditTx{tx}, nil
}

func (d *auditTx) Exec(ctx context.Context, query string, args, v any) error {
	start := time.Now()
	err := d.Tx.Exec(ctx, query, args, v)
	collectMasked(ctx, query, time.Since(start).Milliseconds(), "tx", true, affectedRows(v))
	return err
}

func (d *auditTx) Query(ctx context.Context, query string, args, v any) error {
	start := time.Now()
	err := d.Tx.Query(ctx, query, args, v)
	collectMasked(ctx, query, time.Since(start).Milliseconds(), "tx", false, -1)
	return err
}

// digest 返回 SQL 文本的 SHA256 摘要（十六进制），用于 sql_digest 字段。
func digest(query string) string {
	h := sha256.Sum256([]byte(query))
	return hex.EncodeToString(h[:])
}

// affectedRows 尝试从 ent 传入的结果载体取 RowsAffected，取不到返回 -1。
func affectedRows(v any) int64 {
	if v == nil {
		return -1
	}
	type rowsAffected interface {
		RowsAffected() (int64, error)
	}
	if r, ok := v.(rowsAffected); ok {
		n, err := r.RowsAffected()
		if err != nil {
			return -1
		}
		return n
	}
	return -1
}

// _ 确保 auditDriver 实现 dialect.Driver 接口。
var _ dialect.Driver = (*auditDriver)(nil)
