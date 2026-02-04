package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/timeutil"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/membershiprole"

	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
)

type MembershipRoleRepo struct {
	log             *log.Helper
	entClient       *entCrud.EntClient[*ent.Client]
	statusConverter *mapper.EnumTypeConverter[identityV1.MembershipRole_Status, membershiprole.Status]
}

func NewMembershipRoleRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *MembershipRoleRepo {
	return &MembershipRoleRepo{
		log:       ctx.NewLoggerHelper("membership-role/repo/admin-service"),
		entClient: entClient,
		statusConverter: mapper.NewEnumTypeConverter[identityV1.MembershipRole_Status, membershiprole.Status](
			identityV1.MembershipRole_Status_name,
			identityV1.MembershipRole_Status_value,
		),
	}
}

// CleanRelationsByMembershipID 删除会员的所有角色关联
func (r *MembershipRoleRepo) CleanRelationsByMembershipID(ctx context.Context, tx *ent.Tx, membershipID uint32) error {
	if membershipID == 0 {
		return nil
	}

	if _, err := tx.MembershipRole.Delete().
		Where(
			membershiprole.MembershipIDEQ(membershipID),
		).
		Exec(ctx); err != nil {
		r.log.Errorf("delete old membership roles failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("delete old membership roles failed")
	}
	return nil
}

// CleanRelationsByMembershipIDs 删除多个会员的所有角色关联
func (r *MembershipRoleRepo) CleanRelationsByMembershipIDs(ctx context.Context, tx *ent.Tx, membershipIDs []uint32) error {
	if len(membershipIDs) == 0 {
		return nil
	}

	if _, err := tx.MembershipRole.Delete().
		Where(
			membershiprole.MembershipIDIn(membershipIDs...),
		).
		Exec(ctx); err != nil {
		r.log.Errorf("delete old membership roles by membership ids failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("delete old membership roles by membership ids failed")
	}
	return nil
}

// CleanRelationsByRoleID 删除角色的所有会员关联
func (r *MembershipRoleRepo) CleanRelationsByRoleID(ctx context.Context, tx *ent.Tx, roleID uint32) error {
	if roleID == 0 {
		return nil
	}

	if _, err := tx.MembershipRole.Delete().
		Where(
			membershiprole.RoleIDEQ(roleID),
		).
		Exec(ctx); err != nil {
		r.log.Errorf("delete old membership roles by role id failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("delete old membership roles by role id failed")
	}
	return nil
}

// CleanRelationsByRoleIDs 删除多个角色的所有会员关联
func (r *MembershipRoleRepo) CleanRelationsByRoleIDs(ctx context.Context, tx *ent.Tx, roleIDs []uint32) error {
	if len(roleIDs) == 0 {
		return nil
	}

	if _, err := tx.MembershipRole.Delete().
		Where(
			membershiprole.RoleIDIn(roleIDs...),
		).
		Exec(ctx); err != nil {
		r.log.Errorf("delete old membership roles by role ids failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("delete old membership roles by role ids failed")
	}
	return nil
}

// RemoveRolesFromMembership 移除角色
func (r *MembershipRoleRepo) RemoveRolesFromMembership(ctx context.Context, membershipID uint32, roleIDs []uint32) error {
	if membershipID == 0 || len(roleIDs) == 0 {
		return nil
	}

	_, err := r.entClient.Client().MembershipRole.Delete().
		Where(
			membershiprole.And(
				membershiprole.MembershipIDEQ(membershipID),
				membershiprole.RoleIDIn(roleIDs...),
			),
		).
		Exec(ctx)
	if err != nil {
		r.log.Errorf("remove roles from membership failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("remove roles from membership failed")
	}
	return nil
}

// AssignMembershipRoles 分配角色
func (r *MembershipRoleRepo) AssignMembershipRoles(ctx context.Context,
	tx *ent.Tx,
	membershipID uint32,
	datas []*identityV1.MembershipRole,
) error {
	if membershipID == 0 || len(datas) == 0 {
		return nil
	}

	var err error

	// 删除该用户的所有旧关联
	if err = r.CleanRelationsByMembershipID(ctx, tx, membershipID); err != nil {
		return identityV1.ErrorInternalServerError("clean old membership roles failed")
	}

	now := time.Now()

	var membershipRoleCreates []*ent.MembershipRoleCreate
	for _, data := range datas {
		if data.StartAt == nil {
			data.StartAt = timeutil.TimeToTimestamppb(&now)
		}

		rm := tx.MembershipRole.
			Create().
			SetNillableTenantID(data.TenantId).
			SetMembershipID(data.GetMembershipId()).
			SetRoleID(data.GetRoleId()).
			SetNillableStatus(r.statusConverter.ToEntity(data.Status)).
			SetNillableAssignedBy(data.AssignedBy).
			SetNillableAssignedAt(timeutil.TimestamppbToTime(data.AssignedAt)).
			SetNillableIsPrimary(data.IsPrimary).
			SetNillableStartAt(timeutil.TimestamppbToTime(data.StartAt)).
			SetNillableEndAt(timeutil.TimestamppbToTime(data.EndAt)).
			SetCreatedAt(now).
			SetNillableCreatedBy(data.CreatedBy)
		membershipRoleCreates = append(membershipRoleCreates, rm)
	}

	_, err = tx.MembershipRole.CreateBulk(membershipRoleCreates...).Save(ctx)
	if err != nil {
		r.log.Errorf("assign roles to membership failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("assign roles to membership failed")
	}

	return nil
}

// ListRoleIDs 获取用户关联的角色ID列表
func (r *MembershipRoleRepo) ListRoleIDs(ctx context.Context, membershipID uint32, excludeExpired bool) ([]uint32, error) {
	if membershipID == 0 {
		return []uint32{}, nil
	}

	q := r.entClient.Client().MembershipRole.Query().
		Where(
			membershiprole.MembershipIDEQ(membershipID),
		)

	if excludeExpired {
		now := time.Now()
		q = q.Where(
			membershiprole.Or(
				membershiprole.EndAtIsNil(),
				membershiprole.EndAtGT(now),
			),
		)
	}

	intIDs, err := q.
		Select(membershiprole.FieldRoleID).
		Ints(ctx)
	if err != nil {
		r.log.Errorf("query role ids by membership id failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("query role ids by membership id failed")
	}
	ids := make([]uint32, len(intIDs))
	for i, v := range intIDs {
		ids[i] = uint32(v)
	}
	return ids, nil
}

// ListMembershipIDs 获取角色关联的会员ID列表
func (r *MembershipRoleRepo) ListMembershipIDs(ctx context.Context, roleID uint32, excludeExpired bool) ([]uint32, error) {
	if roleID == 0 {
		return []uint32{}, nil
	}

	q := r.entClient.Client().MembershipRole.Query().
		Where(
			membershiprole.RoleIDEQ(roleID),
		)

	if excludeExpired {
		now := time.Now()
		q = q.Where(
			membershiprole.Or(
				membershiprole.EndAtIsNil(),
				membershiprole.EndAtGT(now),
			),
		)
	}

	intIDs, err := q.
		Select(membershiprole.FieldMembershipID).
		Ints(ctx)
	if err != nil {
		r.log.Errorf("query membership ids by role id failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("query membership ids by role id failed")
	}
	ids := make([]uint32, len(intIDs))
	for i, v := range intIDs {
		ids[i] = uint32(v)
	}
	return ids, nil
}

// ListMembershipIDsByRoleIDs 获取多个角色关联的会员ID列表
func (r *MembershipRoleRepo) ListMembershipIDsByRoleIDs(ctx context.Context, roleIDs []uint32, excludeExpired bool) ([]uint32, error) {
	if len(roleIDs) == 0 {
		return nil, nil
	}

	q := r.entClient.Client().MembershipRole.Query().
		Where(
			membershiprole.RoleIDIn(roleIDs...),
		)

	if excludeExpired {
		now := time.Now()
		q = q.Where(
			membershiprole.Or(
				membershiprole.EndAtIsNil(),
				membershiprole.EndAtGT(now),
			),
		)
	}

	intIDs, err := q.
		Select(membershiprole.FieldMembershipID).
		Ints(ctx)
	if err != nil {
		r.log.Errorf("query membership ids by role ids failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("query membership ids by role ids failed")
	}
	ids := make([]uint32, len(intIDs))
	for i, v := range intIDs {
		ids[i] = uint32(v)
	}
	return ids, nil
}
