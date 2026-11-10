// MyTenantUsageService 的 SQLite 内存库测试。
//
// 租户自助用量把租户 ID 钉死为操作者所属租户（不接受客户端传参），
// 测试盯三件事：
//   - 无操作人：显式报错而非 panic；
//   - 平台用户（tenantId==0）：返回空 Usage（前端不展示该卡片），
//     这是平台管理员进入自助用量页的正确形态；
//   - 租户用户：租户存在时读到本租户用量；租户不存在时 repo 的
//     BadRequest 透传（调用方能区分"租户坏了"与"没用量"）。
package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/enttest"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	"go-wind-admin/pkg/middleware/auth"
)

// newMyTenantUsageServiceForTest 装配被测服务、库句柄与租户用户上下文（tenant 5）。
func newMyTenantUsageServiceForTest(t *testing.T) (*MyTenantUsageService, *entCrud.EntClient[*ent.Client], context.Context) {
	t.Helper()
	entClient := enttest.NewEntClientForTest(t)
	// authenticator 传 nil：GetUsage 不触碰它（有 nil 守卫），测试不覆盖吊销链路
	svc := &MyTenantUsageService{
		usageRepo: data.NewTenantUsageRepoForTest(entClient, nil),
		log:       bLogger.NewHelper(bLogger.NopLogger()),
	}
	opCtx := auth.NewContext(enttest.NewSystemContext(context.Background()),
		&authenticationV1.UserTokenPayload{UserId: 101, TenantId: trans.Ptr(int64(5))})
	return svc, entClient, opCtx
}

// TestMyTenantUsageService_PlatformUserGetsEmptyUsage 平台用户（tenantId==0）
// 返回空 Usage 且不查库——前端据此不渲染用量卡片。
func TestMyTenantUsageService_PlatformUserGetsEmptyUsage(t *testing.T) {
	svc, _, _ := newMyTenantUsageServiceForTest(t)

	platformCtx := auth.NewContext(enttest.NewSystemContext(context.Background()),
		&authenticationV1.UserTokenPayload{
			UserId:          1,
			IsPlatformAdmin: trans.Ptr(true),
		})
	usage, err := svc.GetMyTenantUsage(platformCtx, nil)
	require.NoError(t, err)
	require.NotNil(t, usage)
	require.Equal(t, int64(0), usage.GetTenantId(), "平台用户应返回 TenantId=0 的空 Usage")
}

// TestMyTenantUsageService_TenantUserScopes 无操作人报错、租户存在读到本租户
// 用量、租户不存在透传 BadRequest。
func TestMyTenantUsageService_TenantUserScopes(t *testing.T) {
	svc, entClient, opCtx := newMyTenantUsageServiceForTest(t)

	// 无操作人上下文：显式报错
	_, err := svc.GetMyTenantUsage(context.Background(), nil)
	require.Error(t, err, "缺操作人应报错")

	// 租户不存在：BadRequest 透传
	_, err = svc.GetMyTenantUsage(opCtx, nil)
	require.Error(t, err, "租户不存在应报错")

	// 落一个租户行：租户用户读到本租户用量（零值形态）。
	// 要用真实创建出的租户 ID 重建操作人上下文——租户 ID 自增，payload 里的 5 只是占位。
	sysCtx := enttest.NewSystemContext(context.Background())
	createdTenant, err := entClient.Client().Tenant.Create().
		SetNillableName(trans.Ptr("用量测试租户")).
		SetNillableCode(trans.Ptr("svc_usage_tenant")).
		Save(sysCtx)
	require.NoError(t, err, "租户应落库")

	tenantCtx := auth.NewContext(sysCtx, &authenticationV1.UserTokenPayload{
		UserId:   101,
		TenantId: trans.Ptr(createdTenant.ID),
	})

	usage, err := svc.GetMyTenantUsage(tenantCtx, nil)
	require.NoError(t, err)
	require.NotNil(t, usage)
	require.Equal(t, createdTenant.ID, usage.GetTenantId(), "租户 ID 应钉死为操作者所属租户")
}
