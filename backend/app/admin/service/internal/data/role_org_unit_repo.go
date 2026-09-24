package data

import (
	"context"
	"time"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	entCrud "github.com/tx7do/go-crud/entgo"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/orgunit"
	"go-wind-admin/app/admin/service/internal/data/ent/roleorgunit"

	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
)

// RoleOrgUnitRepo 角色与组织单元关联仓储。
// 仅承载 Role.data_scope = SELECTED_UNITS 档位下的自定义授权单元集。
type RoleOrgUnitRepo struct {
	log       *bLogger.Helper
	entClient *entCrud.EntClient[*ent.Client]
}

func NewRoleOrgUnitRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *RoleOrgUnitRepo {
	repo := &RoleOrgUnitRepo{
		log:       ctx.NewLoggerHelper("role-org-unit/repo/admin-service"),
		entClient: entClient,
	}

	return repo
}

// validateOrgUnitsInTenant 校验单元 ID 集合全部属于指定租户，拒绝跨租户注入。
// 显式 tenant 谓词在平台上下文（TenantPrivacy 跳过注入）下仍然生效；
// 租户上下文下与 TenantPrivacy 的注入叠加为双重防线。
func (r *RoleOrgUnitRepo) validateOrgUnitsInTenant(ctx context.Context, tenantID uint32, orgUnitIDs []uint32) error {
	distinct := make(map[uint32]struct{}, len(orgUnitIDs))
	for _, id := range orgUnitIDs {
		distinct[id] = struct{}{}
	}

	count, err := r.entClient.Client().OrgUnit.Query().
		Where(
			orgunit.IDIn(orgUnitIDs...),
			orgunit.TenantIDEQ(tenantID),
		).
		Count(ctx)
	if err != nil {
		r.log.Errorf(ctx, "validate org units failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("validate org units failed")
	}
	if count != len(distinct) {
		r.log.Errorf(ctx, "org unit selection crosses tenant boundary: tenant=%d want=%d got=%d",
			tenantID, len(distinct), count)
		return permissionV1.ErrorBadRequest("invalid org unit selection")
	}
	return nil
}

// CleanOrgUnits 清理角色的所有组织单元授权
func (r *RoleOrgUnitRepo) CleanOrgUnits(
	ctx context.Context,
	tx *ent.Tx,
	roleID uint32,
) error {
	if _, err := tx.RoleOrgUnit.Delete().
		Where(
			roleorgunit.RoleIDEQ(roleID),
		).
		Exec(ctx); err != nil {
		r.log.Errorf(context.Background(), "delete old role [%d] org units failed: %s", roleID, err.Error())
		return permissionV1.ErrorInternalServerError("delete old role org units failed")
	}
	return nil
}

// AssignOrgUnits 给角色分配组织单元授权（upsert，不删除既有）
func (r *RoleOrgUnitRepo) AssignOrgUnits(ctx context.Context, tx *ent.Tx,
	tenantID, operatorID uint32,
	roleID uint32, orgUnitIDs []uint32,
) error {
	if len(orgUnitIDs) == 0 {
		return nil
	}

	if err := r.validateOrgUnitsInTenant(ctx, tenantID, orgUnitIDs); err != nil {
		return err
	}

	now := time.Now()

	for _, orgUnitID := range orgUnitIDs {
		rou := tx.RoleOrgUnit.
			Create().
			SetTenantID(tenantID).
			SetOrgUnitID(orgUnitID).
			SetRoleID(roleID).
			SetCreatedBy(operatorID).
			SetCreatedAt(now).
			OnConflictColumns(
				roleorgunit.FieldTenantID,
				roleorgunit.FieldRoleID,
				roleorgunit.FieldOrgUnitID,
			).
			UpdateNewValues().
			SetUpdatedBy(operatorID).
			SetUpdatedAt(now)

		if err := rou.Exec(ctx); err != nil {
			r.log.Errorf(ctx, "assign org unit to role failed: %s", err.Error())
			return permissionV1.ErrorInternalServerError("assign org unit to role failed")
		}
	}

	return nil
}

// ReplaceOrgUnits 整体替换角色的组织单元授权：先清空既有授权，再插入新集合。
func (r *RoleOrgUnitRepo) ReplaceOrgUnits(ctx context.Context, tx *ent.Tx,
	tenantID, operatorID uint32,
	roleID uint32, orgUnitIDs []uint32,
) error {
	if err := r.CleanOrgUnits(ctx, tx, roleID); err != nil {
		return err
	}

	if len(orgUnitIDs) == 0 {
		return nil
	}

	return r.AssignOrgUnits(ctx, tx, tenantID, operatorID, roleID, orgUnitIDs)
}

// ListOrgUnitIDs 列出角色的组织单元ID列表
func (r *RoleOrgUnitRepo) ListOrgUnitIDs(ctx context.Context, roleID uint32) ([]uint32, error) {
	intIDs, err := r.entClient.Client().RoleOrgUnit.Query().
		Where(
			roleorgunit.RoleIDEQ(roleID),
		).
		Select(roleorgunit.FieldOrgUnitID).
		Ints(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query org unit ids by role id failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("query org unit ids by role id failed")
	}
	ids := make([]uint32, len(intIDs))
	for i, v := range intIDs {
		ids[i] = uint32(v)
	}
	return ids, nil
}

// ListOrgUnitIDsByRoleIDs 根据角色ID列表获取组织单元ID列表（按角色分组返回）
func (r *RoleOrgUnitRepo) ListOrgUnitIDsByRoleIDs(ctx context.Context, roleIDs []uint32) (map[uint32][]uint32, error) {
	if len(roleIDs) == 0 {
		return map[uint32][]uint32{}, nil
	}

	entities, err := r.entClient.Client().RoleOrgUnit.Query().
		Where(
			roleorgunit.RoleIDIn(roleIDs...),
		).
		Select(roleorgunit.FieldRoleID, roleorgunit.FieldOrgUnitID).
		All(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query org unit ids by role ids failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("query org unit ids by role ids failed")
	}

	result := make(map[uint32][]uint32, len(roleIDs))
	for _, entity := range entities {
		if entity.RoleID == nil || entity.OrgUnitID == nil {
			continue
		}
		result[*entity.RoleID] = append(result[*entity.RoleID], *entity.OrgUnitID)
	}
	return result, nil
}
