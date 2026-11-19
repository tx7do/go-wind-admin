// Package enttest 提供 repo 层集成测试的辅助设施。
//
// 核心能力：用 SQLite 内存库（modernc.org/sqlite，纯 Go 无 CGO）+ ent 生成的
// schema 构造可做真实 CRUD 的 ent client，让 repo 层测试无需 MySQL/PG 或
// testcontainers 即可跑数据库交互。
//
// 用法：
//
//	func TestFooRepo(t *testing.T) {
//	    entClient := enttest.NewEntClientForTest(t)
//	    repo := NewFooRepo(testCtx(), entClient) // 复用生产构造函数
//	    ctx := enttest.NewSystemContext(context.Background())
//	    // ... 对 repo 做 CRUD 断言 ...
//	}
//
// 安全说明：
//   - 每个 entClient 独占一个 SQLite 内存库，测试间互不干扰；
//     t.Cleanup 负责关闭，无需手动清理。
//   - NewSystemContext 注入平台级 SystemContext（tenant_id=0），
//     与生产中系统后台任务的身份一致，满足 ent mixin 的多租户隐私规则。
package enttest

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite" // 纯 Go SQLite（注册名 "sqlite"），无需 CGO

	entsql "entgo.io/ent/dialect/sql"
	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-crud/viewer"
	"go-wind-admin/app/admin/service/internal/data/ent"
)

// NewEntClientForTest 构造一个基于 SQLite 内存库的 entCrud.EntClient，
// 供 repo 层集成测试使用。每个调用独占一个内存库，通过 t.Cleanup 自动关闭。
// 参数为 testing.TB：常规测试（*testing.T）与基准（*testing.B）均可直接使用。
func NewEntClientForTest(t testing.TB) *entCrud.EntClient[*ent.Client] {
	t.Helper()
	// modernc.org/sqlite 以 "sqlite" 名注册到 database/sql。
	// 每次调用用唯一命名的 mode=memory 库：连接池内同 URI 连接共享同一库
	//（schema 迁移与后续读写一致），而不同调用（乃至不同测试进程）各拿各的库。
	// 此前用 file::memory:?cache=shared——同进程内全部 client 实际共享同一个库，
	// 任何一处泄漏写锁（如未回滚事务）会把整个测试二进制毒化成
	// "database is locked"。唯一命名把隔离做实，泄漏只影响单个测试。
	// _pragma=foreign_keys(1) 开启外键（ent 要求）。
	db, err := sql.Open("sqlite", fmt.Sprintf("file:enttest_%d?mode=memory&cache=shared&_pragma=foreign_keys(1)", time.Now().UnixNano()))
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	drv := entsql.OpenDB(dialect.SQLite, db)
	t.Cleanup(func() { drv.Close() })

	client := ent.NewClient(ent.Driver(drv))
	t.Cleanup(func() { client.Close() })
	require.NoError(t, client.Schema.Create(context.Background()), "SQLite schema 迁移失败")

	return entCrud.NewEntClient[*ent.Client](client, drv)
}

// NewSystemContext 返回注入了平台级 SystemContext 的 context，
// 满足 ent mixin 多租户隐私规则对 ViewerContext 的强制要求。
func NewSystemContext(ctx context.Context) context.Context {
	return viewer.WithSystemContext(ctx)
}
