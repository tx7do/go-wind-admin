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
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/ent"
	entPermissionGroup "go-wind-admin/app/admin/service/internal/data/ent/permissiongroup"
	"go-wind-admin/app/admin/service/internal/data/enttest"
	"go-wind-admin/pkg/constants"
	"go-wind-admin/pkg/middleware/auth"
)

// newPermissionGroupServiceForTest 白盒构造 PermissionGroupService，逐字段对齐
// NewPermissionGroupService 的装配：log 用 NopLogger helper，permissionGroupRepo 走
// data.NewPermissionGroupRepoForTest；permissionRepo 置 nil（Delete 路径会先清理组内
// 权限点，属本批次 testkit 未覆盖的 repo，Delete 用例因此不在本文件覆盖范围）。
func newPermissionGroupServiceForTest(t *testing.T, entClient *entCrud.EntClient[*ent.Client]) *PermissionGroupService {
	t.Helper()
	return &PermissionGroupService{
		log:                 bLogger.NewHelper(bLogger.NopLogger()),
		permissionGroupRepo: data.NewPermissionGroupRepoForTest(entClient),
		permissionRepo:      nil,
	}
}

// TestPermissionGroupServiceSqlite_InitSeedsDefaultTree_AndListAssemblesTree
// 空表上调用 init() 应播种 constants.DefaultPermissionGroups（一棵 1+4 的树）；
// 服务层 List 走 treeTravel=true：响应只含根节点，全部子节点组装进根的 Children，
// Total 统计全部行。
func TestPermissionGroupServiceSqlite_InitSeedsDefaultTree_AndListAssemblesTree(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newPermissionGroupServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	svc.init()

	cnt, err := entClient.Client().PermissionGroup.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, len(constants.DefaultPermissionGroups), cnt, "init() 后应播种全部默认权限组")

	listResp, err := svc.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, int64(len(constants.DefaultPermissionGroups)), listResp.Total, "Total 应统计全部已播种行")
	require.Len(t, listResp.Items, 1, "树形组装后只应返回根节点")
	root := listResp.Items[0]
	require.Equal(t, "系统管理", root.GetName(), "根节点应为默认树的根")
	require.Len(t, root.GetChildren(), len(constants.DefaultPermissionGroups)-1, "根节点应组装全部子分组")
	childNames := map[string]bool{}
	for _, child := range root.GetChildren() {
		childNames[child.GetName()] = true
		require.NotNil(t, child.ParentId, "子分组的 ParentId 应指向根")
		require.Equal(t, root.GetId(), *child.ParentId, "子分组的 ParentId 应指向根节点 ID")
	}
	require.Len(t, childNames, len(constants.DefaultPermissionGroups)-1, "四个默认子分组应各自出现一次")
	for _, name := range []string{"系统权限", "租户管理", "审计管理", "安全策略"} {
		require.True(t, childNames[name], "默认子分组 [%s] 应出现在根的 Children 中", name)
	}
}

// TestPermissionGroupServiceSqlite_CreateAndGet 验证服务层创建分组：
// 操作人 ID 盖入 created_by、ParentId 落库为指向父分组的外键、物化路径按父路径拼接；
// 新子分组随后出现在服务层 List 的树形组装中；Get 按主键命中/未命中。
func TestPermissionGroupServiceSqlite_CreateAndGet(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newPermissionGroupServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7, IsPlatformAdmin: trans.Ptr(true)})

	svc.init()

	rootID := int64(0)
	rows, err := entClient.Client().PermissionGroup.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, len(constants.DefaultPermissionGroups))
	for _, r := range rows {
		if r.ParentID == nil {
			rootID = r.ID
		}
	}
	require.NotZero(t, rootID, "播种结果中应找到根分组")

	_, err = svc.Create(opCtx, &permissionV1.CreatePermissionGroupRequest{
		Data: &permissionV1.PermissionGroup{
			Name:        trans.Ptr("服务层新分组"),
			ParentId:    &rootID,
			Module:      trans.Ptr("svc"),
			Status:      permissionV1.PermissionGroup_ON.Enum(),
			SortOrder:   trans.Ptr(uint32(9)),
			Description: trans.Ptr("服务层创建的分组描述"),
		},
	})
	require.NoError(t, err, "服务层 Create 应成功")

	afterRows, err := entClient.Client().PermissionGroup.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, afterRows, len(constants.DefaultPermissionGroups)+1, "播种树之外应新增 1 个分组")
	var createdID int64
	for _, r := range afterRows {
		if r.Name != nil && *r.Name == "服务层新分组" {
			createdID = r.ID
			require.NotNil(t, r.ParentID, "ParentId 应落库为指向父分组的外键")
			require.Equal(t, rootID, *r.ParentID, "外键应指向根分组")
			require.NotNil(t, r.CreatedBy, "created_by 应被服务层盖入操作人 ID")
			require.Equal(t, int64(7), *r.CreatedBy, "created_by 应等于令牌声明中的操作人 ID")
			require.Equal(t, entPermissionGroup.StatusOn, *r.Status, "status 应经转换器落库为 ON")
		}
	}
	require.NotZero(t, createdID, "新建分组应可回查")

	listResp, err := svc.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, int64(len(constants.DefaultPermissionGroups)+1), listResp.Total, "Total 应统计全部行")
	require.Len(t, listResp.Items, 1, "树形组装后仍只应返回根节点")
	require.Len(t, listResp.Items[0].GetChildren(), len(constants.DefaultPermissionGroups), "根的 Children 应包含全部子分组（含新建者）")

	got, err := svc.Get(ctx, &permissionV1.GetPermissionGroupRequest{
		QueryBy: &permissionV1.GetPermissionGroupRequest_Id{Id: createdID},
	})
	require.NoError(t, err, "按已存在主键查询应命中")
	require.Equal(t, "服务层新分组", got.GetName(), "命中记录的 name 应与写入一致")

	_, err = svc.Get(ctx, &permissionV1.GetPermissionGroupRequest{
		QueryBy: &permissionV1.GetPermissionGroupRequest_Id{Id: 9999999},
	})
	require.Error(t, err, "按不存在主键查询应返回错误")
}

// TestPermissionGroupServiceSqlite_Update 验证服务层 Update 在单字段掩码下
// 只更新掩码内字段（name），掩码外字段（module）保持原值。
func TestPermissionGroupServiceSqlite_Update(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newPermissionGroupServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7, IsPlatformAdmin: trans.Ptr(true)})

	_, err := svc.Create(opCtx, &permissionV1.CreatePermissionGroupRequest{
		Data: &permissionV1.PermissionGroup{
			Name:   trans.Ptr("更新前分组名"),
			Module: trans.Ptr("svc-orig"),
			Status: permissionV1.PermissionGroup_ON.Enum(),
		},
	})
	require.NoError(t, err, "创建根分组应成功")

	rows, err := entClient.Client().PermissionGroup.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := rows[0].ID

	_, err = svc.Update(opCtx, &permissionV1.UpdatePermissionGroupRequest{
		Id:         createdID,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"name"}},
		Data:       &permissionV1.PermissionGroup{Name: trans.Ptr("更新后分组名")},
	})
	require.NoError(t, err, "单字段掩码更新应成功")

	after, err := entClient.Client().PermissionGroup.Query().Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "更新后分组名", *after.Name, "掩码内字段 name 应被更新")
	require.Equal(t, "svc-orig", *after.Module, "掩码外字段 module 应保持原值")
}
