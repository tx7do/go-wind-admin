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
	dictV1 "go-wind-admin/api/gen/go/dict/service/v1"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/enttest"
	"go-wind-admin/pkg/middleware/auth"
)

// newDictTypeServiceForTest 白盒构造 DictTypeService，逐字段对齐
// NewDictTypeService 的装配（log 用 NopLogger helper）。
func newDictTypeServiceForTest(t *testing.T, entClient *entCrud.EntClient[*ent.Client]) *DictTypeService {
	t.Helper()
	return &DictTypeService{
		log:          bLogger.NewHelper(bLogger.NopLogger()),
		dictTypeRepo: data.NewDictTypeRepoForTest(entClient),
	}
}

// TestDictTypeServiceSqlite_CreateAndGet 验证服务层创建字典类型后按主键查询命中，
// 且操作人 ID 被盖入 created_by；不存在主键查询应报错。
func TestDictTypeServiceSqlite_CreateAndGet(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newDictTypeServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7})

	_, err := svc.Create(opCtx, &dictV1.CreateDictTypeRequest{
		Data: &dictV1.DictType{
			TypeCode: trans.Ptr("svc-dt-create-code"),
			TypeName: trans.Ptr("服务层字典类型-创建"),
		},
	})
	require.NoError(t, err, "服务层 Create 应成功")

	rows, err := entClient.Client().DictType.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1, "dict_type 表应落库 1 条记录")
	require.Equal(t, "svc-dt-create-code", *rows[0].TypeCode, "type_code 应按请求落库")
	require.Equal(t, "服务层字典类型-创建", *rows[0].TypeName, "type_name 应按请求落库")
	require.NotNil(t, rows[0].CreatedBy, "created_by 应被服务层盖入操作人 ID")
	require.Equal(t, int64(7), *rows[0].CreatedBy, "created_by 应等于令牌声明中的操作人 ID")

	got, err := svc.Get(ctx, &dictV1.GetDictTypeRequest{
		QueryBy: &dictV1.GetDictTypeRequest_Id{Id: rows[0].ID},
	})
	require.NoError(t, err, "按已存在主键查询应命中")
	require.Equal(t, "服务层字典类型-创建", got.GetTypeName(), "命中记录的 type_name 应与写入一致")

	_, err = svc.Get(ctx, &dictV1.GetDictTypeRequest{
		QueryBy: &dictV1.GetDictTypeRequest_Id{Id: 9999999},
	})
	require.Error(t, err, "按不存在主键查询应返回错误")
}

// TestDictTypeServiceSqlite_List 验证服务层 List 的全量返回与
// contains 模糊搜索语义（仓规：搜索条件一律 contains）。
func TestDictTypeServiceSqlite_List(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newDictTypeServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7})

	for _, tc := range []struct{ code, name string }{
		{"svc-dt-list-code-a", "列表-markerdtqwe-甲"},
		{"svc-dt-list-code-b", "列表-无关行-乙"},
	} {
		_, err := svc.Create(opCtx, &dictV1.CreateDictTypeRequest{
			Data: &dictV1.DictType{TypeCode: trans.Ptr(tc.code), TypeName: trans.Ptr(tc.name)},
		})
		require.NoError(t, err)
	}

	all, err := svc.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, int64(2), all.Total, "无过滤时应统计全部 2 条")
	require.Len(t, all.Items, 2, "无过滤时应返回 2 条")

	filtered, err := svc.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_FilterExpr{
			FilterExpr: &paginationV1.FilterExpr{
				Type: paginationV1.ExprType_AND,
				Conditions: []*paginationV1.FilterCondition{
					{
						Field:      "type_name",
						Op:         paginationV1.Operator_CONTAINS,
						ValueOneof: &paginationV1.FilterCondition_Value{Value: "markerdtqwe"},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), filtered.Total, "contains 过滤后 Total 应为 1")
	require.Len(t, filtered.Items, 1, "contains 过滤后只应返回 1 条")
	require.Contains(t, filtered.Items[0].GetTypeName(), "markerdtqwe", "命中行应是携带标记的那条")
}

// TestDictTypeServiceSqlite_Update 验证服务层 Update 在单字段掩码下
// 只更新掩码内字段（type_name），掩码外字段（type_code）保持原值。
func TestDictTypeServiceSqlite_Update(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newDictTypeServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7})

	_, err := svc.Create(opCtx, &dictV1.CreateDictTypeRequest{
		Data: &dictV1.DictType{
			TypeCode: trans.Ptr("svc-dt-update-code"),
			TypeName: trans.Ptr("更新前类型名"),
		},
	})
	require.NoError(t, err)

	rows, err := entClient.Client().DictType.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := rows[0].ID

	_, err = svc.Update(opCtx, &dictV1.UpdateDictTypeRequest{
		Id:         createdID,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"type_name"}},
		Data:       &dictV1.DictType{TypeName: trans.Ptr("更新后类型名-服务层")},
	})
	require.NoError(t, err, "单字段掩码更新应成功")

	after, err := entClient.Client().DictType.Query().Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "更新后类型名-服务层", *after.TypeName, "掩码内字段 type_name 应被更新")
	require.Equal(t, "svc-dt-update-code", *after.TypeCode, "掩码外字段 type_code 应保持原值")
}

// TestDictTypeServiceSqlite_Delete 验证服务层 Delete（按 ID 列表批量删除）后表内计数归零。
func TestDictTypeServiceSqlite_Delete(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newDictTypeServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7})

	for _, code := range []string{"svc-dt-del-code-a", "svc-dt-del-code-b"} {
		_, err := svc.Create(opCtx, &dictV1.CreateDictTypeRequest{
			Data: &dictV1.DictType{TypeCode: trans.Ptr(code), TypeName: trans.Ptr("待删除-" + code)},
		})
		require.NoError(t, err)
	}

	rows, err := entClient.Client().DictType.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	ids := []int64{rows[0].ID, rows[1].ID}

	_, err = svc.Delete(ctx, &dictV1.DeleteDictTypeRequest{Ids: ids})
	require.NoError(t, err, "批量删除应成功")

	cnt, err := entClient.Client().DictType.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, cnt, "删除后 dict_type 表计数应归零")
}
