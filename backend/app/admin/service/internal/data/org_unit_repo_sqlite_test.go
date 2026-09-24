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
	"go-wind-admin/app/admin/service/internal/data/ent/orgunit"
	"go-wind-admin/app/admin/service/internal/data/ent/userorgunit"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newOrgUnitRepoSqlite 用 enttest helper 构造一个可直接做 CRUD 的 OrgUnitRepo。
// 白盒构造逐字段复刻 NewOrgUnitRepo 的初始化；其依赖的 UserOrgUnitRepo
// 在生产构造器中由 DI 注入，这里在同一 entclient 上内联构造（逐字段复刻
// NewUserOrgUnitRepo），保证两者共享同一个 SQLite 内存库。
func newOrgUnitRepoSqlite(t *testing.T) *OrgUnitRepo {
	t.Helper()
	entClient := enttest.NewEntClientForTest(t)
	userOrgUnitRepo := &UserOrgUnitRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
		statusConverter: mapper.NewEnumTypeConverter[identityV1.UserOrgUnit_Status, userorgunit.Status](
			identityV1.UserOrgUnit_Status_name,
			identityV1.UserOrgUnit_Status_value,
		),
	}
	repo := &OrgUnitRepo{
		entClient:       entClient,
		log:             bLogger.NewHelper(bLogger.NopLogger()),
		userOrgUnitRepo: userOrgUnitRepo,
		mapper:          mapper.NewCopierMapper[identityV1.OrgUnit, ent.OrgUnit](),
		typeConverter: mapper.NewEnumTypeConverter[identityV1.OrgUnit_Type, orgunit.Type](
			identityV1.OrgUnit_Type_name,
			identityV1.OrgUnit_Type_value,
		),
		statusConverter: mapper.NewEnumTypeConverter[identityV1.OrgUnit_Status, orgunit.Status](
			identityV1.OrgUnit_Status_name,
			identityV1.OrgUnit_Status_value,
		),
	}

	repo.init()

	return repo
}

// TestOrgUnitRepoSqlite_Create 端到端验证 OrgUnitRepo.Create 落库，
// 并确认 Create 后置的物化路径（path）计算也随事务写入。
func TestOrgUnitRepoSqlite_Create(t *testing.T) {
	repo := newOrgUnitRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	err := repo.Create(ctx, &identityV1.CreateOrgUnitRequest{
		Data: &identityV1.OrgUnit{
			Name:   trans.Ptr("sqlite测试单元A"),
			Code:   trans.Ptr(fmt.Sprintf("ORG_SQLITE_%d", 1001)),
			Status: identityV1.OrgUnit_ON.Enum(),
			Type:   identityV1.OrgUnit_COMPANY.Enum(),
		},
	})
	require.NoError(t, err, "通过 repo.Create 写入 SQLite 应成功")

	rows, err := repo.entClient.Client().OrgUnit.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1, "SQLite 中应有 1 条 org_unit 记录")
	require.Equal(t, "sqlite测试单元A", *rows[0].Name, "name 应按 Create 载荷落库")
	require.Equal(t, fmt.Sprintf("ORG_SQLITE_%d", 1001), *rows[0].Code, "code 应按 Create 载荷落库")
	require.NotNil(t, rows[0].Status, "status 枚举应经转换器落库")
	require.Equal(t, orgunit.StatusOn, *rows[0].Status, "proto OrgUnit_ON 应映射为 ent StatusOn")
	require.NotNil(t, rows[0].Type, "type 枚举应经转换器落库")
	require.Equal(t, orgunit.TypeCompany, *rows[0].Type, "proto OrgUnit_COMPANY 应映射为 ent TypeCompany")
	// 根节点物化路径应为 "/<ID>/" 形式（各根前缀互不相同）
	require.NotNil(t, rows[0].Path, "Create 后置的树路径计算应写入 path")
	require.Equal(t, fmt.Sprintf("/%d/", rows[0].ID), *rows[0].Path, "根节点物化路径应为 /<ID>/")
}

// TestOrgUnitRepoSqlite_List 验证 OrgUnitRepo.List 的分页列表与
// contains 模糊搜索过滤语义（仓规：搜索条件一律 contains）。
func TestOrgUnitRepoSqlite_List(t *testing.T) {
	repo := newOrgUnitRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &identityV1.CreateOrgUnitRequest{
		Data: &identityV1.OrgUnit{
			Name: trans.Ptr("MARKERALPHA 单元"),
			Code: trans.Ptr(fmt.Sprintf("ORG_SQLITE_%d", 2001)),
		},
	}))
	require.NoError(t, repo.Create(ctx, &identityV1.CreateOrgUnitRequest{
		Data: &identityV1.OrgUnit{
			Name: trans.Ptr("MARKERBETA 单元"),
			Code: trans.Ptr(fmt.Sprintf("ORG_SQLITE_%d", 2002)),
		},
	}))

	// 无过滤：两个根节点都应返回
	all, err := repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(2), all.Total, "无过滤时应统计全部 2 条")
	require.Len(t, all.Items, 2, "无过滤时应返回 2 条根节点")
	for _, item := range all.Items {
		// 列表读视图：status/type 未显式指定、按列默认（ON/DEPARTMENT）落库并经回填如实呈现。
		require.Equal(t, identityV1.OrgUnit_ON, item.GetStatus(), "列表读视图应回填列默认 status")
		require.Equal(t, identityV1.OrgUnit_DEPARTMENT, item.GetType(), "列表读视图应回填列默认 type")
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

// TestOrgUnitRepoSqlite_Get 验证 OrgUnitRepo.Get 按主键查询的命中与未命中。
func TestOrgUnitRepoSqlite_Get(t *testing.T) {
	repo := newOrgUnitRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &identityV1.CreateOrgUnitRequest{
		Data: &identityV1.OrgUnit{
			Name:   trans.Ptr("sqlite查询单元"),
			Code:   trans.Ptr(fmt.Sprintf("ORG_SQLITE_%d", 3001)),
			Status: identityV1.OrgUnit_ON.Enum(),
			Type:   identityV1.OrgUnit_TEAM.Enum(),
		},
	}))

	rows, err := repo.entClient.Client().OrgUnit.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := uint32(rows[0].ID)

	// 命中：按主键
	gotByID, err := repo.Get(ctx, &identityV1.GetOrgUnitRequest{
		QueryBy: &identityV1.GetOrgUnitRequest_Id{Id: createdID},
	})
	require.NoError(t, err, "按主键查询已存在记录应命中")
	require.Equal(t, "sqlite查询单元", gotByID.GetName(), "命中记录的 name 应与写入一致")
	// 读视图：status/type 均应经回填如实呈现写入值（type 为非默认值，证明真实往返）。
	require.Equal(t, identityV1.OrgUnit_ON, gotByID.GetStatus(), "读视图应回填 status")
	require.Equal(t, identityV1.OrgUnit_TEAM, gotByID.GetType(), "读视图应回填 type")

	// 未命中：不存在的主键
	_, err = repo.Get(ctx, &identityV1.GetOrgUnitRequest{
		QueryBy: &identityV1.GetOrgUnitRequest_Id{Id: 99999},
	})
	require.Error(t, err, "查询不存在的主键应返回错误")
}

// TestOrgUnitRepoSqlite_Update 验证 OrgUnitRepo.Update 在单字段 updateMask 下
// 只更新掩码内字段，掩码外字段保持原值。
func TestOrgUnitRepoSqlite_Update(t *testing.T) {
	repo := newOrgUnitRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &identityV1.CreateOrgUnitRequest{
		Data: &identityV1.OrgUnit{
			Name: trans.Ptr("更新前单元名"),
			Code: trans.Ptr(fmt.Sprintf("ORG_SQLITE_%d", 4001)),
		},
	}))

	rows, err := repo.entClient.Client().OrgUnit.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := uint32(rows[0].ID)

	err = repo.Update(ctx, &identityV1.UpdateOrgUnitRequest{
		Id:         createdID,
		Data:       &identityV1.OrgUnit{Name: trans.Ptr("更新后单元名")},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"name"}},
	})
	require.NoError(t, err, "单字段掩码更新应成功")

	after, err := repo.entClient.Client().OrgUnit.Query().Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "更新后单元名", *after.Name, "掩码内字段 name 应被更新")
	require.Equal(t, fmt.Sprintf("ORG_SQLITE_%d", 4001), *after.Code, "掩码外字段 code 应保持原值")
}

// TestOrgUnitRepoSqlite_Delete 验证 OrgUnitRepo.Delete 删除记录后表内计数归零。
// Delete 的子树收集依赖按方言生成的递归 CTE（仅 MySQL/PG 有分支），SQLite 下查询串为空、
// 返回 0 行，随后按 [自身ID] 执行删除——即 SQLite 集成测试实际走的是"无子节点单删"路径。
func TestOrgUnitRepoSqlite_Delete(t *testing.T) {
	repo := newOrgUnitRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &identityV1.CreateOrgUnitRequest{
		Data: &identityV1.OrgUnit{
			Name: trans.Ptr("sqlite待删单元"),
			Code: trans.Ptr(fmt.Sprintf("ORG_SQLITE_%d", 5001)),
		},
	}))

	rows, err := repo.entClient.Client().OrgUnit.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := uint32(rows[0].ID)

	require.NoError(t, repo.Delete(ctx, &identityV1.DeleteOrgUnitRequest{
		QueryBy: &identityV1.DeleteOrgUnitRequest_Id{Id: createdID},
	}), "删除已存在的无子节点单元应成功")

	count, err := repo.entClient.Client().OrgUnit.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count, "删除后 org_unit 表计数应归零")
}
