// PrintTemplateService 的 SQLite 内存库测试（白盒，包内测试）。
//
// service 层薄在"操作人盖章 + repo 透传"，外加一个无 repo 参与的 RenderPdf 入口，
// 测试盯三件事：
//   - Create/Update 的操作人盖章（created_by/updated_by 取令牌操作人，掩码自动追加
//     updated_by 路径——repo 侧 FilterByFieldMask 会清掩码外字段，追加缺失时盖章静默失效）；
//   - RenderPdf 的滥用守卫：缺 HTML / 缺纸张尺寸 / 超 8MiB 各自拒绝（真实渲染要
//     headless Chrome，主路径不在内存库覆盖范围）；
//   - init 播种：启动期把 DefaultPrintTemplates（3 套内置单据模板）按编码缺一补一。
package service

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	printV1 "go-wind-admin/api/gen/go/print/service/v1"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/printtemplate"
	"go-wind-admin/app/admin/service/internal/data/enttest"
	"go-wind-admin/pkg/constants"
	"go-wind-admin/pkg/middleware/auth"
)

// newPrintTemplateServiceForTest 白盒装配 PrintTemplateService：log 换 NopLogger，
// repo 走 SQLite 内存库。不调 init()（播种单独测）。
func newPrintTemplateServiceForTest(t *testing.T) (*PrintTemplateService, *entCrud.EntClient[*ent.Client]) {
	t.Helper()
	entClient := enttest.NewEntClientForTest(t)
	return &PrintTemplateService{
		log:               bLogger.NewHelper(bLogger.NopLogger()),
		printTemplateRepo: data.NewPrintTemplateRepoForTest(entClient),
	}, entClient
}

// TestPrintTemplateServiceSqlite_CreateUpdateStampsOperator 验证盖章语义：
// created_by/updated_by 取令牌操作人（客户端伪造无效）；Update 自动把 updated_by
// 追加进掩码，掩码外字段保持原值。
func TestPrintTemplateServiceSqlite_CreateUpdateStampsOperator(t *testing.T) {
	svc, entClient := newPrintTemplateServiceForTest(t)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 91})

	// 客户端伪造 created_by=1：以令牌操作人 91 为准
	_, err := svc.Create(opCtx, &printV1.CreatePrintTemplateRequest{
		Data: &printV1.PrintTemplate{
			Name:      trans.Ptr("出库单模板"),
			Code:      trans.Ptr("svc-tpl-a"),
			CreatedBy: trans.Ptr(uint64(1)),
		},
	})
	require.NoError(t, err, "合法创建应成功")

	row, err := entClient.Client().PrintTemplate.Query().
		Where(printtemplate.CodeEQ("svc-tpl-a")).
		Only(ctx)
	require.NoError(t, err)
	require.NotNil(t, row.CreatedBy)
	require.Equal(t, uint64(91), *row.CreatedBy, "created_by 应取令牌操作人")

	// 带掩码更新：客户端掩码只有 name，service 追加 updated_by 后盖章才能落库
	_, err = svc.Update(opCtx, &printV1.UpdatePrintTemplateRequest{
		Id:         row.ID,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"name"}},
		Data:       &printV1.PrintTemplate{Name: trans.Ptr("改名后的模板")},
	})
	require.NoError(t, err, "掩码更新应成功")
	row, err = entClient.Client().PrintTemplate.Query().
		Where(printtemplate.CodeEQ("svc-tpl-a")).
		Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "改名后的模板", *row.Name, "掩码内 name 应更新")
	require.NotNil(t, row.UpdatedBy)
	require.Equal(t, uint64(91), *row.UpdatedBy, "updated_by 应盖章令牌操作人（service 追加掩码路径）")
	require.Equal(t, uint64(91), *row.CreatedBy, "更新不得改写 created_by（掩码外）")

	// 入参守卫与缺操作人
	_, err = svc.Update(opCtx, nil)
	require.Error(t, err, "nil 请求应返回 BadRequest")
	_, err = svc.Update(opCtx, &printV1.UpdatePrintTemplateRequest{Id: row.ID})
	require.Error(t, err, "nil Data 应返回 BadRequest")
	_, err = svc.Create(ctx, &printV1.CreatePrintTemplateRequest{
		Data: &printV1.PrintTemplate{Code: trans.Ptr("svc-tpl-noop")},
	})
	require.Error(t, err, "缺操作人声明的创建应被拒绝")
}

// TestPrintTemplateServiceSqlite_ListGetDeletePassthrough 验证 List/Get/Delete 透传
// 与内置模板禁删守卫穿透。
func TestPrintTemplateServiceSqlite_ListGetDeletePassthrough(t *testing.T) {
	svc, entClient := newPrintTemplateServiceForTest(t)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 92})

	_, err := svc.Create(opCtx, &printV1.CreatePrintTemplateRequest{
		Data: &printV1.PrintTemplate{Name: trans.Ptr("marker-模板"), Code: trans.Ptr("pass-tpl")},
	})
	require.NoError(t, err)
	require.NoError(t, entClient.Client().PrintTemplate.Create().
		SetName("内置模板").
		SetCode("pass-tpl-builtin").
		SetIsBuiltIn(true).
		Exec(ctx))

	list, err := svc.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(2), list.GetTotal())

	row, err := entClient.Client().PrintTemplate.Query().
		Where(printtemplate.CodeEQ("pass-tpl")).
		Only(ctx)
	require.NoError(t, err)
	got, err := svc.Get(ctx, &printV1.GetPrintTemplateRequest{
		QueryBy: &printV1.GetPrintTemplateRequest_Id{Id: row.ID},
	})
	require.NoError(t, err)
	require.Equal(t, "marker-模板", got.GetName())

	builtin, err := entClient.Client().PrintTemplate.Query().
		Where(printtemplate.CodeEQ("pass-tpl-builtin")).
		Only(ctx)
	require.NoError(t, err)
	_, err = svc.Delete(ctx, &printV1.DeletePrintTemplateRequest{
		QueryBy: &printV1.DeletePrintTemplateRequest_Id{Id: builtin.ID},
	})
	require.Error(t, err, "内置模板删除应被守卫拒绝")
	require.Contains(t, err.Error(), "built-in")

	_, err = svc.Delete(ctx, &printV1.DeletePrintTemplateRequest{
		QueryBy: &printV1.DeletePrintTemplateRequest_Id{Id: row.ID},
	})
	require.NoError(t, err, "普通模板删除应成功")
	cnt, err := entClient.Client().PrintTemplate.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, cnt, "删除后只剩内置模板")
}

// TestPrintTemplateServiceSqlite_RenderPdfGuards 验证 RenderPdf 的滥用守卫：
// 缺 HTML、缺纸张尺寸、HTML 超上限各自拒绝；请求合法时才触达 headless 渲染
// （测试环境无 Chrome，主路径不覆盖——守卫必须挡在渲染之前，这就是本用例的边界）。
func TestPrintTemplateServiceSqlite_RenderPdfGuards(t *testing.T) {
	svc, _ := newPrintTemplateServiceForTest(t)
	ctx := enttest.NewSystemContext(context.Background())

	_, err := svc.RenderPdf(ctx, nil)
	require.Error(t, err, "nil 请求应返回 BadRequest")
	require.Contains(t, err.Error(), "html is required")

	_, err = svc.RenderPdf(ctx, &printV1.RenderPrintTemplateRequest{})
	require.Error(t, err, "缺 HTML 应返回 BadRequest")
	require.Contains(t, err.Error(), "html is required")

	_, err = svc.RenderPdf(ctx, &printV1.RenderPrintTemplateRequest{Html: "<p>单</p>"})
	require.Error(t, err, "缺纸张尺寸应返回 BadRequest")
	require.Contains(t, err.Error(), "paper size")

	_, err = svc.RenderPdf(ctx, &printV1.RenderPrintTemplateRequest{
		Html:         strings.Repeat("a", 8<<20+1),
		PaperWidthMm: 210, PaperHeightMm: 297,
	})
	require.Error(t, err, "超 8MiB 的 HTML 应被拒绝")
	require.Contains(t, err.Error(), "html too large")
}

// TestPrintTemplateServiceSqlite_InitSeedsBuiltIn 验证 init 播种：
// 启动期把 DefaultPrintTemplates 按编码缺一补一；重复播种不重复种、不覆盖既有行。
func TestPrintTemplateServiceSqlite_InitSeedsBuiltIn(t *testing.T) {
	svc, entClient := newPrintTemplateServiceForTest(t)
	ctx := enttest.NewSystemContext(context.Background())
	require.NotEmpty(t, constants.DefaultPrintTemplates, "内置打印模板常量不应为空")

	svc.init()

	rows, err := entClient.Client().PrintTemplate.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, len(constants.DefaultPrintTemplates), "首轮播种应落全部内置模板")
	for _, tpl := range constants.DefaultPrintTemplates {
		require.NotNil(t, tpl.Code)
		row, err := entClient.Client().PrintTemplate.Query().
			Where(printtemplate.CodeEQ(*tpl.Code)).
			Only(ctx)
		require.NoError(t, err, "内置模板 %s 应已播种", *tpl.Code)
		require.NotNil(t, row.IsBuiltIn)
		require.True(t, *row.IsBuiltIn, "播种行应带内置标记")
	}

	// 二次播种：不重复、不覆盖（管理员维护过的行不能被种子冲掉）
	require.NoError(t, entClient.Client().PrintTemplate.Update().
		Where(printtemplate.CodeEQ(*constants.DefaultPrintTemplates[0].Code)).
		SetName("管理员改过名").
		Exec(ctx))
	svc.init()
	rows, err = entClient.Client().PrintTemplate.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, len(constants.DefaultPrintTemplates), "二次播种不得重复种")
	for _, r := range rows {
		if *r.Code == *constants.DefaultPrintTemplates[0].Code {
			require.Equal(t, "管理员改过名", *r.Name, "已存在的编码必须跳过不覆盖")
		}
	}
}
