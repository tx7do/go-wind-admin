// InternalMessageRecipientService 的 SQLite 内存库测试。
//
// 收件箱是头部徽标的轮询高频路径，测试盯两件事：
//   - ListUserInbox 的批量回填：收件行只有 message_id，标题/正文来自父消息——
//     回填必须以 SystemContext 读父消息（公告的父消息落在租户 0，读者 viewer
//     会把租户 0 的行滤掉，"有信无题"分叉），且一次 IN 查询而不是逐条 N+1；
//   - 归属谓词：普通用户只读得到自己的收件行（viewer 按 recipient_user_id 钉死，
//     客户端伪造过滤条件翻别人的收件记录必须无效）。
//
// 已读/删除/状态批量改是 repo 直通（repo 侧语义由 data 包测试覆盖），此处验证服务连通。
package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/enttest"

	crudViewer "github.com/tx7do/go-crud/viewer"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	internalMessageV1 "go-wind-admin/api/gen/go/internal_message/service/v1"
	"go-wind-admin/pkg/middleware/auth"
)

func newRecipientServiceForTest(t *testing.T) (*InternalMessageRecipientService, *data.InternalMessageRecipientRepo, context.Context, context.Context) {
	t.Helper()
	entClient := enttest.NewEntClientForTest(t)
	msgRepo := data.NewInternalMessageRepoForTest(entClient)
	recipientRepo := data.NewInternalMessageRecipientRepoForTest(entClient)

	svc := &InternalMessageRecipientService{
		log:                          bLogger.NewHelper(bLogger.NopLogger()),
		internalMessageRepo:          msgRepo,
		internalMessageRecipientRepo: recipientRepo,
	}
	// 收件箱归属谓词来自 viewer（非平台/非系统上下文才钉 recipient_user_id），
	// 所以操作人上下文必须是用户 viewer（user 71/72，租户 5），再叠 auth claims。
	opCtx71 := auth.NewContext(newUserViewerCtx(71), &authenticationV1.UserTokenPayload{UserId: 71})
	opCtx72 := auth.NewContext(newUserViewerCtx(72), &authenticationV1.UserTokenPayload{UserId: 72})
	return svc, recipientRepo, opCtx71, opCtx72
}

// inboxTestTenantID 是收件箱测试的公共租户（消息与收件行都落它名下）。
const inboxTestTenantID = uint32(5)

// newUserViewerCtx 构造租户 5 的普通用户 viewer 上下文（会触发归属谓词）。
func newUserViewerCtx(uid uint32) context.Context {
	return crudViewer.WithContext(context.Background(),
		crudViewer.NewUserContext(uint64(uid), uint64(inboxTestTenantID), 0, "", nil))
}

// seedInboxRow 落一条父消息 + 指定用户的收件行，返回收件行 ID。
func seedInboxRow(t *testing.T, svc *InternalMessageRecipientService, recipientUserID uint32, title, content string) uint32 {
	t.Helper()
	sysCtx := enttest.NewSystemContext(context.Background())

	msg, err := svc.internalMessageRepo.Create(sysCtx, &internalMessageV1.CreateInternalMessageRequest{
		Data: &internalMessageV1.InternalMessage{
			TenantId: trans.Ptr(inboxTestTenantID),
			Title:    trans.Ptr(title),
			Content:  trans.Ptr(content),
		},
	})
	require.NoError(t, err, "父消息应落库")

	row, err := svc.internalMessageRecipientRepo.Create(sysCtx, &internalMessageV1.InternalMessageRecipient{
		TenantId:        trans.Ptr(inboxTestTenantID),
		MessageId:       trans.Ptr(msg.GetId()),
		RecipientUserId: trans.Ptr(recipientUserID),
		Status:          internalMessageV1.InternalMessageRecipient_RECEIVED.Enum(),
	})
	require.NoError(t, err, "收件行应落库")
	return row.GetId()
}

// TestInternalMessageRecipientService_ListUserInbox_EnrichesAndScopes 验证
// 回填与归属两个核心语义。
func TestInternalMessageRecipientService_ListUserInbox_EnrichesAndScopes(t *testing.T) {
	svc, _, opCtx71, opCtx72 := newRecipientServiceForTest(t)

	seedInboxRow(t, svc, 71, "维护公告", "本周六 02:00-04:00 系统维护")
	seedInboxRow(t, svc, 72, "别人的信", "user 71 不该看到这行")

	// user 71 的收件箱：1 行，且标题/正文已回填
	resp, err := svc.ListUserInbox(opCtx71, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(1), resp.GetTotal(), "user 71 只应看到自己的收件行")
	require.Len(t, resp.GetItems(), 1)
	require.Equal(t, "维护公告", resp.GetItems()[0].GetTitle(), "标题应从父消息回填")
	require.Equal(t, "本周六 02:00-04:00 系统维护", resp.GetItems()[0].GetContent(), "正文应从父消息回填")

	// user 72：归属谓词生效，翻不到 user 71 的收件行
	resp72, err := svc.ListUserInbox(opCtx72, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(1), resp72.GetTotal())
	require.Equal(t, "别人的信", resp72.GetItems()[0].GetTitle())

	// nil 分页请求：repo 守卫透传
	_, err = svc.ListUserInbox(opCtx71, nil)
	require.Error(t, err, "nil 分页请求应报 BadRequest")
}

// TestInternalMessageRecipientService_ReadAndDeletePassthrough 验证
// 已读/状态/删除三个直通方法的服务层连通。
func TestInternalMessageRecipientService_ReadAndDeletePassthrough(t *testing.T) {
	svc, recipientRepo, opCtx71, _ := newRecipientServiceForTest(t)
	id := seedInboxRow(t, svc, 71, "待读通知", "点了已读才安心")

	_, err := svc.MarkNotificationAsRead(opCtx71, &internalMessageV1.MarkNotificationAsReadRequest{
		RecipientIds: []uint32{id},
	})
	require.NoError(t, err)

	sysCtx := enttest.NewSystemContext(context.Background())
	row, err := recipientRepo.Get(sysCtx, &internalMessageV1.GetInternalMessageRecipientRequest{
		QueryBy: &internalMessageV1.GetInternalMessageRecipientRequest_Id{Id: id},
	})
	require.NoError(t, err)
	require.Equal(t, internalMessageV1.InternalMessageRecipient_READ, row.GetStatus(),
		"标记已读后状态应为 READ")

	_, err = svc.MarkNotificationsStatus(opCtx71, &internalMessageV1.MarkNotificationsStatusRequest{
		RecipientIds: []uint32{id},
		NewStatus:    internalMessageV1.InternalMessageRecipient_READ,
	})
	require.NoError(t, err, "批量状态修改应连通")

	_, err = svc.DeleteNotificationFromInbox(opCtx71, &internalMessageV1.DeleteNotificationFromInboxRequest{
		RecipientIds: []uint32{id},
	})
	require.NoError(t, err)

	resp, err := svc.ListUserInbox(opCtx71, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(0), resp.GetTotal(), "删除后收件箱应为空")
}
