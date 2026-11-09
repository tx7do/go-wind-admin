package service

import (
	"context"
	"testing"

	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/authorizer"
	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/ent"
	entApi "go-wind-admin/app/admin/service/internal/data/ent/api"
	"go-wind-admin/app/admin/service/internal/data/enttest"
	"go-wind-admin/pkg/middleware/auth"
)

// stubAuthzProvider 是 authorizer.Provider 的本地测试桩：
// 无模型、无策略数据（noop 引擎的 ResetPolicies 在取数后直接返回 nil）。
type stubAuthzProvider struct{}

func (stubAuthzProvider) ProvideModels(string) authorizer.ModelDataMap { return nil }

func (stubAuthzProvider) ProvidePolicies(context.Context) (authorizer.PermissionDataMap, error) {
	return nil, nil
}

// stubRouteWalker 是 RouteWalker 接口的本地测试桩：按预置路由表回调遍历函数。
type stubRouteWalker struct {
	routes []http.RouteInfo
}

func (s *stubRouteWalker) WalkRoute(fn http.WalkRouteFunc) error {
	for _, r := range s.routes {
		if err := fn(r); err != nil {
			return err
		}
	}
	return nil
}

// newApiServiceForTest 白盒构造 ApiService，逐字段对齐 NewApiService 的装配：
// log 用 NopLogger helper，repo 走 data.NewApiRepoForTest，authorizer 用真实
// NewAuthorizer 构造的 noop 引擎实例（provider 为本地桩，ResetPolicies 走 noop
// 引擎分支直接成功），routeWalker 由各用例按需注入。
func newApiServiceForTest(t *testing.T, entClient *entCrud.EntClient[*ent.Client]) *ApiService {
	t.Helper()
	authz := authorizer.NewAuthorizer(
		context.Background(),
		bLogger.NopLogger(),
		&authorizer.EngineConfig{Type: "noop"},
		stubAuthzProvider{},
	)
	return &ApiService{
		log:         bLogger.NewHelper(bLogger.NopLogger()),
		repo:        data.NewApiRepoForTest(entClient),
		authorizer:  authz,
		routeWalker: nil,
	}
}

// TestApiServiceSqlite_InitSeedsApiTableFromOpenAPI 空表上调用 init() 应从内嵌
// OpenAPI 文档同步出全部接口资源（count==0 守卫的正分支）；表非空时再次 init()
// 不应重复同步；List 应返回同步后的全量行。
func TestApiServiceSqlite_InitSeedsApiTableFromOpenAPI(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newApiServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	svc.init()

	cnt, err := entClient.Client().Api.Query().Count(ctx)
	require.NoError(t, err)
	require.Greater(t, cnt, 0, "init() 后 api 表应从 OpenAPI 文档同步出接口资源")

	listResp, err := svc.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(cnt), listResp.Total, "List 应统计全部已同步行")
	require.Len(t, listResp.Items, cnt, "List 应返回全部已同步行")

	// 表非空：再次 init() 不应重复同步（守卫负分支）
	svc.init()
	cnt2, err := entClient.Client().Api.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, cnt, cnt2, "非空表上重复 init() 不应追加或重置行")
}

// TestApiServiceSqlite_GetWalkRouteData 验证 GetWalkRouteData 把 RouteWalker
// 遍历到的路由组装成列表（Total 与条目一一对应，状态统一置 ON）；
// 未注册 walker 时应返回错误。
func TestApiServiceSqlite_GetWalkRouteData(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newApiServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	_, err := svc.GetWalkRouteData(ctx, &emptypb.Empty{})
	require.Error(t, err, "未注册 RouteWalker 时应返回错误")

	svc.RegisterRouteWalker(&stubRouteWalker{routes: []http.RouteInfo{
		{Path: "/stub-alpha", Method: "GET"},
		{Path: "/stub-beta", Method: "POST"},
	}})

	resp, err := svc.GetWalkRouteData(ctx, &emptypb.Empty{})
	require.NoError(t, err, "注册 walker 后遍历应成功")
	require.Equal(t, uint64(2), resp.GetTotal(), "Total 应等于遍历到的路由数")
	require.Len(t, resp.GetItems(), 2, "条目数应等于遍历到的路由数")
	require.Equal(t, "/stub-alpha", resp.GetItems()[0].GetPath(), "第 1 条应保持 walker 给出的路径")
	require.Equal(t, "GET", resp.GetItems()[0].GetMethod(), "第 1 条应保持 walker 给出的方法")
	require.Equal(t, "/stub-beta", resp.GetItems()[1].GetPath(), "第 2 条应保持 walker 给出的路径")
	require.Equal(t, "POST", resp.GetItems()[1].GetMethod(), "第 2 条应保持 walker 给出的方法")
	for _, item := range resp.GetItems() {
		require.Equal(t, permissionV1.Api_ON, item.GetStatus(), "遍历产物的状态应统一置为 ON")
	}
}

// TestApiServiceSqlite_CreateGetUpdateDelete 验证服务层 CRUD：
// 创建（含 scope/business_module 枚举转换与操作人盖章）、按主键查询的命中/未命中、
// 单字段掩码更新、删除归零。
func TestApiServiceSqlite_CreateGetUpdateDelete(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newApiServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7, IsPlatformAdmin: trans.Ptr(true)})

	_, err := svc.Create(opCtx, &permissionV1.CreateApiRequest{
		Data: &permissionV1.Api{
			Path:              trans.Ptr("/svc/test-path"),
			Method:            trans.Ptr("GET"),
			Operation:         trans.Ptr("SvcTestOperation"),
			Module:            trans.Ptr("svc"),
			ModuleDescription: trans.Ptr("服务层测试模块"),
			Description:       trans.Ptr("服务层创建的接口描述"),
			Scope:             permissionV1.Api_ADMIN.Enum(),
			BusinessModule:    identityV1.Module_DASHBOARD.Enum(),
		},
	})
	require.NoError(t, err, "服务层 Create 应成功")

	rows, err := entClient.Client().Api.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1, "api 表应落库 1 条记录")
	require.Equal(t, "/svc/test-path", *rows[0].Path, "path 应按请求落库")
	require.Equal(t, "GET", *rows[0].Method, "method 应按请求落库")
	require.Equal(t, "SvcTestOperation", *rows[0].Operation, "operation 应按请求落库")
	require.Equal(t, "svc", *rows[0].Module, "module 应按请求落库")
	require.Equal(t, "服务层测试模块", *rows[0].ModuleDescription, "module_description 应按请求落库")
	require.Equal(t, "服务层创建的接口描述", *rows[0].Description, "description 应按请求落库")
	require.Equal(t, entApi.ScopeAdmin, *rows[0].Scope, "scope 应经转换器落库为 ADMIN")
	require.Equal(t, entApi.BusinessModuleDashboard, *rows[0].BusinessModule, "business_module 应经转换器落库为 DASHBOARD")
	require.NotNil(t, rows[0].CreatedBy, "created_by 应被服务层盖入操作人 ID")
	require.Equal(t, uint64(7), *rows[0].CreatedBy, "created_by 应等于令牌声明中的操作人 ID")

	got, err := svc.Get(ctx, &permissionV1.GetApiRequest{
		QueryBy: &permissionV1.GetApiRequest_Id{Id: rows[0].ID},
	})
	require.NoError(t, err, "按已存在主键查询应命中")
	require.Equal(t, "/svc/test-path", got.GetPath(), "命中记录的 path 应与写入一致")

	_, err = svc.Get(ctx, &permissionV1.GetApiRequest{
		QueryBy: &permissionV1.GetApiRequest_Id{Id: 9999999},
	})
	require.Error(t, err, "按不存在主键查询应返回错误")

	_, err = svc.Update(opCtx, &permissionV1.UpdateApiRequest{
		Id:         rows[0].ID,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description"}},
		Data:       &permissionV1.Api{Description: trans.Ptr("更新后的接口描述")},
	})
	require.NoError(t, err, "单字段掩码更新应成功")

	after, err := entClient.Client().Api.Query().Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "更新后的接口描述", *after.Description, "掩码内字段 description 应被更新")
	require.Equal(t, "/svc/test-path", *after.Path, "掩码外字段 path 应保持原值")

	_, err = svc.Delete(auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7, IsPlatformAdmin: trans.Ptr(true)}), &permissionV1.DeleteApiRequest{
		QueryBy: &permissionV1.DeleteApiRequest_Id{Id: after.ID},
	})
	require.NoError(t, err, "删除已存在记录应成功")

	cnt, err := entClient.Client().Api.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, cnt, "删除后 api 表计数应归零")
}
