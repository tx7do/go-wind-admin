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

	userV1 "go-wind-admin/api/gen/go/user/service/v1"
)

type MembershipRoleRepo struct {
	log             *log.Helper
	entClient       *entCrud.EntClient[*ent.Client]
	statusConverter *mapper.EnumTypeConverter[userV1.MembershipRole_Status, membershiprole.Status]
}

func NewMembershipRoleRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *MembershipRoleRepo {
	return &MembershipRoleRepo{
		log:       ctx.NewLoggerHelper("membership-role/repo/admin-service"),
		entClient: entClient,
		statusConverter: mapper.NewEnumTypeConverter[userV1.MembershipRole_Status, membershiprole.Status](
			userV1.MembershipRole_Status_name,
			userV1.MembershipRole_Status_value,
		),
	}
}

func (r *MembershipRoleRepo) CleanRoles(
	ctx context.Context,
	tx *ent.Tx,
	membershipID, tenantID uint32,
) error {
	if _, err := tx.MembershipRole.Delete().
		Where(
			membershiprole.MembershipIDEQ(membershipID),
			membershiprole.TenantIDEQ(tenantID),
		).
		Exec(ctx); err != nil {
		r.log.Errorf("delete old membership roles failed: %s", err.Error())
		return userV1.ErrorInternalServerError("delete old membership roles failed")
	}
	return nil
}

// AssignRoles 分配角色
func (r *MembershipRoleRepo) AssignRoles(ctx context.Context,
	tx *ent.Tx,
	membershipID, tenantID uint32,
	datas []*userV1.MembershipRole,
) error {
	var err error

	// 删除该用户的所有旧关联
	if err = r.CleanRoles(ctx, tx, membershipID, tenantID); err != nil {
		return userV1.ErrorInternalServerError("clean old membership roles failed")
	}

	if len(datas) == 0 {
		return nil
	}

	now := time.Now()

	var membershipRoleCreates []*ent.MembershipRoleCreate
	for _, data := range datas {
		if data.StartAt == nil {
			data.StartAt = timeutil.TimeToTimestamppb(&now)
		}

		rm := tx.MembershipRole.
			Create().
			SetTenantID(tenantID).
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
		return userV1.ErrorInternalServerError("assign roles to membership failed")
	}

	return nil
}

// ListRoleIDs 获取用户关联的角色ID列表
func (r *MembershipRoleRepo) ListRoleIDs(ctx context.Context, membershipID uint32, excludeExpired bool) ([]uint32, error) {
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
		return nil, userV1.ErrorInternalServerError("query role ids by membership id failed")
	}
	ids := make([]uint32, len(intIDs))
	for i, v := range intIDs {
		ids[i] = uint32(v)
	}
	return ids, nil
}

// RemoveRoles 移除角色
func (r *MembershipRoleRepo) RemoveRoles(ctx context.Context, membershipID, tenantID uint32, roleIDs []uint32) error {
	_, err := r.entClient.Client().MembershipRole.Delete().
		Where(
			membershiprole.And(
				membershiprole.MembershipIDEQ(membershipID),
				membershiprole.TenantIDEQ(tenantID),
				membershiprole.RoleIDIn(roleIDs...),
			),
		).
		Exec(ctx)
	if err != nil {
		r.log.Errorf("remove roles from membership failed: %s", err.Error())
		return userV1.ErrorInternalServerError("remove roles from membership failed")
	}
	return nil
}
