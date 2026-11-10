// AiConversationService 的 SQLite 内存库测试（白盒，包内测试）。
//
// 覆盖目标：
//   - List 只返回当前用户自己的会话（Phase 1 聊天语义：服务层把 operator.UserId
//     传给 repo 的强制谓词，不可信前端过滤）；
//   - Get / Update / Delete 的归属校验：他人会话一律 NotFound（不泄露存在性）；
//   - Update 的盖章语义：updated_by 由服务层强制写入并追加进掩码；
//   - Delete 的级联：会话删除时消息行随删（schema 级 Cascade）；
//   - 入参守卫：nil 请求体 / nil Data / 无身份上下文。
package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	entCrud "github.com/tx7do/go-crud/entgo"
	aiV1 "go-wind-admin/api/gen/go/ai/service/v1"
	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/aiconversation"
	"go-wind-admin/app/admin/service/internal/data/ent/aimessage"
	"go-wind-admin/app/admin/service/internal/data/enttest"
	"go-wind-admin/pkg/middleware/auth"
)

// newAiConversationServiceForTest 白盒复刻 NewAiConversationService 的字段初始化。
func newAiConversationServiceForTest(t *testing.T, entClient *entCrud.EntClient[*ent.Client]) *AiConversationService {
	t.Helper()
	return &AiConversationService{
		log:  bLogger.NewHelper(bLogger.NopLogger()),
		repo: data.NewAiConversationRepoForTest(entClient),
	}
}

// aiConvOperator 构造指定用户的操作者上下文。
func aiConvOperator(ctx context.Context, userID int64) context.Context {
	return auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: userID})
}

// TestAiConversationService_ListOwnOnly 验证列表归属过滤：甲只见自己的会话，
// 乙只见自己的；无身份与 nil 请求被拒。
func TestAiConversationService_ListOwnOnly(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newAiConversationServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	aiChatSeedConversation(t, entClient, ctx, "甲会话一", 33)
	aiChatSeedConversation(t, entClient, ctx, "甲会话二", 33)
	aiChatSeedConversation(t, entClient, ctx, "乙会话", 44)

	opCtx := aiConvOperator(ctx, 33)
	resp, err := svc.List(opCtx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, int64(2), resp.GetTotal(), "甲只能看到自己的两条会话")
	for _, item := range resp.GetItems() {
		require.Equal(t, int64(33), item.GetUserId(), "列表里的会话必须全归属当前用户")
	}

	resp, err = svc.List(aiConvOperator(ctx, 44), &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, int64(1), resp.GetTotal(), "乙只能看到自己的一条会话")

	_, err = svc.List(ctx, &paginationV1.PagingRequest{})
	require.Error(t, err, "无操作者身份的会话列表必须被拒绝")
	_, err = svc.List(opCtx, nil)
	require.Error(t, err, "nil 分页请求应返回 BadRequest")
}

// TestAiConversationService_GetOwnership 验证详情归属校验：自己的回读、
// 他人的与不存在的都按 NotFound 拒绝（不泄露他人会话的存在性）。
func TestAiConversationService_GetOwnership(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newAiConversationServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())
	own := aiChatSeedConversation(t, entClient, ctx, "甲的详情会话", 33)
	others := aiChatSeedConversation(t, entClient, ctx, "乙的详情会话", 44)

	dto, err := svc.Get(aiConvOperator(ctx, 33), &aiV1.GetAiConversationRequest{
		QueryBy: &aiV1.GetAiConversationRequest_Id{Id: own},
	})
	require.NoError(t, err, "自己的会话应可回读")
	require.Equal(t, own, dto.GetId())
	require.Equal(t, int64(33), dto.GetUserId())

	_, err = svc.Get(aiConvOperator(ctx, 33), &aiV1.GetAiConversationRequest{
		QueryBy: &aiV1.GetAiConversationRequest_Id{Id: others},
	})
	require.Error(t, err, "读他人会话必须被拒绝")
	require.Contains(t, err.Error(), "ai conversation not found")

	_, err = svc.Get(aiConvOperator(ctx, 33), &aiV1.GetAiConversationRequest{
		QueryBy: &aiV1.GetAiConversationRequest_Id{Id: 424242},
	})
	require.Error(t, err, "不存在的会话应报 NotFound")

	_, err = svc.Get(ctx, &aiV1.GetAiConversationRequest{
		QueryBy: &aiV1.GetAiConversationRequest_Id{Id: own},
	})
	require.Error(t, err, "无操作者身份的详情必须被拒绝")
}

// TestAiConversationService_UpdateStampAndOwnership 验证更新：他人会话 NotFound、
// 自己的更新成功且 updated_by 被强制盖章（连同掩码追加，保证掩码路径不丢盖章）。
func TestAiConversationService_UpdateStampAndOwnership(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newAiConversationServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())
	own := aiChatSeedConversation(t, entClient, ctx, "更新前的标题", 33)
	others := aiChatSeedConversation(t, entClient, ctx, "乙的更新会话", 44)

	// 入参守卫
	_, err := svc.Update(aiConvOperator(ctx, 33), nil)
	require.Error(t, err, "nil 请求体应被拒绝")
	_, err = svc.Update(aiConvOperator(ctx, 33), &aiV1.UpdateAiConversationRequest{})
	require.Error(t, err, "nil Data 应被拒绝")

	// 他人会话：NotFound
	_, err = svc.Update(aiConvOperator(ctx, 33), &aiV1.UpdateAiConversationRequest{
		Id:   others,
		Data: &aiV1.AiConversation{Title: trans.Ptr("越权改名")},
	})
	require.Error(t, err, "更新他人会话必须被拒绝")
	require.Contains(t, err.Error(), "ai conversation not found")

	// 自己的会话：带掩码更新，updated_by 强制盖章并追加进掩码
	req := &aiV1.UpdateAiConversationRequest{
		Id:         own,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"title"}},
		Data:       &aiV1.AiConversation{Title: trans.Ptr("更新后的标题")},
	}
	_, err = svc.Update(aiConvOperator(ctx, 33), req)
	require.NoError(t, err, "自己的会话更新应成功")
	require.Contains(t, req.UpdateMask.Paths, "updated_by", "服务层应把 updated_by 追加进掩码")

	row, err := entClient.Client().AiConversation.Query().
		Where(aiconversation.IDEQ(own)).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "更新后的标题", *row.Title, "掩码内的标题应更新")
	require.NotNil(t, row.UpdatedBy, "updated_by 应被服务层强制盖章")
	require.Equal(t, int64(33), *row.UpdatedBy)
	require.NotNil(t, row.UpdatedAt, "updated_at 应随更新刷新")

	// nil 掩码更新：updated_by 必须以操作者为准，请求体冒充无效
	_, err = svc.Update(aiConvOperator(ctx, 33), &aiV1.UpdateAiConversationRequest{
		Id:   own,
		Data: &aiV1.AiConversation{Title: trans.Ptr("无掩码更新标题"), UpdatedBy: trans.Ptr(int64(999))},
	})
	require.NoError(t, err)
	row, err = entClient.Client().AiConversation.Query().
		Where(aiconversation.IDEQ(own)).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "无掩码更新标题", *row.Title)
	require.Equal(t, int64(33), *row.UpdatedBy, "updated_by 必须以操作者为准，不可被请求体冒充")

	// 无身份
	_, err = svc.Update(ctx, &aiV1.UpdateAiConversationRequest{Id: own, Data: &aiV1.AiConversation{}})
	require.Error(t, err, "无操作者身份的更新必须被拒绝")
}

// TestAiConversationService_DeleteCascadeMessages 验证删除：他人会话 NotFound、
// 自己的删除成功且消息随会话级联删除、nil 请求被拒。
func TestAiConversationService_DeleteCascadeMessages(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newAiConversationServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())
	own := aiChatSeedConversation(t, entClient, ctx, "待删除会话", 33)
	others := aiChatSeedConversation(t, entClient, ctx, "乙的待删会话", 44)

	// 会话里垫两条消息，验证级联删除
	for _, content := range []string{"消息一", "消息二"} {
		_, err := entClient.Client().AiMessage.Create().
			SetNillableConversationID(trans.Ptr(own)).
			SetNillableRole(trans.Ptr(aimessage.RoleUSER)).
			SetNillableContent(trans.Ptr(content)).
			SetNillableUserID(trans.Ptr(int64(33))).
			Save(ctx)
		require.NoError(t, err, "种子消息应直插成功")
	}

	// 他人会话：NotFound
	_, err := svc.Delete(aiConvOperator(ctx, 33), &aiV1.DeleteAiConversationRequest{
		QueryBy: &aiV1.DeleteAiConversationRequest_Id{Id: others},
	})
	require.Error(t, err, "删除他人会话必须被拒绝")
	require.Contains(t, err.Error(), "ai conversation not found")

	// 自己的会话：删除成功，消息级联消失
	_, err = svc.Delete(aiConvOperator(ctx, 33), &aiV1.DeleteAiConversationRequest{
		QueryBy: &aiV1.DeleteAiConversationRequest_Id{Id: own},
	})
	require.NoError(t, err, "删除自己的会话应成功")

	convCnt, err := entClient.Client().AiConversation.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, convCnt, "只剩乙的会话")

	msgCnt, err := entClient.Client().AiMessage.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, msgCnt, "会话删除时消息必须级联删除（否则孤儿消息会漏进他人的列表）")

	// 不存在的会话：repo 报 NotFound
	_, err = svc.Delete(aiConvOperator(ctx, 33), &aiV1.DeleteAiConversationRequest{
		QueryBy: &aiV1.DeleteAiConversationRequest_Id{Id: 424242},
	})
	require.Error(t, err, "删除不存在的会话应报错")

	// nil 请求体：GetId()=0 → 归属校验先拦下（NotFound），不可直达 repo 的无条件删除
	_, err = svc.Delete(aiConvOperator(ctx, 33), nil)
	require.Error(t, err, "nil 请求体应被拒绝")

	// 无身份
	_, err = svc.Delete(ctx, &aiV1.DeleteAiConversationRequest{
		QueryBy: &aiV1.DeleteAiConversationRequest_Id{Id: others},
	})
	require.Error(t, err, "无操作者身份的删除必须被拒绝")
}
