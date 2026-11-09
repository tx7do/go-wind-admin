package data

import (
	"context"
	"strconv"
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
	"go-wind-admin/app/admin/service/internal/data/ent/permissiongroup"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newPermissionGroupRepoSqlite 在给定 enttest client 上白盒构造 PermissionGroupRepo，
// 逐字段复刻 NewPermissionGroupRepo 的 mapper/converter 初始化，再调用 init()。
func newPermissionGroupRepoSqlite(t *testing.T, entClient *entCrud.EntClient[*ent.Client]) *PermissionGroupRepo {
	t.Helper()
	repo := &PermissionGroupRepo{
		entClient: entClient,
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		mapper:    mapper.NewCopierMapper[permissionV1.PermissionGroup, ent.PermissionGroup](),
		statusConverter: mapper.NewEnumTypeConverter[permissionV1.PermissionGroup_Status, permissiongroup.Status](
			permissionV1.PermissionGroup_Status_name, permissionV1.PermissionGroup_Status_value,
		),
	}
	repo.init()
	return repo
}

// TestPermissionGroupRepoSqlite_Create 验证根分组的创建与落库，
// 以及根节点物化路径 "/<id>/" 的计算落库。
func TestPermissionGroupRepoSqlite_Create(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newPermissionGroupRepoSqlite(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	dto, err := repo.Create(ctx, &permissionV1.CreatePermissionGroupRequest{
		Data: &permissionV1.PermissionGroup{
			Name:        trans.Ptr("sqlite分组-根"),
			Module:      trans.Ptr("sqlite-module"),
			Status:      permissionV1.PermissionGroup_ON.Enum(),
			Description: trans.Ptr("根分组描述"),
			SortOrder:   trans.Ptr(uint32(3)),
		},
	})
	require.NoError(t, err, "repo.Create 应写入 SQLite 成功")
	require.NotNil(t, dto, "Create 应返回创建的 DTO")
	require.NotZero(t, dto.GetId(), "创建的分组应有非零 ID")

	rows, err := entClient.Client().PermissionGroup.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1, "SQLite 中应有 1 条 permission_group 记录")
	require.Equal(t, "sqlite分组-根", *rows[0].Name, "name 应按请求落库")
	require.Equal(t, "sqlite-module", *rows[0].Module, "module 应按请求落库")
	require.Equal(t, permissiongroup.StatusOn, *rows[0].Status, "status 枚举应经 converter 落为 ON")
	require.Equal(t, "根分组描述", *rows[0].Description, "description 应按请求落库")
	require.NotNil(t, rows[0].Path, "根节点应有物化路径")
	require.Equal(t, "/"+strconv.FormatUint(uint64(rows[0].ID), 10)+"/", *rows[0].Path, "根节点物化路径应为 /<自身ID>/ 格式")

	// 创建产物读视图：status 为写入值（ON），经 backfillEnumsFrom 如实回显
	//（mapper 的枚举转换对无法赋入指针字段，见仓内 queryEnumsAndBackfill 处注释）。
	require.Equal(t, permissionV1.PermissionGroup_ON, dto.GetStatus(), "创建产物读视图应回填 status")
}

// TestPermissionGroupRepoSqlite_CreateTreePath 验证父子分组的物化路径：
// 子节点路径为父路径 + 自身 ID。
func TestPermissionGroupRepoSqlite_CreateTreePath(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newPermissionGroupRepoSqlite(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	parent, err := repo.Create(ctx, &permissionV1.CreatePermissionGroupRequest{
		Data: &permissionV1.PermissionGroup{Name: trans.Ptr("sqlite分组-父")},
	})
	require.NoError(t, err)
	require.NotZero(t, parent.GetId())

	child, err := repo.Create(ctx, &permissionV1.CreatePermissionGroupRequest{
		Data: &permissionV1.PermissionGroup{
			Name:     trans.Ptr("sqlite分组-子"),
			ParentId: trans.Ptr(parent.GetId()),
		},
	})
	require.NoError(t, err)
	require.NotZero(t, child.GetId())

	rows, err := entClient.Client().PermissionGroup.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	byID := map[uint64]*ent.PermissionGroup{}
	for _, row := range rows {
		byID[row.ID] = row
	}
	parentRow := byID[parent.GetId()]
	childRow := byID[child.GetId()]
	require.Equal(t, "/"+strconv.FormatUint(uint64(parent.GetId()), 10)+"/", *parentRow.Path, "父节点应为根路径格式")
	require.Equal(t, *parentRow.Path+strconv.FormatUint(uint64(child.GetId()), 10)+"/", *childRow.Path, "子节点路径应为父路径拼接自身 ID")
	require.Equal(t, parent.GetId(), *childRow.ParentID, "子节点 parent_id 应指向父分组")
}

// TestPermissionGroupRepoSqlite_ListContainsFilter 验证 List 的 contains 模糊搜索语义。
func TestPermissionGroupRepoSqlite_ListContainsFilter(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newPermissionGroupRepoSqlite(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	require.NoError(t, func() error {
		_, err := repo.Create(ctx, &permissionV1.CreatePermissionGroupRequest{
			Data: &permissionV1.PermissionGroup{Name: trans.Ptr("分组-markermnb-甲")},
		})
		return err
	}())
	require.NoError(t, func() error {
		_, err := repo.Create(ctx, &permissionV1.CreatePermissionGroupRequest{
			Data: &permissionV1.PermissionGroup{Name: trans.Ptr("分组-无关行-乙")},
		})
		return err
	}())

	filtered, err := repo.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_FilterExpr{
			FilterExpr: &paginationV1.FilterExpr{
				Type: paginationV1.ExprType_AND,
				Conditions: []*paginationV1.FilterCondition{
					{
						Field:      "name",
						Op:         paginationV1.Operator_CONTAINS,
						ValueOneof: &paginationV1.FilterCondition_Value{Value: "markermnb"},
					},
				},
			},
		},
	}, false)
	require.NoError(t, err)
	require.Equal(t, uint64(1), filtered.Total, "contains 过滤后 Total 应为 1")
	require.Len(t, filtered.Items, 1, "contains 过滤后只应返回 1 行")
	require.Contains(t, *filtered.Items[0].Name, "markermnb", "命中行应是携带标记的行")

	all, err := repo.List(ctx, &paginationV1.PagingRequest{}, false)
	require.NoError(t, err)
	require.Equal(t, uint64(2), all.Total, "无过滤时应返回全部 2 行")
	require.Len(t, all.Items, 2)
	// 列表读视图：status 未显式指定、按列默认（ON）落库，经 backfillEnumsFrom
	// 如实回显（树化前完成，全量节点覆盖）。
	for _, item := range all.Items {
		require.Equal(t, permissionV1.PermissionGroup_ON, item.GetStatus(), "列表读视图应回填列默认 status")
	}
}

// TestPermissionGroupRepoSqlite_Get 验证 Get 命中/未命中。
func TestPermissionGroupRepoSqlite_Get(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newPermissionGroupRepoSqlite(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	_, err := repo.Create(ctx, &permissionV1.CreatePermissionGroupRequest{
		Data: &permissionV1.PermissionGroup{Name: trans.Ptr("sqlite分组-Get")},
	})
	require.NoError(t, err)
	rows, err := entClient.Client().PermissionGroup.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := rows[0].ID

	hit, err := repo.Get(ctx, &permissionV1.GetPermissionGroupRequest{
		QueryBy: &permissionV1.GetPermissionGroupRequest_Id{Id: createdID},
	})
	require.NoError(t, err, "按存在的 ID 查询应命中")
	require.Equal(t, createdID, hit.GetId())
	// 读视图（Get 按主键）：status 未显式指定、按列默认（ON）落库，经
	// queryEnumsAndBackfill 如实回显。
	require.Equal(t, permissionV1.PermissionGroup_ON, hit.GetStatus(), "读视图应回填列默认 status")

	_, err = repo.Get(ctx, &permissionV1.GetPermissionGroupRequest{
		QueryBy: &permissionV1.GetPermissionGroupRequest_Id{Id: 9999999},
	})
	require.Error(t, err, "不存在的 ID 查询应返回错误")
}

// TestPermissionGroupRepoSqlite_Update 验证 Update 只更新掩码内字段（description），
// 物化路径与 name 不受影响。
func TestPermissionGroupRepoSqlite_Update(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newPermissionGroupRepoSqlite(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	_, err := repo.Create(ctx, &permissionV1.CreatePermissionGroupRequest{
		Data: &permissionV1.PermissionGroup{
			Name:        trans.Ptr("sqlite分组-更新"),
			Description: trans.Ptr("更新前描述"),
		},
	})
	require.NoError(t, err)
	rows, err := entClient.Client().PermissionGroup.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := rows[0].ID
	pathBefore := *rows[0].Path

	err = repo.Update(ctx, &permissionV1.UpdatePermissionGroupRequest{
		Id:         createdID,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"description"}},
		Data: &permissionV1.PermissionGroup{
			Description: trans.Ptr("更新后描述-sqlite"),
		},
	})
	require.NoError(t, err, "更新 description 应成功")

	after, err := entClient.Client().PermissionGroup.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, after, 1)
	require.Equal(t, "更新后描述-sqlite", *after[0].Description, "掩码内字段应被更新")
	require.Equal(t, "sqlite分组-更新", *after[0].Name, "掩码外字段 name 应保持原值")
	require.Equal(t, pathBefore, *after[0].Path, "物化路径不应随非结构更新变化")
}

// TestPermissionGroupRepoSqlite_Delete 验证叶子分组可删、
// 有子分组/有权限点的分组删除被拒绝。
func TestPermissionGroupRepoSqlite_Delete(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newPermissionGroupRepoSqlite(t, entClient)
	permRepo := newPermissionRepoSqlite(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	// 叶子分组：可删
	leaf, err := repo.Create(ctx, &permissionV1.CreatePermissionGroupRequest{
		Data: &permissionV1.PermissionGroup{Name: trans.Ptr("sqlite分组-叶子")},
	})
	require.NoError(t, err)
	require.NoError(t, repo.Delete(ctx, &permissionV1.DeletePermissionGroupRequest{
		QueryBy: &permissionV1.DeletePermissionGroupRequest_Id{Id: leaf.GetId()},
	}), "叶子分组删除应成功")
	cnt, err := entClient.Client().PermissionGroup.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, cnt, "叶子分组删除后表内行数应为 0")

	// 有子分组：拒绝删除
	parent, err := repo.Create(ctx, &permissionV1.CreatePermissionGroupRequest{
		Data: &permissionV1.PermissionGroup{Name: trans.Ptr("sqlite分组-父-删除守卫")},
	})
	require.NoError(t, err)
	_, err = repo.Create(ctx, &permissionV1.CreatePermissionGroupRequest{
		Data: &permissionV1.PermissionGroup{
			Name:     trans.Ptr("sqlite分组-子-删除守卫"),
			ParentId: trans.Ptr(parent.GetId()),
		},
	})
	require.NoError(t, err)
	err = repo.Delete(ctx, &permissionV1.DeletePermissionGroupRequest{
		QueryBy: &permissionV1.DeletePermissionGroupRequest_Id{Id: parent.GetId()},
	})
	require.Error(t, err, "有子分组的删除应被拒绝")

	// 分组下有权限点：拒绝删除
	require.NoError(t, permRepo.Create(ctx, &permissionV1.CreatePermissionRequest{
		Data: &permissionV1.Permission{
			Name:    trans.Ptr("挂靠分组的权限点"),
			Code:    trans.Ptr("sqlite_perm_group_guard"),
			GroupId: trans.Ptr(parent.GetId()),
		},
	}))
	err = repo.Delete(ctx, &permissionV1.DeletePermissionGroupRequest{
		QueryBy: &permissionV1.DeletePermissionGroupRequest_Id{Id: parent.GetId()},
	})
	require.Error(t, err, "分组下存在权限点时删除应被拒绝")

	stillThere, err := entClient.Client().PermissionGroup.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, stillThere, "被拒绝的删除不应移除任何分组")
}

// TestPermissionGroupRepoSqlite_ListByIDs 验证按 ID 集合查询：
// 命中返回对应行，空集合返回空。
func TestPermissionGroupRepoSqlite_ListByIDs(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newPermissionGroupRepoSqlite(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	groupA, err := repo.Create(ctx, &permissionV1.CreatePermissionGroupRequest{
		Data: &permissionV1.PermissionGroup{Name: trans.Ptr("按ID查-甲")},
	})
	require.NoError(t, err)
	groupB, err := repo.Create(ctx, &permissionV1.CreatePermissionGroupRequest{
		Data: &permissionV1.PermissionGroup{Name: trans.Ptr("按ID查-乙")},
	})
	require.NoError(t, err)

	// 命中：只返回集合内的行
	items, err := repo.ListByIDs(ctx, []uint64{groupA.GetId()})
	require.NoError(t, err)
	require.Len(t, items, 1, "按 ID 集合应只返回集合内的行")
	require.Equal(t, groupA.GetId(), items[0].GetId())
	require.Equal(t, "按ID查-甲", items[0].GetName())
	// 读视图（ListByIDs 路径）：status 为列默认（ON），经 backfillEnumsFrom
	// 如实回显。
	require.Equal(t, permissionV1.PermissionGroup_ON, items[0].GetStatus(), "读视图应回填列默认 status")

	// 空集合：返回空
	empty, err := repo.ListByIDs(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, empty, "空 ID 集合应返回空")

	// 含不存在的 ID：只返回存在的行
	partial, err := repo.ListByIDs(ctx, []uint64{groupB.GetId(), 987654})
	require.NoError(t, err)
	require.Len(t, partial, 1, "含不存在 ID 的集合应只返回存在的行")
	require.Equal(t, groupB.GetId(), partial[0].GetId())
	// 读视图（ListByIDs 路径）：同上，列默认 status 如实回显。
	require.Equal(t, permissionV1.PermissionGroup_ON, partial[0].GetStatus(), "读视图应回填列默认 status")
}

// TestPermissionGroupRepoSqlite_ExplicitZeroParentAsRoot parent_id 显式 0 与
// 未设置同义（挂根，落 NULL）：此前 0 直接写库触发自引用 FK 违约。
func TestPermissionGroupRepoSqlite_ExplicitZeroParentAsRoot(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newPermissionGroupRepoSqlite(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	dto, err := repo.Create(ctx, &permissionV1.CreatePermissionGroupRequest{
		Data: &permissionV1.PermissionGroup{
			Name:     trans.Ptr("零父根分组"),
			ParentId: trans.Ptr(uint64(0)),
		},
	})
	require.NoError(t, err, "显式 parent_id=0 的创建应按挂根处理而非 FK 违约")

	row, err := entClient.Client().PermissionGroup.Get(ctx, dto.GetId())
	require.NoError(t, err)
	require.Nil(t, row.ParentID, "parent_id=0 应落 NULL（根）")
	require.NotNil(t, row.Path)
	require.Equal(t, "/"+strconv.FormatUint(uint64(row.ID), 10)+"/", *row.Path, "根节点物化路径应为 /<自身ID>/")
}

// TestPermissionGroupRepoSqlite_UpdateExplicitZeroParentMovesToRoot 显式 0 更新
// = 移到根级，物化路径从父链前缀重算为根路径（此前被静默忽略或 FK 500）。
func TestPermissionGroupRepoSqlite_UpdateExplicitZeroParentMovesToRoot(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newPermissionGroupRepoSqlite(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	parent, err := repo.Create(ctx, &permissionV1.CreatePermissionGroupRequest{
		Data: &permissionV1.PermissionGroup{Name: trans.Ptr("移根父分组")},
	})
	require.NoError(t, err)
	child, err := repo.Create(ctx, &permissionV1.CreatePermissionGroupRequest{
		Data: &permissionV1.PermissionGroup{
			Name:     trans.Ptr("移根子分组"),
			ParentId: trans.Ptr(parent.GetId()),
		},
	})
	require.NoError(t, err)

	// 前置确认子节点带父前缀路径
	childRow, err := entClient.Client().PermissionGroup.Get(ctx, child.GetId())
	require.NoError(t, err)
	require.NotNil(t, childRow.Path)
	require.Contains(t, *childRow.Path, "/"+strconv.FormatUint(uint64(parent.GetId()), 10)+"/")

	require.NoError(t, repo.Update(ctx, &permissionV1.UpdatePermissionGroupRequest{
		Id:   child.GetId(),
		Data: &permissionV1.PermissionGroup{ParentId: trans.Ptr(uint64(0))},
	}), "显式 parent_id=0 的更新应把分组移到根级")

	childRow, err = entClient.Client().PermissionGroup.Get(ctx, child.GetId())
	require.NoError(t, err)
	require.Nil(t, childRow.ParentID, "移根后 parent_id 应落 NULL")
	require.NotNil(t, childRow.Path)
	require.Equal(t, "/"+strconv.FormatUint(uint64(child.GetId()), 10)+"/", *childRow.Path,
		"移根后物化路径应重算为根路径 /<自身ID>/")
}
