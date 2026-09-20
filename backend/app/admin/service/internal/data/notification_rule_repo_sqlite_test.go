// NotificationRuleRepo 的 SQLite 内存库测试（白盒，包内测试）。
//
// 这张表是 SendDirect 每次派发的运行时真相，所以测试盯的是三件"读错一次就发错一条"的事：
//   - 枚举往返：event_type / channel 两个实体侧可空指针枚举列必须经 converter 双向命中
//     （schema 注释承诺"Optional().Nillable() 才让 copier 的指针对生效"，这里就是那句承诺的证据）；
//   - GetByEventType 的"没有这行"与"查库失败"分开（(nil,nil) vs error）：压成同一个错误会让
//     DB 抖动时报出"没有路由规则"，把人往错的方向引；
//   - event_type 唯一索引 + Create 的前置查重：同一事件两行规则时 GetByEventType 的 Only 会
//     直接抛错，整条投递链断掉，所以第二行必须落不进去。
package data

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	notificationV1 "go-wind-admin/api/gen/go/notification/service/v1"
	"go-wind-admin/app/admin/service/internal/data/ent/notificationrule"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newRule 构造一行合法规则载荷（异步、启用），字段可按需覆盖。
func newRule(eventType notificationV1.EventType, channel notificationV1.Channel, isAsync bool) *notificationV1.NotificationRule {
	return &notificationV1.NotificationRule{
		EventType: eventType.Enum(),
		Channel:   channel.Enum(),
		IsAsync:   trans.Ptr(isAsync),
		IsEnabled: trans.Ptr(true),
	}
}

// TestNotificationRuleRepoSqlite_Create 验证字段级落库与枚举双向映射：
// 实体枚举列按 proto 枚举名落库、operatorID 落 created_by、created_at 显式写入；
// Get 回读的 DTO 枚举必须落回正确的 proto 枚举值（读视图一旦失配，服务层拿到的是
// 缺省 UNSPECIFIED，路由与页面显示会一起歪）。
func TestNotificationRuleRepoSqlite_Create(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewNotificationRuleRepoForTest(entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	id, err := repo.Create(ctx, &notificationV1.NotificationRule{
		EventType: notificationV1.EventType_PASSWORD_RESET_CODE.Enum(),
		Channel:   notificationV1.Channel_EMAIL.Enum(),
		IsAsync:   trans.Ptr(true),
		IsEnabled: trans.Ptr(true),
		Remark:    trans.Ptr("找回密码：入队即回，不等 SMTP 往返"),
	}, 1001)
	require.NoError(t, err, "完整载荷的 Create 应成功")
	require.NotZero(t, id)

	row, err := entClient.Client().NotificationRule.Get(ctx, id)
	require.NoError(t, err)
	require.NotNil(t, row.EventType)
	require.Equal(t, notificationrule.EventTypePasswordResetCode, *row.EventType,
		"proto 枚举名应经 converter 落为实体枚举字面量")
	require.NotNil(t, row.Channel)
	require.Equal(t, notificationrule.ChannelEmail, *row.Channel)
	require.NotNil(t, row.IsAsync)
	require.True(t, *row.IsAsync, "is_async 应按载荷落库")
	require.NotNil(t, row.IsEnabled)
	require.True(t, *row.IsEnabled)
	require.NotNil(t, row.Remark)
	require.Equal(t, "找回密码：入队即回，不等 SMTP 往返", *row.Remark)
	require.NotNil(t, row.CreatedBy)
	require.Equal(t, uint32(1001), *row.CreatedBy, "operatorID 应落 created_by")
	require.NotNil(t, row.CreatedAt, "created_at 必须显式写入：留给 NULL 列表页会出一排空行")

	// 读视图：枚举回映射（这一条就是 schema 那句"Optional().Nillable()"承诺的兑现）
	dto, err := repo.Get(ctx, id)
	require.NoError(t, err)
	require.Equal(t, id, dto.GetId())
	require.Equal(t, notificationV1.EventType_PASSWORD_RESET_CODE, dto.GetEventType(),
		"实体枚举应经 converter 回映射为 proto 枚举，不能落 UNSPECIFIED 零值")
	require.Equal(t, notificationV1.Channel_EMAIL, dto.GetChannel())
	require.True(t, dto.GetIsAsync())
	require.True(t, dto.GetIsEnabled())
	require.Equal(t, "找回密码：入队即回，不等 SMTP 往返", dto.GetRemark())

	// 同步规则（WEBHOOK）：operatorID=0 表示系统播种，created_by 不落值
	id2, err := repo.Create(ctx, newRule(notificationV1.EventType_INTERNAL_MESSAGE, notificationV1.Channel_WEBHOOK, false), 0)
	require.NoError(t, err)
	require.NotZero(t, id2)
	require.NotEqual(t, id, id2, "两次创建的 ID 应不同")

	row2, err := entClient.Client().NotificationRule.Get(ctx, id2)
	require.NoError(t, err)
	require.Equal(t, notificationrule.ChannelWebhook, *row2.Channel)
	require.NotNil(t, row2.IsAsync)
	require.False(t, *row2.IsAsync, "is_async=false 也要如实落库，不能因默认值被吞")
	require.Nil(t, row2.CreatedBy, "播种（operatorID=0）不该写出一个不存在的操作人")

	// 入参守卫
	_, err = repo.Create(ctx, nil, 1)
	require.Error(t, err, "nil 载荷应返回 BadRequest")
	_, err = repo.Create(ctx, &notificationV1.NotificationRule{Channel: notificationV1.Channel_EMAIL.Enum()}, 1)
	require.Error(t, err, "缺 event_type 应返回 BadRequest")
	require.Contains(t, err.Error(), "event_type is required")
	_, err = repo.Create(ctx, &notificationV1.NotificationRule{
		EventType: notificationV1.EventType_PASSWORD_RESET_CODE.Enum(),
	}, 1)
	require.Error(t, err, "缺 channel 应返回 BadRequest")
	require.Contains(t, err.Error(), "channel is required")
}

// TestNotificationRuleRepoSqlite_CreateRejectsDuplicateEventType 验证同一事件的第二行
// 被前置查重挡下：一张表容不下"同一事件发多个渠道"，两行并存时 GetByEventType 的 Only
// 直接抛错，整条投递链断掉。
func TestNotificationRuleRepoSqlite_CreateRejectsDuplicateEventType(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewNotificationRuleRepoForTest(entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	_, err := repo.Create(ctx, newRule(notificationV1.EventType_CONTACT_BIND_CODE, notificationV1.Channel_EMAIL, true), 1)
	require.NoError(t, err)

	// 换一个渠道也一样挡：查重只看 event_type
	_, err = repo.Create(ctx, newRule(notificationV1.EventType_CONTACT_BIND_CODE, notificationV1.Channel_WEBHOOK, false), 1)
	require.Error(t, err, "重复 event_type 应被拒绝")
	require.Contains(t, err.Error(), "already has a routing rule")

	cnt, err := repo.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, cnt, "失败的创建不该增加行数")

	// 另一个事件类型仍可入库（查重不能顺手把整张表锁住）
	_, err = repo.Create(ctx, newRule(notificationV1.EventType_CHANNEL_TEST_EMAIL, notificationV1.Channel_EMAIL, false), 1)
	require.NoError(t, err)
	cnt, err = repo.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, cnt)
}

// TestNotificationRuleRepoSqlite_GetByEventType 验证命中回 DTO、未命中回 (nil, nil)、
// UNSPECIFIED 也回 (nil, nil)（枚举名不在受支持取值内 → 空串查不到，而不是查全表）。
func TestNotificationRuleRepoSqlite_GetByEventType(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewNotificationRuleRepoForTest(entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	// 空表：未命中必须是 (nil, nil)，让调用方说人话而不是报"查询失败"
	miss, err := repo.GetByEventType(ctx, notificationV1.EventType_PASSWORD_RESET_CODE)
	require.NoError(t, err, "没有该行不是错误")
	require.Nil(t, miss)

	id, err := repo.Create(ctx, newRule(notificationV1.EventType_CONTACT_BIND_CODE, notificationV1.Channel_WEBHOOK, true), 1)
	require.NoError(t, err)

	got, err := repo.GetByEventType(ctx, notificationV1.EventType_CONTACT_BIND_CODE)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, id, got.GetId())
	require.Equal(t, notificationV1.EventType_CONTACT_BIND_CODE, got.GetEventType())
	require.Equal(t, notificationV1.Channel_WEBHOOK, got.GetChannel())
	require.True(t, got.GetIsAsync())

	// 另一个事件类型没有行：按类型查不能顺手命中别人那行
	miss, err = repo.GetByEventType(ctx, notificationV1.EventType_PASSWORD_RESET_CODE)
	require.NoError(t, err)
	require.Nil(t, miss, "只有 CONTACT_BIND_CODE 有行时其它事件仍应未命中")

	// UNSPECIFIED：查不到也不能命中真实行
	got, err = repo.GetByEventType(ctx, notificationV1.EventType_EVENT_TYPE_UNSPECIFIED)
	require.NoError(t, err)
	require.Nil(t, got, "UNSPECIFIED 不该命中任何规则行")
}

// TestNotificationRuleRepoSqlite_List 验证分页返回与 remark 的 contains 过滤；
// nil 请求返回 BadRequest。
func TestNotificationRuleRepoSqlite_List(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewNotificationRuleRepoForTest(entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	_, err := repo.Create(ctx, &notificationV1.NotificationRule{
		EventType: notificationV1.EventType_PASSWORD_RESET_CODE.Enum(),
		Channel:   notificationV1.Channel_EMAIL.Enum(),
		IsAsync:   trans.Ptr(true),
		IsEnabled: trans.Ptr(true),
		Remark:    trans.Ptr("marker-rule"),
	}, 1)
	require.NoError(t, err)
	id2, err := repo.Create(ctx, &notificationV1.NotificationRule{
		EventType: notificationV1.EventType_INTERNAL_MESSAGE.Enum(),
		Channel:   notificationV1.Channel_INTERNAL.Enum(),
		IsEnabled: trans.Ptr(false),
	}, 1)
	require.NoError(t, err)

	all, err := repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(2), all.GetTotal())
	require.Len(t, all.GetItems(), 2)
	// 列表同样要把枚举读回来（mapper 命中即够，不靠额外回填）
	byID := map[uint32]*notificationV1.NotificationRule{}
	for _, it := range all.GetItems() {
		byID[it.GetId()] = it
	}
	require.Equal(t, notificationV1.EventType_INTERNAL_MESSAGE, byID[id2].GetEventType())
	require.False(t, byID[id2].GetIsEnabled(), "is_enabled=false 应如实回读")

	filtered, err := repo.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_Query{
			Query: `{"remark__contains":"marker"}`,
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), filtered.GetTotal(), "contains 过滤应只统计命中行")
	require.Len(t, filtered.GetItems(), 1)
	require.Equal(t, notificationV1.EventType_PASSWORD_RESET_CODE, filtered.GetItems()[0].GetEventType())

	_, err = repo.List(ctx, nil)
	require.Error(t, err, "nil 分页请求应返回 BadRequest")
}

// TestNotificationRuleRepoSqlite_Update 验证掩码语义：掩码内字段更新（含枚举经
// converter 落库），掩码外字段保持原值，updated_by 盖操作人；不存在 ID 静默无效果。
func TestNotificationRuleRepoSqlite_Update(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewNotificationRuleRepoForTest(entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	id, err := repo.Create(ctx, &notificationV1.NotificationRule{
		EventType: notificationV1.EventType_PASSWORD_RESET_CODE.Enum(),
		Channel:   notificationV1.Channel_EMAIL.Enum(),
		IsAsync:   trans.Ptr(true),
		IsEnabled: trans.Ptr(true),
		Remark:    trans.Ptr("orig-remark"),
	}, 1001)
	require.NoError(t, err)

	// 把这条事件从"异步邮件"改成"同步 webhook"：这正是规则页存在的理由
	require.NoError(t, repo.Update(ctx, &notificationV1.UpdateNotificationRuleRequest{
		Id:         id,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"channel", "is_async"}},
		Data: &notificationV1.NotificationRule{
			Channel: notificationV1.Channel_WEBHOOK.Enum(),
			IsAsync: trans.Ptr(false),
			Remark:  trans.Ptr("masked-out"),
		},
	}, 2001))

	row, err := entClient.Client().NotificationRule.Get(ctx, id)
	require.NoError(t, err)
	require.Equal(t, notificationrule.ChannelWebhook, *row.Channel, "掩码内的 channel 应经 converter 更新")
	require.False(t, *row.IsAsync, "掩码内的 is_async 应更新")
	require.Equal(t, "orig-remark", *row.Remark, "掩码外的 remark 应保持原值")
	require.Equal(t, notificationrule.EventTypePasswordResetCode, *row.EventType, "掩码外的事件类型不得被改走")
	require.NotNil(t, row.UpdatedBy)
	require.Equal(t, uint32(2001), *row.UpdatedBy, "operatorID 应落 updated_by")
	require.NotNil(t, row.UpdatedAt, "更新应写入 updated_at")

	// 停用规则：resolveRoute 据此判定"这条路由不生效"
	require.NoError(t, repo.Update(ctx, &notificationV1.UpdateNotificationRuleRequest{
		Id:         id,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"is_enabled"}},
		Data:       &notificationV1.NotificationRule{IsEnabled: trans.Ptr(false)},
	}, 2002))
	row, err = entClient.Client().NotificationRule.Get(ctx, id)
	require.NoError(t, err)
	require.False(t, *row.IsEnabled)
	require.Equal(t, notificationrule.ChannelWebhook, *row.Channel, "掩码外的渠道应保持上次值")

	// 不存在 ID：无效果、不报错
	require.NoError(t, repo.Update(ctx, &notificationV1.UpdateNotificationRuleRequest{
		Id:         987654,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"is_async"}},
		Data:       &notificationV1.NotificationRule{IsAsync: trans.Ptr(true)},
	}, 1))
	cnt, err := repo.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, cnt, "对不存在 ID 的更新不应新增行")

	// 入参守卫
	require.Error(t, repo.Update(ctx, nil, 1), "nil 请求应返回 BadRequest")
	require.Error(t, repo.Update(ctx, &notificationV1.UpdateNotificationRuleRequest{Id: id}, 1), "nil Data 应返回 BadRequest")
	require.Error(t, repo.Update(ctx, &notificationV1.UpdateNotificationRuleRequest{
		Id:   0,
		Data: &notificationV1.NotificationRule{},
	}, 1), "id=0 应返回 BadRequest")
}

// TestNotificationRuleRepoSqlite_Delete 验证删除指定行、id=0 的 BadRequest、
// 不存在 ID 返回错误；删掉后 GetByEventType 回到 (nil, nil)（路由因此报错而非命中旧值）。
func TestNotificationRuleRepoSqlite_Delete(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewNotificationRuleRepoForTest(entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	idA, err := repo.Create(ctx, newRule(notificationV1.EventType_PASSWORD_RESET_CODE, notificationV1.Channel_EMAIL, true), 1)
	require.NoError(t, err)
	_, err = repo.Create(ctx, newRule(notificationV1.EventType_INTERNAL_MESSAGE, notificationV1.Channel_INTERNAL, false), 1)
	require.NoError(t, err)

	require.NoError(t, repo.Delete(ctx, idA), "按存在 ID 删除应成功")
	miss, err := repo.GetByEventType(ctx, notificationV1.EventType_PASSWORD_RESET_CODE)
	require.NoError(t, err, "删掉规则行不是查询错误")
	require.Nil(t, miss, "删除后该事件应回到「没有这行」，路由据此报错而非命中旧值")

	cnt, err := repo.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, cnt, "只该删掉一行")

	require.Error(t, repo.Delete(ctx, 0), "id=0 应返回 BadRequest")
	require.Error(t, repo.Delete(ctx, 9999999), "删除不存在的 ID 应返回错误")
	cnt, err = repo.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, cnt, "失败删除不应影响行数")
}
