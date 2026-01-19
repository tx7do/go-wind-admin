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
	"go-wind-admin/app/admin/service/internal/data/ent/membershipposition"

	userV1 "go-wind-admin/api/gen/go/user/service/v1"
)

type MembershipPositionRepo struct {
	log             *log.Helper
	entClient       *entCrud.EntClient[*ent.Client]
	statusConverter *mapper.EnumTypeConverter[userV1.MembershipPosition_Status, membershipposition.Status]
}

func NewMembershipPositionRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *MembershipPositionRepo {
	return &MembershipPositionRepo{
		log:       ctx.NewLoggerHelper("membership-position/repo/admin-service"),
		entClient: entClient,
		statusConverter: mapper.NewEnumTypeConverter[userV1.MembershipPosition_Status, membershipposition.Status](
			userV1.MembershipPosition_Status_name,
			userV1.MembershipPosition_Status_value,
		),
	}
}

// CleanRelationsByMembershipID 删除会员在某租户下的所有职位关联
func (r *MembershipPositionRepo) CleanRelationsByMembershipID(ctx context.Context, tx *ent.Tx, membershipID uint32) error {
	if membershipID == 0 {
		return nil
	}

	if _, err := tx.MembershipPosition.Delete().
		Where(
			membershipposition.MembershipIDEQ(membershipID),
		).
		Exec(ctx); err != nil {
		r.log.Errorf("delete old membership positions failed: %s", err.Error())
		return userV1.ErrorInternalServerError("delete old membership positions failed")
	}
	return nil
}

// CleanRelationsByMembershipIDs 删除多个会员的所有职位关联
func (r *MembershipPositionRepo) CleanRelationsByMembershipIDs(ctx context.Context, tx *ent.Tx, membershipIDs []uint32) error {
	if len(membershipIDs) == 0 {
		return nil
	}

	if _, err := tx.MembershipPosition.Delete().
		Where(
			membershipposition.MembershipIDIn(membershipIDs...),
		).
		Exec(ctx); err != nil {
		r.log.Errorf("delete old membership positions by membership ids failed: %s", err.Error())
		return userV1.ErrorInternalServerError("delete old membership positions by membership ids failed")
	}
	return nil
}

// CleanRelationsByPositionID 删除岗位的所有会员关联
func (r *MembershipPositionRepo) CleanRelationsByPositionID(ctx context.Context, tx *ent.Tx, positionID uint32) error {
	if positionID == 0 {
		return nil
	}

	if _, err := tx.MembershipPosition.Delete().
		Where(
			membershipposition.PositionIDEQ(positionID),
		).
		Exec(ctx); err != nil {
		r.log.Errorf("delete old membership positions by position id failed: %s", err.Error())
		return userV1.ErrorInternalServerError("delete old membership positions by position id failed")
	}
	return nil
}

// CleanRelationsByPositionIDs 删除多个岗位的所有会员关联
func (r *MembershipPositionRepo) CleanRelationsByPositionIDs(ctx context.Context, tx *ent.Tx, positionIDs []uint32) error {
	if len(positionIDs) == 0 {
		return nil
	}

	if _, err := tx.MembershipPosition.Delete().
		Where(
			membershipposition.PositionIDIn(positionIDs...),
		).
		Exec(ctx); err != nil {
		r.log.Errorf("delete old membership positions by position ids failed: %s", err.Error())
		return userV1.ErrorInternalServerError("delete old membership positions by position ids failed")
	}
	return nil
}

// RemovePositionsFromMembership 从用户移除岗位
func (r *MembershipPositionRepo) RemovePositionsFromMembership(ctx context.Context, membershipID uint32, positionIDs []uint32) error {
	if membershipID == 0 || len(positionIDs) == 0 {
		return nil
	}

	_, err := r.entClient.Client().MembershipPosition.Delete().
		Where(
			membershipposition.And(
				membershipposition.MembershipIDEQ(membershipID),
				membershipposition.PositionIDIn(positionIDs...),
			),
		).
		Exec(ctx)
	if err != nil {
		r.log.Errorf("remove positions from membership failed: %s", err.Error())
		return userV1.ErrorInternalServerError("remove positions from membership failed")
	}
	return nil
}

// AssignMembershipPositions 分配岗位给用户
func (r *MembershipPositionRepo) AssignMembershipPositions(
	ctx context.Context,
	tx *ent.Tx,
	membershipID uint32,
	datas []*userV1.MembershipPosition,
) error {
	if len(datas) == 0 || membershipID == 0 {
		return nil
	}

	var err error

	// 删除该用户的所有旧关联
	if err = r.CleanRelationsByMembershipID(ctx, tx, membershipID); err != nil {
		return userV1.ErrorInternalServerError("clean old membership positions failed")
	}

	now := time.Now()

	var membershipPositionCreates []*ent.MembershipPositionCreate
	for _, data := range datas {
		if data.StartAt == nil {
			data.StartAt = timeutil.TimeToTimestamppb(&now)
		}
		rm := tx.MembershipPosition.
			Create().
			SetNillableTenantID(data.TenantId).
			SetMembershipID(data.GetMembershipId()).
			SetPositionID(data.GetPositionId()).
			SetNillableStatus(r.statusConverter.ToEntity(data.Status)).
			SetNillableAssignedBy(data.AssignedBy).
			SetNillableAssignedAt(timeutil.TimestamppbToTime(data.AssignedAt)).
			SetNillableIsPrimary(data.IsPrimary).
			SetNillableStartAt(timeutil.TimestamppbToTime(data.StartAt)).
			SetNillableEndAt(timeutil.TimestamppbToTime(data.EndAt)).
			SetNillableCreatedBy(data.CreatedBy).
			SetCreatedAt(now)
		membershipPositionCreates = append(membershipPositionCreates, rm)
	}

	_, err = tx.MembershipPosition.CreateBulk(membershipPositionCreates...).Save(ctx)
	if err != nil {
		r.log.Errorf("assign positions to membership failed: %s", err.Error())
		return userV1.ErrorInternalServerError("assign positions to membership failed")
	}

	return nil
}

// ListPositionIDs 获取用户的岗位ID列表
func (r *MembershipPositionRepo) ListPositionIDs(ctx context.Context, membershipID uint32, excludeExpired bool) ([]uint32, error) {
	if membershipID == 0 {
		return []uint32{}, nil
	}

	q := r.entClient.Client().MembershipPosition.Query().
		Where(
			membershipposition.MembershipIDEQ(membershipID),
		)

	if excludeExpired {
		now := time.Now()
		q = q.Where(
			membershipposition.Or(
				membershipposition.EndAtIsNil(),
				membershipposition.EndAtGT(now),
			),
		)
	}

	intIDs, err := q.
		Select(membershipposition.FieldPositionID).
		Ints(ctx)
	if err != nil {
		r.log.Errorf("query position ids by membership id failed: %s", err.Error())
		return nil, userV1.ErrorInternalServerError("query position ids by membership id failed")
	}
	ids := make([]uint32, len(intIDs))
	for i, v := range intIDs {
		ids[i] = uint32(v)
	}
	return ids, nil
}

// ListMembershipIDs 获取岗位关联的会员ID列表
func (r *MembershipPositionRepo) ListMembershipIDs(ctx context.Context, positionID uint32, excludeExpired bool) ([]uint32, error) {
	if positionID == 0 {
		return []uint32{}, nil
	}

	q := r.entClient.Client().MembershipPosition.Query().
		Where(
			membershipposition.PositionIDEQ(positionID),
		)

	if excludeExpired {
		now := time.Now()
		q = q.Where(
			membershipposition.Or(
				membershipposition.EndAtIsNil(),
				membershipposition.EndAtGT(now),
			),
		)
	}

	intIDs, err := q.
		Select(membershipposition.FieldMembershipID).
		Ints(ctx)
	if err != nil {
		r.log.Errorf("query membership ids by position id failed: %s", err.Error())
		return nil, userV1.ErrorInternalServerError("query membership ids by position id failed")
	}
	ids := make([]uint32, len(intIDs))
	for i, v := range intIDs {
		ids[i] = uint32(v)
	}
	return ids, nil
}

// ListMembershipIDsByPositionIDs 获取多个岗位关联的会员ID列表
func (r *MembershipPositionRepo) ListMembershipIDsByPositionIDs(ctx context.Context, positionIDs []uint32, excludeExpired bool) ([]uint32, error) {
	if len(positionIDs) == 0 {
		return nil, nil
	}

	q := r.entClient.Client().MembershipPosition.Query().
		Where(
			membershipposition.PositionIDIn(positionIDs...),
		)

	if excludeExpired {
		now := time.Now()
		q = q.Where(
			membershipposition.Or(
				membershipposition.EndAtIsNil(),
				membershipposition.EndAtGT(now),
			),
		)
	}

	intIDs, err := q.
		Select(membershipposition.FieldMembershipID).
		Ints(ctx)
	if err != nil {
		r.log.Errorf("query membership ids by position ids failed: %s", err.Error())
		return nil, userV1.ErrorInternalServerError("query membership ids by position ids failed")
	}
	ids := make([]uint32, len(intIDs))
	for i, v := range intIDs {
		ids[i] = uint32(v)
	}
	return ids, nil
}
