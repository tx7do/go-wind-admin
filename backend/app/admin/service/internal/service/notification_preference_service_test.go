package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/enttest"

	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	notificationV1 "go-wind-admin/api/gen/go/notification/service/v1"

	"github.com/tx7do/go-crud/viewer"
)

// ---- inQuietWindow 单元 ----

func quietPref(enabled bool, start, end int32) *notificationV1.NotificationPreference {
	return &notificationV1.NotificationPreference{
		QuietEnabled:     trans.Ptr(enabled),
		QuietStartMinute: trans.Ptr(start),
		QuietEndMinute:   trans.Ptr(end),
	}
}

func TestInQuietWindow(t *testing.T) {
	at := func(h, m int) time.Time { return time.Date(2026, 10, 3, h, m, 0, 0, time.Local) }

	require.False(t, inQuietWindow(nil, at(23, 0)), "nil 偏好不静音")
	require.False(t, inQuietWindow(quietPref(false, 1320, 480), at(23, 0)), "未启用不静音")

	// 常规窗口 02:00→04:00，左闭右开 [start, end)
	require.True(t, inQuietWindow(quietPref(true, 120, 240), at(3, 0)), "窗口内")
	require.False(t, inQuietWindow(quietPref(true, 120, 240), at(5, 0)), "窗口外")
	require.True(t, inQuietWindow(quietPref(true, 120, 240), at(2, 0)), "start 时刻进入窗口")
	require.False(t, inQuietWindow(quietPref(true, 120, 240), at(4, 0)), "end 为开区间：end 时刻已出窗口")

	// 跨零点窗口 22:00→08:00
	pref := quietPref(true, 1320, 480)
	require.True(t, inQuietWindow(pref, at(23, 30)), "跨零点：深夜在窗口")
	require.True(t, inQuietWindow(pref, at(6, 0)), "跨零点：清晨在窗口")
	require.False(t, inQuietWindow(pref, at(12, 0)), "跨零点：白天不在窗口")
	require.True(t, inQuietWindow(pref, at(22, 0)), "跨零点：start 时刻进入窗口")

	// 零宽窗口兜底不静音
	require.False(t, inQuietWindow(quietPref(true, 600, 600), at(10, 0)), "start==end 不静音")
}

// ---- categoryMuted 单元 ----

func TestCategoryMuted(t *testing.T) {
	pref := &notificationV1.NotificationPreference{MutedCategoryIds: []uint64{7, 9}}
	require.True(t, categoryMuted(pref, 7))
	require.False(t, categoryMuted(pref, 8))
	require.False(t, categoryMuted(pref, 0), "未归类消息（categoryId=0）不受退订约束")
	require.False(t, categoryMuted(nil, 7))
}

// ---- validateQuietWindow 单元 ----

func TestValidateQuietWindow(t *testing.T) {
	require.NoError(t, validateQuietWindow(true, 1320, 480), "跨零点合法")
	require.NoError(t, validateQuietWindow(true, 120, 240), "常规窗口合法")
	require.NoError(t, validateQuietWindow(false, 600, 600), "未启用时零宽窗口放行（保存草稿）")
	require.Error(t, validateQuietWindow(true, 600, 600), "启用时零宽窗口拒绝")
	require.Error(t, validateQuietWindow(true, -1, 480), "负分钟数拒绝")
	require.Error(t, validateQuietWindow(true, 0, 1440), "超出 1439 拒绝")
}

// ---- 广播退订 + 静音抑制（SQLite 集成） ----

// prefBroadcastUserRepoStub 广播受众桩：List 按页返回固定用户，Get 供租户打标。
type prefBroadcastUserRepoStub struct {
	data.UserRepo
	tenantByUserID map[uint64]uint64
}

func (s *prefBroadcastUserRepoStub) Get(_ context.Context, req *identityV1.GetUserRequest) (*identityV1.User, error) {
	uid := req.GetId()
	tid, ok := s.tenantByUserID[uid]
	if !ok {
		return nil, identityV1.ErrorNotFound("user [%d] not found", uid)
	}
	return &identityV1.User{Id: trans.Ptr(uid), TenantId: trans.Ptr(tid)}, nil
}

func (s *prefBroadcastUserRepoStub) List(_ context.Context, _ *paginationV1.PagingRequest) (*identityV1.ListUserResponse, error) {
	items := make([]*identityV1.User, 0, len(s.tenantByUserID))
	for uid, tid := range s.tenantByUserID {
		items = append(items, &identityV1.User{Id: trans.Ptr(uid), TenantId: trans.Ptr(tid)})
	}
	return &identityV1.ListUserResponse{Total: uint64(len(items)), Items: items}, nil
}

// TestBroadcastRespectsMutedCategoriesAndQuietHours 钉住 P3 偏好在广播上的两条执行语义：
//   - 退订了消息分类的收件人整行不落库也不推送（点对点不受此约束，见 seam 测试）；
//   - 静音时段的收件人收件行照常落库，但 SSE 实时推送被抑制。
func TestBroadcastRespectsMutedCategoriesAndQuietHours(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	prefRepo := data.NewNotificationPreferenceRepoForTest(entClient)
	pub := &payloadRecordingPublisher{}
	ctx := viewer.WithSystemContext(context.Background())

	im := &InternalMessageService{
		log:                          bLogger.NewHelper(bLogger.NopLogger()),
		internalMessageRepo:          data.NewInternalMessageRepoForTest(entClient),
		internalMessageCategoryRepo:  data.NewInternalMessageCategoryRepoForTest(entClient),
		internalMessageRecipientRepo: data.NewInternalMessageRecipientRepoForTest(entClient),
		notificationPreferenceRepo:   prefRepo,
		userRepo: &prefBroadcastUserRepoStub{
			tenantByUserID: map[uint64]uint64{
				11: 5, // 未配置偏好：正常落库 + 推送
				12: 5, // 退订分类 7：整行不落库
				13: 5, // 静音时段：落库但不推送
			},
		},
		internalMessagePublisher: pub,
		notifier:                 unwiredNotifier{},
	}

	// 12 号退订分类 7；13 号启用静音（窗口 00:00-23:59 全天，任何时刻命中）
	_, err := prefRepo.Upsert(ctx, 12, &notificationV1.UpdateNotificationPreferenceRequest{
		MutedCategoryIds: []uint64{7},
	}, 12)
	require.NoError(t, err)
	_, err = prefRepo.Upsert(ctx, 13, &notificationV1.UpdateNotificationPreferenceRequest{
		QuietEnabled:     trans.Ptr(true),
		QuietStartMinute: trans.Ptr(int32(0)),
		QuietEndMinute:   trans.Ptr(int32(1439)),
	}, 13)
	require.NoError(t, err)

	im.executeBroadcast(ctx, 77, 1, 7, "退订测试广播", "content")

	// 收件行：只有 11 和 13；12 整行缺席
	recipients, err := entClient.Client().InternalMessageRecipient.Query().All(ctx)
	require.NoError(t, err)
	got := make(map[uint64]bool, len(recipients))
	for _, r := range recipients {
		got[*r.RecipientUserID] = true
	}
	require.True(t, got[11], "无偏好收件人正常落库")
	require.False(t, got[12], "退订该分类的收件人整行不落库")
	require.True(t, got[13], "静音时段收件人收件行照常落库（消息不丢）")

	// SSE：只有 11 被实时推送；13（静音）被抑制
	pushed := make(map[string]bool)
	for _, ev := range pub.events {
		pushed[string(ev.Data)] = true
	}
	require.Equal(t, 1, len(pub.events), "只应有一次实时推送（静音者抑制）")
	require.Contains(t, string(pub.events[0].Data), `"recipientUserId":"11"`,
		"推送载荷应指向未配置偏好的 11 号")
}

// TestBroadcastUnmutedCategoryDeliversAll 反向对照：未退订该分类时全员落库。
func TestBroadcastUnmutedCategoryDeliversAll(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	prefRepo := data.NewNotificationPreferenceRepoForTest(entClient)
	pub := &payloadRecordingPublisher{}
	ctx := viewer.WithSystemContext(context.Background())

	im := &InternalMessageService{
		log:                          bLogger.NewHelper(bLogger.NopLogger()),
		internalMessageRepo:          data.NewInternalMessageRepoForTest(entClient),
		internalMessageCategoryRepo:  data.NewInternalMessageCategoryRepoForTest(entClient),
		internalMessageRecipientRepo: data.NewInternalMessageRecipientRepoForTest(entClient),
		notificationPreferenceRepo:   prefRepo,
		userRepo: &prefBroadcastUserRepoStub{
			tenantByUserID: map[uint64]uint64{21: 5, 22: 5},
		},
		internalMessagePublisher: pub,
		notifier:                 unwiredNotifier{},
	}

	// 22 号退订的是分类 9，广播是分类 7：照常投递
	_, err := prefRepo.Upsert(ctx, 22, &notificationV1.UpdateNotificationPreferenceRequest{
		MutedCategoryIds: []uint64{9},
	}, 22)
	require.NoError(t, err)

	im.executeBroadcast(ctx, 88, 1, 7, "对照广播", "content")

	recipients, err := entClient.Client().InternalMessageRecipient.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, recipients, 2, "退订分类与消息分类不同时投递不受影响")
	require.Equal(t, 2, len(pub.events))
}
