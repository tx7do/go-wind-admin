package data

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
	"go-wind-admin/app/admin/service/internal/data/ent"
	entRoleMetadata "go-wind-admin/app/admin/service/internal/data/ent/rolemetadata"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newRoleMetadataRepoSqlite 用 enttest helper 构造一个可直接做 CRUD 的 RoleMetadataRepo。
// 白盒构造逐字段复刻 NewRoleMetadataRepo 的 mapper/converter 初始化，
// 仅将 log 换为 NopLogger、entClient 换为 SQLite 内存库测试 client。
func newRoleMetadataRepoSqlite(t *testing.T) *RoleMetadataRepo {
	t.Helper()
	entClient := enttest.NewEntClientForTest(t)
	repo := &RoleMetadataRepo{
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

	repo.init()

	return repo
}

// TestRoleMetadataRepoSqlite_CreateGetExist 覆盖 RoleMetadataRepo 的写入与读取：
// Create（落库）→ ent client 直查确认 → Get（命中）→ IsExistByRoleID/IsTemplateRole。
func TestRoleMetadataRepoSqlite_CreateGetExist(t *testing.T) {
	repo := newRoleMetadataRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	const roleID = uint32(16001)

	tx, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	// 无论断言成败都释放事务连接：Commit 成功后 Rollback 返回 ErrTxDone 被忽略；
	// 断言失败（Goexit）时回滚，避免残留写锁阻塞后续测试。
	defer func() { _ = tx.Rollback() }()
	require.NoError(t, repo.Create(ctx, tx, &permissionV1.RoleMetadata{
		RoleId:     trans.Ptr(roleID),
		IsTemplate: trans.Ptr(false),
		SyncPolicy: permissionV1.RoleMetadata_AUTO.Enum(),
		Scope:      permissionV1.RoleMetadata_TENANT.Enum(),
	}), "写入角色元数据应成功")
	require.NoError(t, tx.Commit())

	// ent client 直查：记录确实落库，枚举经转换器映射
	rows, err := repo.entClient.Client().RoleMetadata.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1, "sys_role_metadata 应有 1 条记录")
	require.Equal(t, roleID, *rows[0].RoleID, "role_id 应按载荷落库")
	require.NotNil(t, rows[0].SyncPolicy, "sync_policy 应经转换器落库")
	require.Equal(t, entRoleMetadata.SyncPolicyAuto, *rows[0].SyncPolicy, "proto AUTO 应映射为 ent SyncPolicyAuto")
	require.NotNil(t, rows[0].Scope, "scope 应经转换器落库")
	require.Equal(t, entRoleMetadata.ScopeTenant, *rows[0].Scope, "proto TENANT 应映射为 ent ScopeTenant")

	// Get 命中：按 roleID 查询单条
	got, err := repo.Get(ctx, roleID)
	require.NoError(t, err, "按 roleID 查询已存在元数据应命中")
	require.Equal(t, roleID, got.GetRoleId(), "命中记录的 role_id 应与写入一致")
	require.False(t, got.GetIsTemplate(), "命中记录应为非模板")
	// 读视图：两枚举字段经回填如实呈现（与上方 ent 行断言互为印证）。
	require.Equal(t, permissionV1.RoleMetadata_AUTO, got.GetSyncPolicy(), "读视图应回填 sync_policy")
	require.Equal(t, permissionV1.RoleMetadata_TENANT, got.GetScope(), "读视图应回填 scope")

	// 存在性与模板判定
	exist, err := repo.IsExistByRoleID(ctx, roleID)
	require.NoError(t, err)
	require.True(t, exist, "已写入的 roleID 应判定为存在")
	isTpl, err := repo.IsTemplateRole(ctx, roleID)
	require.NoError(t, err)
	require.False(t, isTpl, "非模板记录应判定为非模板")

	// 未命中：Get 不存在的 roleID 应报错
	_, err = repo.Get(ctx, 99999)
	require.Error(t, err, "查询不存在的 roleID 应返回错误")
	notExist, err := repo.IsExistByRoleID(ctx, 99999)
	require.NoError(t, err)
	require.False(t, notExist, "不存在的 roleID 应判定为不存在")
	_, err = repo.IsTemplateRole(ctx, 99999)
	require.Error(t, err, "对不存在的 roleID 判定模板应返回错误")
}

// TestRoleMetadataRepoSqlite_TemplateVersionUpgrade 验证 UpgradeTemplateVersion
// 仅对模板记录生效并递增版本号；非模板记录调用后无变化。
func TestRoleMetadataRepoSqlite_TemplateVersionUpgrade(t *testing.T) {
	repo := newRoleMetadataRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	const (
		templateRoleID = uint32(16002)
		plainRoleID    = uint32(16003)
	)

	// 模板记录 + 普通记录各一条
	tx, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()
	require.NoError(t, repo.Create(ctx, tx, &permissionV1.RoleMetadata{
		RoleId:          trans.Ptr(templateRoleID),
		IsTemplate:      trans.Ptr(true),
		TemplateVersion: trans.Ptr(int32(3)),
		SyncPolicy:      permissionV1.RoleMetadata_AUTO.Enum(),
		Scope:           permissionV1.RoleMetadata_PLATFORM.Enum(),
	}))
	require.NoError(t, repo.Create(ctx, tx, &permissionV1.RoleMetadata{
		RoleId:          trans.Ptr(plainRoleID),
		IsTemplate:      trans.Ptr(false),
		TemplateVersion: trans.Ptr(int32(7)),
		SyncPolicy:      permissionV1.RoleMetadata_AUTO.Enum(),
		Scope:           permissionV1.RoleMetadata_TENANT.Enum(),
	}))
	require.NoError(t, tx.Commit())

	// 模板记录：版本号 +1（3 → 4）
	tx2, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx2.Rollback() }()
	require.NoError(t, repo.UpgradeTemplateVersion(ctx, tx2, templateRoleID))
	require.NoError(t, tx2.Commit())

	tplRow, err := repo.entClient.Client().RoleMetadata.Query().
		Where(entRoleMetadata.RoleIDEQ(templateRoleID)).
		Only(ctx)
	require.NoError(t, err)
	require.Equal(t, int32(4), *tplRow.TemplateVersion, "模板记录版本号应递增为 4")

	// 非模板记录：调用不报错但版本号保持不变
	tx3, err := repo.entClient.Client().Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx3.Rollback() }()
	require.NoError(t, repo.UpgradeTemplateVersion(ctx, tx3, plainRoleID))
	require.NoError(t, tx3.Commit())

	plainRow, err := repo.entClient.Client().RoleMetadata.Query().
		Where(entRoleMetadata.RoleIDEQ(plainRoleID)).
		Only(ctx)
	require.NoError(t, err)
	require.Equal(t, int32(7), *plainRow.TemplateVersion, "非模板记录版本号应保持 7 不变")
}

// TestRoleMetadataRepoSqlite_Upsert 验证 Upsert 语义：
// 冲突目标 (tenant_id, role_id) 与唯一索引 idx_role_metadata_tenant_role 一致后，
// 首次调用走插入分支落新行；同键二次调用命中复合唯一索引走冲突更新分支
// （AddTemplateVersion(1) 使版本号 +1），且不新增行。
// tenant_id 为可空列，显式给非空值才能确定性命中索引（NULL 在唯一索引中互异）。
func TestRoleMetadataRepoSqlite_Upsert(t *testing.T) {
	repo := newRoleMetadataRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	const roleID = uint32(16004)

	payload := &permissionV1.RoleMetadata{
		RoleId:          trans.Ptr(roleID),
		TenantId:        trans.Ptr(uint32(0)),
		IsTemplate:      trans.Ptr(false),
		TemplateVersion: trans.Ptr(int32(5)),
		SyncPolicy:      permissionV1.RoleMetadata_AUTO.Enum(),
		Scope:           permissionV1.RoleMetadata_TENANT.Enum(),
	}

	// 首次 Upsert：无冲突行 → 插入分支，版本号取载荷值
	require.NoError(t, repo.Upsert(ctx, payload), "首次 Upsert 应插入新行")
	rows, err := repo.entClient.Client().RoleMetadata.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1, "首次 Upsert 后应恰有 1 行")
	require.Equal(t, int32(5), *rows[0].TemplateVersion, "插入分支版本号应为载荷值 5")

	// 同键二次 Upsert：命中 (tenant_id, role_id) 复合唯一索引 → 冲突更新分支
	require.NoError(t, repo.Upsert(ctx, payload), "同键二次 Upsert 应走冲突更新分支不报错")
	rows2, err := repo.entClient.Client().RoleMetadata.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows2, 1, "冲突更新不得新增行")
	require.Equal(t, int32(6), *rows2[0].TemplateVersion, "冲突更新分支 AddTemplateVersion(1) 应使版本号递增为 6")
}
