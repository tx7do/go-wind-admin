package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/ent"
	entPlan "go-wind-admin/app/admin/service/internal/data/ent/plan"
	entPlanQuota "go-wind-admin/app/admin/service/internal/data/ent/planquota"
	"go-wind-admin/app/admin/service/internal/data/enttest"
	"go-wind-admin/pkg/middleware/auth"
)

// newPlanQuotaServiceForTest 白盒构造 PlanQuotaService，逐字段对齐
// NewPlanQuotaService 的装配（log 用 NopLogger helper）。
func newPlanQuotaServiceForTest(t *testing.T, entClient *entCrud.EntClient[*ent.Client]) *PlanQuotaService {
	t.Helper()
	return &PlanQuotaService{
		log:           bLogger.NewHelper(bLogger.NopLogger()),
		planQuotaRepo: data.NewPlanQuotaRepoForTest(entClient),
	}
}

// TestPlanQuotaServiceSqlite_Create_AssociatesPlan 验证服务层创建套餐配额时，
// 请求携带的 planId 真实落库为指向父套餐的外键（历史上该条件曾写反导致
// 配额行的 plan_id 永远为 NULL），且配额类型枚举经转换器落库。
func TestPlanQuotaServiceSqlite_Create_AssociatesPlan(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newPlanQuotaServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7, IsPlatformAdmin: trans.Ptr(true)})

	parent, err := entClient.Client().Plan.Create().
		SetName("父套餐-配额关联").
		Save(ctx)
	require.NoError(t, err, "直建父 plan 应成功")

	_, err = svc.Create(opCtx, &identityV1.CreatePlanQuotaRequest{
		Data: &identityV1.PlanQuota{
			QuotaType:  identityV1.PlanQuota_USER_LIMIT.Enum(),
			QuotaValue: trans.Ptr(uint64(42)),
			PlanId:     &parent.ID,
		},
	})
	require.NoError(t, err, "服务层 Create 应成功")

	rows, err := entClient.Client().PlanQuota.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1, "plan_quota 表应落库 1 条记录")
	require.Equal(t, entPlanQuota.QuotaTypeUserLimit, *rows[0].QuotaType, "quota_type 应经转换器落库为 USER_LIMIT")
	require.Equal(t, uint64(42), *rows[0].QuotaValue, "quota_value 应按请求落库")
	require.NotNil(t, rows[0].CreatedBy, "created_by 应被服务层盖入操作人 ID")
	require.Equal(t, uint32(7), *rows[0].CreatedBy, "created_by 应等于令牌声明中的操作人 ID")

	// 外键回归断言：配额行必须挂在请求指定的父套餐上，而非 NULL
	linked, err := entClient.Client().PlanQuota.Query().
		Where(
			entPlanQuota.HasPlanWith(entPlan.IDEQ(parent.ID)),
		).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, linked, "plan_quota 的 plan 外键应指向父 plan")
}

// TestPlanQuotaServiceSqlite_List_BackfillsPlanId 验证服务层 List 从 plan 边
// 回填 PlanId，并把配额字段回读出来。
func TestPlanQuotaServiceSqlite_List_BackfillsPlanId(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newPlanQuotaServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7, IsPlatformAdmin: trans.Ptr(true)})

	parent, err := entClient.Client().Plan.Create().
		SetName("父套餐-配额列表").
		Save(ctx)
	require.NoError(t, err)

	_, err = svc.Create(opCtx, &identityV1.CreatePlanQuotaRequest{
		Data: &identityV1.PlanQuota{
			QuotaType:  identityV1.PlanQuota_STORAGE.Enum(),
			QuotaValue: trans.Ptr(uint64(1024)),
			PlanId:     &parent.ID,
		},
	})
	require.NoError(t, err)

	listResp, err := svc.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(1), listResp.Total, "列表应统计全部 1 条")
	require.Len(t, listResp.Items, 1)
	require.NotNil(t, listResp.Items[0].PlanId, "列表项应从 plan 边回填 PlanId")
	require.Equal(t, parent.ID, *listResp.Items[0].PlanId, "回填的 PlanId 应等于父 plan 的 ID")
	require.Equal(t, identityV1.PlanQuota_STORAGE, listResp.Items[0].GetQuotaType(), "quota_type 应经转换器回读")
}

// TestPlanQuotaServiceSqlite_Create_MissingOperatorRejected 缺少操作人声明时
// 服务层 Create 应直接拒绝，且不落库。
func TestPlanQuotaServiceSqlite_Create_MissingOperatorRejected(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newPlanQuotaServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	_, err := svc.Create(ctx, &identityV1.CreatePlanQuotaRequest{
		Data: &identityV1.PlanQuota{QuotaType: identityV1.PlanQuota_API_CALL.Enum()},
	})
	require.Error(t, err, "缺少操作人声明时应返回 ErrMissingJwtToken")

	cnt, err := entClient.Client().PlanQuota.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, cnt, "被拒绝的请求不应落库")
}

// TestPlanQuotaServiceSqlite_Get 验证服务层 Get 按主键查询的命中与未命中。
func TestPlanQuotaServiceSqlite_Get(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newPlanQuotaServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7, IsPlatformAdmin: trans.Ptr(true)})

	_, err := svc.Create(opCtx, &identityV1.CreatePlanQuotaRequest{
		Data: &identityV1.PlanQuota{
			QuotaType:  identityV1.PlanQuota_USER_LIMIT.Enum(),
			QuotaValue: trans.Ptr(uint64(3)),
		},
	})
	require.NoError(t, err)

	rows, err := entClient.Client().PlanQuota.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := rows[0].ID

	got, err := svc.Get(ctx, &identityV1.GetPlanQuotaRequest{
		QueryBy: &identityV1.GetPlanQuotaRequest_Id{Id: createdID},
	})
	require.NoError(t, err, "按已存在主键查询应命中")
	require.Equal(t, identityV1.PlanQuota_USER_LIMIT, got.GetQuotaType(), "quota_type 应经转换器回读")

	_, err = svc.Get(ctx, &identityV1.GetPlanQuotaRequest{
		QueryBy: &identityV1.GetPlanQuotaRequest_Id{Id: 9999999},
	})
	require.Error(t, err, "按不存在主键查询应返回错误")
}

// TestPlanQuotaServiceSqlite_Update 验证服务层 Update 在单字段掩码下
// 只更新掩码内字段（quota_value），其余保持原值。
func TestPlanQuotaServiceSqlite_Update(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newPlanQuotaServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7, IsPlatformAdmin: trans.Ptr(true)})

	_, err := svc.Create(opCtx, &identityV1.CreatePlanQuotaRequest{
		Data: &identityV1.PlanQuota{
			QuotaType:  identityV1.PlanQuota_STORAGE.Enum(),
			QuotaValue: trans.Ptr(uint64(11)),
		},
	})
	require.NoError(t, err)

	rows, err := entClient.Client().PlanQuota.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := rows[0].ID

	_, err = svc.Update(opCtx, &identityV1.UpdatePlanQuotaRequest{
		Id:         createdID,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"quota_value"}},
		Data:       &identityV1.PlanQuota{QuotaValue: trans.Ptr(uint64(99))},
	})
	require.NoError(t, err, "单字段掩码更新应成功")

	after, err := entClient.Client().PlanQuota.Query().Only(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(99), *after.QuotaValue, "掩码内字段 quota_value 应被更新")
	require.Equal(t, entPlanQuota.QuotaTypeStorage, *after.QuotaType, "掩码外字段 quota_type 应保持原值")
}

// TestPlanQuotaServiceSqlite_Delete 验证服务层 Delete 后表内计数归零。
func TestPlanQuotaServiceSqlite_Delete(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newPlanQuotaServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7, IsPlatformAdmin: trans.Ptr(true)})

	_, err := svc.Create(opCtx, &identityV1.CreatePlanQuotaRequest{
		Data: &identityV1.PlanQuota{
			QuotaType:  identityV1.PlanQuota_API_CALL.Enum(),
			QuotaValue: trans.Ptr(uint64(1)),
		},
	})
	require.NoError(t, err)

	rows, err := entClient.Client().PlanQuota.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := rows[0].ID

	_, err = svc.Delete(auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7, IsPlatformAdmin: trans.Ptr(true)}), &identityV1.DeletePlanQuotaRequest{
		QueryBy: &identityV1.DeletePlanQuotaRequest_Id{Id: createdID},
	})
	require.NoError(t, err, "删除已存在记录应成功")

	cnt, err := entClient.Client().PlanQuota.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, cnt, "删除后 plan_quota 表计数应归零")
}
