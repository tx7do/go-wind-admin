package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	aiV1 "go-wind-admin/api/gen/go/ai/service/v1"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/enttest"
	"go-wind-admin/pkg/middleware/auth"
)

// agentGraphV1 / agentGraphV2 两份合法图（start→end 线性），用于快照/恢复往返。
const agentGraphV1 = `{"nodes":[{"id":"start_1","type":"start","data":{}},{"id":"end_1","type":"end","data":{"output":"v1"}}],"edges":[{"id":"e1","source":"start_1","target":"end_1"}]}`
const agentGraphV2 = `{"nodes":[{"id":"start_1","type":"start","data":{}},{"id":"end_1","type":"end","data":{"output":"v2"}}],"edges":[{"id":"e1","source":"start_1","target":"end_1"}]}`

// TestAiAgentRevision_Sqlite_SaveAndRestore 版本快照链路回归（docs/ai_agent_orchestration.md
// 「版本化」，BI 同款口径）：
// ① 更新且 graph 实际变化时，更新前的 graph 以 SAVE 来源留底；
// ② graph 没变的更新不留快照；
// ③ 恢复把快照写回主表，恢复动作本身先以 RESTORE 来源留底当前 graph（可逆）；
// ④ 快照 graph 不过静态校验的恢复被 400 且不落任何痕迹；
// ⑤ 主表删除级联清理快照。
func TestAiAgentRevision_Sqlite_SaveAndRestore(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	ctx := enttest.NewSystemContext(context.Background())

	svc := &AiAgentService{
		log:          bLogger.NewHelper(bLogger.NopLogger()),
		appRepo:      data.NewAiAgentAppRepoForTest(entClient),
		revisionRepo: data.NewAiAgentRevisionRepoForTest(entClient),
	}

	// 直插一行应用（graph v1 基线）
	row, err := entClient.Client().AiAgentApp.Create().
		SetName("快照样例").
		SetNillableGraph(trans.Ptr(agentGraphV1)).
		SetEnabled(true).
		Save(ctx)
	require.NoError(t, err)
	appID := row.ID

	// 模拟登录态（快照记录触发者）
	ctx = auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7})

	// ① 更新保存：graph v1 → v2
	_, err = svc.Update(ctx, &aiV1.UpdateAiAgentAppRequest{
		Id:   appID,
		Data: &aiV1.AiAgentApp{Graph: trans.Ptr(agentGraphV2)},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"graph"}},
	})
	require.NoError(t, err, "更新保存应成功")

	after, err := entClient.Client().AiAgentApp.Get(ctx, appID)
	require.NoError(t, err)
	require.Equal(t, agentGraphV2, *after.Graph)

	listResp, err := svc.ListRevisions(ctx, &aiV1.ListAiAgentRevisionsRequest{AppId: appID})
	require.NoError(t, err)
	require.Equal(t, uint64(1), listResp.GetTotal(), "应有一行 v1 快照")
	revs := listResp.GetItems()
	require.Equal(t, agentGraphV1, revs[0].GetGraph(), "快照记的是更新前的旧 graph")
	require.Equal(t, "SAVE", revs[0].GetSource())
	require.Equal(t, uint64(7), revs[0].GetCreatedBy())

	// ② graph 没变的更新（只改 enabled）不得产生新快照
	_, err = svc.Update(ctx, &aiV1.UpdateAiAgentAppRequest{
		Id:   appID,
		Data: &aiV1.AiAgentApp{Enabled: trans.Ptr(false), Graph: trans.Ptr(agentGraphV2)},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"enabled", "graph"}},
	})
	require.NoError(t, err)
	listSame, err := svc.ListRevisions(ctx, &aiV1.ListAiAgentRevisionsRequest{AppId: appID})
	require.NoError(t, err)
	require.Equal(t, uint64(1), listSame.GetTotal(), "graph 未变化的更新不应留快照")

	// ③ 恢复到 v1 快照：当前 v2 先以 RESTORE 留底，再把 v1 写回
	_, err = svc.RestoreRevision(ctx, &aiV1.RestoreAiAgentRevisionRequest{Id: revs[0].GetId()})
	require.NoError(t, err, "恢复应成功")

	afterRestore, err := entClient.Client().AiAgentApp.Get(ctx, appID)
	require.NoError(t, err)
	require.Equal(t, agentGraphV1, *afterRestore.Graph, "恢复后 graph 回到快照值")

	listAfter, err := svc.ListRevisions(ctx, &aiV1.ListAiAgentRevisionsRequest{AppId: appID})
	require.NoError(t, err)
	require.Equal(t, uint64(2), listAfter.GetTotal(), "恢复前应先留底当前版本")
	revs2 := listAfter.GetItems()
	sources := map[string]bool{}
	for _, r := range revs2 {
		sources[r.GetSource()] = true
	}
	require.True(t, sources["RESTORE"], "应有一行 RESTORE 来源的留底快照")
	require.True(t, sources["SAVE"], "原 SAVE 快照保留")

	// ④ 快照 graph 过不了静态校验的恢复：400 且不落任何痕迹
	bad := &aiV1.AiAgentRevision{
		AppId:     trans.Ptr(appID),
		Graph:     trans.Ptr(`{"nodes":[]}`),
		Source:    trans.Ptr("SAVE"),
		CreatedBy: trans.Ptr(uint64(7)),
	}
	require.NoError(t, svc.revisionRepo.Create(ctx, bad), "直插一条坏快照作为铺垫")
	listBefore, err := svc.ListRevisions(ctx, &aiV1.ListAiAgentRevisionsRequest{AppId: appID})
	require.NoError(t, err)
	totalBefore := listBefore.GetTotal()

	_, err = svc.RestoreRevision(ctx, &aiV1.RestoreAiAgentRevisionRequest{Id: bad.GetId()})
	require.Error(t, err, "恢复非法快照应被拒绝")
	afterBad, err := entClient.Client().AiAgentApp.Get(ctx, appID)
	require.NoError(t, err)
	require.Equal(t, agentGraphV1, *afterBad.Graph, "非法恢复不得改写主表")
	listAfterBad, err := svc.ListRevisions(ctx, &aiV1.ListAiAgentRevisionsRequest{AppId: appID})
	require.NoError(t, err)
	require.Equal(t, totalBefore, listAfterBad.GetTotal(), "非法恢复不得留 RESTORE 痕迹")

	// ⑤ 主表删除级联清理快照
	require.NoError(t, svc.appRepo.Delete(ctx, appID))
	remaining, err := entClient.Client().AiAgentRevision.Query().
		Where().
		All(ctx)
	require.NoError(t, err)
	agentRevisions := 0
	for _, r := range remaining {
		if r.AppID != nil && *r.AppID == appID {
			agentRevisions++
		}
	}
	require.Zero(t, agentRevisions, "主表删除后快照应被级联清理")
}
