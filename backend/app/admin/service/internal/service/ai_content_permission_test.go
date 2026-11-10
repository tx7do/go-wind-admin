package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"

	aiV1 "go-wind-admin/api/gen/go/ai/service/v1"
	"go-wind-admin/pkg/middleware/auth"

	"go-wind-admin/app/admin/service/internal/data"
	entCrud "github.com/tx7do/go-crud/entgo"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/menu"
	enttest "go-wind-admin/app/admin/service/internal/data/enttest"
)

// roleStubRepo / userStubRepo：仅覆写 visibleMenuIDs 链路用到的方法，
// 其余方法嵌入接口零值（不会被调用）。
// 菜单授权绑定存全路径快照后，roleRepo 产路径、真 MenuRepo 反解回 id。
type roleStubRepo struct {
	data.RoleRepo
	pathsByRoles map[string][]string // key: roleIDs 逗号串
}

func (r *roleStubRepo) GetRolesPermissionMenuFullPaths(_ context.Context, roleIDs []int64) ([]string, error) {
	key := fmt.Sprintf("%v", roleIDs)
	if paths, ok := r.pathsByRoles[key]; ok {
		return paths, nil
	}
	return []string{}, nil
}

type userStubRepo struct {
	data.UserRepo
	usersByID map[int64]*identityV1.User
}

func (r *userStubRepo) Get(_ context.Context, req *identityV1.GetUserRequest) (*identityV1.User, error) {
	if u, ok := r.usersByID[req.GetId()]; ok {
		return u, nil
	}
	return nil, identityV1.ErrorNotFound("user not found")
}

// newVisibleMenuTestFixture 造真 MenuRepo（SQLite 内存库）+ 按路径预置菜单，
// 返回 svc 装配、path→id 期望映射与 ent 客户端（删菜单用例需要）。
func newVisibleMenuTestFixture(t *testing.T, paths ...string) (*AiContentService, map[string]int64, *entCrud.EntClient[*ent.Client]) {
	t.Helper()
	entClient := enttest.NewEntClientForTest(t)
	menuRepo := data.NewMenuRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	pathToID := make(map[string]int64, len(paths))
	for i, p := range paths {
		row, err := entClient.Client().Menu.Create().
			SetName(fmt.Sprintf("Menu%d", i)).
			SetPath(p).
			SetType(menu.TypeMenu).
			Save(ctx)
		require.NoError(t, err, "预置菜单应成功")
		pathToID[p] = row.ID
	}

	svc := &AiContentService{
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		menuRepo:  menuRepo,
		entClient: entClient,
	}
	return svc, pathToID, entClient
}

// TestVisibleMenuIDs 平台管理员不过滤；租户用户按角色解析（全路径快照反解回 id）；
// 无角色返回空集；已删除菜单的快照被反解跳过。
func TestVisibleMenuIDs(t *testing.T) {
	svc, pathToID, entClient := newVisibleMenuTestFixture(t, "/vis/a", "/vis/b", "/vis/c", "/vis/gone")
	ctx := enttest.NewSystemContext(context.Background())

	roleRepo := &roleStubRepo{pathsByRoles: map[string][]string{
		"[1 2]": {"/vis/a", "/vis/b", "/vis/c"},
		"[1]":   {"/vis/a", "/vis/b"},
		"[2]":   {"/vis/b", "/vis/c", "/vis/gone"},
	}}
	svc.roleRepo = roleRepo
	svc.userRepo = &userStubRepo{usersByID: map[int64]*identityV1.User{
		7: {Id: trans.Ptr(int64(7)), RoleIds: []int64{1, 2}},
		8: {Id: trans.Ptr(int64(8)), RoleIds: nil},
	}}

	// 平台管理员：nil = 调用方不过滤
	ids, err := svc.visibleMenuIDs(context.Background(), &authenticationV1.UserTokenPayload{
		UserId: 1, TenantId: trans.Ptr(int64(0)),
	})
	require.NoError(t, err)
	require.Nil(t, ids, "平台管理员应返回 nil（不过滤）")

	// 租户用户：两角色并集去重（全路径反解回当前菜单 id）
	ids, err = svc.visibleMenuIDs(context.Background(), &authenticationV1.UserTokenPayload{
		UserId: 7, TenantId: trans.Ptr(int64(5)),
	})
	require.NoError(t, err)
	require.ElementsMatch(t, []int64{pathToID["/vis/a"], pathToID["/vis/b"], pathToID["/vis/c"]},
		ids, "应取角色菜单并集（反解回当前 id）")

	// 无角色用户：空集
	ids, err = svc.visibleMenuIDs(context.Background(), &authenticationV1.UserTokenPayload{
		UserId: 8, TenantId: trans.Ptr(int64(5)),
	})
	require.NoError(t, err)
	require.Empty(t, ids, "无角色用户应返回空集（AI 搜索不返回任何菜单）")

	// 已删除菜单的快照：反解时跳过，不虚报可见性（角色 [2] 持有 /vis/gone 路径）
	require.NoError(t, entClient.Client().Menu.DeleteOneID(pathToID["/vis/gone"]).Exec(ctx))
	svc.userRepo = &userStubRepo{usersByID: map[int64]*identityV1.User{
		10: {Id: trans.Ptr(int64(10)), RoleIds: []int64{2}},
	}}
	ids, err = svc.visibleMenuIDs(context.Background(), &authenticationV1.UserTokenPayload{
		UserId: 10, TenantId: trans.Ptr(int64(5)),
	})
	require.NoError(t, err)
	require.ElementsMatch(t, []int64{pathToID["/vis/b"], pathToID["/vis/c"]},
		ids, "已删除菜单的路径快照应被反解跳过，其余照常命中")
}

// TestSemanticSearchEmptyVisibleMenus 非平台用户且无可见菜单 → 直接空结果（不查库、不泄露）。
func TestSemanticSearchEmptyVisibleMenus(t *testing.T) {
	svc := &AiContentService{
		userRepo: &userStubRepo{usersByID: map[int64]*identityV1.User{
			9: {Id: trans.Ptr(int64(9)), TenantId: trans.Ptr(int64(5))},
		}},
	}
	ctx := auth.NewContext(context.Background(), &authenticationV1.UserTokenPayload{
		UserId: 9, TenantId: trans.Ptr(int64(5)),
	})
	resp, err := svc.SemanticSearch(ctx, &aiV1.SemanticSearchRequest{
		Query: "用户管理", Limit: trans.Ptr(uint32(8)),
	})
	require.NoError(t, err)
	require.Empty(t, resp.GetItems(), "无可见菜单应返回空结果")
}
