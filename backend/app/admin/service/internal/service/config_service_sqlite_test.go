// ConfigService 的 SQLite 内存库测试。
//
// 参数管理是等保口令策略的运行时开关盘，测试盯三件事：
//   - init() 播种：进程启动时 SeedDefaults 按键缺一补一（读口令策略阈值
//     不能靠"表里恰好有这行"的运气）；
//   - 操作人盖章：Create 落 CreatedBy、Update 落 UpdatedBy 且把 updated_by
//     追加进掩码（调用方掩码没带它也要生效，否则"谁改的参数"无从追溯）；
//   - 守卫：Create/Update 缺 Data 报 BadRequest。
package service

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/sysconfig"
	"go-wind-admin/app/admin/service/internal/data/enttest"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	configV1 "go-wind-admin/api/gen/go/config/service/v1"
	"go-wind-admin/pkg/constants"
	"go-wind-admin/pkg/middleware/auth"
)

// derefStr 安全解引用 *string（nil 返回空串）。
func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// newConfigServiceForTest 装配被测服务与操作人上下文。
func newConfigServiceForTest(t *testing.T) (*ConfigService, *entCrud.EntClient[*ent.Client], context.Context) {
	t.Helper()
	entClient := enttest.NewEntClientForTest(t)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	svc := &ConfigService{
		log:        bLogger.NewHelper(bLogger.NopLogger()),
		configRepo: data.NewConfigRepoForTest(entClient, rdb),
	}
	sysCtx := enttest.NewSystemContext(context.Background())
	opCtx := auth.NewContext(sysCtx, &authenticationV1.UserTokenPayload{UserId: 71, IsPlatformAdmin: trans.Ptr(true)})
	return svc, entClient, opCtx
}

// TestConfigService_InitSeedsDefaults 验证构造期播种：init() 后口令策略阈值
// 三键均可读出播种默认值（而不是读方 fallback）。
func TestConfigService_InitSeedsDefaults(t *testing.T) {
	svc, _, ctx := newConfigServiceForTest(t)

	require.NotPanics(t, func() { svc.init() }, "空库播种不应 panic")
	require.Equal(t, constants.DefaultPasswordHistoryCount,
		svc.configRepo.GetConfigInt(ctx, constants.ConfigKeyPasswordHistoryCount, -1),
		"播种后应能读出历史口令条数默认值")
	require.Equal(t, constants.DefaultPasswordMinLen,
		svc.configRepo.GetConfigInt(ctx, constants.ConfigKeyPasswordMinLen, -1))
	require.Equal(t, constants.DefaultPasswordMaxAgeDays,
		svc.configRepo.GetConfigInt(ctx, constants.ConfigKeyPasswordMaxAgeDays, -1))

	// 重复 init：按键缺一补一、不覆盖既有值（幂等）
	require.NotPanics(t, func() { svc.init() })
	require.Equal(t, constants.DefaultPasswordHistoryCount,
		svc.configRepo.GetConfigInt(ctx, constants.ConfigKeyPasswordHistoryCount, -1))
}

// TestConfigService_CreateStampsOperator 验证 Create 落 CreatedBy + 守卫。
func TestConfigService_CreateStampsOperator(t *testing.T) {
	svc, entClient, opCtx := newConfigServiceForTest(t)
	sysCtx := enttest.NewSystemContext(context.Background())

	_, err := svc.Create(opCtx, &configV1.CreateConfigRequest{
		Data: &configV1.Config{
			Name:      trans.Ptr("测试参数"),
			Key:       trans.Ptr("sys.test.key"),
			Value:     trans.Ptr("42"),
			ValueType: configV1.Config_INT.Enum(),
		},
	})
	require.NoError(t, err)

	row, err := entClient.Client().SysConfig.Query().
		Where(sysconfig.KeyEQ("sys.test.key")).
		Only(sysCtx)
	require.NoError(t, err)
	require.NotNil(t, row.CreatedBy)
	require.Equal(t, int64(71), *row.CreatedBy, "CreatedBy 应落操作人")

	// 守卫
	_, err = svc.Create(opCtx, &configV1.CreateConfigRequest{})
	require.Error(t, err, "缺 Data 应报 BadRequest")
	_, err = svc.Create(opCtx, nil)
	require.Error(t, err, "nil 请求应报 BadRequest")

	// 无操作人上下文：显式拒绝而不是 panic
	_, err = svc.Create(enttest.NewSystemContext(context.Background()), &configV1.CreateConfigRequest{
		Data: &configV1.Config{Key: trans.Ptr("sys.noop.key")},
	})
	require.Error(t, err, "缺操作人应报错")
}

// TestConfigService_UpdateStampsOperatorAndMask 验证 Update：
// UpdatedBy 盖章 + updated_by 自动追加进掩码（调用方掩码没带它也生效）。
func TestConfigService_UpdateStampsOperatorAndMask(t *testing.T) {
	svc, entClient, opCtx := newConfigServiceForTest(t)
	sysCtx := enttest.NewSystemContext(context.Background())

	require.NoError(t, svc.configRepo.Create(sysCtx, &configV1.CreateConfigRequest{
		Data: &configV1.Config{
			Name:      trans.Ptr("原始参数"),
			Key:       trans.Ptr("sys.update.me"),
			Value:     trans.Ptr("1"),
			ValueType: configV1.Config_INT.Enum(),
		},
	}))
	row, err := entClient.Client().SysConfig.Query().
		Where(sysconfig.KeyEQ("sys.update.me")).
		Only(sysCtx)
	require.NoError(t, err)

	// 掩码只带 value：updated_by 由服务层补进掩码
	mask, err := fieldmaskpb.New(&configV1.Config{}, "value")
	require.NoError(t, err)
	_, err = svc.Update(opCtx, &configV1.UpdateConfigRequest{
		Id:         row.ID,
		UpdateMask: mask,
		Data: &configV1.Config{
			Value: trans.Ptr("2"),
		},
	})
	require.NoError(t, err)

	updated, err := entClient.Client().SysConfig.Get(sysCtx, row.ID)
	require.NoError(t, err)
	require.Equal(t, "2", derefStr(updated.Value), "掩码内 value 应更新")
	require.NotNil(t, updated.UpdatedBy, "updated_by 应经追加掩码落库")
	require.Equal(t, int64(71), *updated.UpdatedBy, "谁改的参数必须可追溯")

	// 守卫
	_, err = svc.Update(opCtx, &configV1.UpdateConfigRequest{Id: row.ID})
	require.Error(t, err, "缺 Data 应报 BadRequest")
	_, err = svc.Update(opCtx, nil)
	require.Error(t, err, "nil 请求应报 BadRequest")
}

// TestConfigService_ListGetDeletePassthrough 验证 List/Get/Delete 直通 repo。
func TestConfigService_ListGetDeletePassthrough(t *testing.T) {
	svc, _, opCtx := newConfigServiceForTest(t)

	_, err := svc.Create(opCtx, &configV1.CreateConfigRequest{
		Data: &configV1.Config{
			Name:      trans.Ptr("直通参数"),
			Key:       trans.Ptr("sys.passthrough"),
			Value:     trans.Ptr("v"),
			ValueType: configV1.Config_STRING.Enum(),
		},
	})
	require.NoError(t, err)

	resp, err := svc.List(opCtx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, int64(1), resp.GetTotal())

	got, err := svc.Get(opCtx, &configV1.GetConfigRequest{
		QueryBy: &configV1.GetConfigRequest_Id{Id: 1},
	})
	require.NoError(t, err)
	require.Equal(t, "sys.passthrough", got.GetKey())

	_, err = svc.Delete(opCtx, &configV1.DeleteConfigRequest{
		QueryBy: &configV1.DeleteConfigRequest_Id{Id: 1},
	})
	require.NoError(t, err)
	resp, err = svc.List(opCtx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, int64(0), resp.GetTotal(), "删除后列表应为空")
}
