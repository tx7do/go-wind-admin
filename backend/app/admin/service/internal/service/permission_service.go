package service

import (
	"context"
	"sort"
	"strings"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/go-utils/aggregator"
	"google.golang.org/protobuf/types/known/emptypb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/stringcase"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"go-wind-admin/app/admin/service/internal/data"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"

	"go-wind-admin/pkg/authorizer"
	"go-wind-admin/pkg/constants"
	appViewer "go-wind-admin/pkg/entgo/viewer"
	"go-wind-admin/pkg/middleware/auth"
	"go-wind-admin/pkg/utils/converter"
)

type PermissionService struct {
	adminV1.PermissionServiceHTTPServer

	log *log.Helper

	permissionRepo      *data.PermissionRepo
	permissionGroupRepo *data.PermissionGroupRepo

	menuRepo *data.MenuRepo
	apiRepo  *data.ApiRepo

	roleRepo *data.RoleRepo

	authorizer *authorizer.Authorizer

	menuPermissionConverter *converter.MenuPermissionConverter
	apiPermissionConverter  *converter.ApiPermissionConverter
}

func NewPermissionService(
	ctx *bootstrap.Context,
	permissionRepo *data.PermissionRepo,
	permissionGroupRepo *data.PermissionGroupRepo,
	menuRepo *data.MenuRepo,
	apiRepo *data.ApiRepo,
	roleRepo *data.RoleRepo,
	authorizer *authorizer.Authorizer,
) *PermissionService {
	svc := &PermissionService{
		log:                     ctx.NewLoggerHelper("permission/service/admin-service"),
		permissionRepo:          permissionRepo,
		permissionGroupRepo:     permissionGroupRepo,
		menuRepo:                menuRepo,
		apiRepo:                 apiRepo,
		roleRepo:                roleRepo,
		authorizer:              authorizer,
		menuPermissionConverter: converter.NewMenuPermissionConverter(),
		apiPermissionConverter:  converter.NewApiPermissionConverter(),
	}

	svc.init()

	return svc
}

func (s *PermissionService) init() {
	ctx := appViewer.NewSystemViewerContext(context.Background())
	if count, _ := s.permissionRepo.Count(ctx, []func(s *sql.Selector){}); count == 0 {
		_ = s.createDefaultPermissions(ctx)

		var apiCount int
		apiCount, _ = s.apiRepo.Count(ctx, []func(s *sql.Selector){})

		var menusCount int
		menusCount, _ = s.menuRepo.Count(ctx, []func(s *sql.Selector){})

		if apiCount > 0 && menusCount > 0 {
			_, _ = s.SyncPermissions(ctx, &emptypb.Empty{})
		}
	}
}

func (s *PermissionService) extractRelationIDs(
	permissions []*permissionV1.Permission,
	groupSet aggregator.ResourceMap[uint32, *permissionV1.PermissionGroup],
) {
	for _, p := range permissions {
		if p.GetGroupId() > 0 {
			groupSet[p.GetGroupId()] = nil
		}
	}
}

func (s *PermissionService) fetchRelationInfo(
	ctx context.Context,
	groupSet aggregator.ResourceMap[uint32, *permissionV1.PermissionGroup],
) error {
	if len(groupSet) > 0 {
		groupIds := make([]uint32, 0, len(groupSet))
		for id := range groupSet {
			groupIds = append(groupIds, id)
		}

		groups, err := s.permissionGroupRepo.ListByIDs(ctx, groupIds)
		if err != nil {
			s.log.Errorf("query permission group err: %v", err)
			return err
		}

		for _, g := range groups {
			groupSet[g.GetId()] = g
		}
	}

	return nil
}

func (s *PermissionService) populateRelationInfos(
	permissions []*permissionV1.Permission,
	groupSet aggregator.ResourceMap[uint32, *permissionV1.PermissionGroup],
) {
	aggregator.Populate(
		permissions,
		groupSet,
		func(ou *permissionV1.Permission) uint32 { return ou.GetGroupId() },
		func(ou *permissionV1.Permission, g *permissionV1.PermissionGroup) {
			ou.GroupName = g.Name
		},
	)
}

func (s *PermissionService) List(ctx context.Context, req *paginationV1.PagingRequest) (*permissionV1.ListPermissionResponse, error) {
	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	var limitPermissionIDs []uint32
	if operator.GetTenantId() > 0 {
		limitPermissionIDs, err = s.roleRepo.ListPermissionIDsByRoleCodes(ctx, operator.GetRoles())
		if err != nil {
			return nil, err
		}

		// 没有任何 permission 可访问，直接返回空列表
		if len(limitPermissionIDs) == 0 {
			return &permissionV1.ListPermissionResponse{
				Items: []*permissionV1.Permission{},
				Total: 0,
			}, nil
		}
	}

	resp, err := s.permissionRepo.List(ctx, req, limitPermissionIDs)
	if err != nil {
		return nil, err
	}

	var groupSet = make(aggregator.ResourceMap[uint32, *permissionV1.PermissionGroup])
	s.extractRelationIDs(resp.Items, groupSet)
	_ = s.fetchRelationInfo(ctx, groupSet)
	s.populateRelationInfos(resp.Items, groupSet)

	return resp, nil
}

func (s *PermissionService) Get(ctx context.Context, req *permissionV1.GetPermissionRequest) (*permissionV1.Permission, error) {
	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := s.permissionRepo.Get(ctx, req)
	if err != nil {
		return nil, err
	}

	if operator.GetTenantId() > 0 {
		permissionIDs, err := s.roleRepo.ListPermissionIDsByRoleCodes(ctx, operator.GetRoles())
		if err != nil {
			return nil, err
		}

		found := false
		for _, pid := range permissionIDs {
			if pid == resp.GetId() {
				found = true
				break
			}
		}
		if !found {
			return nil, adminV1.ErrorForbidden("no access to the permission")
		}
	}

	fakeItems := []*permissionV1.Permission{resp}
	var groupSet = make(aggregator.ResourceMap[uint32, *permissionV1.PermissionGroup])
	s.extractRelationIDs(fakeItems, groupSet)
	_ = s.fetchRelationInfo(ctx, groupSet)
	s.populateRelationInfos(fakeItems, groupSet)

	return resp, nil
}

func (s *PermissionService) Create(ctx context.Context, req *permissionV1.CreatePermissionRequest) (*emptypb.Empty, error) {
	if req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.CreatedBy = trans.Ptr(operator.UserId)

	if err = s.permissionRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	// 重置权限策略
	if err = s.authorizer.ResetPolicies(ctx); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *PermissionService) Update(ctx context.Context, req *permissionV1.UpdatePermissionRequest) (*emptypb.Empty, error) {
	if req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.UpdatedBy = trans.Ptr(operator.UserId)
	if req.UpdateMask != nil {
		req.UpdateMask.Paths = append(req.UpdateMask.Paths, "updated_by")
	}

	if err = s.permissionRepo.Update(ctx, req); err != nil {
		return nil, err
	}

	// 重置权限策略
	if err = s.authorizer.ResetPolicies(ctx); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *PermissionService) Delete(ctx context.Context, req *permissionV1.DeletePermissionRequest) (*emptypb.Empty, error) {
	if err := s.permissionRepo.Delete(ctx, req); err != nil {
		return nil, err
	}

	// 重置权限策略
	if err := s.authorizer.ResetPolicies(ctx); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// appendAPis 为权限追加对应的 API 资源 ID 列表
func (s *PermissionService) appendAPis(
	ctx context.Context,
	permissions *[]*permissionV1.Permission,
	mapPermissions *map[string][]*permissionV1.Permission,
	operatorUserId uint32,
) error {
	// 查询所有启用的 API 资源
	apis, err := s.apiRepo.List(ctx, &paginationV1.PagingRequest{
		NoPaging: trans.Ptr(true),
		FilteringType: &paginationV1.PagingRequest_Query{
			Query: `{"status":"ON"}`,
		},
		OrderBy: trans.Ptr("operation"),
	})
	if err != nil {
		return err
	}

	sort.SliceStable(apis.Items, func(i, j int) bool {
		a, b := apis.Items[i], apis.Items[j]
		if a.GetModule() != b.GetModule() {
			return a.GetModule() < b.GetModule()
		}
		if a.GetPath() != b.GetPath() {
			return a.GetPath() > b.GetPath()
		}
		return a.GetOperation() > b.GetOperation()
	})

	type moduleApis struct {
		module string
		apis   []uint32
	}

	codes := make(map[string]*moduleApis)
	for _, api := range apis.Items {
		//code := s.apiPermissionConverter.ConvertCodeByOperationID(api.GetOperation())
		code := s.apiPermissionConverter.ConvertCodeByPath(api.GetMethod(), api.GetPath())
		if code == "" {
			continue
		}

		s.log.Debugf("appendAPis: processing api [%s] [%s] with code [%s]", api.GetMethod(), api.GetPath(), code)

		if curCode, exist := codes[code]; !exist {
			var module string
			for k, perms := range *mapPermissions {
				if len(perms) == 0 {
					continue
				}

				for _, perm := range perms {
					code1Prefix := strings.Split(perm.GetCode(), ":")[0]
					code2Prefix := strings.Split(code, ":")[0]
					if strings.HasPrefix(code2Prefix, code1Prefix) {
						module = k
						break
					}
				}
			}

			if module == "" {
				module = constants.UncategorizedPermissionGroup
			}

			codes[code] = &moduleApis{
				module: module,
				apis:   []uint32{api.GetId()},
			}
		} else {
			curCode.apis = append(curCode.apis, api.GetId())
		}
	}

	for _, perm := range *permissions {
		permIds, exist := codes[perm.GetCode()]
		if exist {
			perm.ApiIds = append(perm.ApiIds, permIds.apis...)
			delete(codes, perm.GetCode())
		}
	}

	//s.log.Debugf("appendAPis: unmatched permission codes: %v", codes)

	for code, apiIDs := range codes {
		name := strings.ReplaceAll(code, ":", "_")
		name = stringcase.ToPascalCase(name)

		perm := &permissionV1.Permission{
			Name:      trans.Ptr(name),
			Code:      trans.Ptr(code),
			Status:    trans.Ptr(permissionV1.Permission_ON),
			ApiIds:    apiIDs.apis,
			CreatedBy: trans.Ptr(operatorUserId),
			UpdatedBy: trans.Ptr(operatorUserId),
		}

		*permissions = append(*permissions, perm)

		(*mapPermissions)[apiIDs.module] = append((*mapPermissions)[apiIDs.module], perm)

		//s.log.Debugf("appendAPis: create permission for api code: [%s][%s]", apiIDs.module, code)
	}

	return nil
}

// menuPathToModuleName 从菜单路径中提取模块名称
func menuPathToModuleName(menuPath string) string {
	var module string
	pathParts := strings.Split(menuPath, "/")
	if len(pathParts) > 1 {
		module = strings.TrimSpace(pathParts[1])
	}
	if module == "" {
		module = constants.DefaultBizPermissionModule
	}
	return module
}

// SyncPermissions 同步权限点
func (s *PermissionService) SyncPermissions(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	// 清理菜单相关权限
	_ = s.permissionRepo.TruncateBizPermissions(ctx)
	_ = s.permissionGroupRepo.TruncateBizGroup(ctx)

	// 查询所有启用的菜单
	menus, err := s.menuRepo.List(ctx, &paginationV1.PagingRequest{
		NoPaging: trans.Ptr(true),
		FilteringType: &paginationV1.PagingRequest_Query{
			Query: `{"status":"ON"}`,
		},
		OrderBy: trans.Ptr("id desc"),
	}, false)
	if err != nil {
		return nil, err
	}

	s.menuPermissionConverter.ComposeMenuPaths(menus.Items)

	sort.SliceStable(menus.Items, func(i, j int) bool {
		return menus.Items[i].GetParentId() < menus.Items[j].GetParentId()
	})

	var permissionGroups []*permissionV1.PermissionGroup
	var permissions []*permissionV1.Permission
	var mapPermissions = make(map[string][]*permissionV1.Permission)

	permissionGroups = append(permissionGroups, &permissionV1.PermissionGroup{
		Name:      trans.Ptr("未分类"),
		Module:    trans.Ptr(constants.UncategorizedPermissionGroup),
		Status:    trans.Ptr(permissionV1.PermissionGroup_ON),
		SortOrder: trans.Ptr(uint32(len(permissionGroups) + 1)),
		CreatedBy: trans.Ptr(operator.UserId),
		UpdatedBy: trans.Ptr(operator.UserId),
	})

	for _, menu := range menus.Items {
		var title string
		//if menu.GetMeta() != nil && menu.GetMeta().GetTitle() != "" {
		//	title = menu.GetMeta().GetTitle()
		//} else {
		//	title = menu.GetName()
		//}
		title = menu.GetName()

		var permissionCode string
		permissionCode = s.menuPermissionConverter.ConvertCode(menu.GetPath(), title, menu.GetType())
		if permissionCode == "" {
			continue
		}

		module := menuPathToModuleName(menu.GetPath())

		// 以目录类型的菜单作为权限组
		if menu.GetType() == permissionV1.Menu_CATALOG {
			permissionGroups = append(permissionGroups, &permissionV1.PermissionGroup{
				Name:      trans.Ptr(title),
				Module:    trans.Ptr(module),
				Status:    trans.Ptr(permissionV1.PermissionGroup_ON),
				SortOrder: trans.Ptr(uint32(len(permissionGroups) + 1)),
				CreatedBy: trans.Ptr(operator.UserId),
				UpdatedBy: trans.Ptr(operator.UserId),
			})
			//s.log.Debugf("SyncPermissions: created permission group for menu %s - %s", menu.GetName(), permissionCode)
		}

		perm := &permissionV1.Permission{
			Name:      trans.Ptr(title),
			Code:      trans.Ptr(permissionCode),
			Status:    trans.Ptr(permissionV1.Permission_ON),
			MenuIds:   []uint32{menu.GetId()},
			CreatedBy: trans.Ptr(operator.UserId),
			UpdatedBy: trans.Ptr(operator.UserId),
		}

		permissions = append(permissions, perm)

		mapPermissions[module] = append(mapPermissions[module], perm)
	}

	// 为权限追加对应的 API 资源 ID 列表
	_ = s.appendAPis(ctx, &permissions, &mapPermissions, operator.UserId)

	var finalPermissionGroups []*permissionV1.PermissionGroup
	if finalPermissionGroups, err = s.permissionGroupRepo.BatchCreate(ctx, permissionGroups); err != nil {
		s.log.Errorf("batch create permission groups failed: %s", err.Error())
		return nil, err
	}
	//for _, pg := range permissionGroups {
	//	var npg *permissionV1.PermissionGroup
	//	if npg, err = s.permissionGroupRepo.Create(ctx, &permissionV1.CreatePermissionGroupRequest{Data: pg}); err != nil {
	//		s.log.Errorf("batch create permission groups failed: %s", err.Error())
	//		return nil, err
	//	}
	//	finalPermissionGroups = append(finalPermissionGroups, npg)
	//}

	// 为权限分配权限组 ID
	for _, pg := range finalPermissionGroups {
		curPers := mapPermissions[pg.GetModule()]
		for _, p := range curPers {
			p.GroupId = pg.Id
		}
	}

	if err = s.permissionRepo.BatchCreate(ctx, permissions); err != nil {
		s.log.Errorf("batch create permissions failed: %s", err.Error())
		return nil, err
	}

	// 重置权限策略
	if err = s.authorizer.ResetPolicies(ctx); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *PermissionService) createDefaultPermissions(ctx context.Context) error {
	var err error

	for _, d := range constants.DefaultPermissions {
		if err = s.permissionRepo.Create(ctx, &permissionV1.CreatePermissionRequest{
			Data: d,
		}); err != nil {
			s.log.Errorf("create default permission %s failed: %v", d.GetCode(), err)
			return err
		}
	}

	return nil
}
