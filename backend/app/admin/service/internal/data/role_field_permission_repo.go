package data

import (
	"context"
	"time"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	entCrud "github.com/tx7do/go-crud/entgo"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/rolefieldpermission"

	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
)

// maxFieldPermissionsPerRole 单角色字段权限条目上限：防止异常配置撑爆令牌与配置 UI。
const maxFieldPermissionsPerRole = 256

// RoleFieldPermissionRepo 角色字段权限配置仓储。
// 黑名单语义：记录角色在各资源上不可见的字段集；未配置即全字段可见。
type RoleFieldPermissionRepo struct {
	log       *bLogger.Helper
	entClient *entCrud.EntClient[*ent.Client]
}

func NewRoleFieldPermissionRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *RoleFieldPermissionRepo {
	repo := &RoleFieldPermissionRepo{
		log:       ctx.NewLoggerHelper("role-field-permission/repo/admin-service"),
		entClient: entClient,
	}

	return repo
}

// validateEntries 校验字段权限条目：资源/字段名非空、字段去重、总量不超上限。
// 资源名不在这里做注册表校验——裁剪器对未知资源天然不生效，垃圾条目无安全影响。
func (r *RoleFieldPermissionRepo) validateEntries(entries []*permissionV1.RoleFieldPermission) error {
	total := 0
	for _, entry := range entries {
		if entry.GetResource() == "" {
			return permissionV1.ErrorBadRequest("field permission resource is required")
		}
		seen := make(map[string]struct{}, len(entry.GetHiddenFields()))
		for _, f := range entry.GetHiddenFields() {
			if f == "" {
				return permissionV1.ErrorBadRequest("field permission field name is required")
			}
			seen[f] = struct{}{}
		}
		total += len(seen)
	}
	if total > maxFieldPermissionsPerRole {
		r.log.Errorf(context.Background(), "field permission entries [%d] exceed cap [%d]", total, maxFieldPermissionsPerRole)
		return permissionV1.ErrorBadRequest("too many field permission entries")
	}
	return nil
}

// CleanFieldPermissions 清理角色的所有字段权限配置
func (r *RoleFieldPermissionRepo) CleanFieldPermissions(
	ctx context.Context,
	tx *ent.Tx,
	roleID uint32,
) error {
	if _, err := tx.RoleFieldPermission.Delete().
		Where(
			rolefieldpermission.RoleIDEQ(roleID),
		).
		Exec(ctx); err != nil {
		r.log.Errorf(context.Background(), "delete role [%d] field permissions failed: %s", roleID, err.Error())
		return permissionV1.ErrorInternalServerError("delete role field permissions failed")
	}
	return nil
}

// ReplaceFieldPermissions 整体替换角色的字段权限配置：先清空既有配置，再插入新集合。
func (r *RoleFieldPermissionRepo) ReplaceFieldPermissions(ctx context.Context, tx *ent.Tx,
	tenantID, operatorID uint32,
	roleID uint32,
	entries []*permissionV1.RoleFieldPermission,
) error {
	if err := r.validateEntries(entries); err != nil {
		return err
	}

	if err := r.CleanFieldPermissions(ctx, tx, roleID); err != nil {
		return err
	}

	if len(entries) == 0 {
		return nil
	}

	now := time.Now()

	for _, entry := range entries {
		// 先按 (role, resource) 去重展开为行，行内字段已由 validateEntries 去重。
		seen := make(map[string]struct{}, len(entry.GetHiddenFields()))
		for _, f := range entry.GetHiddenFields() {
			if _, ok := seen[f]; ok {
				continue
			}
			seen[f] = struct{}{}

			builder := tx.RoleFieldPermission.Create().
				SetTenantID(tenantID).
				SetRoleID(roleID).
				SetResource(entry.GetResource()).
				SetFieldName(f).
				SetCreatedBy(operatorID).
				SetCreatedAt(now)
			if err := builder.
				OnConflictColumns(
					rolefieldpermission.FieldTenantID,
					rolefieldpermission.FieldRoleID,
					rolefieldpermission.FieldResource,
					rolefieldpermission.FieldFieldName,
				).
				UpdateNewValues().
				SetUpdatedBy(operatorID).
				SetUpdatedAt(now).
				Exec(ctx); err != nil {
				r.log.Errorf(ctx, "assign field permission to role failed: %s", err.Error())
				return permissionV1.ErrorInternalServerError("assign field permission to role failed")
			}
		}
	}

	return nil
}

// ListHiddenFieldGroupsByRoleIDs 按角色ID列表取字段权限配置（按角色分组）。
// 登录聚合用：一次查询取全部角色的配置，避免 N+1。
func (r *RoleFieldPermissionRepo) ListHiddenFieldGroupsByRoleIDs(ctx context.Context, roleIDs []uint32) (map[uint32][]*permissionV1.RoleFieldPermission, error) {
	if len(roleIDs) == 0 {
		return map[uint32][]*permissionV1.RoleFieldPermission{}, nil
	}

	entities, err := r.entClient.Client().RoleFieldPermission.Query().
		Where(
			rolefieldpermission.RoleIDIn(roleIDs...),
		).
		Select(
			rolefieldpermission.FieldRoleID,
			rolefieldpermission.FieldResource,
			rolefieldpermission.FieldFieldName,
		).
		All(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query field permissions by role ids failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("query field permissions by role ids failed")
	}

	type groupKey struct {
		roleID   uint32
		resource string
	}

	index := make(map[groupKey]int)
	result := make(map[uint32][]*permissionV1.RoleFieldPermission, len(roleIDs))
	for _, e := range entities {
		if e.RoleID == nil || e.Resource == nil || e.FieldName == nil {
			continue
		}
		key := groupKey{roleID: *e.RoleID, resource: *e.Resource}
		idx, ok := index[key]
		if !ok {
			result[key.roleID] = append(result[key.roleID], &permissionV1.RoleFieldPermission{
				Resource:     *e.Resource,
				HiddenFields: []string{*e.FieldName},
			})
			index[key] = len(result[key.roleID]) - 1
			continue
		}
		result[key.roleID][idx].HiddenFields = append(result[key.roleID][idx].HiddenFields, *e.FieldName)
	}

	return result, nil
}

// ListFieldPermissions 列出角色的字段权限配置（角色详情/列表回显用）
func (r *RoleFieldPermissionRepo) ListFieldPermissions(ctx context.Context, roleID uint32) ([]*permissionV1.RoleFieldPermission, error) {
	groups, err := r.ListHiddenFieldGroupsByRoleIDs(ctx, []uint32{roleID})
	if err != nil {
		return nil, err
	}
	return groups[roleID], nil
}
