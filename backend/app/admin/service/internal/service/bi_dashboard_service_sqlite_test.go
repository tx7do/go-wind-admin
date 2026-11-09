// BiDashboardService 的 SQLite 内存库测试（白盒，包内测试）。
//
// service 层是"操作人盖章 + 乐观锁透传"的薄层，测试盯的是盖章与透传不出错：
//   - Create/Update 盖 operator 章（created_by/updated_by 取令牌操作人，客户端伪造无效），
//     Update 把 req.Id 拷进 Data.Id、把 updated_by 追加进掩码——repo 的 CAS 冲突
//     （stale 版本回 Conflict）必须原样穿透到调用方，不能被 service 层吞成 500；
//   - Copy 派生语义：副本恒非内置、版本归 1、payload 整份随行、名称/编码缺省自动派生
//     且租户内唯一（复制两次不冲突）；
//   - init 播种：进程启动期把 constants.DefaultBiDashboards 按编码缺一补一，
//     重复构造不重复种。
package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	biV1 "go-wind-admin/api/gen/go/bi/service/v1"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/bidashboard"
	"go-wind-admin/app/admin/service/internal/data/enttest"
	"go-wind-admin/pkg/constants"
	"go-wind-admin/pkg/middleware/auth"
)

// newBiDashboardServiceForTest 白盒装配 BiDashboardService：log 换 NopLogger，
// repo 走 SQLite 内存库。不调 init()（播种单独测，避免种子行污染 CRUD 断言）。
func newBiDashboardServiceForTest(t *testing.T) (*BiDashboardService, *entCrud.EntClient[*ent.Client]) {
	t.Helper()
	entClient := enttest.NewEntClientForTest(t)
	return &BiDashboardService{
		log:             bLogger.NewHelper(bLogger.NopLogger()),
		biDashboardRepo: data.NewBiDashboardRepoForTest(entClient),
	}, entClient
}

// biSvcOpCtx 造带操作人的上下文。
func biSvcOpCtx(ctx context.Context, userID uint64) context.Context {
	return auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: userID})
}

// TestBiDashboardServiceSqlite_CreateStampsOperator 验证 Create 的操作人盖章：
// created_by 取令牌操作人（客户端伪造无效）；nil 载荷与缺操作人各自被挡。
func TestBiDashboardServiceSqlite_CreateStampsOperator(t *testing.T) {
	svc, entClient := newBiDashboardServiceForTest(t)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := biSvcOpCtx(ctx, 71)

	// 客户端伪造 created_by=999：以令牌操作人 71 为准
	_, err := svc.Create(opCtx, &biV1.CreateBiDashboardRequest{
		Data: &biV1.BiDashboard{
			Name:      trans.Ptr("服务层建盘"),
			Code:      trans.Ptr("svc-create"),
			CreatedBy: trans.Ptr(uint64(999)),
		},
	})
	require.NoError(t, err, "合法创建应成功")

	row, err := entClient.Client().BiDashboard.Query().
		Where(bidashboard.CodeEQ("svc-create")).
		Only(ctx)
	require.NoError(t, err)
	require.NotNil(t, row.CreatedBy)
	require.Equal(t, uint64(71), *row.CreatedBy, "created_by 应取令牌操作人，客户端伪造无效")

	// 入参守卫
	_, err = svc.Create(opCtx, nil)
	require.Error(t, err, "nil 请求应返回 BadRequest")
	_, err = svc.Create(opCtx, &biV1.CreateBiDashboardRequest{})
	require.Error(t, err, "nil Data 应返回 BadRequest")
	_, err = svc.Create(ctx, &biV1.CreateBiDashboardRequest{ // 无操作人上下文
		Data: &biV1.BiDashboard{Code: trans.Ptr("no-operator")},
	})
	require.Error(t, err, "缺操作人声明的创建应被拒绝")
}

// TestBiDashboardServiceSqlite_UpdateCASPassthrough 验证 Update 的盖章与乐观锁透传：
// Data.Id 从 req.Id 回填、updated_by 盖章并追加进掩码；stale 版本的 CAS 冲突
// 原样穿透为 Conflict（不能被 service 吞成 500，设计器要靠它提示刷新重试）。
func TestBiDashboardServiceSqlite_UpdateCASPassthrough(t *testing.T) {
	svc, entClient := newBiDashboardServiceForTest(t)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := biSvcOpCtx(ctx, 72)

	// 直接经 repo 播一行（版本 1），绕开 service 以隔离被测路径
	require.NoError(t, entClient.Client().BiDashboard.Create().
		SetName("原名称").
		SetCode("svc-cas").
		SetVersion(1).
		SetCreatedAt(time.Now()).
		Exec(ctx), "播种仪表盘应成功")
	src, err := entClient.Client().BiDashboard.Query().
		Where(bidashboard.CodeEQ("svc-cas")).
		Only(ctx)
	require.NoError(t, err)

	// 命中 CAS：客户端只带 name 掩码 + 当前版本 1；service 补 Data.Id 与 updated_by
	_, err = svc.Update(opCtx, &biV1.UpdateBiDashboardRequest{
		Id:         src.ID,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"name"}},
		Data: &biV1.BiDashboard{
			Name:    trans.Ptr("新名称"),
			Version: trans.Ptr(uint32(1)),
		},
	})
	require.NoError(t, err, "回传当前版本的更新应命中 CAS")

	row, err := entClient.Client().BiDashboard.Query().
		Where(bidashboard.CodeEQ("svc-cas")).
		Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "新名称", *row.Name, "掩码内 name 应更新")
	require.Equal(t, uint32(2), *row.Version, "命中 CAS 后版本号应 +1")
	require.NotNil(t, row.UpdatedBy)
	require.Equal(t, uint64(72), *row.UpdatedBy, "updated_by 应取令牌操作人")
	require.Equal(t, src.ID, row.ID, "Data.Id 应从 req.Id 回填，更新落在目标行")

	// stale 版本：拿旧版本 1 再来保存 → Conflict 原样穿透
	_, err = svc.Update(opCtx, &biV1.UpdateBiDashboardRequest{
		Id: src.ID,
		Data: &biV1.BiDashboard{
			Name:    trans.Ptr("stale-writer"),
			Version: trans.Ptr(uint32(1)),
		},
	})
	require.Error(t, err, "stale 版本必须被拒")
	require.Contains(t, err.Error(), "modified by another session",
		"CAS 冲突提示必须穿透到调用方：设计器保存失败时靠它提示刷新重试")

	// 入参守卫
	_, err = svc.Update(opCtx, nil)
	require.Error(t, err, "nil 请求应返回 BadRequest")
	_, err = svc.Update(opCtx, &biV1.UpdateBiDashboardRequest{Id: src.ID})
	require.Error(t, err, "nil Data 应返回 BadRequest")
	_, err = svc.Update(ctx, &biV1.UpdateBiDashboardRequest{
		Id:   src.ID,
		Data: &biV1.BiDashboard{},
	})
	require.Error(t, err, "缺操作人声明的更新应被拒绝")
}

// TestBiDashboardServiceSqlite_ListGetDeletePassthrough 验证 List/Get/Delete 透传：
// 列表 contains 过滤、Get 命中、内置行禁删守卫穿透、普通行删除成功。
func TestBiDashboardServiceSqlite_ListGetDeletePassthrough(t *testing.T) {
	svc, entClient := newBiDashboardServiceForTest(t)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := biSvcOpCtx(ctx, 73)

	_, err := svc.Create(opCtx, &biV1.CreateBiDashboardRequest{
		Data: &biV1.BiDashboard{Name: trans.Ptr("marker-看板"), Code: trans.Ptr("pass-a")},
	})
	require.NoError(t, err)
	_, err = svc.Create(opCtx, &biV1.CreateBiDashboardRequest{
		Data: &biV1.BiDashboard{Name: trans.Ptr("普通大屏"), Code: trans.Ptr("pass-b")},
	})
	require.NoError(t, err)
	// 内置行（播种同款形态）：删除守卫的靶子
	require.NoError(t, entClient.Client().BiDashboard.Create().
		SetName("内置模板").
		SetCode("pass-builtin").
		SetIsBuiltIn(true).
		SetCreatedAt(time.Now()).
		Exec(ctx))

	list, err := svc.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_Query{
			Query: `{"name__contains":"marker"}`,
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), list.GetTotal(), "contains 过滤应只命中 marker 行")
	require.Equal(t, "pass-a", list.GetItems()[0].GetCode())

	rowA, err := entClient.Client().BiDashboard.Query().
		Where(bidashboard.CodeEQ("pass-a")).
		Only(ctx)
	require.NoError(t, err)
	got, err := svc.Get(ctx, &biV1.GetBiDashboardRequest{
		QueryBy: &biV1.GetBiDashboardRequest_Id{Id: rowA.ID},
	})
	require.NoError(t, err, "按 id Get 应命中")
	require.Equal(t, "pass-a", got.GetCode())

	// 内置行禁删（repo 守卫穿透）
	builtinRow, err := entClient.Client().BiDashboard.Query().
		Where(bidashboard.CodeEQ("pass-builtin")).
		Only(ctx)
	require.NoError(t, err)
	_, err = svc.Delete(ctx, &biV1.DeleteBiDashboardRequest{
		QueryBy: &biV1.DeleteBiDashboardRequest_Id{Id: builtinRow.ID},
	})
	require.Error(t, err, "内置行删除应被守卫拒绝")
	require.Contains(t, err.Error(), "built-in")

	// 普通行删除
	_, err = svc.Delete(ctx, &biV1.DeleteBiDashboardRequest{
		QueryBy: &biV1.DeleteBiDashboardRequest_Id{Id: rowA.ID},
	})
	require.NoError(t, err, "普通行删除应成功")
	cnt, err := entClient.Client().BiDashboard.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, cnt, "删除后应剩普通 B 行与内置行")
}

// TestBiDashboardServiceSqlite_CopyDerivesNewRow 验证 Copy 的派生语义：
// 副本整份继承 payload/kind 等展示字段；恒非内置、版本归 1、is_enabled 恒开；
// 名称/编码缺省自动派生且复制两次不冲突；显式名称/编码按请求生效。
func TestBiDashboardServiceSqlite_CopyDerivesNewRow(t *testing.T) {
	svc, entClient := newBiDashboardServiceForTest(t)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := biSvcOpCtx(ctx, 74)

	const sourcePayload = `{"schemaVersion":2,"name":"源看板","charts":[]}`
	require.NoError(t, entClient.Client().BiDashboard.Create().
		SetName("源看板").
		SetCode("src-dash").
		SetKind(bidashboard.KindSCREEN).
		SetPayload(sourcePayload).
		SetChartCount(1).
		SetThemeKey("ocean").
		SetIsBuiltIn(true).
		SetIsPublic(true).
		SetVersion(3).
		SetCreatedAt(time.Now()).
		Exec(ctx), "播种内置源看板应成功")
	src, err := entClient.Client().BiDashboard.Query().
		Where(bidashboard.CodeEQ("src-dash")).
		Only(ctx)
	require.NoError(t, err)

	// 缺省派生：内置源复制两次，副本编码带时间戳后缀互不冲突
	_, err = svc.Copy(opCtx, &biV1.CopyBiDashboardRequest{Id: src.ID})
	require.NoError(t, err, "第一次复制应成功")
	_, err = svc.Copy(opCtx, &biV1.CopyBiDashboardRequest{Id: src.ID})
	require.NoError(t, err, "第二次复制应成功（时间戳后缀保证编码不冲突）")

	copies, err := entClient.Client().BiDashboard.Query().
		Where(bidashboard.CodeNEQ("src-dash")).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, copies, 2, "复制两次应恰好两行副本")
	for _, c := range copies {
		require.Equal(t, "源看板-copy", *c.Name, "名称缺省应派生为「源名-copy」")
		require.True(t, strings.HasPrefix(*c.Code, "src-dash-copy-"), "编码缺省应派生为「源码-copy-时间戳」")
		require.NotNil(t, c.IsBuiltIn)
		require.False(t, *c.IsBuiltIn, "副本恒非内置：内置模板只读，派生出的才是租户资产")
		require.NotNil(t, c.IsEnabled)
		require.True(t, *c.IsEnabled, "副本应默认启用")
		require.Equal(t, sourcePayload, *c.Payload, "payload 应整份随行")
		require.Equal(t, bidashboard.KindSCREEN, *c.Kind, "kind 应随源")
		require.Equal(t, uint32(1), *c.Version, "副本版本应归 1：派生行从零开始计数")
		require.NotNil(t, c.CreatedBy)
		require.Equal(t, uint64(74), *c.CreatedBy, "副本创建人应为令牌操作人")
	}

	// 显式名称/编码：按请求生效
	_, err = svc.Copy(opCtx, &biV1.CopyBiDashboardRequest{
		Id:   src.ID,
		Name: trans.Ptr("定制副本"),
		Code: trans.Ptr("custom-copy"),
	})
	require.NoError(t, err)
	custom, err := entClient.Client().BiDashboard.Query().
		Where(bidashboard.CodeEQ("custom-copy")).
		Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "定制副本", *custom.Name)

	// 入参守卫与源缺失
	_, err = svc.Copy(opCtx, nil)
	require.Error(t, err, "nil 请求应返回 BadRequest")
	_, err = svc.Copy(opCtx, &biV1.CopyBiDashboardRequest{})
	require.Error(t, err, "id=0 应返回 BadRequest")
	_, err = svc.Copy(opCtx, &biV1.CopyBiDashboardRequest{Id: 987654})
	require.Error(t, err, "源不存在应透传 NotFound")
	_, err = svc.Copy(ctx, &biV1.CopyBiDashboardRequest{Id: src.ID})
	require.Error(t, err, "缺操作人声明的复制应被拒绝")
}

// TestBiDashboardServiceSqlite_InitSeedsBuiltIn 验证 init 播种：
// 启动期把 DefaultBiDashboards 按编码缺一补一、内置标记落库；重复播种不重复种。
func TestBiDashboardServiceSqlite_InitSeedsBuiltIn(t *testing.T) {
	svc, entClient := newBiDashboardServiceForTest(t)
	ctx := enttest.NewSystemContext(context.Background())
	require.NotEmpty(t, constants.DefaultBiDashboards, "内置 BI 模板常量不应为空")

	svc.init()

	rows, err := entClient.Client().BiDashboard.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, len(constants.DefaultBiDashboards), "首轮播种应落全部内置模板")
	for _, r := range rows {
		require.NotNil(t, r.IsBuiltIn)
		require.True(t, *r.IsBuiltIn, "播种行应带内置标记（禁删、可复制派生靠它）")
	}

	// 已被管理员改名的内置行不得被二次播种覆盖
	require.NoError(t, entClient.Client().BiDashboard.Update().
		Where(bidashboard.CodeEQ(*constants.DefaultBiDashboards[0].Code)).
		SetName("管理员改过名").
		Exec(ctx))
	svc.init()
	rows, err = entClient.Client().BiDashboard.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, len(constants.DefaultBiDashboards), "二次播种不得重复种")
	for _, r := range rows {
		require.Equal(t, "管理员改过名", *r.Name, "已存在的编码必须跳过不覆盖")
	}
}
