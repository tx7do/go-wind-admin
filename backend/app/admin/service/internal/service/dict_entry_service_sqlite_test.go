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
	entDictEntry "go-wind-admin/app/admin/service/internal/data/ent/dictentry"
	entDictType "go-wind-admin/app/admin/service/internal/data/ent/dicttype"
	"go-wind-admin/app/admin/service/internal/data/enttest"
	"go-wind-admin/pkg/middleware/auth"
)

// newDictEntryServiceForTest 白盒构造 DictEntryService，逐字段对齐
// NewDictEntryService 的装配（log 用 NopLogger helper）。
func newDictEntryServiceForTest(t *testing.T, entClient *entCrud.EntClient[*ent.Client]) *DictEntryService {
	t.Helper()
	return &DictEntryService{
		log:           bLogger.NewHelper(bLogger.NopLogger()),
		dictEntryRepo: data.NewDictEntryRepoForTest(entClient),
		dictTypeRepo:  data.NewDictTypeRepoForTest(entClient),
	}
}

// createDictTypeParent 直建一条父 dict_type 行，返回其主键。
func createDictTypeParent(t *testing.T, entClient *entCrud.EntClient[*ent.Client], ctx context.Context, typeCode string) int64 {
	t.Helper()
	parent, err := entClient.Client().DictType.Create().
		SetNillableTypeCode(trans.Ptr(typeCode)).
		SetNillableTypeName(trans.Ptr("父类型-" + typeCode)).
		Save(ctx)
	require.NoError(t, err, "直建父 dict_type 应成功")
	return parent.ID
}

// TestDictEntryServiceSqlite_Create_PersistsParentAssociation 验证服务层创建字典项时，
// 请求携带的 TypeId 真实落库为指向父类型的外键（历史上该条件曾写反导致 type_id 永远为 NULL）。
func TestDictEntryServiceSqlite_Create_PersistsParentAssociation(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newDictEntryServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7})

	parentID := createDictTypeParent(t, entClient, ctx, "svc-de-create-type")

	_, err := svc.Create(opCtx, &dictV1.CreateDictEntryRequest{
		Data: &dictV1.DictEntry{
			TypeId:     &parentID,
			EntryValue: trans.Ptr("svc-de-create-value"),
			IsEnabled:  trans.Ptr(true),
			SortOrder:  trans.Ptr(uint32(3)),
		},
	})
	require.NoError(t, err, "服务层 Create 应成功")

	rows, err := entClient.Client().DictEntry.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1, "dict_entry 表应落库 1 条记录")
	require.Equal(t, "svc-de-create-value", *rows[0].EntryValue, "entry_value 应按请求落库")
	require.NotNil(t, rows[0].IsEnabled, "is_enabled 应按请求落库")
	require.True(t, *rows[0].IsEnabled, "is_enabled 应落库为 true")
	require.NotNil(t, rows[0].SortOrder, "sort_order 应按请求落库")
	require.Equal(t, uint32(3), *rows[0].SortOrder, "sort_order 应落库为 3")
	require.NotNil(t, rows[0].CreatedBy, "created_by 应被服务层盖入操作人 ID")
	require.Equal(t, int64(7), *rows[0].CreatedBy, "created_by 应等于令牌声明中的操作人 ID")

	// 外键回归断言：entry 必须挂在请求指定的父类型上（边与 TypeId 条件写反的历史 bug）
	linked, err := entClient.Client().DictEntry.Query().
		Where(
			entDictEntry.HasDictTypeWith(entDictType.IDEQ(parentID)),
		).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, linked, "entry 的 dict_type 外键应指向请求的父类型，而非 NULL")
}

// TestDictEntryServiceSqlite_ListByTypeCode 验证服务层按类型代码列条目：
// 仅命中指定父类型（父类型隔离）、仅启用条目（禁用排除）、按 sort_order 升序；
// TypeCode 为空串应被服务层守卫拒绝。
func TestDictEntryServiceSqlite_ListByTypeCode(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newDictEntryServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7})

	parentA := createDictTypeParent(t, entClient, ctx, "svc-de-list-type-a")
	parentB := createDictTypeParent(t, entClient, ctx, "svc-de-list-type-b")

	// parentA：两条启用条目，sort_order 逆序写入（2 在前 1 在后），期望按 1、2 升序返回；
	// 另加一条禁用条目（sort 0），期望被排除。
	for _, e := range []struct {
		value    string
		sort     uint32
		enabled  bool
		parentID int64
	}{
		{"ENTRY-SORT-2", 2, true, parentA},
		{"ENTRY-SORT-1", 1, true, parentA},
		{"ENTRY-DISABLED", 0, false, parentA},
		{"ENTRY-OTHER-PARENT", 1, true, parentB},
	} {
		_, err := svc.Create(opCtx, &dictV1.CreateDictEntryRequest{
			Data: &dictV1.DictEntry{
				TypeId:     &e.parentID,
				EntryValue: trans.Ptr(e.value),
				IsEnabled:  trans.Ptr(e.enabled),
				SortOrder:  trans.Ptr(e.sort),
			},
		})
		require.NoError(t, err, "条目 %s 创建应成功", e.value)
	}

	respA, err := svc.ListByTypeCode(ctx, &dictV1.ListDictEntryByTypeCodeRequest{
		TypeCode: "svc-de-list-type-a",
	})
	require.NoError(t, err, "按父类型 A 的代码列表应成功")
	require.Len(t, respA.Items, 2, "父类型 A 应只返回 2 条启用条目（禁用与其他父类型的条目排除）")
	require.Equal(t, "ENTRY-SORT-1", respA.Items[0].GetEntryValue(), "条目应按 sort_order 升序排列（第 1 条 sort=1）")
	require.Equal(t, "ENTRY-SORT-2", respA.Items[1].GetEntryValue(), "条目应按 sort_order 升序排列（第 2 条 sort=2）")
	// 修复后 ListByTypeCode 与通用 List 一致回填 TypeId（条目本就来自该父类型）。
	for _, item := range respA.Items {
		require.Equal(t, respA.Items[0].GetTypeId(), item.GetTypeId(), "TypeId 应回填为父类型 A 的 ID")
	}

	respB, err := svc.ListByTypeCode(ctx, &dictV1.ListDictEntryByTypeCodeRequest{
		TypeCode: "svc-de-list-type-b",
	})
	require.NoError(t, err, "按父类型 B 的代码列表应成功")
	require.Len(t, respB.Items, 1, "父类型 B 应只返回其自身的 1 条条目")
	require.Equal(t, "ENTRY-OTHER-PARENT", respB.Items[0].GetEntryValue())
	require.NotEqual(t, respA.Items[0].GetTypeId(), respB.Items[0].GetTypeId(), "两父类型的 TypeId 回填应互不相同")

	_, err = svc.ListByTypeCode(ctx, &dictV1.ListDictEntryByTypeCodeRequest{TypeCode: ""})
	require.Error(t, err, "空类型代码应被服务层守卫拒绝（BadRequest）")
}

// TestDictEntryServiceSqlite_List 验证服务层 List 的 contains 模糊搜索语义
// 与 TypeId 从父类型边的回填。
func TestDictEntryServiceSqlite_List(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newDictEntryServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7})

	parentID := createDictTypeParent(t, entClient, ctx, "svc-de-update-type")

	for _, v := range []string{"列表-markerdeqwe-甲", "列表-无关行-乙"} {
		_, err := svc.Create(opCtx, &dictV1.CreateDictEntryRequest{
			Data: &dictV1.DictEntry{
				TypeId:     &parentID,
				EntryValue: trans.Ptr(v),
				IsEnabled:  trans.Ptr(true),
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
						Field:      "entry_value",
						Op:         paginationV1.Operator_CONTAINS,
						ValueOneof: &paginationV1.FilterCondition_Value{Value: "markerdeqwe"},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), filtered.Total, "contains 过滤后 Total 应为 1")
	require.Len(t, filtered.Items, 1, "contains 过滤后只应返回 1 条")
	require.Contains(t, filtered.Items[0].GetEntryValue(), "markerdeqwe", "命中行应是携带标记的那条")

	all, err := svc.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, int64(2), all.Total, "无过滤时应统计全部 2 条")
	require.Len(t, all.Items, 2, "无过滤时应返回 2 条")
	for _, item := range all.Items {
		require.NotNil(t, item.TypeId, "列表项应从父类型边回填 TypeId")
		require.Equal(t, parentID, *item.TypeId, "回填的 TypeId 应等于父类型 ID")
	}
}

// TestDictEntryServiceSqlite_Update 验证服务层 Update 在单字段掩码下
// 只更新掩码内字段（entry_value），掩码外字段（sort_order）保持原值。
func TestDictEntryServiceSqlite_Update(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newDictEntryServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7})

	parentID := createDictTypeParent(t, entClient, ctx, "svc-de-update-type")

	_, err := svc.Create(opCtx, &dictV1.CreateDictEntryRequest{
		Data: &dictV1.DictEntry{
			TypeId:     &parentID,
			EntryValue: trans.Ptr("更新前条目值"),
			IsEnabled:  trans.Ptr(true),
			SortOrder:  trans.Ptr(uint32(5)),
		},
	})
	require.NoError(t, err)

	rows, err := entClient.Client().DictEntry.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := rows[0].ID

	_, err = svc.Update(opCtx, &dictV1.UpdateDictEntryRequest{
		Id:         createdID,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"entry_value"}},
		Data:       &dictV1.DictEntry{EntryValue: trans.Ptr("更新后条目值")},
	})
	require.NoError(t, err, "单字段掩码更新应成功")

	after, err := entClient.Client().DictEntry.Query().Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "更新后条目值", *after.EntryValue, "掩码内字段 entry_value 应被更新")
	require.NotNil(t, after.SortOrder, "掩码外字段 sort_order 应保持原值")
	require.Equal(t, uint32(5), *after.SortOrder, "掩码外字段 sort_order 应保持原值")
}

// TestDictEntryServiceSqlite_Delete 验证服务层 Delete（按 ID 列表批量删除）后表内计数归零。
func TestDictEntryServiceSqlite_Delete(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newDictEntryServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7})

	parentID := createDictTypeParent(t, entClient, ctx, "svc-de-del-type")

	for _, v := range []string{"待删除条目甲", "待删除条目乙"} {
		_, err := svc.Create(opCtx, &dictV1.CreateDictEntryRequest{
			Data: &dictV1.DictEntry{
				TypeId:     &parentID,
				EntryValue: trans.Ptr(v),
				IsEnabled:  trans.Ptr(true),
			},
		})
		require.NoError(t, err)
	}

	rows, err := entClient.Client().DictEntry.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	ids := []int64{rows[0].ID, rows[1].ID}

	_, err = svc.Delete(ctx, &dictV1.DeleteDictEntryRequest{Ids: ids})
	require.NoError(t, err, "批量删除应成功")

	cnt, err := entClient.Client().DictEntry.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, cnt, "删除后 dict_entry 表计数应归零")
}
