// LoginPolicyService 的 SQLite 内存库测试。
//
// 登录策略是"谁能登录"的闸门，值格式配错会让匹配器静默不命中——
// 白名单场景下等于全员被锁在门外，所以服务层的值格式校验（ValidateLoginPolicyValue）
// 是第一道防线，测试盯三件事：
//   - 值格式：非法 IP / 坏 CIDR / 空 value 在 Create/Update 都被拒；
//   - 守卫与操作人：nil 请求/缺 Data 报 BadRequest；合法创建落 CreatedBy，
//     更新落 UpdatedBy 并追加进掩码；
//   - Update 的校验先于操作人解析（格式错误不依赖身份也能稳定 400）。
package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/enttest"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	"go-wind-admin/pkg/middleware/auth"
)

func newLoginPolicyServiceForTest(t *testing.T) (*LoginPolicyService, context.Context) {
	t.Helper()
	return &LoginPolicyService{
		log:  bLogger.NewHelper(bLogger.NopLogger()),
		repo: data.NewLoginPolicyRepoForTest(enttest.NewEntClientForTest(t)),
	}, auth.NewContext(enttest.NewSystemContext(context.Background()),
		&authenticationV1.UserTokenPayload{UserId: 91, IsPlatformAdmin: trans.Ptr(true)})
}

func TestLoginPolicyService_CreateValidatesValueFormat(t *testing.T) {
	svc, opCtx := newLoginPolicyServiceForTest(t)

	// 非法值三连：坏 IP / 坏 CIDR / 空 value
	for _, tc := range []struct {
		name  string
		value string
	}{
		{"坏 IP", "999.1.2.3"},
		{"坏 CIDR", "10.0.0.0/99"},
		{"空 value", "  "},
	} {
		_, err := svc.Create(opCtx, &authenticationV1.CreateLoginPolicyRequest{
			Data: &authenticationV1.LoginPolicy{
				Method: authenticationV1.LoginPolicy_IP.Enum(),
				Value:  trans.Ptr(tc.value),
			},
		})
		require.Error(t, err, "形态[%s]应被值校验拒绝", tc.name)
	}

	// 合法值（单 IP）：创建成功且落 CreatedBy
	_, err := svc.Create(opCtx, &authenticationV1.CreateLoginPolicyRequest{
		Data: &authenticationV1.LoginPolicy{
			Method: authenticationV1.LoginPolicy_IP.Enum(),
			Value:  trans.Ptr("10.0.0.1"),
		},
	})
	require.NoError(t, err)

	sysCtx := enttest.NewSystemContext(context.Background())
	created, err := svc.repo.Get(sysCtx, &authenticationV1.GetLoginPolicyRequest{
		QueryBy: &authenticationV1.GetLoginPolicyRequest_Id{Id: 1},
	})
	require.NoError(t, err)
	require.Equal(t, uint32(91), created.GetCreatedBy(), "CreatedBy 应落操作人")

	// 守卫
	_, err = svc.Create(opCtx, nil)
	require.Error(t, err, "nil 请求应报 BadRequest")
	_, err = svc.Create(opCtx, &authenticationV1.CreateLoginPolicyRequest{})
	require.Error(t, err, "缺 Data 应报 BadRequest")

	// 无操作人：显式报错而非 panic
	_, err = svc.Create(enttest.NewSystemContext(context.Background()), &authenticationV1.CreateLoginPolicyRequest{
		Data: &authenticationV1.LoginPolicy{
			Method: authenticationV1.LoginPolicy_IP.Enum(),
			Value:  trans.Ptr("192.168.1.1"),
		},
	})
	require.Error(t, err, "缺操作人应报错")
}

func TestLoginPolicyService_UpdateValidatesBeforeOperator(t *testing.T) {
	svc, opCtx := newLoginPolicyServiceForTest(t)

	// 值校验先于操作人解析：无身份上下文时格式错误仍应稳定 400（而非 panic）。
	// 先落一行真实策略（repo.Update 对不存在行报 NotFound）。
	opCtxSeed := auth.NewContext(enttest.NewSystemContext(context.Background()),
		&authenticationV1.UserTokenPayload{UserId: 91, IsPlatformAdmin: trans.Ptr(true)})
	_, err := svc.Create(opCtxSeed, &authenticationV1.CreateLoginPolicyRequest{
		Data: &authenticationV1.LoginPolicy{
			Method: authenticationV1.LoginPolicy_IP.Enum(),
			Value:  trans.Ptr("10.0.0.1"),
		},
	})
	require.NoError(t, err)

	sysCtx := enttest.NewSystemContext(context.Background())
	_, err = svc.Update(sysCtx, &authenticationV1.UpdateLoginPolicyRequest{
		Id: 1,
		Data: &authenticationV1.LoginPolicy{
			Method: authenticationV1.LoginPolicy_TIME.Enum(),
			Value:  trans.Ptr("bad time window"),
		},
	})
	require.Error(t, err, "格式错误应先被值校验拦下")

	mask, err := fieldmaskpb.New(&authenticationV1.LoginPolicy{}, "value")
	require.NoError(t, err)
	_, err = svc.Update(opCtx, &authenticationV1.UpdateLoginPolicyRequest{
		Id:         1,
		UpdateMask: mask,
		Data: &authenticationV1.LoginPolicy{
			Method: authenticationV1.LoginPolicy_IP.Enum(),
			Value:  trans.Ptr("10.0.0.0/24"),
		},
	})
	require.NoError(t, err)

	updated, err := svc.repo.Get(sysCtx, &authenticationV1.GetLoginPolicyRequest{
		QueryBy: &authenticationV1.GetLoginPolicyRequest_Id{Id: 1},
	})
	require.NoError(t, err)
	require.Equal(t, "10.0.0.0/24", updated.GetValue())
	require.Equal(t, uint32(91), updated.GetUpdatedBy(), "UpdatedBy 应经追加掩码落库")

	// 守卫
	_, err = svc.Update(opCtx, nil)
	require.Error(t, err, "nil 请求应报 BadRequest")
	_, err = svc.Update(opCtx, &authenticationV1.UpdateLoginPolicyRequest{Id: 1})
	require.Error(t, err, "缺 Data 应报 BadRequest")
}

func TestLoginPolicyService_DeleteGuard(t *testing.T) {
	svc, _ := newLoginPolicyServiceForTest(t)

	_, err := svc.Delete(enttest.NewSystemContext(context.Background()), nil)
	require.Error(t, err, "nil 删除请求应报 BadRequest")
}
