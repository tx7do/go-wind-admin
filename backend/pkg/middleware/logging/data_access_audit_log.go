package logging

import (
	"context"
	"strings"

	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/tx7do/go-utils/trans"

	auditV1 "go-wind-admin/api/gen/go/audit/service/v1"
	"go-wind-admin/pkg/audit"
	appViewer "go-wind-admin/pkg/entgo/viewer"
)

type DataAccessAuditLogMiddleware struct {
	op *options
}

func NewDataAccessAuditLogMiddleware(op *options) *DataAccessAuditLogMiddleware {
	return &DataAccessAuditLogMiddleware{op: op}
}

func (d *DataAccessAuditLogMiddleware) Name() string {
	return "DataAccessAuditLogMiddleware"
}

// accessTypeFromSQL 取 SQL 首词映射 AccessType。
func accessTypeFromSQL(sqlText string) auditV1.DataAccessAuditLog_AccessType {
	s := strings.TrimSpace(sqlText)
	if s == "" {
		return auditV1.DataAccessAuditLog_ACCESS_TYPE_UNSPECIFIED
	}
	head := s
	if i := strings.IndexByte(s, ' '); i > 0 {
		head = s[:i]
	}
	switch strings.ToUpper(head) {
	case "SELECT":
		return auditV1.DataAccessAuditLog_SELECT
	case "INSERT":
		return auditV1.DataAccessAuditLog_INSERT
	case "UPDATE":
		return auditV1.DataAccessAuditLog_UPDATE
	case "DELETE":
		return auditV1.DataAccessAuditLog_DELETE
	default:
		return auditV1.DataAccessAuditLog_ACCESS_TYPE_UNSPECIFIED
	}
}

// Handle 在请求处理后将 wrapper 累积进 ctx 的 SQL 事件逐条落库。
func (d *DataAccessAuditLogMiddleware) Handle(ctx context.Context, htr *http.Transport, middleErr error, latencyMs int64) {
	if d.op.writeDataAccessAuditLogFunc == nil {
		return
	}
	acc, ok := audit.FromContext(ctx)
	if !ok || acc == nil || len(*acc) == 0 {
		return
	}

	clientIp := getClientRealIP(htr.Request())
	reqId := getRequestId(htr.Request())
	ut := extractAuthToken(htr)

	// 落库前植入 sink 标记，短路 wrapper 对审计行自身 INSERT 的采集。
	sinkCtx := context.WithValue(ctx, audit.SinkKey(), true)
	sinkCtx = appViewer.NewSystemViewerContext(sinkCtx)

	for _, ev := range *acc {
		rec := &auditV1.DataAccessAuditLog{}
		rec.SqlText = trans.Ptr(ev.SqlText)
		rec.SqlDigest = trans.Ptr(ev.SqlDigest)
		rec.LatencyMs = trans.Ptr(uint32(ev.Latency))
		rec.DataSource = trans.Ptr(ev.Dialect)
		rec.AccessType = trans.Ptr(accessTypeFromSQL(ev.SqlText))
		// affected_rows 仅在真实取到时落库；-1（如 v==nil 分支）转为 uint32 会溢出成 4294967295。
		if ev.AffectedRows >= 0 {
			rec.AffectedRows = trans.Ptr(uint32(ev.AffectedRows))
		}
		rec.DataMasked = trans.Ptr(ev.DataMasked)
		rec.MaskingRules = trans.Ptr(ev.MaskingRules)
		// 从脱敏 SQL 提取被访问表名（多表按 proto 语义斜线连接），首表映射数据分类。
		if tables := audit.ExtractTables(ev.SqlText); len(tables) > 0 {
			rec.TableName = trans.Ptr(strings.Join(tables, "/"))
			rec.DataCategory = trans.Ptr(audit.ClassifyTable(tables[0]))
		}
		rec.IpAddress = trans.Ptr(clientIp)
		rec.RequestId = trans.Ptr(reqId)
		rec.Success = trans.Ptr(true)
		if ut != nil {
			rec.UserId = trans.Ptr(ut.UserId)
			rec.TenantId = ut.TenantId
			rec.Username = ut.Username
		}
		_ = d.op.writeDataAccessAuditLogFunc(sinkCtx, rec)
	}
	// 清空 accumulator，避免后续复用 ctx 时重复落库。
	*acc = nil
}
