package ent_test

import (
	"context"
	"os"
	"reflect"
	"sort"
	"strconv"
	"testing"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	_ "github.com/lib/pq"

	"github.com/tx7do/go-crud/viewer"

	ent "go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/orgunit"
	_ "go-wind-admin/app/admin/service/internal/data/ent/runtime"
)

// mockScopeViewer 测试用 ViewerContext：可分别指定用户/租户/数据范围。
type mockScopeViewer struct {
	uid    int64
	tid    int64
	scopes []viewer.DataScope
}

func (m mockScopeViewer) UserID() int64                    { return m.uid }
func (m mockScopeViewer) TenantID() int64                  { return m.tid }
func (m mockScopeViewer) OrgUnitID() int64                 { return 0 }
func (m mockScopeViewer) Permissions() []string             { return nil }
func (m mockScopeViewer) Roles() []string                   { return nil }
func (m mockScopeViewer) DataScope() []viewer.DataScope     { return m.scopes }
func (m mockScopeViewer) TraceID() string                   { return "test" }
func (m mockScopeViewer) HasPermission(string, string) bool { return true }
func (m mockScopeViewer) IsPlatformContext() bool           { return m.tid == 0 }
func (m mockScopeViewer) IsTenantContext() bool             { return m.tid > 0 }
func (m mockScopeViewer) IsSystemContext() bool             { return false }
func (m mockScopeViewer) ShouldAudit() bool                 { return false }

func scopeCtx(ctx context.Context, uid, tid int64, scopes ...viewer.DataScope) context.Context {
	return viewer.WithContext(ctx, mockScopeViewer{uid: uid, tid: tid, scopes: scopes})
}

func openScopeTestClient(t *testing.T) *ent.Client {
	t.Helper()
	dsn := os.Getenv("GUARD_TEST_PG_DSN")
	if dsn == "" {
		dsn = guardTestDSN
	}
	drv, err := sql.Open(dialect.Postgres, dsn)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	// 连不通（本地无 citus 容器/库未建/端口漂移）时跳过而非失败：
	// 本文件验证的是编译进 ent 生成代码的数据范围隐私混入，属环境门控的集成测试，
	// 缺库环境下套件应保持绿。
	if err := drv.DB().Ping(); err != nil {
		_ = drv.Close()
		t.Skipf("guard-test postgres unavailable (set GUARD_TEST_PG_DSN to override): %v", err)
	}
	client := ent.NewClient(ent.Driver(drv))

	ctx := context.Background()
	if err := client.Schema.Create(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := drv.DB().Exec("TRUNCATE sys_positions, sys_org_units CASCADE"); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func fmtPair(unit, creator int64) string {
	return "u" + strconv.FormatInt(unit, 10) +
		":c" + strconv.FormatInt(creator, 10)
}

// seedPosition 在指定租户视角下种一行岗位（unit/creator 为行属性）。
func seedPosition(client *ent.Client, ctx context.Context, unit, creator int64, code string) {
	client.Position.Create().
		SetName("pos-" + code).
		SetCode(code).
		SetOrgUnitID(unit).
		SetCreatedBy(creator).
		ExecX(ctx)
}

// visiblePositionPairs 返回当前上下文可见岗位的 (unit, creator) 对集合。
func visiblePositionPairs(client *ent.Client, ctx context.Context) map[string]bool {
	rows := client.Position.Query().AllX(ctx)
	pairs := make(map[string]bool, len(rows))
	for _, row := range rows {
		if row.OrgUnitID == nil || row.CreatedBy == nil {
			continue
		}
		pairs[fmtPair(*row.OrgUnitID, *row.CreatedBy)] = true
	}
	return pairs
}

func expectPairs(t *testing.T, got map[string]bool, want ...string) {
	t.Helper()
	wantSet := make(map[string]bool, len(want))
	for _, w := range want {
		wantSet[w] = true
	}
	if !reflect.DeepEqual(got, wantSet) {
		t.Fatalf("visible pairs = %v, want %v", got, wantSet)
	}
}

// TestPositionDataScopeGuard 验证岗位表数据范围查询过滤矩阵：
// ALL 全量 / UNIT 按单元 / SELF 按创建人 / 并集 / None 拒绝 / 无 scope 拒绝 /
// 平台上下文放行 / 租户隔离与数据范围叠加。
func TestPositionDataScopeGuard(t *testing.T) {
	client := openScopeTestClient(t)
	ctx := context.Background()

	// 种子：租户 1 四行（2 单元 × 2 创建人），租户 2 一行（同单元号 101、
	// 独立创建人 21，验证租户隔离在数据范围之上仍然生效且行可区分）。
	{
		ctx1 := scopeCtx(ctx, 99, 1)
		seedPosition(client, ctx1, 101, 11, "DS_T1_A")
		seedPosition(client, ctx1, 101, 12, "DS_T1_B")
		seedPosition(client, ctx1, 102, 11, "DS_T1_C")
		seedPosition(client, ctx1, 102, 12, "DS_T1_D")
		ctx2 := scopeCtx(ctx, 99, 2)
		seedPosition(client, ctx2, 101, 21, "DS_T2_A")
	}

	unit101 := viewer.DataScope{ScopeType: viewer.ScopeTypeUnit, TargetIDs: []int64{101}}

	// ALL：租户 1 视角下本租户全量（4 行），租户 2 行不可见。
	expectPairs(t, visiblePositionPairs(client, scopeCtx(ctx, 11, 1,
		viewer.DataScope{ScopeType: viewer.ScopeTypeAll})),
		fmtPair(101, 11), fmtPair(101, 12), fmtPair(102, 11), fmtPair(102, 12))

	// UNIT[101]：仅单元 101 的行（2 行）。
	expectPairs(t, visiblePositionPairs(client, scopeCtx(ctx, 11, 1, unit101)),
		fmtPair(101, 11), fmtPair(101, 12))

	// SELF(11)：仅创建人 11 的行（2 行）。
	expectPairs(t, visiblePositionPairs(client, scopeCtx(ctx, 11, 1,
		viewer.DataScope{ScopeType: viewer.ScopeTypeSelf})),
		fmtPair(101, 11), fmtPair(102, 11))

	// UNIT[101] ∪ SELF(12)：并集（3 行：单元 101 两行 + 创建人 12 一行）。
	expectPairs(t, visiblePositionPairs(client, scopeCtx(ctx, 12, 1, unit101,
		viewer.DataScope{ScopeType: viewer.ScopeTypeSelf})),
		fmtPair(101, 11), fmtPair(101, 12), fmtPair(102, 12))

	// None：显式拒绝 → 查询报错。
	if _, err := client.Position.Query().All(scopeCtx(ctx, 11, 1,
		viewer.DataScope{ScopeType: viewer.ScopeTypeNone})); err == nil {
		t.Fatal("NONE scope query should be denied")
	}

	// 无 scope：fail-closed 拒绝。
	if _, err := client.Position.Query().All(scopeCtx(ctx, 11, 1)); err == nil {
		t.Fatal("empty scope query should be denied")
	}

	// 平台上下文（tid=0）：放行两租户全部 5 行。
	expectPairs(t, visiblePositionPairs(client, scopeCtx(ctx, 1, 0,
		viewer.DataScope{ScopeType: viewer.ScopeTypeAll})),
		fmtPair(101, 11), fmtPair(101, 12), fmtPair(102, 11), fmtPair(102, 12),
		fmtPair(101, 21))

	// 租户 2 视角 UNIT[101]：仅本租户的行（1 行），租户 1 行不可见。
	expectPairs(t, visiblePositionPairs(client, scopeCtx(ctx, 21, 2, unit101)),
		fmtPair(101, 21))
}

// seedOrgUnit 在指定租户视角下种一个组织单元（直接写 path 模拟树形关系）。
func seedOrgUnit(client *ent.Client, ctx context.Context, name, code, path string) {
	client.OrgUnit.Create().
		SetName(name).
		SetCode(code).
		SetPath(path).
		ExecX(ctx)
}

// TestOrgUnitPathExpansionTenantBounded 验证子树展开（path 前缀 + 租户谓词）
// 的组合语义：同形 path 跨租户互不串扰。
// 该组合即 ListSelfAndDescendantOrgUnitIds 的查询内核。
func TestOrgUnitPathExpansionTenantBounded(t *testing.T) {
	client := openScopeTestClient(t)
	ctx := context.Background()

	{
		ctx1 := scopeCtx(ctx, 99, 1)
		seedOrgUnit(client, ctx1, "t1-root", "T1R", "/1/")
		seedOrgUnit(client, ctx1, "t1-child-a", "T1CA", "/1/2/")
		seedOrgUnit(client, ctx1, "t1-child-b", "T1CB", "/1/3/")
		seedOrgUnit(client, ctx1, "t1-island", "T1I", "/9/")
		ctx2 := scopeCtx(ctx, 99, 2)
		seedOrgUnit(client, ctx2, "t2-samepath", "T2SP", "/1/2/")
	}

	// 租户 1 + 前缀 /1/：命中 root/child-a/child-b（含自身与后代），不含岛、不含租户 2 同形路径。
	{
		ctx1 := scopeCtx(ctx, 99, 1)
		paths := client.OrgUnit.Query().
			Where(
				orgunit.TenantIDEQ(1),
				orgunit.PathHasPrefix("/1/"),
			).
			Select(orgunit.FieldPath).
			StringsX(ctx1)
		sort.Strings(paths)
		if !reflect.DeepEqual(paths, []string{"/1/", "/1/2/", "/1/3/"}) {
			t.Fatalf("tenant1 expansion = %v, want [/1/ /1/2/ /1/3/]", paths)
		}
	}

	// 租户 2 + 前缀 /1/：仅本租户行，租户 1 的树不可见。
	{
		ctx2 := scopeCtx(ctx, 99, 2)
		paths := client.OrgUnit.Query().
			Where(
				orgunit.TenantIDEQ(2),
				orgunit.PathHasPrefix("/1/"),
			).
			Select(orgunit.FieldPath).
			StringsX(ctx2)
		if !reflect.DeepEqual(paths, []string{"/1/2/"}) {
			t.Fatalf("tenant2 expansion = %v, want [/1/2/]", paths)
		}
	}
}
