package data

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/plan"
	"go-wind-admin/app/admin/service/internal/data/ent/planmodule"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newPlanModuleRepoSqlite 在给定 enttest client 上白盒构造 PlanModuleRepo，
// 逐字段复刻 NewPlanModuleRepo 的 mapper/converter 初始化，再调用 init()。
func newPlanModuleRepoSqlite(t *testing.T, entClient *entCrud.EntClient[*ent.Client]) *PlanModuleRepo {
	t.Helper()
	repo := &PlanModuleRepo{
		entClient: entClient,
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		mapper:    mapper.NewCopierMapper[identityV1.PlanModule, ent.PlanModule](),
		moduleConv: mapper.NewEnumTypeConverter[identityV1.Module, planmodule.Module](
			identityV1.Module_name, identityV1.Module_value,
		),
	}
	repo.init()
	return repo
}

// TestPlanModuleRepoSqlite_Create 带父 plan 创建白名单项，
// 直查断言：行落库、(plan_module → plan) 外键真实落库到请求指定的父行。
// 历史上这里守卫倒置导致 plan_id 永远写空，本用例为其回归测试。
func TestPlanModuleRepoSqlite_Create(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newPlanModuleRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	parent, err := entClient.Client().Plan.Create().
		SetNillableName(trans.Ptr("sqlite_pm_create_plan")).
		Save(ctx)
	require.NoError(t, err, "直建父 plan 应成功")

	err = repo.Create(ctx, &identityV1.CreatePlanModuleRequest{
		Data: &identityV1.PlanModule{
			PlanId: &parent.ID,
			Module: identityV1.Module_DASHBOARD.Enum(),
		},
	})
	require.NoError(t, err, "repo.Create 应写入 SQLite 成功")

	rows, err := entClient.Client().PlanModule.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1, "SQLite 中应有 1 条 plan_module 记录")
	require.NotNil(t, rows[0].Module, "module 枚举应经 converter 落库")
	require.Equal(t, planmodule.ModuleDashboard, *rows[0].Module, "module 枚举应落为 DASHBOARD")

	// 外键回归断言：白名单项必须挂在请求指定的父套餐上
	linked, err := entClient.Client().PlanModule.Query().
		Where(planmodule.HasPlanWith(plan.IDEQ(parent.ID))).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, linked, "plan_module 的 plan 外键应指向请求的父套餐，而非 NULL")
}

// TestPlanModuleRepoSqlite_ListFilter 验证 List 的等值过滤（module 列）与
// 列表路径上 PlanId 从父套餐边正确回填。
func TestPlanModuleRepoSqlite_ListFilter(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newPlanModuleRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	parent, err := entClient.Client().Plan.Create().
		SetNillableName(trans.Ptr("sqlite_pm_list_plan")).
		Save(ctx)
	require.NoError(t, err)

	require.NoError(t, repo.Create(ctx, &identityV1.CreatePlanModuleRequest{
		Data: &identityV1.PlanModule{
			PlanId: &parent.ID,
			Module: identityV1.Module_DASHBOARD.Enum(),
		},
	}))
	require.NoError(t, repo.Create(ctx, &identityV1.CreatePlanModuleRequest{
		Data: &identityV1.PlanModule{
			PlanId: &parent.ID,
			Module: identityV1.Module_SYSTEM.Enum(),
		},
	}))

	// 等值过滤：module = DASHBOARD 只命中第一行
	filtered, err := repo.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_FilterExpr{
			FilterExpr: &paginationV1.FilterExpr{
				Type: paginationV1.ExprType_AND,
				Conditions: []*paginationV1.FilterCondition{
					{
						Field:      "module",
						Op:         paginationV1.Operator_EQ,
						ValueOneof: &paginationV1.FilterCondition_Value{Value: "DASHBOARD"},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), filtered.Total, "等值过滤后 Total 应为 1")
	require.Len(t, filtered.Items, 1, "等值过滤后只应返回 1 行")
	require.NotNil(t, filtered.Items[0].Module, "命中行的 module 应被映射回 DTO")
	require.Equal(t, identityV1.Module_DASHBOARD, *filtered.Items[0].Module, "命中行应为 DASHBOARD 行")

	// 列表路径的 PlanId 回填：应等于父套餐 ID（边加载，而非 NULL）
	require.NotNil(t, filtered.Items[0].PlanId, "列表项应回填父套餐 ID")
	require.Equal(t, parent.ID, *filtered.Items[0].PlanId, "PlanId 应等于父套餐 ID")

	// 无过滤：两行全部返回，PlanId 均回填
	all, err := repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(2), all.Total)
	require.Len(t, all.Items, 2)
	for _, item := range all.Items {
		require.Equal(t, parent.ID, *item.PlanId, "所有条目的 PlanId 均应回填父套餐 ID")
	}
}

// TestPlanModuleRepoSqlite_ListModulesByPlanId 验证按套餐列出的模块白名单：
// 只返回该套餐的模块集合；0 与未知 ID 返回空。
func TestPlanModuleRepoSqlite_ListModulesByPlanId(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newPlanModuleRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	planA, err := entClient.Client().Plan.Create().
		SetNillableName(trans.Ptr("sqlite_pm_lmbpi_a")).
		Save(ctx)
	require.NoError(t, err)
	planB, err := entClient.Client().Plan.Create().
		SetNillableName(trans.Ptr("sqlite_pm_lmbpi_b")).
		Save(ctx)
	require.NoError(t, err)

	require.NoError(t, repo.Create(ctx, &identityV1.CreatePlanModuleRequest{
		Data: &identityV1.PlanModule{PlanId: &planA.ID, Module: identityV1.Module_DASHBOARD.Enum()},
	}))
	require.NoError(t, repo.Create(ctx, &identityV1.CreatePlanModuleRequest{
		Data: &identityV1.PlanModule{PlanId: &planB.ID, Module: identityV1.Module_OPM.Enum()},
	}))

	modulesA, err := repo.ListModulesByPlanId(ctx, planA.ID)
	require.NoError(t, err)
	require.Equal(t, map[identityV1.Module]bool{identityV1.Module_DASHBOARD: true}, modulesA, "套餐 A 的白名单应只含 DASHBOARD")

	modulesB, err := repo.ListModulesByPlanId(ctx, planB.ID)
	require.NoError(t, err)
	require.Equal(t, map[identityV1.Module]bool{identityV1.Module_OPM: true}, modulesB, "套餐 B 的白名单应只含 OPM")

	nilModules, err := repo.ListModulesByPlanId(ctx, 0)
	require.NoError(t, err)
	require.Nil(t, nilModules, "planId=0 应返回 nil")

	emptyModules, err := repo.ListModulesByPlanId(ctx, 987654)
	require.NoError(t, err)
	require.Empty(t, emptyModules, "无白名单的套餐应返回空集合")
}

// TestPlanModuleRepoSqlite_Get 验证 Get 命中/未命中。
func TestPlanModuleRepoSqlite_Get(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newPlanModuleRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	parent, err := entClient.Client().Plan.Create().
		SetNillableName(trans.Ptr("sqlite_pm_get_plan")).
		Save(ctx)
	require.NoError(t, err)

	require.NoError(t, repo.Create(ctx, &identityV1.CreatePlanModuleRequest{
		Data: &identityV1.PlanModule{PlanId: &parent.ID, Module: identityV1.Module_DASHBOARD.Enum()},
	}))
	rows, err := entClient.Client().PlanModule.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := rows[0].ID

	hit, err := repo.Get(ctx, &identityV1.GetPlanModuleRequest{
		QueryBy: &identityV1.GetPlanModuleRequest_Id{Id: createdID},
	})
	require.NoError(t, err, "按存在的 ID 查询应命中")
	require.Equal(t, createdID, hit.GetId())

	_, err = repo.Get(ctx, &identityV1.GetPlanModuleRequest{
		QueryBy: &identityV1.GetPlanModuleRequest_Id{Id: 9999999},
	})
	require.Error(t, err, "不存在的 ID 查询应返回错误")
}

// TestPlanModuleRepoSqlite_ModuleReadView 验证全部 10 个模块枚举值经 converter
// 落库后，在读路径（Get 按主键 / List）的 DTO 视图如实呈现；
// proto 零值（MODULE_UNSPECIFIED）经写路径守卫留空，读视图如实呈 nil。
//
// 枚举字段读视图机制注记：实体侧 module 为可空指针枚举列（*planmodule.Module），
// DTO 侧为可选指针字段。mapper 的枚举转换对（经 &srcType/&dstType 取址注册）
// 恰为指针↔指针形态的键，指针对字段能被 copier 直接转换赋值——与值型实体
// 枚举列（如 position.type、notification_channel.type）读侧被丢弃的情形不同。
// 本测试将该读视图行为钉死。
func TestPlanModuleRepoSqlite_ModuleReadView(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newPlanModuleRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	cases := []struct {
		protoModule identityV1.Module
		wantEnt     planmodule.Module
	}{
		{identityV1.Module_DASHBOARD, planmodule.ModuleDashboard},
		{identityV1.Module_OPM, planmodule.ModuleOpm},
		{identityV1.Module_SYSTEM, planmodule.ModuleSystem},
		{identityV1.Module_DICT, planmodule.ModuleDict},
		{identityV1.Module_TENANT, planmodule.ModuleTenant},
		{identityV1.Module_PERMISSION, planmodule.ModulePermission},
		{identityV1.Module_LOG, planmodule.ModuleLog},
		{identityV1.Module_INTERNAL_MESSAGE, planmodule.ModuleInternalMessage},
		{identityV1.Module_FILE, planmodule.ModuleFile},
		{identityV1.Module_TASK, planmodule.ModuleTask},
	}

	for _, c := range cases {
		// 每个用例独立父套餐，List 匹配依赖 PlanId 回填
		parent, err := entClient.Client().Plan.Create().
			SetNillableName(trans.Ptr(fmt.Sprintf("sqlite_pm_readview_plan_%d", c.protoModule))).
			Save(ctx)
		require.NoError(t, err, "直建父 plan 应成功")

		require.NoError(t, repo.Create(ctx, &identityV1.CreatePlanModuleRequest{
			Data: &identityV1.PlanModule{
				PlanId: &parent.ID,
				Module: c.protoModule.Enum(),
			},
		}), "模块 %v 创建应成功", c.protoModule)

		rows, err := entClient.Client().PlanModule.Query().
			Where(planmodule.HasPlanWith(plan.IDEQ(parent.ID))).
			All(ctx)
		require.NoError(t, err)
		require.Len(t, rows, 1, "按父套餐应反查到刚写入的行")
		require.Equal(t, c.wantEnt, *rows[0].Module, "模块 %v 应经转换器如实落库", c.protoModule)

		// 读路径一：Get 按主键命中后，DTO 视图应如实呈现模块枚举。
		got, err := repo.Get(ctx, &identityV1.GetPlanModuleRequest{
			QueryBy: &identityV1.GetPlanModuleRequest_Id{Id: rows[0].ID},
		})
		require.NoError(t, err, "按主键读取应命中")
		require.Equal(t, c.protoModule, got.GetModule(), "读视图应如实呈现模块 %v", c.protoModule)

		// 读路径二：List 应含该行且 PlanId 回填，DTO 视图如实呈现模块枚举。
		listed, err := repo.List(ctx, &paginationV1.PagingRequest{})
		require.NoError(t, err)
		var hit *identityV1.PlanModule
		for _, item := range listed.Items {
			if item.GetPlanId() == parent.ID {
				hit = item
				break
			}
		}
		require.NotNil(t, hit, "List 应包含父套餐 %d 下的白名单行", parent.ID)
		require.Equal(t, c.protoModule, hit.GetModule(), "List 读视图应如实呈现模块 %v", c.protoModule)
	}

	// proto 零值：写路径守卫（MODULE_UNSPECIFIED 跳过 SetNillableModule）→
	// 落库为 NULL，读视图如实呈 nil（GetModule() 返回零值 MODULE_UNSPECIFIED）。
	parent, err := entClient.Client().Plan.Create().
		SetNillableName(trans.Ptr("sqlite_pm_readview_plan_unspecified")).
		Save(ctx)
	require.NoError(t, err)
	require.NoError(t, repo.Create(ctx, &identityV1.CreatePlanModuleRequest{
		Data: &identityV1.PlanModule{
			PlanId: &parent.ID,
			Module: identityV1.Module_MODULE_UNSPECIFIED.Enum(),
		},
	}), "MODULE_UNSPECIFIED 创建应经守卫成功落空值")
	rows, err := entClient.Client().PlanModule.Query().
		Where(planmodule.HasPlanWith(plan.IDEQ(parent.ID))).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Nil(t, rows[0].Module, "MODULE_UNSPECIFIED 经守卫应落 NULL")
	got, err := repo.Get(ctx, &identityV1.GetPlanModuleRequest{
		QueryBy: &identityV1.GetPlanModuleRequest_Id{Id: rows[0].ID},
	})
	require.NoError(t, err)
	require.Equal(t, identityV1.Module_MODULE_UNSPECIFIED, got.GetModule(), "NULL 列读视图应呈零值")
}

// TestPlanModuleRepoSqlite_Update 验证 Update 掩码内字段（module）更新、
// 父套餐外键不受影响。
func TestPlanModuleRepoSqlite_Update(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newPlanModuleRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	parent, err := entClient.Client().Plan.Create().
		SetNillableName(trans.Ptr("sqlite_pm_update_plan")).
		Save(ctx)
	require.NoError(t, err)

	require.NoError(t, repo.Create(ctx, &identityV1.CreatePlanModuleRequest{
		Data: &identityV1.PlanModule{PlanId: &parent.ID, Module: identityV1.Module_DASHBOARD.Enum()},
	}))
	rows, err := entClient.Client().PlanModule.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := rows[0].ID

	err = repo.Update(ctx, &identityV1.UpdatePlanModuleRequest{
		Id:         createdID,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"module"}},
		Data: &identityV1.PlanModule{
			Module: identityV1.Module_OPM.Enum(),
		},
	})
	require.NoError(t, err, "更新 module 应成功")

	after, err := entClient.Client().PlanModule.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, after, 1)
	require.NotNil(t, after[0].Module)
	require.Equal(t, planmodule.ModuleOpm, *after[0].Module, "掩码内字段 module 应被更新为 OPM")

	linked, err := entClient.Client().PlanModule.Query().
		Where(planmodule.HasPlanWith(plan.IDEQ(parent.ID))).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, linked, "更新不应改变 plan_module 的父套餐外键")
}

// TestPlanModuleRepoSqlite_Delete 验证 Delete 后行数归零。
func TestPlanModuleRepoSqlite_Delete(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newPlanModuleRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	parent, err := entClient.Client().Plan.Create().
		SetNillableName(trans.Ptr("sqlite_pm_del_plan")).
		Save(ctx)
	require.NoError(t, err)

	require.NoError(t, repo.Create(ctx, &identityV1.CreatePlanModuleRequest{
		Data: &identityV1.PlanModule{PlanId: &parent.ID, Module: identityV1.Module_DASHBOARD.Enum()},
	}))
	rows, err := entClient.Client().PlanModule.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)

	require.NoError(t, repo.Delete(ctx, rows[0].ID), "Delete 应成功")
	cnt, err := entClient.Client().PlanModule.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, cnt, "Delete 后表内行数应为 0")
}
