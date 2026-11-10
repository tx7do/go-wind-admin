// AiMessageService 的 SQLite 内存库测试（白盒，包内测试）。
//
// 覆盖目标：
//   - List 的强制归属过滤：同一会话里他人的消息怎么都读不到——连 filter 注入
//     （构造 user_id 条件）也绕不过，因为服务层把 operator.UserId 交给 repo 后
//     repo 层无条件 AND 上 user_id 谓词（防越权兜底的最后一道闸）；
//   - Delete 的归属校验：自己的消息可删、他人的消息 NotFound（不泄露存在性）；
//   - 入参守卫：nil 请求体 / 无身份上下文。
package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	entCrud "github.com/tx7do/go-crud/entgo"
	aiV1 "go-wind-admin/api/gen/go/ai/service/v1"
	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/aimessage"
	"go-wind-admin/app/admin/service/internal/data/enttest"
	"go-wind-admin/pkg/middleware/auth"
)

// newAiMessageServiceForTest 白盒复刻 NewAiMessageService 的字段初始化。
func newAiMessageServiceForTest(t *testing.T, entClient *entCrud.EntClient[*ent.Client]) *AiMessageService {
	t.Helper()
	return &AiMessageService{
		log:  bLogger.NewHelper(bLogger.NopLogger()),
		repo: data.NewAiMessageRepoForTest(entClient),
	}
}

// aiMsgSeedMessage 直插一条归属指定用户/会话的消息行。
func aiMsgSeedMessage(t *testing.T, entClient *entCrud.EntClient[*ent.Client], ctx context.Context, conversationID, userID int64, content string) int64 {
	t.Helper()
	row, err := entClient.Client().AiMessage.Create().
		SetNillableConversationID(trans.Ptr(conversationID)).
		SetNillableUserID(trans.Ptr(userID)).
		SetNillableContent(trans.Ptr(content)).
		Save(ctx)
	require.NoError(t, err, "种子消息应直插成功")
	return row.ID
}

// TestAiMessageService_ListForcedOwnerFilter 验证列表的强制归属过滤：
// 同一会话里甲乙各有消息，甲的列表只出甲的；伪造 user_id 条件的 filter 注入
// 也改变不了"只能读自己"这条底线。
func TestAiMessageService_ListForcedOwnerFilter(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newAiMessageServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	conv := aiChatSeedConversation(t, entClient, ctx, "共用会话", 33)
	aiMsgSeedMessage(t, entClient, ctx, conv, 33, "甲的消息一")
	aiMsgSeedMessage(t, entClient, ctx, conv, 33, "甲的消息二")
	aiMsgSeedMessage(t, entClient, ctx, conv, 44, "乙的消息")
	aiMsgSeedMessage(t, entClient, ctx, conv, 44, "乙的消息二")

	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 33})

	// 按会话过滤：只回自己的消息
	resp, err := svc.List(opCtx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_Query{
			Query: fmt.Sprintf(`{"conversation_id__eq":%d}`, conv),
		},
	})
	require.NoError(t, err)
	require.Equal(t, int64(2), resp.GetTotal(), "同一会话里甲只能看到自己的两条消息")
	for _, item := range resp.GetItems() {
		require.Equal(t, int64(33), item.GetUserId(), "列表里的消息必须全归属当前用户")
	}

	// filter 注入 attempt：伪造 user_id 条件想读乙的消息——归属谓词强制 AND，读不到
	resp, err = svc.List(opCtx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_Query{
			Query: fmt.Sprintf(`{"conversation_id__eq":%d,"user_id__eq":44}`, conv),
		},
	})
	require.NoError(t, err)
	require.Equal(t, int64(0), resp.GetTotal(), "伪造 user_id 的 filter 注入必须被归属谓词压制")

	// 不带过滤：也只见自己的（跨会话同样成立）
	resp, err = svc.List(opCtx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, int64(2), resp.GetTotal(), "不带过滤时甲也应只看到自己的消息")

	// 守卫：无身份与 nil 请求
	_, err = svc.List(ctx, &paginationV1.PagingRequest{})
	require.Error(t, err, "无操作者身份的消息列表必须被拒绝")
	_, err = svc.List(opCtx, nil)
	require.Error(t, err, "nil 分页请求应返回 BadRequest")
}

// TestAiMessageService_DeleteOwnership 验证删除归属：自己的消息删除生效、
// 他人的消息 NotFound、不存在的消息报错、nil 请求被拒。
func TestAiMessageService_DeleteOwnership(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newAiMessageServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	conv := aiChatSeedConversation(t, entClient, ctx, "删除校验会话", 33)
	ownMsg := aiMsgSeedMessage(t, entClient, ctx, conv, 33, "甲的待删消息")
	othersMsg := aiMsgSeedMessage(t, entClient, ctx, conv, 44, "乙的消息")

	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 33})

	// 他人的消息：NotFound（把越权删藏进 404，不暴露消息存在性）
	_, err := svc.Delete(opCtx, &aiV1.DeleteAiMessageRequest{
		QueryBy: &aiV1.DeleteAiMessageRequest_Id{Id: othersMsg},
	})
	require.Error(t, err, "删除他人消息必须被拒绝")
	require.Contains(t, err.Error(), "ai message not found")

	// 他人消息必须原样还在（拒绝≠删除）
	row, err := entClient.Client().AiMessage.Query().
		Where(aimessage.IDEQ(othersMsg)).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(44), *row.UserID, "被拒绝删除的他人消息必须原样保留")

	// 自己的消息：删除生效
	_, err = svc.Delete(opCtx, &aiV1.DeleteAiMessageRequest{
		QueryBy: &aiV1.DeleteAiMessageRequest_Id{Id: ownMsg},
	})
	require.NoError(t, err, "删除自己的消息应成功")
	cnt, err := entClient.Client().AiMessage.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, cnt, "删除后只剩乙的消息")

	// 不存在的消息
	_, err = svc.Delete(opCtx, &aiV1.DeleteAiMessageRequest{
		QueryBy: &aiV1.DeleteAiMessageRequest_Id{Id: 424242},
	})
	require.Error(t, err, "删除不存在的消息应报错")

	// nil 请求体：GetId()=0 → 归属校验先拦下
	_, err = svc.Delete(opCtx, nil)
	require.Error(t, err, "nil 请求体应被拒绝")

	// 无身份
	_, err = svc.Delete(ctx, &aiV1.DeleteAiMessageRequest{
		QueryBy: &aiV1.DeleteAiMessageRequest_Id{Id: ownMsg},
	})
	require.Error(t, err, "无操作者身份的删除必须被拒绝")
}
