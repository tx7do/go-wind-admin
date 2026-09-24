// PermissionService 的 SQLite 内存库集成测试（白盒，包内测试）。
//
// 覆盖目标：
//   - init() 播种链：权限组缺失时默认权限首条即中止（外键违约）；
//     权限组齐备时首条（无关联资源）落库、第二条（关联不存在的
//     api/menu）中止；非空表二次 init 不重复播种。
//   - syncWithOpenAPI 空路径分支（含既定空壳测试 TestPermissionService_syncWithOpenAPI_EmptyPaths）：
//     无 paths 键的文档 → "paths is nil" 内部错误且不落行；
//     空 paths 对象 → 同步空集成功；真实内嵌文档 → 接口表播种（SyncApis 全链路）。
//   - SyncPermissions 菜单派生 + 接口派生：目录菜单成组、子菜单成权限、
//     菜单路径一级段决定归属模块、接口代码前缀匹配模块后生成接口派生权限、
//     权限-组关联（GroupID）与权限-菜单/权限-接口关联行落库、未分类默认组恒建。
//   - List/Get 的租户范围限定：租户操作人无任何可见权限 → 空列表 / Forbidden；
//     平台操作人全量可见 + GroupName 富集（组存在时回填组名）。
//   - Create/Update/Delete：参数校验、操作人盖章、单字段掩码更新、删除清空。
//
// 跳过项：真实 Casbin 策略引擎（authorizer 一律 noop 引擎 + 桩 provider）、
// 跨实例策略重载广播。
package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	conf "github.com/tx7do/kratos-bootstrap/api/gen/go/conf/v1"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	crudViewer "github.com/tx7do/go-crud/viewer"
	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
	"go-wind-admin/app/admin/service/cmd/server/assets"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/enttest"
	"go-wind-admin/pkg/authorizer"
	"go-wind-admin/pkg/constants"
	appViewer "go-wind-admin/pkg/entgo/viewer"
	"go-wind-admin/pkg/middleware/auth"
	"go-wind-admin/pkg/utils/converter"
)

// newPermissionServiceForTest 白盒构造 PermissionService，逐字段对齐
// NewPermissionService 的装配（log 用 NopLogger helper；除 PermissionRepo 外
// 的 repo 均用 testkit 构造器；PermissionRepo 无 testkit 导出，按生产构造器 +
// NopLogger 的 bootstrap 上下文构造，唯一差异仍是日志；converter 与生产一致）。
// init() 不在构造时调用，由用例按需触发。
func newPermissionServiceForTest(t *testing.T) (*PermissionService, *ent.Client) {
	t.Helper()
	entClient := enttest.NewEntClientForTest(t)
	bootstrapCtx := bootstrap.NewContextWithParam(context.Background(), nil,
		&conf.Bootstrap{Authz: &conf.Authorization{Type: "noop"}}, bLogger.NopLogger())
	svc := &PermissionService{
		log:                     bLogger.NewHelper(bLogger.NopLogger()),
		permissionRepo:          data.NewPermissionRepoForTest(entClient),
		permissionGroupRepo:     data.NewPermissionGroupRepoForTest(entClient),
		menuRepo:                data.NewMenuRepoForTest(entClient),
		apiRepo:                 data.NewApiRepoForTest(entClient),
		roleRepo:                data.NewRoleRepoForTest(entClient),
		authorizer:              authorizer.NewAuthorizer(bootstrapCtx, stubAuthzProvider{}),
		menuPermissionConverter: converter.NewMenuPermissionConverter(),
		apiPermissionConverter:  converter.NewApiPermissionConverter(),
	}
	return svc, entClient.Client()
}

// seedPermissionGroups 落 count 条占位权限组（自动 ID 从 1 顺次递增）。
func seedPermissionGroups(t *testing.T, svc *PermissionService, ctx context.Context, count int) {
	t.Helper()
	for i := 1; i <= count; i++ {
		_, err := svc.permissionGroupRepo.Create(ctx, &permissionV1.CreatePermissionGroupRequest{
			Data: &permissionV1.PermissionGroup{
				Name:      trans.Ptr("PermSvcGroup"),
				Module:    trans.Ptr("perm_svc_group"),
				SortOrder: trans.Ptr(uint32(i)),
				Status:    permissionV1.PermissionGroup_ON.Enum(),
			},
		})
		require.NoError(t, err, "落占位权限组 %d 应成功", i)
	}
}

// TestPermissionService_InitSeeding 验证 init() 的默认权限播种：
// 空表播种全部默认权限（关联资源缺失不中止——关联表无外键强制、
// CleanNotExist 与批量插入对缺失资源静默无操作），非空表二次 init 不重复播种。
func TestPermissionService_InitSeeding(t *testing.T) {
	svc, client := newPermissionServiceForTest(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	svc.init()
	cnt, err := client.Permission.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, len(constants.DefaultPermissions), int(cnt),
		"init() 空表应把全部默认权限落库")

	svc.init()
	cnt2, err := client.Permission.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, cnt, cnt2, "非空表二次 init 不应重复播种")
}

// TestPermissionService_InitSkipsSyncWhenNoApisOrMenus 验证 init() 的
// SyncPermissions 触发条件：api 表或菜单表为空时（此例两表皆空）
// 即使权限表为空播种后也不触发同步（menusCount==0 分支）。
func TestPermissionService_InitSkipsSyncWhenNoApisOrMenus(t *testing.T) {
	svc, client := newPermissionServiceForTest(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())
	seedPermissionGroups(t, svc, ctx, 5)

	svc.init()

	groupCnt, err := client.PermissionGroup.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 5, groupCnt, "占位组应保持 5 条（SyncPermissions 未触发，未追加未分类组）")
}

// TestPermissionService_syncWithOpenAPI_EmptyPaths（原空壳测试的实装）：
// 无 paths 键的文档 → "paths is nil" 内部错误且接口表不落行；
// 空 paths 对象 → 同步空集成功不落行；真实内嵌文档 → SyncApis 播种接口表。
func TestPermissionService_syncWithOpenAPI_EmptyPaths(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	apiSvc := newApiServiceForTest(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	orig := assets.OpenApiData
	defer func() { assets.OpenApiData = orig }()

	// 无 paths 键：加载成功但 Paths==nil → 内部错误
	assets.OpenApiData = []byte("openapi: 3.0.0\ninfo:\n  title: empty-paths-test\n  version: \"1.0\"\n")
	err := apiSvc.syncWithOpenAPI(ctx)
	require.Error(t, err, "无 paths 键的文档应触发 paths is nil 内部错误")
	require.True(t, adminV1.IsInternalServerError(err))
	require.ErrorContains(t, err, "paths is nil", "应命中 Paths==nil 分支")
	cnt, qerr := entClient.Client().Api.Query().Count(ctx)
	require.NoError(t, qerr)
	require.Zero(t, cnt, "失败分支不应落任何接口行")

	// 空 paths 对象：同步空集成功
	assets.OpenApiData = []byte("openapi: 3.0.0\ninfo:\n  title: empty-paths-test\n  version: \"1.0\"\npaths: {}\n")
	require.NoError(t, apiSvc.syncWithOpenAPI(ctx), "空 paths 对象应同步空集成功")
	cnt, qerr = entClient.Client().Api.Query().Count(ctx)
	require.NoError(t, qerr)
	require.Zero(t, cnt, "空集同步不应落行")

	// 真实内嵌文档：SyncApis 全链路播种接口表
	assets.OpenApiData = orig
	_, err = apiSvc.SyncApis(ctx, &emptypb.Empty{})
	require.NoError(t, err, "真实文档的 SyncApis 应成功")
	cnt, qerr = entClient.Client().Api.Query().Count(ctx)
	require.NoError(t, qerr)
	require.Greater(t, cnt, 0, "SyncApis 应从内嵌文档播种接口表")
	listResp, lerr := apiSvc.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, lerr)
	require.Equal(t, uint64(cnt), listResp.GetTotal(), "List 应返回同步后的全量行")
	require.Len(t, listResp.GetItems(), cnt)
}

// TestPermissionService_SyncPermissions_MenuAndApiDerived 验证权限同步的
// 菜单派生与接口派生：目录菜单生成组、子菜单生成权限、一级路径段决定归属
// 模块、接口代码经前缀匹配模块后生成接口派生权限；关联行（权限-组、
// 权限-菜单、权限-接口）落库；未分类默认组恒建。
func TestPermissionService_SyncPermissions_MenuAndApiDerived(t *testing.T) {
	svc, client := newPermissionServiceForTest(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7})

	// 目录菜单（根）与页面菜单（子）
	require.NoError(t, svc.menuRepo.Create(ctx, &permissionV1.CreateMenuRequest{
		Data: &permissionV1.Menu{
			Name:   trans.Ptr("PermSvc 目录菜单"),
			Path:   trans.Ptr("/bizgrp"),
			Type:   permissionV1.Menu_CATALOG.Enum(),
			Status: permissionV1.Menu_ON.Enum(),
		},
	}))
	rootRows, err := client.Menu.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rootRows, 1)
	rootID := rootRows[0].ID
	require.NoError(t, svc.menuRepo.Create(ctx, &permissionV1.CreateMenuRequest{
		Data: &permissionV1.Menu{
			Name:     trans.Ptr("PermSvc 子菜单"),
			Path:     trans.Ptr("childpg"),
			ParentId: &rootID,
			Type:     permissionV1.Menu_MENU.Enum(),
			Status:   permissionV1.Menu_ON.Enum(),
		},
	}))

	// 接口行：资源段与目录菜单一级段同前缀（状态缺省即 ON）
	require.NoError(t, svc.apiRepo.Create(ctx, &permissionV1.CreateApiRequest{
		Data: &permissionV1.Api{
			Path:   trans.Ptr("/api/v1/bizgrp/entries"),
			Method: trans.Ptr("GET"),
			Scope:  permissionV1.Api_ADMIN.Enum(),
		},
	}))

	_, err = svc.SyncPermissions(opCtx, &emptypb.Empty{})
	require.NoError(t, err, "SyncPermissions 应成功")

	// 权限组：未分类默认组 + 目录菜单一级段派生组
	groups, err := client.PermissionGroup.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, groups, 2, "应恰好出现未分类组与目录菜单派生组")
	modules := map[string]uint32{}
	var uncategorizedName string
	for _, g := range groups {
		require.NotNil(t, g.Module)
		require.NotNil(t, g.Name)
		modules[*g.Module] = g.ID
		if *g.Module == "uncategorized" {
			uncategorizedName = *g.Name
		}
	}
	require.Len(t, modules, 2, "两个组的模块标识应互不相同")
	require.Equal(t, "未分类", uncategorizedName, "未分类组名应为固定文案")
	bizGroupID, hasBiz := modules["bizgrp"]
	require.True(t, hasBiz, "目录菜单一级段应派生 bizgrp 组")

	// 权限：菜单派生两条（目录/子页面）+ 接口派生一条，全部归组 bizgrp
	perms, err := client.Permission.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, perms, 3, "应落两条菜单派生权限与一条接口派生权限")
	codes := map[string]bool{}
	for _, p := range perms {
		require.NotNil(t, p.Code, "派生权限应带代码")
		require.NotNil(t, p.GroupID, "派生权限应归组")
		require.Equal(t, bizGroupID, *p.GroupID, "派生权限应归入同模块组")
		codes[*p.Code] = true
	}
	require.True(t, codes["bizgrp:dir"], "目录菜单应派生 <一级段>:dir 代码权限（转换器钉死语义）")
	require.True(t, codes["childpg:view"], "子页面菜单应派生 <子段>:view 代码权限（首段丢弃语义）")

	// 关联行：权限-菜单两条、权限-接口一条（接口派生权限关联到接口行）
	menuLinks, err := client.PermissionMenu.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, menuLinks, "两条菜单派生权限应各建一条权限-菜单关联")
	apiLinks, err := client.PermissionApi.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, apiLinks, "接口派生权限应建一条权限-接口关联")
}

// TestPermissionService_ListAndGetTenantScoping 验证 List/Get 的租户范围限定
// 与平台侧全量可见 + 组名富集。
func TestPermissionService_ListAndGetTenantScoping(t *testing.T) {
	svc, _ := newPermissionServiceForTest(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())
	seedPermissionGroups(t, svc, ctx, 5)
	svc.init()
	platformCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7})
	// 租户操作人：查看器须与令牌租户一致（生产由认证中间件成对注入）
	tenantCtx := auth.NewContext(
		crudViewer.WithContext(ctx, appViewer.NewUserViewer(8, 42, 0, "", nil)),
		&authenticationV1.UserTokenPayload{
			UserId: 8, TenantId: trans.Ptr(uint32(42)), Roles: []string{"perm_svc_nonexistent_role"},
		})

	// 平台操作人：全量可见 + 组名富集（组行存在时每条带组引用的权限都回填组名）
	listResp, err := svc.List(platformCtx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(len(constants.DefaultPermissions)), listResp.GetTotal(),
		"平台操作人应看到全部播种的默认权限")
	require.Len(t, listResp.GetItems(), len(constants.DefaultPermissions))
	for _, item := range listResp.GetItems() {
		require.Equal(t, "PermSvcGroup", item.GetGroupName(),
			"组存在时组名应经富集回填（占位组的名称）")
	}
	item := listResp.GetItems()[0]

	// 租户操作人（租户查看器 + 匹配令牌）：无任何可见权限 → 空列表
	tenantList, err := svc.List(tenantCtx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Zero(t, tenantList.GetTotal(), "租户操作人无可见权限时应返回空列表")
	require.Empty(t, tenantList.GetItems())

	// 平台查看器 + 租户令牌声明（查看器与令牌租户不一致）：
	// 按码查角色 ID 的租户闸门按查看器判定，拒绝平台上下文的按码查询
	mismatchCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{
		UserId: 8, TenantId: trans.Ptr(uint32(42)), Roles: []string{"perm_svc_nonexistent_role"},
	})
	_, err = svc.List(mismatchCtx, &paginationV1.PagingRequest{})
	require.Error(t, err, "平台查看器下的按码查角色 ID 应被租户闸门拒绝")
	require.True(t, permissionV1.IsBadRequest(err))

	// Get：平台命中；租户 Forbidden；不存在 ID 报错
	got, err := svc.Get(platformCtx, &permissionV1.GetPermissionRequest{
		QueryBy: &permissionV1.GetPermissionRequest_Id{Id: item.GetId()},
	})
	require.NoError(t, err, "平台操作人按已存在 ID 查询应命中")
	require.NotEmpty(t, got.GetGroupName(), "单条 Get 同样应做组名富集")

	_, err = svc.Get(tenantCtx, &permissionV1.GetPermissionRequest{
		QueryBy: &permissionV1.GetPermissionRequest_Id{Id: item.GetId()},
	})
	require.Error(t, err, "租户操作人访问范围外权限应 Forbidden")
	require.True(t, adminV1.IsForbidden(err))

	_, err = svc.Get(platformCtx, &permissionV1.GetPermissionRequest{
		QueryBy: &permissionV1.GetPermissionRequest_Id{Id: 9999999},
	})
	require.Error(t, err, "不存在的权限 ID 应报错")
}

// TestPermissionService_CreateUpdateDelete 验证权限 CRUD：参数校验、
// 操作人盖章、单字段掩码更新（掩码外保持原值）、删除清空。
func TestPermissionService_CreateUpdateDelete(t *testing.T) {
	svc, client := newPermissionServiceForTest(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7})

	_, err := svc.Create(opCtx, &permissionV1.CreatePermissionRequest{})
	require.Error(t, err, "nil Data 应拒绝")
	require.True(t, adminV1.IsBadRequest(err))

	_, err = svc.Create(opCtx, &permissionV1.CreatePermissionRequest{
		Data: &permissionV1.Permission{
			Name:        trans.Ptr("PermSvc 测试权限"),
			Code:        trans.Ptr("perm_svc_test_code"),
			Description: trans.Ptr("初版描述"),
			Status:      permissionV1.Permission_ON.Enum(),
		},
	})
	require.NoError(t, err, "创建应成功")
	rows, err := client.Permission.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	rowID := rows[0].ID
	require.Equal(t, uint32(7), *rows[0].CreatedBy, "创建行应盖入操作人 ID")
	require.Equal(t, "perm_svc_test_code", *rows[0].Code, "代码应按请求落库")

	// 单字段掩码更新：description 更新，name 保持
	_, err = svc.Update(opCtx, &permissionV1.UpdatePermissionRequest{
		Id:         rowID,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description"}},
		Data:       &permissionV1.Permission{Description: trans.Ptr("更新后的描述")},
	})
	require.NoError(t, err, "单字段掩码更新应成功")
	after, err := client.Permission.Query().Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "更新后的描述", *after.Description, "掩码内字段 description 应被更新")
	require.Equal(t, "PermSvc 测试权限", *after.Name, "掩码外字段 name 应保持原值")
	require.Equal(t, uint32(7), *after.UpdatedBy, "更新行应盖入操作人 ID")

	_, err = svc.Delete(ctx, &permissionV1.DeletePermissionRequest{
		QueryBy: &permissionV1.DeletePermissionRequest_Id{Id: rowID},
	})
	require.NoError(t, err, "删除应成功")
	cnt, err := client.Permission.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, cnt, "删除后权限表应清空")
}
