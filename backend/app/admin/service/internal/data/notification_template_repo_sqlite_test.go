// NotificationTemplateRepo 的 SQLite 内存库测试（白盒，包内测试）。
//
// 模板仓没有枚举列，它的命门是 code：code 是 SendDirect 与测试渲染的引用锚，
// 全局唯一——GetByCode 的"没配这个模板"与"查库失败"必须分开（(nil,nil) vs error，
// 压成同一种错误会把排障方向引反），撞码必须在写入前显式拒掉（唯一索引兜底并发窗口）。
// 另外钉住 Update 的掩码语义：掩码内更新、掩码外原值、updated_by 盖操作人。
package data

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	notificationV1 "go-wind-admin/api/gen/go/notification/service/v1"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newTemplate 构造一份合法模板载荷，字段可按需覆盖。
func newTemplate(code, title, content string) *notificationV1.NotificationTemplate {
	return &notificationV1.NotificationTemplate{
		Name:            trans.Ptr("模板-" + code),
		Code:            trans.Ptr(code),
		TitleTemplate:   trans.Ptr(title),
		ContentTemplate: trans.Ptr(content),
		IsEnabled:       trans.Ptr(true),
	}
}

// TestNotificationTemplateRepoSqlite_Create 验证字段级落库与读回：
// 占位模板四列（name/code/title/content）逐列命中、is_enabled=false 不被默认吞、
// created_at 显式写入（mixin 列没有 DB 默认值）、operatorID 落 created_by、
// operatorID=0（系统播种）不落 created_by。
func TestNotificationTemplateRepoSqlite_Create(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewNotificationTemplateRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	id, err := repo.Create(ctx, &notificationV1.NotificationTemplate{
		Name:            trans.Ptr("找回密码"),
		Code:            trans.Ptr("password_reset_code"),
		TitleTemplate:   trans.Ptr("【{{app}}】找回密码"),
		ContentTemplate: trans.Ptr("验证码 {{code}}，{{minutes}} 分钟内有效"),
		IsEnabled:       trans.Ptr(true),
		Remark:          trans.Ptr(" SendDirect 按 code 引用"),
	}, 1001)
	require.NoError(t, err, "完整载荷的 Create 应成功")
	require.NotZero(t, id)

	row, err := entClient.Client().NotificationTemplate.Get(ctx, id)
	require.NoError(t, err)
	require.NotNil(t, row.Name)
	require.Equal(t, "找回密码", *row.Name)
	require.Equal(t, "password_reset_code", *row.Code)
	require.Equal(t, "【{{app}}】找回密码", *row.TitleTemplate)
	require.Equal(t, "验证码 {{code}}，{{minutes}} 分钟内有效", *row.ContentTemplate)
	require.NotNil(t, row.IsEnabled)
	require.True(t, *row.IsEnabled)
	require.NotNil(t, row.Remark)
	require.Equal(t, " SendDirect 按 code 引用", *row.Remark)
	require.NotNil(t, row.CreatedBy)
	require.Equal(t, uint64(1001), *row.CreatedBy, "operatorID 应落 created_by")
	require.NotNil(t, row.CreatedAt, "created_at 必须显式写入：mixin 列没有 DB 默认值")
	require.Nil(t, row.UpdatedAt, "新建行不该有 updated_at")

	// 读视图：占位模板原样回读（渲染在服务层，仓只管存取不失真）
	dto, err := repo.Get(ctx, id)
	require.NoError(t, err)
	require.Equal(t, id, dto.GetId())
	require.Equal(t, "password_reset_code", dto.GetCode())
	require.Equal(t, "【{{app}}】找回密码", dto.GetTitleTemplate())
	require.Equal(t, "验证码 {{code}}，{{minutes}} 分钟内有效", dto.GetContentTemplate())
	require.True(t, dto.GetIsEnabled())

	// 停用模板：is_enabled=false 必须如实落库，不能因"默认 true"被吞
	// （SendDirect 只用启用的模板，false 存成 true 会让下线模板继续发出去）
	disabledID, err := repo.Create(ctx, &notificationV1.NotificationTemplate{
		Name:      trans.Ptr("已下线模板"),
		Code:      trans.Ptr("disabled_tpl"),
		IsEnabled: trans.Ptr(false),
	}, 1)
	require.NoError(t, err)
	drow, err := entClient.Client().NotificationTemplate.Get(ctx, disabledID)
	require.NoError(t, err)
	require.NotNil(t, drow.IsEnabled)
	require.False(t, *drow.IsEnabled, "is_enabled=false 应如实落库")

	// 播种（operatorID=0）：created_by 不该写出一个不存在的操作人；未指定 is_enabled 落列默认 true
	seededID, err := repo.Create(ctx, newTemplate("seeded_tpl", "{{t}}", "{{c}}"), 0)
	require.NoError(t, err)
	srow, err := entClient.Client().NotificationTemplate.Get(ctx, seededID)
	require.NoError(t, err)
	require.Nil(t, srow.CreatedBy, "播种（operatorID=0）不该写 created_by")
	require.NotNil(t, srow.IsEnabled)
	require.True(t, *srow.IsEnabled, "未指定 is_enabled 应落列默认 true")

	// 入参守卫
	_, err = repo.Create(ctx, nil, 1)
	require.Error(t, err, "nil 载荷应返回 BadRequest")
	_, err = repo.Create(ctx, &notificationV1.NotificationTemplate{Name: trans.Ptr("无编码")}, 1)
	require.Error(t, err, "缺 code 应返回 BadRequest：code 是发送方的引用锚")
	require.Contains(t, err.Error(), "code is required")
}

// TestNotificationTemplateRepoSqlite_CreateRejectsDuplicateCode 验证撞码在写入前
// 被显式拒掉：两个模板共用一个 code 时 GetByCode 的 Only 直接抛错，SendDirect
// 整条引用链断掉，所以第二行必须进不了库。
func TestNotificationTemplateRepoSqlite_CreateRejectsDuplicateCode(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewNotificationTemplateRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	_, err := repo.Create(ctx, newTemplate("dup_tpl", "{{t}}", "{{c}}"), 1)
	require.NoError(t, err)

	_, err = repo.Create(ctx, newTemplate("dup_tpl", "别的标题", "别的正文"), 1)
	require.Error(t, err, "重复 code 应被拒绝")
	require.Contains(t, err.Error(), "already exists")

	// 失败的创建不得增加行数
	cnt, err := entClient.Client().NotificationTemplate.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, cnt)

	// 换个 code 仍可入库（查重不能顺手把整张表锁住）
	_, err = repo.Create(ctx, newTemplate("other_tpl", "{{t}}", "{{c}}"), 1)
	require.NoError(t, err)
}

// TestNotificationTemplateRepoSqlite_GetByCode 验证三态分开：命中回 DTO；
// 没有这行回 (nil, nil)——"没配这个模板"是配置事实，不是查询失败；
// 空 code 也回 (nil, nil)（调用方传参错误同样不该伪装成基础设施故障）。
func TestNotificationTemplateRepoSqlite_GetByCode(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewNotificationTemplateRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	// 空表：未命中必须是 (nil, nil)，让 SendDirect 走"模板未配置"分支而非报错
	miss, err := repo.GetByCode(ctx, "no_such_tpl")
	require.NoError(t, err, "没有该模板不是错误")
	require.Nil(t, miss)

	id, err := repo.Create(ctx, newTemplate("hit_tpl", "{{t}}", "{{c}}"), 1)
	require.NoError(t, err)

	got, err := repo.GetByCode(ctx, "hit_tpl")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, id, got.GetId())
	require.Equal(t, "hit_tpl", got.GetCode())

	// 空 code：(nil, nil)
	miss, err = repo.GetByCode(ctx, "")
	require.NoError(t, err, "空 code 应直接短路为未命中")
	require.Nil(t, miss)

	// 删除后回到 (nil, nil)：SendDirect 据此报"模板未配置"而非命中已删内容
	require.NoError(t, repo.Delete(ctx, id))
	miss, err = repo.GetByCode(ctx, "hit_tpl")
	require.NoError(t, err)
	require.Nil(t, miss, "删除后该 code 应回到未命中")
}

// TestNotificationTemplateRepoSqlite_List 验证分页 total 与 code 的 contains 过滤；
// nil 请求返回 BadRequest。
func TestNotificationTemplateRepoSqlite_List(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewNotificationTemplateRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	_, err := repo.Create(ctx, newTemplate("marker_tpl", "{{t}}", "{{c}}"), 1)
	require.NoError(t, err)
	_, err = repo.Create(ctx, newTemplate("other_tpl", "{{t}}", "{{c}}"), 1)
	require.NoError(t, err)

	all, err := repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(2), all.GetTotal())
	require.Len(t, all.GetItems(), 2)

	filtered, err := repo.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_Query{
			Query: `{"code__contains":"marker"}`,
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), filtered.GetTotal(), "contains 过滤应只统计命中行")
	require.Len(t, filtered.GetItems(), 1)
	require.Equal(t, "marker_tpl", filtered.GetItems()[0].GetCode())

	_, err = repo.List(ctx, nil)
	require.Error(t, err, "nil 分页请求应返回 BadRequest")
}

// TestNotificationTemplateRepoSqlite_Update 验证掩码语义与撞码防护：
//   - 掩码内字段更新、掩码外字段（含 code）保持原值，updated_by/updated_at 落位；
//   - 改 code 时：改成自由 code 放行、改成别人的 code 显式拒掉、保持自己原 code 不触发查重；
//   - 不存在 ID：带 code 走 Get 查重路径报 NotFound；不带 code 的纯掩码更新静默无效果。
func TestNotificationTemplateRepoSqlite_Update(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewNotificationTemplateRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	id, err := repo.Create(ctx, &notificationV1.NotificationTemplate{
		Name:            trans.Ptr("原标题"),
		Code:            trans.Ptr("origin_tpl"),
		TitleTemplate:   trans.Ptr("原 {{title}}"),
		ContentTemplate: trans.Ptr("原 {{content}}"),
		IsEnabled:       trans.Ptr(true),
		Remark:          trans.Ptr("orig-remark"),
	}, 1001)
	require.NoError(t, err)

	// 改标题与正文，掩码外的 code/name/remark 保持原值
	require.NoError(t, repo.Update(ctx, &notificationV1.UpdateNotificationTemplateRequest{
		Id: id,
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{"title_template", "content_template"},
		},
		Data: &notificationV1.NotificationTemplate{
			TitleTemplate:   trans.Ptr("新 {{title}}"),
			ContentTemplate: trans.Ptr("新 {{content}}"),
			Code:            trans.Ptr("masked-out-code"),
			Name:            trans.Ptr("masked-out-name"),
			Remark:          trans.Ptr("masked-out-remark"),
		},
	}, 2001))

	row, err := entClient.Client().NotificationTemplate.Get(ctx, id)
	require.NoError(t, err)
	require.Equal(t, "新 {{title}}", *row.TitleTemplate, "掩码内的标题应更新")
	require.Equal(t, "新 {{content}}", *row.ContentTemplate, "掩码内的正文应更新")
	require.Equal(t, "origin_tpl", *row.Code, "掩码外的 code 应保持原值")
	require.Equal(t, "原标题", *row.Name, "掩码外的 name 应保持原值")
	require.Equal(t, "orig-remark", *row.Remark, "掩码外的 remark 应保持原值")
	require.NotNil(t, row.UpdatedBy)
	require.Equal(t, uint64(2001), *row.UpdatedBy, "operatorID 应落 updated_by")
	require.NotNil(t, row.UpdatedAt, "更新应写入 updated_at")

	// 保持自己的 code 不变：同一 code 不触发查重，更新应放行
	require.NoError(t, repo.Update(ctx, &notificationV1.UpdateNotificationTemplateRequest{
		Id:         id,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"code", "name"}},
		Data: &notificationV1.NotificationTemplate{
			Code: trans.Ptr("origin_tpl"),
			Name: trans.Ptr("改名不改码"),
		},
	}, 2002))
	row, err = entClient.Client().NotificationTemplate.Get(ctx, id)
	require.NoError(t, err)
	require.Equal(t, "origin_tpl", *row.Code)
	require.Equal(t, "改名不改码", *row.Name)

	// 改成自由 code：放行
	require.NoError(t, repo.Update(ctx, &notificationV1.UpdateNotificationTemplateRequest{
		Id:         id,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"code"}},
		Data:       &notificationV1.NotificationTemplate{Code: trans.Ptr("renamed_tpl")},
	}, 2003))
	row, err = entClient.Client().NotificationTemplate.Get(ctx, id)
	require.NoError(t, err)
	require.Equal(t, "renamed_tpl", *row.Code)

	// 改成别人的 code：显式拒掉（唯一索引兜底并发窗口，这里验证的是前置查重）
	_, err = repo.Create(ctx, newTemplate("taken_tpl", "{{t}}", "{{c}}"), 1)
	require.NoError(t, err)
	err = repo.Update(ctx, &notificationV1.UpdateNotificationTemplateRequest{
		Id:         id,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"code"}},
		Data:       &notificationV1.NotificationTemplate{Code: trans.Ptr("taken_tpl")},
	}, 2004)
	require.Error(t, err, "改成他人已占用的 code 应被拒绝")
	require.Contains(t, err.Error(), "already exists")

	// 不存在 ID + code 在掩码内：Get 查重路径报 NotFound
	err = repo.Update(ctx, &notificationV1.UpdateNotificationTemplateRequest{
		Id:         9999999,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"code"}},
		Data:       &notificationV1.NotificationTemplate{Code: trans.Ptr("brand_new")},
	}, 1)
	require.Error(t, err, "对不存在行改 code 应报 NotFound")
	require.Contains(t, err.Error(), "not found")

	// 不存在 ID + 不带 code：纯掩码更新静默无效果、不新增行
	require.NoError(t, repo.Update(ctx, &notificationV1.UpdateNotificationTemplateRequest{
		Id:         9999999,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"name"}},
		Data:       &notificationV1.NotificationTemplate{Name: trans.Ptr("幽灵")},
	}, 1))
	cnt, err := entClient.Client().NotificationTemplate.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, cnt, "对不存在 ID 的更新不应新增行")

	// 入参守卫
	require.Error(t, repo.Update(ctx, nil, 1), "nil 请求应返回 BadRequest")
	require.Error(t, repo.Update(ctx, &notificationV1.UpdateNotificationTemplateRequest{Id: id}, 1),
		"nil Data 应返回 BadRequest")
	require.Error(t, repo.Update(ctx, &notificationV1.UpdateNotificationTemplateRequest{
		Id:   0,
		Data: &notificationV1.NotificationTemplate{},
	}, 1), "id=0 应返回 BadRequest")
}

// TestNotificationTemplateRepoSqlite_Delete 验证删除指定行、id=0 的 BadRequest、
// 不存在 ID 返回错误；删除后 GetByCode 回到 (nil, nil)。
func TestNotificationTemplateRepoSqlite_Delete(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewNotificationTemplateRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	idA, err := repo.Create(ctx, newTemplate("del_a", "{{t}}", "{{c}}"), 1)
	require.NoError(t, err)
	_, err = repo.Create(ctx, newTemplate("del_b", "{{t}}", "{{c}}"), 1)
	require.NoError(t, err)

	require.NoError(t, repo.Delete(ctx, idA), "按存在 ID 删除应成功")

	miss, err := repo.GetByCode(ctx, "del_a")
	require.NoError(t, err)
	require.Nil(t, miss, "删除后该 code 应回到未命中")

	cnt, err := entClient.Client().NotificationTemplate.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, cnt, "只该删掉一行")

	require.Error(t, repo.Delete(ctx, 0), "id=0 应返回 BadRequest")
	require.Error(t, repo.Delete(ctx, 9999999), "删除不存在的 ID 应返回错误")
	cnt, err = entClient.Client().NotificationTemplate.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, cnt, "失败删除不应影响行数")
}
