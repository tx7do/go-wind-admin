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
	entPosition "go-wind-admin/app/admin/service/internal/data/ent/position"
	"go-wind-admin/app/admin/service/internal/data/enttest"
	"go-wind-admin/pkg/middleware/auth"
)

// newPositionServiceForTest 白盒构造 PositionService，逐字段对齐 NewPositionService
// 的装配：log 用 NopLogger helper，positionRepo 走 data.NewPositionRepoForTest；
// orgUnitRepo 为本测试未关联组织单元的职位保持 nil（enrichRelations 在空关联集下
// 不会触碰该字段）。
func newPositionServiceForTest(t *testing.T, entClient *entCrud.EntClient[*ent.Client]) *PositionService {
	t.Helper()
	return &PositionService{
		log:          bLogger.NewHelper(bLogger.NopLogger()),
		positionRepo: data.NewPositionRepoForTest(entClient),
		orgUnitRepo:  nil,
	}
}

// TestPositionServiceSqlite_CreateListCountAndGet 验证服务层创建职位后：
// 枚举字段经转换器落库、操作人 ID 盖入 created_by；List 全量返回且无组织单元关联时
// enrichRelations 为空回填（OrgUnitName 为空）；Count 与 Get 按主键的命中/未命中。
func TestPositionServiceSqlite_CreateListCountAndGet(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newPositionServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7})

	for i, name := range []string{"服务层职位甲", "服务层职位乙"} {
		_, err := svc.Create(opCtx, &identityV1.CreatePositionRequest{
			Data: &identityV1.Position{
				Name:   trans.Ptr(name),
				Code:   trans.Ptr(fmt.Sprintf("POS_SVC_%d", 1000+i)),
				Status: identityV1.Position_ON.Enum(),
				Type:   identityV1.Position_MANAGER.Enum(),
			},
		})
		require.NoError(t, err, fmt.Sprintf("第 %d 个职位创建应成功", i))
	}

	rows, err := entClient.Client().Position.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 2, "position 表应落库 2 条记录")
	for _, r := range rows {
		require.Equal(t, entPosition.StatusOn, *r.Status, "status 应经转换器落库为 ON")
		require.Equal(t, entPosition.TypeManager, r.Type, "type 应经转换器落库为 MANAGER")
		require.NotNil(t, r.CreatedBy, "created_by 应被服务层盖入操作人 ID")
		require.Equal(t, uint64(7), *r.CreatedBy, "created_by 应等于令牌声明中的操作人 ID")
	}

	listResp, err := svc.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(2), listResp.Total, "无过滤时应统计全部 2 条")
	require.Len(t, listResp.Items, 2, "无过滤时应返回 2 条")
	for _, item := range listResp.Items {
		require.Empty(t, item.GetOrgUnitName(), "无组织单元关联的职位不应回填组织单元名")
	}

	countResp, err := svc.Count(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(2), countResp.GetCount(), "Count 应统计全部 2 条")

	got, err := svc.Get(ctx, &identityV1.GetPositionRequest{
		QueryBy: &identityV1.GetPositionRequest_Id{Id: uint64(rows[0].ID)},
	})
	require.NoError(t, err, "按已存在主键查询应命中")
	require.Equal(t, "服务层职位甲", got.GetName(), "命中记录的 name 应与写入一致")

	_, err = svc.Get(ctx, &identityV1.GetPositionRequest{
		QueryBy: &identityV1.GetPositionRequest_Id{Id: 9999999},
	})
	require.Error(t, err, "按不存在主键查询应返回错误")
}

// TestPositionServiceSqlite_List_ContainsFilter 验证服务层 List 的
// contains 模糊搜索语义（仓规：搜索条件一律 contains）。
func TestPositionServiceSqlite_List_ContainsFilter(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newPositionServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7})

	for i, name := range []string{"MARKERPOSALPHA 职位", "无关职位乙"} {
		_, err := svc.Create(opCtx, &identityV1.CreatePositionRequest{
			Data: &identityV1.Position{
				Name:   trans.Ptr(name),
				Code:   trans.Ptr(fmt.Sprintf("POS_SVC_F_%d", 2000+i)),
				Status: identityV1.Position_ON.Enum(),
			},
		})
		require.NoError(t, err)
	}

	filtered, err := svc.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_FilterExpr{
			FilterExpr: &paginationV1.FilterExpr{
				Type: paginationV1.ExprType_AND,
				Conditions: []*paginationV1.FilterCondition{
					{
						Field:      "name",
						Op:         paginationV1.Operator_CONTAINS,
						ValueOneof: &paginationV1.FilterCondition_Value{Value: "MARKERPOSALPHA"},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), filtered.Total, "contains 过滤后 Total 应为 1")
	require.Len(t, filtered.Items, 1, "contains 过滤后只应返回 1 条")
	require.Contains(t, filtered.Items[0].GetName(), "MARKERPOSALPHA", "命中行应是携带标记的那条")
}

// TestPositionServiceSqlite_Update 验证服务层 Update 在单字段掩码下
// 只更新掩码内字段（name），掩码外字段（code）保持原值。
func TestPositionServiceSqlite_Update(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newPositionServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7})

	_, err := svc.Create(opCtx, &identityV1.CreatePositionRequest{
		Data: &identityV1.Position{
			Name:   trans.Ptr("更新前职位名"),
			Code:   trans.Ptr("POS_SVC_U_3001"),
			Status: identityV1.Position_ON.Enum(),
		},
	})
	require.NoError(t, err)

	rows, err := entClient.Client().Position.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := rows[0].ID

	_, err = svc.Update(opCtx, &identityV1.UpdatePositionRequest{
		Id:         createdID,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"name"}},
		Data:       &identityV1.Position{Name: trans.Ptr("更新后职位名")},
	})
	require.NoError(t, err, "单字段掩码更新应成功")

	after, err := entClient.Client().Position.Query().Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "更新后职位名", *after.Name, "掩码内字段 name 应被更新")
	require.Equal(t, "POS_SVC_U_3001", *after.Code, "掩码外字段 code 应保持原值")
}

// TestPositionServiceSqlite_Delete 验证服务层 Delete 后表内计数归零。
func TestPositionServiceSqlite_Delete(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newPositionServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7})

	_, err := svc.Create(opCtx, &identityV1.CreatePositionRequest{
		Data: &identityV1.Position{
			Name:   trans.Ptr("待删除职位"),
			Code:   trans.Ptr("POS_SVC_D_4001"),
			Status: identityV1.Position_ON.Enum(),
		},
	})
	require.NoError(t, err)

	rows, err := entClient.Client().Position.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := rows[0].ID

	_, err = svc.Delete(ctx, &identityV1.DeletePositionRequest{
		QueryBy: &identityV1.DeletePositionRequest_Id{Id: createdID},
	})
	require.NoError(t, err, "删除已存在记录应成功")

	cnt, err := entClient.Client().Position.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, cnt, "删除后 position 表计数应归零")
}
