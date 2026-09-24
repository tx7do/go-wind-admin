package data

import (
	"context"
	"fmt"
	"testing"

	"entgo.io/ent/dialect/sql"
	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	auditV1 "go-wind-admin/api/gen/go/audit/service/v1"
	"go-wind-admin/app/admin/service/internal/data/ent"
	entOperationAuditLog "go-wind-admin/app/admin/service/internal/data/ent/operationauditlog"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newOperationAuditLogRepoSqlite 用 enttest helper 构造一个可直接做 CRUD 的
// OperationAuditLogRepo，逐字段复刻 NewOperationAuditLogRepo 的
// mapper/converter 初始化，再调用 init()。
func newOperationAuditLogRepoSqlite(t *testing.T) *OperationAuditLogRepo {
	t.Helper()
	repo := &OperationAuditLogRepo{
		entClient: enttest.NewEntClientForTest(t),
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		mapper:    mapper.NewCopierMapper[auditV1.OperationAuditLog, ent.OperationAuditLog](),
		actionTypeConverter: mapper.NewEnumTypeConverter[auditV1.OperationAuditLog_ActionType, entOperationAuditLog.Action](
			auditV1.OperationAuditLog_ActionType_name, auditV1.OperationAuditLog_ActionType_value,
		),
		sensitiveLevelConverter: mapper.NewEnumTypeConverter[auditV1.SensitiveLevel, entOperationAuditLog.SensitiveLevel](
			auditV1.SensitiveLevel_name, auditV1.SensitiveLevel_value,
		),
	}
	repo.init()
	return repo
}

// TestOperationAuditLogRepoSqlite_Create 通过 repo.Create 写入一条含全部标量字段的
// 操作审计日志，ent client 直查断言各字段按请求落库。
func TestOperationAuditLogRepoSqlite_Create(t *testing.T) {
	repo := newOperationAuditLogRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	err := repo.Create(ctx, &auditV1.CreateOperationAuditLogRequest{
		Data: &auditV1.OperationAuditLog{
			UserId:         trans.Ptr(uint32(9)),
			Username:       trans.Ptr("sqlite_op_user"),
			ResourceType:   trans.Ptr("tenant"),
			ResourceId:     trans.Ptr("77"),
			Action:         auditV1.OperationAuditLog_CREATE.Enum(),
			BeforeData:     trans.Ptr(`{"status":"OFF"}`),
			AfterData:      trans.Ptr(`{"status":"ON"}`),
			SensitiveLevel: auditV1.SensitiveLevel_INTERNAL.Enum(),
			RequestId:      trans.Ptr("req-sqlite-op-create-1"),
			TraceId:        trans.Ptr("trace-sqlite-op-create-1"),
			Success:        trans.Ptr(false),
			FailureReason:  trans.Ptr("外键冲突"),
			IpAddress:      trans.Ptr("10.0.0.4"),
			LogHash:        trans.Ptr("hash-sqlite-op-create-1"),
			Signature:      []byte{0x0d, 0x0e},
		},
	})
	require.NoError(t, err, "repo.Create 写入 SQLite 应成功")

	rows, err := repo.entClient.Client().OperationAuditLog.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1, "sys_operation_audit_logs 应有 1 条记录")
	row := rows[0]
	require.Equal(t, uint32(9), *row.UserID, "user_id 应按请求落库")
	require.Equal(t, "sqlite_op_user", *row.Username, "username 应按请求落库")
	require.Equal(t, "tenant", *row.ResourceType, "resource_type 应按请求落库")
	require.Equal(t, "77", *row.ResourceID, "resource_id 应按请求落库")
	require.NotNil(t, row.Action, "action 枚举应经 converter 落库")
	require.Equal(t, entOperationAuditLog.ActionCreate, *row.Action,
		"proto CREATE 应映射为 ent ActionCreate")
	require.Equal(t, `{"status":"OFF"}`, *row.BeforeData, "before_data 应按请求落库")
	require.Equal(t, `{"status":"ON"}`, *row.AfterData, "after_data 应按请求落库")
	require.NotNil(t, row.SensitiveLevel, "sensitive_level 枚举应经 converter 落库")
	require.Equal(t, entOperationAuditLog.SensitiveLevelInternal, *row.SensitiveLevel,
		"proto SENSITIVE_LEVEL_INTERNAL 应映射为 ent SensitiveLevelInternal")
	require.Equal(t, "req-sqlite-op-create-1", *row.RequestID, "request_id 应按请求落库")
	require.Equal(t, "trace-sqlite-op-create-1", *row.TraceID, "trace_id 应按请求落库")
	require.Equal(t, false, *row.Success, "success 应按请求落库")
	require.Equal(t, "外键冲突", *row.FailureReason, "failure_reason 应按请求落库")
	require.Equal(t, "10.0.0.4", *row.IPAddress, "ip_address 应按请求落库")
	require.Equal(t, "hash-sqlite-op-create-1", *row.LogHash, "log_hash 应按请求落库")
	require.NotNil(t, row.Signature, "signature 应按请求落库")
	require.Equal(t, []byte{0x0d, 0x0e}, *row.Signature, "signature 应按请求落库")
}

// TestOperationAuditLogRepoSqlite_EnumPairs 对 action 与 sensitive_level 两个枚举的
// 全部有效非零取值逐一建行，断言 proto → ent 与 ent → proto（List 路径）的
// 双向映射逐对成立。行在两阶段间累积，分布断言按字段过滤（nil 字段属另一阶段）。
func TestOperationAuditLogRepoSqlite_EnumPairs(t *testing.T) {
	repo := newOperationAuditLogRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	seeded := 0
	nextMarker := func() string {
		seeded++
		return fmt.Sprintf("op-enum-%d", seeded)
	}

	// —— action 枚举对 ——
	expectedEnt := map[string]int{}
	expectedProto := map[int32]int{}
	for value, name := range auditV1.OperationAuditLog_ActionType_name {
		if value == 0 {
			continue
		}
		expectedEnt[name] = 1
		expectedProto[value] = 1
		require.NoError(t, repo.Create(ctx, &auditV1.CreateOperationAuditLogRequest{
			Data: &auditV1.OperationAuditLog{
				FailureReason: trans.Ptr(nextMarker()),
				Action:        auditV1.OperationAuditLog_ActionType(value).Enum(),
			},
		}), "action=%s 建行应成功", name)
	}
	require.NotEmpty(t, expectedEnt, "应存在有效非零 action 枚举值")

	entRows, err := repo.entClient.Client().OperationAuditLog.Query().All(ctx)
	require.NoError(t, err)
	actualEnt := map[string]int{}
	for _, row := range entRows {
		if row.Action == nil {
			continue
		}
		actualEnt[string(*row.Action)]++
	}
	require.Equal(t, expectedEnt, actualEnt, "ent 侧 action 取值分布应与枚举名集合逐对一致")

	listed, err := repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	actualProto := map[int32]int{}
	for _, item := range listed.Items {
		if item.Action == nil {
			continue
		}
		actualProto[int32(*item.Action)]++
	}
	require.Equal(t, expectedProto, actualProto, "DTO 侧 action 取值分布应与 proto 枚举值集合逐对一致")

	// —— sensitive_level 枚举对 ——
	expectedEnt = map[string]int{}
	expectedProto = map[int32]int{}
	for value, name := range auditV1.SensitiveLevel_name {
		if value == 0 {
			continue
		}
		expectedEnt[name] = 1
		expectedProto[value] = 1
		require.NoError(t, repo.Create(ctx, &auditV1.CreateOperationAuditLogRequest{
			Data: &auditV1.OperationAuditLog{
				FailureReason:  trans.Ptr(nextMarker()),
				SensitiveLevel: auditV1.SensitiveLevel(value).Enum(),
			},
		}), "sensitive_level=%s 建行应成功", name)
	}
	require.NotEmpty(t, expectedEnt, "应存在有效非零 sensitive_level 枚举值")

	entRows, err = repo.entClient.Client().OperationAuditLog.Query().All(ctx)
	require.NoError(t, err)
	actualEnt = map[string]int{}
	for _, row := range entRows {
		if row.SensitiveLevel == nil {
			continue
		}
		actualEnt[string(*row.SensitiveLevel)]++
	}
	require.Equal(t, expectedEnt, actualEnt, "ent 侧 sensitive_level 取值分布应与枚举名集合逐对一致")

	listed, err = repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	actualProto = map[int32]int{}
	for _, item := range listed.Items {
		if item.SensitiveLevel == nil {
			continue
		}
		actualProto[int32(*item.SensitiveLevel)]++
	}
	require.Equal(t, expectedProto, actualProto, "DTO 侧 sensitive_level 取值分布应与 proto 枚举值集合逐对一致")
}

// TestOperationAuditLogRepoSqlite_ListFilterAndPaging 验证 List 的无过滤全量、
// resource_type 列 contains 模糊搜索、id 列等值过滤与分页语义。
func TestOperationAuditLogRepoSqlite_ListFilterAndPaging(t *testing.T) {
	repo := newOperationAuditLogRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	for i, marker := range []string{"MARKERETA", "MARKERTHETA"} {
		require.NoError(t, repo.Create(ctx, &auditV1.CreateOperationAuditLogRequest{
			Data: &auditV1.OperationAuditLog{
				ResourceType:  trans.Ptr(marker + "_resource"),
				FailureReason: trans.Ptr(fmt.Sprintf("list-op-%d", i)),
			},
		}), "写入第 %d 行应成功", i)
	}

	rows, err := repo.entClient.Client().OperationAuditLog.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	firstID := rows[0].ID
	secondID := rows[1].ID

	all, err := repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(2), all.Total, "无过滤时应统计全部 2 条")
	require.Len(t, all.Items, 2)

	filtered, err := repo.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_FilterExpr{
			FilterExpr: &paginationV1.FilterExpr{
				Type: paginationV1.ExprType_AND,
				Conditions: []*paginationV1.FilterCondition{
					{
						Field:      "resource_type",
						Op:         paginationV1.Operator_CONTAINS,
						ValueOneof: &paginationV1.FilterCondition_Value{Value: "MARKERETA"},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), filtered.Total, "contains 过滤应只统计命中行")
	require.Len(t, filtered.Items, 1, "contains 过滤应只返回命中行")
	require.Contains(t, filtered.Items[0].GetResourceType(), "MARKERETA", "命中行应为含标记的那条")

	byID, err := repo.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_FilterExpr{
			FilterExpr: &paginationV1.FilterExpr{
				Type: paginationV1.ExprType_AND,
				Conditions: []*paginationV1.FilterCondition{
					{
						Field:      "id",
						Op:         paginationV1.Operator_EQ,
						ValueOneof: &paginationV1.FilterCondition_Value{Value: fmt.Sprintf("%d", secondID)},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), byID.Total, "id 等值过滤应只统计目标行")
	require.Len(t, byID.Items, 1, "id 等值过滤应只返回目标行")
	require.Equal(t, secondID, byID.Items[0].GetId())

	page1, err := repo.List(ctx, &paginationV1.PagingRequest{
		Page:     trans.Ptr(uint32(1)),
		PageSize: trans.Ptr(uint32(1)),
	})
	require.NoError(t, err)
	require.Equal(t, uint64(2), page1.Total, "分页时 Total 应仍为全量 2")
	require.Len(t, page1.Items, 1, "pageSize=1 第一页应只含 1 行")

	page2, err := repo.List(ctx, &paginationV1.PagingRequest{
		Page:     trans.Ptr(uint32(2)),
		PageSize: trans.Ptr(uint32(1)),
	})
	require.NoError(t, err)
	require.Equal(t, uint64(2), page2.Total, "分页时 Total 应仍为全量 2")
	require.Len(t, page2.Items, 1, "pageSize=1 第二页应只含 1 行")
	require.ElementsMatch(t, []uint32{firstID, secondID},
		[]uint32{page1.Items[0].GetId(), page2.Items[0].GetId()}, "两页合并应覆盖全部行")
}

// TestOperationAuditLogRepoSqlite_Get 验证 Get 按主键的命中与未命中。
func TestOperationAuditLogRepoSqlite_Get(t *testing.T) {
	repo := newOperationAuditLogRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &auditV1.CreateOperationAuditLogRequest{
		Data: &auditV1.OperationAuditLog{
			ResourceType: trans.Ptr("menu_item"),
			ResourceId:   trans.Ptr("5"),
		},
	}))

	rows, err := repo.entClient.Client().OperationAuditLog.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := rows[0].ID

	got, err := repo.Get(ctx, &auditV1.GetOperationAuditLogRequest{
		QueryBy: &auditV1.GetOperationAuditLogRequest_Id{Id: createdID},
	})
	require.NoError(t, err, "按存在的 ID 查询应命中")
	require.Equal(t, createdID, got.GetId())
	require.Equal(t, "menu_item", got.GetResourceType(), "命中记录的 resource_type 应与写入一致")
	require.Equal(t, "5", got.GetResourceId(), "命中记录的 resource_id 应与写入一致")

	_, err = repo.Get(ctx, &auditV1.GetOperationAuditLogRequest{
		QueryBy: &auditV1.GetOperationAuditLogRequest_Id{Id: 99999},
	})
	require.Error(t, err, "查询不存在的主键应返回错误")
}

// TestOperationAuditLogRepoSqlite_CountAndIsExist 验证 Count 的带谓词/无谓词语义
// 与 IsExist 的命中/未命中。
func TestOperationAuditLogRepoSqlite_CountAndIsExist(t *testing.T) {
	repo := newOperationAuditLogRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &auditV1.CreateOperationAuditLogRequest{
		Data: &auditV1.OperationAuditLog{
			ResourceType: trans.Ptr("role"),
			ResourceId:   trans.Ptr("1"),
		},
	}))
	require.NoError(t, repo.Create(ctx, &auditV1.CreateOperationAuditLogRequest{
		Data: &auditV1.OperationAuditLog{
			ResourceType: trans.Ptr("permission"),
			ResourceId:   trans.Ptr("2"),
		},
	}))

	rows, err := repo.entClient.Client().OperationAuditLog.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	createdID := rows[0].ID

	total, err := repo.Count(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, 2, total, "无谓词 Count 应为全量 2")

	byID, err := repo.Count(ctx, []func(s *sql.Selector){entOperationAuditLog.IDEQ(createdID)})
	require.NoError(t, err)
	require.Equal(t, 1, byID, "主键等值谓词应只命中 1 行")

	byResourceType, err := repo.Count(ctx, []func(s *sql.Selector){entOperationAuditLog.ResourceTypeEQ("role")})
	require.NoError(t, err)
	require.Equal(t, 1, byResourceType, "resource_type 等值谓词应只命中 1 行")

	exist, err := repo.IsExist(ctx, createdID)
	require.NoError(t, err)
	require.True(t, exist, "存在的主键 IsExist 应为 true")
	exist, err = repo.IsExist(ctx, 99999)
	require.NoError(t, err)
	require.False(t, exist, "不存在的主键 IsExist 应为 false")
}
