package service

import (
	"context"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/aggregator"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"go-wind-admin/app/admin/service/internal/data"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"

	"github.com/tx7do/go-crud/viewer"
	"github.com/tx7do/go-utils/authorizer"
	"github.com/tx7do/go-utils/sliceutil"
	"go-wind-admin/pkg/constants"
	"go-wind-admin/pkg/middleware/auth"
)

type RoleService struct {
	adminV1.RoleServiceHTTPServer

	log *bLogger.Helper

	authorizer *authorizer.Authorizer

	roleRepo       *data.RoleRepo
	tenantRepo     *data.TenantRepo
	permissionRepo *data.PermissionRepo
}

func NewRoleService(
	ctx *bootstrap.Context,
	authorizer *authorizer.Authorizer,
	roleRepo *data.RoleRepo,
	tenantRepo *data.TenantRepo,
	permissionRepo *data.PermissionRepo,
) *RoleService {
	svc := &RoleService{
		log:            ctx.NewLoggerHelper("role/service/admin-service"),
		authorizer:     authorizer,
		roleRepo:       roleRepo,
		tenantRepo:     tenantRepo,
		permissionRepo: permissionRepo,
	}

	svc.init()

	return svc
}

func (s *RoleService) init() {
	ctx := viewer.WithSystemContext(context.Background())
	if count, _ := s.roleRepo.Count(ctx, nil); count == 0 {
		_ = s.createDefaultRoles(ctx)
	}
}

func (s *RoleService) extractRelationIDs(
	roles []*permissionV1.Role,
	tenantSet aggregator.ResourceMap[int64, *identityV1.Tenant],
	roleSet aggregator.ResourceMap[int64, *permissionV1.Role],
) {
	for _, p := range roles {
		if p.GetTenantId() > 0 {
			tenantSet[p.GetTenantId()] = nil
		}
		if p.GetParentId() > 0 {
			roleSet[p.GetParentId()] = nil
		}
	}
}

func (s *RoleService) fetchRelationInfo(
	ctx context.Context,
	tenantSet aggregator.ResourceMap[int64, *identityV1.Tenant],
	roleSet aggregator.ResourceMap[int64, *permissionV1.Role],
) error {
	if len(tenantSet) > 0 {
		tenantIds := make([]int64, 0, len(tenantSet))
		for id := range tenantSet {
			tenantIds = append(tenantIds, id)
		}

		tenants, err := s.tenantRepo.ListTenantsByIds(ctx, tenantIds)
		if err != nil {
			s.log.Errorf(context.Background(), "query tenants err: %v", err)
			return err
		}

		for _, tenant := range tenants {
			tenantSet[tenant.GetId()] = tenant
		}
	}

	if len(roleSet) > 0 {
		roleIds := make([]int64, 0, len(roleSet))
		for id := range roleSet {
			roleIds = append(roleIds, id)
		}

		parents, err := s.roleRepo.ListRolesByRoleIds(ctx, roleIds)
		if err != nil {
			s.log.Errorf(ctx, "query parent roles err: %v", err)
			return err
		}

		for _, parent := range parents {
			roleSet[parent.GetId()] = parent
		}
	}

	return nil
}

func (s *RoleService) bindRelations(
	roles []*permissionV1.Role,
	tenantSet aggregator.ResourceMap[int64, *identityV1.Tenant],
	roleSet aggregator.ResourceMap[int64, *permissionV1.Role],
) {
	aggregator.Populate(
		roles,
		tenantSet,
		func(ou *permissionV1.Role) int64 { return ou.GetTenantId() },
		func(ou *permissionV1.Role, r *identityV1.Tenant) {
			ou.TenantName = r.Name
		},
	)
	aggregator.Populate(
		roles,
		roleSet,
		func(ou *permissionV1.Role) int64 { return ou.GetParentId() },
		func(ou *permissionV1.Role, r *permissionV1.Role) {
			ou.ParentName = trans.Ptr(r.GetName())
		},
	)
}

func (s *RoleService) enrichRelations(ctx context.Context, roles []*permissionV1.Role) error {
	var tenantSet = make(aggregator.ResourceMap[int64, *identityV1.Tenant])
	var roleSet = make(aggregator.ResourceMap[int64, *permissionV1.Role])
	s.extractRelationIDs(roles, tenantSet, roleSet)
	if err := s.fetchRelationInfo(ctx, tenantSet, roleSet); err != nil {
		return err
	}
	s.bindRelations(roles, tenantSet, roleSet)
	return nil
}

// hasMaskPath 探测 updateMask 是否声明了给定字段（proto 字段名与 json_name 双探测）。
// mask 为 nil 视为全字段写入（FilterByFieldMask 语义），返回 true。
func hasMaskPath(mask *fieldmaskpb.FieldMask, names ...string) bool {
	if mask == nil {
		return true
	}
	for _, p := range mask.GetPaths() {
		for _, n := range names {
			if p == n {
				return true
			}
		}
	}
	return false
}

// validateParentBinding 校验父子角色绑定的合法性（权限继承核心约束，编辑时校验语义）。
// parentID 为 0 表示根角色，直接放行。校验：
//  1. 父角色存在；
//  2. 父角色与本角色同租户（tenantID 为本角色归属租户，Create 侧以操作者上下文折算）；
//  3. wantPerms ⊆ 父角色已绑定权限集。
//
// roleID 为 0 表示 Create 场景（尚无自身 ID）。
func (s *RoleService) validateParentBinding(ctx context.Context, roleID, parentID, tenantID int64, wantPerms []int64) error {
	if parentID == 0 {
		return nil
	}

	parent, err := s.roleRepo.Get(ctx, &permissionV1.GetRoleRequest{
		QueryBy: &permissionV1.GetRoleRequest_Id{Id: parentID},
	})
	if err != nil {
		s.log.Errorf(ctx, "get parent role failed: %v", err)
		return adminV1.ErrorBadRequest("parent role not found")
	}

	if parent.GetTenantId() != tenantID {
		return adminV1.ErrorForbidden("parent role must belong to the same tenant")
	}

	if len(wantPerms) == 0 {
		return nil
	}

	capSet := make(map[int64]struct{}, len(parent.GetPermissions()))
	for _, id := range parent.GetPermissions() {
		capSet[id] = struct{}{}
	}

	var exceeding int
	for _, id := range wantPerms {
		if _, ok := capSet[id]; !ok {
			exceeding++
		}
	}
	if exceeding > 0 {
		return adminV1.ErrorForbidden(
			"%d permission(s) exceed the parent role's granted permission set", exceeding)
	}
	return nil
}

// ensureNoRoleCycle 沿候选父角色上溯检测层级环（Update 改父时调用）：
// 若从候选父出发能回到自身，说明会成环，拒绝。上溯深度设上限防脏数据死循环。
func (s *RoleService) ensureNoRoleCycle(ctx context.Context, roleID, parentID int64) error {
	const maxDepth = 32

	current := parentID
	for depth := 0; current != 0 && depth < maxDepth; depth++ {
		if current == roleID {
			return adminV1.ErrorForbidden("circular role hierarchy detected")
		}
		parent, err := s.roleRepo.Get(ctx, &permissionV1.GetRoleRequest{
			QueryBy: &permissionV1.GetRoleRequest_Id{Id: current},
		})
		if err != nil {
			// 上溯链断裂（父角色不存在）不在此处报错，交给 validateParentBinding
			return nil
		}
		current = parent.GetParentId()
	}
	return nil
}

func (s *RoleService) List(ctx context.Context, req *paginationV1.PagingRequest) (*permissionV1.ListRoleResponse, error) {
	resp, err := s.roleRepo.List(ctx, req)
	if err != nil {
		return nil, err
	}

	_ = s.enrichRelations(ctx, resp.Items)

	return resp, nil
}

func (s *RoleService) Count(ctx context.Context, req *paginationV1.PagingRequest) (*permissionV1.CountRoleResponse, error) {
	count, err := s.roleRepo.Count(ctx, req)
	if err != nil {
		return nil, err
	}

	return &permissionV1.CountRoleResponse{
		Count: int64(count),
	}, nil
}

func (s *RoleService) Get(ctx context.Context, req *permissionV1.GetRoleRequest) (*permissionV1.Role, error) {
	resp, err := s.roleRepo.Get(ctx, req)
	if err != nil {
		return nil, err
	}

	fakeItems := []*permissionV1.Role{resp}
	_ = s.enrichRelations(ctx, fakeItems)

	return resp, nil
}

func (s *RoleService) Create(ctx context.Context, req *permissionV1.CreateRoleRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	// 服务内管理闸门（纵深防御，authz 出厂 noop）：角色是权限的载体，此前无任何
	// 管理权限判断，租户普通用户可创建携带全量 permission id 的根角色再自授
	//（配合 /me 或 UserService.Update 即成租户管理员）。
	if err = requireTenantOrPlatformAdmin(ctx, s.log, "role create"); err != nil {
		return nil, err
	}

	req.Data.CreatedBy = trans.Ptr(operator.UserId)

	if operator.GetTenantId() > 0 && req.Data.GetType() != permissionV1.Role_TENANT {
		req.Data.Type = trans.Ptr(permissionV1.Role_TENANT)
	}

	// 角色码保留字校验：内置身份码（platform:* / template:* / tenant:manager）在
	// 登录链路与鉴权引擎里按码值承载身份判定语义，用户自建角色不得占用，
	// 否则自建 platform:admin 再登录即可拿到平台管理员标志（越权）。
	if constants.IsReservedRoleCode(req.Data.GetCode()) {
		return nil, adminV1.ErrorBadRequest("role code is reserved")
	}

	// 父角色绑定校验（权限继承）。租户操作者的归属租户以操作者上下文为准
	// （与上方类型强转同口径），载荷里的 tenant_id 不可信。
	roleTenantID := req.Data.GetTenantId()
	if operator.GetTenantId() > 0 {
		roleTenantID = operator.GetTenantId()
	}
	if err = s.validateParentBinding(ctx, 0, req.Data.GetParentId(), roleTenantID, req.Data.GetPermissions()); err != nil {
		return nil, err
	}

	if err = s.roleRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	if err = s.authorizer.ResetPolicies(ctx); err != nil {
		s.log.Errorf(ctx, "reset policies error: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (s *RoleService) Update(ctx context.Context, req *permissionV1.UpdateRoleRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		s.log.Errorf(ctx, "get operator from context error: %v", err)
		return nil, err
	}

	// 服务内管理闸门（纵深防御，同 Create）：改角色权限即改权限边界。
	if err = requireTenantOrPlatformAdmin(ctx, s.log, "role update"); err != nil {
		return nil, err
	}

	req.Data.Id = trans.Ptr(req.GetId())

	req.Data.UpdatedBy = trans.Ptr(operator.UserId)
	if req.UpdateMask != nil {
		req.UpdateMask.Paths = append(req.UpdateMask.Paths, "updated_by")
	}

	if operator.GetTenantId() > 0 && req.Data.GetType() != permissionV1.Role_TENANT {
		req.Data.Type = trans.Ptr(permissionV1.Role_TENANT)
	}

	// 角色码保留字校验：与 Create 同口径——受保护角色的 code 会被下方黑名单剥离，
	// 但普通角色的 code 可改，必须拦住改成内置身份码的路径。
	if req.Data.Code != nil && constants.IsReservedRoleCode(req.Data.GetCode()) {
		return nil, adminV1.ErrorBadRequest("role code is reserved")
	}

	r, err := s.roleRepo.Get(ctx, &permissionV1.GetRoleRequest{
		QueryBy: &permissionV1.GetRoleRequest_Id{
			Id: req.GetId(),
		},
	})
	if err != nil {
		return nil, err
	}

	// 非系统管理员禁止修改系统角色
	if r.GetType() == permissionV1.Role_SYSTEM && operator.GetTenantId() > 0 {
		return nil, adminV1.ErrorForbidden("no permission to update system role")
	}

	// 父角色绑定校验（权限继承，编辑时校验语义）：本次请求生效的父角色与权限集
	// 分别按"mask 是否声明"取舍——声明了取载荷，未声明取库中现值（改权限不动父、
	// 或换父不动权限，都要重新对齐约束）。
	parentIDChanged := hasMaskPath(req.UpdateMask, "parent_id", "parentId")
	permsChanged := hasMaskPath(req.UpdateMask, "permissions")
	var effectiveParentID int64
	var effectivePerms []int64
	if parentIDChanged {
		effectiveParentID = req.Data.GetParentId()
	} else {
		effectiveParentID = r.GetParentId()
	}
	if permsChanged {
		effectivePerms = req.Data.GetPermissions()
	} else {
		effectivePerms = r.GetPermissions()
	}
	if effectiveParentID > 0 {
		if parentIDChanged {
			if err = s.ensureNoRoleCycle(ctx, req.GetId(), effectiveParentID); err != nil {
				return nil, err
			}
		}
		if err = s.validateParentBinding(ctx, req.GetId(), effectiveParentID, r.GetTenantId(), effectivePerms); err != nil {
			return nil, err
		}
	}

	// 保护角色字段不可修改：permissions 与 parent_id 一并纳入黑名单——
	// 此前租户普通用户（甚至管理员）可给自己的 tenant:manager 副本追加全量权限
	// 或改其继承链，等于把"受保护"架空（P1 审计）。
	if r.GetIsProtected() {
		if len(req.GetUpdateMask().Paths) > 0 {
			req.GetUpdateMask().Paths = sliceutil.FilterBlacklist(req.GetUpdateMask().Paths, []string{
				"is_protected",
				"type",
				"status",
				"code",
				"permissions",
				"parent_id",
				"parentId",
			})
		} else {
			req.Data.IsProtected = nil
			req.Data.Type = nil
			req.Data.Status = nil
			req.Data.Code = nil
			req.Data.Permissions = nil
			req.Data.ParentId = nil
		}
	}

	if err = s.roleRepo.Update(ctx, req); err != nil {
		s.log.Errorf(ctx, "update role error: %v", err)
		return nil, err
	}

	if err = s.authorizer.ResetPolicies(ctx); err != nil {
		s.log.Errorf(ctx, "reset policies error: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (s *RoleService) Delete(ctx context.Context, req *permissionV1.DeleteRoleRequest) (*emptypb.Empty, error) {
	// 服务内管理闸门（纵深防御，同 Create）。
	if err := requireTenantOrPlatformAdmin(ctx, s.log, "role delete"); err != nil {
		return nil, err
	}

	var err error

	if err = s.roleRepo.Delete(ctx, req); err != nil {
		return nil, err
	}

	if err = s.authorizer.ResetPolicies(ctx); err != nil {
		s.log.Errorf(ctx, "reset policies error: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (s *RoleService) GetRoleCodesByRoleIds(ctx context.Context, req *permissionV1.GetRoleCodesByRoleIdsRequest) (*permissionV1.GetRoleCodesByRoleIdsResponse, error) {
	ids, err := s.roleRepo.ListRoleCodesByRoleIds(ctx, req.GetRoleIds())
	if err != nil {
		return nil, err
	}

	return &permissionV1.GetRoleCodesByRoleIdsResponse{
		RoleCodes: ids,
	}, nil
}

func (s *RoleService) GetRolesByRoleCodes(ctx context.Context, req *permissionV1.GetRolesByRoleCodesRequest) (*permissionV1.ListRoleResponse, error) {
	roles, err := s.roleRepo.ListRolesByRoleCodes(ctx, req.GetRoleCodes())
	if err != nil {
		return nil, err
	}

	return &permissionV1.ListRoleResponse{
		Items: roles,
		Total: int64(len(roles)),
	}, nil
}

func (s *RoleService) GetRolesByRoleIds(ctx context.Context, req *permissionV1.GetRolesByRoleIdsRequest) (*permissionV1.ListRoleResponse, error) {
	roles, err := s.roleRepo.ListRolesByRoleIds(ctx, req.GetRoleIds())
	if err != nil {
		return nil, err
	}

	return &permissionV1.ListRoleResponse{
		Items: roles,
		Total: int64(len(roles)),
	}, nil
}

// createDefaultRoles 创建默认角色(包括超级管理员)。
// 权限引用按 code 解析（constants.DefaultRolePermissionCodes）：权限 id 是播种
// 顺序产物，旧种子按 id（1,2,4 / 1,3）引用只在"权限恰好按切片序落 1..5"时碰巧
// 成立。code 是稳定键；生产 wiring 保证权限先于角色播种。解析出多少绑多少、
// 缺失 WARN（软降级，引导期 best-effort——测试/嵌入式场景无此时序保证时不中止
// 播种，与组名解析的口径一致）。
func (s *RoleService) createDefaultRoles(ctx context.Context) error {
	var err error

	for _, d := range constants.DefaultRoles {
		data := proto.Clone(d).(*permissionV1.Role)

		codes := constants.DefaultRolePermissionCodes[data.GetCode()]
		if len(codes) > 0 {
			ids, rerr := s.permissionRepo.GetPermissionIDsByCodes(ctx, codes)
			if rerr != nil {
				return rerr
			}
			if len(ids) < len(codes) {
				s.log.Warnf(ctx, "default role [%s] resolved %d/%d permission codes (permissions must seed before roles); binding resolved subset",
					data.GetCode(), len(ids), len(codes))
			}
			data.Permissions = ids
		}

		err = s.roleRepo.Create(ctx, &permissionV1.CreateRoleRequest{
			Data: data,
		})
		if err != nil {
			return err
		}
	}

	return nil
}
