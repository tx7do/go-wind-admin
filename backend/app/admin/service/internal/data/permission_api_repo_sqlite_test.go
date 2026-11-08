package data

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	entCrud "github.com/tx7do/go-crud/entgo"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/api"
	"go-wind-admin/app/admin/service/internal/data/ent/permissionapi"
	"go-wind-admin/app/admin/service/internal/data/enttest"
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
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

// newApiRepoWithBindingRepoSqlite 组装带绑定 repo 依赖的 ApiRepo（SyncMerge/
// 级联清理需要），字段复刻 NewApiRepoForTest。
func newApiRepoWithBindingRepoSqlite(t *testing.T, entClient *entCrud.EntClient[*ent.Client]) *ApiRepo {
	t.Helper()
	repo := NewApiRepoForTest(entClient)
	repo.permissionApi = newPermissionApiRepoSqlite(t, entClient)
	return repo
}

// createEndpointApi 直建一条端点身份完整的 API 行，返回其实体。
func createEndpointApi(t *testing.T, entClient *entCrud.EntClient[*ent.Client], ctx context.Context, path, method string) *ent.Api {
	t.Helper()
	row, err := entClient.Client().Api.Create().
		SetNillablePath(trans.Ptr(path)).
		SetNillableMethod(trans.Ptr(method)).
		Save(ctx)
	require.NoError(t, err, "直建 api 应成功")
	return row
}

// TestPermissionApiRepoSqlite_AssignAndListAndDelete 验证 AssignApis 把 api 引用
// 解析成端点快照落库、ListApiIDs 反解回当前 id、Delete 按权限清空。
func TestPermissionApiRepoSqlite_AssignAndListAndDelete(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newPermissionApiRepoSqlite(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	// 父行：权限点与 API 资源各一（表上 required 列：name/code 走必填 setter）
	permRow, err := entClient.Client().Permission.Create().
		SetName("关联测试权限点").
		SetCode("sqlite_pa_assign_perm").
		Save(ctx)
	require.NoError(t, err, "直建父 permission 应成功")
	apiRow := createEndpointApi(t, entClient, ctx, "/sqlite/pa/assign", "GET")

	// AssignApis：写入关联（id → 端点快照）
	require.NoError(t, repo.AssignApis(ctx, permRow.ID, []uint32{apiRow.ID}), "AssignApis 应成功")
	linked, err := entClient.Client().PermissionApi.Query().
		Where(permissionapi.PermissionIDEQ(permRow.ID)).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, linked, 1, "AssignApis 后应有 1 条关联记录")
	require.Equal(t, "/sqlite/pa/assign", *linked[0].Path, "关联应落端点快照 path")
	require.Equal(t, "GET", *linked[0].Method, "关联应落端点快照 method")

	// ListEndpoints：按权限列出端点快照
	endpoints, err := repo.ListEndpoints(ctx, []uint32{permRow.ID})
	require.NoError(t, err)
	require.Len(t, endpoints, 1)
	require.Equal(t, "/sqlite/pa/assign", endpoints[0].Path)

	// ListApiIDs：端点快照反解回当前 API id
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
	ctx := enttest.NewSystemContext(context.Background())

	permRow, err := entClient.Client().Permission.Create().
		SetName("替换语义权限点").
		SetCode("sqlite_pa_replace_perm").
		Save(ctx)
	require.NoError(t, err)
	apiA := createEndpointApi(t, entClient, ctx, "/sqlite/pa/replace/a", "GET")
	apiB := createEndpointApi(t, entClient, ctx, "/sqlite/pa/replace/b", "POST")

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

// TestPermissionApiRepoSqlite_AssignApisMissingIDs 验证 fail-closed：
// 提交了不存在的 api 引用 → 整体 400 带缺失明细，库内零落库（不吞错铁律）。
func TestPermissionApiRepoSqlite_AssignApisMissingIDs(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newPermissionApiRepoSqlite(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	permRow, err := entClient.Client().Permission.Create().
		SetName("缺失引用权限点").
		SetCode("sqlite_pa_missing_perm").
		Save(ctx)
	require.NoError(t, err)
	apiRow := createEndpointApi(t, entClient, ctx, "/sqlite/pa/missing", "GET")

	err = repo.AssignApis(ctx, permRow.ID, []uint32{apiRow.ID, 987654321})
	require.Error(t, err, "含失效引用的分配应整体失败")
	require.ErrorContains(t, err, "987654321", "错误应带缺失 id 明细")

	cnt, err := entClient.Client().PermissionApi.Query().
		Where(permissionapi.PermissionIDEQ(permRow.ID)).
		Count(ctx)
	require.NoError(t, err)
	require.Zero(t, cnt, "失败分支不应落任何绑定行")
}

// TestPermissionApiRepoSqlite_EndpointBindingSurvivesDrift 零漂移核心用例：
// API 行删除后重建（同端点、新 id），绑定快照原样存活且依然有效——这正是
// 旧 api_id 绑定做不到的事。
func TestPermissionApiRepoSqlite_EndpointBindingSurvivesDrift(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newPermissionApiRepoSqlite(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	permRow, err := entClient.Client().Permission.Create().
		SetName("漂移免疫权限点").
		SetCode("sqlite_pa_drift_perm").
		Save(ctx)
	require.NoError(t, err)
	apiRow := createEndpointApi(t, entClient, ctx, "/sqlite/pa/drift", "DELETE")

	require.NoError(t, repo.AssignApis(ctx, permRow.ID, []uint32{apiRow.ID}))

	// 模拟旧版「接口同步」的删了再加：同一端点、id 重新生成
	require.NoError(t, entClient.Client().Api.DeleteOneID(apiRow.ID).Exec(ctx))
	reborn := createEndpointApi(t, entClient, ctx, "/sqlite/pa/drift", "DELETE")
	require.NotEqual(t, apiRow.ID, reborn.ID, "重建行的 id 应不同于原行（自增推进）")

	// 绑定快照原样存活：不受 id 漂移影响
	endpoints, err := repo.ListEndpoints(ctx, []uint32{permRow.ID})
	require.NoError(t, err)
	require.Len(t, endpoints, 1, "端点删除重建后绑定应原样存活")
	require.Equal(t, "/sqlite/pa/drift", endpoints[0].Path)
	require.Equal(t, "DELETE", endpoints[0].Method)

	// 反解回的是重建后的新 id
	apiIDs, err := repo.ListApiIDs(ctx, []uint32{permRow.ID})
	require.NoError(t, err)
	require.Equal(t, []uint32{reborn.ID}, apiIDs, "ListApiIDs 应反解到重建后的新 id")
}

// TestPermissionApiRepoSqlite_DeleteByEndpoints 验证端点级联清理：
// API 行删除流程按 (path, method) 清绑定，消灭孤儿绑定。
func TestPermissionApiRepoSqlite_DeleteByEndpoints(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newPermissionApiRepoSqlite(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	permA, err := entClient.Client().Permission.Create().
		SetName("级联清理权限点A").
		SetCode("sqlite_pa_cascade_perm_a").
		Save(ctx)
	require.NoError(t, err)
	permB, err := entClient.Client().Permission.Create().
		SetName("级联清理权限点B").
		SetCode("sqlite_pa_cascade_perm_b").
		Save(ctx)
	require.NoError(t, err)
	apiRow := createEndpointApi(t, entClient, ctx, "/sqlite/pa/cascade", "PUT")

	require.NoError(t, repo.AssignApis(ctx, permA.ID, []uint32{apiRow.ID}))
	require.NoError(t, repo.AssignApis(ctx, permB.ID, []uint32{apiRow.ID}))

	// 端点级联：两个权限的该端点绑定都被清，其他端点不受影响
	otherAPI := createEndpointApi(t, entClient, ctx, "/sqlite/pa/other", "GET")
	require.NoError(t, repo.AssignApis(ctx, permA.ID, []uint32{otherAPI.ID}))

	require.NoError(t, repo.DeleteByEndpoints(ctx, entClient.Client(), "/sqlite/pa/cascade", "PUT"), "DeleteByEndpoints 应成功")

	leftA, err := repo.ListEndpoints(ctx, []uint32{permA.ID})
	require.NoError(t, err)
	require.Len(t, leftA, 1, "A 应只剩 other 端点绑定")
	require.Equal(t, "/sqlite/pa/other", leftA[0].Path)

	leftB, err := repo.ListEndpoints(ctx, []uint32{permB.ID})
	require.NoError(t, err)
	require.Empty(t, leftB, "B 的被删端点绑定应被级联清空")
}

// TestPermissionApiRepoSqlite_DeleteByPermissionIDs 验证按权限集合清理关联。
func TestPermissionApiRepoSqlite_DeleteByPermissionIDs(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newPermissionApiRepoSqlite(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

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
	apiRow := createEndpointApi(t, entClient, ctx, "/sqlite/pa/del", "GET")

	require.NoError(t, repo.AssignApis(ctx, permA.ID, []uint32{apiRow.ID}))
	require.NoError(t, repo.AssignApis(ctx, permB.ID, []uint32{apiRow.ID}))
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

// TestApiRepoSqlite_SyncMergePreservesIDs 验证 SyncMerge 的 MERGE 语义：
// id 永不重排、元数据变化才 UPDATE、spec 缺失的行删除且级联清绑定。
func TestApiRepoSqlite_SyncMergePreservesIDs(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newApiRepoWithBindingRepoSqlite(t, entClient)
	paRepo := newPermissionApiRepoSqlite(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	permRow, err := entClient.Client().Permission.Create().
		SetName("同步保号权限点").
		SetCode("sqlite_pa_syncmerge_perm").
		Save(ctx)
	require.NoError(t, err)

	// 首轮同步：两行入库
	spec1 := []*permissionV1.Api{
		{Path: trans.Ptr("/sync/alpha"), Method: trans.Ptr("GET"), Module: trans.Ptr("m1"), Description: trans.Ptr("alpha")},
		{Path: trans.Ptr("/sync/beta"), Method: trans.Ptr("POST"), Module: trans.Ptr("m1")},
	}
	added, updated, removed, err := repo.SyncMerge(ctx, spec1, 0)
	require.NoError(t, err)
	require.Equal(t, 2, added)
	require.Zero(t, updated)
	require.Zero(t, removed)

	alpha, err := entClient.Client().Api.Query().
		Where(api.PathEQ("/sync/alpha")).
		Only(ctx)
	require.NoError(t, err)

	// 权限绑定 alpha，记住绑定行 id
	require.NoError(t, paRepo.AssignApis(ctx, permRow.ID, []uint32{alpha.ID}))
	bindingID, err := entClient.Client().PermissionApi.Query().
		Where(permissionapi.PermissionIDEQ(permRow.ID)).
		FirstID(ctx)
	require.NoError(t, err)

	// 二轮同步：beta 从 spec 消失、alpha 元数据变化、新增 gamma
	spec2 := []*permissionV1.Api{
		{Path: trans.Ptr("/sync/alpha"), Method: trans.Ptr("GET"), Module: trans.Ptr("m1"), Description: trans.Ptr("alpha-v2")},
		{Path: trans.Ptr("/sync/gamma"), Method: trans.Ptr("PUT"), Module: trans.Ptr("m2")},
	}
	added, updated, removed, err = repo.SyncMerge(ctx, spec2, 7)
	require.NoError(t, err)
	require.Equal(t, 1, added, "gamma 应新增")
	require.Equal(t, 1, updated, "alpha 元数据变化应更新")
	require.Equal(t, 1, removed, "beta 应被移除")

	// alpha id 不变（零漂移）：重新取行核对 id 与元数据
	alphaAfter, err := entClient.Client().Api.Query().
		Where(api.PathEQ("/sync/alpha")).
		Only(ctx)
	require.NoError(t, err)
	require.Equal(t, alpha.ID, alphaAfter.ID, "MERGE 后同端点 id 必须不变")
	require.Equal(t, "alpha-v2", *alphaAfter.Description, "变化的元数据应更新")
	require.NotNil(t, alphaAfter.UpdatedBy, "操作人应落 updated_by")
	require.Equal(t, uint32(7), *alphaAfter.UpdatedBy)

	// 三轮同步：alpha 元数据无变化 → 不触发 UPDATE（updated_by 不被覆盖）
	added, updated, removed, err = repo.SyncMerge(ctx, spec2, 9)
	require.NoError(t, err)
	require.Zero(t, added)
	require.Zero(t, updated)
	require.Zero(t, removed)
	alphaAgain, err := entClient.Client().Api.Query().
		Where(api.PathEQ("/sync/alpha")).
		Only(ctx)
	require.NoError(t, err)
	require.Equal(t, uint32(7), *alphaAgain.UpdatedBy, "无变化的行不应被 UPDATE（updated_by 保持上一轮）")

	// beta 被移除时其绑定被级联清理（本用例 beta 无绑定，另行验证删除计数）
	cnt, err := entClient.Client().Api.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, cnt, "同步后应只剩 alpha 与 gamma")

	// 绑定行原样存活（绑定的是端点快照，与 id 无关）
	binding, err := entClient.Client().PermissionApi.Query().
		Where(permissionapi.PermissionIDEQ(permRow.ID)).
		Only(ctx)
	require.NoError(t, err)
	require.Equal(t, bindingID, binding.ID, "绑定行应原样存活")
	require.Equal(t, "/sync/alpha", *binding.Path)
}

// TestApiRepoSqlite_SyncMergeCascadeCleansBindings 验证同步删除段级联清绑定：
// spec 移除的端点，其权限绑定同事务消失（孤儿绑定从机制上不存在）。
func TestApiRepoSqlite_SyncMergeCascadeCleansBindings(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newApiRepoWithBindingRepoSqlite(t, entClient)
	paRepo := newPermissionApiRepoSqlite(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	permRow, err := entClient.Client().Permission.Create().
		SetName("同步级联权限点").
		SetCode("sqlite_pa_synccascade_perm").
		Save(ctx)
	require.NoError(t, err)

	spec1 := []*permissionV1.Api{
		{Path: trans.Ptr("/sync/doomed"), Method: trans.Ptr("GET")},
	}
	_, _, _, err = repo.SyncMerge(ctx, spec1, 0)
	require.NoError(t, err)

	doomed, err := entClient.Client().Api.Query().
		Where(api.PathEQ("/sync/doomed")).
		Only(ctx)
	require.NoError(t, err)
	require.NoError(t, paRepo.AssignApis(ctx, permRow.ID, []uint32{doomed.ID}))

	// 二轮同步：端点从 spec 消失
	spec2 := []*permissionV1.Api{}
	_, _, removed, err := repo.SyncMerge(ctx, spec2, 0)
	require.NoError(t, err)
	require.Equal(t, 1, removed)

	apiCnt, err := entClient.Client().Api.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, apiCnt, "端点行应被删除")

	bindingCnt, err := entClient.Client().PermissionApi.Query().
		Where(permissionapi.PermissionIDEQ(permRow.ID)).
		Count(ctx)
	require.NoError(t, err)
	require.Zero(t, bindingCnt, "被移除端点的绑定应被同事务级联清理")
}

// TestApiRepoSqlite_UpdateRejectsIdentityChange 验证身份不可变守卫：
// Update 提交不同的 path/method → 400；提交相同值放行。
func TestApiRepoSqlite_UpdateRejectsIdentityChange(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewApiRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	row := createEndpointApi(t, entClient, ctx, "/guard/keep", "PATCH")

	// 改 path → 400
	err := repo.Update(ctx, &permissionV1.UpdateApiRequest{
		Id:   row.ID,
		Data: &permissionV1.Api{Path: trans.Ptr("/guard/changed")},
	})
	require.Error(t, err, "改 path 应被身份守卫拒绝")
	require.ErrorContains(t, err, "immutable")

	// 改 method → 400
	err = repo.Update(ctx, &permissionV1.UpdateApiRequest{
		Id:   row.ID,
		Data: &permissionV1.Api{Method: trans.Ptr("POST")},
	})
	require.Error(t, err, "改 method 应被身份守卫拒绝")

	// 提交相同值 + 其他元数据 → 放行
	require.NoError(t, repo.Update(ctx, &permissionV1.UpdateApiRequest{
		Id:   row.ID,
		Data: &permissionV1.Api{Path: trans.Ptr("/guard/keep"), Method: trans.Ptr("PATCH"), Description: trans.Ptr("更新描述")},
	}), "相同身份值的更新应放行")
	after, err := entClient.Client().Api.Query().
		Where(api.PathEQ("/guard/keep")).
		Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "更新描述", *after.Description, "非身份字段应正常更新")
}
