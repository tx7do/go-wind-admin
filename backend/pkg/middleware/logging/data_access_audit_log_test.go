// data_access_audit_log_test.go —— DataAccessAuditLogMiddleware 测试。
//
// 覆盖内容：
//  1. 构造器与 Name()；
//  2. accessTypeFromSQL 表驱动：SQL 首词（去空白、大小写不敏感）映射
//     访问类型，非 DML 首词与空串归 UNSPECIFIED；
//  3. 直调守卫分支：写入函数为 nil / 无 accumulator / accumulator 为空
//     时零写入；
//  4. 直调空 Transport 的落库路径：accumulator 事件逐条转换为记录，
//     字段一一映射（SQL 文本/摘要/耗时/方言/访问类型/TableName 多表
//     斜线连接与首表数据分类/AffectedRows 负数不落库/IP 与请求 ID 的
//     nil 来源），落库后 accumulator 必须清空；
//  5. 进程内 server 路径：事件由业务闭包注入（模拟 driver wrapper 采集），
//     IP/请求 ID 从真实请求映射；
//  6. 写入函数为 nil 时跳过落库但 accumulator 仍被清空（与落库路径的
//     清空语义一致，防残留事件被后续复用）。
package logging

import (
	"context"
	nethttp "net/http"
	"testing"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	auditV1 "go-wind-admin/api/gen/go/audit/service/v1"
	"go-wind-admin/pkg/audit"
)

// TestDataAccessAuditLogMiddlewareNameAndConstructor 验证构造器落参与固定名称。
func TestDataAccessAuditLogMiddlewareNameAndConstructor(t *testing.T) {
	op := &options{}
	mw := NewDataAccessAuditLogMiddleware(op)
	require.NotNil(t, mw)
	assert.Equal(t, op, mw.op)
	assert.Equal(t, "DataAccessAuditLogMiddleware", mw.Name())
}

// TestAccessTypeFromSQL 表驱动验证 SQL 首词到访问类型的映射：
// 大小写与首部空白不敏感；非 DML 首词、空串与注释开头归 UNSPECIFIED。
func TestAccessTypeFromSQL(t *testing.T) {
	cases := []struct {
		name string
		sql  string
		want auditV1.DataAccessAuditLog_AccessType
	}{
		{"大写SELECT", "SELECT id FROM t", auditV1.DataAccessAuditLog_SELECT},
		{"小写带空白", "  select * from t", auditV1.DataAccessAuditLog_SELECT},
		{"单词SELECT", "SELECT", auditV1.DataAccessAuditLog_SELECT},
		{"小写insert", "insert into t values (1)", auditV1.DataAccessAuditLog_INSERT},
		{"UPDATE", "UPDATE t SET x=1", auditV1.DataAccessAuditLog_UPDATE},
		{"小写delete", "delete from t where id=1", auditV1.DataAccessAuditLog_DELETE},
		{"TRUNCATE归未指定", "TRUNCATE TABLE t", auditV1.DataAccessAuditLog_ACCESS_TYPE_UNSPECIFIED},
		{"空串", "", auditV1.DataAccessAuditLog_ACCESS_TYPE_UNSPECIFIED},
		{"纯空白", "   ", auditV1.DataAccessAuditLog_ACCESS_TYPE_UNSPECIFIED},
		{"注释开头", "/*comment*/ select", auditV1.DataAccessAuditLog_ACCESS_TYPE_UNSPECIFIED},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, accessTypeFromSQL(tc.sql))
		})
	}
}

// TestDataAccessAuditLogHandleDirectGuards 直调验证三重早退：
// 写入函数为 nil、ctx 无 accumulator、accumulator 为空——零写入。
func TestDataAccessAuditLogHandleDirectGuards(t *testing.T) {
	tr := &khttp.Transport{}
	events := []audit.AuditEvent{{SqlText: "SELECT 1"}}

	// 写入函数为 nil：跳过落库但 accumulator 仍被清空。
	var opNilFunc options
	mwNilFunc := NewDataAccessAuditLogMiddleware(&opNilFunc)
	mwNilFunc.Handle(
		context.WithValue(context.Background(), audit.AccumulatorKey(), &events), tr, nil, 0)
	require.Nil(t, events, "nil 写入函数路径也应清空 accumulator（防残留事件被后续复用）")

	// 写入函数存在但 ctx 无 accumulator。
	var opNoAcc options
	WithWriteDataAccessAuditLogFunc(func(ctx context.Context, d *auditV1.DataAccessAuditLog) error {
		t.Errorf("无 accumulator 时不得写入")
		return nil
	})(&opNoAcc)
	NewDataAccessAuditLogMiddleware(&opNoAcc).Handle(context.Background(), tr, nil, 0)

	// accumulator 为空切片。
	var opEmptyAcc options
	WithWriteDataAccessAuditLogFunc(func(ctx context.Context, d *auditV1.DataAccessAuditLog) error {
		t.Errorf("空 accumulator 时不得写入")
		return nil
	})(&opEmptyAcc)
	empty := []audit.AuditEvent{}
	mwEmpty := NewDataAccessAuditLogMiddleware(&opEmptyAcc)
	mwEmpty.Handle(context.WithValue(context.Background(), audit.AccumulatorKey(), &empty), tr, nil, 0)
}

// TestDataAccessAuditLogHandleDirectRecords 直调空 Transport 落库路径：
// 事件字段到记录字段的一一映射、多表 TableName 斜线连接与首表分类、
// AffectedRows 负数不落库（防 uint32 溢出成 4294967295）、
// IP/请求 ID 的 nil 来源归一空串；落库后 accumulator 必须清空。
func TestDataAccessAuditLogHandleDirectRecords(t *testing.T) {
	var recs []*auditV1.DataAccessAuditLog
	var metas []auditCallMeta
	cap := &auditCapture{}
	var op options
	WithWriteDataAccessAuditLogFunc(func(ctx context.Context, d *auditV1.DataAccessAuditLog) error {
		recs = append(recs, d)
		metas = append(metas, cap.metaOf(ctx))
		return nil
	})(&op)
	mw := NewDataAccessAuditLogMiddleware(&op)

	events := testDAEvents()
	ctx := context.WithValue(context.Background(), audit.AccumulatorKey(), &events)
	mw.Handle(ctx, &khttp.Transport{}, nil, 0)

	require.Len(t, recs, 4, "四条事件应各产生一条记录（顺序一致）")
	require.Len(t, metas, 4)

	// 事件 1：读用户表，AffectedRows 为 -1（v==nil 分支）不落库。
	r1 := recs[0]
	assert.Equal(t, auditV1.DataAccessAuditLog_SELECT, r1.GetAccessType())
	assert.Equal(t, "SELECT id, username FROM sys_users WHERE id = $1", r1.GetSqlText(), "SqlText ← 事件原文")
	assert.Equal(t, "digest-select-users", r1.GetSqlDigest())
	assert.Equal(t, uint32(12), r1.GetLatencyMs(), "LatencyMs ← 事件耗时")
	assert.Equal(t, "postgres", r1.GetDataSource(), "DataSource ← 事件方言")
	assert.True(t, r1.GetDataMasked(), "DataMasked ← 事件标记")
	assert.Equal(t, "rule-a", r1.GetMaskingRules())
	assert.Equal(t, "sys_users", r1.GetTableName(), "TableName ← SQL 提取的被访问表")
	assert.Equal(t, "USER_DATA", r1.GetDataCategory(), "DataCategory ← 首表分类")
	assert.Nil(t, r1.AffectedRows, "AffectedRows 为负时不得落库（防溢出）")
	assert.Empty(t, r1.GetIpAddress(), "nil 请求来源的 IP 归一空串")
	assert.Empty(t, r1.GetRequestId(), "nil 请求来源的请求 ID 归一空串")
	assert.True(t, r1.GetSuccess())
	assert.Zero(t, r1.GetUserId())
	assert.Empty(t, r1.GetUsername())

	// 事件 2：写角色表，AffectedRows=2 正常落库。
	r2 := recs[1]
	assert.Equal(t, auditV1.DataAccessAuditLog_INSERT, r2.GetAccessType())
	assert.Equal(t, "sys_roles", r2.GetTableName())
	assert.Equal(t, "ACCESS_CONTROL", r2.GetDataCategory())
	assert.Equal(t, uint32(2), r2.GetAffectedRows())
	assert.Equal(t, uint32(34), r2.GetLatencyMs())
	assert.Equal(t, "mysql", r2.GetDataSource())
	assert.False(t, r2.GetDataMasked())

	// 事件 3：JOIN 双表 → TableName 斜线连接、分类取首表。
	r3 := recs[2]
	assert.Equal(t, auditV1.DataAccessAuditLog_SELECT, r3.GetAccessType())
	assert.Equal(t, "sys_users/sys_plan_quotas", r3.GetTableName(), "多表按斜线连接")
	assert.Equal(t, "USER_DATA", r3.GetDataCategory(), "分类取首表")
	assert.Equal(t, uint32(0), r3.GetAffectedRows(), "0 属非负，正常落库")
	assert.Equal(t, uint32(56), r3.GetLatencyMs())

	// 事件 4：无表引用 → TableName/分类不落库。
	r4 := recs[3]
	assert.Equal(t, auditV1.DataAccessAuditLog_SELECT, r4.GetAccessType())
	assert.Nil(t, r4.TableName, "无表引用时 TableName 不落库")
	assert.Nil(t, r4.DataCategory)
	assert.Equal(t, uint32(7), r4.GetLatencyMs())

	// 落库后 accumulator 必须清空（防后续复用 ctx 重复落库）。
	require.Len(t, events, 0, "落库后 accumulator 必须清空")

	// 该中间件自身在 sinkCtx 内植入 sink 标记与系统 viewer（区别于
	// 依赖 Server 包装植入的其他审计）。
	for i := range metas {
		assert.True(t, metas[i].Sinking, "数据访问审计落库必须自带 sink 标记")
		assert.True(t, metas[i].SystemViewer, "数据访问审计落库必须切系统 viewer")
	}
}

// TestDataAccessAuditLogHandleViaServer 走 server 验证事件注入与
// 真实请求字段映射（IP ← RemoteAddr、请求 ID ← X-Request-ID 头），
// 以及空事件注入（模拟空采集）零落库。
func TestDataAccessAuditLogHandleViaServer(t *testing.T) {
	t.Run("事件落库与请求字段映射", func(t *testing.T) {
		env := newAuditServer(t)
		env.fire(nethttp.MethodPost, "/case/5", map[string]string{
			"X-Test-Operation":   "/demo.v1.GadgetService/Update",
			"X-Test-Data-Access": "all",
			"X-Request-ID":       "req-da-1",
			"Authorization":      "Bearer " + mintTestToken(t),
		}, "", "127.0.0.1:1234")

		require.Len(t, env.capture.dataAccess, 4, "四条注入事件应各落一条记录")
		for i, rec := range env.capture.dataAccess {
			assert.Equal(t, "127.0.0.1", rec.GetIpAddress(), "IpAddress ← RemoteAddr")
			assert.Equal(t, "req-da-1", rec.GetRequestId(), "RequestId ← X-Request-ID 头")
			assert.Equal(t, testDAEvents()[i].SqlText, rec.GetSqlText(), "事件按注入顺序落库")
			// 令牌身份映射到记录的 UserId/TenantId/Username。
			assert.Equal(t, uint32(42), rec.GetUserId(), "UserId ← 令牌 uid")
			assert.Equal(t, uint32(7), rec.GetTenantId(), "TenantId ← 令牌 tid")
			assert.Equal(t, "alice", rec.GetUsername(), "Username ← 令牌 sub")
		}
		// 落库后 accumulator 必须清空。
		require.NotNil(t, env.acc)
		require.Len(t, *env.acc, 0, "落库后 accumulator 必须清空")
		for _, m := range env.capture.dataAccessMeta {
			require.True(t, m.Sinking)
			require.True(t, m.SystemViewer)
		}
	})

	t.Run("空事件注入零落库", func(t *testing.T) {
		env := newAuditServer(t)
		env.fire(nethttp.MethodPost, "/case/5", map[string]string{
			"X-Test-Operation":   "/demo.v1.GadgetService/Update",
			"X-Test-Data-Access": "empty",
		}, "", "127.0.0.1:1234")
		assert.Empty(t, env.capture.dataAccess, "空 accumulator 必须零落库")
	})
}

// TestDataAccessAuditLogHandleWriteFuncNil 验证写入函数为 nil 时：
// 记录构造前的早退跳过落库，但 accumulator 仍被清空
// （与落库路径的清空语义一致，防残留事件被后续复用）。
func TestDataAccessAuditLogHandleWriteFuncNil(t *testing.T) {
	env := newAuditServer(t, WithWriteDataAccessAuditLogFunc(nil))
	env.fire(nethttp.MethodPost, "/case/5", map[string]string{
		"X-Test-Operation":   "/demo.v1.GadgetService/Update",
		"X-Test-Data-Access": "all",
		"Content-Type":       "application/json",
	}, `{"data":{"name":"x"}}`, "127.0.0.1:1234")

	assert.Empty(t, env.capture.dataAccess, "空写入函数时数据访问审计不得落库")
	assert.Len(t, env.capture.api, 1, "其余审计不受影响")
	assert.Len(t, env.capture.operation, 1)
	assert.Len(t, env.capture.permission, 1)
	// 修复后早退分支同样清空 accumulator。
	require.NotNil(t, env.acc)
	require.Empty(t, *env.acc, "早退分支也应清空 accumulator")
}
