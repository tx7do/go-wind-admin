package data

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	entCrud "github.com/tx7do/go-crud/entgo"

	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/rolepermission"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newRolePermissionRepoSqlite 在给定 enttest client 上白盒构造
// RolePermissionRepo，逐字段复刻 NewRolePermissionRepo 的 mapper/converter
// 初始化，再调用 init()。
func newRolePermissionRepoSqlite(t *testing.T, entClient *entCrud.EntClient[*ent.Client]) *RolePermissionRepo {
	t.Helper()
	repo := &RolePermissionRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[permissionV1.RolePermission, ent.RolePermission](),
		statusConverter: mapper.NewEnumTypeConverter[permissionV1.RolePermission_Status, rolepermission.Status](
			permissionV1.RolePermission_Status_name,
			permissionV1.RolePermission_Status_value,
		),
		effectConverter: mapper.NewEnumTypeConverter[permissionV1.RolePermission_EffectiveStatus, rolepermission.Effect](
			permissionV1.RolePermission_EffectiveStatus_name,
			permissionV1.RolePermission_EffectiveStatus_value,
		),
	}
	repo.init()
	return repo
}

// TestRolePermissionRepoSqlite_EnumReadViewBackfill 验证角色权限关联的
// status/effect 枚举字段在 ListPermissionsByRoleID 读视图中的如实回显：
//   - BatchCreate 显式指定的枚举值（行 A/B）按写入值回显；
//   - AssignPermissions 未指定枚举的行（行 C）按列默认（ON/ALLOW）落库并回显。
//
// 历史缺陷：DTO 侧 status/effect 均为可选指针字段，mapper 的枚举转换对
// （值↔值）无法赋入指针字段而直接丢弃，读视图恒呈零值——行内实际存储的
// 生效方式对读方完全不可见；仓内现经 backfillEnumsFrom 统一回填。
func TestRolePermissionRepoSqlite_EnumReadViewBackfill(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newRolePermissionRepoSqlite(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	const roleID int64 = 7001

	tx, err := entClient.Client().Tx(ctx)
	require.NoError(t, err)
	// 无论断言成败都释放事务连接：Commit 成功后 Rollback 返回 ErrTxDone 被忽略；
	// 断言失败（Goexit）时回滚，避免残留写锁阻塞后续测试。
	defer func() { _ = tx.Rollback() }()

	// 行 A/B：经 BatchCreate 显式指定 status/effect 两个枚举的不同取值组合。
	err = repo.BatchCreate(ctx, tx, []*permissionV1.RolePermission{
		{
			RoleId:       trans.Ptr(roleID),
			PermissionId: trans.Ptr(int64(8001)),
			Status:       permissionV1.RolePermission_ON.Enum(),
			Effect:       permissionV1.RolePermission_ALLOW.Enum(),
		},
		{
			RoleId:       trans.Ptr(roleID),
			PermissionId: trans.Ptr(int64(8002)),
			Status:       permissionV1.RolePermission_OFF.Enum(),
			Effect:       permissionV1.RolePermission_DENY.Enum(),
		},
	})
	require.NoError(t, err, "BatchCreate 写入两行显式枚举应成功")

	// 行 C：经 AssignPermissions 只落关联三元组，status/effect 走列默认
	//（ON/ALLOW，由 ent 的列默认机制在插入时应用）。
	err = repo.AssignPermissions(ctx, tx, 0, 1, roleID, []int64{8003})
	require.NoError(t, err, "AssignPermissions 写入默认枚举行应成功")

	require.NoError(t, tx.Commit())

	// 读路径：ListPermissionsByRoleID 返回该角色全部关联行的 DTO 视图。
	rows, err := repo.ListPermissionsByRoleID(ctx, roleID)
	require.NoError(t, err)
	require.Len(t, rows, 3, "该角色应关联 3 行权限")

	// 期望读视图按 permission_id 区分：A/B 为写入值，C 为列默认。
	expect := map[int64]struct {
		status permissionV1.RolePermission_Status
		effect permissionV1.RolePermission_EffectiveStatus
	}{
		8001: {permissionV1.RolePermission_ON, permissionV1.RolePermission_ALLOW},
		8002: {permissionV1.RolePermission_OFF, permissionV1.RolePermission_DENY},
		8003: {permissionV1.RolePermission_ON, permissionV1.RolePermission_ALLOW},
	}
	viewByPermID := map[int64]*permissionV1.RolePermission{}
	for _, row := range rows {
		viewByPermID[row.GetPermissionId()] = row
	}
	require.Len(t, viewByPermID, 3, "读视图应覆盖全部 3 行")
	for permID, exp := range expect {
		row := viewByPermID[permID]
		require.NotNil(t, row, "读视图应包含权限 %d 的关联行", permID)
		require.Equal(t, exp.status, row.GetStatus(), "权限 %d 关联行的读视图应回显 status", permID)
		require.Equal(t, exp.effect, row.GetEffect(), "权限 %d 关联行的读视图应回显 effect", permID)
	}
}


// seedRoleBinding 直建一条角色授权绑定（绕过 service 层），返回绑定行 id。
func seedRoleBinding(t *testing.T, client *ent.Client, ctx context.Context,
	roleID, tenantID, permissionID int64) int {
	t.Helper()
	row, err := client.RolePermission.Create().
		SetRoleID(roleID).
		SetTenantID(tenantID).
		SetPermissionID(permissionID).
		Save(ctx)
	require.NoError(t, err, "直建角色授权绑定应成功")
	return int(row.ID)
}

func countRolePermissionBindingsByPerm(t *testing.T, client *ent.Client, ctx context.Context, permissionID int64) int {
	t.Helper()
	cnt, err := client.RolePermission.Query().
		Where(rolepermission.PermissionIDEQ(permissionID)).
		Count(ctx)
	require.NoError(t, err)
	return cnt
}

// TestRolePermissionRepoSqlite_DeleteByPermissionIDs 验证按 permission_id 集合
// 清理角色授权（权限删除级联的关键一环）：目标权限的绑定全清（跨角色跨租户）、
// 无关权限保留、空集合与重复清理静默成功（幂等删除语义）。
// 回归背景：权限行删除流程（管理页单删/权限同步 MERGE 删除段）此前不清角色
// 授权，角色手里的业务权限授权在权限同步后静默清零（悬空授权）。
func TestRolePermissionRepoSqlite_DeleteByPermissionIDs(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newRolePermissionRepoSqlite(t, entClient)
	client := entClient.Client()
	ctx := enttest.NewSystemContext(context.Background())

	doomed := int64(900)   // 将被级联清理的权限
	survivor := int64(901) // 无关权限（不得被牵连）

	seedRoleBinding(t, client, ctx, 10, 1, doomed)
	seedRoleBinding(t, client, ctx, 11, 2, doomed)
	seedRoleBinding(t, client, ctx, 10, 1, survivor)
	require.Equal(t, 2, countRolePermissionBindingsByPerm(t, client, ctx, doomed))
	require.Equal(t, 1, countRolePermissionBindingsByPerm(t, client, ctx, survivor))

	// 空集合：no-op（重试与并发清理依赖幂等语义）
	require.NoError(t, repo.DeleteByPermissionIDs(ctx, nil), "空集合应静默成功")
	require.Equal(t, 2, countRolePermissionBindingsByPerm(t, client, ctx, doomed), "空集合不得误删")

	// 级联：doomed 的全部绑定清掉（跨角色跨租户）、survivor 保留
	require.NoError(t, repo.DeleteByPermissionIDs(ctx, []int64{doomed}), "按权限集合清理应成功")
	require.Zero(t, countRolePermissionBindingsByPerm(t, client, ctx, doomed), "目标权限的角色授权应全清")
	require.Equal(t, 1, countRolePermissionBindingsByPerm(t, client, ctx, survivor), "无关权限的绑定不得被牵连")

	// 已清理的集合再清一次：静默成功（幂等）
	require.NoError(t, repo.DeleteByPermissionIDs(ctx, []int64{doomed}), "重复清理应静默成功")
}

// TestRolePermissionRepoSqlite_DeleteByPermissionIDs_CrossTenant 防漂移：
// 级联按 permission_id 判定，不受租户隔离影响——权限行删除是全平台事件，
// 任何租户角色手里的该权限授权都必须清（漏清=悬空授权=越权残留）。
func TestRolePermissionRepoSqlite_DeleteByPermissionIDs_CrossTenant(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newRolePermissionRepoSqlite(t, entClient)
	client := entClient.Client()
	ctx := enttest.NewSystemContext(context.Background())

	permID := int64(910)
	tenants := []int64{0, 1, 42, 777}
	for _, tid := range tenants {
		seedRoleBinding(t, client, ctx, 100+tid, tid, permID)
	}

	require.NoError(t, repo.DeleteByPermissionIDs(ctx, []int64{permID}))
	for _, tid := range tenants {
		require.Zero(t, countRolePermissionBindingsByPerm(t, client, ctx, permID),
			"租户 %d 的该权限授权也应被级联清理", tid)
	}
}
