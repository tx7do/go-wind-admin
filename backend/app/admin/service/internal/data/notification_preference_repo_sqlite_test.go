package data

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"

	notificationV1 "go-wind-admin/api/gen/go/notification/service/v1"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// TestNotificationPreferenceRepoSqlite_UpsertRoundtrip 回归 P3 偏好仓储：
// 首次保存走创建、再次保存走更新（同一行不翻倍），GetByUserID 未配置时回 (nil, nil)。
func TestNotificationPreferenceRepoSqlite_UpsertRoundtrip(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewNotificationPreferenceRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	// 未配置：nil 而非错误，默认值由服务层补
	pref, err := repo.GetByUserID(ctx, 1001)
	require.NoError(t, err)
	require.Nil(t, pref, "未配置偏好的用户应返回 nil 而非错误")

	// 首次保存：创建
	saved, err := repo.Upsert(ctx, 1001, &notificationV1.UpdateNotificationPreferenceRequest{
		QuietEnabled:     trans.Ptr(true),
		QuietStartMinute: trans.Ptr(int32(1320)),
		QuietEndMinute:   trans.Ptr(int32(480)),
		MutedCategoryIds: []uint64{7, 9},
	}, 1001)
	require.NoError(t, err, "首次保存应创建成功")
	require.Equal(t, uint64(1001), saved.GetUserId())
	require.True(t, saved.GetQuietEnabled())
	require.Equal(t, []uint64{7, 9}, saved.GetMutedCategoryIds())

	// 再次保存：更新同一行
	saved, err = repo.Upsert(ctx, 1001, &notificationV1.UpdateNotificationPreferenceRequest{
		QuietEnabled:     trans.Ptr(false),
		QuietStartMinute: trans.Ptr(int32(1260)), // 21:00
		QuietEndMinute:   trans.Ptr(int32(360)),  // 06:00
		MutedCategoryIds: []uint64{9},
	}, 1001)
	require.NoError(t, err, "再次保存应更新成功")

	rows, err := entClient.Client().NotificationPreference.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1, "两次保存应只占一行（按 user_id upsert）")
	require.False(t, rows[0].QuietEnabled, "静音开关应被更新为 false")
	require.Equal(t, []uint64{9}, rows[0].MutedCategoryIds, "退订列表应被整体替换")

	got, err := repo.GetByUserID(ctx, 1001)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, int32(1260), got.GetQuietStartMinute())
	require.Equal(t, int32(360), got.GetQuietEndMinute())
}

// TestNotificationPreferenceRepoSqlite_ListByUserIDs 批量读：缺配置的用户不在返回 map 里。
func TestNotificationPreferenceRepoSqlite_ListByUserIDs(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewNotificationPreferenceRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	_, err := repo.Upsert(ctx, 2001, &notificationV1.UpdateNotificationPreferenceRequest{
		QuietEnabled:     trans.Ptr(true),
		QuietStartMinute: trans.Ptr(int32(1320)),
		QuietEndMinute:   trans.Ptr(int32(480)),
	}, 2001)
	require.NoError(t, err)

	prefs, err := repo.ListByUserIDs(ctx, []uint64{2001, 2002, 2003})
	require.NoError(t, err)
	require.Len(t, prefs, 1, "只有配置过偏好的用户出现在结果里")
	require.Contains(t, prefs, uint64(2001))
	require.True(t, prefs[2001].GetQuietEnabled())
}
