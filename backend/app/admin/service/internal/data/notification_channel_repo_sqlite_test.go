package data

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	notificationChannelV1 "go-wind-admin/api/gen/go/notification_channel/service/v1"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/notificationchannel"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newNotificationChannelRepoSqlite 白盒构造 NotificationChannelRepo：与生产走同一条装配路径，
// 仅将 log 换为 NopLogger、entClient 换为 SQLite 内存库测试 client。
func newNotificationChannelRepoSqlite(t *testing.T, entClient *entCrud.EntClient[*ent.Client]) *NotificationChannelRepo {
	t.Helper()
	return newNotificationChannelRepo(bLogger.NewHelper(bLogger.NopLogger()), entClient)
}

// TestNotificationChannelRepoSqlite_Create 验证 Create 的字段级落库：
// 名称/SMTP 各字段、type 与 smtp_tls 枚举经 converter 映射、
// enabled 布尔派生 status（true→ON / 未传→OFF）、operatorID 落 created_by、
// 请求级 password 经 EncryptIfNeeded 透传落 smtp_password；未携带密码时该列为空。
func TestNotificationChannelRepoSqlite_Create(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newNotificationChannelRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	id, err := repo.Create(ctx, &notificationChannelV1.CreateNotificationChannelRequest{
		Data: &notificationChannelV1.NotificationChannel{
			Name:         trans.Ptr("sqlite-nc-alpha"),
			Type:         notificationChannelV1.NotificationChannel_EMAIL.Enum(),
			SmtpHost:     trans.Ptr("mail.alpha.example"),
			SmtpPort:     trans.Ptr(uint32(465)),
			SmtpUsername: trans.Ptr("alpha-user"),
			SmtpFrom:     trans.Ptr("noreply@alpha.example"),
			SmtpTls:      notificationChannelV1.NotificationChannel_SSL.Enum(),
			Enabled:      trans.Ptr(true),
			Remark:       trans.Ptr("渠道-alpha"),
		},
		Password: trans.Ptr("alpha-secret"),
	}, 1001)
	require.NoError(t, err, "携带完整载荷的 Create 应成功")
	require.NotZero(t, id)

	rows, err := entClient.Client().NotificationChannel.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1, "SQLite 中应有 1 条渠道记录")
	row := rows[0]
	require.Equal(t, "sqlite-nc-alpha", row.Name, "name 应按载荷落库")
	require.Equal(t, notificationchannel.TypeEmail, row.Type, "proto EMAIL 应经 converter 映射为 ent TypeEmail")
	require.NotNil(t, row.SMTPHost)
	require.Equal(t, "mail.alpha.example", *row.SMTPHost, "smtp_host 应按载荷落库")
	require.NotNil(t, row.SMTPPort)
	require.Equal(t, uint32(465), *row.SMTPPort, "smtp_port 应按载荷落库")
	require.NotNil(t, row.SMTPUsername)
	require.Equal(t, "alpha-user", *row.SMTPUsername, "smtp_username 应按载荷落库")
	require.NotNil(t, row.SMTPFrom)
	require.Equal(t, "noreply@alpha.example", *row.SMTPFrom, "smtp_from 应按载荷落库")
	require.NotNil(t, row.SMTPTLS)
	require.Equal(t, notificationchannel.SMTPTLSSsl, *row.SMTPTLS, "proto SSL 应经 converter 映射为 ent SMTPTLSSsl")
	require.NotNil(t, row.Status)
	require.Equal(t, notificationchannel.StatusOn, *row.Status, "enabled=true 应派生 status=ON")
	require.NotNil(t, row.Remark)
	require.Equal(t, "渠道-alpha", *row.Remark, "remark 应按载荷落库")
	require.NotNil(t, row.CreatedBy)
	require.Equal(t, uint32(1001), *row.CreatedBy, "operatorID 应落 created_by")
	// 未初始化全局加密器时 EncryptIfNeeded 为透传
	require.NotNil(t, row.SMTPPassword)
	require.Equal(t, "alpha-secret", *row.SMTPPassword, "请求级 password 应经 EncryptIfNeeded（透传）落 smtp_password")
	require.NotNil(t, row.CreatedAt)

	// 未启用、无密码：status=OFF、密码列空
	id2, err := repo.Create(ctx, &notificationChannelV1.CreateNotificationChannelRequest{
		Data: &notificationChannelV1.NotificationChannel{
			Name:    trans.Ptr("sqlite-nc-beta"),
			Type:    notificationChannelV1.NotificationChannel_WEBHOOK.Enum(),
			Enabled: trans.Ptr(false),
		},
	}, 1002)
	require.NoError(t, err)
	require.NotZero(t, id2)
	require.NotEqual(t, id, id2, "两次创建的 ID 应不同")

	rows, err = entClient.Client().NotificationChannel.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	var beta *ent.NotificationChannel
	for _, r := range rows {
		if r.ID == id2 {
			beta = r
		}
	}
	require.NotNil(t, beta, "第二条记录应可查回")
	require.Equal(t, notificationchannel.TypeWebhook, beta.Type, "proto WEBHOOK 应映射为 ent TypeWebhook")
	require.NotNil(t, beta.Status)
	require.Equal(t, notificationchannel.StatusOff, *beta.Status, "enabled=false 应派生 status=OFF")
	require.Nil(t, beta.SMTPPassword, "未携带 password 时 smtp_password 应为空")
	require.NotNil(t, beta.CreatedBy)
	require.Equal(t, uint32(1002), *beta.CreatedBy)

	// 参数校验分支
	_, err = repo.Create(ctx, nil, 1)
	require.Error(t, err, "nil 请求应返回 BadRequest")
	_, err = repo.Create(ctx, &notificationChannelV1.CreateNotificationChannelRequest{}, 1)
	require.Error(t, err, "nil Data 应返回 BadRequest")
	_, err = repo.Create(ctx, &notificationChannelV1.CreateNotificationChannelRequest{
		Data: &notificationChannelV1.NotificationChannel{Name: trans.Ptr("")},
	}, 1)
	require.Error(t, err, "空渠道名应返回 BadRequest")

	// 唯一索引：同名渠道重复创建应报错
	_, err = repo.Create(ctx, &notificationChannelV1.CreateNotificationChannelRequest{
		Data: &notificationChannelV1.NotificationChannel{
			Name: trans.Ptr("sqlite-nc-alpha"),
			Type: notificationChannelV1.NotificationChannel_EMAIL.Enum(),
		},
	}, 1)
	require.Error(t, err, "重复渠道名应触发唯一索引冲突并返回错误")
	cnt, err := entClient.Client().NotificationChannel.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, cnt, "失败创建不应增加行数")
}

// TestNotificationChannelRepoSqlite_Get 验证 Get 的字段回读：
// 名称/SMTP 字段/枚举经 converter 回映射；HasPassword 按库里密码列有无回填；
// 不存在的 ID 返回 NotFound。
func TestNotificationChannelRepoSqlite_Get(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newNotificationChannelRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	idWithPwd, err := repo.Create(ctx, &notificationChannelV1.CreateNotificationChannelRequest{
		Data: &notificationChannelV1.NotificationChannel{
			Name:         trans.Ptr("sqlite-nc-get-a"),
			Type:         notificationChannelV1.NotificationChannel_EMAIL.Enum(),
			SmtpHost:     trans.Ptr("mail.get.example"),
			SmtpPort:     trans.Ptr(uint32(587)),
			SmtpUsername: trans.Ptr("get-user"),
			SmtpFrom:     trans.Ptr("noreply@get.example"),
			SmtpTls:      notificationChannelV1.NotificationChannel_START_TLS.Enum(),
			Enabled:      trans.Ptr(true),
		},
		Password: trans.Ptr("get-secret"),
	}, 1)
	require.NoError(t, err)
	idWithoutPwd, err := repo.Create(ctx, &notificationChannelV1.CreateNotificationChannelRequest{
		Data: &notificationChannelV1.NotificationChannel{
			Name: trans.Ptr("sqlite-nc-get-b"),
			Type: notificationChannelV1.NotificationChannel_WEBHOOK.Enum(),
		},
	}, 1)
	require.NoError(t, err)

	// 命中（带密码）：字段与枚举回映射、HasPassword=true
	dto, err := repo.Get(ctx, idWithPwd)
	require.NoError(t, err, "按存在 ID 查询应命中")
	require.Equal(t, idWithPwd, dto.GetId(), "DTO 应带回正确 ID")
	require.Equal(t, "sqlite-nc-get-a", dto.GetName(), "DTO 应回读 name")
	require.Equal(t, notificationChannelV1.NotificationChannel_EMAIL, dto.GetType(), "ent TypeEmail 应回映射为 proto EMAIL")
	require.Equal(t, "mail.get.example", dto.GetSmtpHost(), "DTO 应回读 smtp_host")
	require.Equal(t, uint32(587), dto.GetSmtpPort(), "DTO 应回读 smtp_port")
	require.Equal(t, "get-user", dto.GetSmtpUsername(), "DTO 应回读 smtp_username")
	require.Equal(t, "noreply@get.example", dto.GetSmtpFrom(), "DTO 应回读 smtp_from")
	require.Equal(t, notificationChannelV1.NotificationChannel_START_TLS, dto.GetSmtpTls(), "ent SMTPTLSStartTls 应回映射为 proto START_TLS")
	require.True(t, dto.GetHasPassword(), "库里存在密码列时 HasPassword 应为 true")

	// 命中（无密码）：HasPassword=false；显式声明 WEBHOOK 的行如实回读
	// WEBHOOK（Type 回填修复前该字段被丢弃、呈缺省 EMAIL 零值）。
	dto, err = repo.Get(ctx, idWithoutPwd)
	require.NoError(t, err)
	require.Equal(t, idWithoutPwd, dto.GetId())
	require.Equal(t, notificationChannelV1.NotificationChannel_WEBHOOK, dto.GetType(), "显式 WEBHOOK 应回读 WEBHOOK")
	require.False(t, dto.GetHasPassword(), "库里无密码列时 HasPassword 应为 false")

	// 未命中
	_, err = repo.Get(ctx, 9999999)
	require.Error(t, err, "不存在的 ID 应返回 NotFound 错误")
}

// TestNotificationChannelRepoSqlite_List 验证 List 分页返回与
// queryHasPasswordByIDs 对 HasPassword 标识的填充（有密码/无密码两种）；
// 以及 name 的 contains 过滤与 nil 请求的 BadRequest。
func TestNotificationChannelRepoSqlite_List(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newNotificationChannelRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	idWithPwd, err := repo.Create(ctx, &notificationChannelV1.CreateNotificationChannelRequest{
		Data: &notificationChannelV1.NotificationChannel{
			Name: trans.Ptr("channel-alpha"),
			Type: notificationChannelV1.NotificationChannel_EMAIL.Enum(),
		},
		Password: trans.Ptr("list-secret"),
	}, 1)
	require.NoError(t, err)
	idWithoutPwd, err := repo.Create(ctx, &notificationChannelV1.CreateNotificationChannelRequest{
		Data: &notificationChannelV1.NotificationChannel{
			Name: trans.Ptr("channel-markerout-beta"),
			Type: notificationChannelV1.NotificationChannel_EMAIL.Enum(),
		},
	}, 1)
	require.NoError(t, err)

	// 无过滤：两行全部返回，HasPassword 按各自密码列有无回填
	all, err := repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(2), all.Total, "无过滤时应统计全部 2 条")
	require.Len(t, all.Items, 2, "无过滤时应返回 2 行")
	hasPwd := map[uint32]bool{}
	for _, it := range all.Items {
		hasPwd[it.GetId()] = it.GetHasPassword()
	}
	require.Equal(t, true, hasPwd[idWithPwd], "带密码行的 HasPassword 应为 true")
	require.Equal(t, false, hasPwd[idWithoutPwd], "无密码行的 HasPassword 应为 false")

	// name contains 过滤：markerout 仅命中名称含该串的 B 行（A 行不含）
	filtered, err := repo.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_Query{
			Query: `{"name__contains":"markerout"}`,
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), filtered.Total, "contains 过滤应只统计命中行")
	require.Len(t, filtered.Items, 1, "contains 过滤应只返回命中行")
	require.Equal(t, idWithoutPwd, filtered.Items[0].GetId(), "命中行应为名称含 markerout 的 B 行")
	require.False(t, filtered.Items[0].GetHasPassword())

	// nil 请求
	_, err = repo.List(ctx, nil)
	require.Error(t, err, "nil 分页请求应返回 BadRequest")
}

// TestNotificationChannelRepoSqlite_IsExist 验证 IsExist 的命中/未命中。
func TestNotificationChannelRepoSqlite_IsExist(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newNotificationChannelRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	id, err := repo.Create(ctx, &notificationChannelV1.CreateNotificationChannelRequest{
		Data: &notificationChannelV1.NotificationChannel{
			Name: trans.Ptr("exist-probe"),
			Type: notificationChannelV1.NotificationChannel_EMAIL.Enum(),
		},
	}, 1)
	require.NoError(t, err)

	exist, err := repo.IsExist(ctx, id)
	require.NoError(t, err)
	require.True(t, exist, "已写入的渠道应判定为存在")

	exist, err = repo.IsExist(ctx, 9999999)
	require.NoError(t, err)
	require.False(t, exist, "不存在的渠道应判定为不存在")
}

// TestNotificationChannelRepoSqlite_Update 验证 Update 的掩码语义：
// 掩码内字段更新（含枚举/status 派生），掩码外字段保持原值；
// 请求级 password 非空时更新 smtp_password、为空时保持；
// operatorID 落 updated_by；不存在 ID 的更新无效果且不报错。
func TestNotificationChannelRepoSqlite_Update(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newNotificationChannelRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	id, err := repo.Create(ctx, &notificationChannelV1.CreateNotificationChannelRequest{
		Data: &notificationChannelV1.NotificationChannel{
			Name:         trans.Ptr("orig-name"),
			Type:         notificationChannelV1.NotificationChannel_EMAIL.Enum(),
			SmtpHost:     trans.Ptr("h1.example"),
			SmtpPort:     trans.Ptr(uint32(25)),
			SmtpUsername: trans.Ptr("orig-user"),
			SmtpFrom:     trans.Ptr("orig@example.com"),
			SmtpTls:      notificationChannelV1.NotificationChannel_START_TLS.Enum(),
			Enabled:      trans.Ptr(false),
			Remark:       trans.Ptr("orig-remark"),
		},
		Password: trans.Ptr("orig-secret"),
	}, 1001)
	require.NoError(t, err)

	// 第一次更新：掩码 [smtp_host, smtp_tls, enabled]；
	// 掩码外的 name/remark 应保持原值；password 为空 → 密码列保持原值
	err = repo.Update(ctx, &notificationChannelV1.UpdateNotificationChannelRequest{
		Id:         id,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"smtp_host", "smtp_tls", "enabled"}},
		Data: &notificationChannelV1.NotificationChannel{
			Name:     trans.Ptr("masked-out-name"),
			SmtpHost: trans.Ptr("h2.example"),
			SmtpTls:  notificationChannelV1.NotificationChannel_SSL.Enum(),
			Enabled:  trans.Ptr(true),
			Remark:   trans.Ptr("masked-out-remark"),
		},
	}, 2001)
	require.NoError(t, err, "掩码内字段更新应成功")

	row, err := entClient.Client().NotificationChannel.Get(ctx, id)
	require.NoError(t, err)
	require.Equal(t, "orig-name", row.Name, "掩码外的 name 应保持原值")
	require.Equal(t, "orig-remark", *row.Remark, "掩码外的 remark 应保持原值")
	require.Equal(t, "orig@example.com", *row.SMTPFrom, "掩码外的 smtp_from 应保持原值")
	require.Equal(t, "orig-user", *row.SMTPUsername, "掩码外的 smtp_username 应保持原值")
	require.Equal(t, uint32(25), *row.SMTPPort, "掩码外的 smtp_port 应保持原值")
	require.Equal(t, "h2.example", *row.SMTPHost, "掩码内的 smtp_host 应更新")
	require.Equal(t, notificationchannel.SMTPTLSSsl, *row.SMTPTLS, "掩码内的 smtp_tls 应更新为 SSL")
	require.Equal(t, notificationchannel.StatusOn, *row.Status, "掩码内的 enabled=true 应派生 status=ON")
	require.Equal(t, "orig-secret", *row.SMTPPassword, "password 为空时 smtp_password 应保持原值")
	require.NotNil(t, row.UpdatedBy)
	require.Equal(t, uint32(2001), *row.UpdatedBy, "operatorID 应落 updated_by")
	require.NotNil(t, row.UpdatedAt, "更新应写入 updated_at")

	// 第二次更新：掩码 [smtp_from, smtp_username, remark, smtp_port]，password 非空
	err = repo.Update(ctx, &notificationChannelV1.UpdateNotificationChannelRequest{
		Id:         id,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"smtp_from", "smtp_username", "remark", "smtp_port"}},
		Data: &notificationChannelV1.NotificationChannel{
			SmtpFrom:     trans.Ptr("new@example.com"),
			SmtpUsername: trans.Ptr("new-user"),
			SmtpPort:     trans.Ptr(uint32(465)),
			Remark:       trans.Ptr("new-remark"),
			SmtpHost:     trans.Ptr("h999.example"),
			Enabled:      trans.Ptr(false),
		},
		Password: trans.Ptr("rotated-secret"),
	}, 2002)
	require.NoError(t, err)

	row, err = entClient.Client().NotificationChannel.Get(ctx, id)
	require.NoError(t, err)
	require.Equal(t, "new@example.com", *row.SMTPFrom, "掩码内的 smtp_from 应更新")
	require.Equal(t, "new-user", *row.SMTPUsername, "掩码内的 smtp_username 应更新")
	require.Equal(t, uint32(465), *row.SMTPPort, "掩码内的 smtp_port 应更新")
	require.Equal(t, "new-remark", *row.Remark, "掩码内的 remark 应更新")
	require.Equal(t, "h2.example", *row.SMTPHost, "掩码外的 smtp_host 应保持上次值")
	require.Equal(t, notificationchannel.StatusOn, *row.Status, "掩码外的 enabled 应保持上次值")
	require.Equal(t, "rotated-secret", *row.SMTPPassword, "请求级 password 非空时应更新 smtp_password（透传）")

	// 不存在 ID：无效果、不报错
	err = repo.Update(ctx, &notificationChannelV1.UpdateNotificationChannelRequest{
		Id:         987654,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"smtp_host"}},
		Data:       &notificationChannelV1.NotificationChannel{SmtpHost: trans.Ptr("ghost.example")},
	}, 1)
	require.NoError(t, err, "对不存在 ID 的更新应静默无效果")
	cnt, err := entClient.Client().NotificationChannel.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, cnt, "对不存在 ID 的更新不应新增行")
	row, err = entClient.Client().NotificationChannel.Get(ctx, id)
	require.NoError(t, err)
	require.Equal(t, "h2.example", *row.SMTPHost, "对不存在 ID 的更新不应影响已有行")

	// 参数校验
	require.Error(t, repo.Update(ctx, nil, 1), "nil 请求应返回 BadRequest")
	require.Error(t, repo.Update(ctx, &notificationChannelV1.UpdateNotificationChannelRequest{Id: id}, 1), "nil Data 应返回 BadRequest")
	require.Error(t, repo.Update(ctx, &notificationChannelV1.UpdateNotificationChannelRequest{
		Id:   0,
		Data: &notificationChannelV1.NotificationChannel{},
	}, 1), "id=0 应返回 BadRequest")
}

// TestNotificationChannelRepoSqlite_Delete 验证 Delete 删除指定行、
// id=0 的 BadRequest、不存在 ID 返回错误。
func TestNotificationChannelRepoSqlite_Delete(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newNotificationChannelRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	idA, err := repo.Create(ctx, &notificationChannelV1.CreateNotificationChannelRequest{
		Data: &notificationChannelV1.NotificationChannel{
			Name: trans.Ptr("del-a"),
			Type: notificationChannelV1.NotificationChannel_EMAIL.Enum(),
		},
	}, 1)
	require.NoError(t, err)
	idB, err := repo.Create(ctx, &notificationChannelV1.CreateNotificationChannelRequest{
		Data: &notificationChannelV1.NotificationChannel{
			Name: trans.Ptr("del-b"),
			Type: notificationChannelV1.NotificationChannel_EMAIL.Enum(),
		},
	}, 1)
	require.NoError(t, err)

	// 删除 A：仅剩 B
	require.NoError(t, repo.Delete(ctx, idA), "按存在 ID 删除应成功")
	rows, err := entClient.Client().NotificationChannel.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1, "删除后应仅剩 1 行")
	require.Equal(t, idB, rows[0].ID, "剩余行应为未删除的 B")

	// id=0：BadRequest
	require.Error(t, repo.Delete(ctx, 0), "id=0 应返回 BadRequest")

	// 不存在的 ID：NotFound 包装为错误
	require.Error(t, repo.Delete(ctx, 9999999), "删除不存在的 ID 应返回错误")
	cnt, err := entClient.Client().NotificationChannel.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, cnt, "失败删除不应影响行数")
}

// TestNotificationChannelRepoSqlite_GetFirstEnabledEmailChannel 验证
// 选取规则：仅 EMAIL 类型且 status=ON 的行参与，按 ID 升序取首条；
// 非 EMAIL 或未启用的行被跳过；首条被删除后轮到下一条；
// SMTP 配置（含密码透传解密）逐字段回读。
func TestNotificationChannelRepoSqlite_GetFirstEnabledEmailChannel(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newNotificationChannelRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	// 干扰行：WEBHOOK 启用（类型不符）、EMAIL 未启用（状态不符）
	_, err := repo.Create(ctx, &notificationChannelV1.CreateNotificationChannelRequest{
		Data: &notificationChannelV1.NotificationChannel{
			Name:    trans.Ptr("noise-webhook"),
			Type:    notificationChannelV1.NotificationChannel_WEBHOOK.Enum(),
			Enabled: trans.Ptr(true),
		},
	}, 1)
	require.NoError(t, err)
	_, err = repo.Create(ctx, &notificationChannelV1.CreateNotificationChannelRequest{
		Data: &notificationChannelV1.NotificationChannel{
			Name:    trans.Ptr("noise-email-off"),
			Type:    notificationChannelV1.NotificationChannel_EMAIL.Enum(),
			Enabled: trans.Ptr(false),
		},
	}, 1)
	require.NoError(t, err)

	// 候选行 A（ID 较小）与 B：均为启用 EMAIL
	idA, err := repo.Create(ctx, &notificationChannelV1.CreateNotificationChannelRequest{
		Data: &notificationChannelV1.NotificationChannel{
			Name:         trans.Ptr("cand-a"),
			Type:         notificationChannelV1.NotificationChannel_EMAIL.Enum(),
			SmtpHost:     trans.Ptr("smtp.a.example"),
			SmtpPort:     trans.Ptr(uint32(587)),
			SmtpUsername: trans.Ptr("user-a"),
			SmtpFrom:     trans.Ptr("from-a@example.com"),
			SmtpTls:      notificationChannelV1.NotificationChannel_START_TLS.Enum(),
			Enabled:      trans.Ptr(true),
		},
		Password: trans.Ptr("secret-a"),
	}, 1)
	require.NoError(t, err)
	idB, err := repo.Create(ctx, &notificationChannelV1.CreateNotificationChannelRequest{
		Data: &notificationChannelV1.NotificationChannel{
			Name:         trans.Ptr("cand-b"),
			Type:         notificationChannelV1.NotificationChannel_EMAIL.Enum(),
			SmtpHost:     trans.Ptr("smtp.b.example"),
			SmtpPort:     trans.Ptr(uint32(465)),
			SmtpUsername: trans.Ptr("user-b"),
			SmtpFrom:     trans.Ptr("from-b@example.com"),
			SmtpTls:      notificationChannelV1.NotificationChannel_SSL.Enum(),
			Enabled:      trans.Ptr(true),
		},
		Password: trans.Ptr("secret-b"),
	}, 1)
	require.NoError(t, err)

	// 首次选取：ID 升序命中 A，干扰行被跳过
	acct, err := repo.GetFirstEnabledEmailChannel(ctx)
	require.NoError(t, err, "存在启用的 EMAIL 渠道时应命中")
	require.Equal(t, idA, acct.ID, "命中行的主键要带回来：台账的 channel_id 只认这个")
	require.Equal(t, "smtp.a.example", acct.Host, "应命中 ID 较小的候选 A")
	require.Equal(t, uint32(587), acct.Port)
	require.Equal(t, "user-a", acct.Username)
	require.Equal(t, "from-a@example.com", acct.From)
	require.Equal(t, "secret-a", acct.Password, "密码应为透传解密结果")
	require.Equal(t, "START_TLS", acct.TlsMode, "TLS 模式应回读字符串值")
	require.True(t, acct.Enabled, "命中行应为启用状态")

	// 删除 A 后：轮到 B
	require.NoError(t, repo.Delete(ctx, idA), "删除候选 A 应成功")
	acct, err = repo.GetFirstEnabledEmailChannel(ctx)
	require.NoError(t, err, "删除 A 后仍存在启用的 EMAIL 渠道（B）应命中")
	require.Equal(t, "smtp.b.example", acct.Host, "删除 A 后应命中 B")
	require.Equal(t, idB, acct.ID, "换了一条就要换 ID，不能留上一次的")
	require.Equal(t, "SSL", acct.TlsMode)
	require.Equal(t, "secret-b", acct.Password)
}

// TestNotificationChannelRepoSqlite_GetFirstEnabledEmailChannel_None
// 验证无启用 EMAIL 渠道时返回 NotFound。
func TestNotificationChannelRepoSqlite_GetFirstEnabledEmailChannel_None(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newNotificationChannelRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	_, err := repo.Create(ctx, &notificationChannelV1.CreateNotificationChannelRequest{
		Data: &notificationChannelV1.NotificationChannel{
			Name:    trans.Ptr("only-disabled"),
			Type:    notificationChannelV1.NotificationChannel_EMAIL.Enum(),
			Enabled: trans.Ptr(false),
		},
	}, 1)
	require.NoError(t, err)

	_, err = repo.GetFirstEnabledEmailChannel(ctx)
	require.Error(t, err, "无启用的 EMAIL 渠道时应返回 NotFound")
}

// TestNotificationChannelRepoSqlite_GetDecryptedSmtpAccount 验证按 ID 取
// 解密 SMTP 配置：字段逐一回读、Enabled 按 status 映射、密码透传解密；
// 不存在 ID 返回 NotFound。
func TestNotificationChannelRepoSqlite_GetDecryptedSmtpAccount(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newNotificationChannelRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	idOn, err := repo.Create(ctx, &notificationChannelV1.CreateNotificationChannelRequest{
		Data: &notificationChannelV1.NotificationChannel{
			Name:         trans.Ptr("acct-on"),
			Type:         notificationChannelV1.NotificationChannel_EMAIL.Enum(),
			SmtpHost:     trans.Ptr("smtp.on.example"),
			SmtpPort:     trans.Ptr(uint32(25)),
			SmtpUsername: trans.Ptr("user-on"),
			SmtpFrom:     trans.Ptr("from-on@example.com"),
			SmtpTls:      notificationChannelV1.NotificationChannel_NONE.Enum(),
			Enabled:      trans.Ptr(true),
		},
		Password: trans.Ptr("secret-on"),
	}, 1)
	require.NoError(t, err)
	idOff, err := repo.Create(ctx, &notificationChannelV1.CreateNotificationChannelRequest{
		Data: &notificationChannelV1.NotificationChannel{
			Name:    trans.Ptr("acct-off"),
			Type:    notificationChannelV1.NotificationChannel_EMAIL.Enum(),
			Enabled: trans.Ptr(false),
		},
	}, 1)
	require.NoError(t, err)

	acct, err := repo.GetDecryptedSmtpAccount(ctx, idOn)
	require.NoError(t, err, "按存在 ID 查询应命中")
	require.Equal(t, idOn, acct.ID, "取回的账号要自带主键：显式指定渠道时错误文本靠它回指")
	require.Equal(t, "smtp.on.example", acct.Host, "Host 应回读")
	require.Equal(t, uint32(25), acct.Port, "Port 应回读")
	require.Equal(t, "user-on", acct.Username, "Username 应回读")
	require.Equal(t, "from-on@example.com", acct.From, "From 应回读")
	require.Equal(t, "secret-on", acct.Password, "Password 应为透传解密结果")
	require.Equal(t, "NONE", acct.TlsMode, "TlsMode 应回读字符串值")
	require.True(t, acct.Enabled, "status=ON 时 Enabled 应为 true")

	acct, err = repo.GetDecryptedSmtpAccount(ctx, idOff)
	require.NoError(t, err)
	require.False(t, acct.Enabled, "status=OFF 时 Enabled 应为 false")

	_, err = repo.GetDecryptedSmtpAccount(ctx, 9999999)
	require.Error(t, err, "不存在的 ID 应返回 NotFound")
}

// TestNotificationChannelRepoSqlite_WebhookColumns 验证 WEBHOOK 两列的读写：
// webhook_url 按载荷落库并经 copier 回读（实体 WebhookURL ↔ DTO WebhookUrl 只有大小写差，
// 与 SMTPHost ↔ SmtpHost 同一形态）；webhook_secret 只出现在写请求里，读视图一律以
// HasWebhookSecret 布尔代替——把整份读结果序列化成 JSON 断言密钥原文不在其中，
// 挡的是以后有人顺手给 DTO 加一列回显密钥。
func TestNotificationChannelRepoSqlite_WebhookColumns(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newNotificationChannelRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	idWithSecret, err := repo.Create(ctx, &notificationChannelV1.CreateNotificationChannelRequest{
		Data: &notificationChannelV1.NotificationChannel{
			Name:       trans.Ptr("wh-with-secret"),
			Type:       notificationChannelV1.NotificationChannel_WEBHOOK.Enum(),
			WebhookUrl: trans.Ptr("https://hooks.example.test/im"),
			Enabled:    trans.Ptr(true),
		},
		WebhookSecret: trans.Ptr("wh-secret-1"),
	}, 1)
	require.NoError(t, err, "携带 webhook 载荷的 Create 应成功")

	idNoSecret, err := repo.Create(ctx, &notificationChannelV1.CreateNotificationChannelRequest{
		Data: &notificationChannelV1.NotificationChannel{
			Name: trans.Ptr("wh-no-secret"),
			Type: notificationChannelV1.NotificationChannel_WEBHOOK.Enum(),
		},
	}, 1)
	require.NoError(t, err)

	// 落库侧：url/secret 各自落在自己的列，且没有串到 smtp_password（两列密钥共用
	// 同一条 EncryptIfNeeded 出口，写错列时 HasPassword 会跟着亮，读路径看不出差别）。
	row, err := entClient.Client().NotificationChannel.Get(ctx, idWithSecret)
	require.NoError(t, err)
	require.NotNil(t, row.WebhookURL)
	require.Equal(t, "https://hooks.example.test/im", *row.WebhookURL, "webhook_url 应按载荷落库")
	// 全局加密器未初始化时 EncryptIfNeeded 为透传
	require.NotNil(t, row.WebhookSecret)
	require.Equal(t, "wh-secret-1", *row.WebhookSecret, "webhook_secret 应经 EncryptIfNeeded（透传）落库")
	require.Nil(t, row.SMTPPassword, "webhook 密钥不该写到 smtp_password 列")
	require.Equal(t, notificationchannel.TypeWebhook, row.Type)

	row2, err := entClient.Client().NotificationChannel.Get(ctx, idNoSecret)
	require.NoError(t, err)
	require.Nil(t, row2.WebhookURL, "未传 webhook_url 时该列应为空")
	require.Nil(t, row2.WebhookSecret, "未传密钥时该列应为空")

	// 读视图：URL 回读（copier 大小写差命中），密钥只以布尔标识出现
	dto, err := repo.Get(ctx, idWithSecret)
	require.NoError(t, err)
	require.Equal(t, "https://hooks.example.test/im", dto.GetWebhookUrl(), "Get 应经 copier 回读 webhook_url")
	require.True(t, dto.GetHasWebhookSecret(), "库里存在密钥列时 HasWebhookSecret 应为 true")
	require.False(t, dto.GetHasPassword(), "只有 webhook 密钥的行不该亮 HasPassword")

	dto, err = repo.Get(ctx, idNoSecret)
	require.NoError(t, err)
	require.Empty(t, dto.GetWebhookUrl(), "未配置地址的行应回空串")
	require.False(t, dto.GetHasWebhookSecret())

	// 列表视图：两列标识逐行回填
	all, err := repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(2), all.GetTotal())
	flags := map[uint32]bool{}
	for _, it := range all.GetItems() {
		flags[it.GetId()] = it.GetHasWebhookSecret()
	}
	require.True(t, flags[idWithSecret], "带密钥行的 HasWebhookSecret 应为 true")
	require.False(t, flags[idNoSecret], "不带密钥行的 HasWebhookSecret 应为 false")

	// 密钥不得出现在任何读响应里（proto 里没有这个字段，但一旦有人加列这条断言就会响）
	asJSON, err := protojson.Marshal(all)
	require.NoError(t, err)
	require.NotContains(t, string(asJSON), "wh-secret-1", "读路径不得带出签名密钥原文")
	single, err := repo.Get(ctx, idWithSecret)
	require.NoError(t, err)
	asJSON, err = protojson.Marshal(single)
	require.NoError(t, err)
	require.NotContains(t, string(asJSON), "wh-secret-1")

	// 更新地址：掩码内 webhook_url 生效，请求级密钥同时轮换
	require.NoError(t, repo.Update(ctx, &notificationChannelV1.UpdateNotificationChannelRequest{
		Id:            idNoSecret,
		UpdateMask:    &fieldmaskpb.FieldMask{Paths: []string{"webhook_url"}},
		Data:          &notificationChannelV1.NotificationChannel{WebhookUrl: trans.Ptr("https://hooks.example.test/rotated")},
		WebhookSecret: trans.Ptr("wh-secret-2"),
	}, 2))
	row2, err = entClient.Client().NotificationChannel.Get(ctx, idNoSecret)
	require.NoError(t, err)
	require.Equal(t, "https://hooks.example.test/rotated", *row2.WebhookURL, "掩码内的 webhook_url 应更新")
	require.Equal(t, "wh-secret-2", *row2.WebhookSecret, "请求级密钥非空时应更新（透传）")

	// 只动备注：地址与密钥都保持原值（掩码外的 Data 字段会被 FilterByFieldMask 清零，
	// 所以 SetNillableWebhookURL 拿到的是 nil；密钥是请求级字段，留空即不改）
	require.NoError(t, repo.Update(ctx, &notificationChannelV1.UpdateNotificationChannelRequest{
		Id:         idNoSecret,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"remark"}},
		Data:       &notificationChannelV1.NotificationChannel{Remark: trans.Ptr("只改备注")},
	}, 3))
	row2, err = entClient.Client().NotificationChannel.Get(ctx, idNoSecret)
	require.NoError(t, err)
	require.Equal(t, "https://hooks.example.test/rotated", *row2.WebhookURL, "掩码外的 webhook_url 应保持原值")
	require.Equal(t, "wh-secret-2", *row2.WebhookSecret, "未携带密钥时不该清空已存密钥")
	require.Equal(t, "只改备注", *row2.Remark)
}

// TestNotificationChannelRepoSqlite_WebhookSignStyleColumns 验证本块新加两列的全链路读写：
// 写侧枚举落成列值字符串、读侧经 converter 回到 DTO 成员，以及投递侧唯一取数口
// （GetDecryptedWebhookAccount / GetFirstEnabledWebhookChannel）有没有把两列带进 WebhookAccount。
//
// 枚举这一趟必须实测而不能靠看一眼：repo 的 EnumTypeConverter 按**名字字符串**配对，
// proto 成员名与 ent 列值只要有一侧被加前缀，编译与 vet 都不报错，读写各自静默退化成
// nil / 列默认值——线上表现是"管理员选了钉钉，出去的一直是 CUSTOM 那份 JSON"。
func TestNotificationChannelRepoSqlite_WebhookSignStyleColumns(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newNotificationChannelRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	const dingTemplate = `{"msgtype":"text","text":{"content":"{{title}}\n{{content}}"}}`

	idDing, err := repo.Create(ctx, &notificationChannelV1.CreateNotificationChannelRequest{
		Data: &notificationChannelV1.NotificationChannel{
			Name:                   trans.Ptr("wh-style-ding"),
			Type:                   notificationChannelV1.NotificationChannel_WEBHOOK.Enum(),
			WebhookUrl:             trans.Ptr("https://oapi.example.test/robot?access_token=t"),
			Enabled:                trans.Ptr(true),
			WebhookSignStyle:       notificationChannelV1.SignStyle_DINGTALK.Enum(),
			WebhookPayloadTemplate: trans.Ptr(dingTemplate),
		},
	}, 1)
	require.NoError(t, err)

	row, err := entClient.Client().NotificationChannel.Get(ctx, idDing)
	require.NoError(t, err)
	require.NotNil(t, row.WebhookSignStyle)
	require.Equal(t, notificationchannel.WebhookSignStyleDingtalk, *row.WebhookSignStyle,
		"枚举要落成列值字符串（不是数字）：converter 就是按名字配对的")
	require.NotNil(t, row.WebhookPayloadTemplate)
	require.Equal(t, dingTemplate, *row.WebhookPayloadTemplate, "模板原样入库——校验与渲染是 sender 的事")

	dto, err := repo.Get(ctx, idDing)
	require.NoError(t, err)
	require.Equal(t, notificationChannelV1.SignStyle_DINGTALK, dto.GetWebhookSignStyle())
	require.Equal(t, dingTemplate, dto.GetWebhookPayloadTemplate())

	items, err := repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Len(t, items.GetItems(), 1)
	require.Equal(t, notificationChannelV1.SignStyle_DINGTALK, items.GetItems()[0].GetWebhookSignStyle(),
		"列表列与编辑表单回填读的是 List 这一条路径，与 Get 共用 converter 但值得各钉一次")

	acct, err := repo.GetDecryptedWebhookAccount(ctx, idDing)
	require.NoError(t, err)
	require.Equal(t, "DINGTALK", acct.SignStyle, "sender 只看 WebhookAccount，两列必须在这儿落地")
	require.Equal(t, dingTemplate, acct.PayloadTemplate)

	first, err := repo.GetFirstEnabledWebhookChannel(ctx)
	require.NoError(t, err)
	require.Equal(t, idDing, first.ID)
	require.Equal(t, "DINGTALK", first.SignStyle)
	require.Equal(t, dingTemplate, first.PayloadTemplate)

	// 未传风格：SetNillable 跳过该列 → 落 ent 的列默认值（新建行不会是 NULL，
	// NULL 只代表这两列加进来之前就存在的存量行）。
	idPlain, err := repo.Create(ctx, &notificationChannelV1.CreateNotificationChannelRequest{
		Data: &notificationChannelV1.NotificationChannel{
			Name:       trans.Ptr("wh-style-plain"),
			Type:       notificationChannelV1.NotificationChannel_WEBHOOK.Enum(),
			WebhookUrl: trans.Ptr("https://plain.example.test/hook"),
			Enabled:    trans.Ptr(true),
		},
	}, 1)
	require.NoError(t, err)

	rowPlain, err := entClient.Client().NotificationChannel.Get(ctx, idPlain)
	require.NoError(t, err)
	require.NotNil(t, rowPlain.WebhookSignStyle)
	require.Equal(t, notificationchannel.WebhookSignStyleCustom, *rowPlain.WebhookSignStyle)
	require.Nil(t, rowPlain.WebhookPayloadTemplate, "模板没有默认值：留空即 NULL，由 sender 按风格取内置形状")

	// 越界的枚举数值（protojson 允许数字形式的枚举字段）：converter 查不到名字就返回 nil，
	// 于是这一列按没传处理。记在这儿是因为它是**静默**的——将来若加校验，这条断言会先响。
	idJunk, err := repo.Create(ctx, &notificationChannelV1.CreateNotificationChannelRequest{
		Data: &notificationChannelV1.NotificationChannel{
			Name:             trans.Ptr("wh-style-junk"),
			Type:             notificationChannelV1.NotificationChannel_WEBHOOK.Enum(),
			Enabled:          trans.Ptr(true),
			WebhookSignStyle: notificationChannelV1.SignStyle(999).Enum(),
		},
	}, 1)
	require.NoError(t, err)
	rowJunk, err := entClient.Client().NotificationChannel.Get(ctx, idJunk)
	require.NoError(t, err)
	require.Equal(t, notificationchannel.WebhookSignStyleCustom, *rowJunk.WebhookSignStyle,
		"认不出的枚举值落列默认值 CUSTOM，而不是把 999 塞进 enum 列（ent 的列校验会 500）")

	// 存量行：把两列清成 NULL，验读取链路对"迁移前就存在的行"的表现。
	require.NoError(t, entClient.Client().NotificationChannel.UpdateOneID(idPlain).
		ClearWebhookSignStyle().ClearWebhookPayloadTemplate().Exec(ctx))

	dtoLegacy, err := repo.Get(ctx, idPlain)
	require.NoError(t, err)
	require.Equal(t, notificationChannelV1.SignStyle_CUSTOM, dtoLegacy.GetWebhookSignStyle(),
		"NULL 读回来是零值，而零值恰好就是 CUSTOM：读侧不必判空")

	acctLegacy, err := repo.GetDecryptedWebhookAccount(ctx, idPlain)
	require.NoError(t, err)
	require.Empty(t, acctLegacy.SignStyle, "账号结构里保留 NULL 的原样（空串），归一化留给 sender 报错/兜默认")
	require.Empty(t, acctLegacy.PayloadTemplate)

	// 掩码内只改风格：模板不在掩码里，必须保持原值。
	require.NoError(t, repo.Update(ctx, &notificationChannelV1.UpdateNotificationChannelRequest{
		Id:            idDing,
		UpdateMask:    &fieldmaskpb.FieldMask{Paths: []string{"webhook_sign_style"}},
		Data:          &notificationChannelV1.NotificationChannel{WebhookSignStyle: notificationChannelV1.SignStyle_FEISHU.Enum()},
		WebhookSecret: trans.Ptr("ding-rotated-secret"),
	}, 2))
	row, err = entClient.Client().NotificationChannel.Get(ctx, idDing)
	require.NoError(t, err)
	require.Equal(t, notificationchannel.WebhookSignStyleFeishu, *row.WebhookSignStyle)
	require.Equal(t, dingTemplate, *row.WebhookPayloadTemplate, "改风格不该顺手清空模板")

	dto, err = repo.Get(ctx, idDing)
	require.NoError(t, err)
	require.Equal(t, notificationChannelV1.SignStyle_FEISHU, dto.GetWebhookSignStyle())

	// 只动备注：两列都保持。
	require.NoError(t, repo.Update(ctx, &notificationChannelV1.UpdateNotificationChannelRequest{
		Id:         idDing,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"remark"}},
		Data:       &notificationChannelV1.NotificationChannel{Remark: trans.Ptr("只改备注")},
	}, 3))
	row, err = entClient.Client().NotificationChannel.Get(ctx, idDing)
	require.NoError(t, err)
	require.Equal(t, notificationchannel.WebhookSignStyleFeishu, *row.WebhookSignStyle, "掩码外的风格应保持原值")
	require.Equal(t, dingTemplate, *row.WebhookPayloadTemplate, "掩码外的模板应保持原值")

	// 模板清空：写进去的是空串而不是 NULL —— 空串的语义是"用该风格的内置默认形状"。
	require.NoError(t, repo.Update(ctx, &notificationChannelV1.UpdateNotificationChannelRequest{
		Id:         idDing,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"webhook_payload_template"}},
		Data:       &notificationChannelV1.NotificationChannel{WebhookPayloadTemplate: trans.Ptr("")},
	}, 4))
	row, err = entClient.Client().NotificationChannel.Get(ctx, idDing)
	require.NoError(t, err)
	require.NotNil(t, row.WebhookPayloadTemplate)
	require.Equal(t, "", *row.WebhookPayloadTemplate)
	acct, err = repo.GetDecryptedWebhookAccount(ctx, idDing)
	require.NoError(t, err)
	require.Empty(t, acct.PayloadTemplate)
	require.Equal(t, "FEISHU", acct.SignStyle, "改模板也不该动风格")
}

// TestNotificationChannelRepoSqlite_GetFirstEnabledWebhookChannel 验证自选 WEBHOOK 账号
// 的选取规则：仅 WEBHOOK 类型且 status=ON 的行参与、按 ID 升序取首条，密钥随账号
// 一并解出；无可用行时返回 NotFound（服务层据此记 SKIPPED）。
func TestNotificationChannelRepoSqlite_GetFirstEnabledWebhookChannel(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newNotificationChannelRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	// 干扰行：启用 EMAIL（类型不符）、停用 WEBHOOK（状态不符）
	_, err := repo.Create(ctx, &notificationChannelV1.CreateNotificationChannelRequest{
		Data: &notificationChannelV1.NotificationChannel{
			Name:    trans.Ptr("wh-noise-email-on"),
			Type:    notificationChannelV1.NotificationChannel_EMAIL.Enum(),
			Enabled: trans.Ptr(true),
		},
	}, 1)
	require.NoError(t, err)
	_, err = repo.Create(ctx, &notificationChannelV1.CreateNotificationChannelRequest{
		Data: &notificationChannelV1.NotificationChannel{
			Name:       trans.Ptr("wh-noise-off"),
			Type:       notificationChannelV1.NotificationChannel_WEBHOOK.Enum(),
			WebhookUrl: trans.Ptr("https://off.example.test"),
			Enabled:    trans.Ptr(false),
		},
	}, 1)
	require.NoError(t, err)

	idA, err := repo.Create(ctx, &notificationChannelV1.CreateNotificationChannelRequest{
		Data: &notificationChannelV1.NotificationChannel{
			Name:       trans.Ptr("wh-cand-a"),
			Type:       notificationChannelV1.NotificationChannel_WEBHOOK.Enum(),
			WebhookUrl: trans.Ptr("https://a.example.test/hook"),
			Enabled:    trans.Ptr(true),
		},
		WebhookSecret: trans.Ptr("secret-a"),
	}, 1)
	require.NoError(t, err)
	// 候选 B 刻意不配密钥：验证 Secret 走"列不存在→空串"而不是 nil 解引用
	idB, err := repo.Create(ctx, &notificationChannelV1.CreateNotificationChannelRequest{
		Data: &notificationChannelV1.NotificationChannel{
			Name:       trans.Ptr("wh-cand-b"),
			Type:       notificationChannelV1.NotificationChannel_WEBHOOK.Enum(),
			WebhookUrl: trans.Ptr("https://b.example.test/hook"),
			Enabled:    trans.Ptr(true),
		},
	}, 1)
	require.NoError(t, err)
	require.Greater(t, idB, idA)

	acct, err := repo.GetFirstEnabledWebhookChannel(ctx)
	require.NoError(t, err, "存在启用 WEBHOOK 渠道时应命中")
	require.Equal(t, idA, acct.ID, "按 ID 升序取首条，且主键要带回来供台账记 channel_id")
	require.Equal(t, "https://a.example.test/hook", acct.URL)
	require.Equal(t, "secret-a", acct.Secret, "密钥应为透传解密结果")

	require.NoError(t, repo.Delete(ctx, idA))
	acct, err = repo.GetFirstEnabledWebhookChannel(ctx)
	require.NoError(t, err, "删除 A 后应轮到 B")
	require.Equal(t, idB, acct.ID)
	require.Equal(t, "https://b.example.test/hook", acct.URL)
	require.Empty(t, acct.Secret, "未配置密钥的行应回空串（签名头不下发）")

	require.NoError(t, repo.Delete(ctx, idB))
	_, err = repo.GetFirstEnabledWebhookChannel(ctx)
	require.Error(t, err, "只剩干扰行时应返回 NotFound")
}

// TestNotificationChannelRepoSqlite_GetDecryptedWebhookAccount 验证显式指定渠道时取
// WEBHOOK 账号：按 ID 命中、启用状态不参与该函数（守卫在 sender 侧）、类型不符即拒绝
// （SMTP 行被当 webhook 目标使，等于把回调打到别人家地址上）。
func TestNotificationChannelRepoSqlite_GetDecryptedWebhookAccount(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newNotificationChannelRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	idWh, err := repo.Create(ctx, &notificationChannelV1.CreateNotificationChannelRequest{
		Data: &notificationChannelV1.NotificationChannel{
			Name:       trans.Ptr("wh-by-id"),
			Type:       notificationChannelV1.NotificationChannel_WEBHOOK.Enum(),
			WebhookUrl: trans.Ptr("https://by-id.example.test/hook"),
			Enabled:    trans.Ptr(false),
		},
		WebhookSecret: trans.Ptr("by-id-secret"),
	}, 1)
	require.NoError(t, err)

	idEmail, err := repo.Create(ctx, &notificationChannelV1.CreateNotificationChannelRequest{
		Data: &notificationChannelV1.NotificationChannel{
			Name:    trans.Ptr("wh-type-guard"),
			Type:    notificationChannelV1.NotificationChannel_EMAIL.Enum(),
			Enabled: trans.Ptr(true),
		},
	}, 1)
	require.NoError(t, err)

	acct, err := repo.GetDecryptedWebhookAccount(ctx, idWh)
	require.NoError(t, err)
	require.Equal(t, idWh, acct.ID, "取回的账号要自带主键：错误文本与台账都靠它回指")
	require.Equal(t, "https://by-id.example.test/hook", acct.URL)
	require.Equal(t, "by-id-secret", acct.Secret)

	_, err = repo.GetDecryptedWebhookAccount(ctx, idEmail)
	require.Error(t, err, "非 WEBHOOK 渠道应按类型守卫拒绝")
	require.Contains(t, err.Error(), "is not a WEBHOOK channel")

	_, err = repo.GetDecryptedWebhookAccount(ctx, 9999999)
	require.Error(t, err, "不存在的 ID 应返回 NotFound")
}
