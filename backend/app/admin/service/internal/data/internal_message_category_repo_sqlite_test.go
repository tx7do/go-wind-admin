// InternalMessageCategoryRepo 的 SQLite 内存库测试（白盒，包内测试）。
//
// 消息分类是租户范围的平铺字典（无 parent_id、不做树形级联），测试盯四件事：
//   - (tenant_id, code) 组合唯一：同租户撞码必须被 DB 挡下，跨租户同码合法
//     （分类编码是各租户自己的引用锚，不是全局的）；
//   - Update 的 allow_missing 路径：资源不存在时改为建行，且把 UpdatedBy 挪成
//     CreatedBy（一次"补建"只有一个操作人，不该在 updated_by 里复制出第二个人）；
//   - 掩码语义：掩码内更新、掩码外原值；注意本仓的 updated_by 取自 req.Data.UpdatedBy，
//     要落库必须随掩码一起进来（与 template/monitor 仓的 operatorID 参数形态不同）；
//   - Delete 平铺语义：只删分类自身、不做级联；对不存在 ID 静默无效果。
package data

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"entgo.io/ent/dialect/sql"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	internalMessageV1 "go-wind-admin/api/gen/go/internal_message/service/v1"
	"go-wind-admin/app/admin/service/internal/data/ent/internalmessagecategory"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newCategory 构造一份合法分类载荷，字段可按需覆盖。
func newCategory(tenantID int64, code string) *internalMessageV1.InternalMessageCategory {
	return &internalMessageV1.InternalMessageCategory{
		TenantId:  trans.Ptr(tenantID),
		Name:      trans.Ptr("分类-" + code),
		Code:      trans.Ptr(code),
		IsEnabled: trans.Ptr(true),
		SortOrder: trans.Ptr(uint32(10)),
	}
}

// TestInternalMessageCategoryRepoSqlite_Create 验证字段级落库与读回：
// 租户 ID、编码、排序、启用、created_by 逐列命中；显式给 ID 的播种路径（默认数据
// 同步按指定 ID 建行）必须生效。
func TestInternalMessageCategoryRepoSqlite_Create(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewInternalMessageCategoryRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	err := repo.Create(ctx, &internalMessageV1.CreateInternalMessageCategoryRequest{
		Data: &internalMessageV1.InternalMessageCategory{
			TenantId:  trans.Ptr(int64(3)),
			Name:      trans.Ptr("系统公告"),
			Code:      trans.Ptr("announcement"),
			IconUrl:   trans.Ptr("https://cdn.example.com/ann.png"),
			SortOrder: trans.Ptr(uint32(5)),
			IsEnabled: trans.Ptr(true),
			CreatedBy: trans.Ptr(int64(1001)),
		},
	})
	require.NoError(t, err, "完整载荷的 Create 应成功")

	created, err := entClient.Client().InternalMessageCategory.Query().
		Where(internalmessagecategory.CodeEQ("announcement")).
		Only(ctx)
	require.NoError(t, err)
	require.NotNil(t, created.TenantID)
	require.Equal(t, int64(3), *created.TenantID, "租户 ID 应按载荷落库")
	require.NotNil(t, created.Name)
	require.Equal(t, "系统公告", *created.Name)
	require.NotNil(t, created.IconURL)
	require.Equal(t, "https://cdn.example.com/ann.png", *created.IconURL)
	require.NotNil(t, created.SortOrder)
	require.Equal(t, uint32(5), *created.SortOrder)
	require.NotNil(t, created.IsEnabled)
	require.True(t, *created.IsEnabled)
	require.NotNil(t, created.CreatedBy)
	require.Equal(t, int64(1001), *created.CreatedBy, "created_by 应按载荷落库")
	require.NotNil(t, created.CreatedAt, "created_at 必须显式写入：mixin 列没有 DB 默认值")

	// 读视图：Get 按命中回 DTO
	dto, err := repo.Get(ctx, &internalMessageV1.GetInternalMessageCategoryRequest{
		QueryBy: &internalMessageV1.GetInternalMessageCategoryRequest_Id{Id: created.ID},
	})
	require.NoError(t, err)
	require.Equal(t, created.ID, dto.GetId())
	require.Equal(t, "announcement", dto.GetCode())
	require.Equal(t, "系统公告", dto.GetName())
	require.Equal(t, int64(3), dto.GetTenantId())
	require.True(t, dto.GetIsEnabled())

	// 显式 ID 播种路径：Data.Id 给定时必须按它建行（系统默认数据同步依赖这条）
	err = repo.Create(ctx, &internalMessageV1.CreateInternalMessageCategoryRequest{
		Data: &internalMessageV1.InternalMessageCategory{
			Id:       trans.Ptr(int64(500)),
			TenantId: trans.Ptr(int64(0)),
			Name:     trans.Ptr("系统通知"),
			Code:     trans.Ptr("system"),
		},
	})
	require.NoError(t, err, "显式 ID 的播种建行应成功")
	exist, err := repo.IsExist(ctx, 500)
	require.NoError(t, err)
	require.True(t, exist, "显式 ID=500 的行应存在")

	// 入参守卫
	require.Error(t, repo.Create(ctx, nil), "nil 请求应返回 BadRequest")
	require.Error(t, repo.Create(ctx, &internalMessageV1.CreateInternalMessageCategoryRequest{}),
		"nil Data 应返回 BadRequest")
}

// TestInternalMessageCategoryRepoSqlite_UniqueTenantCode 验证 (tenant_id, code) 组合唯一：
// 同租户撞码被 DB 挡下（失败创建不得增加行数）；跨租户同码合法——
// 分类编码是各租户自己的引用锚，唯一性不能越出租户边界。
func TestInternalMessageCategoryRepoSqlite_UniqueTenantCode(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewInternalMessageCategoryRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	err := repo.Create(ctx, &internalMessageV1.CreateInternalMessageCategoryRequest{Data: newCategory(1, "alert")})
	require.NoError(t, err)

	// 同租户同 code：第二行必须被唯一索引拒绝
	err = repo.Create(ctx, &internalMessageV1.CreateInternalMessageCategoryRequest{Data: newCategory(1, "alert")})
	require.Error(t, err, "同租户重复 code 应被拒绝")
	require.Contains(t, err.Error(), "insert internal message category failed")

	cnt, err := entClient.Client().InternalMessageCategory.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, cnt, "失败的创建不该增加行数")

	// 跨租户同 code：合法
	err = repo.Create(ctx, &internalMessageV1.CreateInternalMessageCategoryRequest{Data: newCategory(2, "alert")})
	require.NoError(t, err, "不同租户复用同一 code 应合法")

	cnt, err = entClient.Client().InternalMessageCategory.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, cnt)
}

// TestInternalMessageCategoryRepoSqlite_Get 验证按 ID 命中、view_mask 局部投影、
// nil 请求 BadRequest、不存在 ID 回错。
func TestInternalMessageCategoryRepoSqlite_Get(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewInternalMessageCategoryRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	err := repo.Create(ctx, &internalMessageV1.CreateInternalMessageCategoryRequest{Data: newCategory(1, "billing")})
	require.NoError(t, err)
	row, err := entClient.Client().InternalMessageCategory.Query().
		Where(internalmessagecategory.CodeEQ("billing")).
		Only(ctx)
	require.NoError(t, err)

	// view_mask 局部投影：只要 name/code，其余字段如实缺失（列表卡片场景少拉大字段）
	masked, err := repo.Get(ctx, &internalMessageV1.GetInternalMessageCategoryRequest{
		QueryBy:  &internalMessageV1.GetInternalMessageCategoryRequest_Id{Id: row.ID},
		ViewMask: &fieldmaskpb.FieldMask{Paths: []string{"name", "code"}},
	})
	require.NoError(t, err, "view_mask 投影应正常返回")
	require.Equal(t, "billing", masked.GetCode())
	require.Equal(t, "分类-billing", masked.GetName())
	require.Nil(t, masked.IconUrl, "投影外字段应保持 nil")

	// 不存在的 ID：回错而非 (nil, nil)——本仓 Get 与 template 仓的 GetByCode 语义不同，
	// 走的是 repository.Get 的 Only 路径，没有行就是错误
	_, err = repo.Get(ctx, &internalMessageV1.GetInternalMessageCategoryRequest{
		QueryBy: &internalMessageV1.GetInternalMessageCategoryRequest_Id{Id: 9999999},
	})
	require.Error(t, err, "不存在的 ID 应返回 NotFound")

	// 入参守卫
	_, err = repo.Get(ctx, nil)
	require.Error(t, err, "nil 请求应返回 BadRequest")
}

// TestInternalMessageCategoryRepoSqlite_IsExistAndCount 验证存在性判断与计数：
// Count 无条件全表；带 whereCond 时按条件过滤（本仓 Count 是服务层分页 total 的来源，
// 条件计数错一分页就错）。
func TestInternalMessageCategoryRepoSqlite_IsExistAndCount(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewInternalMessageCategoryRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	// 空表：Count=0、IsExist=false
	cnt, err := repo.Count(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, 0, cnt, "空表计数应为 0")
	exist, err := repo.IsExist(ctx, 1)
	require.NoError(t, err)
	require.False(t, exist)

	err = repo.Create(ctx, &internalMessageV1.CreateInternalMessageCategoryRequest{Data: newCategory(1, "alpha")})
	require.NoError(t, err)
	err = repo.Create(ctx, &internalMessageV1.CreateInternalMessageCategoryRequest{Data: newCategory(1, "beta")})
	require.NoError(t, err)
	err = repo.Create(ctx, &internalMessageV1.CreateInternalMessageCategoryRequest{Data: newCategory(2, "gamma")})
	require.NoError(t, err)

	row, err := entClient.Client().InternalMessageCategory.Query().
		Where(internalmessagecategory.CodeEQ("alpha")).
		Only(ctx)
	require.NoError(t, err)

	exist, err = repo.IsExist(ctx, row.ID)
	require.NoError(t, err)
	require.True(t, exist, "存在的 ID 应命中")
	exist, err = repo.IsExist(ctx, 9999999)
	require.NoError(t, err)
	require.False(t, exist, "不存在的 ID 应如实为 false 而非报错")

	cnt, err = repo.Count(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, 3, cnt)

	// 条件计数：只数租户 1 的两行
	cnt, err = repo.Count(ctx, []func(*sql.Selector){
		func(s *sql.Selector) { s.Where(sql.EQ(internalmessagecategory.FieldTenantID, 1)) },
	})
	require.NoError(t, err)
	require.Equal(t, 2, cnt, "条件计数应只统计租户 1 的行")

	// 条件计数：按 code 精确命中
	cnt, err = repo.Count(ctx, []func(*sql.Selector){
		func(s *sql.Selector) { s.Where(sql.EQ(internalmessagecategory.FieldCode, "gamma")) },
	})
	require.NoError(t, err)
	require.Equal(t, 1, cnt)
}

// TestInternalMessageCategoryRepoSqlite_ListAndListByIds 验证分页 total、nil 请求
// BadRequest，以及 ListCategoriesByIds 的空入参与部分命中语义（批量查分类挂名，
// 查不到的 ID 静默跳过，调用方只拿到存在的行）。
func TestInternalMessageCategoryRepoSqlite_ListAndListByIds(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewInternalMessageCategoryRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	err := repo.Create(ctx, &internalMessageV1.CreateInternalMessageCategoryRequest{Data: newCategory(1, "one")})
	require.NoError(t, err)
	err = repo.Create(ctx, &internalMessageV1.CreateInternalMessageCategoryRequest{Data: newCategory(1, "two")})
	require.NoError(t, err)

	all, err := repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, int64(2), all.GetTotal())
	require.Len(t, all.GetItems(), 2)

	_, err = repo.List(ctx, nil)
	require.Error(t, err, "nil 分页请求应返回 BadRequest")

	// 空入参：返回空切片而非 nil/错误（批量路径的常规短路）
	got, err := repo.ListCategoriesByIds(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Empty(t, got, "空 ID 列表应返回空切片")

	rowOne, err := entClient.Client().InternalMessageCategory.Query().
		Where(internalmessagecategory.CodeEQ("one")).
		Only(ctx)
	require.NoError(t, err)

	// 部分命中：只回存在的行
	got, err = repo.ListCategoriesByIds(ctx, []int64{rowOne.ID, 9999999})
	require.NoError(t, err)
	require.Len(t, got, 1, "不存在的 ID 应被跳过")
	require.Equal(t, "one", got[0].GetCode())
}

// TestInternalMessageCategoryRepoSqlite_Update 验证掩码语义与 allow_missing 补建：
//   - 掩码内更新、掩码外（code/is_enabled）保持原值；updated_by 取自 req.Data.UpdatedBy，
//     必须随掩码一起进来才落库（与 operatorID 参数形态的仓不同，掩码外会被滤掉）；
//   - allow_missing=true 且行不存在：转为建行，UpdatedBy 挪成 CreatedBy、updated_by 不落；
//   - allow_missing=true 且行存在：走普通更新，不多出一行。
func TestInternalMessageCategoryRepoSqlite_Update(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewInternalMessageCategoryRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	err := repo.Create(ctx, &internalMessageV1.CreateInternalMessageCategoryRequest{
		Data: &internalMessageV1.InternalMessageCategory{
			TenantId:  trans.Ptr(int64(1)),
			Name:      trans.Ptr("原名"),
			Code:      trans.Ptr("up_code"),
			IsEnabled: trans.Ptr(true),
			SortOrder: trans.Ptr(uint32(1)),
		},
	})
	require.NoError(t, err)
	row, err := entClient.Client().InternalMessageCategory.Query().
		Where(internalmessagecategory.CodeEQ("up_code")).
		Only(ctx)
	require.NoError(t, err)

	// 改名与排序（掩码内），code/is_enabled 保持原值；updated_by 随掩码进来
	require.NoError(t, repo.Update(ctx, &internalMessageV1.UpdateInternalMessageCategoryRequest{
		Id: row.ID,
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{"name", "sort_order", "updated_by"},
		},
		Data: &internalMessageV1.InternalMessageCategory{
			Name:      trans.Ptr("新名"),
			SortOrder: trans.Ptr(uint32(9)),
			UpdatedBy: trans.Ptr(int64(2001)),
			Code:      trans.Ptr("masked-out-code"),
			IsEnabled: trans.Ptr(false), // 掩码外
		},
	}))
	updated, err := entClient.Client().InternalMessageCategory.Get(ctx, row.ID)
	require.NoError(t, err)
	require.Equal(t, "新名", *updated.Name, "掩码内的名称应更新")
	require.Equal(t, uint32(9), *updated.SortOrder, "掩码内的排序应更新")
	require.Equal(t, "up_code", *updated.Code, "掩码外的 code 应保持原值")
	require.True(t, *updated.IsEnabled, "掩码外的 is_enabled 应保持原值")
	require.NotNil(t, updated.UpdatedBy)
	require.Equal(t, int64(2001), *updated.UpdatedBy, "updated_by 应按载荷落库")
	require.NotNil(t, updated.UpdatedAt, "更新应写入 updated_at")

	// allow_missing=true + 不存在的 ID：转为建行（服务层 upsert 语义的落点）
	err = repo.Update(ctx, &internalMessageV1.UpdateInternalMessageCategoryRequest{
		Id: 888,
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{"name", "code", "updated_by"},
		},
		Data: &internalMessageV1.InternalMessageCategory{
			TenantId:  trans.Ptr(int64(4)),
			Name:      trans.Ptr("补建分类"),
			Code:      trans.Ptr("backfill"),
			UpdatedBy: trans.Ptr(int64(77)),
		},
		AllowMissing: trans.Ptr(true),
	})
	require.NoError(t, err, "allow_missing 的补建应成功")
	backfilled, err := entClient.Client().InternalMessageCategory.Query().
		Where(internalmessagecategory.CodeEQ("backfill")).
		Only(ctx)
	require.NoError(t, err)
	require.NotNil(t, backfilled.CreatedBy)
	require.Equal(t, int64(77), *backfilled.CreatedBy,
		"补建时 UpdatedBy 应挪成 CreatedBy：一次补建只有一个操作人")
	require.Nil(t, backfilled.UpdatedBy, "补建不是更新，updated_by 不该落值")
	require.Equal(t, int64(4), *backfilled.TenantID)

	// allow_missing=true + 存在的 ID：普通更新，不多出一行
	err = repo.Update(ctx, &internalMessageV1.UpdateInternalMessageCategoryRequest{
		Id: row.ID,
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{"name", "updated_by"},
		},
		Data: &internalMessageV1.InternalMessageCategory{
			Name:      trans.Ptr("再改一次"),
			UpdatedBy: trans.Ptr(int64(2002)),
		},
		AllowMissing: trans.Ptr(true),
	})
	require.NoError(t, err)
	updated, err = entClient.Client().InternalMessageCategory.Get(ctx, row.ID)
	require.NoError(t, err)
	require.Equal(t, "再改一次", *updated.Name)
	cnt, err := entClient.Client().InternalMessageCategory.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, cnt, "对存在行的 allow_missing 更新不应多出一行")

	// 入参守卫
	require.Error(t, repo.Update(ctx, nil), "nil 请求应返回 BadRequest")
	require.Error(t, repo.Update(ctx, &internalMessageV1.UpdateInternalMessageCategoryRequest{}),
		"nil Data 应返回 BadRequest")
	require.Error(t, repo.Update(ctx, &internalMessageV1.UpdateInternalMessageCategoryRequest{
		Id:   0,
		Data: &internalMessageV1.InternalMessageCategory{},
	}), "id=0 应返回 BadRequest")
}

// TestInternalMessageCategoryRepoSqlite_Delete 验证平铺删除语义：只删分类自身、
// 不做树形级联；nil 请求 BadRequest；对不存在 ID 静默无效果（Delete 走批量路径，
// 0 行受影响不是错误）。
func TestInternalMessageCategoryRepoSqlite_Delete(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewInternalMessageCategoryRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	err := repo.Create(ctx, &internalMessageV1.CreateInternalMessageCategoryRequest{Data: newCategory(1, "del_a")})
	require.NoError(t, err)
	err = repo.Create(ctx, &internalMessageV1.CreateInternalMessageCategoryRequest{Data: newCategory(1, "del_b")})
	require.NoError(t, err)

	rowA, err := entClient.Client().InternalMessageCategory.Query().
		Where(internalmessagecategory.CodeEQ("del_a")).
		Only(ctx)
	require.NoError(t, err)

	require.NoError(t, repo.Delete(ctx, &internalMessageV1.DeleteInternalMessageCategoryRequest{
		QueryBy: &internalMessageV1.DeleteInternalMessageCategoryRequest_Id{Id: rowA.ID},
	}), "按存在 ID 删除应成功")

	exist, err := repo.IsExist(ctx, rowA.ID)
	require.NoError(t, err)
	require.False(t, exist, "删除后该行应不存在")

	cnt, err := entClient.Client().InternalMessageCategory.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, cnt, "平铺结构只删目标行，不多不少")

	// 不存在 ID：静默无效果（0 行受影响不是错误）
	require.NoError(t, repo.Delete(ctx, &internalMessageV1.DeleteInternalMessageCategoryRequest{
		QueryBy: &internalMessageV1.DeleteInternalMessageCategoryRequest_Id{Id: 9999999},
	}), "对不存在 ID 的删除应静默无效果")
	cnt, err = entClient.Client().InternalMessageCategory.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, cnt, "空删除不应影响行数")

	// 入参守卫
	require.Error(t, repo.Delete(ctx, nil), "nil 请求应返回 BadRequest")
}
