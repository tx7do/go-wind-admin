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
	"go-wind-admin/app/admin/service/internal/data/ent/userposition"

	userV1 "go-wind-admin/api/gen/go/user/service/v1"
)

type UserPositionRepo struct {
	log             *log.Helper
	entClient       *entCrud.EntClient[*ent.Client]
	statusConverter *mapper.EnumTypeConverter[userV1.UserPosition_Status, userposition.Status]
}

func NewUserPositionRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *UserPositionRepo {
	return &UserPositionRepo{
		log:       ctx.NewLoggerHelper("user-position/repo/admin-service"),
		entClient: entClient,
		statusConverter: mapper.NewEnumTypeConverter[userV1.UserPosition_Status, userposition.Status](
			userV1.UserPosition_Status_name,
			userV1.UserPosition_Status_value,
		),
	}
}

// CleanPositions 删除会员在某租户下的所有职位关联
func (r *UserPositionRepo) CleanPositions(
	ctx context.Context,
	tx *ent.Tx,
	userID, tenantID uint32,
) error {
	if _, err := tx.UserPosition.Delete().
		Where(
			userposition.UserIDEQ(userID),
			userposition.TenantIDEQ(tenantID),
		).
		Exec(ctx); err != nil {
		r.log.Errorf("delete old user positions failed: %s", err.Error())
		return userV1.ErrorInternalServerError("delete old user positions failed")
	}
	return nil
}

// AssignPositions 分配岗位给用户
func (r *UserPositionRepo) AssignPositions(
	ctx context.Context,
	tx *ent.Tx,
	userID, tenantID uint32,
	datas []*userV1.UserPosition,
) error {
	var err error

	// 删除该用户的所有旧关联
	if err = r.CleanPositions(ctx, tx, userID, tenantID); err != nil {
		return userV1.ErrorInternalServerError("clean old user positions failed")
	}

	// 如果没有分配任何，则直接提交事务返回
	if len(datas) == 0 {
		return nil
	}

	now := time.Now()

	var userPositionCreates []*ent.UserPositionCreate
	for _, data := range datas {
		if data.StartAt == nil {
			data.StartAt = timeutil.TimeToTimestamppb(&now)
		}
		rm := tx.UserPosition.
			Create().
			SetTenantID(data.GetTenantId()).
			SetUserID(data.GetUserId()).
			SetPositionID(data.GetPositionId()).
			SetNillableStatus(r.statusConverter.ToEntity(data.Status)).
			SetNillableAssignedBy(data.AssignedBy).
			SetNillableAssignedAt(timeutil.TimestamppbToTime(data.AssignedAt)).
			SetNillableIsPrimary(data.IsPrimary).
			SetNillableStartAt(timeutil.TimestamppbToTime(data.StartAt)).
			SetNillableEndAt(timeutil.TimestamppbToTime(data.EndAt)).
			SetNillableCreatedBy(data.CreatedBy).
			SetCreatedAt(now)
		userPositionCreates = append(userPositionCreates, rm)
	}

	_, err = tx.UserPosition.CreateBulk(userPositionCreates...).Save(ctx)
	if err != nil {
		r.log.Errorf("assign positions to user failed: %s", err.Error())
		return userV1.ErrorInternalServerError("assign positions to user failed")
	}

	return nil
}

// ListPositionIDs 获取用户的岗位ID列表
func (r *UserPositionRepo) ListPositionIDs(ctx context.Context, userID uint32, excludeExpired bool) ([]uint32, error) {
	q := r.entClient.Client().UserPosition.Query().
		Where(
			userposition.UserIDEQ(userID),
		)

	if excludeExpired {
		now := time.Now()
		q = q.Where(
			userposition.Or(
				userposition.EndAtIsNil(),
				userposition.EndAtGT(now),
			),
		)
	}

	intIDs, err := q.
		Select(userposition.FieldPositionID).
		Ints(ctx)
	if err != nil {
		r.log.Errorf("query position ids by user id failed: %s", err.Error())
		return nil, userV1.ErrorInternalServerError("query position ids by user id failed")
	}
	ids := make([]uint32, len(intIDs))
	for i, v := range intIDs {
		ids[i] = uint32(v)
	}
	return ids, nil
}

// RemovePositions 从用户移除岗位
func (r *UserPositionRepo) RemovePositions(ctx context.Context, userID, tenantID uint32, positionIDs []uint32) error {
	_, err := r.entClient.Client().UserPosition.Delete().
		Where(
			userposition.And(
				userposition.UserIDEQ(userID),
				userposition.TenantIDEQ(tenantID),
				userposition.PositionIDIn(positionIDs...),
			),
		).
		Exec(ctx)
	if err != nil {
		r.log.Errorf("remove positions from user failed: %s", err.Error())
		return userV1.ErrorInternalServerError("remove positions from user failed")
	}
	return nil
}
