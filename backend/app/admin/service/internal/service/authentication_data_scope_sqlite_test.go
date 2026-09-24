// authentication_data_scope.go 的 SQLite 内存库集成测试（白盒，包内测试）。
//
// 覆盖目标（aggregateDataScopes 各分支）：
//   - 平台上下文（tenantID==0）：整体 [ALL] 且单值镜像 ALL。
//   - 任一角色 ALL → [ALL] + 镜像 ALL（后续角色不再扫描）。
//   - 仅 SELF → [SELF] + 镜像 SELF。
//   - 无匹配角色 / 全 UNSPECIFIED：退化 [UNSPECIFIED]，无镜像、无单元目标集。
//   - UNIT_ONLY：自身单元集经租户过滤（跨租户单元被剔除）后 [UNIT_ONLY]，
//     无单值镜像。
//   - UNIT_AND_CHILD：单元集经后代展开（路径前缀）纳入子单元。
//   - SELECTED_UNITS：角色配置单元集经二次租户过滤（纵深防御）。
//   - SELF 与 UNIT_ONLY 并存：活动类型取并集，无单值镜像。
//
// 跳过项：>256 单元目标集上限分支（需 257 个租户内单元行，此批不覆盖）。
package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/ent"
	entOrgUnit "go-wind-admin/app/admin/service/internal/data/ent/orgunit"
	entRole "go-wind-admin/app/admin/service/internal/data/ent/role"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// dataScopeUserRepoStub 是 data.UserRepo 的本地桩：
// 嵌入接口获得默认方法集（未覆写方法一旦被调用即 nil 接口 panic），
// 只覆写 aggregateDataScopes 用到的 ListOrgUnitIDsByUserID（返回预置自身单元集）。
type dataScopeUserRepoStub struct {
	data.UserRepo
	ownUnits []uint32
}

func (s *dataScopeUserRepoStub) ListOrgUnitIDsByUserID(_ context.Context, _ uint32) ([]uint32, error) {
	return s.ownUnits, nil
}

// newDataScopeAuthServiceForTest 白盒构造 AuthenticationService 的
// aggregateDataScopes 相关字段：log 用 NopLogger helper；roleRepo/orgUnitRepo
// 走 testkit；userRepo 用本地桩；roleOrgUnitRepo 无 testkit 导出，按生产构造器 +
// NopLogger 的 bootstrap 上下文构造（与生产构造器逐字段一致，唯一差异是日志）。
func newDataScopeAuthServiceForTest(t *testing.T, ownUnits []uint32) (*AuthenticationService, *ent.Client) {
	t.Helper()
	entClient := enttest.NewEntClientForTest(t)
	bootstrapCtx := bootstrap.NewContextWithParam(context.Background(), nil, nil, bLogger.NopLogger())
	svc := &AuthenticationService{
		log:             bLogger.NewHelper(bLogger.NopLogger()),
		userRepo:        &dataScopeUserRepoStub{ownUnits: ownUnits},
		roleRepo:        data.NewRoleRepoForTest(entClient),
		orgUnitRepo:     data.NewOrgUnitRepoForTest(entClient),
		roleOrgUnitRepo: data.NewRoleOrgUnitRepo(bootstrapCtx, entClient),
	}
	return svc, entClient.Client()
}

// seedScopedRole 落一条带数据范围的角色并返回其 ID（按唯一 code 反查）。
func seedScopedRole(t *testing.T, client *ent.Client, ctx context.Context, svc *AuthenticationService, code string, scope identityV1.DataScope) uint32 {
	t.Helper()
	require.NoError(t, svc.roleRepo.Create(ctx, &permissionV1.CreateRoleRequest{
		Data: &permissionV1.Role{
			TenantId:  trans.Ptr(uint32(42)),
			Name:      trans.Ptr("数据范围测试角色 " + code),
			Code:      trans.Ptr(code),
			Status:    permissionV1.Role_ON.Enum(),
			Type:      permissionV1.Role_TENANT.Enum(),
			DataScope: scope.Enum(),
		},
	}))
	row, err := client.Role.Query().Where(entRole.CodeEQ(code)).Only(ctx)
	require.NoError(t, err, "按 code 反查刚创建的角色应命中")
	return row.ID
}

// seedOrgUnit 落一条组织单元（指定租户；parent>0 时挂父节点）并返回其 ID。
func seedOrgUnit(t *testing.T, client *ent.Client, ctx context.Context, svc *AuthenticationService, tenantID uint32, name, code string, parent uint32) uint32 {
	t.Helper()
	req := &identityV1.CreateOrgUnitRequest{
		Data: &identityV1.OrgUnit{
			TenantId: trans.Ptr(tenantID),
			Name:     trans.Ptr(name),
			Code:     trans.Ptr(code),
			Status:   identityV1.OrgUnit_ON.Enum(),
			Type:     identityV1.OrgUnit_COMPANY.Enum(),
		},
	}
	if parent > 0 {
		req.Data.ParentId = trans.Ptr(parent)
	}
	require.NoError(t, svc.orgUnitRepo.Create(ctx, req))
	row, err := client.OrgUnit.Query().Where(entOrgUnit.CodeEQ(code)).Only(ctx)
	require.NoError(t, err, "按 code 反查刚创建的组织单元应命中")
	return row.ID
}

// TestAggregateDataScopes_PlatformContextIsAll 平台上下文：整体 [ALL] 且单值镜像 ALL。
func TestAggregateDataScopes_PlatformContextIsAll(t *testing.T) {
	svc, _ := newDataScopeAuthServiceForTest(t, nil)
	ctx := enttest.NewSystemViewerCtx(context.Background())
	payload := &authenticationV1.UserTokenPayload{}

	svc.aggregateDataScopes(ctx, 0, 1, nil, payload)

	require.Equal(t, []identityV1.DataScope{identityV1.DataScope_ALL}, payload.GetDataScopes(),
		"平台上下文应整体 [ALL]")
	require.Equal(t, identityV1.DataScope_ALL, payload.GetDataScope(), "单值镜像应为 ALL")
	require.Empty(t, payload.GetDataScopeUnitIds(), "平台上下文不应有单元目标集")
}

// TestAggregateDataScopes_AnyRoleAllShortCircuits 任一角色 ALL → [ALL] + 镜像，
// 其余角色不再扫描。
func TestAggregateDataScopes_AnyRoleAllShortCircuits(t *testing.T) {
	svc, client := newDataScopeAuthServiceForTest(t, nil)
	ctx := enttest.NewSystemViewerCtx(context.Background())
	allID := seedScopedRole(t, client, ctx, svc, "DS_ALL_ROLE", identityV1.DataScope_ALL)
	selfID := seedScopedRole(t, client, ctx, svc, "DS_SELF_ROLE", identityV1.DataScope_SELF)
	payload := &authenticationV1.UserTokenPayload{}

	svc.aggregateDataScopes(ctx, 42, 9, []uint32{allID, selfID}, payload)

	require.Equal(t, []identityV1.DataScope{identityV1.DataScope_ALL}, payload.GetDataScopes(),
		"任一角色 ALL 应整体 [ALL]")
	require.Equal(t, identityV1.DataScope_ALL, payload.GetDataScope(), "单值镜像应为 ALL")
	require.Empty(t, payload.GetDataScopeUnitIds())
}

// TestAggregateDataScopes_SelfOnlyMirror 仅 SELF → [SELF] + 镜像 SELF。
func TestAggregateDataScopes_SelfOnlyMirror(t *testing.T) {
	svc, client := newDataScopeAuthServiceForTest(t, nil)
	ctx := enttest.NewSystemViewerCtx(context.Background())
	selfID := seedScopedRole(t, client, ctx, svc, "DS_SELF_ONLY_ROLE", identityV1.DataScope_SELF)
	payload := &authenticationV1.UserTokenPayload{}

	svc.aggregateDataScopes(ctx, 42, 9, []uint32{selfID}, payload)

	require.Equal(t, []identityV1.DataScope{identityV1.DataScope_SELF}, payload.GetDataScopes(),
		"仅 SELF 角色应为 [SELF]")
	require.Equal(t, identityV1.DataScope_SELF, payload.GetDataScope(), "单值镜像应为 SELF")
	require.Empty(t, payload.GetDataScopeUnitIds(), "SELF 不产生单元目标集")
}

// TestAggregateDataScopes_Degenerate 无匹配角色 / 令牌无角色声明：
// 退化 [UNSPECIFIED]，无镜像、无单元目标集。
func TestAggregateDataScopes_Degenerate(t *testing.T) {
	svc, _ := newDataScopeAuthServiceForTest(t, nil)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	for _, roleIDs := range [][]uint32{nil, {777777}} {
		payload := &authenticationV1.UserTokenPayload{}
		svc.aggregateDataScopes(ctx, 42, 9, roleIDs, payload)
		require.Equal(t, []identityV1.DataScope{identityV1.DataScope_DATA_SCOPE_UNSPECIFIED},
			payload.GetDataScopes(), "无有效角色配置应退化 [UNSPECIFIED]")
		// 单值镜像字段未写入时 getter 返回枚举零值（UNSPECIFIED），即"无镜像"
		require.Equal(t, identityV1.DataScope_DATA_SCOPE_UNSPECIFIED, payload.GetDataScope(),
			"退化不写单值镜像（getter 返回零值）")
		require.Empty(t, payload.GetDataScopeUnitIds(), "退化不写单元目标集")
	}
}

// TestAggregateDataScopes_UnitOnlyTenantFiltered UNIT_ONLY：
// 自身单元集（含跨租户单元）经租户过滤后仅剩本租户单元。
func TestAggregateDataScopes_UnitOnlyTenantFiltered(t *testing.T) {
	svc, client := newDataScopeAuthServiceForTest(t, []uint32{1, 2}) // 占位，稍后以真实 ID 重设
	ctx := enttest.NewSystemViewerCtx(context.Background())
	inTenant := seedOrgUnit(t, client, ctx, svc, 42, "数据范围本租户单元", "DS_ORG_INTENANT", 0)
	outTenant := seedOrgUnit(t, client, ctx, svc, 999, "数据范围他租户单元", "DS_ORG_OUTTENANT", 0)
	svc.userRepo.(*dataScopeUserRepoStub).ownUnits = []uint32{inTenant, outTenant}

	unitRoleID := seedScopedRole(t, client, ctx, svc, "DS_UNIT_ONLY_ROLE", identityV1.DataScope_UNIT_ONLY)
	payload := &authenticationV1.UserTokenPayload{}
	svc.aggregateDataScopes(ctx, 42, 9, []uint32{unitRoleID}, payload)

	require.Equal(t, []identityV1.DataScope{identityV1.DataScope_UNIT_ONLY}, payload.GetDataScopes(),
		"UNIT_ONLY 角色应得 [UNIT_ONLY]")
	// UNIT 类无法用单值无损表达，不写镜像（getter 返回零值）
	require.Equal(t, identityV1.DataScope_DATA_SCOPE_UNSPECIFIED, payload.GetDataScope(),
		"UNIT_ONLY 不写单值镜像（无法无损表达）")
	require.Equal(t, []uint64{uint64(inTenant)}, payload.GetDataScopeUnitIds(),
		"单元目标集应仅含本租户单元（跨租户单元被显式租户谓词剔除）")
}

// TestAggregateDataScopes_UnitAndChildDescendants UNIT_AND_CHILD：
// 后代展开（物化路径前缀）把本租户子单元纳入目标集。
func TestAggregateDataScopes_UnitAndChildDescendants(t *testing.T) {
	svc, client := newDataScopeAuthServiceForTest(t, nil)
	ctx := enttest.NewSystemViewerCtx(context.Background())
	root := seedOrgUnit(t, client, ctx, svc, 42, "数据范围根单元", "DS_ORG_ROOT", 0)
	child := seedOrgUnit(t, client, ctx, svc, 42, "数据范围子单元", "DS_ORG_CHILD", root)
	svc.userRepo.(*dataScopeUserRepoStub).ownUnits = []uint32{root}

	unitRoleID := seedScopedRole(t, client, ctx, svc, "DS_UNIT_CHILD_ROLE", identityV1.DataScope_UNIT_AND_CHILD)
	payload := &authenticationV1.UserTokenPayload{}
	svc.aggregateDataScopes(ctx, 42, 9, []uint32{unitRoleID}, payload)

	require.Equal(t, []identityV1.DataScope{identityV1.DataScope_UNIT_AND_CHILD}, payload.GetDataScopes(),
		"UNIT_AND_CHILD 角色应得 [UNIT_AND_CHILD]")
	require.Equal(t, identityV1.DataScope_DATA_SCOPE_UNSPECIFIED, payload.GetDataScope(),
		"UNIT 类不写单值镜像（getter 返回零值）")
	require.ElementsMatch(t,
		[]uint64{uint64(root), uint64(child)}, payload.GetDataScopeUnitIds(),
		"后代展开应把本租户根与子单元一并纳入目标集")
}

// TestAggregateDataScopes_SelectedUnitsSecondFilter SELECTED_UNITS：
// 角色配置的单元集（含跨租户单元）经二次租户过滤（纵深防御）。
func TestAggregateDataScopes_SelectedUnitsSecondFilter(t *testing.T) {
	svc, client := newDataScopeAuthServiceForTest(t, nil)
	ctx := enttest.NewSystemViewerCtx(context.Background())
	inTenant := seedOrgUnit(t, client, ctx, svc, 42, "选定单元本租户", "DS_ORG_SEL_INTENANT", 0)
	outTenant := seedOrgUnit(t, client, ctx, svc, 999, "选定单元他租户", "DS_ORG_SEL_OUTTENANT", 0)
	selRoleID := seedScopedRole(t, client, ctx, svc, "DS_SELECTED_ROLE", identityV1.DataScope_SELECTED_UNITS)

	// 角色配置单元集：本租户一条 + 跨租户一条（配置侧本应校验同租户，这里
	// 直接落关联行以验证执行侧二次过滤的纵深防御）
	_, err := client.RoleOrgUnit.Create().
		SetRoleID(selRoleID).
		SetOrgUnitID(inTenant).
		Save(ctx)
	require.NoError(t, err)
	_, err = client.RoleOrgUnit.Create().
		SetRoleID(selRoleID).
		SetOrgUnitID(outTenant).
		Save(ctx)
	require.NoError(t, err)

	payload := &authenticationV1.UserTokenPayload{}
	svc.aggregateDataScopes(ctx, 42, 9, []uint32{selRoleID}, payload)

	require.Equal(t, []identityV1.DataScope{identityV1.DataScope_SELECTED_UNITS}, payload.GetDataScopes(),
		"SELECTED_UNITS 角色应得 [SELECTED_UNITS]")
	require.Equal(t, identityV1.DataScope_DATA_SCOPE_UNSPECIFIED, payload.GetDataScope(),
		"UNIT 类不写单值镜像（getter 返回零值）")
	require.Equal(t, []uint64{uint64(inTenant)}, payload.GetDataScopeUnitIds(),
		"配置集应经二次租户过滤，跨租户单元被剔除")
}

// TestAggregateDataScopes_SelfPlusUnitUnion SELF 与 UNIT_ONLY 并存：
// 活动类型取并集，无单值镜像。
func TestAggregateDataScopes_SelfPlusUnitUnion(t *testing.T) {
	svc, client := newDataScopeAuthServiceForTest(t, nil)
	ctx := enttest.NewSystemViewerCtx(context.Background())
	inTenant := seedOrgUnit(t, client, ctx, svc, 42, "并集测试本租户单元", "DS_ORG_UNION_INTENANT", 0)
	svc.userRepo.(*dataScopeUserRepoStub).ownUnits = []uint32{inTenant}

	selfID := seedScopedRole(t, client, ctx, svc, "DS_UNION_SELF_ROLE", identityV1.DataScope_SELF)
	unitID := seedScopedRole(t, client, ctx, svc, "DS_UNION_UNIT_ROLE", identityV1.DataScope_UNIT_ONLY)
	payload := &authenticationV1.UserTokenPayload{}
	svc.aggregateDataScopes(ctx, 42, 9, []uint32{selfID, unitID}, payload)

	require.ElementsMatch(t,
		[]identityV1.DataScope{identityV1.DataScope_SELF, identityV1.DataScope_UNIT_ONLY},
		payload.GetDataScopes(), "SELF 与 UNIT_ONLY 并存应取并集")
	require.Equal(t, identityV1.DataScope_DATA_SCOPE_UNSPECIFIED, payload.GetDataScope(),
		"多活动类型不写单值镜像（getter 返回零值）")
	require.Equal(t, []uint64{uint64(inTenant)}, payload.GetDataScopeUnitIds())
}
