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
	entPermissionAuditLog "go-wind-admin/app/admin/service/internal/data/ent/permissionauditlog"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newPermissionAuditLogRepoSqlite 用 enttest helper 构造一个可直接做 CRUD 的
// PermissionAuditLogRepo，逐字段复刻 NewPermissionAuditLogRepo 的
// mapper/converter 初始化，再调用 init()。
func newPermissionAuditLogRepoSqlite(t *testing.T) *PermissionAuditLogRepo {
	t.Helper()
	repo := &PermissionAuditLogRepo{
		entClient: enttest.NewEntClientForTest(t),
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		mapper:    mapper.NewCopierMapper[auditV1.PermissionAuditLog, ent.PermissionAuditLog](),
		actionTypeConverter: mapper.NewEnumTypeConverter[auditV1.PermissionAuditLog_ActionType, entPermissionAuditLog.Action](
			auditV1.PermissionAuditLog_ActionType_name, auditV1.PermissionAuditLog_ActionType_value,
		),
	}
	repo.init()
	return repo
}

// TestPermissionAuditLogRepoSqlite_Create 通过 repo.Create 写入一条含全部标量字段的
// 权限变更审计日志，ent client 直查断言各字段按请求落库。
func TestPermissionAuditLogRepoSqlite_Create(t *testing.T) {
	repo := newPermissionAuditLogRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	err := repo.Create(ctx, &auditV1.CreatePermissionAuditLogRequest{
		Data: &auditV1.PermissionAuditLog{
			OperatorId:   trans.Ptr(uint32(11)),
			OperatorName: trans.Ptr("sqlite_perm_op"),
			TargetType:   trans.Ptr("role"),
			TargetId:     trans.Ptr("3"),
			TargetName:   trans.Ptr("role-x"),
			Action:       auditV1.PermissionAuditLog_GRANT.Enum(),
			OldValue:     trans.Ptr(`{"perms":[]}`),
			NewValue:     trans.Ptr(`{"perms":[1]}`),
			IpAddress:    trans.Ptr("10.0.0.5"),
			RequestId:    trans.Ptr("req-sqlite-pal-create-1"),
			Reason:       trans.Ptr("业务需要"),
			LogHash:      trans.Ptr("hash-sqlite-pal-create-1"),
			Signature:    []byte{0x0e},
		},
	})
	require.NoError(t, err, "repo.Create 写入 SQLite 应成功")

	rows, err := repo.entClient.Client().PermissionAuditLog.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1, "sys_permission_audit_logs 应有 1 条记录")
	row := rows[0]
	require.Equal(t, uint32(11), *row.OperatorID, "operator_id 应按请求落库")
	require.Equal(t, "sqlite_perm_op", *row.OperatorName, "operator_name 应按请求落库")
	require.Equal(t, "role", *row.TargetType, "target_type 应按请求落库")
	require.Equal(t, "3", *row.TargetID, "target_id 应按请求落库")
	require.Equal(t, "role-x", *row.TargetName, "target_name 应按请求落库")
	require.NotNil(t, row.Action, "action 枚举应经 converter 落库")
	require.Equal(t, entPermissionAuditLog.ActionGrant, *row.Action,
		"proto GRANT 应映射为 ent ActionGrant")
	require.Equal(t, `{"perms":[]}`, *row.OldValue, "old_value 应按请求落库")
	require.Equal(t, `{"perms":[1]}`, *row.NewValue, "new_value 应按请求落库")
	require.Equal(t, "10.0.0.5", *row.IPAddress, "ip_address 应按请求落库")
	require.Equal(t, "req-sqlite-pal-create-1", *row.RequestID, "request_id 应按请求落库")
	require.Equal(t, "业务需要", *row.Reason, "reason 应按请求落库")
	require.Equal(t, "hash-sqlite-pal-create-1", *row.LogHash, "log_hash 应按请求落库")
	require.NotNil(t, row.Signature, "signature 应按请求落库")
	require.Equal(t, []byte{0x0e}, *row.Signature, "signature 应按请求落库")
}

// TestPermissionAuditLogRepoSqlite_ActionEnumPairs 对 action 枚举的全部有效非零
// 取值逐一建行，断言 proto → ent 与 ent → proto（List 路径）的双向映射逐对成立。
func TestPermissionAuditLogRepoSqlite_ActionEnumPairs(t *testing.T) {
	repo := newPermissionAuditLogRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	expectedEnt := map[string]int{}
	expectedProto := map[int32]int{}
	serial := 0
	for value, name := range auditV1.PermissionAuditLog_ActionType_name {
		if value == 0 {
			continue
		}
		serial++
		expectedEnt[name] = 1
		expectedProto[value] = 1
		require.NoError(t, repo.Create(ctx, &auditV1.CreatePermissionAuditLogRequest{
			Data: &auditV1.PermissionAuditLog{
				Reason: trans.Ptr(fmt.Sprintf("pal-enum-%d", serial)),
				Action: auditV1.PermissionAuditLog_ActionType(value).Enum(),
			},
		}), "action=%s 建行应成功", name)
	}
	require.NotEmpty(t, expectedEnt, "应存在有效非零枚举值")

	entRows, err := repo.entClient.Client().PermissionAuditLog.Query().All(ctx)
	require.NoError(t, err)
	actualEnt := map[string]int{}
	for _, row := range entRows {
		require.NotNil(t, row.Action, "action 不应为空")
		actualEnt[string(*row.Action)]++
	}
	require.Equal(t, expectedEnt, actualEnt, "ent 侧 action 取值分布应与枚举名集合逐对一致")

	listed, err := repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	actualProto := map[int32]int{}
	for _, item := range listed.Items {
		require.NotNil(t, item.Action, "DTO 的 action 应被 converter 回填")
		actualProto[int32(*item.Action)]++
	}
	require.Equal(t, expectedProto, actualProto, "DTO 侧 action 取值分布应与 proto 枚举值集合逐对一致")
}

// TestPermissionAuditLogRepoSqlite_ListFilterAndPaging 验证 List 的无过滤全量、
// operator_name 列 contains 模糊搜索、id 列等值过滤与分页语义。
func TestPermissionAuditLogRepoSqlite_ListFilterAndPaging(t *testing.T) {
	repo := newPermissionAuditLogRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	for i, marker := range []string{"MARKERIOTA", "MARKERKAPPA"} {
		require.NoError(t, repo.Create(ctx, &auditV1.CreatePermissionAuditLogRequest{
			Data: &auditV1.PermissionAuditLog{
				OperatorName: trans.Ptr(marker + "_op"),
				Reason:       trans.Ptr(fmt.Sprintf("list-pal-%d", i)),
			},
		}), "写入第 %d 行应成功", i)
	}

	rows, err := repo.entClient.Client().PermissionAuditLog.Query().All(ctx)
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
						Field:      "operator_name",
						Op:         paginationV1.Operator_CONTAINS,
						ValueOneof: &paginationV1.FilterCondition_Value{Value: "MARKERIOTA"},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), filtered.Total, "contains 过滤应只统计命中行")
	require.Len(t, filtered.Items, 1, "contains 过滤应只返回命中行")
	require.Contains(t, filtered.Items[0].GetOperatorName(), "MARKERIOTA", "命中行应为含标记的那条")

	byID, err := repo.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_FilterExpr{
			FilterExpr: &paginationV1.FilterExpr{
				Type: paginationV1.ExprType_AND,
				Conditions: []*paginationV1.FilterCondition{
					{
						Field:      "id",
						Op:         paginationV1.Operator_EQ,
						ValueOneof: &paginationV1.FilterCondition_Value{Value: fmt.Sprintf("%d", firstID)},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), byID.Total, "id 等值过滤应只统计目标行")
	require.Len(t, byID.Items, 1, "id 等值过滤应只返回目标行")
	require.Equal(t, firstID, byID.Items[0].GetId())

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
	require.Len(t, page2.Items, 1, "pageSize=1 第二页应只只含 1 行")
	require.ElementsMatch(t, []uint32{firstID, secondID},
		[]uint32{page1.Items[0].GetId(), page2.Items[0].GetId()}, "两页合并应覆盖全部行")
}

// TestPermissionAuditLogRepoSqlite_Get 验证 Get 按主键的命中与未命中。
func TestPermissionAuditLogRepoSqlite_Get(t *testing.T) {
	repo := newPermissionAuditLogRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &auditV1.CreatePermissionAuditLogRequest{
		Data: &auditV1.PermissionAuditLog{
			TargetType: trans.Ptr("permission"),
			TargetId:   trans.Ptr("8"),
		},
	}))

	rows, err := repo.entClient.Client().PermissionAuditLog.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := rows[0].ID

	got, err := repo.Get(ctx, &auditV1.GetPermissionAuditLogRequest{
		QueryBy: &auditV1.GetPermissionAuditLogRequest_Id{Id: createdID},
	})
	require.NoError(t, err, "按存在的 ID 查询应命中")
	require.Equal(t, createdID, got.GetId())
	require.Equal(t, "permission", got.GetTargetType(), "命中记录的 target_type 应与写入一致")
	require.Equal(t, "8", got.GetTargetId(), "命中记录的 target_id 应与写入一致")

	_, err = repo.Get(ctx, &auditV1.GetPermissionAuditLogRequest{
		QueryBy: &auditV1.GetPermissionAuditLogRequest_Id{Id: 99999},
	})
	require.Error(t, err, "查询不存在的主键应返回错误")
}

// TestPermissionAuditLogRepoSqlite_CountAndIsExist 验证 Count 的带谓词/无谓词语义
// 与 IsExist 的命中/未命中。
func TestPermissionAuditLogRepoSqlite_CountAndIsExist(t *testing.T) {
	repo := newPermissionAuditLogRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &auditV1.CreatePermissionAuditLogRequest{
		Data: &auditV1.PermissionAuditLog{
			TargetType: trans.Ptr("role"),
		},
	}))
	require.NoError(t, repo.Create(ctx, &auditV1.CreatePermissionAuditLogRequest{
		Data: &auditV1.PermissionAuditLog{
			TargetType: trans.Ptr("permission"),
		},
	}))

	rows, err := repo.entClient.Client().PermissionAuditLog.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	createdID := rows[0].ID

	total, err := repo.Count(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, 2, total, "无谓词 Count 应为全量 2")

	byID, err := repo.Count(ctx, []func(s *sql.Selector){entPermissionAuditLog.IDEQ(createdID)})
	require.NoError(t, err)
	require.Equal(t, 1, byID, "主键等值谓词应只命中 1 行")

	byTargetType, err := repo.Count(ctx, []func(s *sql.Selector){entPermissionAuditLog.TargetTypeEQ("role")})
	require.NoError(t, err)
	require.Equal(t, 1, byTargetType, "target_type 等值谓词应只命中 1 行")

	exist, err := repo.IsExist(ctx, createdID)
	require.NoError(t, err)
	require.True(t, exist, "存在的主键 IsExist 应为 true")
	exist, err = repo.IsExist(ctx, 99999)
	require.NoError(t, err)
	require.False(t, exist, "不存在的主键 IsExist 应为 false")
}
