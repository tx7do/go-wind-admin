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
	"go-wind-admin/app/admin/service/internal/data/ent/planquota"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newPlanQuotaRepoSqlite 在给定 enttest client 上白盒构造 PlanQuotaRepo，
// 逐字段复刻 NewPlanQuotaRepo 的 mapper/converter 初始化，再调用 init()。
func newPlanQuotaRepoSqlite(t *testing.T, entClient *entCrud.EntClient[*ent.Client]) *PlanQuotaRepo {
	t.Helper()
	repo := &PlanQuotaRepo{
		entClient: entClient,
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		mapper:    mapper.NewCopierMapper[identityV1.PlanQuota, ent.PlanQuota](),
		quotaTypeConv: mapper.NewEnumTypeConverter[identityV1.PlanQuota_QuotaType, planquota.QuotaType](
			identityV1.PlanQuota_QuotaType_name, identityV1.PlanQuota_QuotaType_value,
		),
	}
	repo.init()
	return repo
}

// TestPlanQuotaRepoSqlite_Create 带父 plan 创建配额项，
// 直查断言：行落库、(plan_quota → plan) 外键真实落库到请求指定的父行。
// 历史上这里条件写反导致 plan_id 落库为 NULL，本用例为其回归测试。
func TestPlanQuotaRepoSqlite_Create(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newPlanQuotaRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	parent, err := entClient.Client().Plan.Create().
		SetNillableName(trans.Ptr("sqlite_pq_create_plan")).
		Save(ctx)
	require.NoError(t, err, "直建父 plan 应成功")

	err = repo.Create(ctx, &identityV1.CreatePlanQuotaRequest{
		Data: &identityV1.PlanQuota{
			PlanId:     &parent.ID,
			QuotaType:  identityV1.PlanQuota_USER_LIMIT.Enum(),
			QuotaValue: trans.Ptr(uint64(100)),
		},
	})
	require.NoError(t, err, "repo.Create 应写入 SQLite 成功")

	rows, err := entClient.Client().PlanQuota.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1, "SQLite 中应有 1 条 plan_quota 记录")
	require.NotNil(t, rows[0].QuotaType, "quota_type 枚举应经 converter 落库")
	require.Equal(t, planquota.QuotaTypeUserLimit, *rows[0].QuotaType, "quota_type 枚举应落为 USER_LIMIT")
	require.NotNil(t, rows[0].QuotaValue)
	require.Equal(t, uint64(100), *rows[0].QuotaValue, "quota_value 应按请求落库")

	// 外键回归断言：配额项必须挂在请求指定的父套餐上
	linked, err := entClient.Client().PlanQuota.Query().
		Where(planquota.HasPlanWith(plan.IDEQ(parent.ID))).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, linked, "plan_quota 的 plan 外键应指向请求的父套餐，而非 NULL")
}

// TestPlanQuotaRepoSqlite_ListFilter 验证 List 的等值过滤（quota_type 列）与
// 列表路径上 PlanId 从父套餐边正确回填。
func TestPlanQuotaRepoSqlite_ListFilter(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newPlanQuotaRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	parent, err := entClient.Client().Plan.Create().
		SetNillableName(trans.Ptr("sqlite_pq_list_plan")).
		Save(ctx)
	require.NoError(t, err)

	require.NoError(t, repo.Create(ctx, &identityV1.CreatePlanQuotaRequest{
		Data: &identityV1.PlanQuota{
			PlanId:     &parent.ID,
			QuotaType:  identityV1.PlanQuota_USER_LIMIT.Enum(),
			QuotaValue: trans.Ptr(uint64(1)),
		},
	}))
	require.NoError(t, repo.Create(ctx, &identityV1.CreatePlanQuotaRequest{
		Data: &identityV1.PlanQuota{
			PlanId:     &parent.ID,
			QuotaType:  identityV1.PlanQuota_STORAGE.Enum(),
			QuotaValue: trans.Ptr(uint64(2)),
		},
	}))

	// 等值过滤：quota_type = USER_LIMIT 只命中第一行
	filtered, err := repo.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_FilterExpr{
			FilterExpr: &paginationV1.FilterExpr{
				Type: paginationV1.ExprType_AND,
				Conditions: []*paginationV1.FilterCondition{
					{
						Field:      "quota_type",
						Op:         paginationV1.Operator_EQ,
						ValueOneof: &paginationV1.FilterCondition_Value{Value: "USER_LIMIT"},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), filtered.Total, "等值过滤后 Total 应为 1")
	require.Len(t, filtered.Items, 1, "等值过滤后只应返回 1 行")
	require.NotNil(t, filtered.Items[0].QuotaType, "命中行的 quota_type 应被映射回 DTO")
	require.Equal(t, identityV1.PlanQuota_USER_LIMIT, *filtered.Items[0].QuotaType, "命中行应为 USER_LIMIT 行")

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

// TestPlanQuotaRepoSqlite_Get 验证 Get 命中/未命中。
func TestPlanQuotaRepoSqlite_Get(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newPlanQuotaRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	parent, err := entClient.Client().Plan.Create().
		SetNillableName(trans.Ptr("sqlite_pq_get_plan")).
		Save(ctx)
	require.NoError(t, err)

	require.NoError(t, repo.Create(ctx, &identityV1.CreatePlanQuotaRequest{
		Data: &identityV1.PlanQuota{
			PlanId:     &parent.ID,
			QuotaType:  identityV1.PlanQuota_USER_LIMIT.Enum(),
			QuotaValue: trans.Ptr(uint64(5)),
		},
	}))
	rows, err := entClient.Client().PlanQuota.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := rows[0].ID

	hit, err := repo.Get(ctx, &identityV1.GetPlanQuotaRequest{
		QueryBy: &identityV1.GetPlanQuotaRequest_Id{Id: createdID},
	})
	require.NoError(t, err, "按存在的 ID 查询应命中")
	require.Equal(t, createdID, hit.GetId())

	_, err = repo.Get(ctx, &identityV1.GetPlanQuotaRequest{
		QueryBy: &identityV1.GetPlanQuotaRequest_Id{Id: 9999999},
	})
	require.Error(t, err, "不存在的 ID 查询应返回错误")
}

// TestPlanQuotaRepoSqlite_QuotaTypeReadView 验证全部 3 个配额类型枚举值经
// converter 落库后，在读路径（Get 按主键 / List）的 DTO 视图如实呈现。
//
// 枚举字段读视图机制注记：实体侧 quota_type 为可空指针枚举列
// （*planquota.QuotaType），DTO 侧为可选指针字段。mapper 的枚举转换对
// （经 &srcType/&dstType 取址注册）恰为指针↔指针形态的键，指针对字段能被
// copier 直接转换赋值——与值型实体枚举列（如 position.type、
// notification_channel.type）读侧被丢弃的情形不同。本测试将该读视图行为钉死。
// 注：PLAN_QUOTA_TYPE_UNSPECIFIED 为 ent schema 未声明的值，直传会触发
// ValueType 系校验失败（写路径无零值守卫），故只覆盖 3 个合法值。
func TestPlanQuotaRepoSqlite_QuotaTypeReadView(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newPlanQuotaRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	cases := []struct {
		protoQuotaType identityV1.PlanQuota_QuotaType
		wantEnt        planquota.QuotaType
	}{
		{identityV1.PlanQuota_USER_LIMIT, planquota.QuotaTypeUserLimit},
		{identityV1.PlanQuota_STORAGE, planquota.QuotaTypeStorage},
		{identityV1.PlanQuota_API_CALL, planquota.QuotaTypeApiCall},
	}

	for _, c := range cases {
		// 每个用例独立父套餐，List 匹配依赖 PlanId 回填
		parent, err := entClient.Client().Plan.Create().
			SetNillableName(trans.Ptr(fmt.Sprintf("sqlite_pq_readview_plan_%d", c.protoQuotaType))).
			Save(ctx)
		require.NoError(t, err, "直建父 plan 应成功")

		require.NoError(t, repo.Create(ctx, &identityV1.CreatePlanQuotaRequest{
			Data: &identityV1.PlanQuota{
				PlanId:     &parent.ID,
				QuotaType:  c.protoQuotaType.Enum(),
				QuotaValue: trans.Ptr(uint64(1)),
			},
		}), "配额类型 %v 创建应成功", c.protoQuotaType)

		rows, err := entClient.Client().PlanQuota.Query().
			Where(planquota.HasPlanWith(plan.IDEQ(parent.ID))).
			All(ctx)
		require.NoError(t, err)
		require.Len(t, rows, 1, "按父套餐应反查到刚写入的行")
		require.Equal(t, c.wantEnt, *rows[0].QuotaType, "配额类型 %v 应经转换器如实落库", c.protoQuotaType)

		// 读路径一：Get 按主键命中后，DTO 视图应如实呈现配额类型枚举。
		got, err := repo.Get(ctx, &identityV1.GetPlanQuotaRequest{
			QueryBy: &identityV1.GetPlanQuotaRequest_Id{Id: rows[0].ID},
		})
		require.NoError(t, err, "按主键读取应命中")
		require.Equal(t, c.protoQuotaType, got.GetQuotaType(), "读视图应如实呈现配额类型 %v", c.protoQuotaType)

		// 读路径二：List 应含该行且 PlanId 回填，DTO 视图如实呈现配额类型枚举。
		listed, err := repo.List(ctx, &paginationV1.PagingRequest{})
		require.NoError(t, err)
		var hit *identityV1.PlanQuota
		for _, item := range listed.Items {
			if item.GetPlanId() == parent.ID {
				hit = item
				break
			}
		}
		require.NotNil(t, hit, "List 应包含父套餐 %d 下的配额行", parent.ID)
		require.Equal(t, c.protoQuotaType, hit.GetQuotaType(), "List 读视图应如实呈现配额类型 %v", c.protoQuotaType)
	}
}

// TestPlanQuotaRepoSqlite_Update 验证 Update 掩码内字段（quota_value）更新、
// quota_type 与父套餐外键不受影响。
func TestPlanQuotaRepoSqlite_Update(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newPlanQuotaRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	parent, err := entClient.Client().Plan.Create().
		SetNillableName(trans.Ptr("sqlite_pq_update_plan")).
		Save(ctx)
	require.NoError(t, err)

	require.NoError(t, repo.Create(ctx, &identityV1.CreatePlanQuotaRequest{
		Data: &identityV1.PlanQuota{
			PlanId:     &parent.ID,
			QuotaType:  identityV1.PlanQuota_STORAGE.Enum(),
			QuotaValue: trans.Ptr(uint64(1)),
		},
	}))
	rows, err := entClient.Client().PlanQuota.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := rows[0].ID

	err = repo.Update(ctx, &identityV1.UpdatePlanQuotaRequest{
		Id:         createdID,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"quota_value"}},
		Data: &identityV1.PlanQuota{
			QuotaValue: trans.Ptr(uint64(999)),
		},
	})
	require.NoError(t, err, "更新 quota_value 应成功")

	after, err := entClient.Client().PlanQuota.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, after, 1)
	require.NotNil(t, after[0].QuotaValue)
	require.Equal(t, uint64(999), *after[0].QuotaValue, "掩码内字段 quota_value 应被更新")
	require.Equal(t, planquota.QuotaTypeStorage, *after[0].QuotaType, "掩码外字段 quota_type 应保持原值")

	linked, err := entClient.Client().PlanQuota.Query().
		Where(planquota.HasPlanWith(plan.IDEQ(parent.ID))).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, linked, "更新不应改变 plan_quota 的父套餐外键")
}

// TestPlanQuotaRepoSqlite_Delete 验证 Delete 后行数归零。
func TestPlanQuotaRepoSqlite_Delete(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newPlanQuotaRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	parent, err := entClient.Client().Plan.Create().
		SetNillableName(trans.Ptr("sqlite_pq_del_plan")).
		Save(ctx)
	require.NoError(t, err)

	require.NoError(t, repo.Create(ctx, &identityV1.CreatePlanQuotaRequest{
		Data: &identityV1.PlanQuota{
			PlanId:     &parent.ID,
			QuotaType:  identityV1.PlanQuota_USER_LIMIT.Enum(),
			QuotaValue: trans.Ptr(uint64(7)),
		},
	}))
	rows, err := entClient.Client().PlanQuota.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)

	require.NoError(t, repo.Delete(ctx, rows[0].ID), "Delete 应成功")
	cnt, err := entClient.Client().PlanQuota.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, cnt, "Delete 后表内行数应为 0")
}
