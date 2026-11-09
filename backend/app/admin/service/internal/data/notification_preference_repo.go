package data

import (
	"context"

	"github.com/tx7do/go-utils/timeutil"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	entCrud "github.com/tx7do/go-crud/entgo"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/notificationpreference"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	notificationV1 "go-wind-admin/api/gen/go/notification/service/v1"
)

// NotificationPreferenceRepo 用户通知偏好仓储。
// schema 刻意无租户 mixin（偏好跟人不跟租户），读锚是 user_id。
type NotificationPreferenceRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *bLogger.Helper
}

func NewNotificationPreferenceRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *NotificationPreferenceRepo {
	return &NotificationPreferenceRepo{
		log:       ctx.NewLoggerHelper("notification-preference/repo/admin-service"),
		entClient: entClient,
	}
}

func toNotificationPreferenceDTO(entity *ent.NotificationPreference) *notificationV1.NotificationPreference {
	if entity == nil {
		return nil
	}
	return &notificationV1.NotificationPreference{
		UserId:           trans.Ptr(entity.UserID),
		QuietEnabled:     trans.Ptr(entity.QuietEnabled),
		QuietStartMinute: trans.Ptr(entity.QuietStartMinute),
		QuietEndMinute:   trans.Ptr(entity.QuietEndMinute),
		MutedCategoryIds: entity.MutedCategoryIds,
		UpdatedAt:        timeutil.TimeToTimestamppb(entity.UpdatedAt),
	}
}

// GetByUserID 读某用户的偏好；从未配置过时返回 (nil, nil)，由调用方决定默认值。
func (r *NotificationPreferenceRepo) GetByUserID(ctx context.Context, userID uint64) (*notificationV1.NotificationPreference, error) {
	entity, err := r.entClient.Client().NotificationPreference.Query().
		Where(notificationpreference.UserIDEQ(userID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		r.log.Errorf(ctx, "query notification preference by user [%d] failed: %s", userID, err.Error())
		return nil, adminV1.ErrorInternalServerError("query notification preference failed")
	}

	return toNotificationPreferenceDTO(entity), nil
}

// ListByUserIDs 批量读一组用户的偏好（广播按页处理受众时用），缺偏好的用户不在返回值里。
func (r *NotificationPreferenceRepo) ListByUserIDs(ctx context.Context, userIDs []uint64) (map[uint64]*notificationV1.NotificationPreference, error) {
	if len(userIDs) == 0 {
		return map[uint64]*notificationV1.NotificationPreference{}, nil
	}

	entities, err := r.entClient.Client().NotificationPreference.Query().
		Where(notificationpreference.UserIDIn(userIDs...)).
		All(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query notification preferences by users failed: %s", err.Error())
		return nil, adminV1.ErrorInternalServerError("query notification preferences failed")
	}

	result := make(map[uint64]*notificationV1.NotificationPreference, len(entities))
	for _, entity := range entities {
		result[entity.UserID] = toNotificationPreferenceDTO(entity)
	}
	return result, nil
}

// Upsert 按用户保存偏好（存在则更新，不存在则创建）；userID 由调用方从操作人钉定。
func (r *NotificationPreferenceRepo) Upsert(ctx context.Context, userID uint64, pref *notificationV1.UpdateNotificationPreferenceRequest, operatorUserID uint64) (*notificationV1.NotificationPreference, error) {
	if pref == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	existing, err := r.entClient.Client().NotificationPreference.Query().
		Where(notificationpreference.UserIDEQ(userID)).
		Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		r.log.Errorf(ctx, "query notification preference for upsert failed: %s", err.Error())
		return nil, adminV1.ErrorInternalServerError("save notification preference failed")
	}

	muted := make([]uint64, 0, len(pref.GetMutedCategoryIds()))
	muted = append(muted, pref.GetMutedCategoryIds()...)

	if existing == nil {
		entity, err := r.entClient.Client().NotificationPreference.Create().
			SetUserID(userID).
			SetQuietEnabled(pref.GetQuietEnabled()).
			SetQuietStartMinute(pref.GetQuietStartMinute()).
			SetQuietEndMinute(pref.GetQuietEndMinute()).
			SetMutedCategoryIds(muted).
			SetNillableCreatedBy(operatorUserIDOrNil(operatorUserID)).
			Save(ctx)
		if err != nil {
			r.log.Errorf(ctx, "create notification preference for user [%d] failed: %s", userID, err.Error())
			return nil, adminV1.ErrorInternalServerError("save notification preference failed")
		}
		return toNotificationPreferenceDTO(entity), nil
	}

	entity, err := existing.Update().
		SetQuietEnabled(pref.GetQuietEnabled()).
		SetQuietStartMinute(pref.GetQuietStartMinute()).
		SetQuietEndMinute(pref.GetQuietEndMinute()).
		SetMutedCategoryIds(muted).
		SetNillableUpdatedBy(operatorUserIDOrNil(operatorUserID)).
		Save(ctx)
	if err != nil {
		r.log.Errorf(ctx, "update notification preference for user [%d] failed: %s", userID, err.Error())
		return nil, adminV1.ErrorInternalServerError("save notification preference failed")
	}

	return toNotificationPreferenceDTO(entity), nil
}

func operatorUserIDOrNil(userID uint64) *uint64 {
	if userID == 0 {
		return nil
	}
	return trans.Ptr(userID)
}
