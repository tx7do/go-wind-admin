package data

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/api"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newApiRepoSqlite 在给定 enttest client 上白盒构造 ApiRepo，
// 逐字段复刻 NewApiRepo 的 mapper/converter 初始化，再调用 init()。
func newApiRepoSqlite(t *testing.T, entClient *entCrud.EntClient[*ent.Client]) *ApiRepo {
	t.Helper()
	repo := &ApiRepo{
		entClient: entClient,
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		mapper:    mapper.NewCopierMapper[permissionV1.Api, ent.Api](),
		statusConverter: mapper.NewEnumTypeConverter[permissionV1.Api_Status, api.Status](
			permissionV1.Api_Status_name, permissionV1.Api_Status_value,
		),
		scopeConverter: mapper.NewEnumTypeConverter[permissionV1.Api_Scope, api.Scope](
			permissionV1.Api_Scope_name, permissionV1.Api_Scope_value,
		),
		businessModuleConverter: mapper.NewEnumTypeConverter[identityV1.Module, api.BusinessModule](
			identityV1.Module_name, identityV1.Module_value,
		),
	}
	repo.init()
	return repo
}

// TestApiRepoSqlite_Create 通过 repo.Create 写入后直查 SQLite 断言落库
// （含 scope / business_module 枚举经 converter 的落库值）。
func TestApiRepoSqlite_Create(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newApiRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	err := repo.Create(ctx, &permissionV1.CreateApiRequest{
		Data: &permissionV1.Api{
			Operation:         trans.Ptr("ListFoo"),
			Path:              trans.Ptr("/sqlite/api/create"),
			Method:            trans.Ptr("GET"),
			Module:            trans.Ptr("sqlite-module"),
			ModuleDescription: trans.Ptr("模块描述"),
			Description:       trans.Ptr("接口描述"),
			Scope:             permissionV1.Api_ADMIN.Enum(),
			BusinessModule:    identityV1.Module_SYSTEM.Enum(),
		},
	})
	require.NoError(t, err, "repo.Create 应写入 SQLite 成功")

	rows, err := repo.entClient.Client().Api.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1, "SQLite 中应有 1 条 api 记录")
	require.Equal(t, "ListFoo", *rows[0].Operation, "operation 应按请求落库")
	require.Equal(t, "/sqlite/api/create", *rows[0].Path, "path 应按请求落库")
	require.Equal(t, "GET", *rows[0].Method, "method 应按请求落库")
	require.Equal(t, "sqlite-module", *rows[0].Module, "module 应按请求落库")
	require.Equal(t, api.ScopeAdmin, *rows[0].Scope, "scope 枚举应经 converter 落为 ADMIN")
	require.NotNil(t, rows[0].BusinessModule, "business_module 枚举应经 converter 落库")
	require.Equal(t, api.BusinessModuleSystem, *rows[0].BusinessModule, "business_module 枚举应落为 SYSTEM")

	// 读视图（Get 按主键）：scope/business_module 为写入值、status 未显式指定
	// 按列默认（ON）落库，三者均经 queryEnumsAndBackfill 如实回显。
	got, err := repo.Get(ctx, &permissionV1.GetApiRequest{
		QueryBy: &permissionV1.GetApiRequest_Id{Id: rows[0].ID},
	})
	require.NoError(t, err, "按主键读取刚创建的行应命中")
	require.Equal(t, permissionV1.Api_ADMIN, got.GetScope(), "读视图应回填写入的 scope")
	require.Equal(t, identityV1.Module_SYSTEM, got.GetBusinessModule(), "读视图应回填写入的 business_module")
	require.Equal(t, permissionV1.Api_ON, got.GetStatus(), "读视图应回填列默认 status")
}

// TestApiRepoSqlite_ListContainsFilter 验证 List 的 contains 模糊搜索语义。
func TestApiRepoSqlite_ListContainsFilter(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newApiRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &permissionV1.CreateApiRequest{
		Data: &permissionV1.Api{
			Path:   trans.Ptr("/markerghi/a"),
			Method: trans.Ptr("GET"),
		},
	}))
	require.NoError(t, repo.Create(ctx, &permissionV1.CreateApiRequest{
		Data: &permissionV1.Api{
			Path:   trans.Ptr("/unrelated/b"),
			Method: trans.Ptr("POST"),
		},
	}))

	filtered, err := repo.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_FilterExpr{
			FilterExpr: &paginationV1.FilterExpr{
				Type: paginationV1.ExprType_AND,
				Conditions: []*paginationV1.FilterCondition{
					{
						Field:      "path",
						Op:         paginationV1.Operator_CONTAINS,
						ValueOneof: &paginationV1.FilterCondition_Value{Value: "markerghi"},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), filtered.Total, "contains 过滤后 Total 应为 1")
	require.Len(t, filtered.Items, 1, "contains 过滤后只应返回 1 行")
	require.Contains(t, *filtered.Items[0].Path, "markerghi", "命中行应是携带标记的行")

	none, err := repo.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_FilterExpr{
			FilterExpr: &paginationV1.FilterExpr{
				Type: paginationV1.ExprType_AND,
				Conditions: []*paginationV1.FilterCondition{
					{
						Field:      "path",
						Op:         paginationV1.Operator_CONTAINS,
						ValueOneof: &paginationV1.FilterCondition_Value{Value: "no-such-marker-zzz"},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(0), none.Total, "无命中 contains 应返回 Total=0")
	require.Empty(t, none.Items)

	all, err := repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(2), all.Total, "无过滤时应返回全部 2 行")
	require.Len(t, all.Items, 2)
	// 列表读视图：status/scope 未显式指定、均按列默认（ON/ADMIN）落库，经
	// queryEnumsAndBackfill 如实回显；business_module 列为 NULL，无可回填值、
	// 读视图保持缺省。
	for _, item := range all.Items {
		require.Equal(t, permissionV1.Api_ON, item.GetStatus(), "列表读视图应回填列默认 status")
		require.Equal(t, permissionV1.Api_ADMIN, item.GetScope(), "列表读视图应回填列默认 scope")
	}
}

// TestApiRepoSqlite_Get 验证 Get 命中/未命中与 GetApiByEndpoint 的命中/未命中。
func TestApiRepoSqlite_Get(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newApiRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &permissionV1.CreateApiRequest{
		Data: &permissionV1.Api{
			Operation: trans.Ptr("GetFoo"),
			Path:      trans.Ptr("/sqlite/api/get"),
			Method:    trans.Ptr("PUT"),
		},
	}))
	rows, err := repo.entClient.Client().Api.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := rows[0].ID

	// Get 命中/未命中
	hit, err := repo.Get(ctx, &permissionV1.GetApiRequest{
		QueryBy: &permissionV1.GetApiRequest_Id{Id: createdID},
	})
	require.NoError(t, err, "按存在的 ID 查询应命中")
	require.Equal(t, createdID, hit.GetId())
	// 读视图（Get 按主键）：status/scope 未显式指定、均按列默认（ON/ADMIN）
	// 落库，经 queryEnumsAndBackfill 如实回显；business_module 列为 NULL，
	// 无可回填值、读视图保持缺省。
	require.Equal(t, permissionV1.Api_ON, hit.GetStatus(), "读视图应回填列默认 status")
	require.Equal(t, permissionV1.Api_ADMIN, hit.GetScope(), "读视图应回填列默认 scope")
	_, err = repo.Get(ctx, &permissionV1.GetApiRequest{
		QueryBy: &permissionV1.GetApiRequest_Id{Id: 9999999},
	})
	require.Error(t, err, "不存在的 ID 查询应返回错误")

	// GetApiByEndpoint 命中
	byEndpoint, err := repo.GetApiByEndpoint(ctx, "/sqlite/api/get", "PUT")
	require.NoError(t, err, "按存在的路径+方法查询应命中")
	require.Equal(t, createdID, byEndpoint.GetId())
	// 读视图（GetApiByEndpoint 路径）：同上，列默认 status/scope 如实回显。
	require.Equal(t, permissionV1.Api_ON, byEndpoint.GetStatus(), "读视图应回填列默认 status")
	require.Equal(t, permissionV1.Api_ADMIN, byEndpoint.GetScope(), "读视图应回填列默认 scope")

	// GetApiByEndpoint 未命中
	_, err = repo.GetApiByEndpoint(ctx, "/no/such/path", "GET")
	require.Error(t, err, "不存在的路径+方法查询应返回 NotFound")

	// GetApiByEndpoint 参数校验：空路径/空方法
	_, err = repo.GetApiByEndpoint(ctx, "", "GET")
	require.Error(t, err, "空路径应返回 BadRequest")
	_, err = repo.GetApiByEndpoint(ctx, "/x", "")
	require.Error(t, err, "空方法应返回 BadRequest")

	// GetApiByIDs 命中/空集合
	byIDs, err := repo.GetApiByIDs(ctx, []uint32{createdID})
	require.NoError(t, err)
	require.Len(t, byIDs, 1, "按存在的 ID 集合应返回 1 行")
	// 读视图（GetApiByIDs 路径）：同上，列默认 status/scope 如实回显。
	require.Equal(t, permissionV1.Api_ON, byIDs[0].GetStatus(), "读视图应回填列默认 status")
	require.Equal(t, permissionV1.Api_ADMIN, byIDs[0].GetScope(), "读视图应回填列默认 scope")
	_, err = repo.GetApiByIDs(ctx, nil)
	require.Error(t, err, "空 ID 集合应返回 BadRequest")
}

// TestApiRepoSqlite_Update 验证 Update 只更新掩码内字段（description），
// 掩码外字段（path）保持原值。
func TestApiRepoSqlite_Update(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newApiRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &permissionV1.CreateApiRequest{
		Data: &permissionV1.Api{
			Path:        trans.Ptr("/sqlite/api/update"),
			Method:      trans.Ptr("DELETE"),
			Description: trans.Ptr("更新前描述"),
		},
	}))
	rows, err := repo.entClient.Client().Api.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := rows[0].ID

	err = repo.Update(ctx, &permissionV1.UpdateApiRequest{
		Id:         createdID,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description"}},
		Data: &permissionV1.Api{
			Description: trans.Ptr("更新后描述-sqlite"),
		},
	})
	require.NoError(t, err, "更新 description 应成功")

	after, err := repo.entClient.Client().Api.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, after, 1)
	require.Equal(t, "更新后描述-sqlite", *after[0].Description, "掩码内字段应被更新")
	require.Equal(t, "/sqlite/api/update", *after[0].Path, "掩码外字段 path 应保持原值")
}

// TestApiRepoSqlite_Delete 验证 Delete 后行数归零。
func TestApiRepoSqlite_Delete(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newApiRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &permissionV1.CreateApiRequest{
		Data: &permissionV1.Api{
			Path:   trans.Ptr("/sqlite/api/delete"),
			Method: trans.Ptr("GET"),
		},
	}))
	rows, err := repo.entClient.Client().Api.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)

	require.NoError(t, repo.Delete(ctx, &permissionV1.DeleteApiRequest{
		QueryBy: &permissionV1.DeleteApiRequest_Id{Id: rows[0].ID},
	}), "Delete 应成功")
	cnt, err := repo.entClient.Client().Api.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, cnt, "Delete 后表内行数应为 0")
}
