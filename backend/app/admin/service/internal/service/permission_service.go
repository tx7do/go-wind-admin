package service

import (
	"context"
	"sort"
	"strings"

	"entgo.io/ent/dialect/sql"
	"github.com/tx7do/go-utils/aggregator"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/stringcase"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"go-wind-admin/app/admin/service/internal/data"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"

	"github.com/tx7do/go-crud/viewer"
	"github.com/tx7do/go-utils/authorizer"
	goconv "github.com/tx7do/go-utils/converter"
	"go-wind-admin/pkg/constants"
	"go-wind-admin/pkg/middleware/auth"
	"go-wind-admin/pkg/utils/converter"
)

type PermissionService struct {
	adminV1.PermissionServiceHTTPServer

	log *bLogger.Helper

	permissionRepo      *data.PermissionRepo
	permissionGroupRepo *data.PermissionGroupRepo

	menuRepo *data.MenuRepo
	apiRepo  *data.ApiRepo

	roleRepo *data.RoleRepo

	authorizer *authorizer.Authorizer

	menuPermissionConverter *converter.MenuPermissionConverter
	apiPermissionConverter  *goconv.ApiPermissionConverter
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
		apiPermissionConverter:  goconv.NewApiPermissionConverter(),
	}

	svc.init()

	return svc
}

func (s *PermissionService) init() {
	ctx := viewer.WithSystemContext(context.Background())
	if count, _ := s.permissionRepo.Count(ctx, nil); count.Count == 0 {
		_ = s.createDefaultPermissions(ctx)

		apiCount, _ := s.apiRepo.Count(ctx, nil)

		var menusCount int
		menusCount, _ = s.menuRepo.Count(ctx, []func(s *sql.Selector){})

		if apiCount.Count > 0 && menusCount > 0 {
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
			s.log.Errorf(context.Background(), "query permission group err: %v", err)
			return err
		}

		for _, g := range groups {
			groupSet[g.GetId()] = g
		}
	}

	return nil
}

func (s *PermissionService) bindRelations(
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

func (s *PermissionService) enrichRelations(ctx context.Context, permissions []*permissionV1.Permission) error {
	var groupSet = make(aggregator.ResourceMap[uint32, *permissionV1.PermissionGroup])
	s.extractRelationIDs(permissions, groupSet)
	if err := s.fetchRelationInfo(ctx, groupSet); err != nil {
		return err
	}
	s.bindRelations(permissions, groupSet)
	return nil
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

	_ = s.enrichRelations(ctx, resp.Items)

	return resp, nil
}

func (s *PermissionService) Count(ctx context.Context, req *paginationV1.PagingRequest) (*permissionV1.CountPermissionResponse, error) {
	return s.permissionRepo.Count(ctx, req)
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
	_ = s.enrichRelations(ctx, fakeItems)

	return resp, nil
}

func (s *PermissionService) Create(ctx context.Context, req *permissionV1.CreatePermissionRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.CreatedBy = trans.Ptr(operator.UserId)

	// 权限是全局定义（权限→API 绑定影响全平台的 authz 策略装载），租户操作者
	// 不可增删改；平台侧也不得占用 sys: 保留码——sys:access_backend 等在登录
	// 闸门按码值精确匹配，自建同码权限即可伪造登录资格。
	if err = requirePlatformAdmin(ctx, s.log, "permission management"); err != nil {
		return nil, err
	}
	if constants.IsReservedPermissionCode(req.Data.GetCode()) {
		return nil, adminV1.ErrorBadRequest("permission code is reserved")
	}

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
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.Id = trans.Ptr(req.GetId())

	req.Data.UpdatedBy = trans.Ptr(operator.UserId)
	if req.UpdateMask != nil {
		req.UpdateMask.Paths = append(req.UpdateMask.Paths, "updated_by")
	}

	// 与 Create 同口径：全局定义仅平台管理员可改；保留权限码不可**占用**——
	// 防的是把自建权限的 code 改成 sys: 等保留码（伪造登录闸门按码匹配的资格）。
	// 本就持有保留码的系统权限行（提交 code 与库存一致）做勾菜单/接口等合法
	// 更新必须放行——否则"权限管理页勾新菜单"的运维指引永远 400（前端走查实锤）。
	if err = requirePlatformAdmin(ctx, s.log, "permission management"); err != nil {
		return nil, err
	}
	if req.Data.Code != nil && constants.IsReservedPermissionCode(req.Data.GetCode()) {
		existing, gerr := s.permissionRepo.Get(ctx, &permissionV1.GetPermissionRequest{
			QueryBy: &permissionV1.GetPermissionRequest_Id{Id: req.GetId()},
		})
		if gerr != nil {
			return nil, gerr
		}
		if existing.GetCode() != req.Data.GetCode() {
			return nil, adminV1.ErrorBadRequest("permission code is reserved")
		}
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
	// 全局定义仅平台管理员可删（受保护码另有 ProtectedPermissionCodes 拦截）。
	if err := requirePlatformAdmin(ctx, s.log, "permission management"); err != nil {
		return nil, err
	}

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

		s.log.Debugf(ctx, "appendAPis: processing api [%s] [%s] with code [%s]", api.GetMethod(), api.GetPath(), code)

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

	//s.log.Debugf(ctx, "appendAPis: unmatched permission codes: %v", codes)

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

		//s.log.Debugf(ctx, "appendAPis: create permission for api code: [%s][%s]", apiIDs.module, code)
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

// SyncPermissions 同步权限点。
//
// 按自然键 code 的 **MERGE 增量对齐**（与接口同步/菜单同步 MERGE 同一教义，仓内
// 已不存在删除重建型同步）：派生集（菜单/接口 → 权限码）里 code 已存在的权限行
// 原位更新（**保 id**——sys_role_permissions 角色授权、sys_permission_policies
// 策略联动、租户模板副本引用全部因此免于漂移），缺失才插入；库里有、派生集没有
// **且码已无法从任何现存菜单派生**（含停用菜单）的业务权限才删除并级联清理其
// 角色授权/端点/菜单绑定/策略行——管理员临时停用菜单不触发删权限。管理端手工
// 停用的权限 status 不被同步触碰（与菜单同步 MERGE 同口径）。
// 分组行无授权绑定引用，保持原重建语义（TruncateBizGroup + 重插）。
// 注：init() 的启动期自动同步本就以系统上下文无操作者而静默跳过（播种走
// createDefaultPermissions），此处加闸门不改变启动期行为。
func (s *PermissionService) SyncPermissions(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	if err := requirePlatformAdmin(ctx, s.log, "permission management"); err != nil {
		return nil, err
	}

	// 获取操作人信息（同步落库的 created_by/updated_by 用）
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	// 同步是系统级全量操作，数据读写统一走 SystemContext：平台管理员的用户
	// viewer 会把基线查询（ListBizPermissions 等）按租户过滤成空——首轮表空
	// 不可见，二轮起基线恒空、派生集全判"新增"而撞 code 唯一约束（实测）。
	// 与 authorizer.RequirePolicies 的"装载统一以 SystemContext 跑"同一教义。
	ctx = viewer.WithSystemContext(ctx)

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
	// 不同层级的菜单可能推导出同一个权限码（如菜单同步 MERGE 后新旧路径形态
	// 并存），code 唯一约束会让整批插入失败；同码只保留首个，后续跳过
	var seenPermissionCodes = make(map[string]struct{})

	permissionGroups = append(permissionGroups, &permissionV1.PermissionGroup{
		Name:      trans.Ptr("未分类"),
		Module:    trans.Ptr(constants.UncategorizedPermissionGroup),
		Status:    trans.Ptr(permissionV1.PermissionGroup_ON),
		SortOrder: trans.Ptr(uint32(len(permissionGroups) + 1)),
		CreatedBy: trans.Ptr(operator.UserId),
		UpdatedBy: trans.Ptr(operator.UserId),
	})

	for _, menu := range menus.Items {
		title := menu.GetName()

		permissionCode := s.menuPermissionConverter.ConvertCode(menu.GetPath(), title, menu.GetType())
		if permissionCode == "" {
			continue
		}
		if _, exists := seenPermissionCodes[permissionCode]; exists {
			continue
		}
		seenPermissionCodes[permissionCode] = struct{}{}

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
			//s.log.Debugf(ctx, "SyncPermissions: created permission group for menu %s - %s", menu.GetName(), permissionCode)
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

	// 组表保持"截断重建"语义（组行无授权绑定引用，权限行的 group_id 会在下方
	// MERGE 更新段原位改写到新组 id）。**必须先截断再重插**：name 只有普通索引
	// 没有唯一约束，不截断的话每跑一次同步整个组表翻倍一份。
	if err = s.permissionGroupRepo.TruncateBizGroup(ctx); err != nil {
		return nil, err
	}

	var finalPermissionGroups []*permissionV1.PermissionGroup
	if finalPermissionGroups, err = s.permissionGroupRepo.BatchCreate(ctx, permissionGroups); err != nil {
		s.log.Errorf(ctx, "batch create permission groups failed: %s", err.Error())
		return nil, err
	}

	// 为权限分配权限组 ID
	for _, pg := range finalPermissionGroups {
		curPers := mapPermissions[pg.GetModule()]
		for _, p := range curPers {
			p.GroupId = pg.Id
		}
	}

	// —— MERGE 分段：按 code 对齐派生集与库存 ——
	derivedByCode := make(map[string]*permissionV1.Permission, len(permissions))
	for _, p := range permissions {
		derivedByCode[p.GetCode()] = p
	}

	existing, err := s.permissionRepo.ListBizPermissions(ctx)
	if err != nil {
		return nil, err
	}
	existingByCode := make(map[string]*permissionV1.Permission, len(existing))
	for _, e := range existing {
		existingByCode[e.GetCode()] = e
	}
	// 更新段：code 已存在 → 原位更新（**保 id**：角色授权/策略联动/租户模板副本
	// 引用全部因此免于漂移）。不触碰 status（与菜单同步 MERGE 同口径：管理端
	// 手工停用的权限不被同步重新启用）；绑定（端点/菜单快照）按派生集整体替换
	//（AssignApis/AssignMenus 自带替换语义）。
	// mergeHit 记录已消化码：插入段与删除段都以它判定，不能复用被清空的
	// existingByCode（清空后插入段会把全部派生码误判为"库没有"而必然撞码）。
	updated, added, removed := 0, 0, 0
	mergeHit := make(map[string]struct{}, len(derivedByCode))
	goneIDs := make([]uint32, 0)
	for code, d := range derivedByCode {
		exist, ok := existingByCode[code]
		if !ok {
			continue
		}
		if err = s.permissionRepo.UpdateForSync(ctx, exist.GetId(), d); err != nil {
			s.log.Errorf(ctx, "sync permissions update [%s] failed: %s", code, err.Error())
			return nil, err
		}
		if err = s.permissionRepo.AssignSyncBindings(ctx, exist.GetId(), d.GetApiIds(), d.GetMenuIds()); err != nil {
			return nil, err
		}
		updated++
		mergeHit[code] = struct{}{}
	}

	// 插入段：派生集有、库没有
	toInsert := make([]*permissionV1.Permission, 0, len(derivedByCode))
	for _, d := range derivedByCode {
		if _, hit := mergeHit[d.GetCode()]; hit {
			continue
		}
		if _, inDB := existingByCode[d.GetCode()]; inDB {
			continue // 防御：理论不可达（命中必进更新段），保留护栏
		}
		toInsert = append(toInsert, d)
		added++
	}
	if len(toInsert) > 0 {
		if err = s.permissionRepo.BatchCreate(ctx, toInsert); err != nil {
			s.log.Errorf(ctx, "batch create permissions failed: %s", err.Error())
			return nil, err
		}
	}

	// 删除段防误杀基线：全部菜单（**含停用**）可派生的权限码集合。派生集只取
	// 启用菜单，若管理员临时停用某菜单，同步会把其权限码误判为"已消失"而删行
	// 并级联清全部角色授权；重新启用+再同步后权限重建换 id，授权全丢——MERGE
	// 的保 id 教义被 status 过滤击穿。码仍可从任何现存菜单派生的权限行不删
	//（绑定保持现状，启用后再同步收敛）。
	allMenus, err := s.menuRepo.List(ctx, &paginationV1.PagingRequest{
		NoPaging: trans.Ptr(true),
		OrderBy:  trans.Ptr("id desc"),
	}, false)
	if err != nil {
		return nil, err
	}
	s.menuPermissionConverter.ComposeMenuPaths(allMenus.Items)
	aliveCodes := make(map[string]struct{}, len(allMenus.Items))
	for _, m := range allMenus.Items {
		if c := s.menuPermissionConverter.ConvertCode(m.GetPath(), m.GetName(), m.GetType()); c != "" {
			aliveCodes[c] = struct{}{}
		}
	}

	// 删除段：库有、派生集没有且码已无法从任何现存菜单派生 → 删行并级联清
	// 角色授权/端点/菜单绑定/策略行（角色授权此前漏清会留悬空授权，是本轮
	// 彻底化的最后一块）。放最后：中途失败残留的是"本该删的旧码"，重跑同步
	// 即自愈，好过删多了。
	for code, e := range existingByCode {
		if _, hit := mergeHit[code]; hit {
			continue
		}
		if _, alive := aliveCodes[code]; alive {
			continue
		}
		goneIDs = append(goneIDs, e.GetId())
		removed++
	}
	if len(goneIDs) > 0 {
		if err = s.permissionRepo.DeleteByIDs(ctx, goneIDs); err != nil {
			return nil, err
		}
	}
	s.log.Infof(ctx, "权限同步完成：新增 %d、更新 %d、移除 %d", added, updated, removed)

	// 重置权限策略
	if err = s.authorizer.ResetPolicies(ctx); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *PermissionService) createDefaultPermissions(ctx context.Context) error {
	var err error

	// —— 组就位（B4）：权限种子的 GroupName 要解析成组 id，先确保默认组已播种。
	// 旧时序里组播种晚于权限播种（wiring 顺序），GroupId=2/3/4 全靠"组恰好按序
	// 补位 1..5"的约定成立；现把组播种内聚到这里（组 service 的 count==0 播种
	// 与此幂等互斥，谁先跑都一样）。
	groupCount, err := s.permissionGroupRepo.Count(ctx, []func(*sql.Selector){})
	if err != nil {
		return err
	}
	if groupCount == 0 {
		if _, err = s.permissionGroupRepo.BatchCreate(ctx, constants.DefaultPermissionGroups); err != nil {
			s.log.Errorf(ctx, "seed default permission groups failed: %v", err)
			return err
		}
	}
	// 平铺列表（treeTravel=false）：树形遍历会把子组嵌进 Children，平铺才能按 name 建全量索引
groups, err := s.permissionGroupRepo.List(ctx, &paginationV1.PagingRequest{NoPaging: trans.Ptr(true)}, false)
	if err != nil {
		return err
	}
	groupIDByName := make(map[string]uint32, len(groups.GetItems()))
	for _, g := range groups.GetItems() {
		groupIDByName[g.GetName()] = g.GetId()
	}

	// —— 菜单引用收敛（B1）：菜单 id 在已有库中可能被菜单同步/重建改变，
	// 种子引用一律收敛到真实存在的行；超管的菜单集 = 当前全量（语义：
	// 平台管理员看得见一切，DefaultMenus 新增行自动跟上，不再手工维护清单）。
	menuIDs, err := s.menuRepo.ListAllIDs(ctx)
	if err != nil {
		return err
	}
	menuIDSet := make(map[uint32]struct{}, len(menuIDs))
	for _, id := range menuIDs {
		menuIDSet[id] = struct{}{}
	}

	// —— 端点引用语义化（A）：超管 = 全部端点；租户管理员 = 全部端点 −
	// 平台专属黑名单（path 前缀，见 constants.TenantManagerDeniedApiPrefixes）。
	// 旧硬编码 id 段（1..128）在接口表长到 280+ 条且 id 按 map 随机序发放的
	// 现实里早已是"随机子集抽签"，与本意（全集减平台端点）对不上。
	apis, err := s.apiRepo.ListAll(ctx)
	if err != nil {
		return err
	}
	allApiIDs := make([]uint32, 0, len(apis))
	tenantApiIDs := make([]uint32, 0, len(apis))
	for _, a := range apis {
		allApiIDs = append(allApiIDs, a.GetId())
		if !constants.IsTenantManagerDeniedApiPath(a.GetPath()) {
			tenantApiIDs = append(tenantApiIDs, a.GetId())
		}
	}

	for _, d := range constants.DefaultPermissions {
		data := proto.Clone(d).(*permissionV1.Permission)

		// GroupName → GroupId：解析不到软降级（组名留空、权限落"未分组"）+ WARN，
		// 不中止播种——引导期可能存在管理员自建分组体系，种子分组只是初值
		//（与"关联资源缺失不中止"的播种既有口径一致）。
		if data.GroupName != nil {
			if gid, ok := groupIDByName[data.GetGroupName()]; ok {
				data.GroupId = trans.Ptr(gid)
			} else {
				s.log.Warnf(ctx, "default permission [%s] references unknown group [%s]; seeding ungrouped",
					d.GetCode(), d.GetGroupName())
				data.GroupId = nil
			}
		}

		switch data.GetCode() {
		case constants.SystemPlatformAdminPermissionCode:
			data.MenuIds = append(data.MenuIds[:0:0], menuIDs...)
			data.ApiIds = append(data.ApiIds[:0:0], allApiIDs...)
		case constants.SystemTenantManagerPermissionCode:
			kept := make([]uint32, 0, len(data.GetMenuIds()))
			for _, id := range data.GetMenuIds() {
				if _, ok := menuIDSet[id]; ok {
					kept = append(kept, id)
				}
			}
			if dropped := len(data.GetMenuIds()) - len(kept); dropped > 0 {
				s.log.Warnf(ctx, "default permission [%s] drops %d stale menu id refs (seed binds existing menus only)",
					d.GetCode(), dropped)
			}
			data.MenuIds = kept
			data.ApiIds = tenantApiIDs
		}

		if err = s.permissionRepo.Create(ctx, &permissionV1.CreatePermissionRequest{
			Data: data,
		}); err != nil {
			s.log.Errorf(ctx, "create default permission %s failed: %v", d.GetCode(), err)
			return err
		}
	}

	return nil
}
