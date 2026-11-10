// RoleFieldPermissionRepo 的 SQLite 内存库测试（白盒，包内测试）。
//
// 这张表是字段级数据安全的黑名单配置源：登录聚合按角色并集取隐藏字段，写错一行
// 就是"该藏的字段看见了"。测试盯四件事：
//   - 行级展开：(role, resource, field) 一字段一行，条目内重复字段去重、条目间重复
//     走 (tenant, role, resource, field) 唯一索引的 upsert 冲突分支，绝不重复落行；
//   - 整体替换语义：先清空该角色旧配置再插新集合——撤销一个资源的同时另一个资源的
//     配置必须原样重写，不能因为"先清后插"把没改的资源丢了；
//   - 校验闸门：空资源名/空字段名/总条数超 256 拒绝，且拒绝发生在清理之前
//     （校验失败不得把角色已有配置清没）；
//   - 分组读视图：按 (role, resource) 聚合成 DTO，字段列表完整、多角色互不串组。
package data

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	entCrud "github.com/tx7do/go-crud/entgo"

	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/rolefieldpermission"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// TestRoleFieldPermissionRepoSqlite_ReplaceAndList 验证整体替换与分组读取主路径：
// 条目按 (role, resource, field) 展开成行、tenant_id/operatorID 逐列落库、
// 二次替换先清旧集合、条目间重复资源走 upsert 冲突分支不重复落行、
// ListHiddenFieldGroupsByRoleIDs 按 (role, resource) 聚合。
func TestRoleFieldPermissionRepoSqlite_ReplaceAndList(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewRoleFieldPermissionRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	const (
		tenantID   = int64(7)
		operatorID = int64(42)
		roleID     = int64(500)
	)

	// 首次替换：User 藏 email/phone，Order 藏 amount
	tx, err := entClient.Client().Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()
	require.NoError(t, repo.ReplaceFieldPermissions(ctx, tx, tenantID, operatorID, roleID, []*permissionV1.RoleFieldPermission{
		{Resource: "User", HiddenFields: []string{"email", "phone"}},
		{Resource: "Order", HiddenFields: []string{"amount"}},
	}), "合法条目的整体替换应成功")
	require.NoError(t, tx.Commit())

	rows, err := entClient.Client().RoleFieldPermission.Query().
		Where(rolefieldpermission.RoleIDEQ(roleID)).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 3, "两个字段 + 一个字段应展开为 3 行")
	for _, r := range rows {
		require.NotNil(t, r.TenantID)
		require.Equal(t, tenantID, *r.TenantID, "tenant_id 应按入参落库：唯一索引与读取都按租户隔离")
		require.NotNil(t, r.RoleID)
		require.Equal(t, roleID, *r.RoleID)
		require.NotNil(t, r.CreatedBy)
		require.Equal(t, operatorID, *r.CreatedBy, "operatorID 应落 created_by")
		require.NotNil(t, r.CreatedAt, "created_at 必须显式写入")
	}

	// 分组读视图：按 (role, resource) 聚合，字段列表完整
	groups, err := repo.ListHiddenFieldGroupsByRoleIDs(ctx, []int64{roleID})
	require.NoError(t, err)
	require.Len(t, groups[roleID], 2, "User 与 Order 应聚成两组")
	byResource := map[string]*permissionV1.RoleFieldPermission{}
	for _, g := range groups[roleID] {
		byResource[g.GetResource()] = g
	}
	require.ElementsMatch(t, []string{"email", "phone"}, byResource["User"].GetHiddenFields(),
		"User 组的隐藏字段应完整聚合")
	require.Equal(t, []string{"amount"}, byResource["Order"].GetHiddenFields())

	// 单角色回显入口与分组读视图一致
	list, err := repo.ListFieldPermissions(ctx, roleID)
	require.NoError(t, err)
	require.Len(t, list, 2, "ListFieldPermissions 应与分组读视图一致")

	// 二次替换：User 全撤、改藏 Department；条目间重复资源（两条 User 条目且含同字段）
	// 走唯一索引冲突更新分支，不得重复落行
	tx2, err := entClient.Client().Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx2.Rollback() }()
	require.NoError(t, repo.ReplaceFieldPermissions(ctx, tx2, tenantID, operatorID, roleID, []*permissionV1.RoleFieldPermission{
		{Resource: "User", HiddenFields: []string{"department"}},
		{Resource: "User", HiddenFields: []string{"department", "mobile"}},
	}), "含条目间重复资源的替换应成功（重复字段命中 upsert 冲突分支）")
	require.NoError(t, tx2.Commit())

	rows, err = entClient.Client().RoleFieldPermission.Query().
		Where(rolefieldpermission.RoleIDEQ(roleID)).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 2, "旧集合（Order.amount）应被清掉，新集合去重后 User.department + User.mobile 共 2 行")
	fieldSet := map[string]struct{}{}
	for _, r := range rows {
		require.NotNil(t, r.FieldName)
		fieldSet[*r.FieldName] = struct{}{}
	}
	require.Len(t, fieldSet, 2, "department 与 mobile 各一行，重复条目不得产生重复行")

	// 冲突更新分支盖章：第二次插入命中唯一索引时 updated_by/updated_at 要被刷新
	rowDepartment, err := entClient.Client().RoleFieldPermission.Query().
		Where(
			rolefieldpermission.RoleIDEQ(roleID),
			rolefieldpermission.FieldNameEQ("department"),
		).
		Only(ctx)
	require.NoError(t, err)
	require.NotNil(t, rowDepartment.UpdatedBy)
	require.Equal(t, operatorID, *rowDepartment.UpdatedBy, "冲突更新分支应刷新 updated_by")
	require.NotNil(t, rowDepartment.UpdatedAt, "冲突更新分支应刷新 updated_at")

	// 替换为空集合：合法（撤销全部），只清理不插入
	tx3, err := entClient.Client().Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx3.Rollback() }()
	require.NoError(t, repo.ReplaceFieldPermissions(ctx, tx3, tenantID, operatorID, roleID, nil),
		"空集合的替换应成功（撤销全部字段权限）")
	require.NoError(t, tx3.Commit())
	require.Equal(t, 0, countRoleFieldPermissionRows(t, entClient, ctx, roleID), "撤销全部后该角色应无配置行")

	// 撤销后读视图：nil 而不是报错（未配置即全字段可见，调用方按 nil 处理）
	list, err = repo.ListFieldPermissions(ctx, roleID)
	require.NoError(t, err)
	require.Nil(t, list, "无配置的角色应返回 nil 组")

	// 空 roleIDs：直接空表返回，不打库
	groups, err = repo.ListHiddenFieldGroupsByRoleIDs(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, groups, "空角色列表应返回空 map")
}

// TestRoleFieldPermissionRepoSqlite_MultiRole 验证多角色聚合并集且互不串组：
// 登录聚合一次查全部角色的配置，按角色分桶回填。
func TestRoleFieldPermissionRepoSqlite_MultiRole(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewRoleFieldPermissionRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	tx, err := entClient.Client().Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()
	require.NoError(t, repo.ReplaceFieldPermissions(ctx, tx, 7, 1, 601, []*permissionV1.RoleFieldPermission{
		{Resource: "User", HiddenFields: []string{"salary"}},
	}))
	require.NoError(t, repo.ReplaceFieldPermissions(ctx, tx, 7, 1, 602, []*permissionV1.RoleFieldPermission{
		{Resource: "User", HiddenFields: []string{"email", "phone"}},
		{Resource: "Order", HiddenFields: []string{"profit"}},
	}))
	require.NoError(t, tx.Commit())

	groups, err := repo.ListHiddenFieldGroupsByRoleIDs(ctx, []int64{601, 602, 603})
	require.NoError(t, err)
	require.Len(t, groups[601], 1, "角色 601 应只有 User 组")
	require.Equal(t, []string{"salary"}, groups[601][0].GetHiddenFields())
	require.Len(t, groups[602], 2, "角色 602 应有 User 与 Order 两组")
	require.Len(t, groups, 2, "无配置的角色 603 不该出现在结果里（不回填空组）")
	for _, g := range groups[602] {
		if g.GetResource() == "User" {
			require.ElementsMatch(t, []string{"email", "phone"}, g.GetHiddenFields(),
				"多角色并集聚合时各角色的组不得串字段")
		}
	}
}

// TestRoleFieldPermissionRepoSqlite_Validate 验证校验闸门与失败不破坏现场：
// 空资源名/空字段名拒绝、总量超 256 拒绝、条目内重复字段按去重后计数；
// 校验失败发生在清理之前，角色已有配置必须原样保留。
func TestRoleFieldPermissionRepoSqlite_Validate(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewRoleFieldPermissionRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	const roleID = int64(700)

	// 先落一份合法配置，作为"校验失败不得破坏现场"的对照组
	tx, err := entClient.Client().Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()
	require.NoError(t, repo.ReplaceFieldPermissions(ctx, tx, 7, 1, roleID, []*permissionV1.RoleFieldPermission{
		{Resource: "User", HiddenFields: []string{"email"}},
	}))
	require.NoError(t, tx.Commit())

	// 空资源名：条目挂在哪个资源上都没有，裁剪器无从命中，必须拒绝
	tx2, err := entClient.Client().Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx2.Rollback() }()
	err = repo.ReplaceFieldPermissions(ctx, tx2, 7, 1, roleID, []*permissionV1.RoleFieldPermission{
		{Resource: "", HiddenFields: []string{"email"}},
	})
	require.Error(t, err, "空资源名应被拒绝")
	require.Contains(t, err.Error(), "resource is required")

	// 空字段名：黑名单里一个空串字段什么都裁不掉，还会污染读视图
	err = repo.ReplaceFieldPermissions(ctx, tx2, 7, 1, roleID, []*permissionV1.RoleFieldPermission{
		{Resource: "User", HiddenFields: []string{"email", ""}},
	})
	require.Error(t, err, "空字段名应被拒绝")
	require.Contains(t, err.Error(), "field name is required")
	require.NoError(t, tx2.Rollback(), "校验失败的事务应回滚")

	// 超上限：3 条 x 100 字段 = 300 > 256。上限防的是异常配置撑爆令牌与配置 UI。
	over := make([]*permissionV1.RoleFieldPermission, 0, 3)
	for i := 0; i < 3; i++ {
		fields := make([]string, 0, 100)
		for j := 0; j < 100; j++ {
			fields = append(fields, fmt.Sprintf("field_%d_%d", i, j))
		}
		over = append(over, &permissionV1.RoleFieldPermission{Resource: fmt.Sprintf("Res%d", i), HiddenFields: fields})
	}
	tx3, err := entClient.Client().Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx3.Rollback() }()
	err = repo.ReplaceFieldPermissions(ctx, tx3, 7, 1, roleID, over)
	require.Error(t, err, "超上限应被拒绝")
	require.Contains(t, err.Error(), "too many field permission entries")
	require.NoError(t, tx3.Rollback())

	// 恰好 256（去重后）：2 条 x 128 = 256，不超限应放行
	atCap := make([]*permissionV1.RoleFieldPermission, 0, 2)
	for i := 0; i < 2; i++ {
		fields := make([]string, 0, 128)
		for j := 0; j < 128; j++ {
			fields = append(fields, fmt.Sprintf("field_%d_%d", i, j))
		}
		atCap = append(atCap, &permissionV1.RoleFieldPermission{Resource: fmt.Sprintf("Res%d", i), HiddenFields: fields})
	}
	tx4, err := entClient.Client().Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx4.Rollback() }()
	require.NoError(t, repo.ReplaceFieldPermissions(ctx, tx4, 7, 1, roleID, atCap), "恰好 256 条应放行")
	require.NoError(t, tx4.Commit())

	// 条目内重复字段按去重计数：同一字段重复 300 次只算 1 条，应放行且只落 1 行
	dup := []*permissionV1.RoleFieldPermission{
		{Resource: "User", HiddenFields: repeatString("email", 300)},
	}
	tx5, err := entClient.Client().Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx5.Rollback() }()
	require.NoError(t, repo.ReplaceFieldPermissions(ctx, tx5, 7, 1, roleID, dup),
		"重复字段按去重后计数应放行")
	require.NoError(t, tx5.Commit())
	require.Equal(t, 1, countRoleFieldPermissionRows(t, entClient, ctx, roleID),
		"去重后只应落 1 行：重复条目靠唯一索引 upsert 收敛")

	// 前面所有校验失败的尝试都回滚了：对照组配置之外不应有残留
	rows, err := entClient.Client().RoleFieldPermission.Query().
		Where(rolefieldpermission.RoleIDEQ(roleID)).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1, "校验失败不得破坏角色已有配置")
}

// TestRoleFieldPermissionRepoSqlite_Clean 验证按角色清理：
// 只清目标角色的行，其他角色与租户的行不受牵连。
func TestRoleFieldPermissionRepoSqlite_Clean(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewRoleFieldPermissionRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	tx, err := entClient.Client().Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()
	require.NoError(t, repo.ReplaceFieldPermissions(ctx, tx, 7, 1, 800, []*permissionV1.RoleFieldPermission{
		{Resource: "User", HiddenFields: []string{"email", "phone"}},
	}))
	require.NoError(t, repo.ReplaceFieldPermissions(ctx, tx, 7, 1, 801, []*permissionV1.RoleFieldPermission{
		{Resource: "User", HiddenFields: []string{"salary"}},
	}))
	require.NoError(t, tx.Commit())

	// 清理角色 800：角色 801 的配置必须毫发无损（删错行就是把别的角色的隐藏字段放出来）
	tx2, err := entClient.Client().Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx2.Rollback() }()
	require.NoError(t, repo.CleanFieldPermissions(ctx, tx2, 800), "清理角色 800 应成功")
	require.NoError(t, tx2.Commit())

	require.Equal(t, 0, countRoleFieldPermissionRows(t, entClient, ctx, 800), "角色 800 的配置应被清空")
	require.Equal(t, 1, countRoleFieldPermissionRows(t, entClient, ctx, 801), "角色 801 的配置不得被牵连")

	// 清理无配置的角色：幂等成功
	tx3, err := entClient.Client().Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx3.Rollback() }()
	require.NoError(t, repo.CleanFieldPermissions(ctx, tx3, 999999), "清理无配置的角色应幂等成功")
	require.NoError(t, tx3.Commit())
}

// countRoleFieldPermissionRows 统计某角色的配置行数。
func countRoleFieldPermissionRows(t *testing.T, entClient *entCrud.EntClient[*ent.Client], ctx context.Context, roleID int64) int {
	t.Helper()
	cnt, err := entClient.Client().RoleFieldPermission.Query().
		Where(rolefieldpermission.RoleIDEQ(roleID)).
		Count(ctx)
	require.NoError(t, err)
	return cnt
}

// repeatString 生成 n 个 s 的切片（测试用）。
func repeatString(s string, n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = s
	}
	return out
}
