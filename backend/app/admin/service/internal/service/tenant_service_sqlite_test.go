// TenantService 的 SQLite 内存库集成测试（白盒，包内测试）。
//
// 覆盖目标：
//   - List / Get 的 enrichment：AdminUserName（经 userRepo.ListUsersByIds 桩回填）
//     与 MemberCount（经 userRepo.CountByTenantIDs 桩回填）；无 admin 的租户不回填用户名。
//   - TenantExists 的 code / name 各自唯一冲突检测（OR 语义）。
//   - Create / Delete 基本路径：Create 走 auth.FromContext 操作人注入（CreatedBy 回填），
//     Delete 后列表清空。
//
// 跳过项：CreateTenantWithAdminUser（涉 userCredentialsRepo 密码哈希与 authorizer
// 重置链路，本批次不测，构造时对应字段置 nil）；GetUsage/CleanupData 未在覆盖范围。
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

// tenantServiceUserRepoStub 是 TenantService enrichment 专用的 data.UserRepo 桩：
// 嵌入接口获得默认方法集（未覆写方法一旦被调用即 nil 接口 panic，测试即失败），
// 只覆写 enrichment 路径会用到的 ListUsersByIds / CountByTenantIDs，
// 按 id 机械返回占位用户与固定计数，验证回填链路而非用户数据本身。
type tenantServiceUserRepoStub struct {
	data.UserRepo
}

func (s *tenantServiceUserRepoStub) ListUsersByIds(_ context.Context, ids []int64) ([]*identityV1.User, error) {
	users := make([]*identityV1.User, 0, len(ids))
	for _, id := range ids {
		users = append(users, &identityV1.User{
			Id:       trans.Ptr(id),
			Username: trans.Ptr(fmt.Sprintf("stub-admin-%d", id)),
		})
	}
	return users, nil
}

func (s *tenantServiceUserRepoStub) CountByTenantIDs(_ context.Context, tenantIDs []int64) (map[int64]int, error) {
	counts := make(map[int64]int, len(tenantIDs))
	for _, id := range tenantIDs {
		counts[id] = 3
	}
	return counts, nil
}

// tenantAdminCtx 平台管理员操作者上下文（四读口已收平台闸）。
func tenantPlatAdminCtx(ctx context.Context) context.Context {
	return auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 4242, IsPlatformAdmin: trans.Ptr(true)})
}

// newTenantServiceForTest 白盒复刻 NewTenantService 的字段初始化：
// log 换 NopLogger，repo 用 testkit 构造器，userRepo 用本文件桩；
// userCredentialsRepo / authorizer 仅 CreateTenantWithAdminUser 使用，置 nil。

func newTenantServiceForTest(t *testing.T) *TenantService {
	t.Helper()
	entClient := enttest.NewEntClientForTest(t)
	return &TenantService{
		log:                 bLogger.NewHelper(bLogger.NopLogger()),
		tenantRepo:          data.NewTenantRepoForTest(entClient),
		tenantUsageRepo:     data.NewTenantUsageRepoForTest(entClient, nil),
		userRepo:            &tenantServiceUserRepoStub{},
		userCredentialsRepo: nil,
		roleRepo:            data.NewRoleRepoForTest(entClient),
		authorizer:          nil,
	}
}

// TestTenantServiceSqlite_ListEnrichment 验证 List 的 AdminUserName / MemberCount 回填：
// 带 adminUserId 的租户回填占位用户名，未带的保持空；MemberCount 对两条均回填桩计数。
func TestTenantServiceSqlite_ListEnrichment(t *testing.T) {
	svc := newTenantServiceForTest(t)
	ctx := enttest.NewSystemContext(context.Background())

	withAdmin, err := svc.tenantRepo.Create(ctx, &identityV1.Tenant{
		Name:        trans.Ptr("TenantSvc 富集租户甲"),
		Code:        trans.Ptr("TENANTSVC_ENRICH_A"),
		Status:      identityV1.Tenant_ON.Enum(),
		Type:        identityV1.Tenant_TRIAL.Enum(),
		AuditStatus: identityV1.Tenant_APPROVED.Enum(),
		AdminUserId: trans.Ptr(int64(9901)),
	})
	require.NoError(t, err)
	require.NotNil(t, withAdmin)

	withoutAdmin, err := svc.tenantRepo.Create(ctx, &identityV1.Tenant{
		Name:        trans.Ptr("TenantSvc 富集租户乙"),
		Code:        trans.Ptr("TENANTSVC_ENRICH_B"),
		Status:      identityV1.Tenant_ON.Enum(),
		Type:        identityV1.Tenant_TRIAL.Enum(),
		AuditStatus: identityV1.Tenant_APPROVED.Enum(),
	})
	require.NoError(t, err)
	require.NotNil(t, withoutAdmin)

	resp, err := svc.List(tenantPlatAdminCtx(ctx), &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, int64(2), resp.GetTotal(), "两条租户都应出现在列表中")
	require.Len(t, resp.GetItems(), 2)

	for _, item := range resp.GetItems() {
		require.NotNil(t, item.MemberCount, "MemberCount 应回填桩计数")
		require.EqualValues(t, 3, item.GetMemberCount(), "MemberCount 应为桩返回的固定计数")
		switch item.GetCode() {
		case "TENANTSVC_ENRICH_A":
			require.Equal(t, "stub-admin-9901", item.GetAdminUserName(),
				"带 adminUserId 的租户应回填占位管理员用户名")
		case "TENANTSVC_ENRICH_B":
			require.Empty(t, item.GetAdminUserName(),
				"未带 adminUserId 的租户不应回填管理员用户名")
		default:
			t.Fatalf("列表中出现未创建的租户 code=%q", item.GetCode())
		}
	}
}

// TestTenantServiceSqlite_GetEnrichment 验证 Get 单条查询的 enrichment 回填。
func TestTenantServiceSqlite_GetEnrichment(t *testing.T) {
	svc := newTenantServiceForTest(t)
	ctx := enttest.NewSystemContext(context.Background())

	created, err := svc.tenantRepo.Create(ctx, &identityV1.Tenant{
		Name:        trans.Ptr("TenantSvc 富集租户丙"),
		Code:        trans.Ptr("TENANTSVC_ENRICH_C"),
		Status:      identityV1.Tenant_ON.Enum(),
		Type:        identityV1.Tenant_TRIAL.Enum(),
		AuditStatus: identityV1.Tenant_APPROVED.Enum(),
		AdminUserId: trans.Ptr(int64(9902)),
	})
	require.NoError(t, err)
	require.NotNil(t, created.Id)

	resp, err := svc.Get(tenantPlatAdminCtx(ctx), &identityV1.GetTenantRequest{
		QueryBy: &identityV1.GetTenantRequest_Id{Id: created.GetId()},
	})
	require.NoError(t, err)
	require.Equal(t, "TenantSvc 富集租户丙", resp.GetName())
	require.Equal(t, "stub-admin-9902", resp.GetAdminUserName(),
		"Get 应对 adminUserId 命中的租户回填占位管理员用户名")
	require.EqualValues(t, 3, resp.GetMemberCount(), "Get 应回填 MemberCount 桩计数")
}

// TestTenantServiceSqlite_TenantExists 验证 code / name 各自唯一的冲突检测：
// 任一命中即存在；两者皆空时按存在任意行处理；未命中返回不存在。
func TestTenantServiceSqlite_TenantExists(t *testing.T) {
	svc := newTenantServiceForTest(t)
	ctx := enttest.NewSystemContext(context.Background())

	_, err := svc.tenantRepo.Create(ctx, &identityV1.Tenant{
		Name:        trans.Ptr("TenantSvc 存在性租户甲"),
		Code:        trans.Ptr("TENANTSVC_EXISTS_A"),
		Status:      identityV1.Tenant_ON.Enum(),
		Type:        identityV1.Tenant_TRIAL.Enum(),
		AuditStatus: identityV1.Tenant_APPROVED.Enum(),
	})
	require.NoError(t, err)

	byCode, err := svc.TenantExists(tenantPlatAdminCtx(ctx), &identityV1.TenantExistsRequest{Code: "TENANTSVC_EXISTS_A"})
	require.NoError(t, err)
	require.True(t, byCode.GetExist(), "按已存在 code 查询应返回存在")

	byName, err := svc.TenantExists(tenantPlatAdminCtx(ctx), &identityV1.TenantExistsRequest{Name: "TenantSvc 存在性租户甲"})
	require.NoError(t, err)
	require.True(t, byName.GetExist(), "按已存在 name 查询应返回存在")

	byAbsent, err := svc.TenantExists(tenantPlatAdminCtx(ctx), &identityV1.TenantExistsRequest{Code: "TENANTSVC_EXISTS_NOT_EXIST"})
	require.NoError(t, err)
	require.False(t, byAbsent.GetExist(), "按不存在的 code 查询应返回不存在")
}

// TestTenantServiceSqlite_CreateAndDelete 验证 Create 的操作人注入与 Delete 的清理：
// Create 后列表可见且 CreatedBy 为操作人 ID；Delete 后列表清空。
func TestTenantServiceSqlite_CreateAndDelete(t *testing.T) {
	svc := newTenantServiceForTest(t)
	baseCtx := enttest.NewSystemContext(context.Background())
	// 操作人上下文：Create 走 auth.FromContext 取 operator.UserId 注入 CreatedBy。
	opCtx := auth.NewContext(baseCtx, &authenticationV1.UserTokenPayload{UserId: 4242, IsPlatformAdmin: trans.Ptr(true)})

	_, err := svc.Create(opCtx, &identityV1.CreateTenantRequest{
		Data: &identityV1.Tenant{
			Name:        trans.Ptr("TenantSvc 创建租户甲"),
			Code:        trans.Ptr("TENANTSVC_CREATE_A"),
			Status:      identityV1.Tenant_ON.Enum(),
			Type:        identityV1.Tenant_TRIAL.Enum(),
			AuditStatus: identityV1.Tenant_APPROVED.Enum(),
		},
	})
	require.NoError(t, err, "service.Create 应走完操作人注入与事务创建")

	var createdID int64
	listResp, err := svc.List(tenantPlatAdminCtx(baseCtx), &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, int64(1), listResp.GetTotal(), "创建后列表应恰好包含该租户")
	require.Len(t, listResp.GetItems(), 1)
	item := listResp.GetItems()[0]
	createdID = item.GetId()
	require.EqualValues(t, 4242, item.GetCreatedBy(),
		"CreatedBy 应为操作人注入的 ID")

	_, err = svc.Delete(opCtx, &identityV1.DeleteTenantRequest{
		QueryBy: &identityV1.DeleteTenantRequest_Id{Id: createdID},
	})
	require.NoError(t, err, "service.Delete 应删除该租户")

	after, err := svc.List(tenantPlatAdminCtx(baseCtx), &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, int64(0), after.GetTotal(), "删除后列表应为空")
	require.Empty(t, after.GetItems())
}
