package data

import (
	"context"
	"testing"

	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/trans"

	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/enttest"
	entPosition "go-wind-admin/app/admin/service/internal/data/ent/position"
)

// 本文件用 PositionRepo 作为样例，演示 enttest helper 的用法：
//
//	entClient := enttest.NewEntClientForTest(t)        // SQLite 内存库 ent client
//	repo := &PositionRepo{entClient: entClient, ...}   // 白盒构造 repo
//	repo.init()
//	ctx := enttest.NewSystemViewerCtx(context.Background())
//	// ... 对 repo 做 CRUD 断言 ...
//
// 详见 enttest 包文档。

// newPositionRepoSqlite 用 enttest helper 构造一个可直接做 CRUD 的 PositionRepo。
func newPositionRepoSqlite(t *testing.T) *PositionRepo {
	t.Helper()
	entClient := enttest.NewEntClientForTest(t)
	// 白盒构造 PositionRepo，复用 NewPositionRepo.init() 的 mapper/converter 初始化逻辑
	repo := &PositionRepo{
		entClient: entClient,
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		mapper:    mapper.NewCopierMapper[identityV1.Position, ent.Position](),
		statusConverter: mapper.NewEnumTypeConverter[identityV1.Position_Status, entPosition.Status](
			identityV1.Position_Status_name, identityV1.Position_Status_value,
		),
		typeConverter: mapper.NewEnumTypeConverter[identityV1.Position_Type, entPosition.Type](
			identityV1.Position_Type_name, identityV1.Position_Type_value,
		),
	}
	repo.init()
	return repo
}

// TestPositionRepoSqlite_Create 端到端验证：用 SQLite 内存库（经 enttest helper）
// 对 PositionRepo 执行 Create，证明 repo 层集成测试基建可用（写入真实数据库表）。
func TestPositionRepoSqlite_Create(t *testing.T) {
	repo := newPositionRepoSqlite(t)
	// 注入系统级 ViewerContext，满足 ent mixin 的多租户隐私规则要求
	ctx := enttest.NewSystemViewerCtx(context.Background())

	// 通过 repo 的 Create API 写入一条记录，走完 mapper/converter → ent builder → SQLite 的完整链路
	err := repo.Create(ctx, &identityV1.CreatePositionRequest{
		Data: &identityV1.Position{
			Name:   trans.Ptr("测试职位"),
			Code:   trans.Ptr("TEST_POS_1"),
			Status: identityV1.Position_ON.Enum(),
		},
	})
	require.NoError(t, err, "通过 repo.Create 写入 SQLite 应成功")

	// 直接用 ent client 验证记录确实落库（绕过 repo，独立确认）
	count, err := repo.entClient.Client().Position.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count, "SQLite 中应有 1 条 position 记录")
}
