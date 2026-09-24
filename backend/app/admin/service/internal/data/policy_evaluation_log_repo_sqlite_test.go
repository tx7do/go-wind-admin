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

	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
	"go-wind-admin/app/admin/service/internal/data/ent"
	entPolicyEvaluationLog "go-wind-admin/app/admin/service/internal/data/ent/policyevaluationlog"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newPolicyEvaluationLogRepoSqlite 用 enttest helper 构造一个可直接做 CRUD 的
// PolicyEvaluationLogRepo，逐字段复刻 NewPolicyEvaluationLogRepo 的初始化，再调用 init()。
func newPolicyEvaluationLogRepoSqlite(t *testing.T) *PolicyEvaluationLogRepo {
	t.Helper()
	repo := &PolicyEvaluationLogRepo{
		entClient: enttest.NewEntClientForTest(t),
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		mapper:    mapper.NewCopierMapper[permissionV1.PolicyEvaluationLog, ent.PolicyEvaluationLog](),
	}
	repo.init()
	return repo
}

// TestPolicyEvaluationLogRepoSqlite_Create 通过 repo.Create 写入一条含全部标量字段的
// 策略评估日志，ent client 直查断言各字段按请求落库。
func TestPolicyEvaluationLogRepoSqlite_Create(t *testing.T) {
	repo := newPolicyEvaluationLogRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	err := repo.Create(ctx, &permissionV1.CreatePolicyEvaluationLogRequest{
		Data: &permissionV1.PolicyEvaluationLog{
			TenantId:          trans.Ptr(uint32(3)),
			UserId:            trans.Ptr(uint32(12)),
			MembershipId:      trans.Ptr(uint32(13)),
			PermissionId:      trans.Ptr(uint32(14)),
			PolicyId:          trans.Ptr(uint32(15)),
			RequestPath:       trans.Ptr("/sqlite/policy-eval/create"),
			RequestMethod:     trans.Ptr("POST"),
			Result:            trans.Ptr(false),
			EffectDetails:     trans.Ptr(`{"deny":"role_not_bound"}`),
			ScopeSql:          trans.Ptr("tenant_id = 3"),
			IpAddress:         trans.Ptr("10.0.0.6"),
			TraceId:           trans.Ptr("trace-sqlite-pel-create-1"),
			EvaluationContext: trans.Ptr(`{"roles":["r1"]}`),
			LogHash:           trans.Ptr("hash-sqlite-pel-create-1"),
			Signature:         []byte{0x0f},
		},
	})
	require.NoError(t, err, "repo.Create 写入 SQLite 应成功")

	rows, err := repo.entClient.Client().PolicyEvaluationLog.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1, "sys_policy_evaluation_logs 应有 1 条记录")
	row := rows[0]
	require.Equal(t, uint32(3), *row.TenantID, "tenant_id 应按请求落库")
	require.Equal(t, uint32(12), *row.UserID, "user_id 应按请求落库")
	require.Equal(t, uint32(13), *row.MembershipID, "membership_id 应按请求落库")
	require.Equal(t, uint32(14), *row.PermissionID, "permission_id 应按请求落库")
	require.Equal(t, uint32(15), *row.PolicyID, "policy_id 应按请求落库")
	require.Equal(t, "/sqlite/policy-eval/create", *row.RequestPath, "request_path 应按请求落库")
	require.Equal(t, "POST", *row.RequestMethod, "request_method 应按请求落库")
	require.Equal(t, false, *row.Result, "result 应按请求落库")
	require.Equal(t, `{"deny":"role_not_bound"}`, *row.EffectDetails, "effect_details 应按请求落库")
	require.Equal(t, "tenant_id = 3", *row.ScopeSQL, "scope_sql 应按请求落库")
	require.Equal(t, "10.0.0.6", *row.IPAddress, "ip_address 应按请求落库")
	require.Equal(t, "trace-sqlite-pel-create-1", *row.TraceID, "trace_id 应按请求落库")
	require.Equal(t, `{"roles":["r1"]}`, *row.EvaluationContext, "evaluation_context 应按请求落库")
	require.Equal(t, "hash-sqlite-pel-create-1", *row.LogHash, "log_hash 应按请求落库")
	require.NotNil(t, row.Signature, "signature 应按请求落库")
	require.Equal(t, []byte{0x0f}, *row.Signature, "signature 应按请求落库")
}

// TestPolicyEvaluationLogRepoSqlite_ListFilterAndPaging 验证 List 的无过滤全量、
// request_path 列 contains 模糊搜索、id 列等值过滤与分页语义。
func TestPolicyEvaluationLogRepoSqlite_ListFilterAndPaging(t *testing.T) {
	repo := newPolicyEvaluationLogRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	for i, marker := range []string{"MARKERLAMBDA", "MARKERMU"} {
		require.NoError(t, repo.Create(ctx, &permissionV1.CreatePolicyEvaluationLogRequest{
			Data: &permissionV1.PolicyEvaluationLog{
				RequestPath:   trans.Ptr("/" + marker + "/path"),
				RequestMethod: trans.Ptr(fmt.Sprintf("list-pel-%d", i)),
			},
		}), "写入第 %d 行应成功", i)
	}

	rows, err := repo.entClient.Client().PolicyEvaluationLog.Query().All(ctx)
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
						Field:      "request_path",
						Op:         paginationV1.Operator_CONTAINS,
						ValueOneof: &paginationV1.FilterCondition_Value{Value: "MARKERLAMBDA"},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), filtered.Total, "contains 过滤应只统计命中行")
	require.Len(t, filtered.Items, 1, "contains 过滤应只返回命中行")
	require.Contains(t, filtered.Items[0].GetRequestPath(), "MARKERLAMBDA", "命中行应为含标记的那条")

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

// TestPolicyEvaluationLogRepoSqlite_Get 验证 Get 按主键的命中与未命中。
func TestPolicyEvaluationLogRepoSqlite_Get(t *testing.T) {
	repo := newPolicyEvaluationLogRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &permissionV1.CreatePolicyEvaluationLogRequest{
		Data: &permissionV1.PolicyEvaluationLog{
			RequestPath:   trans.Ptr("/sqlite/policy-eval/get"),
			RequestMethod: trans.Ptr("GET"),
		},
	}))

	rows, err := repo.entClient.Client().PolicyEvaluationLog.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := rows[0].ID

	got, err := repo.Get(ctx, &permissionV1.GetPolicyEvaluationLogRequest{
		QueryBy: &permissionV1.GetPolicyEvaluationLogRequest_Id{Id: createdID},
	})
	require.NoError(t, err, "按存在的 ID 查询应命中")
	require.Equal(t, createdID, got.GetId())
	require.Equal(t, "/sqlite/policy-eval/get", got.GetRequestPath(), "命中记录的 request_path 应与写入一致")
	require.Equal(t, "GET", got.GetRequestMethod(), "命中记录的 request_method 应与写入一致")

	_, err = repo.Get(ctx, &permissionV1.GetPolicyEvaluationLogRequest{
		QueryBy: &permissionV1.GetPolicyEvaluationLogRequest_Id{Id: 99999},
	})
	require.Error(t, err, "查询不存在的主键应返回错误")
}

// TestPolicyEvaluationLogRepoSqlite_CountAndIsExist 验证 Count 的带谓词/无谓词语义
// 与 IsExist 的命中/未命中。
func TestPolicyEvaluationLogRepoSqlite_CountAndIsExist(t *testing.T) {
	repo := newPolicyEvaluationLogRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &permissionV1.CreatePolicyEvaluationLogRequest{
		Data: &permissionV1.PolicyEvaluationLog{
			RequestPath:   trans.Ptr("/a/count"),
			RequestMethod: trans.Ptr("GET"),
		},
	}))
	require.NoError(t, repo.Create(ctx, &permissionV1.CreatePolicyEvaluationLogRequest{
		Data: &permissionV1.PolicyEvaluationLog{
			RequestPath:   trans.Ptr("/b/count"),
			RequestMethod: trans.Ptr("POST"),
		},
	}))

	rows, err := repo.entClient.Client().PolicyEvaluationLog.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	createdID := rows[0].ID

	total, err := repo.Count(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, 2, total, "无谓词 Count 应为全量 2")

	byID, err := repo.Count(ctx, []func(s *sql.Selector){entPolicyEvaluationLog.IDEQ(createdID)})
	require.NoError(t, err)
	require.Equal(t, 1, byID, "主键等值谓词应只命中 1 行")

	byMethod, err := repo.Count(ctx, []func(s *sql.Selector){entPolicyEvaluationLog.RequestMethodEQ("POST")})
	require.NoError(t, err)
	require.Equal(t, 1, byMethod, "request_method 等值谓词应只命中 1 行")

	exist, err := repo.IsExist(ctx, createdID)
	require.NoError(t, err)
	require.True(t, exist, "存在的主键 IsExist 应为 true")
	exist, err = repo.IsExist(ctx, 99999)
	require.NoError(t, err)
	require.False(t, exist, "不存在的主键 IsExist 应为 false")
}

// TestPolicyEvaluationLogRepoSqlite_ResolvePermissionPolicyByRoute 覆盖
// ResolvePermissionPolicyByRoute 反查链路的各分支：
// 空参数、路径/方法未登记、API 无权限点挂接、角色未持有权限、
// 完整链路取 eval_order 最小策略、完整链路但无策略（permissionID 保留）。
func TestPolicyEvaluationLogRepoSqlite_ResolvePermissionPolicyByRoute(t *testing.T) {
	repo := newPolicyEvaluationLogRepoSqlite(t)
	client := repo.entClient.Client()
	ctx := enttest.NewSystemViewerCtx(context.Background())

	// 评估埋点行（验证 Resolve 本身不依赖日志行存在）
	require.NoError(t, repo.Create(ctx, &permissionV1.CreatePolicyEvaluationLogRequest{
		Data: &permissionV1.PolicyEvaluationLog{
			RequestPath:   trans.Ptr("/sqlite/resolve/nop"),
			RequestMethod: trans.Ptr("GET"),
		},
	}))

	// —— 场景数据：完整链路（apiFull → permFull ← roleFull，挂两条策略）——
	apiFull, err := client.Api.Create().
		SetNillablePath(trans.Ptr("/sqlite/resolve/full")).
		SetNillableMethod(trans.Ptr("GET")).
		SetNillableOperation(trans.Ptr("sqliteResolveFull")).
		Save(ctx)
	require.NoError(t, err, "直建 api 行应成功")
	permFull, err := client.Permission.Create().
		SetName("sqlite_perm_full").
		SetCode("SPF").
		Save(ctx)
	require.NoError(t, err, "直建 permission 行应成功")
	require.NoError(t, client.PermissionApi.Create().
		SetAPIID(apiFull.ID).
		SetPermissionID(permFull.ID).
		Exec(ctx), "直建 permission_api 关联应成功")
	roleFull, err := client.Role.Create().
		SetNillableName(trans.Ptr("sqlite_role_full")).
		SetNillableCode(trans.Ptr("SQLITE_RESOLVE_ROLE_FULL")).
		Save(ctx)
	require.NoError(t, err, "直建 role 行应成功")
	require.NoError(t, client.RolePermission.Create().
		SetRoleID(roleFull.ID).
		SetPermissionID(permFull.ID).
		Exec(ctx), "直建 role_permission 关联应成功")
	policyLate, err := client.PermissionPolicy.Create().
		SetPermissionID(permFull.ID).
		SetEvalOrder(9).
		Save(ctx)
	require.NoError(t, err, "直建 eval_order=9 策略应成功")
	policyEarly, err := client.PermissionPolicy.Create().
		SetPermissionID(permFull.ID).
		SetEvalOrder(1).
		Save(ctx)
	require.NoError(t, err, "直建 eval_order=1 策略应成功")

	// —— 场景数据：链路无策略（apiNoPol → permNoPol ← roleNoPol，无策略行）——
	apiNoPol, err := client.Api.Create().
		SetNillablePath(trans.Ptr("/sqlite/resolve/nopolicy")).
		SetNillableMethod(trans.Ptr("GET")).
		SetNillableOperation(trans.Ptr("sqliteResolveNoPolicy")).
		Save(ctx)
	require.NoError(t, err)
	permNoPol, err := client.Permission.Create().
		SetName("sqlite_perm_nopol").
		SetCode("SPN").
		Save(ctx)
	require.NoError(t, err)
	require.NoError(t, client.PermissionApi.Create().
		SetAPIID(apiNoPol.ID).
		SetPermissionID(permNoPol.ID).
		Exec(ctx))
	roleNoPol, err := client.Role.Create().
		SetNillableName(trans.Ptr("sqlite_role_nopol")).
		SetNillableCode(trans.Ptr("SQLITE_RESOLVE_ROLE_NOPOL")).
		Save(ctx)
	require.NoError(t, err)
	require.NoError(t, client.RolePermission.Create().
		SetRoleID(roleNoPol.ID).
		SetPermissionID(permNoPol.ID).
		Exec(ctx))

	// —— 场景数据：API 已登记但无权限点挂接 ——
	apiOrphan, err := client.Api.Create().
		SetNillablePath(trans.Ptr("/sqlite/resolve/orphan")).
		SetNillableMethod(trans.Ptr("GET")).
		SetNillableOperation(trans.Ptr("sqliteResolveOrphan")).
		Save(ctx)
	require.NoError(t, err)

	// —— 场景数据：权限点挂接但角色未持有 ——
	apiNotHeld, err := client.Api.Create().
		SetNillablePath(trans.Ptr("/sqlite/resolve/notheld")).
		SetNillableMethod(trans.Ptr("GET")).
		SetNillableOperation(trans.Ptr("sqliteResolveNotHeld")).
		Save(ctx)
	require.NoError(t, err)
	permNotHeld, err := client.Permission.Create().
		SetName("sqlite_perm_notheld").
		SetCode("SPNH").
		Save(ctx)
	require.NoError(t, err)
	require.NoError(t, client.PermissionApi.Create().
		SetAPIID(apiNotHeld.ID).
		SetPermissionID(permNotHeld.ID).
		Exec(ctx))

	// 分支：空参数
	permID, policyID := repo.ResolvePermissionPolicyByRoute(ctx, "", "/x", "GET")
	require.Zero(t, permID, "空角色码应返回 0")
	require.Zero(t, policyID, "空角色码应返回 0")
	permID, policyID = repo.ResolvePermissionPolicyByRoute(ctx, "R", "", "GET")
	require.Zero(t, permID, "空路径应返回 0")
	require.Zero(t, policyID, "空路径应返回 0")
	permID, policyID = repo.ResolvePermissionPolicyByRoute(ctx, "R", "/x", "")
	require.Zero(t, permID, "空方法应返回 0")
	require.Zero(t, policyID, "空方法应返回 0")

	// 分支：路径/方法未登记
	permID, policyID = repo.ResolvePermissionPolicyByRoute(ctx, "SQLITE_RESOLVE_ROLE_FULL", "/no/such/path", "GET")
	require.Zero(t, permID, "未登记路径应返回 0")
	require.Zero(t, policyID, "未登记路径应返回 0")

	// 分支：API 已登记但无权限点挂接
	permID, policyID = repo.ResolvePermissionPolicyByRoute(ctx, "SQLITE_RESOLVE_ROLE_FULL", "/sqlite/resolve/orphan", "GET")
	require.Zero(t, permID, "无权限点挂接的 API 应返回 0")
	require.Zero(t, policyID, "无权限点挂接的 API 应返回 0")
	require.NotZero(t, apiOrphan.ID)

	// 分支：权限点挂接但角色未持有该权限
	permID, policyID = repo.ResolvePermissionPolicyByRoute(ctx, "SQLITE_RESOLVE_ROLE_FULL", "/sqlite/resolve/notheld", "GET")
	require.Zero(t, permID, "角色未持有权限应返回 0")
	require.Zero(t, policyID, "角色未持有权限应返回 0")
	require.NotZero(t, permNotHeld.ID)

	// 分支：角色码不存在
	permID, policyID = repo.ResolvePermissionPolicyByRoute(ctx, "NO_SUCH_ROLE", "/sqlite/resolve/full", "GET")
	require.Zero(t, permID, "角色码不存在应返回 0")
	require.Zero(t, policyID, "角色码不存在应返回 0")

	// 分支：完整链路，方法大小写归一（入参小写 get），
	// 多条策略按 eval_order 升序取首条（eval_order=1 那条，而非 9）。
	permID, policyID = repo.ResolvePermissionPolicyByRoute(ctx, "SQLITE_RESOLVE_ROLE_FULL", "/sqlite/resolve/full", "get")
	require.Equal(t, permFull.ID, permID, "完整链路应返回挂接的权限点 ID")
	require.Equal(t, policyEarly.ID, policyID, "多策略应取 eval_order 最小（1）的策略")
	require.NotEqual(t, policyLate.ID, policyID, "eval_order=9 的策略不应被选中")

	// 分支：完整链路但无策略。
	// 已知缺陷（按现状断言）：该分支生产实现为 `return 0, permissionID`——
	// 权限点 ID 被错位落到第二个返回值（policyID）、第一个返回值（permissionID）归零，
	// 与注释声明的"对应 ID 返回 0"（permissionID 保留、policyID 为 0）语义相反。
	permID, policyID = repo.ResolvePermissionPolicyByRoute(ctx, "SQLITE_RESOLVE_ROLE_NOPOL", "/sqlite/resolve/nopolicy", "GET")
	require.Zero(t, permID, "无策略链路现状：permissionID 返回 0（缺陷：应保留权限点 ID）")
	require.Equal(t, permNoPol.ID, policyID, "无策略链路现状：权限点 ID 被错位落到 policyID（缺陷：应为 0）")
}
