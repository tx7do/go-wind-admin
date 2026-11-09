// AiConversationRepo 的 SQLite 内存库测试（白盒，包内测试）。
//
// 会话是聊天场景的归属边界，测试盯三件事：
//   - List 的 ownerUserID 强制过滤：聊天场景不可信前端过滤，repo 必须把"只看自己的
//     会话"钉死在 SQL 里——这条失守就是横向越权读他人会话列表；
//   - Delete 的级联：schema 声明 messages 边 OnDelete Cascade，会话删除时消息行必须
//     随删（孤儿消息行会让"按会话取历史"的上下文裁剪读到幽灵记录）；
//   - TouchLastMessage：chat 主链路每条回复都会触发的高频小写，存在性语义要分明
//     （会话在→更新成功；会话不在→报错而不是静默假装成功）。
package data

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	aiV1 "go-wind-admin/api/gen/go/ai/service/v1"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newConversation 载荷归属 user 33、租户 7，字段可按需覆盖。
func newConversation(title string) *aiV1.AiConversation {
	return &aiV1.AiConversation{
		Title:      trans.Ptr(title),
		ProviderId: trans.Ptr(uint64(11)),
		UserId:     trans.Ptr(uint64(33)),
		TenantId:   trans.Ptr(uint64(7)),
	}
}

// TestAiConversationRepoSqlite_Create 验证字段级落库：标题/提供商/用户/租户归属、
// operatorID 落 created_by、created_at 显式写入、显式 ID 直插；nil 载荷 BadRequest。
func TestAiConversationRepoSqlite_Create(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewAiConversationRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	entity, err := repo.Create(ctx, newConversation("帮我排查内存泄漏"))
	require.NoError(t, err, "完整载荷的 Create 应成功")
	require.NotZero(t, entity.ID)

	require.Equal(t, "帮我排查内存泄漏", *entity.Title, "title 应按载荷落库")
	require.Equal(t, uint64(11), *entity.ProviderID)
	require.Equal(t, uint64(33), *entity.UserID, "归属用户必须落库：owner 过滤全靠它")
	require.Equal(t, uint64(7), *entity.TenantID)
	require.NotNil(t, entity.CreatedAt, "created_at 必须显式写入")
	require.Nil(t, entity.LastMessageAt, "新会话还没有消息，last_message_at 应为 NULL")

	// 显式 ID 直插（chat 主链路迁移/回放场景）
	fixed := newConversation("固定ID会话")
	fixed.Id = trans.Ptr(uint64(88001))
	entity2, err := repo.Create(ctx, fixed)
	require.NoError(t, err)
	require.Equal(t, uint64(88001), entity2.ID, "载荷带 Id 时应按指定 ID 落库")

	_, err = repo.Create(ctx, nil)
	require.Error(t, err, "nil 载荷应返回 BadRequest")
}

// TestAiConversationRepoSqlite_List 验证 ownerUserID 强制过滤：>0 只回该用户的会话、
// =0 全量（管理后台视角）；分页 total 与 nil 请求 BadRequest。
func TestAiConversationRepoSqlite_List(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewAiConversationRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	_, err := repo.Create(ctx, newConversation("用户33的会话"))
	require.NoError(t, err)
	_, err = repo.Create(ctx, func() *aiV1.AiConversation {
		c := newConversation("用户44的会话")
		c.UserId = trans.Ptr(uint64(44))
		return c
	}())
	require.NoError(t, err)
	_, err = repo.Create(ctx, func() *aiV1.AiConversation {
		c := newConversation("租户9的会话")
		c.UserId = trans.Ptr(uint64(55))
		c.TenantId = trans.Ptr(uint64(9))
		return c
	}())
	require.NoError(t, err)

	// ownerUserID>0：聊天场景只可见自己的会话
	own, err := repo.List(ctx, &paginationV1.PagingRequest{}, 33)
	require.NoError(t, err)
	require.Equal(t, uint64(1), own.GetTotal(), "owner 过滤应只统计该用户的会话")
	require.Len(t, own.GetItems(), 1)
	require.Equal(t, "用户33的会话", own.GetItems()[0].GetTitle())

	// ownerUserID=0：管理后台全量视角
	all, err := repo.List(ctx, &paginationV1.PagingRequest{}, 0)
	require.NoError(t, err)
	require.Equal(t, uint64(3), all.GetTotal(), "owner=0 是管理视角，不应过滤")

	// 分页：PageSize 限 2 时 total 仍是全量、items 截断（分页走 Page+PageSize 路径）
	paged, err := repo.List(ctx, &paginationV1.PagingRequest{
		Page:     trans.Ptr(uint32(1)),
		PageSize: trans.Ptr(uint32(2)),
	}, 0)
	require.NoError(t, err)
	require.Equal(t, uint64(3), paged.GetTotal(), "total 应是过滤后全量，不受 page size 影响")
	require.Len(t, paged.GetItems(), 2, "items 应按 page size 截断")

	// 标题 contains 过滤
	filtered, err := repo.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_Query{
			Query: `{"title__contains":"用户33"}`,
		},
	}, 0)
	require.NoError(t, err)
	require.Equal(t, uint64(1), filtered.GetTotal(), "contains 过滤应只统计命中行")

	_, err = repo.List(ctx, nil, 0)
	require.Error(t, err, "nil 分页请求应返回 BadRequest")
}

// TestAiConversationRepoSqlite_GetAndGetEntityByID 验证按 ID 取 DTO 与实体的两条通道，
// 不存在 ID 一律 NotFound（service 层据此做归属校验前的存在性判断）。
func TestAiConversationRepoSqlite_GetAndGetEntityByID(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewAiConversationRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	entity, err := repo.Create(ctx, newConversation("取数样例"))
	require.NoError(t, err)

	dto, err := repo.Get(ctx, &aiV1.GetAiConversationRequest{
		QueryBy: &aiV1.GetAiConversationRequest_Id{Id: entity.ID},
	})
	require.NoError(t, err)
	require.Equal(t, entity.ID, dto.GetId())
	require.Equal(t, "取数样例", dto.GetTitle())
	require.Equal(t, uint64(33), dto.GetUserId())

	got, err := repo.GetEntityByID(ctx, entity.ID)
	require.NoError(t, err)
	require.Equal(t, entity.ID, got.ID)

	_, err = repo.Get(ctx, &aiV1.GetAiConversationRequest{
		QueryBy: &aiV1.GetAiConversationRequest_Id{Id: 987654},
	})
	require.Error(t, err, "不存在的 ID 应返回 NotFound")
	_, err = repo.GetEntityByID(ctx, 987654)
	require.Error(t, err, "不存在的 ID 应返回 NotFound")
	require.Contains(t, err.Error(), "ai conversation not found")

	_, err = repo.Get(ctx, nil)
	require.Error(t, err, "nil 请求应返回 BadRequest")
}

// TestAiConversationRepoSqlite_Update 验证掩码语义：掩码内字段更新（含 last_message_at
// 的 tsOf 转换）、掩码外保持原值、updated_by/updated_at；id=0 / nil 返回 BadRequest；
// 不存在 ID 无效果不新增。
func TestAiConversationRepoSqlite_Update(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewAiConversationRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	entity, err := repo.Create(ctx, newConversation("原始标题"))
	require.NoError(t, err)

	// 改标题 + 钉最近消息时间：掩码外的 provider/tenant 不得被顺手动走
	lastMsg := timestamppb.New(time.Now().Add(-time.Minute))
	require.NoError(t, repo.Update(ctx, &aiV1.UpdateAiConversationRequest{
		Id:         entity.ID,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"title", "last_message_at"}},
		Data: &aiV1.AiConversation{
			Title:         trans.Ptr("改后的标题"),
			LastMessageAt: lastMsg,
			ProviderId:    trans.Ptr(uint64(999)),
		},
	}, ))

	row, err := entClient.Client().AiConversation.Get(ctx, entity.ID)
	require.NoError(t, err)
	require.Equal(t, "改后的标题", *row.Title, "掩码内的 title 应更新")
	require.NotNil(t, row.LastMessageAt)
	require.WithinDuration(t, lastMsg.AsTime(), *row.LastMessageAt, time.Second,
		"掩码内的 last_message_at 应经 tsOf 转换落库")
	require.Equal(t, uint64(11), *row.ProviderID, "掩码外的 provider_id 应保持原值（载荷里的 999 不许写进去）")
	require.Equal(t, uint64(7), *row.TenantID, "租户归属不因更新漂移")
	require.NotNil(t, row.UpdatedAt, "更新应写入 updated_at")

	// updated_by 按载荷盖操作人（无掩码全量更新路径）
	require.NoError(t, repo.Update(ctx, &aiV1.UpdateAiConversationRequest{
		Id:   entity.ID,
		Data: &aiV1.AiConversation{Title: trans.Ptr("二改标题"), UpdatedBy: trans.Ptr(uint64(2001))},
	}))
	row, err = entClient.Client().AiConversation.Get(ctx, entity.ID)
	require.NoError(t, err)
	require.Equal(t, "二改标题", *row.Title)
	require.NotNil(t, row.UpdatedBy)
	require.Equal(t, uint64(2001), *row.UpdatedBy, "updated_by 应按载荷盖成新操作人")
	require.NotNil(t, row.LastMessageAt, "无掩码更新未带 last_message_at 时应保持原值（SetNillable no-op）")

	// 不存在 ID：无效果、不新增行
	require.NoError(t, repo.Update(ctx, &aiV1.UpdateAiConversationRequest{
		Id:   987654,
		Data: &aiV1.AiConversation{Title: trans.Ptr("幽灵会话")},
	}))
	cnt, err := repo.Count(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, uint64(1), cnt.GetCount(), "对不存在 ID 的更新不应新增行")

	// 入参守卫
	require.Error(t, repo.Update(ctx, nil), "nil 请求应返回 BadRequest")
	require.Error(t, repo.Update(ctx, &aiV1.UpdateAiConversationRequest{Id: entity.ID}), "nil Data 应返回 BadRequest")
	require.Error(t, repo.Update(ctx, &aiV1.UpdateAiConversationRequest{
		Data: &aiV1.AiConversation{},
	}), "id=0 应返回 BadRequest")
}

// TestAiConversationRepoSqlite_TouchLastMessage 验证 chat 主链路的高频小写：
// 会话在→last_message_at 被刷成当前时刻；会话不在→报错（静默成功会让消息挂在
// 不存在的会话上还没人知道）。
func TestAiConversationRepoSqlite_TouchLastMessage(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewAiConversationRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	entity, err := repo.Create(ctx, newConversation("待触摸会话"))
	require.NoError(t, err)
	require.Nil(t, entity.LastMessageAt)

	require.NoError(t, repo.TouchLastMessage(ctx, entity.ID), "对存在会话的触摸应成功")

	row, err := entClient.Client().AiConversation.Get(ctx, entity.ID)
	require.NoError(t, err)
	require.NotNil(t, row.LastMessageAt, "触摸后 last_message_at 应被写入")
	require.WithinDuration(t, time.Now(), *row.LastMessageAt, 5*time.Second, "last_message_at 应接近当前时刻")
	require.NotNil(t, row.UpdatedAt)

	err = repo.TouchLastMessage(ctx, 987654)
	require.Error(t, err, "对不存在会话的触摸应报错而不是静默成功")
	require.Contains(t, err.Error(), "update ai conversation failed")
}

// TestAiConversationRepoSqlite_Delete 验证删除会话时消息级联随删（schema 级 Cascade）、
// 删除后行数变化、nil 请求 BadRequest；不存在 ID 无错误。
func TestAiConversationRepoSqlite_Delete(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewAiConversationRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	entity, err := repo.Create(ctx, newConversation("待删会话"))
	require.NoError(t, err)
	msgRepo := NewAiMessageRepoForTest(entClient)
	_, err = msgRepo.Create(ctx, &aiV1.AiMessage{
		ConversationId: trans.Ptr(entity.ID),
		Role:           aiV1.AiRole_USER.Enum(),
		Content:        trans.Ptr("会话内的消息一"),
		UserId:         trans.Ptr(uint64(33)),
	})
	require.NoError(t, err)
	_, err = msgRepo.Create(ctx, &aiV1.AiMessage{
		ConversationId: trans.Ptr(entity.ID),
		Role:           aiV1.AiRole_ASSISTANT.Enum(),
		Content:        trans.Ptr("会话内的消息二"),
		UserId:         trans.Ptr(uint64(33)),
	})
	require.NoError(t, err)

	// 别的会话的消息：级联不能顺手殃及
	other, err := repo.Create(ctx, newConversation("幸存会话"))
	require.NoError(t, err)
	_, err = msgRepo.Create(ctx, &aiV1.AiMessage{
		ConversationId: trans.Ptr(other.ID),
		Role:           aiV1.AiRole_USER.Enum(),
		Content:        trans.Ptr("幸存会话的消息"),
		UserId:         trans.Ptr(uint64(33)),
	})
	require.NoError(t, err)

	require.NoError(t, repo.Delete(ctx, &aiV1.DeleteAiConversationRequest{
		QueryBy: &aiV1.DeleteAiConversationRequest_Id{Id: entity.ID},
	}), "按存在 ID 删除应成功")

	_, err = msgRepo.GetEntityByID(ctx, entity.ID)
	require.Error(t, err, "会话删除后该行应查不到")

	msgCount, err := entClient.Client().AiMessage.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, msgCount, "会话删除必须级联删掉其消息行，只留幸存会话的 1 条")

	cnt, err := repo.Count(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, uint64(1), cnt.GetCount(), "只该删掉一个会话")

	require.Error(t, repo.Delete(ctx, nil), "nil 请求应返回 BadRequest")
	require.NoError(t, repo.Delete(ctx, &aiV1.DeleteAiConversationRequest{
		QueryBy: &aiV1.DeleteAiConversationRequest_Id{Id: 9999999},
	}), "删除不存在的 ID 不应报错（ent 删除 0 行不视为失败）")
}

// TestAiConversationRepoSqlite_ToDTO 验证实转 DTO 的 nil 安全（chat 主链路在 service
// 层组装响应，实体缺席时必须回 nil 而不是崩）。
func TestAiConversationRepoSqlite_ToDTO(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewAiConversationRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	require.Nil(t, repo.ToDTO(nil), "nil 实体应回 nil DTO")

	entity, err := repo.Create(ctx, newConversation("转DTO样例"))
	require.NoError(t, err)
	dto := repo.ToDTO(entity)
	require.NotNil(t, dto)
	require.Equal(t, entity.ID, dto.GetId())
	require.Equal(t, "转DTO样例", dto.GetTitle())
}
