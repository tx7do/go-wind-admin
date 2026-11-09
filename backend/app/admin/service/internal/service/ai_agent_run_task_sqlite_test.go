package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	entgo "github.com/tx7do/go-crud/entgo"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/aiagentrun"
	"go-wind-admin/app/admin/service/internal/data/ent/internalmessagerecipient"
	"go-wind-admin/app/admin/service/internal/data/enttest"
	"go-wind-admin/pkg/task"
)

// agentRunTaskSmokeGraph execute 契约的最小脚本图：start(必填 question) → script → end。
const agentRunTaskSmokeGraph = `{"nodes":[` +
	`{"id":"start_1","type":"start","position":{"x":0,"y":0},"data":{"inputVars":[{"name":"question","label":"问题","required":true}]}},` +
	`{"id":"script_1","type":"script","position":{"x":0,"y":0},"data":{"language":"lua","source":"function execute()\n  local c = __get_ctx()\n  local start = c.get(\"start_1\")\n  local q = start and start.question or \"\"\n  __set_ctx(\"answer\", \"脚本输出:\" .. q)\n  return true\nend"}},` +
	`{"id":"end_1","type":"end","position":{"x":0,"y":0},"data":{"output":"{{script_1.answer}}"}}` +
	`],"edges":[` +
	`{"id":"e1","source":"start_1","target":"script_1"},` +
	`{"id":"e2","source":"script_1","target":"end_1"}]}`

// newAgentRunTaskServiceForTest 白盒构造被测服务：Run 全依赖（app/run/usage 仓 + noop
// SSE 发布者）+ 站内信白盒（收件仓真库、偏好仓缺省 fail-open、发布者 noop）。
func newAgentRunTaskServiceForTest(t *testing.T, entClient *entgo.EntClient[*ent.Client]) *AiAgentService {
	t.Helper()
	recipientRepo := data.NewInternalMessageRecipientRepoForTest(entClient)
	internalMessages := &InternalMessageService{
		log:                          bLogger.NewHelper(bLogger.NopLogger()),
		internalMessageRecipientRepo: recipientRepo,
		// 收件行租户打标的数据源：notifyUserId=5 → 租户 7（与冒烟应用归属租户一致）
		userRepo:                   &internalMessageServiceUserRepoStub{tenantByUserID: map[uint64]uint64{5: 7}},
		notificationPreferenceRepo: nil, // 未接线 = 无静音偏好，推送放行（fail-open）
		internalMessagePublisher:   noopInternalMessagePublisher{},
	}
	return &AiAgentService{
		log:              bLogger.NewHelper(bLogger.NopLogger()),
		appRepo:          data.NewAiAgentAppRepoForTest(entClient),
		runRepo:          data.NewAiAgentRunRepoForTest(entClient),
		usageLogRepo:     data.NewAiUsageLogRepoForTest(entClient),
		scriptRuntime:    newScriptRuntimeForTest(t, entClient, true),
		internalMessages: internalMessages,
		internalMsgRepo:  data.NewInternalMessageRepoForTest(entClient),
		runPublisher:     noopAiAgentRunPublisher{},
	}
}

// TestAiAgentRunTask_Sqlite_EndToEnd 定时触发任务桥（agent_run）：
// ① enabled 应用 → handler 执行 → 台账 SUCCEEDED + 产出含脚本回声 + 站内信收件行产生；
// ② NotifyUserId=0 → 不产生收件行（只看台账）；
// ③ disabled 应用 → 跳过不执行（不落台账、不通知）。
func TestAiAgentRunTask_Sqlite_EndToEnd(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	ctx := enttest.NewSystemContext(context.Background())
	svc := newAgentRunTaskServiceForTest(t, entClient)

	// 两个应用：启用（带通知人）/ 停用
	enabledApp, err := entClient.Client().AiAgentApp.Create().
		SetName("定时冒烟-启用").
		SetNillableGraph(trans.Ptr(agentRunTaskSmokeGraph)).
		SetEnabled(true).
		SetNillableTenantID(trans.Ptr(uint64(7))).
		Save(ctx)
	require.NoError(t, err)
	_, err = entClient.Client().AiAgentApp.Create().
		SetName("定时冒烟-停用").
		SetNillableGraph(trans.Ptr(agentRunTaskSmokeGraph)).
		SetEnabled(false).
		Save(ctx)
	require.NoError(t, err)

	// ① 启用应用 + NotifyUserId=5 → 执行成功并通知
	require.NoError(t, svc.AsyncAgentRun(task.AgentRunTaskType, &task.AgentRunTaskData{
		AppId:        enabledApp.ID,
		Input:        map[string]string{"question": "什么是退款政策"},
		NotifyUserId: 5,
	}), "enabled 应用应执行成功")

	runs, err := entClient.Client().AiAgentRun.Query().
		Where(aiagentrun.AppIDEQ(enabledApp.ID)).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, runs, 1, "应落一行运行台账")
	errText, outText := "", ""
	if runs[0].Error != nil {
		errText = *runs[0].Error
	}
	if runs[0].Output != nil {
		outText = *runs[0].Output
	}
	t.Logf("run status=%s error=%q output=%q", runs[0].Status, errText, outText)
	require.Equal(t, aiagentrun.StatusSucceeded, runs[0].Status)
	require.Contains(t, outText, "脚本输出:什么是退款政策")
	require.Equal(t, uint64(0), *runs[0].CreatedBy, "机器身份执行：created_by=0")

	recipients, err := entClient.Client().InternalMessageRecipient.Query().
		Where(internalmessagerecipient.RecipientUserIDEQ(5)).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, recipients, 1, "NotifyUserId=5 应产生一条收件行")

	// ② NotifyUserId=0 → 只执行不通知
	require.NoError(t, svc.AsyncAgentRun(task.AgentRunTaskType, &task.AgentRunTaskData{
		AppId: enabledApp.ID,
		Input: map[string]string{"question": "第二问"},
	}))
	recipients2, err := entClient.Client().InternalMessageRecipient.Query().
		Where(internalmessagerecipient.RecipientUserIDEQ(5)).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, recipients2, 1, "NotifyUserId=0 不应新增收件行")

	// ③ disabled 应用 → 跳过（不落台账）
	disabledRowsBefore, err := entClient.Client().AiAgentRun.Query().All(ctx)
	require.NoError(t, err)
	before := len(disabledRowsBefore)
	require.NoError(t, svc.AsyncAgentRun(task.AgentRunTaskType, &task.AgentRunTaskData{
		AppId: enabledApp.ID + 1,
	}))
	disabledRowsAfter, err := entClient.Client().AiAgentRun.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, disabledRowsAfter, before, "停用应用应被跳过、不落台账")
}
