// BiDashboardRepo 的 SQLite 内存库测试（白盒，包内测试）。
//
// 这张表是 BI 设计器的整份保存真相，测试盯三件"丢一次就是用户辛苦搭的仪表盘没了"的事：
//   - 乐观锁（version CAS）：设计器整份覆盖保存，多标签/多人同时编辑靠 (id, version) 命中
//     才写入；stale 写必须回 Conflict 而不是静默覆盖别人的成果，命中不了的两种可能
//     （行已删 / 版本过期）要给出可区分的错误；
//   - payload JSONB 列：服务端不解析只透传，但"透传"的前提是逐字节无损——SQLite 下
//     JSON 落 text，写侧经 ent 默认 string 类型、读侧经 copier 回 DTO，两端都要实测；
//   - 枚举三形态：kind 为可空指针枚举列，proto ↔ ent 经 converter 按名字配对，
//     读视图必须落回正确的 proto 成员（落成零值 DASHBOARD 时大屏会被错渲染成仪表盘）。
package data

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	biV1 "go-wind-admin/api/gen/go/bi/service/v1"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/bidashboard"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// designerPayloadJSON 模拟设计器产出的 schema 原文：嵌套对象 + 中文 + 数字。
// payload 是设计器原文透传，测试同时用 JSONEq（键值等价）与字符串全等
// （逐字节不漂移）两道断言：后者挡的是"被重新序列化/转义改写"这类静默漂移。
const designerPayloadJSON = `{"version":"1.0","theme":{"key":"ocean-dark"},"charts":[{"id":"c1","type":"line","title":"月度销售额","layout":{"x":0,"y":0,"w":6,"h":4}}],"datasets":[{"name":"sales","query":"SELECT month, amount FROM sales"}]}`

// newBiDashboardPayload 构造一份合法的仪表盘载荷，编码可指定，其余字段按需覆盖。
func newBiDashboardPayload(code string) *biV1.BiDashboard {
	return &biV1.BiDashboard{
		Name:       trans.Ptr("销售分析看板"),
		Code:       trans.Ptr(code),
		Kind:       biV1.BiDashboardKind_SCREEN.Enum(),
		Remark:     trans.Ptr("月度经营复盘"),
		Payload:    trans.Ptr(designerPayloadJSON),
		ChartCount: trans.Ptr(uint32(1)),
		ThemeKey:   trans.Ptr("ocean-dark"),
		Thumbnail:  trans.Ptr("https://cdn.example.test/thumb/c1.png"),
		IsBuiltIn:  trans.Ptr(false),
		IsPublic:   trans.Ptr(true),
		IsEnabled:  trans.Ptr(true),
		SortOrder:  trans.Ptr(uint32(10)),
	}
}

// mustGetBiDashboardRow 按编码直查 ent 行（测试内绕开 repo 读路径做落库侧断言）。
func mustGetBiDashboardRow(t *testing.T, entClient *ent.Client, ctx context.Context, code string) *ent.BiDashboard {
	t.Helper()
	row, err := entClient.BiDashboard.Query().
		Where(bidashboard.CodeEQ(code)).
		Only(ctx)
	require.NoError(t, err, "按编码 %s 直查 ent 行应命中", code)
	return row
}

// TestBiDashboardRepoSqlite_Create 验证字段级落库与读视图：
// 各业务列按载荷落库（含 payload JSON 原文）、kind 枚举经 converter 落实体字面量、
// created_by 按载荷落库、未显式给 version 时落列默认 1（乐观锁的起点）；
// 未携带创建人时不得写出不存在的操作人；nil 载荷与重复编码各自被挡。
func TestBiDashboardRepoSqlite_Create(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewBiDashboardRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	payload := newBiDashboardPayload("sales-monthly")
	payload.CreatedBy = trans.Ptr(int64(1001))
	require.NoError(t, repo.Create(ctx, &biV1.CreateBiDashboardRequest{Data: payload}),
		"完整载荷的 Create 应成功")

	row := mustGetBiDashboardRow(t, entClient.Client(), ctx, "sales-monthly")
	require.NotNil(t, row.Name)
	require.Equal(t, "销售分析看板", *row.Name, "name 应按载荷落库")
	require.NotNil(t, row.Code)
	require.Equal(t, "sales-monthly", *row.Code, "code 应按载荷落库")
	require.NotNil(t, row.Kind)
	require.Equal(t, bidashboard.KindSCREEN, *row.Kind, "proto SCREEN 应经 converter 落为实体 KindSCREEN")
	require.NotNil(t, row.Remark)
	require.Equal(t, "月度经营复盘", *row.Remark)
	require.NotNil(t, row.Payload)
	require.JSONEq(t, designerPayloadJSON, *row.Payload, "payload 应以合法 JSON 落库（SQLite 下为 text）")
	require.Equal(t, designerPayloadJSON, *row.Payload, "payload 是设计器原文透传，不得被重新序列化或转义改写")
	require.NotNil(t, row.ChartCount)
	require.Equal(t, uint32(1), *row.ChartCount, "chart_count 冗余列应按载荷落库")
	require.NotNil(t, row.ThemeKey)
	require.Equal(t, "ocean-dark", *row.ThemeKey)
	require.NotNil(t, row.Thumbnail)
	require.NotNil(t, row.IsBuiltIn)
	require.False(t, *row.IsBuiltIn, "is_built_in=false 要如实落库，删除守卫靠它区分内置行")
	require.NotNil(t, row.IsPublic)
	require.True(t, *row.IsPublic)
	require.NotNil(t, row.IsEnabled)
	require.True(t, *row.IsEnabled)
	require.NotNil(t, row.SortOrder)
	require.Equal(t, uint32(10), *row.SortOrder)
	require.NotNil(t, row.CreatedBy)
	require.Equal(t, int64(1001), *row.CreatedBy, "载荷的 createdBy 应落 created_by")
	require.Nil(t, row.UpdatedBy, "Create 不写 updated_by")
	require.NotNil(t, row.CreatedAt, "created_at 必须显式写入")
	require.NotNil(t, row.Version)
	require.Equal(t, uint32(1), *row.Version, "未显式给 version 时应落列默认 1：乐观锁从 1 起步")
	require.NotNil(t, row.TenantID)
	require.Equal(t, int64(0), *row.TenantID, "系统上下文落平台租户 0")

	// 读视图：ent 行 → DTO 全量回读（payload 逐字节、枚举回 proto 成员）
	dto, err := repo.Get(ctx, &biV1.GetBiDashboardRequest{
		QueryBy: &biV1.GetBiDashboardRequest_Id{Id: row.ID},
	})
	require.NoError(t, err)
	require.Equal(t, row.ID, dto.GetId())
	require.Equal(t, "销售分析看板", dto.GetName())
	require.Equal(t, biV1.BiDashboardKind_SCREEN, dto.GetKind(),
		"实体枚举应经 converter 回映射为 proto 成员，不能落零值 DASHBOARD")
	require.Equal(t, designerPayloadJSON, dto.GetPayload(), "读视图的 payload 必须与写入原文逐字节相等")
	require.Equal(t, uint32(1), dto.GetChartCount())
	require.Equal(t, "ocean-dark", dto.GetThemeKey())
	require.True(t, dto.GetIsPublic())
	require.False(t, dto.GetIsBuiltIn())
	require.Equal(t, int64(1001), dto.GetCreatedBy())
	require.Equal(t, uint32(1), dto.GetVersion(), "乐观锁版本号必须出现在读视图里：设计器保存时要回传它")

	// 未携带创建人的创建（系统播种路径）：不写出不存在的操作人
	require.NoError(t, repo.Create(ctx, &biV1.CreateBiDashboardRequest{
		Data: newBiDashboardPayload("seeded-no-op"),
	}))
	row2 := mustGetBiDashboardRow(t, entClient.Client(), ctx, "seeded-no-op")
	require.Nil(t, row2.CreatedBy, "未携带创建人时不该写出一个不存在的操作人")

	// 显式带 version 的创建（导入/恢复场景）：版本号按载荷落，而不是被默认值吞掉
	payloadV5 := newBiDashboardPayload("imported-v5")
	payloadV5.Version = trans.Ptr(uint32(5))
	require.NoError(t, repo.Create(ctx, &biV1.CreateBiDashboardRequest{Data: payloadV5}))
	require.Equal(t, uint32(5), *mustGetBiDashboardRow(t, entClient.Client(), ctx, "imported-v5").Version,
		"显式 version 应按载荷落库")

	// 入参守卫
	require.Error(t, repo.Create(ctx, nil), "nil 请求应返回 BadRequest")
	require.Error(t, repo.Create(ctx, &biV1.CreateBiDashboardRequest{}), "nil Data 应返回 BadRequest")

	// 唯一索引 (code, tenant_id)：同租户重复编码 → 约束错误翻译成 BadRequest，
	// 让页面报"编码已存在"而不是 500
	require.Error(t, repo.Create(ctx, &biV1.CreateBiDashboardRequest{
		Data: newBiDashboardPayload("sales-monthly"),
	}), "同租户重复编码应被唯一索引挡下")
	cnt, err := entClient.Client().BiDashboard.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 3, cnt, "失败的创建不应增加行数")
}

// TestBiDashboardRepoSqlite_GetIsExist 验证 Get 命中/未命中与 nil 守卫、
// IsExist 的真假两侧。
func TestBiDashboardRepoSqlite_GetIsExist(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewBiDashboardRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	require.NoError(t, repo.Create(ctx, &biV1.CreateBiDashboardRequest{
		Data: newBiDashboardPayload("get-probe"),
	}))
	row := mustGetBiDashboardRow(t, entClient.Client(), ctx, "get-probe")

	// 字段回读已在 Create 用例覆盖，这里钉"带 id 的请求能取到"
	dto, err := repo.Get(ctx, &biV1.GetBiDashboardRequest{
		QueryBy: &biV1.GetBiDashboardRequest_Id{Id: row.ID},
	})
	require.NoError(t, err, "按存在 ID 查询应命中")
	require.Equal(t, "get-probe", dto.GetCode())

	// 未命中
	_, err = repo.Get(ctx, &biV1.GetBiDashboardRequest{
		QueryBy: &biV1.GetBiDashboardRequest_Id{Id: 9999999},
	})
	require.Error(t, err, "不存在的 ID 应返回错误")

	// nil 请求
	_, err = repo.Get(ctx, nil)
	require.Error(t, err, "nil 请求应返回 BadRequest")

	// IsExist
	exist, err := repo.IsExist(ctx, row.ID)
	require.NoError(t, err)
	require.True(t, exist, "已写入的仪表盘应判定为存在")
	exist, err = repo.IsExist(ctx, 9999999)
	require.NoError(t, err)
	require.False(t, exist, "不存在的仪表盘应判定为不存在")
}

// TestBiDashboardRepoSqlite_List 验证分页 total 与 name 的 contains 过滤；
// Count 与 List 的 total 同源；nil 请求返回 BadRequest。
func TestBiDashboardRepoSqlite_List(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewBiDashboardRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	payloadA := newBiDashboardPayload("list-alpha")
	payloadA.Name = trans.Ptr("marker-看板-alpha")
	require.NoError(t, repo.Create(ctx, &biV1.CreateBiDashboardRequest{Data: payloadA}))

	payloadB := newBiDashboardPayload("list-beta")
	payloadB.Name = trans.Ptr("大屏-beta")
	payloadB.Kind = biV1.BiDashboardKind_TEMPLATE.Enum()
	require.NoError(t, repo.Create(ctx, &biV1.CreateBiDashboardRequest{Data: payloadB}))
	rowB := mustGetBiDashboardRow(t, entClient.Client(), ctx, "list-beta")

	// 无过滤：total 与行数一致，kind 枚举在列表读视图同样要回 proto 成员
	all, err := repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, int64(2), all.GetTotal(), "无过滤时 total 应统计全部行")
	require.Len(t, all.GetItems(), 2)
	byID := map[int64]*biV1.BiDashboard{}
	for _, it := range all.GetItems() {
		byID[it.GetId()] = it
	}
	require.Equal(t, biV1.BiDashboardKind_TEMPLATE, byID[rowB.ID].GetKind(),
		"列表与详情共用读映射：模板库筛选靠 kind，不能落零值")

	// contains 过滤：只统计命中行（全仓铁律：搜索一律 contains）
	filtered, err := repo.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_Query{
			Query: `{"name__contains":"marker"}`,
		},
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), filtered.GetTotal(), "contains 过滤的 total 应只统计命中行")
	require.Len(t, filtered.GetItems(), 1)
	require.Equal(t, "list-alpha", filtered.GetItems()[0].GetCode())

	// nil 请求
	_, err = repo.List(ctx, nil)
	require.Error(t, err, "nil 分页请求应返回 BadRequest")

	// Count 与 List total 同源
	cnt, err := repo.Count(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, int64(2), cnt, "Count 应与无过滤 List 的 total 一致")
}

// TestBiDashboardRepoSqlite_UpdateCAS 验证乐观锁主路径：
// 回传当前 version 才允许覆盖（命中的行版本 +1、字段更新、updated_by 盖章）；
// stale 版本必须回 Conflict（不能静默覆盖别人刚保存的成果）；行已删回 NotFound；
// nil/id=0 守卫齐全。
func TestBiDashboardRepoSqlite_UpdateCAS(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewBiDashboardRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	require.NoError(t, repo.Create(ctx, &biV1.CreateBiDashboardRequest{
		Data: newBiDashboardPayload("cas-dash"),
	}))
	row := mustGetBiDashboardRow(t, entClient.Client(), ctx, "cas-dash")
	require.Equal(t, uint32(1), *row.Version)

	// 命中 CAS：设计器整份覆盖保存（新 payload + 新名称），版本 1 → 2
	updatedPayload := `{"version":"1.1","theme":{"key":"light"},"charts":[]}`
	require.NoError(t, repo.Update(ctx, &biV1.UpdateBiDashboardRequest{
		Id: row.ID,
		Data: &biV1.BiDashboard{
			Name:      trans.Ptr("改名后的看板"),
			Payload:   trans.Ptr(updatedPayload),
			Kind:      biV1.BiDashboardKind_DASHBOARD.Enum(),
			Version:   trans.Ptr(uint32(1)),
			UpdatedBy: trans.Ptr(int64(2001)),
		},
	}), "回传当前版本 1 的 CAS 更新应命中")

	row = mustGetBiDashboardRow(t, entClient.Client(), ctx, "cas-dash")
	require.Equal(t, uint32(2), *row.Version, "命中 CAS 后版本号应 +1（1 → 2）")
	require.Equal(t, "改名后的看板", *row.Name, "CAS 命中应应用新名称")
	require.NotNil(t, row.Payload)
	require.Equal(t, updatedPayload, *row.Payload, "CAS 命中应整份覆盖 payload")
	require.Equal(t, bidashboard.KindDASHBOARD, *row.Kind, "CAS 命中应更新 kind")
	require.NotNil(t, row.UpdatedBy)
	require.Equal(t, int64(2001), *row.UpdatedBy, "operatorID 应落 updated_by")
	require.NotNil(t, row.UpdatedAt, "更新应写入 updated_at")

	// stale 版本：拿着旧版本 1 再来保存（模拟另一个标签页先保存过）
	err := repo.Update(ctx, &biV1.UpdateBiDashboardRequest{
		Id: row.ID,
		Data: &biV1.BiDashboard{
			Name:    trans.Ptr("stale-writer"),
			Version: trans.Ptr(uint32(1)),
		},
	})
	require.Error(t, err, "stale 版本必须被拒")
	require.Contains(t, err.Error(), "modified by another session",
		"冲突提示要能被用户认出是「别人改过、请刷新重试」，这是设计器保存失败时唯一的线索")
	// 拒绝后数据保持 CAS 命中后的样子
	row = mustGetBiDashboardRow(t, entClient.Client(), ctx, "cas-dash")
	require.Equal(t, "改名后的看板", *row.Name, "stale 写被拒后不得覆盖已保存内容")
	require.Equal(t, uint32(2), *row.Version, "被拒的写不得推进版本号")

	// 行不存在 + 带 version：NotFound（与版本冲突可区分，删过的行不用引导用户刷新）
	err = repo.Update(ctx, &biV1.UpdateBiDashboardRequest{
		Id:   9999999,
		Data: &biV1.BiDashboard{Version: trans.Ptr(uint32(1))},
	})
	require.Error(t, err, "不存在的 ID 的 CAS 更新应返回 NotFound")
	require.Contains(t, err.Error(), "not found")

	// 命中路径再走一轮：版本 2 → 3，确认乐观锁可持续推进而不是一次性
	require.NoError(t, repo.Update(ctx, &biV1.UpdateBiDashboardRequest{
		Id:   row.ID,
		Data: &biV1.BiDashboard{Version: trans.Ptr(uint32(2)), IsPublic: trans.Ptr(false)},
	}))
	row = mustGetBiDashboardRow(t, entClient.Client(), ctx, "cas-dash")
	require.Equal(t, uint32(3), *row.Version, "版本 2 命中后应推进到 3")
	require.False(t, *row.IsPublic)

	// 入参守卫
	require.Error(t, repo.Update(ctx, nil), "nil 请求应返回 BadRequest")
	require.Error(t, repo.Update(ctx, &biV1.UpdateBiDashboardRequest{Id: row.ID}), "nil Data 应返回 BadRequest")
	require.Error(t, repo.Update(ctx, &biV1.UpdateBiDashboardRequest{
		Id:   0,
		Data: &biV1.BiDashboard{},
	}), "id=0 应返回 BadRequest")
}

// TestBiDashboardRepoSqlite_UpdatePlain 验证不带 version 的普通更新路径：
// 列表页轻量改字段不走 CAS，但版本号照样 +1（让持有旧版本的编辑器下次保存能发现冲突）；
// 掩码内字段更新、掩码外保持原值；不存在 ID 静默无效果。
func TestBiDashboardRepoSqlite_UpdatePlain(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewBiDashboardRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	require.NoError(t, repo.Create(ctx, &biV1.CreateBiDashboardRequest{
		Data: newBiDashboardPayload("plain-dash"),
	}))
	row := mustGetBiDashboardRow(t, entClient.Client(), ctx, "plain-dash")

	// 无掩码整份更新：Data 里显式给的字段落库，version 1 → 2
	require.NoError(t, repo.Update(ctx, &biV1.UpdateBiDashboardRequest{
		Id: row.ID,
		Data: &biV1.BiDashboard{
			Name:      trans.Ptr("无掩码改名"),
			Remark:    trans.Ptr("列表页轻量编辑"),
			IsEnabled: trans.Ptr(false),
			UpdatedBy: trans.Ptr(int64(3001)),
		},
	}), "不带 version 的普通更新应成功")

	row = mustGetBiDashboardRow(t, entClient.Client(), ctx, "plain-dash")
	require.Equal(t, "无掩码改名", *row.Name)
	require.Equal(t, "列表页轻量编辑", *row.Remark)
	require.False(t, *row.IsEnabled, "is_enabled=false 也要如实更新")
	require.Equal(t, uint32(2), *row.Version, "普通更新不校验版本，但版本号照样 +1：挡住旧编辑器的下一次 stale 保存")
	require.NotNil(t, row.UpdatedBy)
	require.Equal(t, int64(3001), *row.UpdatedBy)
	require.Equal(t, bidashboard.KindSCREEN, *row.Kind, "Data 未给 kind 时不得被改走")

	// 带掩码：掩码内 remark 更新，掩码外的 name 保持原值；版本继续推进
	require.NoError(t, repo.Update(ctx, &biV1.UpdateBiDashboardRequest{
		Id:         row.ID,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"remark"}},
		Data: &biV1.BiDashboard{
			Remark: trans.Ptr("掩码内改动"),
			Name:   trans.Ptr("掩码外不该生效"),
		},
	}))
	row = mustGetBiDashboardRow(t, entClient.Client(), ctx, "plain-dash")
	require.Equal(t, "掩码内改动", *row.Remark, "掩码内字段应更新")
	require.Equal(t, "无掩码改名", *row.Name, "掩码外字段应保持原值")
	require.Equal(t, uint32(3), *row.Version, "掩码内轻量更新同样推进版本号")

	// 不存在 ID：普通更新静默无效果（无 allowMissing 时不该凭空造行）
	require.NoError(t, repo.Update(ctx, &biV1.UpdateBiDashboardRequest{
		Id:   987654,
		Data: &biV1.BiDashboard{Name: trans.Ptr("ghost")},
	}), "对不存在 ID 的普通更新应静默无效果")
	cnt, err := entClient.Client().BiDashboard.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, cnt, "对不存在 ID 的更新不应新增行")
}

// TestBiDashboardRepoSqlite_Delete 验证删除守卫：普通行可删、内置行禁删、
// 不存在的 ID 幂等成功、nil 请求 BadRequest。
func TestBiDashboardRepoSqlite_Delete(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewBiDashboardRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	require.NoError(t, repo.Create(ctx, &biV1.CreateBiDashboardRequest{
		Data: newBiDashboardPayload("del-normal"),
	}))
	rowNormal := mustGetBiDashboardRow(t, entClient.Client(), ctx, "del-normal")

	payloadBuiltin := newBiDashboardPayload("del-builtin")
	payloadBuiltin.IsBuiltIn = trans.Ptr(true)
	require.NoError(t, repo.Create(ctx, &biV1.CreateBiDashboardRequest{Data: payloadBuiltin}))
	rowBuiltin := mustGetBiDashboardRow(t, entClient.Client(), ctx, "del-builtin")

	// 内置行禁删：内置仪表盘是产品能力的一部分，删了采用者升级后就没了
	err := repo.Delete(ctx, &biV1.DeleteBiDashboardRequest{
		QueryBy: &biV1.DeleteBiDashboardRequest_Id{Id: rowBuiltin.ID},
	})
	require.Error(t, err, "内置仪表盘应被禁删守卫挡下")
	require.Contains(t, err.Error(), "built-in")

	// 普通行删除
	require.NoError(t, repo.Delete(ctx, &biV1.DeleteBiDashboardRequest{
		QueryBy: &biV1.DeleteBiDashboardRequest_Id{Id: rowNormal.ID},
	}), "普通行删除应成功")
	cnt, err := entClient.Client().BiDashboard.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, cnt, "只该删掉普通行")

	// 幂等删除：目标不存在视为已删除（批量清理与重试场景依赖这个语义）
	require.NoError(t, repo.Delete(ctx, &biV1.DeleteBiDashboardRequest{
		QueryBy: &biV1.DeleteBiDashboardRequest_Id{Id: 9999999},
	}), "删除不存在的 ID 应幂等成功")

	// nil 请求
	require.Error(t, repo.Delete(ctx, nil), "nil 请求应返回 BadRequest")
}

// TestBiDashboardRepoSqlite_SeedDefaults 验证按编码缺一补一的播种语义：
// 编码不存在才创建、已存在跳过不覆盖、空编码/nil 条目跳过；重复调用不产生重复行。
func TestBiDashboardRepoSqlite_SeedDefaults(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewBiDashboardRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	defaults := []*biV1.BiDashboard{
		newBiDashboardPayload("seed-a"),
		func(d *biV1.BiDashboard) *biV1.BiDashboard {
			d.Code = trans.Ptr("seed-b")
			d.Name = trans.Ptr("内置大屏")
			return d
		}(newBiDashboardPayload("seed-b")),
		nil,                   // nil 条目跳过
		{Code: trans.Ptr("")}, // 空编码跳过：编码是播种幂等的键，没有键就没法判重
	}
	require.NoError(t, repo.SeedDefaults(ctx, defaults), "首轮播种应成功")

	// 模拟内置行被管理员改过名（或上轮播种后有人维护过）
	_, err := entClient.Client().BiDashboard.Update().
		Where(bidashboard.CodeEQ("seed-a")).
		SetName("管理员改过名").
		Save(ctx)
	require.NoError(t, err)

	require.NoError(t, repo.SeedDefaults(ctx, defaults), "二次播种应成功（已存在的跳过）")

	rows, err := entClient.Client().BiDashboard.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 2, "重复播种不得产生重复行")
	byCode := map[string]*ent.BiDashboard{}
	for _, r := range rows {
		byCode[*r.Code] = r
	}
	require.Equal(t, "内置大屏", *byCode["seed-b"].Name, "内置行应按默认值落库")
	require.Equal(t, "管理员改过名", *byCode["seed-a"].Name,
		"已存在的编码必须跳过不覆盖：播种无权动管理员维护过的行")

	// 补种新编码：只有缺失的进来
	defaults = append(defaults, newBiDashboardPayload("seed-c"))
	require.NoError(t, repo.SeedDefaults(ctx, defaults))
	cnt, err := entClient.Client().BiDashboard.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 3, cnt, "第三轮播种只应新增 seed-c 一行")
}
