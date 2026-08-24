package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/sliceutil"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-admin/app/admin/service/internal/data"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"

	"go-wind-admin/pkg/middleware/auth"
)

type AdminPortalService struct {
	adminV1.AdminPortalServiceHTTPServer

	log *bLogger.Helper

	menuRepo        *data.MenuRepo
	roleRepo        *data.RoleRepo
	userRepo        data.UserRepo
	permissionRepo  *data.PermissionRepo
	planModuleRepo  *data.PlanModuleRepo
	tenantRepo      *data.TenantRepo
}

func NewAdminPortalService(
	ctx *bootstrap.Context,
	menuRepo *data.MenuRepo,
	roleRepo *data.RoleRepo,
	userRepo data.UserRepo,
	permissionRepo *data.PermissionRepo,
	planModuleRepo *data.PlanModuleRepo,
	tenantRepo *data.TenantRepo,
) *AdminPortalService {
	return &AdminPortalService{
		log:            ctx.NewLoggerHelper("admin-portal/service/admin-service"),
		menuRepo:       menuRepo,
		roleRepo:       roleRepo,
		userRepo:       userRepo,
		permissionRepo: permissionRepo,
		planModuleRepo: planModuleRepo,
		tenantRepo:      tenantRepo,
	}
}

func (s *AdminPortalService) menuListToQueryString(menus []uint32, onlyButton bool) string {
	var ids []string
	for _, menu := range menus {
		ids = append(ids, fmt.Sprintf("\"%d\"", menu))
	}
	idsStr := fmt.Sprintf("[%s]", strings.Join(ids, ", "))
	query := map[string]string{"id__in": idsStr}

	if onlyButton {
		query["type"] = permissionV1.Menu_BUTTON.String()
	} else {
		query["type__not"] = permissionV1.Menu_BUTTON.String()
	}

	query["status"] = "ON"

	queryStr, err := json.Marshal(query)
	if err != nil {
		return ""
	}

	return string(queryStr)
}

// queryMultipleRolesMenusByRoleCodes 使用RoleCodes查询菜单，即多个角色的菜单
func (s *AdminPortalService) queryMultipleRolesMenusByRoleCodes(ctx context.Context, roleIDs []uint32) ([]uint32, error) {
	var menuIDs []uint32
	var err error
	menuIDs, err = s.roleRepo.GetRolesPermissionMenuIDs(ctx, roleIDs)
	if err != nil {
		return nil, adminV1.ErrorInternalServerError("query roles menuIDs failed")
	}

	s.log.Infof(ctx, "queryMultipleRolesMenusByRoleCodes menuIDs: %+v", menuIDs)

	menuIDs = sliceutil.Unique(menuIDs)

	return menuIDs, nil
}
func (s *AdminPortalService) GetMyPermissionCode(ctx context.Context, _ *emptypb.Empty) (*adminV1.ListPermissionCodeResponse, error) {
	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.Get(ctx, &identityV1.GetUserRequest{
		QueryBy: &identityV1.GetUserRequest_Id{
			Id: operator.UserId,
		},
	})
	if err != nil {
		s.log.Errorf(ctx, "query user failed[%s]", err.Error())
		return nil, adminV1.ErrorInternalServerError("query user failed")
	}

	permissionIDs, err := s.roleRepo.ListPermissionIDsByRoleIDs(ctx, user.GetRoleIds())
	if err != nil {
		return nil, err
	}

	var permissionCodes []string
	permissionCodes, err = s.permissionRepo.GetPermissionCodesByIDs(ctx, permissionIDs)
	if err != nil {
		return nil, err
	}

	return &adminV1.ListPermissionCodeResponse{
		Codes: permissionCodes,
	}, nil
}

func (s *AdminPortalService) fillRouteItem(menus []*permissionV1.Menu) []*permissionV1.MenuRouteItem {
	if len(menus) == 0 {
		return nil
	}

	var routers []*permissionV1.MenuRouteItem

	for _, v := range menus {
		if v.GetStatus() != permissionV1.Menu_ON {
			continue
		}
		if v.GetType() == permissionV1.Menu_BUTTON {
			continue
		}

		item := &permissionV1.MenuRouteItem{
			Path:      v.Path,
			Component: v.Component,
			Name:      v.Name,
			Redirect:  v.Redirect,
			Alias:     v.Alias,
			Meta:      v.Meta,
		}

		if len(v.Children) > 0 {
			item.Children = s.fillRouteItem(v.Children)
		}

		routers = append(routers, item)
	}

	return routers
}

func (s *AdminPortalService) GetNavigation(ctx context.Context, _ *emptypb.Empty) (*adminV1.ListRouteResponse, error) {
	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.Get(ctx, &identityV1.GetUserRequest{
		QueryBy: &identityV1.GetUserRequest_Id{
			Id: operator.UserId,
		},
	})
	if err != nil {
		s.log.Errorf(ctx, "query user failed[%s]", err.Error())
		return nil, adminV1.ErrorInternalServerError("query user failed")
	}

	// 多角色的菜单
	roleMenus, err := s.queryMultipleRolesMenusByRoleCodes(ctx, user.GetRoleIds())
	if err != nil {
		return nil, err
	}

	menuList, err := s.menuRepo.List(ctx, &paginationV1.PagingRequest{
		NoPaging: trans.Ptr(true),
		FilteringType: &paginationV1.PagingRequest_Query{
			Query: s.menuListToQueryString(roleMenus, false),
		},
	}, true)
	if err != nil {
		s.log.Errorf(ctx, "list route failed [%s]", err.Error())
		return nil, adminV1.ErrorInternalServerError("list route failed")
	}

	// 套餐模块白名单过滤：仅对租户用户（tenantId>0）生效。
	// 平台管理员（tenantId=0）不过滤，看到全部授权菜单。
	tid := operator.GetTenantId()
	if tid > 0 {
		menuList.Items = s.filterMenusByPlanWhitelist(ctx, menuList.Items, tid)
	}

	return &adminV1.ListRouteResponse{Items: s.fillRouteItem(menuList.Items)}, nil
}

// filterMenusByPlanWhitelist 按当前租户订阅套餐的模块白名单过滤菜单。
// 菜单的 module 字段不在白名单内（或套餐无白名单）则剔除。
func (s *AdminPortalService) filterMenusByPlanWhitelist(ctx context.Context, items []*permissionV1.Menu, tenantId uint32) []*permissionV1.Menu {
	// 查租户的 plan_id
	t, err := s.tenantRepo.Get(ctx, &identityV1.GetTenantRequest{
		QueryBy: &identityV1.GetTenantRequest_Id{
			Id: tenantId,
		},
	})
	if err != nil || t == nil || t.PlanId == nil || *t.PlanId == 0 {
		// 无套餐或查询失败 → 清空菜单（租户无任何模块权限）
		return nil
	}
	planId := *t.PlanId

	// 查该套餐的白名单模块集合
	allowed, err := s.planModuleRepo.ListModulesByPlanId(ctx, planId)
	if err != nil || len(allowed) == 0 {
		return nil
	}

	filtered := make([]*permissionV1.Menu, 0, len(items))
	for _, m := range items {
		if m.Module == nil {
			// catalog 容器或未归类：保留（容器本身无业务功能，其子项会被各自过滤）
			filtered = append(filtered, m)
			continue
		}
		if allowed[*m.Module] {
			filtered = append(filtered, m)
		}
	}
	return filtered
}
