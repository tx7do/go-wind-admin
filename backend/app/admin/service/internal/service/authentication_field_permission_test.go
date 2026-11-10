// aggregateHiddenFields（令牌承载的黑名单式字段权限）的测试。
//
// 字段权限是纵深防御手段，语义取向是"登录可用性优先"（fail-open），
// 测试盯三件事：
//   - 平台上下文（tenantID==0）不聚合：平台管理员字段全可见；
//   - 多角色按并集聚合且跨角色去重："user.phone" 两角色都配了只进一次令牌
//     （重复条目会白白膨胀每个 JWT 的体积）；
//   - 脏条目（resource/field 为空串）跳过，不产出 "." 这样的坏键。
package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// TestAggregateHiddenFields_PlatformContextSkips 平台上下文不聚合：
// 不查库、不改 payload（平台管理员字段全可见）。
func TestAggregateHiddenFields_PlatformContextSkips(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := &AuthenticationService{
		log:                     bLogger.NewHelper(bLogger.NopLogger()),
		roleFieldPermissionRepo: data.NewRoleFieldPermissionRepoForTest(entClient),
	}
	sysCtx := enttest.NewSystemContext(context.Background())

	// 落一行角色 11 的隐藏字段配置：平台上下文也不该带出
	_, err := entClient.Client().RoleFieldPermission.Create().
		SetRoleID(11).
		SetResource("user").
		SetFieldName("phone").
		Save(sysCtx)
	require.NoError(t, err)

	payload := &authenticationV1.UserTokenPayload{}
	svc.aggregateHiddenFields(sysCtx, 0, []int64{11}, payload)
	require.Empty(t, payload.GetHiddenFields(), "平台上下文（tenantID==0）不应聚合隐藏字段")
}

// TestAggregateHiddenFields_UnionsAndDedupes 多角色并集聚合 + 跨角色去重 + 脏条目跳过。
func TestAggregateHiddenFields_UnionsAndDedupes(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := &AuthenticationService{
		log:                     bLogger.NewHelper(bLogger.NopLogger()),
		roleFieldPermissionRepo: data.NewRoleFieldPermissionRepoForTest(entClient),
	}
	sysCtx := enttest.NewSystemContext(context.Background())

	seed := func(roleID int64, resource, field string) {
		_, err := entClient.Client().RoleFieldPermission.Create().
			SetRoleID(roleID).
			SetResource(resource).
			SetFieldName(field).
			Save(sysCtx)
		require.NoError(t, err)
	}
	// 角色 11：user.phone / user.id_card / 空资源脏行
	seed(11, "user", "phone")
	seed(11, "user", "id_card")
	seed(11, "", "orphan_field")
	// 角色 12：user.phone（与 11 重复）/ order.email / 空 field 脏行
	seed(12, "user", "phone")
	seed(12, "order", "email")
	seed(12, "order", "")

	payload := &authenticationV1.UserTokenPayload{}
	svc.aggregateHiddenFields(sysCtx, 5, []int64{11, 12}, payload)

	// 分组来自 map 遍历，条目顺序不承诺；语义只有并集+去重+脏条目剔除
	require.ElementsMatch(t, []string{"user.phone", "user.id_card", "order.email"},
		payload.GetHiddenFields(),
		"应按并集聚合、跨角色去重、跳过空 resource/field 的脏条目")

	// 空 roleIDs：无隐藏字段
	empty := &authenticationV1.UserTokenPayload{}
	svc.aggregateHiddenFields(sysCtx, 5, nil, empty)
	require.Empty(t, empty.GetHiddenFields(), "无角色不应产生隐藏字段")
}
