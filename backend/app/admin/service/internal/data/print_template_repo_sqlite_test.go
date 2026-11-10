// PrintTemplateRepo 的 SQLite 内存库测试（白盒，包内测试）。
//
// 打印模板与 BI 仪表盘同构（设计器 JSON 原文透传 + 冗余列），但没有乐观锁——
// 这里盯的是另一组事：
//   - payload（Text 列）逐字节往返：模板 JSON（go-wind-print TemplateData）被改写一个
//     键，用户面单的毫米布局就整体漂移，"服务端不解析"不等于"可以动内容"；
//   - 删除守卫：is_built_in 行是产品内置模板（禁删），普通行可删、缺失幂等；
//   - Update 的 allow_missing 分支：客户端"保存即存在"的语义落在 repo 层，
//     创建时要正确搬运操作人（created_by ← updated_by，且不双写）。
package data

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	printV1 "go-wind-admin/api/gen/go/print/service/v1"
	"go-wind-admin/app/admin/service/internal/data/ent/printtemplate"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// templatePayloadJSON 模拟 go-wind-print 内核的 TemplateData：毫米布局 + schemaVersion，
// 键序与数值原样保留，逐字节断言才能挡住布局漂移。
const templatePayloadJSON = `{"schemaVersion":"1.0","paperSize":"A4","orientation":"portrait","elements":[{"type":"text","field":"order_no","x":10.5,"y":8,"width":60,"height":8,"fontSize":12},{"type":"table","fields":["sku","name","qty"],"x":10,"y":30,"width":190}]}`

// newPrintTemplatePayload 构造一份合法的打印模板载荷，编码可指定，其余按需覆盖。
func newPrintTemplatePayload(code string) *printV1.PrintTemplate {
	return &printV1.PrintTemplate{
		Name:        trans.Ptr("销售出库单-A4"),
		Code:        trans.Ptr(code),
		ResourceKey: trans.Ptr("app.admin.service.OrderService"),
		Payload:     trans.Ptr(templatePayloadJSON),
		IsBuiltIn:   trans.Ptr(false),
		PaperSize:   trans.Ptr("A4"),
		Orientation: trans.Ptr("portrait"),
		IsEnabled:   trans.Ptr(true),
		SortOrder:   trans.Ptr(uint32(5)),
	}
}

// TestPrintTemplateRepoSqlite_Create 验证字段级落库与读视图：
// 业务列按载荷落库（含 payload 原文与冗余列）、created_by 按载荷落库、
// 未携带创建人不写出幻影操作人；nil 载荷与重复编码各自被挡。
func TestPrintTemplateRepoSqlite_Create(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewPrintTemplateRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	payload := newPrintTemplatePayload("order-out-a4")
	payload.CreatedBy = trans.Ptr(int64(1001))
	require.NoError(t, repo.Create(ctx, &printV1.CreatePrintTemplateRequest{Data: payload}),
		"完整载荷的 Create 应成功")

	rows, err := entClient.Client().PrintTemplate.Query().
		Where(printtemplate.CodeEQ("order-out-a4")).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1, "SQLite 中应有 1 条模板记录")
	row := rows[0]
	require.NotNil(t, row.Name)
	require.Equal(t, "销售出库单-A4", *row.Name, "name 应按载荷落库")
	require.NotNil(t, row.Code)
	require.Equal(t, "order-out-a4", *row.Code)
	require.NotNil(t, row.ResourceKey)
	require.Equal(t, "app.admin.service.OrderService", *row.ResourceKey,
		"resource_key 绑定资源标识应按载荷落库：字段面板按它拉字段 schema")
	require.NotNil(t, row.Payload)
	require.JSONEq(t, templatePayloadJSON, *row.Payload, "payload 应以合法 JSON 落库")
	require.Equal(t, templatePayloadJSON, *row.Payload, "payload 是模板原文透传，不得被重新序列化改写布局")
	require.NotNil(t, row.IsBuiltIn)
	require.False(t, *row.IsBuiltIn, "is_built_in=false 要如实落库，删除守卫靠它区分内置模板")
	require.NotNil(t, row.PaperSize)
	require.Equal(t, "A4", *row.PaperSize, "paper_size 冗余列应按载荷落库")
	require.NotNil(t, row.Orientation)
	require.Equal(t, "portrait", *row.Orientation)
	require.NotNil(t, row.IsEnabled)
	require.True(t, *row.IsEnabled)
	require.NotNil(t, row.SortOrder)
	require.Equal(t, uint32(5), *row.SortOrder)
	require.NotNil(t, row.CreatedBy)
	require.Equal(t, int64(1001), *row.CreatedBy, "载荷的 createdBy 应落 created_by")
	require.NotNil(t, row.CreatedAt, "created_at 必须显式写入")
	require.NotNil(t, row.TenantID)
	require.Equal(t, int64(0), *row.TenantID, "系统上下文落平台租户 0")

	// 读视图：DTO 全量回读
	dto, err := repo.Get(ctx, &printV1.GetPrintTemplateRequest{
		QueryBy: &printV1.GetPrintTemplateRequest_Id{Id: row.ID},
	})
	require.NoError(t, err)
	require.Equal(t, row.ID, dto.GetId())
	require.Equal(t, "销售出库单-A4", dto.GetName())
	require.Equal(t, "app.admin.service.OrderService", dto.GetResourceKey())
	require.Equal(t, templatePayloadJSON, dto.GetPayload(), "读视图的 payload 必须与写入原文逐字节相等")
	require.Equal(t, "A4", dto.GetPaperSize())
	require.Equal(t, "portrait", dto.GetOrientation())
	require.False(t, dto.GetIsBuiltIn())
	require.Equal(t, int64(1001), dto.GetCreatedBy())

	// 未携带创建人的创建（系统播种路径）：不写出不存在的操作人
	require.NoError(t, repo.Create(ctx, &printV1.CreatePrintTemplateRequest{
		Data: newPrintTemplatePayload("seeded-no-op"),
	}))
	row2, err := entClient.Client().PrintTemplate.Query().
		Where(printtemplate.CodeEQ("seeded-no-op")).
		Only(ctx)
	require.NoError(t, err)
	require.Nil(t, row2.CreatedBy, "未携带创建人时不该写出一个不存在的操作人")

	// 入参守卫
	require.Error(t, repo.Create(ctx, nil), "nil 请求应返回 BadRequest")
	require.Error(t, repo.Create(ctx, &printV1.CreatePrintTemplateRequest{}), "nil Data 应返回 BadRequest")

	// 唯一索引 (code, tenant_id)：同租户重复编码 → 约束错误翻译成 BadRequest
	require.Error(t, repo.Create(ctx, &printV1.CreatePrintTemplateRequest{
		Data: newPrintTemplatePayload("order-out-a4"),
	}), "同租户重复编码应被唯一索引挡下")
	cnt, err := entClient.Client().PrintTemplate.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, cnt, "失败的创建不应增加行数")
}

// TestPrintTemplateRepoSqlite_GetIsExist 验证 Get 命中/未命中与 nil 守卫、
// IsExist 的真假两侧。
func TestPrintTemplateRepoSqlite_GetIsExist(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewPrintTemplateRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	require.NoError(t, repo.Create(ctx, &printV1.CreatePrintTemplateRequest{
		Data: newPrintTemplatePayload("get-probe"),
	}))
	row, err := entClient.Client().PrintTemplate.Query().
		Where(printtemplate.CodeEQ("get-probe")).
		Only(ctx)
	require.NoError(t, err)

	dto, err := repo.Get(ctx, &printV1.GetPrintTemplateRequest{
		QueryBy: &printV1.GetPrintTemplateRequest_Id{Id: row.ID},
	})
	require.NoError(t, err, "按存在 ID 查询应命中")
	require.Equal(t, "get-probe", dto.GetCode())

	_, err = repo.Get(ctx, &printV1.GetPrintTemplateRequest{
		QueryBy: &printV1.GetPrintTemplateRequest_Id{Id: 9999999},
	})
	require.Error(t, err, "不存在的 ID 应返回错误")

	_, err = repo.Get(ctx, nil)
	require.Error(t, err, "nil 请求应返回 BadRequest")

	exist, err := repo.IsExist(ctx, row.ID)
	require.NoError(t, err)
	require.True(t, exist, "已写入的模板应判定为存在")
	exist, err = repo.IsExist(ctx, 9999999)
	require.NoError(t, err)
	require.False(t, exist, "不存在的模板应判定为不存在")
}

// TestPrintTemplateRepoSqlite_List 验证分页 total 与 name 的 contains 过滤；
// Count 与 List 的 total 同源；nil 请求返回 BadRequest。
func TestPrintTemplateRepoSqlite_List(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewPrintTemplateRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	payloadA := newPrintTemplatePayload("list-alpha")
	payloadA.Name = trans.Ptr("出库单-marker")
	require.NoError(t, repo.Create(ctx, &printV1.CreatePrintTemplateRequest{Data: payloadA}))

	payloadB := newPrintTemplatePayload("list-beta")
	payloadB.Name = trans.Ptr("盘点单")
	payloadB.ResourceKey = trans.Ptr("app.admin.service.StockService")
	require.NoError(t, repo.Create(ctx, &printV1.CreatePrintTemplateRequest{Data: payloadB}))

	all, err := repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, int64(2), all.GetTotal(), "无过滤时 total 应统计全部行")
	require.Len(t, all.GetItems(), 2)
	// 列表读视图同样带回 payload 与冗余列（列表页直显免解析）
	byCode := map[string]*printV1.PrintTemplate{}
	for _, it := range all.GetItems() {
		byCode[it.GetCode()] = it
	}
	require.Equal(t, "app.admin.service.StockService", byCode["list-beta"].GetResourceKey())
	require.Equal(t, templatePayloadJSON, byCode["list-alpha"].GetPayload(), "列表行的 payload 也要无损回读")

	filtered, err := repo.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_Query{
			Query: `{"name__contains":"marker"}`,
		},
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), filtered.GetTotal(), "contains 过滤的 total 应只统计命中行")
	require.Len(t, filtered.GetItems(), 1)
	require.Equal(t, "list-alpha", filtered.GetItems()[0].GetCode())

	_, err = repo.List(ctx, nil)
	require.Error(t, err, "nil 分页请求应返回 BadRequest")

	cnt, err := repo.Count(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, int64(2), cnt, "Count 应与无过滤 List 的 total 一致")
}

// TestPrintTemplateRepoSqlite_Update 验证更新语义：
// 掩码内字段更新、掩码外保持原值、updated_by 盖操作人；
// allow_missing=true 且行不存在时按载荷创建（created_by ← updated_by 且不双写）；
// 不存在 ID 的普通更新静默无效果；nil/id=0 守卫齐全。
func TestPrintTemplateRepoSqlite_Update(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewPrintTemplateRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	require.NoError(t, repo.Create(ctx, &printV1.CreatePrintTemplateRequest{
		Data: newPrintTemplatePayload("upd-tpl"),
	}))
	row, err := entClient.Client().PrintTemplate.Query().
		Where(printtemplate.CodeEQ("upd-tpl")).
		Only(ctx)
	require.NoError(t, err)

	// 掩码 [paper_size, orientation]：模板设计器只改纸张，payload 原文必须原样保留
	rotatedPayload := `{"schemaVersion":"1.0","paperSize":"A4","orientation":"landscape","elements":[]}`
	require.NoError(t, repo.Update(ctx, &printV1.UpdatePrintTemplateRequest{
		Id:         row.ID,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"paper_size", "orientation", "payload"}},
		Data: &printV1.PrintTemplate{
			PaperSize:   trans.Ptr("80mm"),
			Orientation: trans.Ptr("landscape"),
			Payload:     trans.Ptr(rotatedPayload),
			Name:        trans.Ptr("掩码外不该生效"),
		},
	}), "掩码内字段更新应成功")

	row, err = entClient.Client().PrintTemplate.Query().
		Where(printtemplate.CodeEQ("upd-tpl")).
		Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "80mm", *row.PaperSize, "掩码内的 paper_size 应更新")
	require.Equal(t, "landscape", *row.Orientation, "掩码内的 orientation 应更新")
	require.Equal(t, rotatedPayload, *row.Payload, "掩码内的 payload 应整份更新")
	require.Equal(t, "销售出库单-A4", *row.Name, "掩码外的 name 应保持原值")
	require.Nil(t, row.UpdatedBy, "未携带更新人时不该写幻影操作人")

	// 带更新人的更新：updated_by 盖章（与生产 service 层对齐：调用方会把 updated_by
	// 追加进掩码——否则 FilterByFieldMask 会把掩码外的 UpdatedBy 清零，这是掩码语义本身）
	require.NoError(t, repo.Update(ctx, &printV1.UpdatePrintTemplateRequest{
		Id:         row.ID,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"name", "updated_by"}},
		Data: &printV1.PrintTemplate{
			Name:      trans.Ptr("改名后的模板"),
			UpdatedBy: trans.Ptr(int64(2001)),
		},
	}))
	row, err = entClient.Client().PrintTemplate.Query().
		Where(printtemplate.CodeEQ("upd-tpl")).
		Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "改名后的模板", *row.Name)
	require.NotNil(t, row.UpdatedBy)
	require.Equal(t, int64(2001), *row.UpdatedBy, "operatorID 应落 updated_by")
	require.NotNil(t, row.UpdatedAt, "更新应写入 updated_at")
	require.Equal(t, "80mm", *row.PaperSize, "掩码外的 paper_size 应保持上次值")

	// allow_missing=true 且行不存在：按载荷创建（保存即存在的语义）。
	// 与生产调用方对齐：service 层会先把 req.Id 拷进 Data.Id 再进 repo，
	// 创建分支的 SetID 依赖它，这里如实带上。
	require.NoError(t, repo.Update(ctx, &printV1.UpdatePrintTemplateRequest{
		Id:           424242,
		AllowMissing: trans.Ptr(true),
		Data: func(d *printV1.PrintTemplate) *printV1.PrintTemplate {
			d.Id = trans.Ptr(int64(424242))
			d.Code = trans.Ptr("allow-missing-tpl")
			d.UpdatedBy = trans.Ptr(int64(3001))
			return d
		}(newPrintTemplatePayload("allow-missing-tpl")),
	}), "allow_missing 且行不存在时应按载荷创建")
	rowMissing, err := entClient.Client().PrintTemplate.Query().
		Where(printtemplate.CodeEQ("allow-missing-tpl")).
		Only(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(424242), rowMissing.ID, "创建应落在请求指定的 ID 上")
	require.NotNil(t, rowMissing.CreatedBy)
	require.Equal(t, int64(3001), *rowMissing.CreatedBy,
		"创建分支的 created_by 应取自 updated_by：操作人不能在两条审计列里丢一个")
	require.Nil(t, rowMissing.UpdatedBy, "创建分支不应双写 updated_by")

	// 不存在 ID 且未开 allow_missing：静默无效果（不该凭空造行）
	require.NoError(t, repo.Update(ctx, &printV1.UpdatePrintTemplateRequest{
		Id:   987654,
		Data: &printV1.PrintTemplate{Name: trans.Ptr("ghost")},
	}), "对不存在 ID 的普通更新应静默无效果")
	cnt, err := entClient.Client().PrintTemplate.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, cnt, "对不存在 ID 的更新不应新增行")

	// 入参守卫
	require.Error(t, repo.Update(ctx, nil), "nil 请求应返回 BadRequest")
	require.Error(t, repo.Update(ctx, &printV1.UpdatePrintTemplateRequest{Id: row.ID}), "nil Data 应返回 BadRequest")
	require.Error(t, repo.Update(ctx, &printV1.UpdatePrintTemplateRequest{
		Id:   0,
		Data: &printV1.PrintTemplate{},
	}), "id=0 应返回 BadRequest")
}

// TestPrintTemplateRepoSqlite_Delete 验证删除守卫：普通行可删、内置模板禁删、
// 不存在的 ID 幂等成功、nil 请求 BadRequest。
func TestPrintTemplateRepoSqlite_Delete(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewPrintTemplateRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	require.NoError(t, repo.Create(ctx, &printV1.CreatePrintTemplateRequest{
		Data: newPrintTemplatePayload("del-normal"),
	}))
	rowNormal, err := entClient.Client().PrintTemplate.Query().
		Where(printtemplate.CodeEQ("del-normal")).
		Only(ctx)
	require.NoError(t, err)

	payloadBuiltin := newPrintTemplatePayload("del-builtin")
	payloadBuiltin.IsBuiltIn = trans.Ptr(true)
	require.NoError(t, repo.Create(ctx, &printV1.CreatePrintTemplateRequest{Data: payloadBuiltin}))
	rowBuiltin, err := entClient.Client().PrintTemplate.Query().
		Where(printtemplate.CodeEQ("del-builtin")).
		Only(ctx)
	require.NoError(t, err)

	// 内置模板禁删：内置模板是产品能力的一部分，删了采用者升级后就没了
	err = repo.Delete(ctx, &printV1.DeletePrintTemplateRequest{
		QueryBy: &printV1.DeletePrintTemplateRequest_Id{Id: rowBuiltin.ID},
	})
	require.Error(t, err, "内置模板应被禁删守卫挡下")
	require.Contains(t, err.Error(), "built-in")

	// 普通行删除
	require.NoError(t, repo.Delete(ctx, &printV1.DeletePrintTemplateRequest{
		QueryBy: &printV1.DeletePrintTemplateRequest_Id{Id: rowNormal.ID},
	}), "普通行删除应成功")
	cnt, err := entClient.Client().PrintTemplate.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, cnt, "只该删掉普通行")

	// 幂等删除：目标不存在视为已删除（批量清理与重试场景依赖这个语义）
	require.NoError(t, repo.Delete(ctx, &printV1.DeletePrintTemplateRequest{
		QueryBy: &printV1.DeletePrintTemplateRequest_Id{Id: 9999999},
	}), "删除不存在的 ID 应幂等成功")

	require.Error(t, repo.Delete(ctx, nil), "nil 请求应返回 BadRequest")
}

// TestPrintTemplateRepoSqlite_SeedDefaults 验证按编码缺一补一的播种语义：
// 编码不存在才创建、已存在跳过不覆盖、空编码/nil 条目跳过；重复调用不产生重复行。
func TestPrintTemplateRepoSqlite_SeedDefaults(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewPrintTemplateRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	defaults := []*printV1.PrintTemplate{
		newPrintTemplatePayload("seed-a"),
		func(d *printV1.PrintTemplate) *printV1.PrintTemplate {
			d.Code = trans.Ptr("seed-b")
			d.Name = trans.Ptr("内置盘点单模板")
			return d
		}(newPrintTemplatePayload("seed-b")),
		nil,                   // nil 条目跳过
		{Code: trans.Ptr("")}, // 空编码跳过：编码是播种幂等的键
	}
	require.NoError(t, repo.SeedDefaults(ctx, defaults), "首轮播种应成功")

	// 模拟内置模板被管理员改过名
	_, err := entClient.Client().PrintTemplate.Update().
		Where(printtemplate.CodeEQ("seed-a")).
		SetName("管理员改过名").
		Save(ctx)
	require.NoError(t, err)

	require.NoError(t, repo.SeedDefaults(ctx, defaults), "二次播种应成功（已存在的跳过）")

	rows, err := entClient.Client().PrintTemplate.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 2, "重复播种不得产生重复行")
	for _, r := range rows {
		require.NotNil(t, r.Code)
		require.NotNil(t, r.Name)
		switch *r.Code {
		case "seed-a":
			require.Equal(t, "管理员改过名", *r.Name,
				"已存在的编码必须跳过不覆盖：播种无权动管理员维护过的行")
		case "seed-b":
			require.Equal(t, "内置盘点单模板", *r.Name, "内置行应按默认值落库")
		}
	}

	// 补种新编码：只有缺失的进来
	defaults = append(defaults, newPrintTemplatePayload("seed-c"))
	require.NoError(t, repo.SeedDefaults(ctx, defaults))
	cnt, err := entClient.Client().PrintTemplate.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 3, cnt, "第三轮播种只应新增 seed-c 一行")
}
