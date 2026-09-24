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
	ctx := enttest.NewSystemViewerCtx(context.Background())

	const roleID uint32 = 7001

	tx, err := entClient.Client().Tx(ctx)
	require.NoError(t, err)
	// 无论断言成败都释放事务连接：Commit 成功后 Rollback 返回 ErrTxDone 被忽略；
	// 断言失败（Goexit）时回滚，避免残留写锁阻塞后续测试。
	defer func() { _ = tx.Rollback() }()

	// 行 A/B：经 BatchCreate 显式指定 status/effect 两个枚举的不同取值组合。
	err = repo.BatchCreate(ctx, tx, []*permissionV1.RolePermission{
		{
			RoleId:       trans.Ptr(roleID),
			PermissionId: trans.Ptr(uint32(8001)),
			Status:       permissionV1.RolePermission_ON.Enum(),
			Effect:       permissionV1.RolePermission_ALLOW.Enum(),
		},
		{
			RoleId:       trans.Ptr(roleID),
			PermissionId: trans.Ptr(uint32(8002)),
			Status:       permissionV1.RolePermission_OFF.Enum(),
			Effect:       permissionV1.RolePermission_DENY.Enum(),
		},
	})
	require.NoError(t, err, "BatchCreate 写入两行显式枚举应成功")

	// 行 C：经 AssignPermissions 只落关联三元组，status/effect 走列默认
	//（ON/ALLOW，由 ent 的列默认机制在插入时应用）。
	err = repo.AssignPermissions(ctx, tx, 0, 1, roleID, []uint32{8003})
	require.NoError(t, err, "AssignPermissions 写入默认枚举行应成功")

	require.NoError(t, tx.Commit())

	// 读路径：ListPermissionsByRoleID 返回该角色全部关联行的 DTO 视图。
	rows, err := repo.ListPermissionsByRoleID(ctx, roleID)
	require.NoError(t, err)
	require.Len(t, rows, 3, "该角色应关联 3 行权限")

	// 期望读视图按 permission_id 区分：A/B 为写入值，C 为列默认。
	expect := map[uint32]struct {
		status permissionV1.RolePermission_Status
		effect permissionV1.RolePermission_EffectiveStatus
	}{
		8001: {permissionV1.RolePermission_ON, permissionV1.RolePermission_ALLOW},
		8002: {permissionV1.RolePermission_OFF, permissionV1.RolePermission_DENY},
		8003: {permissionV1.RolePermission_ON, permissionV1.RolePermission_ALLOW},
	}
	viewByPermID := map[uint32]*permissionV1.RolePermission{}
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
