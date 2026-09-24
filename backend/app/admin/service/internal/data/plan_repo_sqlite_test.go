package data

import (
	"context"
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
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newPlanRepoSqlite 在给定 enttest client 上白盒构造 PlanRepo，
// 逐字段复刻 NewPlanRepo 的 mapper/converter 初始化，再调用 init()。
func newPlanRepoSqlite(t *testing.T, entClient *entCrud.EntClient[*ent.Client]) *PlanRepo {
	t.Helper()
	repo := &PlanRepo{
		entClient: entClient,
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		mapper:    mapper.NewCopierMapper[identityV1.Plan, ent.Plan](),
		versionConverter: mapper.NewEnumTypeConverter[identityV1.Plan_Version, plan.Version](
			identityV1.Plan_Version_name, identityV1.Plan_Version_value,
		),
		expiryPolicyConv: mapper.NewEnumTypeConverter[identityV1.Plan_ExpiryPolicy, plan.ExpiryPolicy](
			identityV1.Plan_ExpiryPolicy_name, identityV1.Plan_ExpiryPolicy_value,
		),
	}
	repo.init()
	return repo
}

// TestPlanRepoSqlite_Create 通过 repo.Create 写入后直查 SQLite 断言落库
// （含枚举字段经 converter 的落库值）。
func TestPlanRepoSqlite_Create(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newPlanRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	err := repo.Create(ctx, &identityV1.CreatePlanRequest{
		Data: &identityV1.Plan{
			Name:              trans.Ptr("sqlite套餐-创建"),
			Version:           identityV1.Plan_FREE.Enum(),
			ExpiryPolicy:      identityV1.Plan_READONLY.Enum(),
			DataRetentionDays: trans.Ptr(uint32(30)),
		},
	})
	require.NoError(t, err, "repo.Create 应写入 SQLite 成功")

	rows, err := repo.entClient.Client().Plan.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1, "SQLite 中应有 1 条 plan 记录")
	require.Equal(t, "sqlite套餐-创建", *rows[0].Name, "name 应按请求落库")
	require.Equal(t, plan.VersionFree, *rows[0].Version, "version 枚举应经 converter 落为 FREE")
	require.Equal(t, plan.ExpiryPolicyReadonly, *rows[0].ExpiryPolicy, "expiry_policy 枚举应经 converter 落为 READONLY")
	require.NotNil(t, rows[0].DataRetentionDays)
	require.Equal(t, uint32(30), *rows[0].DataRetentionDays, "data_retention_days 应按请求落库")
}

// TestPlanRepoSqlite_CreateDuplicateName 验证套餐名唯一索引：
// 重名创建返回 400（BadRequest）而非 500。
func TestPlanRepoSqlite_CreateDuplicateName(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newPlanRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &identityV1.CreatePlanRequest{
		Data: &identityV1.Plan{Name: trans.Ptr("sqlite套餐-重名")},
	}))
	err := repo.Create(ctx, &identityV1.CreatePlanRequest{
		Data: &identityV1.Plan{Name: trans.Ptr("sqlite套餐-重名")},
	})
	require.Error(t, err, "重名创建应命中唯一索引报错")
}

// TestPlanRepoSqlite_ListContainsFilter 验证 List 的 contains 模糊搜索语义。
func TestPlanRepoSqlite_ListContainsFilter(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newPlanRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &identityV1.CreatePlanRequest{
		Data: &identityV1.Plan{Name: trans.Ptr("套餐-markerpoi-甲")},
	}))
	require.NoError(t, repo.Create(ctx, &identityV1.CreatePlanRequest{
		Data: &identityV1.Plan{Name: trans.Ptr("套餐-无关行-乙")},
	}))

	filtered, err := repo.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_FilterExpr{
			FilterExpr: &paginationV1.FilterExpr{
				Type: paginationV1.ExprType_AND,
				Conditions: []*paginationV1.FilterCondition{
					{
						Field:      "name",
						Op:         paginationV1.Operator_CONTAINS,
						ValueOneof: &paginationV1.FilterCondition_Value{Value: "markerpoi"},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), filtered.Total, "contains 过滤后 Total 应为 1")
	require.Len(t, filtered.Items, 1, "contains 过滤后只应返回 1 行")
	require.Contains(t, *filtered.Items[0].Name, "markerpoi", "命中行应是携带标记的行")

	none, err := repo.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_FilterExpr{
			FilterExpr: &paginationV1.FilterExpr{
				Type: paginationV1.ExprType_AND,
				Conditions: []*paginationV1.FilterCondition{
					{
						Field:      "name",
						Op:         paginationV1.Operator_CONTAINS,
						ValueOneof: &paginationV1.FilterCondition_Value{Value: "no-such-marker-zzz"},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(0), none.Total, "无命中 contains 应返回 Total=0")
	require.Empty(t, none.Items)

	all, err := repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(2), all.Total, "无过滤时应返回全部 2 行")
	require.Len(t, all.Items, 2)
}

// TestPlanRepoSqlite_Get 验证 Get 命中/未命中。
func TestPlanRepoSqlite_Get(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newPlanRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &identityV1.CreatePlanRequest{
		Data: &identityV1.Plan{Name: trans.Ptr("sqlite套餐-Get")},
	}))
	rows, err := repo.entClient.Client().Plan.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := rows[0].ID

	hit, err := repo.Get(ctx, &identityV1.GetPlanRequest{
		QueryBy: &identityV1.GetPlanRequest_Id{Id: createdID},
	})
	require.NoError(t, err, "按存在的 ID 查询应命中")
	require.Equal(t, createdID, hit.GetId())

	_, err = repo.Get(ctx, &identityV1.GetPlanRequest{
		QueryBy: &identityV1.GetPlanRequest_Id{Id: 9999999},
	})
	require.Error(t, err, "不存在的 ID 查询应返回错误")
}

// TestPlanRepoSqlite_Update 验证 Update 只更新掩码内字段（description），
// 掩码外字段（name）保持原值。
func TestPlanRepoSqlite_Update(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newPlanRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &identityV1.CreatePlanRequest{
		Data: &identityV1.Plan{
			Name:        trans.Ptr("sqlite套餐-更新"),
			Description: trans.Ptr("更新前描述"),
		},
	}))
	rows, err := repo.entClient.Client().Plan.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := rows[0].ID

	err = repo.Update(ctx, &identityV1.UpdatePlanRequest{
		Id:         createdID,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description"}},
		Data: &identityV1.Plan{
			Description: trans.Ptr("更新后描述-sqlite"),
		},
	})
	require.NoError(t, err, "更新 description 应成功")

	after, err := repo.entClient.Client().Plan.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, after, 1)
	require.Equal(t, "更新后描述-sqlite", *after[0].Description, "掩码内字段应被更新")
	require.Equal(t, "sqlite套餐-更新", *after[0].Name, "掩码外字段应保持原值")
}

// TestPlanRepoSqlite_EnumReadView 验证全部套餐版本/到期处置策略枚举值
// 经 converter 落库后，在读路径（Get 按主键 / List）的 DTO 视图如实呈现。
//
// 枚举字段读视图机制注记：实体侧 version/expiry_policy 为可空指针枚举列
// （*plan.Version / *plan.ExpiryPolicy），DTO 侧为可选指针字段。mapper 的
// 枚举转换对（EnumTypeConverter.NewConverterPair → NewGenericTypeConverterPair，
// 经 &srcType/&dstType 取址注册）恰为指针↔指针形态的键，指针对字段能被
// copier 直接转换赋值——与值型实体枚举列（如 position.type、
// notification_channel.type，带列默认、无指针）读侧被丢弃的情形不同。
// 本测试将该读视图行为钉死：若日后注册形态或 copier 匹配语义变化导致
// 读丢弃，此处会立即翻红。
func TestPlanRepoSqlite_EnumReadView(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newPlanRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	cases := []struct {
		protoVersion   identityV1.Plan_Version
		wantEntVersion plan.Version
		protoPolicy    identityV1.Plan_ExpiryPolicy
		wantEntPolicy  plan.ExpiryPolicy
		marker         string
	}{
		{
			identityV1.Plan_FREE, plan.VersionFree,
			identityV1.Plan_READONLY, plan.ExpiryPolicyReadonly,
			"sqlite_plan_enum_free_readonly",
		},
		{
			identityV1.Plan_STANDARD, plan.VersionStandard,
			identityV1.Plan_BLOCK_LOGIN, plan.ExpiryPolicyBlockLogin,
			"sqlite_plan_enum_standard_blocklogin",
		},
		{
			identityV1.Plan_ENTERPRISE, plan.VersionEnterprise,
			identityV1.Plan_FREEZE, plan.ExpiryPolicyFreeze,
			"sqlite_plan_enum_enterprise_freeze",
		},
	}

	for _, c := range cases {
		require.NoError(t, repo.Create(ctx, &identityV1.CreatePlanRequest{
			Data: &identityV1.Plan{
				Name:         trans.Ptr(c.marker),
				Version:      c.protoVersion.Enum(),
				ExpiryPolicy: c.protoPolicy.Enum(),
			},
		}), "版本 %v/策略 %v 创建应成功", c.protoVersion, c.protoPolicy)

		rows, err := entClient.Client().Plan.Query().
			Where(plan.NameEQ(c.marker)).
			All(ctx)
		require.NoError(t, err)
		require.Len(t, rows, 1, "按名应反查到刚写入的行")
		require.Equal(t, c.wantEntVersion, *rows[0].Version, "版本 %v 应经转换器如实落库", c.protoVersion)
		require.Equal(t, c.wantEntPolicy, *rows[0].ExpiryPolicy, "策略 %v 应经转换器如实落库", c.protoPolicy)

		// 读路径一：Get 按主键命中后，DTO 视图应如实呈现两枚举字段。
		got, err := repo.Get(ctx, &identityV1.GetPlanRequest{
			QueryBy: &identityV1.GetPlanRequest_Id{Id: rows[0].ID},
		})
		require.NoError(t, err, "按主键读取应命中")
		require.Equal(t, c.protoVersion, got.GetVersion(), "读视图应如实呈现版本 %v", c.protoVersion)
		require.Equal(t, c.protoPolicy, got.GetExpiryPolicy(), "读视图应如实呈现策略 %v", c.protoPolicy)

		// 读路径二：List 无过滤返回该行，DTO 视图应如实呈现两枚举字段。
		listed, err := repo.List(ctx, &paginationV1.PagingRequest{})
		require.NoError(t, err)
		var hit *identityV1.Plan
		for _, item := range listed.Items {
			if item.GetName() == c.marker {
				hit = item
				break
			}
		}
		require.NotNil(t, hit, "List 应包含刚写入的行 %s", c.marker)
		require.Equal(t, c.protoVersion, hit.GetVersion(), "List 读视图应如实呈现版本 %v", c.protoVersion)
		require.Equal(t, c.protoPolicy, hit.GetExpiryPolicy(), "List 读视图应如实呈现策略 %v", c.protoPolicy)
	}
}

// TestPlanRepoSqlite_Delete 验证 Delete 后行数归零。
func TestPlanRepoSqlite_Delete(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newPlanRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &identityV1.CreatePlanRequest{
		Data: &identityV1.Plan{Name: trans.Ptr("sqlite套餐-待删除")},
	}))
	rows, err := repo.entClient.Client().Plan.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)

	require.NoError(t, repo.Delete(ctx, rows[0].ID), "Delete 应成功")
	cnt, err := repo.entClient.Client().Plan.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, cnt, "Delete 后表内行数应为 0")
}
