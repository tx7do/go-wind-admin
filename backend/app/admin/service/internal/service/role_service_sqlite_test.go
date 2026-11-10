// RoleService 的 SQLite 内存库集成测试（白盒，包内测试）。
//
// 覆盖目标：
//   - List / Get 的 enrichment：TenantName 经 tenantRepo.ListTenantsByIds 从租户表回填；
//     平台级角色（TenantId 未设置）不回填。
//   - Create / Delete 基本路径：Create 走 auth.FromContext 操作人注入（CreatedBy 回填）
//     与默认角色播种；Delete 走非保护角色删除 + 关联清理。
//   - 保留角色码校验：内置身份码（platform:* / template:* / tenant:manager）
//     不可经 Create/Update 占用（防伪装内置身份越权）。
//
// 桩说明：Create/Update/Delete 尾部会调 authorizer.ResetPolicies，nil 会 panic，
// 按 NewAuthorizer 的 noop 引擎形态构造最小可用 authorizer（Provider 桩返回空策略，
// noop 引擎的 ResetPolicies 直接返回 nil）。
package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/enttest"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"

	"github.com/tx7do/go-utils/authorizer"
	"go-wind-admin/pkg/constants"
	"go-wind-admin/pkg/middleware/auth"
)

// roleServiceAuthProviderStub：authorizer.Provider 桩，返回空模型/空策略，
// 仅供 noop 引擎的 ResetPolicies 走通（noop 分支不消费策略数据）。
type roleServiceAuthProviderStub struct{}

func (roleServiceAuthProviderStub) ProvideModels(string) authorizer.ModelDataMap { return nil }

func (roleServiceAuthProviderStub) ProvidePolicies(context.Context) (authorizer.PermissionDataMap, error) {
	return nil, nil
}

// newRoleServiceForTest 白盒复刻 NewRoleService 的字段初始化：
// log 换 NopLogger，repo 用 testkit 构造器，authorizer 用 noop 引擎最小桩；
// 并复刻生产构造器的 svc.init()（空表时播种 constants.DefaultRoles）。
func newRoleServiceForTest(t *testing.T) *RoleService {
	t.Helper()
	entClient := enttest.NewEntClientForTest(t)
	// noop 引擎形态：EngineConfig 空类型按 noop 分支处理，日志用 NopLogger。
	svc := &RoleService{
		log:            bLogger.NewHelper(bLogger.NopLogger()),
		authorizer:     authorizer.NewAuthorizer(context.Background(), bLogger.NopLogger(), &authorizer.EngineConfig{}, roleServiceAuthProviderStub{}),
		roleRepo:       data.NewRoleRepoForTest(entClient),
		tenantRepo:     data.NewTenantRepoForTest(entClient),
		permissionRepo: data.NewPermissionRepoForTest(entClient),
	}
	svc.init()
	return svc
}

// TestRoleServiceSqlite_ListEnrichment 验证 List 的 TenantName 回填：
// 租户级角色回填租户名，平台级角色（默认播种）不回填。
func TestRoleServiceSqlite_ListEnrichment(t *testing.T) {
	svc := newRoleServiceForTest(t)
	ctx := enttest.NewSystemContext(context.Background())

	tenant, err := svc.tenantRepo.Create(ctx, &identityV1.Tenant{
		Name:        trans.Ptr("RoleSvc 富集租户甲"),
		Code:        trans.Ptr("ROLESVC_TENANT_A"),
		Status:      identityV1.Tenant_ON.Enum(),
		Type:        identityV1.Tenant_TRIAL.Enum(),
		AuditStatus: identityV1.Tenant_APPROVED.Enum(),
	})
	require.NoError(t, err)
	require.NotNil(t, tenant.Id)

	err = svc.roleRepo.Create(ctx, &permissionV1.CreateRoleRequest{
		Data: &permissionV1.Role{
			TenantId: tenant.Id,
			Name:     trans.Ptr("RoleSvc 租户角色甲"),
			Code:     trans.Ptr("ROLESVC_ROLE_TENANT_A"),
			Status:   permissionV1.Role_ON.Enum(),
			Type:     permissionV1.Role_TENANT.Enum(),
		},
	})
	require.NoError(t, err)

	resp, err := svc.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)

	var sawTenantRole bool
	var sawPlatformRoleWithoutTenant bool
	for _, item := range resp.GetItems() {
		if item.GetCode() == "ROLESVC_ROLE_TENANT_A" {
			sawTenantRole = true
			require.Equal(t, "RoleSvc 富集租户甲", item.GetTenantName(),
				"租户级角色应回填租户名")
			continue
		}
		// 默认播种的平台级角色（TenantId 未设置）：不应回填租户名。
		if item.GetTenantId() == 0 {
			require.Empty(t, item.GetTenantName(),
				"平台级角色不应回填租户名")
			sawPlatformRoleWithoutTenant = true
		}
	}
	require.True(t, sawTenantRole, "列表应包含新建的租户级角色")
	require.True(t, sawPlatformRoleWithoutTenant, "列表应包含默认播种的平台级角色")
}

// TestRoleServiceSqlite_GetEnrichment 验证 Get 单条查询的 TenantName 回填。
func TestRoleServiceSqlite_GetEnrichment(t *testing.T) {
	svc := newRoleServiceForTest(t)
	ctx := enttest.NewSystemContext(context.Background())

	tenant, err := svc.tenantRepo.Create(ctx, &identityV1.Tenant{
		Name:        trans.Ptr("RoleSvc 富集租户乙"),
		Code:        trans.Ptr("ROLESVC_TENANT_B"),
		Status:      identityV1.Tenant_ON.Enum(),
		Type:        identityV1.Tenant_TRIAL.Enum(),
		AuditStatus: identityV1.Tenant_APPROVED.Enum(),
	})
	require.NoError(t, err)
	require.NotNil(t, tenant.Id)

	err = svc.roleRepo.Create(ctx, &permissionV1.CreateRoleRequest{
		Data: &permissionV1.Role{
			TenantId: tenant.Id,
			Name:     trans.Ptr("RoleSvc 租户角色乙"),
			Code:     trans.Ptr("ROLESVC_ROLE_TENANT_B"),
			Status:   permissionV1.Role_ON.Enum(),
			Type:     permissionV1.Role_TENANT.Enum(),
		},
	})
	require.NoError(t, err)

	// RoleRepo.Create 只返回 error，角色 ID 从列表反查（按唯一 code 定位）。
	listResp, err := svc.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	var roleID int64
	var found bool
	for _, item := range listResp.GetItems() {
		if item.GetCode() == "ROLESVC_ROLE_TENANT_B" {
			roleID = item.GetId()
			found = true
		}
	}
	require.True(t, found, "创建后列表应包含新角色")

	resp, err := svc.Get(ctx, &permissionV1.GetRoleRequest{
		QueryBy: &permissionV1.GetRoleRequest_Id{Id: roleID},
	})
	require.NoError(t, err)
	require.Equal(t, "RoleSvc 租户角色乙", resp.GetName())
	require.Equal(t, "RoleSvc 富集租户乙", resp.GetTenantName(),
		"租户级角色的 Get 应回填租户名")
}

// TestRoleServiceSqlite_CreateAndDelete 验证 Create 的操作人注入与
// Delete 的非保护角色删除路径。
func TestRoleServiceSqlite_CreateAndDelete(t *testing.T) {
	svc := newRoleServiceForTest(t)
	baseCtx := enttest.NewSystemContext(context.Background())
	opCtx := auth.NewContext(baseCtx, &authenticationV1.UserTokenPayload{UserId: 4242, IsTenantAdmin: trans.Ptr(true)})

	// Create：操作人注入 CreatedBy，角色本体与角色元数据同事务落库。
	_, err := svc.Create(opCtx, &permissionV1.CreateRoleRequest{
		Data: &permissionV1.Role{
			Name:   trans.Ptr("RoleSvc 创建角色甲"),
			Code:   trans.Ptr("ROLESVC_CREATE_A"),
			Status: permissionV1.Role_ON.Enum(),
			Type:   permissionV1.Role_TENANT.Enum(),
		},
	})
	require.NoError(t, err, "service.Create 应走完操作人注入与事务创建")

	var createdID int64
	listResp, err := svc.List(baseCtx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	var found bool
	for _, item := range listResp.GetItems() {
		if item.GetCode() == "ROLESVC_CREATE_A" {
			found = true
			createdID = item.GetId()
			require.EqualValues(t, 4242, item.GetCreatedBy(),
				"CreatedBy 应为操作人注入的 ID")
		}
	}
	require.True(t, found, "创建后列表应包含新角色")

	// Delete：非保护角色应删除成功（含关联行清理）。
	_, err = svc.Delete(opCtx, &permissionV1.DeleteRoleRequest{
		QueryBy: &permissionV1.DeleteRoleRequest_Id{Id: createdID},
	})
	require.NoError(t, err, "非保护角色的 service.Delete 应成功")

	after, err := svc.List(baseCtx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	for _, item := range after.GetItems() {
		require.NotEqual(t, "ROLESVC_CREATE_A", item.GetCode(),
			"删除后列表不应再包含该角色")
	}
}

// createServiceRoleForTest 经 svc.Create 建角色并反查 ID（按唯一 code 定位）。
// operatorCtx 走平台操作人（tenant 0），角色默认落平台侧。
func createServiceRoleForTest(t *testing.T, svc *RoleService, opCtx context.Context, code, name string, parentID int64, perms []int64) int64 {
	t.Helper()
	data := &permissionV1.Role{
		Name:   trans.Ptr(name),
		Code:   trans.Ptr(code),
		Status: permissionV1.Role_ON.Enum(),
		Type:   permissionV1.Role_TENANT.Enum(),
	}
	if parentID > 0 {
		data.ParentId = trans.Ptr(parentID)
	}
	if len(perms) > 0 {
		data.Permissions = perms
	}
	_, err := svc.Create(opCtx, &permissionV1.CreateRoleRequest{Data: data})
	require.NoError(t, err, "创建角色 %s 应成功", code)

	listResp, err := svc.List(enttest.NewSystemContext(context.Background()), &paginationV1.PagingRequest{})
	require.NoError(t, err)
	for _, item := range listResp.GetItems() {
		if item.GetCode() == code {
			return item.GetId()
		}
	}
	t.Fatalf("创建后列表应包含角色 %s", code)
	return 0
}

// TestRoleServiceSqlite_ParentBindingValidation 验证权限继承核心约束（编辑时校验）：
// 子角色权限 ⊆ 父角色已绑定权限集，越界被拒、子集放行、父不存在被拒。
func TestRoleServiceSqlite_ParentBindingValidation(t *testing.T) {
	svc := newRoleServiceForTest(t)
	baseCtx := enttest.NewSystemContext(context.Background())
	opCtx := auth.NewContext(baseCtx, &authenticationV1.UserTokenPayload{UserId: 4242, IsTenantAdmin: trans.Ptr(true)})

	parentID := createServiceRoleForTest(t, svc, opCtx, "ROLESVC_PB_PARENT", "继承父角色", 0, []int64{101, 102})

	// 子集：放行
	childID := createServiceRoleForTest(t, svc, opCtx, "ROLESVC_PB_CHILD_OK", "合法子角色", parentID, []int64{101})
	require.NotZero(t, childID)

	// 越界：拒绝
	_, err := svc.Create(opCtx, &permissionV1.CreateRoleRequest{
		Data: &permissionV1.Role{
			Name:        trans.Ptr("越界子角色"),
			Code:        trans.Ptr("ROLESVC_PB_CHILD_BAD"),
			Status:      permissionV1.Role_ON.Enum(),
			Type:        permissionV1.Role_TENANT.Enum(),
			ParentId:    trans.Ptr(parentID),
			Permissions: []int64{101, 999},
		},
	})
	require.Error(t, err, "子角色权限超出父角色应被拒绝")
	require.Contains(t, err.Error(), "exceed", "越界错误应包含超集提示")

	// 父角色不存在：拒绝
	_, err = svc.Create(opCtx, &permissionV1.CreateRoleRequest{
		Data: &permissionV1.Role{
			Name:     trans.Ptr("孤儿父引用角色"),
			Code:     trans.Ptr("ROLESVC_PB_ORPHAN"),
			Status:   permissionV1.Role_ON.Enum(),
			Type:     permissionV1.Role_TENANT.Enum(),
			ParentId: trans.Ptr(int64(999999)),
		},
	})
	require.Error(t, err, "父角色不存在应被拒绝")
	require.Contains(t, err.Error(), "parent role not found")
}

// TestRoleServiceSqlite_UpdateParentCycleAndSubset 验证 Update 侧的环检测与
// 换父/改权限两个维度上的约束重新对齐。
func TestRoleServiceSqlite_UpdateParentCycleAndSubset(t *testing.T) {
	svc := newRoleServiceForTest(t)
	baseCtx := enttest.NewSystemContext(context.Background())
	opCtx := auth.NewContext(baseCtx, &authenticationV1.UserTokenPayload{UserId: 4242, IsTenantAdmin: trans.Ptr(true)})

	roleAID := createServiceRoleForTest(t, svc, opCtx, "ROLESVC_CYC_A", "环检测角色甲", 0, []int64{101, 102})
	roleBID := createServiceRoleForTest(t, svc, opCtx, "ROLESVC_CYC_B", "环检测角色乙", 0, []int64{101})
	roleCID := createServiceRoleForTest(t, svc, opCtx, "ROLESVC_CYC_C", "环检测角色丙", 0, []int64{101, 102})
	roleDID := createServiceRoleForTest(t, svc, opCtx, "ROLESVC_CYC_D", "环检测角色丁", 0, nil)

	var err error

	// 甲 → 丙：合法（甲 [101,102] ⊆ 丙 [101,102]）
	_, err = svc.Update(opCtx, &permissionV1.UpdateRoleRequest{
		Id:         roleAID,
		Data:       &permissionV1.Role{ParentId: trans.Ptr(roleCID)},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"parent_id"}},
	})
	require.NoError(t, err, "设置无环父角色应成功")

	// 乙 → 甲：合法（乙 [101] ⊆ 甲 [101,102]）
	_, err = svc.Update(opCtx, &permissionV1.UpdateRoleRequest{
		Id:         roleBID,
		Data:       &permissionV1.Role{ParentId: trans.Ptr(roleAID)},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"parent_id"}},
	})
	require.NoError(t, err, "设置无环父子链应成功")

	// 甲 → 乙：成环——此时甲的父是丙、乙的父是甲；把甲的父改为乙会形成
	// 甲→乙→甲 环，应拒绝（环检测先于子集校验触发）
	_, err = svc.Update(opCtx, &permissionV1.UpdateRoleRequest{
		Id:         roleAID,
		Data:       &permissionV1.Role{ParentId: trans.Ptr(roleBID)},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"parent_id"}},
	})
	require.Error(t, err, "换父成环应被拒绝")
	require.Contains(t, err.Error(), "circular")

	// 自环：把乙的父设为乙自己，应拒绝
	_, err = svc.Update(opCtx, &permissionV1.UpdateRoleRequest{
		Id:         roleBID,
		Data:       &permissionV1.Role{ParentId: trans.Ptr(roleBID)},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"parent_id"}},
	})
	require.Error(t, err, "自引用成环应被拒绝")

	// 改权限越界：乙的父是甲（cap=101,102），改权限为 [101,999] 应拒绝
	_, err = svc.Update(opCtx, &permissionV1.UpdateRoleRequest{
		Id:         roleBID,
		Data:       &permissionV1.Role{Permissions: []int64{101, 999}},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"permissions"}},
	})
	require.Error(t, err, "子角色改权限越出父角色上限应被拒绝")

	// 改权限子集：应放行
	_, err = svc.Update(opCtx, &permissionV1.UpdateRoleRequest{
		Id:         roleBID,
		Data:       &permissionV1.Role{Permissions: []int64{101}},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"permissions"}},
	})
	require.NoError(t, err, "子角色改权限仍在父角色上限内应放行")

	// 换更严格的父不改权限：乙权限 [101]，换父到丁（cap 空）应拒绝
	_, err = svc.Update(opCtx, &permissionV1.UpdateRoleRequest{
		Id:         roleBID,
		Data:       &permissionV1.Role{ParentId: trans.Ptr(roleDID)},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"parent_id"}},
	})
	require.Error(t, err, "换父后存量权限越出父角色上限应被拒绝")
}

// TestRoleServiceSqlite_DeleteParentDetachesChildren 验证删除父角色后子角色
// 自动解除父子关系且绑定权限保持不变。
func TestRoleServiceSqlite_DeleteParentDetachesChildren(t *testing.T) {
	svc := newRoleServiceForTest(t)
	baseCtx := enttest.NewSystemContext(context.Background())
	opCtx := auth.NewContext(baseCtx, &authenticationV1.UserTokenPayload{UserId: 4242, IsTenantAdmin: trans.Ptr(true)})

	parentID := createServiceRoleForTest(t, svc, opCtx, "ROLESVC_DET_PARENT", "待删除父角色", 0, []int64{101, 102})
	childID := createServiceRoleForTest(t, svc, opCtx, "ROLESVC_DET_CHILD", "挂父子角色", parentID, []int64{101})

	_, err := svc.Delete(opCtx, &permissionV1.DeleteRoleRequest{
		QueryBy: &permissionV1.DeleteRoleRequest_Id{Id: parentID},
	})
	require.NoError(t, err, "删除有子角色的父角色应成功（自动解除）")

	after, err := svc.Get(baseCtx, &permissionV1.GetRoleRequest{
		QueryBy: &permissionV1.GetRoleRequest_Id{Id: childID},
	})
	require.NoError(t, err, "子角色应仍可查询")
	require.Equal(t, int64(0), after.GetParentId(), "删除父角色后子角色 parent_id 应解除为 0")
	require.Empty(t, after.GetParentName(), "解除后不应再回填父角色名")
	require.ElementsMatch(t, []int64{101}, after.GetPermissions(), "子角色绑定权限应保持不变")
}

// TestRoleServiceSqlite_ParentNameEnrichment 验证 List / Get 的 ParentName 回填。
func TestRoleServiceSqlite_ParentNameEnrichment(t *testing.T) {
	svc := newRoleServiceForTest(t)
	baseCtx := enttest.NewSystemContext(context.Background())
	opCtx := auth.NewContext(baseCtx, &authenticationV1.UserTokenPayload{UserId: 4242, IsTenantAdmin: trans.Ptr(true)})

	parentID := createServiceRoleForTest(t, svc, opCtx, "ROLESVC_NAME_PARENT", "富集父角色名", 0, nil)
	childID := createServiceRoleForTest(t, svc, opCtx, "ROLESVC_NAME_CHILD", "富集子角色", parentID, nil)

	listResp, err := svc.List(baseCtx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	var sawChild bool
	for _, item := range listResp.GetItems() {
		if item.GetCode() == "ROLESVC_NAME_CHILD" {
			sawChild = true
			require.Equal(t, "富集父角色名", item.GetParentName(),
				"列表应回填子角色的父角色名")
		}
	}
	require.True(t, sawChild, "列表应包含子角色")

	got, err := svc.Get(baseCtx, &permissionV1.GetRoleRequest{
		QueryBy: &permissionV1.GetRoleRequest_Id{Id: childID},
	})
	require.NoError(t, err)
	require.Equal(t, "富集父角色名", got.GetParentName(),
		"Get 应回填子角色的父角色名")
}

// TestRoleServiceSqlite_ReservedRoleCodeRejected 验证角色码保留字校验：
// 内置身份码（platform:* / template:* / tenant:manager）不可经 Create/Update
// API 占用——登录链路按码值精确匹配填充管理员标志，自建同码角色即可伪装
// 内置身份越权；普通码（含 tenant: 前缀的非内置值）不受影响。
func TestRoleServiceSqlite_ReservedRoleCodeRejected(t *testing.T) {
	svc := newRoleServiceForTest(t)
	baseCtx := enttest.NewSystemContext(context.Background())
	opCtx := auth.NewContext(baseCtx, &authenticationV1.UserTokenPayload{UserId: 4242, IsTenantAdmin: trans.Ptr(true)})

	// Create：全部保留码形态被拒。
	reservedCodes := []string{
		constants.PlatformAdminRoleCode,      // platform:admin
		constants.TenantAdminRoleCode,        // tenant:manager
		constants.TenantAdminTemplateRoleCode, // template:tenant:manager
		"platform:custom",                    // 保留命名空间
		"template:foo",                       // 保留命名空间
	}
	for _, code := range reservedCodes {
		_, err := svc.Create(opCtx, &permissionV1.CreateRoleRequest{
			Data: &permissionV1.Role{
				Name:   trans.Ptr("保留码角色 " + code),
				Code:   trans.Ptr(code),
				Status: permissionV1.Role_ON.Enum(),
				Type:   permissionV1.Role_TENANT.Enum(),
			},
		})
		require.Error(t, err, "保留角色码 %s 的 Create 应被拒绝", code)
	}

	// Create：非内置的 tenant: 前缀普通码不受影响。
	_, err := svc.Create(opCtx, &permissionV1.CreateRoleRequest{
		Data: &permissionV1.Role{
			Name:   trans.Ptr("普通租户角色"),
			Code:   trans.Ptr("tenant:sales"),
			Status: permissionV1.Role_ON.Enum(),
			Type:   permissionV1.Role_TENANT.Enum(),
		},
	})
	require.NoError(t, err, "非内置的 tenant:sales 应可正常创建")

	// Update：把普通角色 code 改成保留码被拒。
	listResp, err := svc.List(baseCtx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	var targetID int64
	for _, item := range listResp.GetItems() {
		if item.GetCode() == "tenant:sales" {
			targetID = item.GetId()
		}
	}
	require.NotZero(t, targetID, "应能反查到 tenant:sales 角色")

	mask, err := fieldmaskpb.New(&permissionV1.Role{}, "code")
	require.NoError(t, err)
	_, err = svc.Update(opCtx, &permissionV1.UpdateRoleRequest{
		Id: targetID,
		Data: &permissionV1.Role{
			Code: trans.Ptr(constants.PlatformAdminRoleCode),
		},
		UpdateMask: mask,
	})
	require.Error(t, err, "把已有角色 code 改成保留码的 Update 应被拒绝")

	// Update：改成普通码不受影响。
	_, err = svc.Update(opCtx, &permissionV1.UpdateRoleRequest{
		Id: targetID,
		Data: &permissionV1.Role{
			Code: trans.Ptr("tenant:sales2"),
		},
		UpdateMask: mask,
	})
	require.NoError(t, err, "把 code 改成普通码的 Update 应成功")
}
