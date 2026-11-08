// AuthorizerProvider 的 SQLite 内存库测试。
//
// Provider 是权限引擎（casbin/OPA）的数据源，装错一条策略 = 多放行一个接口，
// 测试盯三件事：
//   - 模型数据按引擎名分发：casbin 空表、OPA 给内置 rego、未知引擎 nil；
//   - 策略装载的过滤链：模板角色不入策略表（它们是"角色模板"，不是可授权主体）、
//     无权限绑定的角色跳过、path/method 缺失的 API 行跳过——三类脏数据都不能
//     变成一条空/错的 casbin policy；
//   - domain 用租户 ID（casbin 多租户域语义），租户 0（平台角色）也如实落 "0"。
package data

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	entCrud "github.com/tx7do/go-crud/entgo"

	"go-wind-admin/pkg/constants"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newAuthorizerProviderForTest 组装被测 Provider（真 RoleRepo 走 SQLite 内存库；
// 绑定存端点快照后装载不再需要 ApiRepo）。
func newAuthorizerProviderForTest(t *testing.T) (*AuthorizerProvider, *entCrud.EntClient[*ent.Client]) {
	t.Helper()
	entClient := enttest.NewEntClientForTest(t)
	p := &AuthorizerProvider{
		log:      bLogger.NewHelper(bLogger.NopLogger()),
		roleRepo: NewRoleRepoForTest(entClient),
	}
	return p, entClient
}

// TestAuthorizerProvider_ProvideModels 验证模型数据按引擎名分发。
func TestAuthorizerProvider_ProvideModels(t *testing.T) {
	p, _ := newAuthorizerProviderForTest(t)

	casbin := p.ProvideModels("casbin")
	require.NotNil(t, casbin, "casbin 应返回非 nil 空表（策略全走 ProvidePolicies）")

	opa := p.ProvideModels("opa")
	require.NotEmpty(t, opa, "OPA 应带内置模型资产")
	require.NotEmpty(t, opa["rbac.rego"], "OPA 模型应含 rbac.rego 且非空")

	require.Nil(t, p.ProvideModels("unknown-engine"),
		"未知引擎应返回 nil（调用方据此判不支持，不能误拿别的引擎的模型）")
}

// TestAuthorizerProvider_ProvidePoliciesFilters 验证策略装载的过滤链与 domain 语义。
func TestAuthorizerProvider_ProvidePoliciesFilters(t *testing.T) {
	p, entClient := newAuthorizerProviderForTest(t)
	ctx := enttest.NewSystemContext(context.Background())
	client := entClient.Client()

	// —— 场景数据 ——
	// sys_apis 里有这条端点（现实形态）；但装载不再回查它——绑定快照即策略
	_, err := client.Api.Create().
		SetNillablePath(trans.Ptr("/admin/v1/orders")).
		SetNillableMethod(trans.Ptr("GET")).
		Save(ctx)
	require.NoError(t, err, "直建 api 应成功")

	perm, err := client.Permission.Create().
		SetName("策略装载测试权限").
		SetCode("authz_provider_perm").
		Save(ctx)
	require.NoError(t, err)

	// 绑定存端点快照：正常绑定落 (path, method)；脏绑定缺 path/method（历史
	// 数据/迁移半途的残留），装载器按空端点跳过
	require.NoError(t, client.PermissionApi.Create().
		SetPermissionID(perm.ID).
		SetPath("/admin/v1/orders").
		SetMethod("GET").
		Exec(ctx), "权限绑定正常端点快照")
	require.NoError(t, client.PermissionApi.Create().
		SetPermissionID(perm.ID).
		SetMethod("POST").
		Exec(ctx), "权限绑定缺 path 的脏绑定行（应被装载器跳过）")

	// 正常角色（租户 7）：绑定权限 → 应产出一条 policy
	roleNormal, err := client.Role.Create().
		SetNillableName(trans.Ptr("策略装载角色")).
		SetNillableCode(trans.Ptr("authz_provider_role")).
		SetNillableTenantID(trans.Ptr(uint32(7))).
		Save(ctx)
	require.NoError(t, err)
	require.NoError(t, client.RolePermission.Create().
		SetRoleID(roleNormal.ID).
		SetPermissionID(perm.ID).
		Exec(ctx))

	// 模板角色：同样绑定权限，但模板角色不是可授权主体，不应入策略表
	roleTemplate, err := client.Role.Create().
		SetNillableName(trans.Ptr("模板角色")).
		SetNillableCode(trans.Ptr(constants.TemplateRoleCodePrefix+"admin")).
		Save(ctx)
	require.NoError(t, err)
	require.NoError(t, client.RolePermission.Create().
		SetRoleID(roleTemplate.ID).
		SetPermissionID(perm.ID).
		Exec(ctx))

	// 无任何权限绑定的角色：跳过（GetApiByIDs 对空集合会 400，装载器不刷错误日志）
	_, err = client.Role.Create().
		SetNillableName(trans.Ptr("空权限角色")).
		SetNillableCode(trans.Ptr("authz_provider_empty")).
		Save(ctx)
	require.NoError(t, err)

	// —— 断言 ——
	policies, err := p.ProvidePolicies(ctx)
	require.NoError(t, err)
	require.Len(t, policies, 1, "只有正常角色应产出策略组")

	arr := policies["authz_provider_role"]
	require.NotNil(t, arr, "正常角色的策略组应存在")
	require.Len(t, arr, 1, "缺 path 的 api 应被跳过，只留一条")
	require.Equal(t, "/admin/v1/orders", arr[0].Path)
	require.Equal(t, "GET", arr[0].Method)
	require.Equal(t, "7", arr[0].Domain, "domain 应为租户 ID 的十进制字符串（casbin 域语义）")

	_, hasTemplate := policies["template:admin"]
	require.False(t, hasTemplate, "模板角色不应入策略表")

	_, hasEmpty := policies["authz_provider_empty"]
	require.False(t, hasEmpty, "无权限绑定的角色不应产出空策略组")
}

// TestAuthorizerProvider_ProvidePolicies_EmptyRepo 空库装载：返回空 map 而非错误
// （引擎启动期首次装载时表还是空的属于正常形态）。
func TestAuthorizerProvider_ProvidePolicies_EmptyRepo(t *testing.T) {
	p, _ := newAuthorizerProviderForTest(t)
	ctx := enttest.NewSystemContext(context.Background())

	policies, err := p.ProvidePolicies(ctx)
	require.NoError(t, err)
	require.NotNil(t, policies, "空库应返回空 map 而非 nil（nil 会被引擎当失败）")
	require.Empty(t, policies)
}
