package data

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	dictV1 "go-wind-admin/api/gen/go/dict/service/v1"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newLanguageRepoSqlite 用 enttest helper 构造一个可直接做 CRUD 的 LanguageRepo。
// 白盒构造逐字段复刻 NewLanguageRepo 的 mapper 初始化（Language 无枚举转换器），
// 仅将 log 换为 NopLogger、entClient 换为 SQLite 内存库测试 client。
func newLanguageRepoSqlite(t *testing.T) *LanguageRepo {
	t.Helper()
	entClient := enttest.NewEntClientForTest(t)
	repo := &LanguageRepo{
		entClient: entClient,
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		mapper:    mapper.NewCopierMapper[dictV1.Language, ent.Language](),
	}

	repo.init()

	return repo
}

// TestLanguageRepoSqlite_Create 端到端验证 LanguageRepo.Create 落库
// （请求体按 proto 约定包 Data 字段）。
func TestLanguageRepoSqlite_Create(t *testing.T) {
	repo := newLanguageRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	err := repo.Create(ctx, &dictV1.CreateLanguageRequest{
		Data: &dictV1.Language{
			LanguageName: trans.Ptr("sqlite测试语言"),
			LanguageCode: trans.Ptr(fmt.Sprintf("LANG_SQLITE_%d", 1001)),
			NativeName:   trans.Ptr("native-name-1001"),
		},
	})
	require.NoError(t, err, "通过 repo.Create 写入 SQLite 应成功")

	rows, err := repo.entClient.Client().Language.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1, "SQLite 中应有 1 条 language 记录")
	require.Equal(t, "sqlite测试语言", *rows[0].LanguageName, "language_name 应按 Create 载荷落库")
	require.Equal(t, fmt.Sprintf("LANG_SQLITE_%d", 1001), *rows[0].LanguageCode, "language_code 应按 Create 载荷落库")
}

// TestLanguageRepoSqlite_List 验证 LanguageRepo.List 的分页列表与
// contains 模糊搜索过滤语义（仓规：搜索条件一律 contains）。
func TestLanguageRepoSqlite_List(t *testing.T) {
	repo := newLanguageRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &dictV1.CreateLanguageRequest{
		Data: &dictV1.Language{
			LanguageName: trans.Ptr("MARKERALPHA 语言"),
			LanguageCode: trans.Ptr(fmt.Sprintf("LANG_SQLITE_%d", 2001)),
			NativeName:   trans.Ptr("native-alpha-2001"),
		},
	}))
	require.NoError(t, repo.Create(ctx, &dictV1.CreateLanguageRequest{
		Data: &dictV1.Language{
			LanguageName: trans.Ptr("MARKERBETA 语言"),
			LanguageCode: trans.Ptr(fmt.Sprintf("LANG_SQLITE_%d", 2002)),
			NativeName:   trans.Ptr("native-beta-2002"),
		},
	}))

	// 无过滤：应返回全部 2 条
	all, err := repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(2), all.Total, "无过滤时应统计全部 2 条")
	require.Len(t, all.Items, 2, "无过滤时应返回 2 条")

	// contains 过滤：仅命中名称含 MARKERALPHA 的那条
	filtered, err := repo.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_Query{
			Query: `{"language_name__contains":"MARKERALPHA"}`,
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), filtered.Total, "contains 过滤应只统计命中行")
	require.Len(t, filtered.Items, 1, "contains 过滤应只返回命中行")
	require.Contains(t, filtered.Items[0].GetLanguageName(), "MARKERALPHA", "命中行应为含标记的那条")
}

// TestLanguageRepoSqlite_Get 验证 LanguageRepo.Get 按主键查询的命中与未命中。
func TestLanguageRepoSqlite_Get(t *testing.T) {
	repo := newLanguageRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &dictV1.CreateLanguageRequest{
		Data: &dictV1.Language{
			LanguageName: trans.Ptr("sqlite查询语言"),
			LanguageCode: trans.Ptr(fmt.Sprintf("LANG_SQLITE_%d", 3001)),
			NativeName:   trans.Ptr("native-query-3001"),
		},
	}))

	rows, err := repo.entClient.Client().Language.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := uint32(rows[0].ID)

	// 命中：按主键
	gotByID, err := repo.Get(ctx, &dictV1.GetLanguageRequest{
		QueryBy: &dictV1.GetLanguageRequest_Id{Id: createdID},
	})
	require.NoError(t, err, "按主键查询已存在记录应命中")
	require.Equal(t, "sqlite查询语言", gotByID.GetLanguageName(), "命中记录的 language_name 应与写入一致")

	// 未命中：不存在的主键
	_, err = repo.Get(ctx, &dictV1.GetLanguageRequest{
		QueryBy: &dictV1.GetLanguageRequest_Id{Id: 99999},
	})
	require.Error(t, err, "查询不存在的主键应返回错误")
}

// TestLanguageRepoSqlite_Update 验证 LanguageRepo.Update 在单字段 updateMask 下
// 只更新掩码内字段，掩码外字段保持原值。
func TestLanguageRepoSqlite_Update(t *testing.T) {
	repo := newLanguageRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &dictV1.CreateLanguageRequest{
		Data: &dictV1.Language{
			LanguageName: trans.Ptr("更新前语言名"),
			LanguageCode: trans.Ptr(fmt.Sprintf("LANG_SQLITE_%d", 4001)),
			NativeName:   trans.Ptr("更新前本地名"),
		},
	}))

	rows, err := repo.entClient.Client().Language.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := uint32(rows[0].ID)

	// 注意：LanguageRepo.Update 的 mutate 回调对 language_name/native_name 使用
	// 非 nillable Set + NotEmpty 校验，掩码清掉任一字段都会以空串触发校验失败，
	// 因此可用的最小掩码必须同时覆盖两者；language_code 不在掩码内且更新路径根本不写它。
	err = repo.Update(ctx, &dictV1.UpdateLanguageRequest{
		Id: createdID,
		Data: &dictV1.Language{
			LanguageName: trans.Ptr("更新后语言名"),
			NativeName:   trans.Ptr("更新前本地名"),
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"language_name", "native_name"}},
	})
	require.NoError(t, err, "掩码更新应成功")

	after, err := repo.entClient.Client().Language.Query().Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "更新后语言名", *after.LanguageName, "掩码内字段 language_name 应被更新")
	require.Equal(t, "更新前本地名", *after.NativeName, "载荷未变更的字段 native_name 应保持原值")
	require.Equal(t, fmt.Sprintf("LANG_SQLITE_%d", 4001), *after.LanguageCode, "掩码外字段 language_code 应保持原值")
}

// TestLanguageRepoSqlite_Delete 验证 LanguageRepo.Delete 删除记录后表内计数归零。
func TestLanguageRepoSqlite_Delete(t *testing.T) {
	repo := newLanguageRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &dictV1.CreateLanguageRequest{
		Data: &dictV1.Language{
			LanguageName: trans.Ptr("待删除语言"),
			LanguageCode: trans.Ptr(fmt.Sprintf("LANG_SQLITE_%d", 5001)),
			NativeName:   trans.Ptr("native-delete-5001"),
		},
	}))

	rows, err := repo.entClient.Client().Language.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := uint32(rows[0].ID)

	require.NoError(t, repo.Delete(ctx, &dictV1.DeleteLanguageRequest{
		QueryBy: &dictV1.DeleteLanguageRequest_Id{Id: createdID},
	}), "删除已存在记录应成功")

	count, err := repo.entClient.Client().Language.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count, "删除后 language 表计数应归零")
}
