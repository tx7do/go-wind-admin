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

	dictV1 "go-wind-admin/api/gen/go/dict/service/v1"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newDictTypeRepoSqlite 在给定 enttest client 上白盒构造 DictTypeRepo，
// 逐字段复刻 NewDictTypeRepo 的 mapper 初始化，再调用 init()。
func newDictTypeRepoSqlite(t *testing.T, entClient *entCrud.EntClient[*ent.Client]) *DictTypeRepo {
	t.Helper()
	repo := &DictTypeRepo{
		entClient: entClient,
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		mapper:    mapper.NewCopierMapper[dictV1.DictType, ent.DictType](),
	}
	repo.init()
	return repo
}

// TestDictTypeRepoSqlite_Create 通过 repo.Create 写入后直查 SQLite 断言落库。
func TestDictTypeRepoSqlite_Create(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newDictTypeRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	err := repo.Create(ctx, &dictV1.CreateDictTypeRequest{
		Data: &dictV1.DictType{
			TypeCode: trans.Ptr("sqlite_dt_create_code"),
			TypeName: trans.Ptr("SQLite集成测试-创建"),
		},
	})
	require.NoError(t, err, "repo.Create 应写入 SQLite 成功")

	rows, err := repo.entClient.Client().DictType.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1, "SQLite 中应有 1 条 dict_type 记录")
	require.Equal(t, "sqlite_dt_create_code", *rows[0].TypeCode, "type_code 应按请求落库")
	require.Equal(t, "SQLite集成测试-创建", *rows[0].TypeName, "type_name 应按请求落库")
}

// TestDictTypeRepoSqlite_ListContainsFilter 验证 List 的 contains 模糊搜索语义：
// 只返回 type_name 含指定标记的行，Total 与过滤后行数一致；无过滤时返回全部。
func TestDictTypeRepoSqlite_ListContainsFilter(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newDictTypeRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	// 两行携带互斥标记，只有第一行含 "markerqwe"
	require.NoError(t, repo.Create(ctx, &dictV1.CreateDictTypeRequest{
		Data: &dictV1.DictType{
			TypeCode: trans.Ptr("sqlite_dt_list_code_a"),
			TypeName: trans.Ptr("列表过滤-markerqwe-甲"),
		},
	}))
	require.NoError(t, repo.Create(ctx, &dictV1.CreateDictTypeRequest{
		Data: &dictV1.DictType{
			TypeCode: trans.Ptr("sqlite_dt_list_code_b"),
			TypeName: trans.Ptr("列表过滤-无关行-乙"),
		},
	}))

	// contains 过滤：只命中第一行
	filtered, err := repo.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_FilterExpr{
			FilterExpr: &paginationV1.FilterExpr{
				Type: paginationV1.ExprType_AND,
				Conditions: []*paginationV1.FilterCondition{
					{
						Field:      "type_name",
						Op:         paginationV1.Operator_CONTAINS,
						ValueOneof: &paginationV1.FilterCondition_Value{Value: "markerqwe"},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), filtered.Total, "contains 过滤后 Total 应为 1")
	require.Len(t, filtered.Items, 1, "contains 过滤后只应返回 1 行")
	require.Contains(t, *filtered.Items[0].TypeName, "markerqwe", "命中行应是携带标记的行")

	// contains 无命中：标记不存在
	none, err := repo.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_FilterExpr{
			FilterExpr: &paginationV1.FilterExpr{
				Type: paginationV1.ExprType_AND,
				Conditions: []*paginationV1.FilterCondition{
					{
						Field:      "type_name",
						Op:         paginationV1.Operator_CONTAINS,
						ValueOneof: &paginationV1.FilterCondition_Value{Value: "no-such-marker-zzz"},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(0), none.Total, "无命中 contains 应返回 Total=0")
	require.Empty(t, none.Items, "无命中 contains 应返回空列表")

	// 无过滤：返回全部两行
	all, err := repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(2), all.Total, "无过滤时应返回全部 2 行")
	require.Len(t, all.Items, 2)
}

// TestDictTypeRepoSqlite_Get 验证 Get 命中 / 未命中，
// 以及平台上下文按 type_code 查询被拒绝的分支。
func TestDictTypeRepoSqlite_Get(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newDictTypeRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &dictV1.CreateDictTypeRequest{
		Data: &dictV1.DictType{
			TypeCode: trans.Ptr("sqlite_dt_get_code"),
			TypeName: trans.Ptr("SQLite集成测试-Get"),
		},
	}))
	rows, err := repo.entClient.Client().DictType.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := rows[0].ID

	// 命中：按 ID 查询
	hit, err := repo.Get(ctx, &dictV1.GetDictTypeRequest{
		QueryBy: &dictV1.GetDictTypeRequest_Id{Id: createdID},
	})
	require.NoError(t, err, "按存在的 ID 查询应命中")
	require.Equal(t, createdID, hit.GetId(), "DTO 应带回正确的 ID")
	require.Equal(t, "SQLite集成测试-Get", hit.GetTypeName(), "DTO 应带回 type_name")

	// 未命中：不存在的 ID
	_, err = repo.Get(ctx, &dictV1.GetDictTypeRequest{
		QueryBy: &dictV1.GetDictTypeRequest_Id{Id: 9999999},
	})
	require.Error(t, err, "不存在的 ID 查询应返回错误")

	// 平台上下文（无租户）按 type_code 查询应被拒绝
	_, err = repo.Get(ctx, &dictV1.GetDictTypeRequest{
		QueryBy: &dictV1.GetDictTypeRequest_Code{Code: "sqlite_dt_get_code"},
	})
	require.Error(t, err, "系统级上下文按 type_code 查询应返回 BadRequest")
}

// TestDictTypeRepoSqlite_Update 验证 Update 只更新掩码内字段，
// 掩码外字段（type_code）保持原值。
func TestDictTypeRepoSqlite_Update(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newDictTypeRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &dictV1.CreateDictTypeRequest{
		Data: &dictV1.DictType{
			TypeCode: trans.Ptr("sqlite_dt_update_code"),
			TypeName: trans.Ptr("更新前名称"),
		},
	}))
	rows, err := repo.entClient.Client().DictType.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := rows[0].ID

	err = repo.Update(ctx, &dictV1.UpdateDictTypeRequest{
		Id:         createdID,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"type_name"}},
		Data: &dictV1.DictType{
			TypeName: trans.Ptr("更新后名称-sqlite"),
		},
	})
	require.NoError(t, err, "更新 type_name 应成功")

	after, err := repo.entClient.Client().DictType.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, after, 1)
	require.Equal(t, "更新后名称-sqlite", *after[0].TypeName, "掩码内字段应被更新")
	require.Equal(t, "sqlite_dt_update_code", *after[0].TypeCode, "掩码外字段应保持原值")
}

// TestDictTypeRepoSqlite_Delete 验证 Delete 与 BatchDelete 后行数归零。
func TestDictTypeRepoSqlite_Delete(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newDictTypeRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &dictV1.CreateDictTypeRequest{
		Data: &dictV1.DictType{
			TypeCode: trans.Ptr("sqlite_dt_del_code_a"),
			TypeName: trans.Ptr("待删除-甲"),
		},
	}))
	rows, err := repo.entClient.Client().DictType.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)

	require.NoError(t, repo.Delete(ctx, rows[0].ID), "Delete 应成功")
	cnt, err := repo.entClient.Client().DictType.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, cnt, "Delete 后表内行数应为 0")

	// BatchDelete：两行一起删
	require.NoError(t, repo.Create(ctx, &dictV1.CreateDictTypeRequest{
		Data: &dictV1.DictType{
			TypeCode: trans.Ptr("sqlite_dt_del_code_b"),
			TypeName: trans.Ptr("待删除-乙"),
		},
	}))
	require.NoError(t, repo.Create(ctx, &dictV1.CreateDictTypeRequest{
		Data: &dictV1.DictType{
			TypeCode: trans.Ptr("sqlite_dt_del_code_c"),
			TypeName: trans.Ptr("待删除-丙"),
		},
	}))
	rows, err = repo.entClient.Client().DictType.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.NoError(t, repo.BatchDelete(ctx, []uint32{rows[0].ID, rows[1].ID}), "BatchDelete 应成功")
	cnt, err = repo.entClient.Client().DictType.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, cnt, "BatchDelete 后表内行数应为 0")
}
