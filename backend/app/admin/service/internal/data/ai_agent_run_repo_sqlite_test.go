// AiAgentRunRepo 的 SQLite 内存库测试（白盒，包内测试）。
//
// 运行台账是 Agent 同步执行一次图的唯一留痕，测试盯两件事：
//   - 生命周期两步写的完整性：CreateRunning 建 RUNNING 行（trace 先占位 "[]"），
//     SettleResult 回写终态与产出/错误/轨迹/用量/耗时——中间任何一列丢了，排障时
//     就只剩"跑过"两个字；
//   - status 读路径回填：status 是值型枚举列（Default 无 Nillable）→ DTO 可选指针，
//     copier 失配，读路径经 converter 回填——回填失守 Get/List 会把所有运行显示成
//     UNSPECIFIED，台账页面直接失明。
package data

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	aiV1 "go-wind-admin/api/gen/go/ai/service/v1"
	"go-wind-admin/app/admin/service/internal/data/ent/aiagentrun"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// TestAiAgentRunRepoSqlite_CreateRunning 验证 RUNNING 起步行：状态/入参/轨迹占位/
// 租户/操作人/created_at 一次落齐（Run 流程第一步，缺列会让进行中的运行查无对证）。
func TestAiAgentRunRepoSqlite_CreateRunning(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewAiAgentRunRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	id, err := repo.CreateRunning(ctx, 42, `{"city":"杭州"}`, 1001, 7)
	require.NoError(t, err, "CreateRunning 应成功")
	require.NotZero(t, id)

	row, err := entClient.Client().AiAgentRun.Get(ctx, id)
	require.NoError(t, err)
	require.Equal(t, int64(42), *row.AppID, "app_id 必须落库：台账按应用追溯全靠它")
	require.Equal(t, aiagentrun.StatusRunning, row.Status, "起步行状态应为 RUNNING")
	require.NotNil(t, row.Input)
	require.Equal(t, `{"city":"杭州"}`, *row.Input, "运行入参应原样落库")
	require.Equal(t, "[]", row.Trace, "trace 应先占位为空数组：SettleResult 前的运行也要有合法 JSON（实体侧是值型列）")
	require.Equal(t, int64(7), *row.TenantID, "租户归属必须落库")
	require.Equal(t, int64(1001), *row.CreatedBy, "operatorID 应落 created_by")
	require.NotNil(t, row.CreatedAt, "created_at 必须显式写入")
	require.Nil(t, row.Output, "起步行还没有产出")
	require.Nil(t, row.Error, "起步行还没有失败原因")

	cnt, err := entClient.Client().AiAgentRun.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, cnt)
}

// TestAiAgentRunRepoSqlite_SettleResult 验证终态回写：SUCCEEDED 带 output/轨迹/用量/
// 耗时、FAILED 带 error；output/error 空串不覆盖；对不存在的运行报错（静默成功会让
// 失败的运行永远停在 RUNNING，台账里的僵尸行就是这么来的）。
func TestAiAgentRunRepoSqlite_SettleResult(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewAiAgentRunRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	okID, err := repo.CreateRunning(ctx, 42, `{"city":"杭州"}`, 1001, 7)
	require.NoError(t, err)
	failID, err := repo.CreateRunning(ctx, 43, `{"city":"北京"}`, 1001, 7)
	require.NoError(t, err)

	// 成功终态：产出 + 轨迹 + 用量三件套 + 耗时
	require.NoError(t, repo.SettleResult(ctx, okID, aiagentrun.StatusSucceeded,
		"周报已生成", "", `[{"nodeId":"start","status":"ok"}]`,
		AgentRunTokens{Prompt: 120, Completion: 480, Total: 600}, 3200,
	), "成功终态的回写应成功")

	row, err := entClient.Client().AiAgentRun.Get(ctx, okID)
	require.NoError(t, err)
	require.Equal(t, aiagentrun.StatusSucceeded, row.Status, "终态应为 SUCCEEDED")
	require.NotNil(t, row.Output)
	require.Equal(t, "周报已生成", *row.Output, "产出必须落库：同步执行的返回值从台账回读")
	require.Nil(t, row.Error, "成功的运行不得带错误信息")
	require.Equal(t, `[{"nodeId":"start","status":"ok"}]`, row.Trace, "逐节点轨迹必须落库：排障唯一依据")
	require.Equal(t, uint32(120), row.PromptTokens, "输入 token 合计应落库")
	require.Equal(t, uint32(480), row.CompletionTokens)
	require.Equal(t, uint32(600), row.TotalTokens, "总 token 与 usage_logs 同源累计，必须保值")
	require.Equal(t, uint32(3200), row.DurationMs)

	// 失败终态：error 有值、output 不写
	require.NoError(t, repo.SettleResult(ctx, failID, aiagentrun.StatusFailed,
		"", "LLM 调用超时", `[{"nodeId":"start","status":"failed"}]`,
		AgentRunTokens{Prompt: 50, Completion: 0, Total: 50}, 9000,
	), "失败终态的回写应成功")

	row, err = entClient.Client().AiAgentRun.Get(ctx, failID)
	require.NoError(t, err)
	require.Equal(t, aiagentrun.StatusFailed, row.Status)
	require.Equal(t, "LLM 调用超时", *row.Error, "失败原因必须落库：用户可见的失败消息从这里来")
	require.Nil(t, row.Output, "失败的运行没有产出，error 空时 output 不得误写")

	// output/error 传空串时不覆盖（SettleResult 的空串守卫）
	require.NoError(t, repo.SettleResult(ctx, okID, aiagentrun.StatusSucceeded,
		"", "", `[{"nodeId":"start","status":"ok"}]`,
		AgentRunTokens{Prompt: 0, Completion: 0, Total: 0}, 0,
	), "空产出/空错误的重复结算不应报错")
	row, err = entClient.Client().AiAgentRun.Get(ctx, okID)
	require.NoError(t, err)
	require.Equal(t, "周报已生成", *row.Output, "output 空串不得覆盖已写产出")
	require.Nil(t, row.Error, "error 空串不得把原 NULL 覆盖成空串")

	// 对不存在的运行结算：必须报错
	err = repo.SettleResult(ctx, 9999999, aiagentrun.StatusSucceeded,
		"幽灵产出", "", "[]", AgentRunTokens{}, 0)
	require.Error(t, err, "对不存在运行的结算应报错而不是静默假装成功")
}

// TestAiAgentRunRepoSqlite_Get 验证按 ID 取 DTO 与 status 回填：值型枚举列经
// converter 回映射为 proto 枚举（回填失守会落 UNSPECIFIED）；不存在 ID NotFound。
func TestAiAgentRunRepoSqlite_Get(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewAiAgentRunRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	id, err := repo.CreateRunning(ctx, 42, `{"q":1}`, 1001, 7)
	require.NoError(t, err)

	dto, err := repo.Get(ctx, id)
	require.NoError(t, err)
	require.Equal(t, id, dto.GetId())
	require.Equal(t, int64(42), dto.GetAppId())
	require.Equal(t, aiV1.AiAgentRun_RUNNING, dto.GetStatus(),
		"RUNNING 应经回填读回 proto 枚举，不能落 STATUS_UNSPECIFIED 零值")
	require.NotNil(t, dto.Trace)
	require.Equal(t, "[]", dto.GetTrace())

	// 终态枚举同样要回对：FAILED 是台账页最常见的筛选值
	require.NoError(t, repo.SettleResult(ctx, id, aiagentrun.StatusFailed,
		"", "boom", "[]", AgentRunTokens{}, 0))
	dto, err = repo.Get(ctx, id)
	require.NoError(t, err)
	require.Equal(t, aiV1.AiAgentRun_FAILED, dto.GetStatus(), "FAILED 应经回填读回 proto 枚举")
	require.Equal(t, "boom", dto.GetError())

	_, err = repo.Get(ctx, 987654)
	require.Error(t, err, "不存在的 ID 应返回 NotFound")
}

// TestAiAgentRunRepoSqlite_List 验证分页 total、app_id 过滤、status 批量回填、
// nil 请求 BadRequest。
func TestAiAgentRunRepoSqlite_List(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewAiAgentRunRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	idA, err := repo.CreateRunning(ctx, 42, `{"q":1}`, 1001, 7)
	require.NoError(t, err)
	idB, err := repo.CreateRunning(ctx, 43, `{"q":2}`, 1001, 7)
	require.NoError(t, err)
	require.NoError(t, repo.SettleResult(ctx, idB, aiagentrun.StatusFailed,
		"", "boom", "[]", AgentRunTokens{}, 0))

	all, err := repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, int64(2), all.GetTotal())
	require.Len(t, all.GetItems(), 2)
	statusByID := map[int64]aiV1.AiAgentRun_Status{}
	for _, it := range all.GetItems() {
		require.NotNil(t, it.Status, "status 应经读路径回填为指针")
		statusByID[it.GetId()] = it.GetStatus()
	}
	require.Equal(t, aiV1.AiAgentRun_RUNNING, statusByID[idA], "进行中的运行应回 RUNNING")
	require.Equal(t, aiV1.AiAgentRun_FAILED, statusByID[idB], "失败的运行应回 FAILED（回填失守台账页面全变 UNSPECIFIED）")

	filtered, err := repo.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_Query{
			Query: `{"app_id__eq":42}`,
		},
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), filtered.GetTotal(), "按应用过滤应只统计该应用的运行")
	require.Len(t, filtered.GetItems(), 1)
	require.Equal(t, idA, filtered.GetItems()[0].GetId())

	_, err = repo.List(ctx, nil)
	require.Error(t, err, "nil 分页请求应返回 BadRequest")
}

// compile-time 守卫：SettleResult 的终态入参就是 ent 枚举类型（repo 不做二次映射）。
var _ aiagentrun.Status = aiagentrun.StatusRunning
