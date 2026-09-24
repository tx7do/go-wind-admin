package data

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
	"go-wind-admin/app/admin/service/internal/data/ent"
	entPermission "go-wind-admin/app/admin/service/internal/data/ent/permission"
	entRole "go-wind-admin/app/admin/service/internal/data/ent/role"
	entRoleMetadata "go-wind-admin/app/admin/service/internal/data/ent/rolemetadata"
	entRolePermission "go-wind-admin/app/admin/service/internal/data/ent/rolepermission"
	"go-wind-admin/app/admin/service/internal/data/enttest"
	"go-wind-admin/pkg/constants"
)

// newRoleRepoSqlite 用 enttest helper 构造一个可直接做 CRUD 的 RoleRepo。
// 白盒构造逐字段复刻 NewRoleRepo 的 mapper/converter 初始化；其五个依赖仓库
// 在生产构造器中由 DI 注入，这里在同一 entclient 上内联构造（各自逐字段复刻
// 生产构造器），保证全部共享同一个 SQLite 内存库。
func newRoleRepoSqlite(t *testing.T) *RoleRepo {
	t.Helper()
	entClient := enttest.NewEntClientForTest(t)

	// 内联依赖 1/5：PermissionApiRepo（复刻 NewPermissionApiRepo）
	permissionApiRepo := &PermissionApiRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
	}

	// 内联依赖 2/5：PermissionMenuRepo（复刻 NewPermissionMenuRepo）
	permissionMenuRepo := &PermissionMenuRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
	}

	// 内联依赖 3/5：PermissionRepo（复刻 NewPermissionRepo 及其 init）
	permissionRepo := &PermissionRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[permissionV1.Permission, ent.Permission](),
		statusConverter: mapper.NewEnumTypeConverter[permissionV1.Permission_Status, entPermission.Status](
			permissionV1.Permission_Status_name,
			permissionV1.Permission_Status_value,
		),
		permissionApiRepo:  permissionApiRepo,
		permissionMenuRepo: permissionMenuRepo,
	}
	permissionRepo.init()

	// 内联依赖 4/5 之一：RolePermissionRepo（复刻 NewRolePermissionRepo 及其 init）
	rolePermissionRepo := &RolePermissionRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[permissionV1.RolePermission, ent.RolePermission](),
		statusConverter: mapper.NewEnumTypeConverter[permissionV1.RolePermission_Status, entRolePermission.Status](
			permissionV1.RolePermission_Status_name,
			permissionV1.RolePermission_Status_value,
		),
		effectConverter: mapper.NewEnumTypeConverter[permissionV1.RolePermission_EffectiveStatus, entRolePermission.Effect](
			permissionV1.RolePermission_EffectiveStatus_name,
			permissionV1.RolePermission_EffectiveStatus_value,
		),
	}
	rolePermissionRepo.init()

	// 内联依赖 4/5 之二：RoleOrgUnitRepo（复刻 NewRoleOrgUnitRepo，仅两个字段）
	roleOrgUnitRepo := &RoleOrgUnitRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
	}

	// 内联依赖 4/5 之三：RoleMetadataRepo（复刻 NewRoleMetadataRepo 及其 init）
	roleMetadataRepo := &RoleMetadataRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[permissionV1.RoleMetadata, ent.RoleMetadata](),
		syncPolicyConverter: mapper.NewEnumTypeConverter[permissionV1.RoleMetadata_SyncPolicy, entRoleMetadata.SyncPolicy](
			permissionV1.RoleMetadata_SyncPolicy_name,
			permissionV1.RoleMetadata_SyncPolicy_value,
		),
		scopeConverter: mapper.NewEnumTypeConverter[permissionV1.RoleMetadata_Scope, entRoleMetadata.Scope](
			permissionV1.RoleMetadata_Scope_name,
			permissionV1.RoleMetadata_Scope_value,
		),
	}
	roleMetadataRepo.init()

	// 内联依赖 5/5：RoleFieldPermissionRepo（复刻 NewRoleFieldPermissionRepo，仅两个字段）
	roleFieldPermissionRepo := &RoleFieldPermissionRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
	}

	repo := &RoleRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[permissionV1.Role, ent.Role](),
		statusConverter: mapper.NewEnumTypeConverter[permissionV1.Role_Status, entRole.Status](
			permissionV1.Role_Status_name,
			permissionV1.Role_Status_value,
		),
		typeConverter: mapper.NewEnumTypeConverter[permissionV1.Role_Type, entRole.Type](
			permissionV1.Role_Type_name,
			permissionV1.Role_Type_value,
		),
		dataScopeConverter: mapper.NewEnumTypeConverter[identityV1.DataScope, entRole.DataScope](
			identityV1.DataScope_name,
			identityV1.DataScope_value,
		),
		permissionRepo:          permissionRepo,
		rolePermissionRepo:      rolePermissionRepo,
		roleOrgUnitRepo:         roleOrgUnitRepo,
		roleMetadataRepo:        roleMetadataRepo,
		roleFieldPermissionRepo: roleFieldPermissionRepo,
	}

	repo.init()

	return repo
}

// TestRoleRepoSqlite_CreateAndMetadataCascade 端到端验证 RoleRepo.Create：
// 角色行落库（含枚举转换）且按类型联动创建角色元数据行（TENANT 类型 → TENANT 作用域、
// 非模板），并用 ent client 直查独立确认。
func TestRoleRepoSqlite_CreateAndMetadataCascade(t *testing.T) {
	repo := newRoleRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	err := repo.Create(ctx, &permissionV1.CreateRoleRequest{
		Data: &permissionV1.Role{
			Name:      trans.Ptr("sqlite测试租户角色"),
			Code:      trans.Ptr(fmt.Sprintf("ROLE_SQLITE_%d", 11001)),
			Status:    permissionV1.Role_ON.Enum(),
			Type:      permissionV1.Role_TENANT.Enum(),
			DataScope: identityV1.DataScope_ALL.Enum(),
		},
	})
	require.NoError(t, err, "通过 repo.Create 写入 SQLite 应成功")

	roles, err := repo.entClient.Client().Role.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, roles, 1, "sys_roles 应有 1 条角色记录")
	roleRow := roles[0]
	require.Equal(t, "sqlite测试租户角色", *roleRow.Name, "name 应按 Create 载荷落库")
	require.Equal(t, fmt.Sprintf("ROLE_SQLITE_%d", 11001), *roleRow.Code, "code 应按 Create 载荷落库")
	require.NotNil(t, roleRow.Status, "status 枚举应经转换器落库")
	require.Equal(t, entRole.StatusOn, *roleRow.Status, "proto Role_ON 应映射为 ent StatusOn")
	require.NotNil(t, roleRow.Type, "type 枚举应经转换器落库")
	require.Equal(t, entRole.TypeTenant, *roleRow.Type, "proto Role_TENANT 应映射为 ent TypeTenant")
	require.NotNil(t, roleRow.DataScope, "data_scope 枚举应经转换器落库")
	require.Equal(t, entRole.DataScopeAll, *roleRow.DataScope, "proto DataScope_ALL 应映射为 ent DataScopeAll")

	// Create 的级联：为该角色创建元数据行（TENANT 类型 → TENANT 作用域、非模板）
	metas, err := repo.entClient.Client().RoleMetadata.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, metas, 1, "sys_role_metadata 应有 1 条联动创建的元数据记录")
	require.Equal(t, roleRow.ID, *metas[0].RoleID, "元数据行应挂到新角色上")
	require.NotNil(t, metas[0].Scope, "scope 枚举应经转换器落库")
	require.Equal(t, entRoleMetadata.ScopeTenant, *metas[0].Scope, "TENANT 类型角色的元数据作用域应为 TENANT")
	require.NotNil(t, metas[0].IsTemplate)
	require.False(t, *metas[0].IsTemplate, "非模板角色创建的元数据应为非模板")
}

// TestRoleRepoSqlite_ListAndGet 验证 RoleRepo.List 的分页/contains 过滤与
// Get 按主键命中、按名称/编码在平台上下文被拒，及按 ID 列表查询。
func TestRoleRepoSqlite_ListAndGet(t *testing.T) {
	repo := newRoleRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &permissionV1.CreateRoleRequest{
		Data: &permissionV1.Role{
			Name:   trans.Ptr("MARKERALPHA 角色"),
			Code:   trans.Ptr(fmt.Sprintf("ROLE_SQLITE_%d", 12001)),
			Status: permissionV1.Role_ON.Enum(),
			Type:   permissionV1.Role_TENANT.Enum(),
		},
	}))
	require.NoError(t, repo.Create(ctx, &permissionV1.CreateRoleRequest{
		Data: &permissionV1.Role{
			Name:   trans.Ptr("MARKERBETA 角色"),
			Code:   trans.Ptr(fmt.Sprintf("ROLE_SQLITE_%d", 12002)),
			Status: permissionV1.Role_ON.Enum(),
			Type:   permissionV1.Role_TENANT.Enum(),
		},
	}))

	roles, err := repo.entClient.Client().Role.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, roles, 2)
	roleAID, roleBID := uint32(roles[0].ID), uint32(roles[1].ID)

	// 无过滤：应返回全部 2 条
	all, err := repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(2), all.Total, "无过滤时应统计全部 2 条")
	require.Len(t, all.Items, 2, "无过滤时应返回 2 条")
	// 列表读视图：status/type 为写入值，data_scope 未显式指定、按列默认（ALL）
	// 落库——三者均经 queryEnumsAndBackfill 如实回显。
	for _, item := range all.Items {
		require.Equal(t, permissionV1.Role_ON, item.GetStatus(), "列表读视图应回填 status")
		require.Equal(t, permissionV1.Role_TENANT, item.GetType(), "列表读视图应回填 type")
		require.Equal(t, identityV1.DataScope_ALL, item.GetDataScope(), "列表读视图应回填列默认 data_scope")
	}

	// contains 过滤：仅命中名称含 MARKERALPHA 的那条
	filtered, err := repo.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_Query{
			Query: `{"name__contains":"MARKERALPHA"}`,
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), filtered.Total, "contains 过滤应只统计命中行")
	require.Len(t, filtered.Items, 1, "contains 过滤应只返回命中行")
	require.Contains(t, filtered.Items[0].GetName(), "MARKERALPHA", "命中行应为含标记的那条")

	// 命中：按主键
	gotByID, err := repo.Get(ctx, &permissionV1.GetRoleRequest{
		QueryBy: &permissionV1.GetRoleRequest_Id{Id: roleAID},
	})
	require.NoError(t, err, "按主键查询已存在记录应命中")
	require.Equal(t, "MARKERALPHA 角色", gotByID.GetName(), "命中记录的 name 应与写入一致")
	// 读视图（Get 按主键）：status/type 为写入值，data_scope 为列默认（ALL），
	// 均经 queryEnumsAndBackfill 如实回显。
	require.Equal(t, permissionV1.Role_ON, gotByID.GetStatus(), "读视图应回填 status")
	require.Equal(t, permissionV1.Role_TENANT, gotByID.GetType(), "读视图应回填 type")
	require.Equal(t, identityV1.DataScope_ALL, gotByID.GetDataScope(), "读视图应回填列默认 data_scope")

	// 未命中：不存在的主键
	_, err = repo.Get(ctx, &permissionV1.GetRoleRequest{
		QueryBy: &permissionV1.GetRoleRequest_Id{Id: 99999},
	})
	require.Error(t, err, "查询不存在的主键应返回错误")

	// 平台上下文按名称/编码查询：跨租户风险，应被租户闸门拒绝
	_, err = repo.Get(ctx, &permissionV1.GetRoleRequest{
		QueryBy: &permissionV1.GetRoleRequest_Name{Name: "MARKERALPHA 角色"},
	})
	require.Error(t, err, "平台上下文按 name 查询角色应被拒绝")
	_, err = repo.Get(ctx, &permissionV1.GetRoleRequest{
		QueryBy: &permissionV1.GetRoleRequest_Code{Code: fmt.Sprintf("ROLE_SQLITE_%d", 12001)},
	})
	require.Error(t, err, "平台上下文按 code 查询角色应被拒绝")

	// 按 ID 列表查询（角色与角色编码）
	byIDs, err := repo.ListRolesByRoleIds(ctx, []uint32{roleAID, roleBID})
	require.NoError(t, err)
	require.Len(t, byIDs, 2, "按 ID 列表应返回两条角色")
	// 读视图（ListXxxByIds 路径）：status/type 为写入值、data_scope 为列默认，
	// 经 backfillEnumsFrom 如实回显。
	for _, item := range byIDs {
		require.Equal(t, permissionV1.Role_ON, item.GetStatus(), "按 ID 列表读视图应回填 status")
		require.Equal(t, permissionV1.Role_TENANT, item.GetType(), "按 ID 列表读视图应回填 type")
		require.Equal(t, identityV1.DataScope_ALL, item.GetDataScope(), "按 ID 列表读视图应回填列默认 data_scope")
	}
	codes, err := repo.ListRoleCodesByRoleIds(ctx, []uint32{roleAID, roleBID})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{
		fmt.Sprintf("ROLE_SQLITE_%d", 12001),
		fmt.Sprintf("ROLE_SQLITE_%d", 12002),
	}, codes, "按 ID 列表应返回两条角色编码")

	// 平台上下文按编码列表查询：应被租户闸门拒绝
	_, err = repo.ListRolesByRoleCodes(ctx, []string{fmt.Sprintf("ROLE_SQLITE_%d", 12001)})
	require.Error(t, err, "平台上下文按编码列表查询角色应被拒绝")
	_, err = repo.ListRoleIDsByRoleCodes(ctx, []string{fmt.Sprintf("ROLE_SQLITE_%d", 12001)})
	require.Error(t, err, "平台上下文按编码列表查询角色 ID 应被拒绝")

	// 无权限关联时的权限 API/菜单 ID 列表应为空
	apiIDs, err := repo.GetRolePermissionApiIDs(ctx, roleAID)
	require.NoError(t, err)
	require.Empty(t, apiIDs, "未关联权限的角色应无 API 资源 ID")
	menuIDs, err := repo.GetRolePermissionMenuIDs(ctx, roleAID)
	require.NoError(t, err)
	require.Empty(t, menuIDs, "未关联权限的角色应无菜单 ID")
}

// TestRoleRepoSqlite_Update 验证 RoleRepo.Update 在单字段 updateMask 下
// 只更新掩码内字段，掩码外字段保持原值。
func TestRoleRepoSqlite_Update(t *testing.T) {
	repo := newRoleRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &permissionV1.CreateRoleRequest{
		Data: &permissionV1.Role{
			Name:   trans.Ptr("更新前角色名"),
			Code:   trans.Ptr(fmt.Sprintf("ROLE_SQLITE_%d", 13001)),
			Status: permissionV1.Role_ON.Enum(),
			Type:   permissionV1.Role_TENANT.Enum(),
		},
	}))

	roles, err := repo.entClient.Client().Role.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, roles, 1)
	createdID := uint32(roles[0].ID)

	err = repo.Update(ctx, &permissionV1.UpdateRoleRequest{
		Id:         createdID,
		Data:       &permissionV1.Role{Name: trans.Ptr("更新后角色名")},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"name"}},
	})
	require.NoError(t, err, "单字段掩码更新应成功")

	after, err := repo.entClient.Client().Role.Query().Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "更新后角色名", *after.Name, "掩码内字段 name 应被更新")
	require.Equal(t, fmt.Sprintf("ROLE_SQLITE_%d", 13001), *after.Code, "掩码外字段 code 应保持原值")
}

// TestRoleRepoSqlite_Delete 验证 RoleRepo.Delete 删除非保护角色后
// 角色表与其元数据、三张关联表一并清空。
func TestRoleRepoSqlite_Delete(t *testing.T) {
	repo := newRoleRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &permissionV1.CreateRoleRequest{
		Data: &permissionV1.Role{
			Name:   trans.Ptr("待删除角色"),
			Code:   trans.Ptr(fmt.Sprintf("ROLE_SQLITE_%d", 14001)),
			Status: permissionV1.Role_ON.Enum(),
			Type:   permissionV1.Role_TENANT.Enum(),
		},
	}))

	roles, err := repo.entClient.Client().Role.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, roles, 1)
	createdID := uint32(roles[0].ID)

	require.NoError(t, repo.Delete(ctx, &permissionV1.DeleteRoleRequest{
		QueryBy: &permissionV1.DeleteRoleRequest_Id{Id: createdID},
	}), "删除非保护角色应成功")

	count, err := repo.entClient.Client().Role.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count, "删除后 sys_roles 计数应归零")

	metaCount, err := repo.entClient.Client().RoleMetadata.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, metaCount, "角色元数据应随角色删除一并清空，不留孤儿行")
}

// TestRoleRepoSqlite_ProtectedDeleteRejected 验证受保护角色禁止删除。
func TestRoleRepoSqlite_ProtectedDeleteRejected(t *testing.T) {
	repo := newRoleRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &permissionV1.CreateRoleRequest{
		Data: &permissionV1.Role{
			Name:        trans.Ptr("受保护角色"),
			Code:        trans.Ptr(fmt.Sprintf("ROLE_SQLITE_%d", 14002)),
			Status:      permissionV1.Role_ON.Enum(),
			Type:        permissionV1.Role_TENANT.Enum(),
			IsProtected: trans.Ptr(true),
		},
	}))

	roles, err := repo.entClient.Client().Role.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, roles, 1)
	createdID := uint32(roles[0].ID)

	require.Error(t, repo.Delete(ctx, &permissionV1.DeleteRoleRequest{
		QueryBy: &permissionV1.DeleteRoleRequest_Id{Id: createdID},
	}), "删除受保护角色应被拒绝")

	count, err := repo.entClient.Client().Role.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count, "被拒绝的删除不应移除角色记录")
}

// TestRoleRepoSqlite_CanAssignRole 验证角色可分配性判定的各分支：
// 启用的普通角色可分配；停用角色、模板角色不可分配。
func TestRoleRepoSqlite_CanAssignRole(t *testing.T) {
	repo := newRoleRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	// 启用的普通租户角色：可分配
	require.NoError(t, repo.Create(ctx, &permissionV1.CreateRoleRequest{
		Data: &permissionV1.Role{
			Name:   trans.Ptr("可分配角色"),
			Code:   trans.Ptr(fmt.Sprintf("ROLE_SQLITE_%d", 15001)),
			Status: permissionV1.Role_ON.Enum(),
			Type:   permissionV1.Role_TENANT.Enum(),
		},
	}))
	roles, err := repo.entClient.Client().Role.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, roles, 1)
	normalRoleID := uint32(roles[0].ID)
	ok, err := repo.CanAssignRole(ctx, normalRoleID)
	require.NoError(t, err)
	require.True(t, ok, "启用且非模板的普通角色应可分配")

	// 停用角色：拒绝
	require.NoError(t, repo.Create(ctx, &permissionV1.CreateRoleRequest{
		Data: &permissionV1.Role{
			Name:   trans.Ptr("停用角色"),
			Code:   trans.Ptr(fmt.Sprintf("ROLE_SQLITE_%d", 15002)),
			Status: permissionV1.Role_OFF.Enum(),
			Type:   permissionV1.Role_TENANT.Enum(),
		},
	}))
	roles, err = repo.entClient.Client().Role.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, roles, 2)
	var disabledRoleID uint32
	for _, r := range roles {
		if r.Status != nil && *r.Status == entRole.StatusOff {
			disabledRoleID = uint32(r.ID)
		}
	}
	require.NotZero(t, disabledRoleID, "应找到停用角色")
	_, err = repo.CanAssignRole(ctx, disabledRoleID)
	require.Error(t, err, "停用角色应不可分配")

	// 模板角色：拒绝（Create 以 TEMPLATE 类型联动创建的元数据带模板标记）
	require.NoError(t, repo.Create(ctx, &permissionV1.CreateRoleRequest{
		Data: &permissionV1.Role{
			Name:   trans.Ptr("模板角色"),
			Code:   trans.Ptr(fmt.Sprintf("ROLE_SQLITE_%d", 15003)),
			Status: permissionV1.Role_ON.Enum(),
			Type:   permissionV1.Role_TEMPLATE.Enum(),
		},
	}))
	roles, err = repo.entClient.Client().Role.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, roles, 3)
	var templateRoleID uint32
	for _, r := range roles {
		if r.Type != nil && *r.Type == entRole.TypeTemplate {
			templateRoleID = uint32(r.ID)
		}
	}
	require.NotZero(t, templateRoleID, "应找到模板角色")
	_, err = repo.CanAssignRole(ctx, templateRoleID)
	require.Error(t, err, "模板角色应不可分配")
}

// TestRoleRepoSqlite_TemplateInstantiation 验证模板角色的专用查询与
// 从模板实例化租户管理员角色的完整链路：模板按（前缀编码、启用、受保护、
// 平台归属）四重条件命中；实例化产物挂到目标租户且编码去掉模板前缀。
func TestRoleRepoSqlite_TemplateInstantiation(t *testing.T) {
	repo := newRoleRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	// 造一个与生产播种同构的租户管理员模板角色
	require.NoError(t, repo.Create(ctx, &permissionV1.CreateRoleRequest{
		Data: &permissionV1.Role{
			Name:        trans.Ptr(constants.DefaultTenantManagerRoleName + "模板"),
			Code:        trans.Ptr(constants.TenantAdminTemplateRoleCode),
			Status:      permissionV1.Role_ON.Enum(),
			Type:        permissionV1.Role_TEMPLATE.Enum(),
			IsProtected: trans.Ptr(true),
		},
	}))

	// 专用查询应命中该模板（四重条件全部满足）
	template, err := repo.GetTemplateRole(ctx, constants.TenantAdminRoleCode)
	require.NoError(t, err, "按模板编码查询已播种模板应命中")
	require.Equal(t, constants.TenantAdminTemplateRoleCode, template.GetCode(), "命中的应为模板角色本体")
	// 读视图（GetTemplateRole 路径）：status/type 为写入值、data_scope 为列默认，
	// 经 queryEnumsAndBackfill 如实回显。
	require.Equal(t, permissionV1.Role_ON, template.GetStatus(), "读视图应回填 status")
	require.Equal(t, permissionV1.Role_TEMPLATE, template.GetType(), "读视图应回填 type")
	require.Equal(t, identityV1.DataScope_ALL, template.GetDataScope(), "读视图应回填列默认 data_scope")

	// 未命中的模板编码应报错
	_, err = repo.GetTemplateRole(ctx, "not_exists_template")
	require.Error(t, err, "查询不存在的模板应返回错误")

	// 空编码直接拒绝
	_, err = repo.GetTemplateRole(ctx, "")
	require.Error(t, err, "空模板编码应直接拒绝")

	// 从模板实例化租户管理员角色（挂到租户 5）
	tx, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	// 无论断言成败都释放事务连接：Commit 成功后 Rollback 返回 ErrTxDone 被忽略；
	// 断言失败（Goexit）时回滚，避免残留写锁阻塞后续测试。
	defer func() { _ = tx.Rollback() }()
	_, err = repo.CreateTenantRoleFromTemplate(ctx, tx, 5, 1)
	require.NoError(t, err, "从模板实例化租户管理员角色应成功")
	require.NoError(t, tx.Commit())

	// 实例化产物：挂到租户 5、编码去前缀、名称为租户管理员默认名、类型 TENANT、受保护
	clones, err := repo.entClient.Client().Role.Query().
		Where(entRole.TenantIDEQ(5)).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, clones, 1, "目标租户下应有一条实例化角色")
	require.Equal(t, constants.TenantAdminRoleCode, *clones[0].Code, "实例化角色编码应为去前缀后的租户管理员编码")
	require.Equal(t, constants.DefaultTenantManagerRoleName, *clones[0].Name, "实例化角色名称应为默认租户管理员名")
	require.Equal(t, entRole.TypeTenant, *clones[0].Type, "实例化角色类型应为 TENANT")

	// 模板本体仍在平台侧：经 repo 创建的行 tenant_id 落默认值 0（列默认，非 NULL）
	platformRoles, err := repo.entClient.Client().Role.Query().
		Where(entRole.TenantIDEQ(0)).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, platformRoles, 1, "平台侧应仍只有模板本体")
}
