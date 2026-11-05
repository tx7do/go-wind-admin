package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	"github.com/tx7do/go-utils/trans"
	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/ent"
	entPlan "go-wind-admin/app/admin/service/internal/data/ent/plan"
	entPlanModule "go-wind-admin/app/admin/service/internal/data/ent/planmodule"
	"go-wind-admin/app/admin/service/internal/data/enttest"
	"go-wind-admin/pkg/middleware/auth"
)

// newPlanModuleServiceForTest 白盒构造 PlanModuleService，逐字段对齐
// NewPlanModuleService 的装配（log 用 NopLogger helper）。
func newPlanModuleServiceForTest(t *testing.T, entClient *entCrud.EntClient[*ent.Client]) *PlanModuleService {
	t.Helper()
	return &PlanModuleService{
		log:            bLogger.NewHelper(bLogger.NopLogger()),
		planModuleRepo: data.NewPlanModuleRepoForTest(entClient),
	}
}

// TestPlanModuleServiceSqlite_Create_AssociatesPlan 验证服务层创建套餐模块白名单项时，
// 请求携带的 planId 真实落库为指向父套餐的外键（历史上该条件曾写反导致关联永不落库），
// 且 List 路径应从 plan 边回填 PlanId。
func TestPlanModuleServiceSqlite_Create_AssociatesPlan(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newPlanModuleServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7, IsPlatformAdmin: trans.Ptr(true)})

	parent, err := entClient.Client().Plan.Create().
		SetName("父套餐-白名单关联").
		Save(ctx)
	require.NoError(t, err, "直建父 plan 应成功")

	_, err = svc.Create(opCtx, &identityV1.CreatePlanModuleRequest{
		Data: &identityV1.PlanModule{
			Module: identityV1.Module_DASHBOARD.Enum(),
			PlanId: &parent.ID,
		},
	})
	require.NoError(t, err, "服务层 Create 应成功")

	rows, err := entClient.Client().PlanModule.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1, "plan_module 表应落库 1 条记录")
	require.Equal(t, entPlanModule.ModuleDashboard, *rows[0].Module, "module 应经转换器落库为 DASHBOARD")
	require.NotNil(t, rows[0].CreatedBy, "created_by 应被服务层盖入操作人 ID")
	require.Equal(t, uint32(7), *rows[0].CreatedBy, "created_by 应等于令牌声明中的操作人 ID")

	// 外键回归断言：白名单行必须挂在请求指定的父套餐上，而非 NULL
	linked, err := entClient.Client().PlanModule.Query().
		Where(
			entPlanModule.HasPlanWith(entPlan.IDEQ(parent.ID)),
		).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, linked, "plan_module 的 plan 外键应指向父 plan")
}

// TestPlanModuleServiceSqlite_List_BackfillsPlanId 验证服务层 List 从 plan 边
// 回填 PlanId（边加载路径）。
func TestPlanModuleServiceSqlite_List_BackfillsPlanId(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newPlanModuleServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7, IsPlatformAdmin: trans.Ptr(true)})

	parent, err := entClient.Client().Plan.Create().
		SetName("父套餐-列表回填").
		Save(ctx)
	require.NoError(t, err)

	_, err = svc.Create(opCtx, &identityV1.CreatePlanModuleRequest{
		Data: &identityV1.PlanModule{
			Module: identityV1.Module_TASK.Enum(),
			PlanId: &parent.ID,
		},
	})
	require.NoError(t, err)

	listResp, err := svc.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(1), listResp.Total, "列表应统计全部 1 条")
	require.Len(t, listResp.Items, 1)
	require.NotNil(t, listResp.Items[0].PlanId, "列表项应从 plan 边回填 PlanId")
	require.Equal(t, parent.ID, *listResp.Items[0].PlanId, "回填的 PlanId 应等于父 plan 的 ID")
	require.Equal(t, identityV1.Module_TASK, listResp.Items[0].GetModule(), "module 应经转换器回读")
}

// TestPlanModuleServiceSqlite_Create_MissingOperatorRejected 缺少操作人声明时
// 服务层 Create 应直接拒绝，且不落库。
func TestPlanModuleServiceSqlite_Create_MissingOperatorRejected(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newPlanModuleServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	_, err := svc.Create(ctx, &identityV1.CreatePlanModuleRequest{
		Data: &identityV1.PlanModule{Module: identityV1.Module_LOG.Enum()},
	})
	require.Error(t, err, "缺少操作人声明时应返回 ErrMissingJwtToken")

	cnt, err := entClient.Client().PlanModule.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, cnt, "被拒绝的请求不应落库")
}

// TestPlanModuleServiceSqlite_Get 验证服务层 Get 按主键查询的命中与未命中。
func TestPlanModuleServiceSqlite_Get(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newPlanModuleServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7, IsPlatformAdmin: trans.Ptr(true)})

	parent, err := entClient.Client().Plan.Create().
		SetName("父套餐-Get路径").
		Save(ctx)
	require.NoError(t, err)

	_, err = svc.Create(opCtx, &identityV1.CreatePlanModuleRequest{
		Data: &identityV1.PlanModule{
			Module: identityV1.Module_FILE.Enum(),
			PlanId: &parent.ID,
		},
	})
	require.NoError(t, err)

	rows, err := entClient.Client().PlanModule.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := rows[0].ID

	got, err := svc.Get(ctx, &identityV1.GetPlanModuleRequest{
		QueryBy: &identityV1.GetPlanModuleRequest_Id{Id: createdID},
	})
	require.NoError(t, err, "按已存在主键查询应命中")
	require.Equal(t, identityV1.Module_FILE, got.GetModule(), "module 应经转换器回读")

	_, err = svc.Get(ctx, &identityV1.GetPlanModuleRequest{
		QueryBy: &identityV1.GetPlanModuleRequest_Id{Id: 9999999},
	})
	require.Error(t, err, "按不存在主键查询应返回错误")
}

// TestPlanModuleServiceSqlite_Update 验证服务层 Update 在单字段掩码下
// 只更新掩码内字段（module），其余保持原值。
func TestPlanModuleServiceSqlite_Update(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newPlanModuleServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7, IsPlatformAdmin: trans.Ptr(true)})

	parent, err := entClient.Client().Plan.Create().
		SetName("父套餐-更新路径").
		Save(ctx)
	require.NoError(t, err)

	_, err = svc.Create(opCtx, &identityV1.CreatePlanModuleRequest{
		Data: &identityV1.PlanModule{
			Module: identityV1.Module_DICT.Enum(),
			PlanId: &parent.ID,
		},
	})
	require.NoError(t, err)

	rows, err := entClient.Client().PlanModule.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := rows[0].ID

	_, err = svc.Update(opCtx, &identityV1.UpdatePlanModuleRequest{
		Id:         createdID,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"module"}},
		Data:       &identityV1.PlanModule{Module: identityV1.Module_OPM.Enum()},
	})
	require.NoError(t, err, "单字段掩码更新应成功")

	after, err := entClient.Client().PlanModule.Query().Only(ctx)
	require.NoError(t, err)
	require.Equal(t, entPlanModule.ModuleOpm, *after.Module, "掩码内字段 module 应被更新为 OPM")

	// plan_id 未在掩码内：外键应保持指向父 plan（FK 字段未导出，经边谓词断言）
	linked, err := entClient.Client().PlanModule.Query().
		Where(
			entPlanModule.HasPlanWith(entPlan.IDEQ(parent.ID)),
		).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, linked, "掩码外字段 plan_id 应保持指向父 plan")
}

// TestPlanModuleServiceSqlite_Delete 验证服务层 Delete 后表内计数归零。
func TestPlanModuleServiceSqlite_Delete(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newPlanModuleServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7, IsPlatformAdmin: trans.Ptr(true)})

	_, err := svc.Create(opCtx, &identityV1.CreatePlanModuleRequest{
		Data: &identityV1.PlanModule{Module: identityV1.Module_SYSTEM.Enum()},
	})
	require.NoError(t, err)

	rows, err := entClient.Client().PlanModule.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := rows[0].ID

	_, err = svc.Delete(auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7, IsPlatformAdmin: trans.Ptr(true)}), &identityV1.DeletePlanModuleRequest{
		QueryBy: &identityV1.DeletePlanModuleRequest_Id{Id: createdID},
	})
	require.NoError(t, err, "删除已存在记录应成功")

	cnt, err := entClient.Client().PlanModule.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, cnt, "删除后 plan_module 表计数应归零")
}
