package data

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	entCrud "github.com/tx7do/go-crud/entgo"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/permissionapi"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newPermissionApiRepoSqlite 在给定 enttest client 上白盒构造 PermissionApiRepo，
// 逐字段复刻 NewPermissionApiRepo（简单构造，无 init()）。
func newPermissionApiRepoSqlite(t *testing.T, entClient *entCrud.EntClient[*ent.Client]) *PermissionApiRepo {
	t.Helper()
	return &PermissionApiRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
	}
}

// TestPermissionApiRepoSqlite_AssignAndListAndDelete 验证 AssignApi 落库、
// ListApiIDs 按权限列出、Delete 按权限清空（两侧父行经直建提供）。
func TestPermissionApiRepoSqlite_AssignAndListAndDelete(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newPermissionApiRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	// 父行：权限点与 API 资源各一（表上 required 列：name/code 走必填 setter）
	permRow, err := entClient.Client().Permission.Create().
		SetName("关联测试权限点").
		SetCode("sqlite_pa_assign_perm").
		Save(ctx)
	require.NoError(t, err, "直建父 permission 应成功")
	apiRow, err := entClient.Client().Api.Create().
		SetNillablePath(trans.Ptr("/sqlite/pa/assign")).
		SetNillableMethod(trans.Ptr("GET")).
		Save(ctx)
	require.NoError(t, err, "直建父 api 应成功")

	// AssignApi：写入关联
	require.NoError(t, repo.AssignApi(ctx, permRow.ID, apiRow.ID), "AssignApi 应成功")
	linked, err := entClient.Client().PermissionApi.Query().
		Where(permissionapi.PermissionIDEQ(permRow.ID)).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, linked, "AssignApi 后应有 1 条关联记录")

	// ListApiIDs：按权限列出关联的 API ID
	apiIDs, err := repo.ListApiIDs(ctx, []uint32{permRow.ID})
	require.NoError(t, err)
	require.Equal(t, []uint32{apiRow.ID}, apiIDs, "ListApiIDs 应返回关联的 API ID")

	// ListApiIDs：未知权限返回空
	none, err := repo.ListApiIDs(ctx, []uint32{876543})
	require.NoError(t, err)
	require.Empty(t, none, "无关联的权限应返回空列表")

	// Delete：按权限清空
	require.NoError(t, repo.Delete(ctx, permRow.ID), "Delete 应成功")
	after, err := entClient.Client().PermissionApi.Query().
		Where(permissionapi.PermissionIDEQ(permRow.ID)).
		Count(ctx)
	require.NoError(t, err)
	require.Zero(t, after, "Delete 后该权限的关联应为 0")
}

// TestPermissionApiRepoSqlite_AssignApisReplaces 验证 AssignApis 的替换语义：
// 后一次分配清理前一次不在集合内的关联（CleanNotExistApis），只保留最新集合。
func TestPermissionApiRepoSqlite_AssignApisReplaces(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newPermissionApiRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	permRow, err := entClient.Client().Permission.Create().
		SetName("替换语义权限点").
		SetCode("sqlite_pa_replace_perm").
		Save(ctx)
	require.NoError(t, err)
	apiA, err := entClient.Client().Api.Create().
		SetNillablePath(trans.Ptr("/sqlite/pa/replace/a")).
		SetNillableMethod(trans.Ptr("GET")).
		Save(ctx)
	require.NoError(t, err)
	apiB, err := entClient.Client().Api.Create().
		SetNillablePath(trans.Ptr("/sqlite/pa/replace/b")).
		SetNillableMethod(trans.Ptr("POST")).
		Save(ctx)
	require.NoError(t, err)

	// 第一次：{A}
	require.NoError(t, repo.AssignApis(ctx, permRow.ID, []uint32{apiA.ID}))
	idsA, err := repo.ListApiIDs(ctx, []uint32{permRow.ID})
	require.NoError(t, err)
	require.Equal(t, []uint32{apiA.ID}, idsA, "第一次分配后应只含 A")

	// 第二次：{B} —— A 应被清理
	require.NoError(t, repo.AssignApis(ctx, permRow.ID, []uint32{apiB.ID}))
	idsB, err := repo.ListApiIDs(ctx, []uint32{permRow.ID})
	require.NoError(t, err)
	require.Equal(t, []uint32{apiB.ID}, idsB, "第二次分配后应只含 B（A 被清理）")
	total, err := entClient.Client().PermissionApi.Query().
		Where(permissionapi.PermissionIDEQ(permRow.ID)).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, total, "替换语义下该权限的关联总数应为 1")

	// 空集合：清空全部
	require.NoError(t, repo.AssignApis(ctx, permRow.ID, nil))
	idsC, err := repo.ListApiIDs(ctx, []uint32{permRow.ID})
	require.NoError(t, err)
	require.Empty(t, idsC, "空集合分配应清空全部关联")
}

// TestPermissionApiRepoSqlite_DeleteByPermissionIDs 验证按权限集合清理关联。
func TestPermissionApiRepoSqlite_DeleteByPermissionIDs(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newPermissionApiRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	permA, err := entClient.Client().Permission.Create().
		SetName("集合清理权限点A").
		SetCode("sqlite_pa_del_perm_a").
		Save(ctx)
	require.NoError(t, err)
	permB, err := entClient.Client().Permission.Create().
		SetName("集合清理权限点B").
		SetCode("sqlite_pa_del_perm_b").
		Save(ctx)
	require.NoError(t, err)
	apiRow, err := entClient.Client().Api.Create().
		SetNillablePath(trans.Ptr("/sqlite/pa/del")).
		SetNillableMethod(trans.Ptr("GET")).
		Save(ctx)
	require.NoError(t, err)

	require.NoError(t, repo.AssignApi(ctx, permA.ID, apiRow.ID))
	require.NoError(t, repo.AssignApi(ctx, permB.ID, apiRow.ID))
	cntA, err := entClient.Client().PermissionApi.Query().
		Where(permissionapi.PermissionIDEQ(permA.ID)).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, cntA)
	cntB, err := entClient.Client().PermissionApi.Query().
		Where(permissionapi.PermissionIDEQ(permB.ID)).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, cntB)

	// 只清 A：B 保留
	require.NoError(t, repo.DeleteByPermissionIDs(ctx, []uint32{permA.ID}), "DeleteByPermissionIDs 应成功")
	cntA, err = entClient.Client().PermissionApi.Query().
		Where(permissionapi.PermissionIDEQ(permA.ID)).Count(ctx)
	require.NoError(t, err)
	require.Zero(t, cntA, "指定权限的关联应被清除")
	cntB, err = entClient.Client().PermissionApi.Query().
		Where(permissionapi.PermissionIDEQ(permB.ID)).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, cntB, "未指定权限的关联应保留")

	// 全清
	require.NoError(t, repo.DeleteByPermissionIDs(ctx, []uint32{permA.ID, permB.ID}))
	total, err := entClient.Client().PermissionApi.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, total, "全量清理后表内行数应为 0")
}
