// 本文件覆盖 internal/data 此前无测试的四个文件：
//   - bi_dashboard_revision_repo.go：BI 仪表盘版本快照仓——唯一索引 (dashboard_id, version)、
//     payload 逐字节透传、version desc 排序、分页默认值、空结果非 nil、Get NotFound；
//   - ai_agent_revision_repo.go：Agent 图快照仓——守卫、created_at desc + id 兜底排序、
//     Graph 逐字节透传、同 app 可重复快照（无唯一索引）、Get NotFound；
//   - tree_walk.go：collectTreeIDs 纯函数——环收敛、菱形去重、两种泛型实例化；
//   - audit_created_at.go：auditCreatedAt 的三态语义。
// ent 装配复用 enttest（SQLite 内存库）+ repo_testkit 构造器，与包内其他测试同范式。
package data

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	"google.golang.org/protobuf/types/known/timestamppb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	aiV1 "go-wind-admin/api/gen/go/ai/service/v1"
	biV1 "go-wind-admin/api/gen/go/bi/service/v1"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// revisionPayload 模拟设计器快照原文：嵌套 + 中文 + 数字。
const revisionPayload = `{"charts":[{"id":"c1","title":"月度销售额"}],"theme":"ocean-dark"}`

// seedBiDashboard 直连 ent 造一张主表仪表盘行（revision.dashboard_id 是外键，
// enttest 开启外键约束，必须先有主表行才能落快照）。
// code 后缀用计数器而不是 UnixNano：Windows 时钟粒度粗，同一测试内两次连续
// 调用会撞出相同纳秒值，触发 UNIQUE(code, tenant_id) 冲突（曾稳定复现）。
var seedBiDashboardSeq atomic.Int64

func seedBiDashboard(t *testing.T, entClient *ent.Client, ctx context.Context) int64 {
	t.Helper()
	row, err := entClient.BiDashboard.Create().
		SetName("销售分析看板").
		SetCode(fmt.Sprintf("dash-%d-%d", time.Now().UnixNano(), seedBiDashboardSeq.Add(1))).
		SetPayload("{}").
		Save(ctx)
	require.NoError(t, err)
	return row.ID
}

// seedAiAgentApp 直连 ent 造一个 Agent 应用主表行（revision.app_id 是外键）。
func seedAiAgentApp(t *testing.T, entClient *ent.Client, ctx context.Context) int64 {
	t.Helper()
	row, err := entClient.AiAgentApp.Create().
		SetName("巡检助手-" + fmt.Sprintf("%d", time.Now().UnixNano())).
		Save(ctx)
	require.NoError(t, err)
	return row.ID
}


// --- BI 仪表盘版本快照仓 ---

func TestBiDashboardRevisionRepo_CreateGuards(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewBiDashboardRevisionRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	require.Error(t, repo.Create(ctx, nil), "nil revision must be rejected")
	require.Error(t, repo.Create(ctx, &biV1.BiDashboardRevision{}), "dashboard_id=0 must be rejected")
}

func TestBiDashboardRevisionRepo_CreateAndVersionUniqueIndex(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewBiDashboardRevisionRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())
	dashID := seedBiDashboard(t, entClient.Client(), ctx)

	require.NoError(t, repo.Create(ctx, &biV1.BiDashboardRevision{
		DashboardId: dashID,
		Version:     3,
		Name:        trans.Ptr("v3 快照"),
		Payload:     trans.Ptr(revisionPayload),
		Source:      trans.Ptr("SAVE"),
		CreatedBy:   trans.Ptr(int64(9)),
	}))

	// 落库侧直查：payload 逐字节无损、source 原样。
	rows, err := entClient.Client().BiDashboardRevision.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, uint32(3), *rows[0].Version)
	require.Equal(t, revisionPayload, *rows[0].Payload)
	require.Equal(t, "SAVE", string(*rows[0].Source))
	require.Equal(t, int64(9), *rows[0].CreatedBy)

	// 唯一索引 (dashboard_id, version)：同键二次 Create 报错且不落行。
	err = repo.Create(ctx, &biV1.BiDashboardRevision{DashboardId: dashID, Version: 3})
	require.Error(t, err, "duplicate (dashboard_id, version) must violate the unique index")
	count, err := entClient.Client().BiDashboardRevision.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestBiDashboardRevisionRepo_SourceDefaultOnEmpty(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewBiDashboardRevisionRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())
	dashID := seedBiDashboard(t, entClient.Client(), ctx)

	// Source 空串：schema 默认 SAVE。
	require.NoError(t, repo.Create(ctx, &biV1.BiDashboardRevision{DashboardId: dashID, Version: 1}))
	row, err := entClient.Client().BiDashboardRevision.Query().Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "SAVE", string(*row.Source))
}

func TestBiDashboardRevisionRepo_ListOrderingAndPaging(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewBiDashboardRevisionRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())
	dashID := seedBiDashboard(t, entClient.Client(), ctx)
	otherDashID := seedBiDashboard(t, entClient.Client(), ctx)

	for _, v := range []uint32{1, 2, 3, 4} {
		require.NoError(t, repo.Create(ctx, &biV1.BiDashboardRevision{DashboardId: dashID, Version: v}))
	}
	// 另一 dashboard 的行不得混入。
	require.NoError(t, repo.Create(ctx, &biV1.BiDashboardRevision{DashboardId: otherDashID, Version: 99}))

	// 默认排序 version desc, id desc（新快照在前）。
	items, total, err := repo.List(ctx, 1, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, int64(4), total)
	for i := 0; i < len(items)-1; i++ {
		require.Greater(t, items[i].GetVersion(), items[i+1].GetVersion(),
			"versions must be listed newest-first")
	}

	// 分页：page=2, pageSize=2。
	paged, total2, err := repo.List(ctx, 1, &paginationV1.PagingRequest{
		Page: trans.Ptr(uint32(2)), PageSize: trans.Ptr(uint32(2)),
	})
	require.NoError(t, err)
	require.Equal(t, int64(4), total2)
	require.Len(t, paged, 2)
	require.Equal(t, uint32(2), paged[0].GetVersion())
	require.Equal(t, uint32(1), paged[1].GetVersion())

	// Offset 超出 total：items 空、total 仍正确。
	over, total3, err := repo.List(ctx, 1, &paginationV1.PagingRequest{Page: trans.Ptr(uint32(9)), PageSize: trans.Ptr(uint32(10))})
	require.NoError(t, err)
	require.Empty(t, over)
	require.Equal(t, int64(4), total3)

	// 空结果非 nil（调用方 range 安全）。
	none, total4, err := repo.List(ctx, 404, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.NotNil(t, none)
	require.Empty(t, none)
	require.Zero(t, total4)
}

func TestBiDashboardRevisionRepo_Get(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewBiDashboardRevisionRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())
	dashID := seedBiDashboard(t, entClient.Client(), ctx)

	require.NoError(t, repo.Create(ctx, &biV1.BiDashboardRevision{
		DashboardId: dashID, Version: 7,
		Name:    trans.Ptr("v7"),
		Payload: trans.Ptr(revisionPayload),
	}))
	row, err := entClient.Client().BiDashboardRevision.Query().Only(ctx)
	require.NoError(t, err)

	got, err := repo.Get(ctx, row.ID)
	require.NoError(t, err)
	require.Equal(t, int64(row.ID), got.GetId())
	require.Equal(t, dashID, got.GetDashboardId())
	require.Equal(t, uint32(7), got.GetVersion())
	require.Equal(t, "v7", got.GetName())
	require.Equal(t, revisionPayload, got.GetPayload())

	// CreatedAt 是格式化字符串（2006-01-02 15:04:05），不是 timestamppb。
	// SQLite 测试环境里 ent 不回填 created_at（生产库由 DB 默认值填充），
	// 这里显式 SetCreatedAt 造一行验证格式化映射；nil 时不拷贝（保持 nil）。
	// 注：SetCreatedAt 需要直连 ent 构造。
	at := time.Date(2026, 11, 9, 12, 30, 0, 0, time.Local)
	row2, err := entClient.Client().BiDashboardRevision.Create().
		SetNillableDashboardID(&dashID).
		SetCreatedAt(at).
		Save(ctx)
	require.NoError(t, err)
	got2, err := repo.Get(ctx, row2.ID)
	require.NoError(t, err)
	require.Equal(t, "2026-11-09 12:30:00", got2.GetCreatedAt())
	_, perr := time.ParseInLocation("2006-01-02 15:04:05", got2.GetCreatedAt(), time.Local)
	require.NoError(t, perr, "CreatedAt must be formatted as 2006-01-02 15:04:05")

	// 不存在 → NotFound。
	_, err = repo.Get(ctx, row.ID+1000)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

// --- Agent 图快照仓 ---

func TestAiAgentRevisionRepo_CreateGuards(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewAiAgentRevisionRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	require.Error(t, repo.Create(ctx, nil))
	require.Error(t, repo.Create(ctx, &aiV1.AiAgentRevision{}), "app_id=0 must be rejected")
}

func TestAiAgentRevisionRepo_GraphPassthroughAndRepeatedSnapshots(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewAiAgentRevisionRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())
	appID := seedAiAgentApp(t, entClient.Client(), ctx)

	graph := `{"nodes":[{"key":"start","name":"开始","type":"START"}],"edges":[]}`
	// 同 app 连续两次快照：无唯一索引，均落行。
	require.NoError(t, repo.Create(ctx, &aiV1.AiAgentRevision{AppId: trans.Ptr(appID), Graph: trans.Ptr(graph), Source: trans.Ptr("SAVE")}))
	require.NoError(t, repo.Create(ctx, &aiV1.AiAgentRevision{AppId: trans.Ptr(appID), Graph: trans.Ptr(graph), Source: trans.Ptr("RESTORE")}))

	rows, err := entClient.Client().AiAgentRevision.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, graph, *rows[0].Graph)
	require.Equal(t, "SAVE", string(*rows[0].Source))
	require.Equal(t, "RESTORE", string(*rows[1].Source))

	// List：created_at desc, id desc——快速连续创建时 created_at 可能同值，
	// id 兜底保证后创建的（RESTORE）在前。
	items, total, err := repo.List(ctx, appID, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Greater(t, items[0].GetId(), items[1].GetId(), "later-created snapshot must come first")

	// DTO：Id 是 *int64 指针（与 BI 的非指针不同）。
	require.NotNil(t, items[0].Id)

	// 空结果非 nil。
	none, total2, err := repo.List(ctx, 404, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.NotNil(t, none)
	require.Empty(t, none)
	require.Zero(t, total2)
}

func TestAiAgentRevisionRepo_Get(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewAiAgentRevisionRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())
	appID := seedAiAgentApp(t, entClient.Client(), ctx)

	require.NoError(t, repo.Create(ctx, &aiV1.AiAgentRevision{AppId: trans.Ptr(appID), Graph: trans.Ptr("{}")}))
	row, err := entClient.Client().AiAgentRevision.Query().Only(ctx)
	require.NoError(t, err)

	got, err := repo.Get(ctx, row.ID)
	require.NoError(t, err)
	require.Equal(t, int64(row.ID), got.GetId())
	require.Equal(t, appID, got.GetAppId())

	_, err = repo.Get(ctx, row.ID+500)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

// --- collectTreeIDs（tree_walk.go） ---

func TestCollectTreeIDs_StructShape(t *testing.T) {
	// menu 形态：map[int64][]*ent.Menu + 节点取 ID 函数（用本地结构体等价替身）。
	type node struct{ id int64 }
	tree := map[int64][]node{
		1: {{2}, {3}},
		2: {{4}},
		3: {{5}},
	}
	ids := collectTreeIDs(tree, func(n node) int64 { return n.id }, 1)
	require.ElementsMatch(t, []int64{1, 2, 3, 4, 5}, ids)
	require.Equal(t, int64(1), ids[0], "root must be collected first")

	// 单节点 / 无子键。
	require.Equal(t, []int64{9}, collectTreeIDs(tree, func(n node) int64 { return n.id }, 9))

	// nil map / 空 map：只返回根。
	require.Equal(t, []int64{1}, collectTreeIDs[int64](nil, func(n int64) int64 { return n }, 1))
	require.Equal(t, []int64{1}, collectTreeIDs(map[int64][]node{}, func(n node) int64 { return n.id }, 1))
}

func TestCollectTreeIDs_IdentityShape(t *testing.T) {
	// org_unit 形态：map[int64][]int64 + 恒等函数。
	tree := map[int64][]int64{1: {2, 3}, 2: {4}, 4: {5}}
	ids := collectTreeIDs(tree, func(v int64) int64 { return v }, 1)
	require.ElementsMatch(t, []int64{1, 2, 3, 4, 5}, ids)

	// 深链全量收集。
	deep := map[int64][]int64{}
	for i := int64(1); i < 1000; i++ {
		deep[i] = []int64{i + 1}
	}
	ids = collectTreeIDs(deep, func(v int64) int64 { return v }, 1)
	require.Len(t, ids, 1000)
}

func TestCollectTreeIDs_CyclesConverge(t *testing.T) {
	identity := func(v int64) int64 { return v }

	// 环：A→B→A。
	cycle := map[int64][]int64{1: {2}, 2: {1}}
	ids := collectTreeIDs(cycle, identity, 1)
	require.ElementsMatch(t, []int64{1, 2}, ids)

	// 自环。
	self := map[int64][]int64{1: {1}}
	require.Equal(t, []int64{1}, collectTreeIDs(self, identity, 1))

	// 根在环里（1→2→3→1）。
	ring := map[int64][]int64{1: {2}, 2: {3}, 3: {1}}
	require.ElementsMatch(t, []int64{1, 2, 3}, collectTreeIDs(ring, identity, 1))

	// 菱形：同一 id 从两个父可达，只收集一次。
	diamond := map[int64][]int64{1: {2, 3}, 2: {4}, 3: {4}, 4: {5}}
	ids = collectTreeIDs(diamond, identity, 1)
	require.Len(t, ids, 5)
	require.Equal(t, int64(1), countOf(ids, 4), "diamond target collected exactly once")
}

func countOf(ids []int64, want int64) int64 {
	n := int64(0)
	for _, v := range ids {
		if v == want {
			n++
		}
	}
	return n
}

// --- auditCreatedAt（audit_created_at.go） ---

func TestAuditCreatedAt(t *testing.T) {
	// 非 nil：以 DTO 为准（异步消费回放事件时间）。
	at := time.Date(2026, 11, 9, 12, 0, 0, 0, time.Local)
	require.True(t, auditCreatedAt(timestamppb.New(at)).Equal(at))

	// nil：取当前时间（同步路径行为不变）。
	now := time.Now()
	got := auditCreatedAt(nil)
	require.WithinDuration(t, now, got, time.Second)

	// 零值 Timestamp：同样"以 DTO 为准"——返回零值，不做 now 兜底。
	require.True(t, auditCreatedAt(timestamppb.New(time.Time{})).IsZero())
}
