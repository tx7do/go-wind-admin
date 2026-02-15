package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-kratos/kratos/v2/log"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/sliceutil"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-admin/app/admin/service/internal/data"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	resourceV1 "go-wind-admin/api/gen/go/resource/service/v1"

	"go-wind-admin/pkg/middleware/auth"
)

type AdminPortalService struct {
	adminV1.AdminPortalServiceHTTPServer

	log *log.Helper

	menuRepo       *data.MenuRepo
	roleRepo       *data.RoleRepo
	userRepo       data.UserRepo
	permissionRepo *data.PermissionRepo
}

func NewAdminPortalService(
	ctx *bootstrap.Context,
	menuRepo *data.MenuRepo,
	roleRepo *data.RoleRepo,
	userRepo data.UserRepo,
	permissionRepo *data.PermissionRepo,
) *AdminPortalService {
	return &AdminPortalService{
		log:            ctx.NewLoggerHelper("admin-portal/service/admin-service"),
		menuRepo:       menuRepo,
		roleRepo:       roleRepo,
		userRepo:       userRepo,
		permissionRepo: permissionRepo,
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
		query["type"] = resourceV1.Menu_BUTTON.String()
	} else {
		query["type__not"] = resourceV1.Menu_BUTTON.String()
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

	s.log.Infof("queryMultipleRolesMenusByRoleCodes menuIDs: %+v", menuIDs)

	menuIDs = sliceutil.Unique(menuIDs)

	return menuIDs, nil
}

// queryMultipleRolesMenusByRoleIds 使用RoleIDs查询菜单，即多个角色的菜单
func (s *AdminPortalService) queryMultipleRolesMenusByRoleIds(ctx context.Context, roleIDs []uint32) ([]uint32, error) {
	menus, err := s.roleRepo.GetRolesPermissionMenuIDs(ctx, roleIDs)
	if err != nil {
		return nil, adminV1.ErrorInternalServerError("query roles menus failed")
	}

	menus = sliceutil.Unique(menus)

	return menus, nil
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
		s.log.Errorf("query user failed[%s]", err.Error())
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

func (s *AdminPortalService) fillRouteItem(menus []*resourceV1.Menu) []*resourceV1.MenuRouteItem {
	if len(menus) == 0 {
		return nil
	}

	var routers []*resourceV1.MenuRouteItem

	for _, v := range menus {
		if v.GetStatus() != resourceV1.Menu_ON {
			continue
		}
		if v.GetType() == resourceV1.Menu_BUTTON {
			continue
		}

		item := &resourceV1.MenuRouteItem{
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
		s.log.Errorf("query user failed[%s]", err.Error())
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
		s.log.Errorf("list route failed [%s]", err.Error())
		return nil, adminV1.ErrorInternalServerError("list route failed")
	}

	return &adminV1.ListRouteResponse{Items: s.fillRouteItem(menuList.Items)}, nil
}
