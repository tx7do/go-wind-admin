package data

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	"go-wind-admin/app/admin/service/internal/data/ent"
	entPosition "go-wind-admin/app/admin/service/internal/data/ent/position"
	"go-wind-admin/app/admin/service/internal/data/enttest"
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

// TestPositionRepoSqlite_List 验证 PositionRepo.List 的分页列表与
// contains 模糊搜索过滤语义（仓规：搜索条件一律 contains）。
func TestPositionRepoSqlite_List(t *testing.T) {
	repo := newPositionRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	// 两条带可区分标记的记录
	require.NoError(t, repo.Create(ctx, &identityV1.CreatePositionRequest{
		Data: &identityV1.Position{
			Name:   trans.Ptr("MARKERALPHA 职位"),
			Code:   trans.Ptr(fmt.Sprintf("POS_SQLITE_%d", 2001)),
			Status: identityV1.Position_ON.Enum(),
		},
	}))
	require.NoError(t, repo.Create(ctx, &identityV1.CreatePositionRequest{
		Data: &identityV1.Position{
			Name:   trans.Ptr("MARKERBETA 职位"),
			Code:   trans.Ptr(fmt.Sprintf("POS_SQLITE_%d", 2002)),
			Status: identityV1.Position_ON.Enum(),
		},
	}))

	// 无过滤：应返回全部 2 条
	all, err := repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(2), all.Total, "无过滤时应统计全部 2 条")
	require.Len(t, all.Items, 2, "无过滤时应返回 2 条")
	for _, item := range all.Items {
		// 列表读视图：status 经回填如实呈现；type 未显式指定、按列默认落库并回填。
		require.Equal(t, identityV1.Position_ON, item.GetStatus(), "列表读视图应回填 status")
		require.Equal(t, identityV1.Position_REGULAR, item.GetType(), "列表读视图应回填列默认 type")
	}

	// contains 过滤：仅命中名称含 MARKERALPHA 的那条
	filtered, err := repo.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_Query{
			Query: `{"name__contains":"MARKERALPHA"}`,
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), filtered.Total, "contains 过滤应只统计命中行")
	require.Len(t, filtered.Items, 1, "contains 过滤应只返回命中行")
	require.Contains(t, filtered.Items[0].GetName(), "MARKERALPHA", "命中行应为含标记的那条")
}

// TestPositionRepoSqlite_Get 验证 PositionRepo.Get 按主键查询的命中与未命中，
// 以及按名称/编码查询在平台上下文（tid=0）下被租户闸门拒绝。
func TestPositionRepoSqlite_Get(t *testing.T) {
	repo := newPositionRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &identityV1.CreatePositionRequest{
		Data: &identityV1.Position{
			Name:   trans.Ptr("sqlite查询职位"),
			Code:   trans.Ptr(fmt.Sprintf("POS_SQLITE_%d", 3001)),
			Status: identityV1.Position_ON.Enum(),
		},
	}))

	rows, err := repo.entClient.Client().Position.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := uint32(rows[0].ID)

	// 命中：按主键
	gotByID, err := repo.Get(ctx, &identityV1.GetPositionRequest{
		QueryBy: &identityV1.GetPositionRequest_Id{Id: createdID},
	})
	require.NoError(t, err, "按主键查询已存在记录应命中")
	require.Equal(t, "sqlite查询职位", gotByID.GetName(), "命中记录的 name 应与写入一致")
	require.Equal(t, entPosition.StatusOn, *rows[0].Status, "status 应经转换器落库为 ON")
	// 读视图：status 经回填如实呈现；type 未显式指定、按列默认 REGULAR 落库并回填。
	require.Equal(t, identityV1.Position_ON, gotByID.GetStatus(), "读视图应回填 status")
	require.Equal(t, identityV1.Position_REGULAR, gotByID.GetType(), "读视图应回填列默认 type")

	// 未命中：不存在的主键
	_, err = repo.Get(ctx, &identityV1.GetPositionRequest{
		QueryBy: &identityV1.GetPositionRequest_Id{Id: 99999},
	})
	require.Error(t, err, "查询不存在的主键应返回错误")

	// 平台上下文下按名称查询：跨租户多行风险，应被租户闸门拒绝
	_, err = repo.Get(ctx, &identityV1.GetPositionRequest{
		QueryBy: &identityV1.GetPositionRequest_Name{Name: "sqlite查询职位"},
	})
	require.Error(t, err, "平台上下文按 name 查询应被拒绝")

	// 平台上下文下按编码查询：同上
	_, err = repo.Get(ctx, &identityV1.GetPositionRequest{
		QueryBy: &identityV1.GetPositionRequest_Code{Code: fmt.Sprintf("POS_SQLITE_%d", 3001)},
	})
	require.Error(t, err, "平台上下文按 code 查询应被拒绝")
}

// TestPositionRepoSqlite_Update 验证 PositionRepo.Update 在单字段 updateMask 下
// 只更新掩码内字段，掩码外字段保持原值。
func TestPositionRepoSqlite_Update(t *testing.T) {
	repo := newPositionRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &identityV1.CreatePositionRequest{
		Data: &identityV1.Position{
			Name:   trans.Ptr("更新前职位名"),
			Code:   trans.Ptr(fmt.Sprintf("POS_SQLITE_%d", 4001)),
			Status: identityV1.Position_ON.Enum(),
		},
	}))

	rows, err := repo.entClient.Client().Position.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := uint32(rows[0].ID)

	err = repo.Update(ctx, &identityV1.UpdatePositionRequest{
		Id:         createdID,
		Data:       &identityV1.Position{Name: trans.Ptr("更新后职位名")},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"name"}},
	})
	require.NoError(t, err, "单字段掩码更新应成功")

	after, err := repo.entClient.Client().Position.Query().Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "更新后职位名", *after.Name, "掩码内字段 name 应被更新")
	require.Equal(t, fmt.Sprintf("POS_SQLITE_%d", 4001), *after.Code, "掩码外字段 code 应保持原值")
}

// TestPositionRepoSqlite_TypeAllValuesLand 验证全部 6 个岗位类型枚举值
// 经 typeConverter 的 ToEntity 转换后如实落库，并在读路径（Get）经
// queryEnumsAndBackfill 如实回填。
//
// 历史缺陷取证：proto 枚举名与 ent 枚举 DB 值此前在 LEADER↔LEAD 一处错位
// （其余 5 值两侧全大写一致、往返正常），converter 按 proto 枚举名直转后
// LEADER 产出非法枚举值被列校验拒绝——显式指定领导岗位类型从未生效过。
// 本测试对全部 6 值逐一断言如实落库。
//
// 第二处历史缺陷：读路径（Get/List）DTO 的 type/status 为指针字段，mapper
// 的枚举转换对无法赋入指针字段而丢弃（与通知渠道 Type 读丢失同型），仓内
// 现经 converter 统一回填——落库断言走 ent 行值，回填断言走 DTO 读视图。
func TestPositionRepoSqlite_TypeAllValuesLand(t *testing.T) {
	repo := newPositionRepoSqlite(t)
	// 注入系统级 ViewerContext，满足 ent mixin 的多租户隐私规则要求
	ctx := enttest.NewSystemViewerCtx(context.Background())

	cases := []struct {
		protoType identityV1.Position_Type
		wantEnt   entPosition.Type
	}{
		{identityV1.Position_REGULAR, entPosition.TypeRegular},
		{identityV1.Position_LEADER, entPosition.TypeLead},
		{identityV1.Position_MANAGER, entPosition.TypeManager},
		{identityV1.Position_INTERN, entPosition.TypeIntern},
		{identityV1.Position_CONTRACT, entPosition.TypeContract},
		{identityV1.Position_OTHER, entPosition.TypeOther},
	}
	for i, c := range cases {
		code := fmt.Sprintf("POS_SQLITE_TYPE_%d", 6000+i)
		require.NoError(t, repo.Create(ctx, &identityV1.CreatePositionRequest{
			Data: &identityV1.Position{
				Name:   trans.Ptr("类型落库职位"),
				Code:   trans.Ptr(code),
				Status: identityV1.Position_ON.Enum(),
				Type:   c.protoType.Enum(),
			},
		}), "类型 %v 创建应成功", c.protoType)

		rows, err := repo.entClient.Client().Position.Query().
			Where(entPosition.CodeEQ(code)).
			All(ctx)
		require.NoError(t, err)
		require.Len(t, rows, 1, "按编码应反查到刚写入的行")
		require.Equal(t, c.wantEnt, rows[0].Type, "类型 %v 应经转换器如实落库", c.protoType)

		// 读路径：Get 按主键命中后，type/status 应经回填在 DTO 视图如实呈现。
		got, err := repo.Get(ctx, &identityV1.GetPositionRequest{
			QueryBy: &identityV1.GetPositionRequest_Id{Id: uint32(rows[0].ID)},
		})
		require.NoError(t, err, "按主键读取应命中")
		require.Equal(t, c.protoType, got.GetType(), "读视图应回填类型 %v", c.protoType)
		require.Equal(t, identityV1.Position_ON, got.GetStatus(), "读视图应回填状态")
	}
}

// TestPositionRepoSqlite_Delete 验证 PositionRepo.Delete 删除记录后表内计数归零，
// 且删除不存在的记录返回错误。
func TestPositionRepoSqlite_Delete(t *testing.T) {
	repo := newPositionRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &identityV1.CreatePositionRequest{
		Data: &identityV1.Position{
			Name:   trans.Ptr("待删除职位"),
			Code:   trans.Ptr(fmt.Sprintf("POS_SQLITE_%d", 5001)),
			Status: identityV1.Position_ON.Enum(),
		},
	}))

	rows, err := repo.entClient.Client().Position.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := uint32(rows[0].ID)

	require.NoError(t, repo.Delete(ctx, &identityV1.DeletePositionRequest{
		QueryBy: &identityV1.DeletePositionRequest_Id{Id: createdID},
	}), "删除已存在记录应成功")

	count, err := repo.entClient.Client().Position.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count, "删除后 position 表计数应归零")

	// PositionRepo.Delete 走按谓词的批量删除：删除不存在的 ID 命中 0 行、不报错（与
	// TenantRepo 的 DeleteOneID 报 NotFound 不同），表内仍为 0 条。
	require.NoError(t, repo.Delete(ctx, &identityV1.DeletePositionRequest{
		QueryBy: &identityV1.DeletePositionRequest_Id{Id: 99999},
	}), "按谓词批量删除不存在的 ID 应为无命中且不报错")

	count, err = repo.entClient.Client().Position.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count, "表内计数应保持为 0")
}
