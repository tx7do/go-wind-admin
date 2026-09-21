package data

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// TestIsPostgresDriver 钉住"只有 PG 需要序列对齐"的判定：ent 的两种 PG 方言
// （lib/pq 的 "postgres"、jackc/pgx 的 "pgx"）都要放行，MySQL/SQLite 与
// 配置缺失（空串，测试构造器直接 new struct 时的取值）都要拦住。
func TestIsPostgresDriver(t *testing.T) {
	for _, d := range []string{"postgres", "pgx"} {
		require.True(t, isPostgresDriver(d), "%s 应判定为 PG", d)
	}
	for _, d := range []string{"mysql", "sqlite", "", "SQLite"} {
		require.False(t, isPostgresDriver(d), "%q 不应判定为 PG", d)
	}
}

// TestAlignIdentitySequenceSqliteIsNoop 验证非 PG 驱动下对齐是 no-op，
// 并且"显式 ID 插入 + 无 ID 插入"这条在 PG 上会撞主键的路径在 SQLite 上照常走通
// （SQLite 的 rowid 会自己抬到 max+1，这正是它不需要修复的原因）。
func TestAlignIdentitySequenceSqliteIsNoop(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, entClient.Client().Menu.Create().
		SetID(9001).
		SetPath("/sqlite/identity/explicit").
		Exec(ctx), "显式 ID 插入应成功")

	require.NoError(t, alignIdentitySequence(ctx, entClient.DB(), "sqlite", identityTableMenus),
		"SQLite 上对齐应为 no-op")

	created, err := entClient.Client().Menu.Create().
		SetPath("/sqlite/identity/next").
		Save(ctx)
	require.NoError(t, err, "no-op 之后无 ID 插入应照常成功")
	require.Greater(t, created.ID, uint32(9001), "SQLite 自增应自动抬过显式 ID")
}

// TestAlignIdentitySequenceNilHandle 验证句柄缺失时报错而不是静默成功——
// 静默成功会让"序列已对齐"这个前提在无日志的情况下落空。
func TestAlignIdentitySequenceNilHandle(t *testing.T) {
	err := alignIdentitySequence(context.Background(), nil, "postgres", identityTableMenus)
	require.Error(t, err)
	require.Contains(t, err.Error(), identityTableMenus)
}

// TestAlignIdentitySequenceRunsOnlyOnPostgresBranch 反向证明驱动判定就是唯一的门：
// 以 "postgres" 调用时确实会去跑那条 PG 语法（setval），在 SQLite 库上必然失败。
func TestAlignIdentitySequenceRunsOnlyOnPostgresBranch(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	err := alignIdentitySequence(ctx, entClient.DB(), "postgres", identityTableMenus)
	require.Error(t, err, "PG 分支的 setval 语句在 SQLite 上应失败")
	require.Contains(t, err.Error(), identityTableMenus)
}

// TestMenuRepoAlignIdentitySequenceSqlite 走 repo 出口：测试构造器不填 driverName，
// 因此按"非 PG"处理并成功返回——菜单服务启动期的那次调用在 SQLite 集成测试里应无副作用。
func TestMenuRepoAlignIdentitySequenceSqlite(t *testing.T) {
	repo := newMenuRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.AlignIdentitySequence(ctx))
}
