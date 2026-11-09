package service

import (
	"context"
	"fmt"
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
	"go-wind-admin/app/admin/service/internal/data/enttest"
	"go-wind-admin/pkg/middleware/auth"
)

// newPlanServiceForTest 白盒构造 PlanService，逐字段对齐 NewPlanService 的装配：
// log 用 NopLogger helper，planRepo 用 data.NewPlanRepoForTest（SQLite 内存库）。
func newPlanServiceForTest(t *testing.T, entClient *entCrud.EntClient[*ent.Client]) *PlanService {
	t.Helper()
	return &PlanService{
		log:      bLogger.NewHelper(bLogger.NopLogger()),
		planRepo: data.NewPlanRepoForTest(entClient),
	}
}

// TestPlanServiceSqlite_Create 通过服务层创建套餐，直查断言：
// 普通字段、两个枚举字段（经转换器）按请求落库，且服务层把操作人 ID 盖到 created_by。
func TestPlanServiceSqlite_Create(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newPlanServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7, IsPlatformAdmin: trans.Ptr(true)})

	_, err := svc.Create(opCtx, &identityV1.CreatePlanRequest{
		Data: &identityV1.Plan{
			Name:              trans.Ptr("服务层套餐-创建"),
			Version:           identityV1.Plan_FREE.Enum(),
			ExpiryPolicy:      identityV1.Plan_READONLY.Enum(),
			DataRetentionDays: trans.Ptr(uint32(30)),
			Description:       trans.Ptr("服务层创建的套餐描述"),
			Remark:            trans.Ptr("服务层创建的备注"),
		},
	})
	require.NoError(t, err, "服务层 Create 应成功")

	rows, err := entClient.Client().Plan.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1, "plan 表应落库 1 条记录")
	require.Equal(t, "服务层套餐-创建", *rows[0].Name, "name 应按请求落库")
	require.Equal(t, entPlan.VersionFree, *rows[0].Version, "version 应经转换器落库为 FREE")
	require.Equal(t, entPlan.ExpiryPolicyReadonly, *rows[0].ExpiryPolicy, "expiry_policy 应经转换器落库为 READONLY")
	require.Equal(t, uint32(30), *rows[0].DataRetentionDays, "data_retention_days 应按请求落库")
	require.Equal(t, "服务层创建的套餐描述", *rows[0].Description, "description 应按请求落库")
	require.Equal(t, "服务层创建的备注", *rows[0].Remark, "remark 应按请求落库")
	require.NotNil(t, rows[0].CreatedBy, "created_by 应被服务层盖入操作人 ID")
	require.Equal(t, uint64(7), *rows[0].CreatedBy, "created_by 应等于令牌声明中的操作人 ID")
}

// TestPlanServiceSqlite_CreateGuards 服务层 Create 的入参守卫：
// Data 为 nil 返回 BadRequest；缺少操作人声明返回 ErrMissingJwtToken。
func TestPlanServiceSqlite_CreateGuards(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newPlanServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	_, err := svc.Create(ctx, &identityV1.CreatePlanRequest{})
	require.Error(t, err, "Data 为 nil 应返回错误")
	require.NotPanics(t, func() {
		_, _ = svc.Create(ctx, nil)
	}, "nil 请求应被守卫拦下而不 panic")

	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7, IsPlatformAdmin: trans.Ptr(true)})
	_, err = svc.Create(opCtx, &identityV1.CreatePlanRequest{
		Data: &identityV1.Plan{Name: trans.Ptr("守卫套餐")},
	})
	require.NoError(t, err, "带操作人声明的最小创建载荷应成功")

	cnt, err := entClient.Client().Plan.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, cnt, "仅守卫用例中成功的那一条应落库")
}

// TestPlanServiceSqlite_List 验证服务层 List 的全量返回与
// contains 模糊搜索语义（仓规：搜索条件一律 contains）。
func TestPlanServiceSqlite_List(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newPlanServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7, IsPlatformAdmin: trans.Ptr(true)})

	for i, name := range []string{"MARKERALPHA 套餐", "MARKERBETA 套餐"} {
		_, err := svc.Create(opCtx, &identityV1.CreatePlanRequest{
			Data: &identityV1.Plan{Name: trans.Ptr(name), Version: identityV1.Plan_STANDARD.Enum()},
		})
		require.NoError(t, err, fmt.Sprintf("第 %d 条套餐创建应成功", i))
	}

	all, err := svc.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(2), all.Total, "无过滤时应统计全部 2 条")
	require.Len(t, all.Items, 2, "无过滤时应返回 2 条")

	filtered, err := svc.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_FilterExpr{
			FilterExpr: &paginationV1.FilterExpr{
				Type: paginationV1.ExprType_AND,
				Conditions: []*paginationV1.FilterCondition{
					{
						Field:      "name",
						Op:         paginationV1.Operator_CONTAINS,
						ValueOneof: &paginationV1.FilterCondition_Value{Value: "MARKERALPHA"},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), filtered.Total, "contains 过滤后 Total 应为 1")
	require.Len(t, filtered.Items, 1, "contains 过滤后只应返回 1 条")
	require.Contains(t, filtered.Items[0].GetName(), "MARKERALPHA", "命中行应是携带标记的那条")
}

// TestPlanServiceSqlite_Get 验证服务层 Get 按主键查询的命中与未命中。
func TestPlanServiceSqlite_Get(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newPlanServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7, IsPlatformAdmin: trans.Ptr(true)})

	_, err := svc.Create(opCtx, &identityV1.CreatePlanRequest{
		Data: &identityV1.Plan{Name: trans.Ptr("服务层查询套餐"), Version: identityV1.Plan_ENTERPRISE.Enum()},
	})
	require.NoError(t, err)

	rows, err := entClient.Client().Plan.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := rows[0].ID

	got, err := svc.Get(ctx, &identityV1.GetPlanRequest{
		QueryBy: &identityV1.GetPlanRequest_Id{Id: createdID},
	})
	require.NoError(t, err, "按已存在主键查询应命中")
	require.Equal(t, "服务层查询套餐", got.GetName(), "命中记录的 name 应与写入一致")
	require.Equal(t, identityV1.Plan_ENTERPRISE, got.GetVersion(), "命中记录的 version 应经转换器回读")

	_, err = svc.Get(ctx, &identityV1.GetPlanRequest{
		QueryBy: &identityV1.GetPlanRequest_Id{Id: 9999999},
	})
	require.Error(t, err, "按不存在主键查询应返回错误")
}

// TestPlanServiceSqlite_Update 验证服务层 Update 在单字段 updateMask 下
// 只更新掩码内字段；并断言服务层把 updated_by 追加进掩码、盖入操作人 ID。
func TestPlanServiceSqlite_Update(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newPlanServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7, IsPlatformAdmin: trans.Ptr(true)})

	_, err := svc.Create(opCtx, &identityV1.CreatePlanRequest{
		Data: &identityV1.Plan{
			Name:        trans.Ptr("更新前套餐名"),
			Version:     identityV1.Plan_STANDARD.Enum(),
			Description: trans.Ptr("更新前描述"),
		},
	})
	require.NoError(t, err)

	rows, err := entClient.Client().Plan.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := rows[0].ID

	_, err = svc.Update(opCtx, &identityV1.UpdatePlanRequest{
		Id:         createdID,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"name"}},
		Data:       &identityV1.Plan{Name: trans.Ptr("更新后套餐名")},
	})
	require.NoError(t, err, "单字段掩码更新应成功")

	after, err := entClient.Client().Plan.Query().Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "更新后套餐名", *after.Name, "掩码内字段 name 应被更新")
	require.Equal(t, entPlan.VersionStandard, *after.Version, "掩码外字段 version 应保持原值")
	require.Equal(t, "更新前描述", *after.Description, "掩码外字段 description 应保持原值")
	require.NotNil(t, after.UpdatedBy, "服务层应把操作人 ID 盖入 updated_by")
	require.Equal(t, uint64(7), *after.UpdatedBy, "updated_by 应等于令牌声明中的操作人 ID")
}

// TestPlanServiceSqlite_Delete 验证服务层 Delete 后表内计数归零。
func TestPlanServiceSqlite_Delete(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newPlanServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7, IsPlatformAdmin: trans.Ptr(true)})

	_, err := svc.Create(opCtx, &identityV1.CreatePlanRequest{
		Data: &identityV1.Plan{Name: trans.Ptr("待删除套餐"), Version: identityV1.Plan_FREE.Enum()},
	})
	require.NoError(t, err)

	rows, err := entClient.Client().Plan.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := rows[0].ID

	_, err = svc.Delete(auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7, IsPlatformAdmin: trans.Ptr(true)}), &identityV1.DeletePlanRequest{
		QueryBy: &identityV1.DeletePlanRequest_Id{Id: createdID},
	})
	require.NoError(t, err, "删除已存在记录应成功")

	cnt, err := entClient.Client().Plan.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, cnt, "删除后 plan 表计数应归零")
}
