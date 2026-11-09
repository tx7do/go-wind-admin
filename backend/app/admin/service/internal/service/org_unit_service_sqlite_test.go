// OrgUnitService 的 SQLite 内存库集成测试（白盒，包内测试）。
//
// 覆盖目标：
//   - List 的树组装（pagination.BuildTree：根节点提升到顶层、子节点挂到 Children）
//     与物化路径（setTreePath：根 "/<id>/"、子 "/<父路径>/<id>/"）。
//   - List / Get 的 enrichment：LeaderName / ContactUserName 经 userRepo.ListUsersByIds
//     桩回填；未设置负责人的单元不回填。
//   - Create 的操作人注入与 Delete 的基本路径。
//
// 已知方言限制（与 org_unit_repo_sqlite_test.go 的既有记录一致）：
// QueryAllChildrenIds 的递归 CTE 仅有 MySQL/PG 分支，SQLite 下子树收集返回 0 行，
// Delete 实际按 [自身ID] 删除——子节点在 SQLite 测试中会成为孤儿行，
// 且孤儿行会被 BuildTree 跳过（不出现在列表里）。
package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/enttest"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"

	"go-wind-admin/pkg/middleware/auth"
)

// orgUnitServiceUserRepoStub 是 OrgUnitService enrichment 专用的 data.UserRepo 桩：
// 嵌入接口获得默认方法集（未覆写方法被调用即 nil 接口 panic），只覆写
// ListUsersByIds，按 id 机械返回占位用户，验证 LeaderName/ContactUserName 回填链路。
type orgUnitServiceUserRepoStub struct {
	data.UserRepo
}

func (s *orgUnitServiceUserRepoStub) ListUsersByIds(_ context.Context, ids []uint64) ([]*identityV1.User, error) {
	users := make([]*identityV1.User, 0, len(ids))
	for _, id := range ids {
		users = append(users, &identityV1.User{
			Id:       trans.Ptr(id),
			Username: trans.Ptr(fmt.Sprintf("stub-user-%d", id)),
		})
	}
	return users, nil
}

// newOrgUnitServiceForTest 白盒复刻 NewOrgUnitService 的字段初始化：
// log 换 NopLogger，orgUnitRepo 用 testkit 构造器，userRepo 用本文件桩。
func newOrgUnitServiceForTest(t *testing.T) *OrgUnitService {
	t.Helper()
	entClient := enttest.NewEntClientForTest(t)
	return &OrgUnitService{
		log:         bLogger.NewHelper(bLogger.NopLogger()),
		orgUnitRepo: data.NewOrgUnitRepoForTest(entClient),
		userRepo:    &orgUnitServiceUserRepoStub{},
	}
}

// TestOrgUnitServiceSqlite_ListTreeAssemblyAndEnrichment 验证 List 的树组装、
// 物化路径与 LeaderName 回填。
func TestOrgUnitServiceSqlite_ListTreeAssemblyAndEnrichment(t *testing.T) {
	svc := newOrgUnitServiceForTest(t)
	ctx := enttest.NewSystemContext(context.Background())

	// 根节点：带 LeaderId（enrichment 命中）；子节点：无负责人（不回填）。
	require.NoError(t, svc.orgUnitRepo.Create(ctx, &identityV1.CreateOrgUnitRequest{
		Data: &identityV1.OrgUnit{
			Name:     trans.Ptr("OrgSvc 根单元甲"),
			Code:     trans.Ptr("ORGSVC_ROOT_A"),
			Status:   identityV1.OrgUnit_ON.Enum(),
			Type:     identityV1.OrgUnit_DEPARTMENT.Enum(),
			LeaderId: trans.Ptr(uint64(5551)),
		},
	}))
	// 取根节点 ID（repo Create 不返回 DTO，从列表反查）。
	rootList, err := svc.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Len(t, rootList.GetItems(), 1, "当前仅根节点存在，应作为唯一顶层节点")
	rootID := rootList.GetItems()[0].GetId()

	require.NoError(t, svc.orgUnitRepo.Create(ctx, &identityV1.CreateOrgUnitRequest{
		Data: &identityV1.OrgUnit{
			Name:     trans.Ptr("OrgSvc 子单元甲"),
			Code:     trans.Ptr("ORGSVC_CHILD_A"),
			Status:   identityV1.OrgUnit_ON.Enum(),
			Type:     identityV1.OrgUnit_DEPARTMENT.Enum(),
			ParentId: trans.Ptr(rootID),
		},
	}))

	// 列表：根节点提升到顶层，子节点挂到 Children（树组装）。
	resp, err := svc.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(2), resp.GetTotal(), "计数统计应包含根与子共 2 行")
	require.Len(t, resp.GetItems(), 1, "只有根节点出现在顶层")
	root := resp.GetItems()[0]
	require.Equal(t, "OrgSvc 根单元甲", root.GetName())
	require.Len(t, root.GetChildren(), 1, "子节点应挂到根节点的 Children")
	child := root.GetChildren()[0]
	require.Equal(t, "OrgSvc 子单元甲", child.GetName())

	// 物化路径：根 "/<rootID>/"，子 "/<rootID>/<childID>/"（setTreePath 落库）。
	require.Equal(t, fmt.Sprintf("/%d/", root.GetId()), root.GetPath(),
		"根节点物化路径应为 /<rootID>/")
	require.Equal(t, fmt.Sprintf("/%d/%d/", root.GetId(), child.GetId()), child.GetPath(),
		"子节点物化路径应为 /<父路径>/<childID>/")

	// enrichment：带 LeaderId 的根节点回填占位负责人名，子节点保持空。
	require.Equal(t, "stub-user-5551", root.GetLeaderName(),
		"带 LeaderId 的节点应回填占位负责人名")
	require.Empty(t, child.GetLeaderName(),
		"未设置负责人的节点不应回填负责人名")
}

// TestOrgUnitServiceSqlite_GetEnrichment 验证 Get 单条查询的
// LeaderName / ContactUserName 回填。
func TestOrgUnitServiceSqlite_GetEnrichment(t *testing.T) {
	svc := newOrgUnitServiceForTest(t)
	ctx := enttest.NewSystemContext(context.Background())

	require.NoError(t, svc.orgUnitRepo.Create(ctx, &identityV1.CreateOrgUnitRequest{
		Data: &identityV1.OrgUnit{
			Name:          trans.Ptr("OrgSvc 单条富集单元甲"),
			Code:          trans.Ptr("ORGSVC_GET_A"),
			Status:        identityV1.OrgUnit_ON.Enum(),
			Type:          identityV1.OrgUnit_DEPARTMENT.Enum(),
			LeaderId:      trans.Ptr(uint64(5552)),
			ContactUserId: trans.Ptr(uint64(5553)),
		},
	}))
	listResp, err := svc.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Len(t, listResp.GetItems(), 1)
	unitID := listResp.GetItems()[0].GetId()

	resp, err := svc.Get(ctx, &identityV1.GetOrgUnitRequest{
		QueryBy: &identityV1.GetOrgUnitRequest_Id{Id: unitID},
	})
	require.NoError(t, err)
	require.Equal(t, "OrgSvc 单条富集单元甲", resp.GetName())
	require.Equal(t, "stub-user-5552", resp.GetLeaderName(),
		"Get 应对 LeaderId 命中的单元回填占位负责人名")
	require.Equal(t, "stub-user-5553", resp.GetContactUserName(),
		"Get 应对 ContactUserId 命中的单元回填占位联系人名")
}

// TestOrgUnitServiceSqlite_CreateAndDelete 验证 Create 的操作人注入与
// Delete 的基本路径（SQLite 方言下按 [自身ID] 删除，子节点成为孤儿行且被
// BuildTree 跳过——见文件头已知方言限制）。
func TestOrgUnitServiceSqlite_CreateAndDelete(t *testing.T) {
	svc := newOrgUnitServiceForTest(t)
	baseCtx := enttest.NewSystemContext(context.Background())
	opCtx := auth.NewContext(baseCtx, &authenticationV1.UserTokenPayload{UserId: 4242})

	// Create：操作人注入 CreatedBy。
	_, err := svc.Create(opCtx, &identityV1.CreateOrgUnitRequest{
		Data: &identityV1.OrgUnit{
			Name:   trans.Ptr("OrgSvc 创建根单元乙"),
			Code:   trans.Ptr("ORGSVC_CREATE_ROOT_B"),
			Status: identityV1.OrgUnit_ON.Enum(),
			Type:   identityV1.OrgUnit_DEPARTMENT.Enum(),
		},
	})
	require.NoError(t, err, "service.Create 应走完操作人注入与落库")

	listResp, err := svc.List(baseCtx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Len(t, listResp.GetItems(), 1, "创建后应作为唯一顶层节点出现")
	root := listResp.GetItems()[0]
	rootID := root.GetId()
	require.EqualValues(t, 4242, root.GetCreatedBy(),
		"CreatedBy 应为操作人注入的 ID")

	// Delete：SQLite 下按 [自身ID] 删除（子树收集无方言分支，返回 0 行）。
	_, err = svc.Delete(baseCtx, &identityV1.DeleteOrgUnitRequest{
		QueryBy: &identityV1.DeleteOrgUnitRequest_Id{Id: rootID},
	})
	require.NoError(t, err, "无子节点的单元删除应成功")

	after, err := svc.List(baseCtx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Empty(t, after.GetItems(), "删除后列表不应再包含该节点")
	require.Equal(t, uint64(0), after.GetTotal(), "删除后表内计数应归零")
}
