package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-kratos/kratos/v2/log"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/sliceutil"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"go-wind-admin/app/admin/service/internal/data"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	resourceV1 "go-wind-admin/api/gen/go/resource/service/v1"

	"go-wind-admin/pkg/middleware/auth"
)

type AdminPortalService struct {
	adminV1.AdminPortalServiceHTTPServer

	log *log.Helper

	menuRepo *data.MenuRepo
	roleRepo *data.RoleRepo
	userRepo data.UserRepo
}

func NewAdminPortalService(
	ctx *bootstrap.Context,
	menuRepo *data.MenuRepo,
	roleRepo *data.RoleRepo,
	userRepo data.UserRepo,
) *AdminPortalService {
	return &AdminPortalService{
		log:      ctx.NewLoggerHelper("admin-portal/service/admin-service"),
		menuRepo: menuRepo,
		roleRepo: roleRepo,
		userRepo: userRepo,
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
func (s *AdminPortalService) queryMultipleRolesMenusByRoleCodes(ctx context.Context, roleCodes []string) ([]uint32, error) {
	roleIDs, err := s.roleRepo.ListRoleIDsByRoleCodes(ctx, roleCodes)
	if err != nil {
		return nil, adminV1.ErrorInternalServerError("query roles failed")
	}

	var menus []uint32

	menus, err = s.roleRepo.GetRolesPermissionMenuIDs(ctx, roleIDs)
	if err != nil {
		return nil, adminV1.ErrorInternalServerError("query roles menus failed")
	}

	menus = sliceutil.Unique(menus)

	return menus, nil
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

	s.log.Debugf("user info: %+v", user.GetRoleIds())

	// 多角色的菜单
	roleMenus, err := s.queryMultipleRolesMenusByRoleIds(ctx, user.GetRoleIds())
	if err != nil {
		return nil, err
	}

	menus, err := s.menuRepo.List(ctx, &paginationV1.PagingRequest{
		NoPaging: trans.Ptr(true),
		FilteringType: &paginationV1.PagingRequest_Query{
			Query: s.menuListToQueryString(roleMenus, true),
		},
		FieldMask: &fieldmaskpb.FieldMask{
			Paths: []string{"id", "meta"},
		},
	}, false)
	if err != nil {
		s.log.Errorf("list permission code failed [%s]", err.Error())
		return nil, adminV1.ErrorInternalServerError("list permission code failed")
	}

	var codes []string
	for menu := range menus.Items {
		if menus.Items[menu].GetMeta() == nil {
			continue
		}
		if len(menus.Items[menu].GetMeta().GetAuthority()) == 0 {
			continue
		}

		codes = append(codes, menus.Items[menu].GetMeta().GetAuthority()...)
	}

	return &adminV1.ListPermissionCodeResponse{
		Codes: codes,
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

	// 单角色的菜单
	//roleMenus, err := s.queryOneRoleMenus(ctx, user.GetRoleId())
	//if err != nil {
	//	return nil, err
	//}

	// 多角色的菜单
	roleMenus, err := s.queryMultipleRolesMenusByRoleCodes(ctx, user.GetRoles())
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

	resp := &adminV1.ListRouteResponse{Items: s.fillRouteItem(menuList.Items)}

	return resp, nil
}
