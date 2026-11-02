// AiMessageRepo 的 SQLite 内存库测试（白盒，包内测试）。
//
// 消息行是 LLM 上下文的唯一来源，测试盯两件事：
//   - ListRecentByConversation 的排序与裁剪契约：必须"按会话过滤 + 取最近 N 条 +
//     时间升序返回"——顺序错了发给模型的上下文就是乱的，模型会拿用户的话当自己的回复；
//   - role 枚举往返：role 是实体侧可空指针枚举列（Optional+Nillable），经 converter
//     指针对双向精确命中是读路径本就流通的形态，这里钉住读视图（按枚举铁律不做回填）。
package data

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	aiV1 "go-wind-admin/api/gen/go/ai/service/v1"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/aimessage"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// aimessageRoleOf 把 proto 角色枚举转为实体枚举（种子直插用，与 repo 的 converter 同映射）。
func aimessageRoleOf(r aiV1.AiRole) aimessage.Role {
	switch r {
	case aiV1.AiRole_ASSISTANT:
		return aimessage.RoleASSISTANT
	case aiV1.AiRole_SYSTEM:
		return aimessage.RoleSYSTEM
	default:
		return aimessage.RoleUSER
	}
}

// seedConversation 直插一个真实会话行：conversation_id 是指向会话的外键
//（_pragma=foreign_keys(1) 开启），消息必须挂在存在的会话上。
func seedConversation(t *testing.T, entClient *entCrud.EntClient[*ent.Client], ctx context.Context, title string) uint32 {
	t.Helper()
	row, err := entClient.Client().AiConversation.Create().
		SetTitle(title).
		SetUserID(33).
		SetTenantID(7).
		Save(ctx)
	require.NoError(t, err, "种子会话应直插成功")
	return row.ID
}

// newMessage 构造归属 user 33 / 租户 7 的用户消息载荷，字段可按需覆盖。
func newMessage(conversationID uint32, role aiV1.AiRole, content string) *aiV1.AiMessage {
	return &aiV1.AiMessage{
		ConversationId:   trans.Ptr(conversationID),
		Role:             role.Enum(),
		Content:          trans.Ptr(content),
		ModelName:        trans.Ptr("qwen-max"),
		PromptTokens:     trans.Ptr(uint32(120)),
		CompletionTokens: trans.Ptr(uint32(480)),
		DurationMs:       trans.Ptr(uint32(2300)),
		UserId:           trans.Ptr(uint32(33)),
		TenantId:         trans.Ptr(uint32(7)),
	}
}

// TestAiMessageRepoSqlite_Create 验证字段级落库与 role 枚举双向映射：
// role 按 proto 枚举名落为实体枚举字面量、token 快照/耗时/错误信息保值、
// created_at 显式写入、显式 ID 直插；nil 载荷 BadRequest。
func TestAiMessageRepoSqlite_Create(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewAiMessageRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	convID := seedConversation(t, entClient, ctx, "冒烟会话")

	entity, err := repo.Create(ctx, newMessage(convID, aiV1.AiRole_USER, "帮我写个冒烟测试"))
	require.NoError(t, err, "完整载荷的 Create 应成功")
	require.NotZero(t, entity.ID)

	require.Equal(t, convID, *entity.ConversationID, "会话外键必须落库")
	require.NotNil(t, entity.Role)
	require.Equal(t, aimessage.RoleUSER, *entity.Role, "proto 枚举名应经 converter 落为实体枚举字面量")
	require.Equal(t, "帮我写个冒烟测试", *entity.Content)
	require.Equal(t, "qwen-max", *entity.ModelName)
	require.Equal(t, uint32(120), *entity.PromptTokens)
	require.Equal(t, uint32(480), *entity.CompletionTokens)
	require.Equal(t, uint32(2300), *entity.DurationMs)
	require.Equal(t, uint32(33), *entity.UserID, "归属用户必须落库：owner 过滤与越权兜底全靠它")
	require.Equal(t, uint32(7), *entity.TenantID)
	require.NotNil(t, entity.CreatedAt, "created_at 必须显式写入：ListRecentByConversation 按它排序")
	require.Nil(t, entity.ErrorMessage, "成功消息不应有错误信息")

	// 失败的助手回复：error_message 有值、ASSISTANT 枚举落对字面量
	failed := newMessage(convID, aiV1.AiRole_ASSISTANT, "")
	failed.ErrorMessage = trans.Ptr("upstream timeout")
	failed.Id = trans.Ptr(uint32(99002))
	entity2, err := repo.Create(ctx, failed)
	require.NoError(t, err)
	require.Equal(t, uint32(99002), entity2.ID, "载荷带 Id 时应按指定 ID 落库")
	require.Equal(t, aimessage.RoleASSISTANT, *entity2.Role)
	require.Equal(t, "upstream timeout", *entity2.ErrorMessage, "失败原因必须落库：前端靠它展示失败消息")

	// SYSTEM 角色：三态枚举的第三态同样要走对
	system := newMessage(convID, aiV1.AiRole_SYSTEM, "你是一个测试助手")
	entity3, err := repo.Create(ctx, system)
	require.NoError(t, err)
	require.Equal(t, aimessage.RoleSYSTEM, *entity3.Role)

	// 读视图：role 经 converter 回映射为 proto 枚举（Nillable 枚举读路径铁律的兑现）
	dto := repo.ToDTO(entity2)
	require.NotNil(t, dto)
	require.Equal(t, aiV1.AiRole_ASSISTANT, dto.GetRole(), "实体枚举应经 converter 回映射，不能落 UNSPECIFIED 零值")
	require.Equal(t, "upstream timeout", dto.GetErrorMessage())

	require.Nil(t, repo.ToDTO(nil), "nil 实体应回 nil DTO（chat 主链路组装响应的 nil 安全）")

	_, err = repo.Create(ctx, nil)
	require.Error(t, err, "nil 载荷应返回 BadRequest")
}

// TestAiMessageRepoSqlite_List 验证 ownerUserID 强制过滤：>0 只回该用户的消息
// （防 filter 注入越权读他人在其他会话里的消息）、=0 全量；nil 请求 BadRequest。
func TestAiMessageRepoSqlite_List(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewAiMessageRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	convA := seedConversation(t, entClient, ctx, "甲的会话")
	convB := seedConversation(t, entClient, ctx, "乙的会话")

	msgA, err := repo.Create(ctx, newMessage(convA, aiV1.AiRole_USER, "甲的消息"))
	require.NoError(t, err)
	_, err = repo.Create(ctx, func() *aiV1.AiMessage {
		m := newMessage(convB, aiV1.AiRole_USER, "乙的消息")
		m.UserId = trans.Ptr(uint32(44))
		return m
	}())
	require.NoError(t, err)

	own, err := repo.List(ctx, &paginationV1.PagingRequest{}, 33)
	require.NoError(t, err)
	require.Equal(t, uint64(1), own.GetTotal(), "owner 过滤应只统计该用户的消息")
	require.Len(t, own.GetItems(), 1)
	require.Equal(t, msgA.ID, own.GetItems()[0].GetId())

	all, err := repo.List(ctx, &paginationV1.PagingRequest{}, 0)
	require.NoError(t, err)
	require.Equal(t, uint64(2), all.GetTotal(), "owner=0 是管理视角，不应过滤")

	filtered, err := repo.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_Query{
			Query: `{"conversation_id__eq":` + strconv.FormatUint(uint64(convB), 10) + `}`,
		},
	}, 0)
	require.NoError(t, err)
	require.Equal(t, uint64(1), filtered.GetTotal(), "按会话过滤应只统计该会话的消息")
	require.Len(t, filtered.GetItems(), 1)
	require.Equal(t, convB, filtered.GetItems()[0].GetConversationId())

	_, err = repo.List(ctx, nil, 0)
	require.Error(t, err, "nil 分页请求应返回 BadRequest")
}

// TestAiMessageRepoSqlite_ListRecentByConversation 验证 LLM 上下文裁剪的三重契约：
// 只取该会话、最多 N 条（取的是最近 N 条）、返回按时间升序（旧→新）；
// limit<=0 落到默认 20。
func TestAiMessageRepoSqlite_ListRecentByConversation(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewAiMessageRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	convA := seedConversation(t, entClient, ctx, "裁剪窗口会话")
	convB := seedConversation(t, entClient, ctx, "干扰会话")

	// 直插实体种子：repo 建行时 created_at=now，快机器上可能同刻、Update builder 又
	// 保护 created_at 不可改——要钉单调递增的时间序，只能建行时显式 SetCreatedAt。
	// 这正是白盒测试的便利：造一份时间可控的历史，再让被测的读路径去裁剪它。
	base := time.Now().Add(-time.Hour)
	seed := func(convID uint32, offsetMin int, role aiV1.AiRole, content string) uint32 {
		entity, err := entClient.Client().AiMessage.Create().
			SetConversationID(convID).
			SetRole(aimessageRoleOf(role)).
			SetContent(content).
			SetUserID(33).
			SetTenantID(7).
			SetCreatedAt(base.Add(time.Duration(offsetMin) * time.Minute)).
			Save(ctx)
		require.NoError(t, err, "种子消息应直插成功")
		return entity.ID
	}

	// 会话 A 的 5 条消息（时间递增），B 里放 1 条干扰行
	ids := make([]uint32, 0, 5)
	for i := 0; i < 5; i++ {
		role := aiV1.AiRole_USER
		if i%2 == 1 {
			role = aiV1.AiRole_ASSISTANT
		}
		ids = append(ids, seed(convA, i, role, "消息一"))
	}
	seed(convB, 2, aiV1.AiRole_USER, "别家会话的消息")

	// limit=3：取最近 3 条且升序返回
	recent, err := repo.ListRecentByConversation(ctx, convA, 3)
	require.NoError(t, err)
	require.Len(t, recent, 3, "limit 应把历史裁到最近 3 条")
	for i, m := range recent {
		require.Equal(t, convA, *m.ConversationID, "只能取到该会话的消息")
		if i > 0 {
			require.True(t, recent[i-1].CreatedAt.Before(*recent[i].CreatedAt),
				"返回必须按时间升序（旧→新）：LLM 上下文要求按序，第 %d 对失序", i)
		}
	}
	// 最近的 3 条应是后建的 3 行（ids[2..4]）
	require.Equal(t, ids[2], recent[0].ID, "裁剪窗口应保留最近的消息而不是最早的")
	require.Equal(t, ids[4], recent[2].ID)

	// limit 覆盖全部：升序 + 不含别家会话
	all, err := repo.ListRecentByConversation(ctx, convA, 50)
	require.NoError(t, err)
	require.Len(t, all, 5, "limit 足够大时应取全会话")
	require.Equal(t, ids[0], all[0].ID, "最早的行应排最前")

	// limit<=0：落到默认 20（业务默认值，不是报错）
	def, err := repo.ListRecentByConversation(ctx, convA, 0)
	require.NoError(t, err)
	require.Len(t, def, 5, "limit<=0 应回落到默认 20 并取全（不足 20 条时全部返回）")

	// 空会话：空列表而不是报错（新会话还没有任何消息时 LLM 上下文为空）
	emptyConv := seedConversation(t, entClient, ctx, "空会话")
	empty, err := repo.ListRecentByConversation(ctx, emptyConv, 10)
	require.NoError(t, err)
	require.Empty(t, empty, "没有消息的会话应返回空列表")
}

// TestAiMessageRepoSqlite_Delete 验证删除指定行、nil 请求 BadRequest、
// 不存在 ID 无错误且不影响其他行。
func TestAiMessageRepoSqlite_Delete(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewAiMessageRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	convID := seedConversation(t, entClient, ctx, "删除样例会话")

	msgA, err := repo.Create(ctx, newMessage(convID, aiV1.AiRole_USER, "待删消息"))
	require.NoError(t, err)
	msgB, err := repo.Create(ctx, newMessage(convID, aiV1.AiRole_ASSISTANT, "留任消息"))
	require.NoError(t, err)

	require.NoError(t, repo.Delete(ctx, &aiV1.DeleteAiMessageRequest{
		QueryBy: &aiV1.DeleteAiMessageRequest_Id{Id: msgA.ID},
	}), "按存在 ID 删除应成功")

	_, err = repo.GetEntityByID(ctx, msgA.ID)
	require.Error(t, err, "删除后该行应查不到")
	got, err := repo.GetEntityByID(ctx, msgB.ID)
	require.NoError(t, err, "别的行不应被误删")
	require.Equal(t, msgB.ID, got.ID)

	require.Error(t, repo.Delete(ctx, nil), "nil 请求应返回 BadRequest")
	require.NoError(t, repo.Delete(ctx, &aiV1.DeleteAiMessageRequest{
		QueryBy: &aiV1.DeleteAiMessageRequest_Id{Id: 9999999},
	}), "删除不存在的 ID 不应报错（ent 删除 0 行不视为失败）")
}

// TestAiMessageRepoSqlite_GetEntityByID 验证实体取数通道与 NotFound（service 层
// 据此做消息归属校验）。
func TestAiMessageRepoSqlite_GetEntityByID(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewAiMessageRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	convID := seedConversation(t, entClient, ctx, "归属校验会话")

	entity, err := repo.Create(ctx, newMessage(convID, aiV1.AiRole_USER, "归属校验样例"))
	require.NoError(t, err)

	got, err := repo.GetEntityByID(ctx, entity.ID)
	require.NoError(t, err)
	require.Equal(t, entity.ID, got.ID)
	require.Equal(t, "归属校验样例", *got.Content)

	_, err = repo.GetEntityByID(ctx, 424242)
	require.Error(t, err, "不存在的 ID 应返回 NotFound")
	require.Contains(t, err.Error(), "ai message not found")
}
