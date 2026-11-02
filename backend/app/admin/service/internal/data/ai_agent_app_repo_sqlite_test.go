// AiAgentAppRepo 的 SQLite 内存库测试（白盒，包内测试）。
//
// 应用表是 Agent 运行的入口闸门，测试盯三件事：
//   - 租户内 name 唯一约束：同租户重名必须转成可读的 400（直接漏 500 会让前端把
//     "名字撞了"当"系统炸了"），跨租户同名必须放行（唯一索引是 (tenant_id, name)）；
//   - enabled 读路径回填：实体 enabled 是值型 bool、DTO 是可选指针，copier 对这一
//     形态失配，读路径批量回填——这条失守所有应用都会显示成"未启用"；
//   - Update 的掩码语义：停用应用（enabled=false）靠掩码点名，掩码外 graph 原文
//     不得被顺手动走（graph 是画布用户的劳动成果）。
package data

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	aiV1 "go-wind-admin/api/gen/go/ai/service/v1"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newAppPayload 构造一份启用状态的应用载荷，字段可按需覆盖。
func newAppPayload(name string) *aiV1.AiAgentApp {
	return &aiV1.AiAgentApp{
		Name:        trans.Ptr(name),
		Description: trans.Ptr("周报汇总应用"),
		Graph:       trans.Ptr(`{"nodes":[{"id":"start"}],"edges":[]}`),
		Enabled:     trans.Ptr(true),
		TenantId:    trans.Ptr(uint32(7)),
	}
}

// TestAiAgentAppRepoSqlite_Create 验证字段级落库、operatorID 落 created_by、
// created_at 显式写入、返回自增 ID；name 缺失与同租户重名转 BadRequest、
// 跨租户同名放行。
func TestAiAgentAppRepoSqlite_Create(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewAiAgentAppRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	id, err := repo.Create(ctx, newAppPayload("周报助手"), 1001)
	require.NoError(t, err, "完整载荷的 Create 应成功")
	require.NotZero(t, id)

	row, err := entClient.Client().AiAgentApp.Get(ctx, id)
	require.NoError(t, err)
	require.Equal(t, "周报助手", row.Name)
	require.Equal(t, "周报汇总应用", *row.Description)
	require.Equal(t, `{"nodes":[{"id":"start"}],"edges":[]}`, *row.Graph, "graph 原文必须保值：运行引擎靠它重放图")
	require.True(t, row.Enabled, "enabled 应按载荷落库")
	require.Equal(t, uint32(7), *row.TenantID, "租户归属必须落库：唯一索引与隔离都按它")
	require.Equal(t, uint32(1001), *row.CreatedBy, "operatorID 应落 created_by")
	require.NotNil(t, row.CreatedAt, "created_at 必须显式写入")

	// 同租户重名：唯一约束冲突必须转成可读的 400，而不是 500
	_, err = repo.Create(ctx, newAppPayload("周报助手"), 1002)
	require.Error(t, err, "同租户重名应被唯一索引拒绝")
	require.Contains(t, err.Error(), "agent app name already exists")

	// 跨租户同名：唯一索引是 (tenant_id, name)，别家可以叫同一个名字
	idOther, err := repo.Create(ctx, func() *aiV1.AiAgentApp {
		p := newAppPayload("周报助手")
		p.TenantId = trans.Ptr(uint32(9))
		return p
	}(), 1002)
	require.NoError(t, err, "跨租户同名应放行（唯一索引按租户分桶）")
	require.NotEqual(t, id, idOther)

	// 入参守卫
	_, err = repo.Create(ctx, nil, 1)
	require.Error(t, err, "nil 载荷应返回 BadRequest")
	_, err = repo.Create(ctx, func() *aiV1.AiAgentApp {
		p := newAppPayload("缺名应用")
		p.Name = nil
		return p
	}(), 1)
	require.Error(t, err, "缺 name 应返回 BadRequest")
	require.Contains(t, err.Error(), "name is required")
}

// TestAiAgentAppRepoSqlite_List 验证分页 total、name contains 过滤、nil 请求
// BadRequest，以及 enabled 指针回填（值型 bool 列 + DTO 指针的已知失配形态，
// 回填失守则停用应用在列表里假装还在运行）。
func TestAiAgentAppRepoSqlite_List(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewAiAgentAppRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	idEnabled, err := repo.Create(ctx, newAppPayload("marker-启用应用"), 1)
	require.NoError(t, err)
	disabled := newAppPayload("别家-停用应用")
	disabled.Enabled = trans.Ptr(false)
	idDisabled, err := repo.Create(ctx, disabled, 1)
	require.NoError(t, err)

	all, err := repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(2), all.GetTotal())
	require.Len(t, all.GetItems(), 2)

	// enabled 回填：停用行也必须如实回 false，不能因 copier 失配全变 true/nil
	enabledFlags := map[uint32]bool{}
	for _, it := range all.GetItems() {
		require.NotNil(t, it.Enabled, "enabled 应经读路径回填为指针")
		enabledFlags[it.GetId()] = it.GetEnabled()
	}
	require.Len(t, enabledFlags, 2)
	require.True(t, enabledFlags[idEnabled], "启用行应回 true")
	require.False(t, enabledFlags[idDisabled], "停用行应如实回 false（回填失守则停用应用假装在运行）")

	filtered, err := repo.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_Query{
			Query: `{"name__contains":"marker"}`,
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), filtered.GetTotal(), "contains 过滤应只统计命中行")
	require.Equal(t, "marker-启用应用", filtered.GetItems()[0].GetName())

	_, err = repo.List(ctx, nil)
	require.Error(t, err, "nil 分页请求应返回 BadRequest")
}

// TestAiAgentAppRepoSqlite_GetAndGetEntityByID 验证按 ID 取 DTO（enabled 指针回填）
// 与实体（Run 流程要 graph 原文与 enabled）两条通道；不存在 ID NotFound。
func TestAiAgentAppRepoSqlite_GetAndGetEntityByID(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewAiAgentAppRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	id, err := repo.Create(ctx, newAppPayload("取数样例"), 1001)
	require.NoError(t, err)

	dto, err := repo.Get(ctx, id)
	require.NoError(t, err)
	require.Equal(t, id, dto.GetId())
	require.Equal(t, "取数样例", dto.GetName())
	require.NotNil(t, dto.Enabled, "DTO 的 enabled 应经 toDTO 回填为指针")
	require.True(t, dto.GetEnabled())

	entity, err := repo.GetEntityByID(ctx, id)
	require.NoError(t, err)
	require.Equal(t, id, entity.ID)
	require.Equal(t, `{"nodes":[{"id":"start"}],"edges":[]}`, *entity.Graph)

	_, err = repo.Get(ctx, 987654)
	require.Error(t, err, "不存在的 ID 应返回 NotFound")
	_, err = repo.GetEntityByID(ctx, 987654)
	require.Error(t, err, "不存在的 ID 应返回 NotFound")
}

// TestAiAgentAppRepoSqlite_Update 验证掩码语义：掩码内字段更新（含 enabled 停用）、
// 掩码外 graph 保持原值、operatorID 落 updated_by、updated_at 写入；无掩码全量更新
// 的 enabled 缺席语义（值型列直写 false）钉住现状；id=0 / nil 返回 BadRequest。
func TestAiAgentAppRepoSqlite_Update(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewAiAgentAppRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	id, err := repo.Create(ctx, newAppPayload("待改应用"), 1001)
	require.NoError(t, err)

	// 停用 + 改描述：graph 原文在掩码外，不许被动
	require.NoError(t, repo.Update(ctx, id, &aiV1.AiAgentApp{
		Description: trans.Ptr("已下线"),
		Enabled:     trans.Ptr(false),
	}, []string{"description", "enabled"}, 2001))

	row, err := entClient.Client().AiAgentApp.Get(ctx, id)
	require.NoError(t, err)
	require.Equal(t, "已下线", *row.Description, "掩码内的 description 应更新")
	require.False(t, row.Enabled, "掩码内的 enabled=false 应生效（停用是运行闸门）")
	require.Equal(t, "待改应用", row.Name, "掩码外未点名的 name 应保持原值")
	require.Equal(t, `{"nodes":[{"id":"start"}],"edges":[]}`, *row.Graph, "掩码外的 graph 原文不得被顺手动走")
	require.Equal(t, uint32(2001), *row.UpdatedBy, "operatorID 应落 updated_by")
	require.NotNil(t, row.UpdatedAt, "更新应写入 updated_at")

	// 无掩码全量更新：name/description/graph 直写；enabled 缺席 → 值型列直写 false
	//（现状钉住：schema 默认 true 被 repo 的 SetEnabled(data.GetEnabled()) 压过，
	//  调用方全量更新时必须显式带上 enabled，否则应用会被静默停用）
	require.NoError(t, repo.Update(ctx, id, &aiV1.AiAgentApp{
		Name:        trans.Ptr("改名应用"),
		Description: trans.Ptr("全量更新"),
		Graph:       trans.Ptr(`{"nodes":[{"id":"start"},{"id":"end"}]}`),
	}, nil, 2002))

	row, err = entClient.Client().AiAgentApp.Get(ctx, id)
	require.NoError(t, err)
	require.Equal(t, "改名应用", row.Name)
	require.Equal(t, "全量更新", *row.Description)
	require.Equal(t, `{"nodes":[{"id":"start"},{"id":"end"}]}`, *row.Graph, "无掩码时 graph 应按载荷直写")
	require.False(t, row.Enabled, "无掩码全量更新未带 enabled 时按现状直写 false（schema 默认 true 被显式 SetEnabled(false) 压过）")

	// 掩码内点名 enabled 才能重新启用
	require.NoError(t, repo.Update(ctx, id, &aiV1.AiAgentApp{Enabled: trans.Ptr(true)}, []string{"enabled"}, 2003))
	row, err = entClient.Client().AiAgentApp.Get(ctx, id)
	require.NoError(t, err)
	require.True(t, row.Enabled, "掩码内点名 enabled=true 应重新启用")

	// 不存在 ID：无效果、不新增行
	require.NoError(t, repo.Update(ctx, 987654, &aiV1.AiAgentApp{Name: trans.Ptr("幽灵应用")}, nil, 1))
	_, err = entClient.Client().AiAgentApp.Get(ctx, 987654)
	require.Error(t, err, "对不存在 ID 的更新不应创建行")

	// 入参守卫
	require.Error(t, repo.Update(ctx, 0, newAppPayload("任意"), nil, 1), "id=0 应返回 BadRequest")
	require.Error(t, repo.Update(ctx, id, nil, nil, 1), "nil 载荷应返回 BadRequest")
}

// TestAiAgentAppRepoSqlite_Delete 验证删除指定行、id=0 BadRequest、
// 不存在 ID 无错误且行数不变。
func TestAiAgentAppRepoSqlite_Delete(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewAiAgentAppRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	idA, err := repo.Create(ctx, newAppPayload("待删应用"), 1)
	require.NoError(t, err)
	_, err = repo.Create(ctx, newAppPayload("留任应用"), 1)
	require.NoError(t, err)

	require.NoError(t, repo.Delete(ctx, idA), "按存在 ID 删除应成功")
	_, err = repo.GetEntityByID(ctx, idA)
	require.Error(t, err, "删除后该行应查不到")

	all, err := repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(1), all.GetTotal(), "只该删掉一行")

	require.Error(t, repo.Delete(ctx, 0), "id=0 应返回 BadRequest")
	// ent 的 DeleteOneID 对 0 行返回 NotFoundError，repo 包装成 500：与按 ID 删实体的
	// UpdateOneID 同型（不存在就是不存在，不能静默假装删掉）
	err = repo.Delete(ctx, 9999999)
	require.Error(t, err, "删除不存在的 ID 应报错（ent DeleteOneID 0 行 NotFound）")
	require.Contains(t, err.Error(), "delete ai agent app failed")
	all, err = repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(1), all.GetTotal(), "失败删除不应影响行数")
}
