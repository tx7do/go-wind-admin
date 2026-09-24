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

	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/permission"
	"go-wind-admin/app/admin/service/internal/data/ent/permissionapi"
	"go-wind-admin/app/admin/service/internal/data/ent/permissionmenu"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newPermissionRepoSqlite 在给定 enttest client 上白盒构造 PermissionRepo。
// 与生产 NewPermissionRepo 一致：内含同库的 PermissionApiRepo / PermissionMenuRepo
// 依赖（二者为无 init() 的简单构造）与 mapper/converter 初始化。
func newPermissionRepoSqlite(t *testing.T, entClient *entCrud.EntClient[*ent.Client]) *PermissionRepo {
	t.Helper()
	permissionApiRepo := &PermissionApiRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
	}
	permissionMenuRepo := &PermissionMenuRepo{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		entClient: entClient,
	}
	repo := &PermissionRepo{
		entClient: entClient,
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		mapper:    mapper.NewCopierMapper[permissionV1.Permission, ent.Permission](),
		statusConverter: mapper.NewEnumTypeConverter[permissionV1.Permission_Status, permission.Status](
			permissionV1.Permission_Status_name, permissionV1.Permission_Status_value,
		),
		permissionApiRepo:  permissionApiRepo,
		permissionMenuRepo: permissionMenuRepo,
	}
	repo.init()
	return repo
}

// TestPermissionRepoSqlite_Create 通过 repo.Create 写入后直查 SQLite 断言落库。
func TestPermissionRepoSqlite_Create(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newPermissionRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	err := repo.Create(ctx, &permissionV1.CreatePermissionRequest{
		Data: &permissionV1.Permission{
			Name:        trans.Ptr("sqlite权限点-创建"),
			Code:        trans.Ptr("sqlite_perm_create_code"),
			Status:      permissionV1.Permission_ON.Enum(),
			Description: trans.Ptr("创建用途描述"),
		},
	})
	require.NoError(t, err, "repo.Create 应写入 SQLite 成成功")

	rows, err := repo.entClient.Client().Permission.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1, "SQLite 中应有 1 条 permission 记录")
	require.Equal(t, "sqlite权限点-创建", *rows[0].Name, "name 应按请求落库")
	require.Equal(t, "sqlite_perm_create_code", *rows[0].Code, "code 应按请求落库")
	require.Equal(t, permission.StatusOn, *rows[0].Status, "status 枚举应经 converter 落为 ON")
	require.Equal(t, "创建用途描述", *rows[0].Description, "description 应按请求落库")
}

// TestPermissionRepoSqlite_ListContainsFilter 验证 List 的 contains 模糊搜索语义。
func TestPermissionRepoSqlite_ListContainsFilter(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newPermissionRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &permissionV1.CreatePermissionRequest{
		Data: &permissionV1.Permission{
			Name: trans.Ptr("权限-markerjkl-甲"),
			Code: trans.Ptr("sqlite_perm_list_a"),
		},
	}))
	require.NoError(t, repo.Create(ctx, &permissionV1.CreatePermissionRequest{
		Data: &permissionV1.Permission{
			Name: trans.Ptr("权限-无关行-乙"),
			Code: trans.Ptr("sqlite_perm_list_b"),
		},
	}))

	filtered, err := repo.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_FilterExpr{
			FilterExpr: &paginationV1.FilterExpr{
				Type: paginationV1.ExprType_AND,
				Conditions: []*paginationV1.FilterCondition{
					{
						Field:      "name",
						Op:         paginationV1.Operator_CONTAINS,
						ValueOneof: &paginationV1.FilterCondition_Value{Value: "markerjkl"},
					},
				},
			},
		},
	}, nil)
	require.NoError(t, err)
	require.Equal(t, uint64(1), filtered.Total, "contains 过滤后 Total 应为 1")
	require.Len(t, filtered.Items, 1, "contains 过滤后只应返回 1 行")
	require.Contains(t, *filtered.Items[0].Name, "markerjkl", "命中行应是携带标记的行")

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
	}, nil)
	require.NoError(t, err)
	require.Equal(t, uint64(0), none.Total, "无命中 contains 应返回 Total=0")
	require.Empty(t, none.Items)

	all, err := repo.List(ctx, &paginationV1.PagingRequest{}, nil)
	require.NoError(t, err)
	require.Equal(t, uint64(2), all.Total, "无过滤时应返回全部 2 行")
	require.Len(t, all.Items, 2)
	// 列表读视图：status 未显式指定、按列默认（ON）落库，经 queryEnumsAndBackfill
	// 如实回显。
	for _, item := range all.Items {
		require.Equal(t, permissionV1.Permission_ON, item.GetStatus(), "列表读视图应回填列默认 status")
	}
}

// TestPermissionRepoSqlite_Get 验证按 ID 与按 code 的命中/未命中。
func TestPermissionRepoSqlite_Get(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newPermissionRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &permissionV1.CreatePermissionRequest{
		Data: &permissionV1.Permission{
			Name: trans.Ptr("sqlite权限点-Get"),
			Code: trans.Ptr("sqlite_perm_get_code"),
		},
	}))
	rows, err := repo.entClient.Client().Permission.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := rows[0].ID

	// 命中：按 ID
	hit, err := repo.Get(ctx, &permissionV1.GetPermissionRequest{
		QueryBy: &permissionV1.GetPermissionRequest_Id{Id: createdID},
	})
	require.NoError(t, err, "按存在的 ID 查询应命中")
	require.Equal(t, createdID, hit.GetId())
	// 读视图（Get 按主键）：status 未显式指定、按列默认（ON）落库，经
	// queryEnumsAndBackfill 如实回显。
	require.Equal(t, permissionV1.Permission_ON, hit.GetStatus(), "读视图应回填列默认 status")

	// 命中：按 code
	byCode, err := repo.Get(ctx, &permissionV1.GetPermissionRequest{
		QueryBy: &permissionV1.GetPermissionRequest_Code{Code: "sqlite_perm_get_code"},
	})
	require.NoError(t, err, "按存在的 code 查询应命中")
	require.Equal(t, createdID, byCode.GetId(), "按 code 命中应带回同一行的 ID")
	// 读视图（Get 按 code）：同上，列默认 status 如实回显。
	require.Equal(t, permissionV1.Permission_ON, byCode.GetStatus(), "读视图应回填列默认 status")

	// 未命中：不存在的 ID / 不存在的 code
	_, err = repo.Get(ctx, &permissionV1.GetPermissionRequest{
		QueryBy: &permissionV1.GetPermissionRequest_Id{Id: 9999999},
	})
	require.Error(t, err, "不存在的 ID 查询应返回错误")
	_, err = repo.Get(ctx, &permissionV1.GetPermissionRequest{
		QueryBy: &permissionV1.GetPermissionRequest_Code{Code: "no-such-code-zzz"},
	})
	require.Error(t, err, "不存在的 code 查询应返回错误")
}

// TestPermissionRepoSqlite_CodesAndIdsLookup 验证 ID↔code 双向映射查询。
func TestPermissionRepoSqlite_CodesAndIdsLookup(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newPermissionRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &permissionV1.CreatePermissionRequest{
		Data: &permissionV1.Permission{
			Name: trans.Ptr("映射-甲"),
			Code: trans.Ptr("sqlite_perm_lookup_a"),
		},
	}))
	require.NoError(t, repo.Create(ctx, &permissionV1.CreatePermissionRequest{
		Data: &permissionV1.Permission{
			Name: trans.Ptr("映射-乙"),
			Code: trans.Ptr("sqlite_perm_lookup_b"),
		},
	}))
	rows, err := repo.entClient.Client().Permission.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	idByCode := map[string]uint32{}
	for _, row := range rows {
		idByCode[*row.Code] = row.ID
	}

	codes, err := repo.GetPermissionCodesByIDs(ctx, []uint32{idByCode["sqlite_perm_lookup_a"], idByCode["sqlite_perm_lookup_b"]})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"sqlite_perm_lookup_a", "sqlite_perm_lookup_b"}, codes, "按 ID 列表应取回全部对应 code")

	ids, err := repo.GetPermissionIDsByCodes(ctx, []string{"sqlite_perm_lookup_a"})
	require.NoError(t, err)
	require.Equal(t, []uint32{idByCode["sqlite_perm_lookup_a"]}, ids, "按 code 列表应取回对应 ID")

	emptyCodes, err := repo.GetPermissionCodesByIDs(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, emptyCodes, "空 ID 列表应返回空")
}

// TestPermissionRepoSqlite_Update 验证 Update 只更新掩码内字段（description），
// 掩码外字段（name/code/status）保持原值。
func TestPermissionRepoSqlite_Update(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newPermissionRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &permissionV1.CreatePermissionRequest{
		Data: &permissionV1.Permission{
			Name:        trans.Ptr("sqlite权限点-更新"),
			Code:        trans.Ptr("sqlite_perm_update_code"),
			Description: trans.Ptr("更新前描述"),
		},
	}))
	rows, err := repo.entClient.Client().Permission.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := rows[0].ID

	err = repo.Update(ctx, &permissionV1.UpdatePermissionRequest{
		Id:         createdID,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description"}},
		Data: &permissionV1.Permission{
			Description: trans.Ptr("更新后描述-sqlite"),
		},
	})
	require.NoError(t, err, "更新 description 应成功")

	after, err := repo.entClient.Client().Permission.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, after, 1)
	require.Equal(t, "更新后描述-sqlite", *after[0].Description, "掩码内字段应被更新")
	require.Equal(t, "sqlite权限点-更新", *after[0].Name, "掩码外字段 name 应保持原值")
	require.Equal(t, "sqlite_perm_update_code", *after[0].Code, "掩码外字段 code 应保持原值")
}

// TestPermissionRepoSqlite_UpdateKeepsRelationGrants 是回归测试：编辑权限点曾把它自己的
// 「菜单 / 接口授权」整片清空。原因是 UpdateOne 内部的 FilterByFieldMask 会把不在 mask 里的
// Data 字段清零，而 api_ids / menu_ids 又被黑名单移出 mask —— Assign* 于是拿到空集，
// 而它们的语义是「空集＝删光该权限的全部关联」。三端权限抽屉都带 menuIds/apiIds 提交，
// 所以这条路径是「保存一次权限，丢光一次授权」。
func TestPermissionRepoSqlite_UpdateKeepsRelationGrants(t *testing.T) {
	newPerm := func(t *testing.T, code string) (*PermissionRepo, context.Context, uint32) {
		t.Helper()
		entClient := enttest.NewEntClientForTest(t)
		repo := newPermissionRepoSqlite(t, entClient)
		ctx := enttest.NewSystemViewerCtx(context.Background())
		require.NoError(t, repo.Create(ctx, &permissionV1.CreatePermissionRequest{
			Data: &permissionV1.Permission{
				Name:    trans.Ptr("sqlite权限点-关联保持"),
				Code:    trans.Ptr(code),
				MenuIds: []uint32{1, 2, 3},
				ApiIds:  []uint32{10, 11},
			},
		}))
		rows, err := repo.entClient.Client().Permission.Query().All(ctx)
		require.NoError(t, err)
		require.Len(t, rows, 1)
		return repo, ctx, rows[0].ID
	}

	menuIDs := func(t *testing.T, repo *PermissionRepo, ctx context.Context, permID uint32) []uint32 {
		t.Helper()
		links, err := repo.entClient.Client().PermissionMenu.Query().
			Where(permissionmenu.PermissionIDEQ(permID)).All(ctx)
		require.NoError(t, err)
		ids := make([]uint32, 0, len(links))
		for _, l := range links {
			require.NotNil(t, l.MenuID, "关联行的 menu_id 不应为 NULL")
			ids = append(ids, *l.MenuID)
		}
		return ids
	}

	t.Run("掩码带关联字段（三端权限抽屉的形状）应落新集", func(t *testing.T) {
		repo, ctx, id := newPerm(t, "sqlite_perm_rel_mask")
		require.NoError(t, repo.Update(ctx, &permissionV1.UpdatePermissionRequest{
			Id:         id,
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"name", "menu_ids", "api_ids"}},
			Data: &permissionV1.Permission{
				Name:    trans.Ptr("sqlite权限点-关联改写"),
				MenuIds: []uint32{3, 72, 73},
				ApiIds:  []uint32{20},
			},
		}))
		require.ElementsMatch(t, []uint32{3, 72, 73}, menuIDs(t, repo, ctx, id), "提交的菜单集应整体替换旧的")
		apis, err := repo.entClient.Client().PermissionApi.Query().
			Where(permissionapi.PermissionIDEQ(id)).All(ctx)
		require.NoError(t, err)
		require.Len(t, apis, 1, "提交的接口集应整体替换旧的")
	})

	t.Run("掩码不含关联字段（只改名）不得动授权", func(t *testing.T) {
		repo, ctx, id := newPerm(t, "sqlite_perm_rel_rename")
		require.NoError(t, repo.Update(ctx, &permissionV1.UpdatePermissionRequest{
			Id:         id,
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"name"}},
			Data:       &permissionV1.Permission{Name: trans.Ptr("sqlite权限点-仅改名")},
		}))
		require.ElementsMatch(t, []uint32{1, 2, 3}, menuIDs(t, repo, ctx, id), "没提交 menu_ids 就该维持原授权")
	})

	t.Run("camelCase 掩码路径同样受保护", func(t *testing.T) {
		// protojson 会把 json_name 规范成 snake_case，但直连 gRPC/测试的调用方给什么就是什么，
		// 两种拼写都得认出，否则同样的清空只在 HTTP 侧消失。
		repo, ctx, id := newPerm(t, "sqlite_perm_rel_camel")
		require.NoError(t, repo.Update(ctx, &permissionV1.UpdatePermissionRequest{
			Id:         id,
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"menuIds", "apiIds"}},
			Data: &permissionV1.Permission{
				MenuIds: []uint32{4},
				ApiIds:  []uint32{30, 31},
			},
		}))
		require.Equal(t, []uint32{4}, menuIDs(t, repo, ctx, id))
	})

	t.Run("显式提交空集仍是清空", func(t *testing.T) {
		repo, ctx, id := newPerm(t, "sqlite_perm_rel_clear")
		require.NoError(t, repo.Update(ctx, &permissionV1.UpdatePermissionRequest{
			Id:         id,
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"menu_ids", "api_ids"}},
			Data: &permissionV1.Permission{
				Name:    trans.Ptr("sqlite权限点-清空授权"),
				MenuIds: []uint32{},
				ApiIds:  []uint32{},
			},
		}))
		require.Empty(t, menuIDs(t, repo, ctx, id), "全不勾是合法操作：提交了就该清空")
	})
}

// TestPermissionRepoSqlite_Delete 验证按 ID 与按 code 删除后行数归零。
func TestPermissionRepoSqlite_Delete(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newPermissionRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	// 按 ID 删除
	require.NoError(t, repo.Create(ctx, &permissionV1.CreatePermissionRequest{
		Data: &permissionV1.Permission{
			Name: trans.Ptr("待删除-按ID"),
			Code: trans.Ptr("sqlite_perm_del_by_id"),
		},
	}))
	rows, err := repo.entClient.Client().Permission.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.NoError(t, repo.Delete(ctx, &permissionV1.DeletePermissionRequest{
		QueryBy: &permissionV1.DeletePermissionRequest_Id{Id: rows[0].ID},
	}), "按 ID 删除应成功")
	cnt, err := repo.entClient.Client().Permission.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, cnt, "按 ID 删除后表内行数应为 0")

	// 按 code 删除
	require.NoError(t, repo.Create(ctx, &permissionV1.CreatePermissionRequest{
		Data: &permissionV1.Permission{
			Name: trans.Ptr("待删除-按code"),
			Code: trans.Ptr("sqlite_perm_del_by_code"),
		},
	}))
	require.NoError(t, repo.Delete(ctx, &permissionV1.DeletePermissionRequest{
		QueryBy: &permissionV1.DeletePermissionRequest_Code{Code: "sqlite_perm_del_by_code"},
	}), "按 code 删除应成功")
	cnt, err = repo.entClient.Client().Permission.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, cnt, "按 code 删除后表内行数应为 0")
}
