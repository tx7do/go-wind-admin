package ent_test

import (
	"context"
	"os"
	"testing"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	_ "github.com/lib/pq"

	"github.com/tx7do/go-crud/viewer"

	ent "go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/accesskey"
	"go-wind-admin/app/admin/service/internal/data/ent/dicttype"
	_ "go-wind-admin/app/admin/service/internal/data/ent/runtime"
)

// guardTestDSN 守卫测试库连接串。默认指向本地 citus 容器（宿主端口随容器重建漂移，
// 以 GUARD_TEST_PG_DSN 环境变量覆盖）。
const guardTestDSN = "host=127.0.0.1 port=5432 user=postgres password=*Abcd123456 dbname=gwa_guard_test sslmode=disable"

// mockViewer 测试用 ViewerContext：模拟指定租户的用户。
type mockViewer struct {
	tid int64
}

func (m mockViewer) UserID() int64                     { return m.tid }
func (m mockViewer) TenantID() int64                   { return m.tid }
func (m mockViewer) OrgUnitID() int64                  { return 0 }
func (m mockViewer) Permissions() []string             { return nil }
func (m mockViewer) Roles() []string                   { return nil }
func (m mockViewer) DataScope() []viewer.DataScope     { return nil }
func (m mockViewer) TraceID() string                   { return "test" }
func (m mockViewer) HasPermission(string, string) bool { return true }
func (m mockViewer) IsPlatformContext() bool           { return false }
func (m mockViewer) IsTenantContext() bool             { return m.tid > 0 }
func (m mockViewer) IsSystemContext() bool             { return false }
func (m mockViewer) ShouldAudit() bool                 { return false }

func openGuardTestClient(t *testing.T) *ent.Client {
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
	// 本文件验证的是编译进 ent 生成代码的租户隐私混入，属环境门控的集成测试，
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
	if _, err := drv.DB().Exec("TRUNCATE sys_dict_types, sys_dict_entries, sys_access_keys CASCADE"); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func tenantCtx(ctx context.Context, tid int64) context.Context {
	return viewer.WithContext(ctx, mockViewer{tid: tid})
}

// TestTenantMutationGuard 验证跨租户 Update/Delete 被 TenantMutationGuardPolicy 拦截，
// 本租户变更不受影响。
func TestTenantMutationGuard(t *testing.T) {
	client := openGuardTestClient(t)
	ctx := context.Background()

	// 种数据：分别属于租户 1 / 2（用租户自身 viewer 写入，Create 强制覆盖 tenant_id）
	{
		ctx1 := tenantCtx(ctx, 1)
		client.DictType.Create().SetTypeCode("GUARD_T1").SetTypeName("tenant1-type").SetIsEnabled(true).ExecX(ctx1)
		ctx2 := tenantCtx(ctx, 2)
		client.DictType.Create().SetTypeCode("GUARD_T2").SetTypeName("tenant2-type").SetIsEnabled(true).ExecX(ctx2)
	}

	ctx1 := tenantCtx(ctx, 1)
	ctx2 := tenantCtx(ctx, 2)

	// 用 type_code 反查主键 id（供 Update/Delete 使用）
	t1ID := dictTypeIDByCode(client, ctx1, "GUARD_T1")
	t2ID := dictTypeIDByCode(client, ctx2, "GUARD_T2")

	// Update：租户 1 用户改租户 2 的行 → 必须命中 0 行（隔离生效）
	n := client.DictType.Update().Where(dicttype.IDEQ(t2ID)).SetTypeName("hacked").SaveX(ctx1)
	if n != 0 {
		t.Fatalf("cross-tenant update affected %d rows, want 0", n)
	}

	// 跨租户改名后，租户 2 视角下的行保持原状
	if got := *client.DictType.GetX(ctx2, t2ID).TypeName; got != "tenant2-type" {
		t.Fatalf("cross-tenant update leaked: name=%q", got)
	}

	// Update：本租户行正常更新
	n = client.DictType.Update().Where(dicttype.IDEQ(t1ID)).SetTypeName("t1-renamed").SaveX(ctx1)
	if n != 1 {
		t.Fatalf("own update affected %d rows, want 1", n)
	}

	// Delete：租户 1 用户删租户 2 的行 → 必须命中 0 行（隔离生效）
	n = client.DictType.Delete().Where(dicttype.IDEQ(t2ID)).ExecX(ctx1)
	if n != 0 {
		t.Fatalf("cross-tenant delete affected %d rows, want 0", n)
	}

	// 跨租户行未被删除
	if !client.DictType.Query().Where(dicttype.IDEQ(t2ID)).ExistX(ctx2) {
		t.Fatal("cross-tenant delete removed the row")
	}

	// Delete：本租户行正常删除
	n = client.DictType.Delete().Where(dicttype.IDEQ(t1ID)).ExecX(ctx1)
	if n != 1 {
		t.Fatalf("own delete affected %d rows, want 1", n)
	}
}

// dictTypeIDByCode 在指定租户视角下按 type_code 反查主键。
func dictTypeIDByCode(client *ent.Client, ctx context.Context, code string) int64 {
	row := client.DictType.Query().Where(dicttype.TypeCodeEQ(code)).OnlyX(ctx)
	return row.ID
}

// TestTenantMutationGuardAccessKey 验证 access_key（AK/SK 凭证表）的跨租户
// Update/Delete 被租户写隔离防线拦截。该表 2026-09-12 前独缺仓内守卫层，
// 本用例钉住其防线不再回退（防线=库层 TenantPrivacy + 仓内守卫，同谓词叠加）。
func TestTenantMutationGuardAccessKey(t *testing.T) {
	client := openGuardTestClient(t)
	ctx := context.Background()

	// 种数据：分别属于租户 1 / 2（Create 强制覆盖 tenant_id 为写入者租户）
	{
		ctx1 := tenantCtx(ctx, 1)
		client.AccessKey.Create().SetAccessKey("GUARD_AK_T1").SetName("tenant1-ak").ExecX(ctx1)
		ctx2 := tenantCtx(ctx, 2)
		client.AccessKey.Create().SetAccessKey("GUARD_AK_T2").SetName("tenant2-ak").ExecX(ctx2)
	}

	ctx1 := tenantCtx(ctx, 1)
	ctx2 := tenantCtx(ctx, 2)

	// 用 access_key 反查主键（供 Update/Delete 使用）
	t1ID := accessKeyIDByKey(client, ctx1, "GUARD_AK_T1")
	t2ID := accessKeyIDByKey(client, ctx2, "GUARD_AK_T2")

	// Update：租户 1 用户改租户 2 的 AK 行 → 必须命中 0 行（隔离生效）
	n := client.AccessKey.Update().Where(accesskey.IDEQ(t2ID)).SetName("hacked").SaveX(ctx1)
	if n != 0 {
		t.Fatalf("cross-tenant update affected %d rows, want 0", n)
	}

	// 跨租户改名后，租户 2 视角下的行保持原状
	if got := *client.AccessKey.GetX(ctx2, t2ID).Name; got != "tenant2-ak" {
		t.Fatalf("cross-tenant update leaked: name=%q", got)
	}

	// Update：本租户行正常更新
	n = client.AccessKey.Update().Where(accesskey.IDEQ(t1ID)).SetName("t1-renamed").SaveX(ctx1)
	if n != 1 {
		t.Fatalf("own update affected %d rows, want 1", n)
	}

	// Delete：租户 1 用户删租户 2 的 AK 行 → 必须命中 0 行（隔离生效）
	n = client.AccessKey.Delete().Where(accesskey.IDEQ(t2ID)).ExecX(ctx1)
	if n != 0 {
		t.Fatalf("cross-tenant delete affected %d rows, want 0", n)
	}

	// 跨租户行未被删除
	if !client.AccessKey.Query().Where(accesskey.IDEQ(t2ID)).ExistX(ctx2) {
		t.Fatal("cross-tenant delete removed the row")
	}

	// Delete：本租户行正常删除
	n = client.AccessKey.Delete().Where(accesskey.IDEQ(t1ID)).ExecX(ctx1)
	if n != 1 {
		t.Fatalf("own delete affected %d rows, want 1", n)
	}
}

// accessKeyIDByKey 在指定租户视角下按 access_key 反查主键。
func accessKeyIDByKey(client *ent.Client, ctx context.Context, key string) int64 {
	row := client.AccessKey.Query().Where(accesskey.AccessKeyEQ(key)).OnlyX(ctx)
	return row.ID
}
