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
	"go-wind-admin/app/admin/service/internal/data/ent/dictentry"
	"go-wind-admin/app/admin/service/internal/data/ent/dictentryi18n"
	"go-wind-admin/app/admin/service/internal/data/ent/dicttype"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newDictEntryRepoSqlite 在给定 enttest client 上白盒构造 DictEntryRepo。
// 与生产 NewDictEntryRepo 一致：内含同库的 DictEntryI18nRepo 依赖与 mapper 初始化。
func newDictEntryRepoSqlite(t *testing.T, entClient *entCrud.EntClient[*ent.Client]) *DictEntryRepo {
	t.Helper()
	i18nRepo := &DictEntryI18nRepo{
		entClient: entClient,
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		mapper:    mapper.NewCopierMapper[dictV1.DictEntryI18N, ent.DictEntryI18n](),
	}
	i18nRepo.init()
	repo := &DictEntryRepo{
		entClient: entClient,
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		mapper:    mapper.NewCopierMapper[dictV1.DictEntry, ent.DictEntry](),
		i18n:      i18nRepo,
	}
	repo.init()
	return repo
}

// TestDictEntryRepoSqlite_Create 带父 dict_type 创建字典项，
// 直查断言：行落库、且 (entry → type) 外键真实落库到请求指定的父行。
// 历史上这里有过"边与 TypeId 条件写反"导致 type_id 永远为 NULL 的 bug，本用例为其回归测试。
func TestDictEntryRepoSqlite_Create(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newDictEntryRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	parent, err := entClient.Client().DictType.Create().
		SetNillableTypeCode(trans.Ptr("sqlite_de_create_type")).
		SetNillableTypeName(trans.Ptr("父类型-创建")).
		Save(ctx)
	require.NoError(t, err, "直建父 dict_type 应成功")

	err = repo.Create(ctx, &dictV1.CreateDictEntryRequest{
		Data: &dictV1.DictEntry{
			TypeId:     &parent.ID,
			EntryValue: trans.Ptr("sqlite_de_create_value"),
		},
	})
	require.NoError(t, err, "repo.Create 应写入 SQLite 成功")

	rows, err := entClient.Client().DictEntry.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1, "SQLite 中应有 1 条 dict_entry 记录")
	require.Equal(t, "sqlite_de_create_value", *rows[0].EntryValue, "entry_value 应按请求落库")

	// 外键回归断言：该 entry 必须挂在请求指定的父类型上
	linked, err := entClient.Client().DictEntry.Query().
		Where(dictentry.HasDictTypeWith(dicttype.IDEQ(parent.ID))).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, linked, "entry 的 dict_type 外键应指向请求的父类型，而非 NULL")
}

// TestDictEntryRepoSqlite_ListContainsFilter 验证 List 的 contains 模糊搜索语义，
// 以及列表路径上 TypeId 从父类型边正确回填。
func TestDictEntryRepoSqlite_ListContainsFilter(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newDictEntryRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	parent, err := entClient.Client().DictType.Create().
		SetNillableTypeCode(trans.Ptr("sqlite_de_list_type")).
		SetNillableTypeName(trans.Ptr("父类型-列表")).
		Save(ctx)
	require.NoError(t, err)

	require.NoError(t, repo.Create(ctx, &dictV1.CreateDictEntryRequest{
		Data: &dictV1.DictEntry{
			TypeId:     &parent.ID,
			EntryValue: trans.Ptr("列表-markerasd-甲"),
		},
	}))
	require.NoError(t, repo.Create(ctx, &dictV1.CreateDictEntryRequest{
		Data: &dictV1.DictEntry{
			TypeId:     &parent.ID,
			EntryValue: trans.Ptr("列表-无关行-乙"),
		},
	}))

	// contains 过滤：只命中携带标记的行
	filtered, err := repo.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_FilterExpr{
			FilterExpr: &paginationV1.FilterExpr{
				Type: paginationV1.ExprType_AND,
				Conditions: []*paginationV1.FilterCondition{
					{
						Field:      "entry_value",
						Op:         paginationV1.Operator_CONTAINS,
						ValueOneof: &paginationV1.FilterCondition_Value{Value: "markerasd"},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), filtered.Total, "contains 过滤后 Total 应为 1")
	require.Len(t, filtered.Items, 1, "contains 过滤后只应返回 1 行")
	require.Contains(t, *filtered.Items[0].EntryValue, "markerasd", "命中行应是携带标记的行")

	// 列表路径的 TypeId 回填：应等于父类型 ID（边加载，而非 NULL）
	require.NotNil(t, filtered.Items[0].TypeId, "列表项应回填父类型 ID")
	require.Equal(t, parent.ID, *filtered.Items[0].TypeId, "TypeId 应等于父类型 ID")

	// 无过滤：两行全部返回
	all, err := repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(2), all.Total)
	require.Len(t, all.Items, 2)
	for _, item := range all.Items {
		require.Equal(t, parent.ID, *item.TypeId, "所有条目的 TypeId 均应回填父类型 ID")
	}
}

// TestDictEntryRepoSqlite_Get 验证按 ID 与按 entry_value 的命中/未命中。
func TestDictEntryRepoSqlite_Get(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newDictEntryRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	parent, err := entClient.Client().DictType.Create().
		SetNillableTypeCode(trans.Ptr("sqlite_de_get_type")).
		SetNillableTypeName(trans.Ptr("父类型-Get")).
		Save(ctx)
	require.NoError(t, err)

	require.NoError(t, repo.Create(ctx, &dictV1.CreateDictEntryRequest{
		Data: &dictV1.DictEntry{
			TypeId:     &parent.ID,
			EntryValue: trans.Ptr("sqlite_de_get_value"),
		},
	}))
	rows, err := entClient.Client().DictEntry.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := rows[0].ID

	// 命中：按 ID
	hit, err := repo.Get(ctx, &dictV1.GetDictEntryRequest{
		QueryBy: &dictV1.GetDictEntryRequest_Id{Id: createdID},
	})
	require.NoError(t, err)
	require.Equal(t, createdID, hit.GetId())
	require.Equal(t, "sqlite_de_get_value", hit.GetEntryValue())

	// 未命中：不存在的 ID
	_, err = repo.Get(ctx, &dictV1.GetDictEntryRequest{
		QueryBy: &dictV1.GetDictEntryRequest_Id{Id: 9999999},
	})
	require.Error(t, err, "不存在的 ID 查询应返回错误")

	// 命中：按 entry_value
	byValue, err := repo.Get(ctx, &dictV1.GetDictEntryRequest{
		QueryBy: &dictV1.GetDictEntryRequest_Value{Value: "sqlite_de_get_value"},
	})
	require.NoError(t, err, "按存在的 entry_value 查询应命中")
	require.Equal(t, "sqlite_de_get_value", byValue.GetEntryValue())

	// 未命中：不存在的 entry_value
	_, err = repo.Get(ctx, &dictV1.GetDictEntryRequest{
		QueryBy: &dictV1.GetDictEntryRequest_Value{Value: "no-such-entry-value-zzz"},
	})
	require.Error(t, err, "不存在的 entry_value 查询应返回错误")
}

// TestDictEntryRepoSqlite_Update 验证 Update 掩码内字段更新、掩码外字段保持原值、
// 父类型外键不受影响。
func TestDictEntryRepoSqlite_Update(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newDictEntryRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	parent, err := entClient.Client().DictType.Create().
		SetNillableTypeCode(trans.Ptr("sqlite_de_update_type")).
		SetNillableTypeName(trans.Ptr("父类型-更新")).
		Save(ctx)
	require.NoError(t, err)

	require.NoError(t, repo.Create(ctx, &dictV1.CreateDictEntryRequest{
		Data: &dictV1.DictEntry{
			TypeId:     &parent.ID,
			EntryValue: trans.Ptr("更新前-entry-value"),
		},
	}))
	rows, err := entClient.Client().DictEntry.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := rows[0].ID

	err = repo.Update(ctx, &dictV1.UpdateDictEntryRequest{
		Id:         createdID,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"numeric_value"}},
		Data: &dictV1.DictEntry{
			NumericValue: trans.Ptr(int32(777)),
		},
	})
	require.NoError(t, err, "更新 numeric_value 应成功")

	after, err := entClient.Client().DictEntry.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, after, 1)
	require.NotNil(t, after[0].NumericValue, "掩码内字段 numeric_value 应被更新")
	require.Equal(t, int32(777), *after[0].NumericValue, "numeric_value 应更新为请求值")
	require.Equal(t, "更新前-entry-value", *after[0].EntryValue, "掩码外字段 entry_value 应保持原值")

	// 外键不受更新影响
	linked, err := entClient.Client().DictEntry.Query().
		Where(dictentry.HasDictTypeWith(dicttype.IDEQ(parent.ID))).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, linked, "更新不应改变 entry 的父类型外键")
}

// TestDictEntryRepoSqlite_UpdateI18nReplace 验证 Update 携带 i18n 时走
// ReplaceByEntryID：旧行被清理、新语言集合整体写入。
func TestDictEntryRepoSqlite_UpdateI18nReplace(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newDictEntryRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	parent, err := entClient.Client().DictType.Create().
		SetNillableTypeCode(trans.Ptr("sqlite_de_updi18n_type")).
		SetNillableTypeName(trans.Ptr("父类型-i18n替换")).
		Save(ctx)
	require.NoError(t, err)

	// 创建时带 zh-CN 翻译
	require.NoError(t, repo.Create(ctx, &dictV1.CreateDictEntryRequest{
		Data: &dictV1.DictEntry{
			TypeId:     &parent.ID,
			EntryValue: trans.Ptr("sqlite_de_updi18n_value"),
			I18N: map[string]*dictV1.DictEntryI18N{
				"zh-CN": {EntryLabel: "初始标签"},
			},
		},
	}))
	rows, err := entClient.Client().DictEntry.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	entryID := rows[0].ID

	i18nRows, err := entClient.Client().DictEntryI18n.Query().
		Where(dictentryi18n.HasDictEntryWith(dictentry.IDEQ(entryID))).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, i18nRows, 1, "创建后应有 1 条 zh-CN 翻译")
	require.Equal(t, "zh-CN", *i18nRows[0].LanguageCode)

	// 更新：i18n 换成 en-US 集合，同时更新 numeric_value
	err = repo.Update(ctx, &dictV1.UpdateDictEntryRequest{
		Id:         entryID,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"numeric_value", "i18n"}},
		Data: &dictV1.DictEntry{
			NumericValue: trans.Ptr(int32(11)),
			I18N: map[string]*dictV1.DictEntryI18N{
				"en-US": {EntryLabel: "replaced-label"},
			},
		},
	})
	require.NoError(t, err, "携带 i18n 的更新应走 ReplaceByEntryID 成功")

	// entry 主表：numeric_value 已更新
	after, err := entClient.Client().DictEntry.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, after, 1)
	require.NotNil(t, after[0].NumericValue)
	require.Equal(t, int32(11), *after[0].NumericValue)

	// i18n：zh-CN 被清理，只余 en-US
	i18nAfter, err := entClient.Client().DictEntryI18n.Query().
		Where(dictentryi18n.HasDictEntryWith(dictentry.IDEQ(entryID))).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, i18nAfter, 1, "ReplaceByEntryID 后只应存在新集合的行")
	require.Equal(t, "en-US", *i18nAfter[0].LanguageCode, "残留行应为新集合的语言")
}

// TestDictEntryRepoSqlite_Delete 验证 Delete / BatchDelete 后行数归零。
func TestDictEntryRepoSqlite_Delete(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newDictEntryRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	parent, err := entClient.Client().DictType.Create().
		SetNillableTypeCode(trans.Ptr("sqlite_de_del_type")).
		SetNillableTypeName(trans.Ptr("父类型-删除")).
		Save(ctx)
	require.NoError(t, err)

	require.NoError(t, repo.Create(ctx, &dictV1.CreateDictEntryRequest{
		Data: &dictV1.DictEntry{
			TypeId:     &parent.ID,
			EntryValue: trans.Ptr("待删除-甲"),
		},
	}))
	rows, err := entClient.Client().DictEntry.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.NoError(t, repo.Delete(ctx, rows[0].ID), "Delete 应成功")
	cnt, err := entClient.Client().DictEntry.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, cnt, "Delete 后表内行数应为 0")

	require.NoError(t, repo.Create(ctx, &dictV1.CreateDictEntryRequest{
		Data: &dictV1.DictEntry{
			TypeId:     &parent.ID,
			EntryValue: trans.Ptr("待删除-乙"),
		},
	}))
	require.NoError(t, repo.Create(ctx, &dictV1.CreateDictEntryRequest{
		Data: &dictV1.DictEntry{
			TypeId:     &parent.ID,
			EntryValue: trans.Ptr("待删除-丙"),
		},
	}))
	rows, err = entClient.Client().DictEntry.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.NoError(t, repo.BatchDelete(ctx, []uint32{rows[0].ID, rows[1].ID}), "BatchDelete 应成功")
	cnt, err = entClient.Client().DictEntry.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, cnt, "BatchDelete 后表内行数应为 0")
}

// TestDictEntryRepoSqlite_ListByTypeCode 验证按类型编码列出的过滤语义：
// 只返回该类型下的启用条目，其他类型与禁用条目不得混入。
func TestDictEntryRepoSqlite_ListByTypeCode(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newDictEntryRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	typeA, err := entClient.Client().DictType.Create().
		SetNillableTypeCode(trans.Ptr("sqlite_de_ltc_a")).
		SetNillableTypeName(trans.Ptr("父类型-A")).
		Save(ctx)
	require.NoError(t, err)
	typeB, err := entClient.Client().DictType.Create().
		SetNillableTypeCode(trans.Ptr("sqlite_de_ltc_b")).
		SetNillableTypeName(trans.Ptr("父类型-B")).
		Save(ctx)
	require.NoError(t, err)

	// A 下：一条启用、一条禁用；B 下：一条启用
	require.NoError(t, repo.Create(ctx, &dictV1.CreateDictEntryRequest{
		Data: &dictV1.DictEntry{
			TypeId:     &typeA.ID,
			EntryValue: trans.Ptr("A-启用条目"),
			IsEnabled:  trans.Ptr(true),
		},
	}))
	require.NoError(t, repo.Create(ctx, &dictV1.CreateDictEntryRequest{
		Data: &dictV1.DictEntry{
			TypeId:     &typeA.ID,
			EntryValue: trans.Ptr("A-禁用条目"),
			IsEnabled:  trans.Ptr(false),
		},
	}))
	require.NoError(t, repo.Create(ctx, &dictV1.CreateDictEntryRequest{
		Data: &dictV1.DictEntry{
			TypeId:     &typeB.ID,
			EntryValue: trans.Ptr("B-启用条目"),
			IsEnabled:  trans.Ptr(true),
		},
	}))

	// 按类型 A 列出：只有 A 的启用条目
	resA, err := repo.ListByTypeCode(ctx, &dictV1.ListDictEntryByTypeCodeRequest{
		TypeCode: "sqlite_de_ltc_a",
	})
	require.NoError(t, err)
	require.Len(t, resA.Items, 1, "类型 A 只应返回其启用条目")
	require.Equal(t, "A-启用条目", *resA.Items[0].EntryValue, "返回的应是 A 的启用条目")
	require.Equal(t, typeA.ID, resA.Items[0].GetTypeId(), "TypeId 应回填为父类型 A 的 ID（与通用 List 一致）")

	// 按类型 B 列出：B 的启用条目
	resB, err := repo.ListByTypeCode(ctx, &dictV1.ListDictEntryByTypeCodeRequest{
		TypeCode: "sqlite_de_ltc_b",
	})
	require.NoError(t, err)
	require.Len(t, resB.Items, 1)
	require.Equal(t, "B-启用条目", *resB.Items[0].EntryValue)
	require.Equal(t, typeB.ID, resB.Items[0].GetTypeId(), "TypeId 应回填为父类型 B 的 ID（与通用 List 一致）")

	// 不存在的类型编码：空
	resNone, err := repo.ListByTypeCode(ctx, &dictV1.ListDictEntryByTypeCodeRequest{
		TypeCode: "no-such-type-code-zzz",
	})
	require.NoError(t, err)
	require.Empty(t, resNone.Items, "不存在的类型编码应返回空列表")
}
