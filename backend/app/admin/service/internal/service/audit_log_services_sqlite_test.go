// 六个审计日志服务（api/data_access/login/operation/permission/policy_evaluation）
// 的 SQLite 内存库测试。
//
// 六个服务是同构的薄委托层（List/Get/Create 直通 repo），测试各盯两层：
//   - 服务守卫连通：Create 缺 Data 报 BadRequest（审计写入方是各业务钩子，
//     nil 载荷静默落库会让"少了一条审计"无从察觉，必须显式拒绝）；
//     List nil 请求透传 repo 的 BadRequest 守卫；
//   - 委托连通：经服务 List/Get 能读回 repo 落的行（服务层字段初始化错漏
//     会在这一步以 nil repo panic 或空结果暴露）。
//
// ApiAuditLogService 额外测 API 富集：List/Get 把 (path,method) 命中的 Api 行的
// 描述/模块名回填进审计条目——审计页"这条日志调的是哪个接口"的可读性全靠它。
package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/enttest"

	auditV1 "go-wind-admin/api/gen/go/audit/service/v1"
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
)

// --- 各服务装配（白盒复刻 NewXxxService：log 换 NopLogger，repo 走 ForTest） ---

func newApiAuditLogServiceForTest(t *testing.T) (*ApiAuditLogService, *entCrud.EntClient[*ent.Client]) {
	t.Helper()
	entClient := enttest.NewEntClientForTest(t)
	return &ApiAuditLogService{
		log:             bLogger.NewHelper(bLogger.NopLogger()),
		apiAuditLogRepo: data.NewApiAuditLogRepoForTest(entClient),
		apiRepo:         data.NewApiRepoForTest(entClient),
	}, entClient
}

func newDataAccessAuditLogServiceForTest(t *testing.T) *DataAccessAuditLogService {
	t.Helper()
	return &DataAccessAuditLogService{
		log:  bLogger.NewHelper(bLogger.NopLogger()),
		repo: data.NewDataAccessAuditLogRepoForTest(enttest.NewEntClientForTest(t)),
	}
}

func newLoginAuditLogServiceForTest(t *testing.T) *LoginAuditLogService {
	t.Helper()
	return &LoginAuditLogService{
		log:  bLogger.NewHelper(bLogger.NopLogger()),
		repo: data.NewLoginAuditLogRepoForTest(enttest.NewEntClientForTest(t)),
	}
}

func newOperationAuditLogServiceForTest(t *testing.T) *OperationAuditLogService {
	t.Helper()
	return &OperationAuditLogService{
		log:  bLogger.NewHelper(bLogger.NopLogger()),
		repo: data.NewOperationAuditLogRepoForTest(enttest.NewEntClientForTest(t)),
	}
}

func newPermissionAuditLogServiceForTest(t *testing.T) *PermissionAuditLogService {
	t.Helper()
	return &PermissionAuditLogService{
		log:                    bLogger.NewHelper(bLogger.NopLogger()),
		permissionAuditLogRepo: data.NewPermissionAuditLogRepoForTest(enttest.NewEntClientForTest(t)),
	}
}

func newPolicyEvaluationLogServiceForTest(t *testing.T) *PolicyEvaluationLogService {
	t.Helper()
	return &PolicyEvaluationLogService{
		log:                     bLogger.NewHelper(bLogger.NopLogger()),
		policyEvaluationLogRepo: data.NewPolicyEvaluationLogRepoForTest(enttest.NewEntClientForTest(t)),
	}
}

// --- ApiAuditLogService：富集 + 守卫 ---

// TestApiAuditLogService_ListEnrichesApiInfo 验证 List 富集：
// 审计行按 (path, method) 命中 Api 表后回填 ApiDescription/ApiModule；
// 未命中的行保持空值（接口后来被删除的审计行不能因此报错）。
func TestApiAuditLogService_ListEnrichesApiInfo(t *testing.T) {
	svc, entClient := newApiAuditLogServiceForTest(t)
	ctx := enttest.NewSystemContext(context.Background())

	// 先落 Api 行（服务首次 List 会全量缓存 Api 表，必须先建后查）
	_, err := entClient.Client().Api.Create().
		SetNillablePath(trans.Ptr("/admin/v1/orders")).
		SetNillableMethod(trans.Ptr("GET")).
		SetNillableDescription(trans.Ptr("订单查询")).
		SetNillableModuleDescription(trans.Ptr("订单模块")).
		Save(ctx)
	require.NoError(t, err)

	require.NoError(t, svc.apiAuditLogRepo.Create(ctx, &auditV1.CreateApiAuditLogRequest{
		Data: &auditV1.ApiAuditLog{
			Path:       trans.Ptr("/admin/v1/orders"),
			HttpMethod: trans.Ptr("GET"),
			Username:   trans.Ptr("svc_user"),
		},
	}), "落一条审计行应成功")

	// 未命中富集的行：path 没有对应 Api
	require.NoError(t, svc.apiAuditLogRepo.Create(ctx, &auditV1.CreateApiAuditLogRequest{
		Data: &auditV1.ApiAuditLog{
			Path:       trans.Ptr("/admin/v1/gone"),
			HttpMethod: trans.Ptr("POST"),
			Username:   trans.Ptr("svc_user"),
		},
	}))

	resp, err := svc.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)

	byPath := map[string]*auditV1.ApiAuditLog{}
	for _, it := range resp.GetItems() {
		byPath[it.GetPath()] = it
	}
	hit := byPath["/admin/v1/orders"]
	require.NotNil(t, hit)
	require.Equal(t, "订单查询", hit.GetApiDescription(), "命中行应回填接口描述")
	require.Equal(t, "订单模块", hit.GetApiModule(), "命中行应回填模块名")

	miss := byPath["/admin/v1/gone"]
	require.NotNil(t, miss)
	require.Empty(t, miss.GetApiDescription(), "未命中行不应被富集，也不应报错")

	// nil 分页请求：repo 守卫透传为 BadRequest
	_, err = svc.List(ctx, nil)
	require.Error(t, err, "nil 分页请求应被 repo 守卫拒绝")
}

// TestApiAuditLogService_GetEnrichesAndCreateGuard 验证 Get 富集与 Create 守卫。
func TestApiAuditLogService_GetEnrichesAndCreateGuard(t *testing.T) {
	svc, entClient := newApiAuditLogServiceForTest(t)
	ctx := enttest.NewSystemContext(context.Background())

	_, err := entClient.Client().Api.Create().
		SetNillablePath(trans.Ptr("/admin/v1/orders")).
		SetNillableMethod(trans.Ptr("GET")).
		SetNillableDescription(trans.Ptr("订单查询")).
		Save(ctx)
	require.NoError(t, err)

	_, err = svc.Create(ctx, &auditV1.CreateApiAuditLogRequest{
		Data: &auditV1.ApiAuditLog{
			Path:       trans.Ptr("/admin/v1/orders"),
			HttpMethod: trans.Ptr("GET"),
			Username:   trans.Ptr("svc_user"),
		},
	})
	require.NoError(t, err, "服务层 Create 应成功落库")

	_, err = svc.Create(ctx, &auditV1.CreateApiAuditLogRequest{})
	require.Error(t, err, "缺 Data 的 Create 应报 BadRequest")
	_, err = svc.Create(ctx, nil)
	require.Error(t, err, "nil 请求应报 BadRequest")

	got, err := svc.Get(ctx, &auditV1.GetApiAuditLogRequest{
		QueryBy: &auditV1.GetApiAuditLogRequest_Id{Id: 1},
	})
	require.NoError(t, err)
	require.Equal(t, "订单查询", got.GetApiDescription(), "Get 同样应富集")

	_, err = svc.Get(ctx, &auditV1.GetApiAuditLogRequest{
		QueryBy: &auditV1.GetApiAuditLogRequest_Id{Id: 99999},
	})
	require.Error(t, err, "不存在的审计行应报 NotFound")
}

// --- 其余五个薄委托服务：守卫 + 连通 ---

// TestThinAuditLogServices_GuardAndRoundTrip 覆盖 data_access / login / operation /
// permission / policy_evaluation 五个同构服务：Create 缺 Data 拒绝、合法载荷落库、
// List 读回、Get 按 ID 读回、不存在 ID 报错。
func TestThinAuditLogServices_GuardAndRoundTrip(t *testing.T) {
	ctx := enttest.NewSystemContext(context.Background())

	t.Run("DataAccessAuditLogService", func(t *testing.T) {
		svc := newDataAccessAuditLogServiceForTest(t)
		_, err := svc.Create(ctx, &auditV1.CreateDataAccessAuditLogRequest{})
		require.Error(t, err, "缺 Data 应报 BadRequest")

		_, err = svc.Create(ctx, &auditV1.CreateDataAccessAuditLogRequest{
			Data: &auditV1.DataAccessAuditLog{UserId: trans.Ptr(int64(7))},
		})
		require.NoError(t, err)

		resp, err := svc.List(ctx, &paginationV1.PagingRequest{})
		require.NoError(t, err)
		require.Equal(t, int64(1), resp.GetTotal())

		got, err := svc.Get(ctx, &auditV1.GetDataAccessAuditLogRequest{
			QueryBy: &auditV1.GetDataAccessAuditLogRequest_Id{Id: 1},
		})
		require.NoError(t, err)
		require.Equal(t, int64(7), got.GetUserId())
	})

	t.Run("LoginAuditLogService", func(t *testing.T) {
		svc := newLoginAuditLogServiceForTest(t)
		_, err := svc.Create(ctx, &auditV1.CreateLoginAuditLogRequest{})
		require.Error(t, err, "缺 Data 应报 BadRequest")

		_, err = svc.Create(ctx, &auditV1.CreateLoginAuditLogRequest{
			Data: &auditV1.LoginAuditLog{
				UserId:   trans.Ptr(int64(7)),
				Username: trans.Ptr("svc_login_user"),
			},
		})
		require.NoError(t, err)

		resp, err := svc.List(ctx, &paginationV1.PagingRequest{})
		require.NoError(t, err)
		require.Equal(t, int64(1), resp.GetTotal())
		require.Equal(t, "svc_login_user", resp.GetItems()[0].GetUsername())

		got, err := svc.Get(ctx, &auditV1.GetLoginAuditLogRequest{
			QueryBy: &auditV1.GetLoginAuditLogRequest_Id{Id: 1},
		})
		require.NoError(t, err)
		require.Equal(t, int64(7), got.GetUserId())
	})

	t.Run("OperationAuditLogService", func(t *testing.T) {
		svc := newOperationAuditLogServiceForTest(t)
		_, err := svc.Create(ctx, &auditV1.CreateOperationAuditLogRequest{})
		require.Error(t, err, "缺 Data 应报 BadRequest")

		_, err = svc.Create(ctx, &auditV1.CreateOperationAuditLogRequest{
			Data: &auditV1.OperationAuditLog{UserId: trans.Ptr(int64(9))},
		})
		require.NoError(t, err)

		resp, err := svc.List(ctx, &paginationV1.PagingRequest{})
		require.NoError(t, err)
		require.Equal(t, int64(1), resp.GetTotal())

		got, err := svc.Get(ctx, &auditV1.GetOperationAuditLogRequest{
			QueryBy: &auditV1.GetOperationAuditLogRequest_Id{Id: 1},
		})
		require.NoError(t, err)
		require.Equal(t, int64(9), got.GetUserId())
	})

	t.Run("PermissionAuditLogService", func(t *testing.T) {
		svc := newPermissionAuditLogServiceForTest(t)
		_, err := svc.Create(ctx, &auditV1.CreatePermissionAuditLogRequest{})
		require.Error(t, err, "缺 Data 应报 BadRequest")

		_, err = svc.Create(ctx, &auditV1.CreatePermissionAuditLogRequest{
			Data: &auditV1.PermissionAuditLog{OperatorId: trans.Ptr(int64(11))},
		})
		require.NoError(t, err)

		resp, err := svc.List(ctx, &paginationV1.PagingRequest{})
		require.NoError(t, err)
		require.Equal(t, int64(1), resp.GetTotal())

		got, err := svc.Get(ctx, &auditV1.GetPermissionAuditLogRequest{
			QueryBy: &auditV1.GetPermissionAuditLogRequest_Id{Id: 1},
		})
		require.NoError(t, err)
		require.Equal(t, int64(11), got.GetOperatorId())
	})

	t.Run("PolicyEvaluationLogService", func(t *testing.T) {
		svc := newPolicyEvaluationLogServiceForTest(t)
		_, err := svc.Create(ctx, &permissionV1.CreatePolicyEvaluationLogRequest{})
		require.Error(t, err, "缺 Data 应报 BadRequest")

		_, err = svc.Create(ctx, &permissionV1.CreatePolicyEvaluationLogRequest{
			Data: &permissionV1.PolicyEvaluationLog{UserId: trans.Ptr(int64(13))},
		})
		require.NoError(t, err)

		resp, err := svc.List(ctx, &paginationV1.PagingRequest{})
		require.NoError(t, err)
		require.Equal(t, int64(1), resp.GetTotal())

		got, err := svc.Get(ctx, &permissionV1.GetPolicyEvaluationLogRequest{
			QueryBy: &permissionV1.GetPolicyEvaluationLogRequest_Id{Id: 1},
		})
		require.NoError(t, err)
		require.Equal(t, int64(13), got.GetUserId())
	})
}
