//go:build gorm_backend
// +build gorm_backend

// Package gorm 中的仓储是 ent 仓储的平行 gorm 镜像，作为"ent 为主力、gorm 为备选"脚手架的完整代码。
// 这些仓储为死代码：不接入 wire、不被 service 引用；采用者需要时自行装配。
//
// gorm 仓储不做租户隔离（ent 侧靠编译进生成代码的 privacy 策略自动注入，gorm 侧无此机制）。
// 直接切换 gorm 后端会有跨租户数据泄露风险，采用者须自行加 scope/plugin。
package gorm

import (
	"context"
	"errors"
	"time"

	gormDB "gorm.io/gorm"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	gormCrud "github.com/tx7do/go-crud/gorm"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/trans"

	"go-wind-admin/app/admin/service/internal/data/gorm/models"

	internalMessageV1 "go-wind-admin/api/gen/go/internal_message/service/v1"
)

type InternalMessageRecipientRepo struct {
	client     *gormCrud.Client
	log        *log.Helper
	mapper     *mapper.CopierMapper[internalMessageV1.InternalMessageRecipient, models.InternalMessageRecipient]
	repository *gormCrud.Repository[internalMessageV1.InternalMessageRecipient, models.InternalMessageRecipient]

	statusConverter *mapper.EnumTypeConverter[internalMessageV1.InternalMessageRecipient_Status, string]
}

func NewInternalMessageRecipientRepo(
	ctx *bootstrap.Context,
	client *gormCrud.Client,
) *InternalMessageRecipientRepo {
	repo := &InternalMessageRecipientRepo{
		log:    ctx.NewLoggerHelper("internal-message-recipient/gorm-repo/admin-service"),
		client: client,
		mapper: mapper.NewCopierMapper[internalMessageV1.InternalMessageRecipient, models.InternalMessageRecipient](),

		statusConverter: mapper.NewEnumTypeConverter[internalMessageV1.InternalMessageRecipient_Status, string](internalMessageV1.InternalMessageRecipient_Status_name, internalMessageV1.InternalMessageRecipient_Status_value),
	}

	repo.init()

	return repo
}

func (r *InternalMessageRecipientRepo) init() {
	r.repository = gormCrud.NewRepository[internalMessageV1.InternalMessageRecipient, models.InternalMessageRecipient](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
}

func (r *InternalMessageRecipientRepo) Count(ctx context.Context, scopes []func(*gormDB.DB) *gormDB.DB) (int, error) {
	count, err := r.repository.Count(ctx, r.client.DB, scopes)
	if err != nil {
		r.log.Errorf("query count failed: %s", err.Error())
		return 0, internalMessageV1.ErrorInternalServerError("query count failed")
	}

	return int(count), nil
}

func (r *InternalMessageRecipientRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.repository.ExistsWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", id) },
	})
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, internalMessageV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *InternalMessageRecipientRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*internalMessageV1.ListUserInboxResponse, error) {
	if req == nil {
		return nil, internalMessageV1.ErrorBadRequest("invalid parameter")
	}

	ret, err := r.repository.ListWithPaging(ctx, r.client.DB, req)
	if err != nil {
		return nil, internalMessageV1.ErrorInternalServerError("query list failed")
	}
	if ret == nil {
		return &internalMessageV1.ListUserInboxResponse{Total: 0, Items: nil}, nil
	}

	return &internalMessageV1.ListUserInboxResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *InternalMessageRecipientRepo) Get(ctx context.Context, req *internalMessageV1.GetInternalMessageRecipientRequest) (*internalMessageV1.InternalMessageRecipient, error) {
	if req == nil {
		return nil, internalMessageV1.ErrorBadRequest("invalid parameter")
	}

	var scopes []func(*gormDB.DB) *gormDB.DB
	switch req.QueryBy.(type) {
	default:
	case *internalMessageV1.GetInternalMessageRecipientRequest_Id:
		scopes = append(scopes, func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", req.GetId()) })
	}

	dto, err := r.repository.GetWithFilters(ctx, r.client.DB, scopes, req.GetViewMask())
	if err != nil {
		if errors.Is(err, gormDB.ErrRecordNotFound) {
			return nil, internalMessageV1.ErrorNotFound("internal message recipient not found")
		}
		r.log.Errorf("query internal message recipient failed: %s", err.Error())
		return nil, internalMessageV1.ErrorInternalServerError("query internal message recipient failed")
	}

	return dto, nil
}

func (r *InternalMessageRecipientRepo) Create(ctx context.Context, req *internalMessageV1.InternalMessageRecipient) (*internalMessageV1.InternalMessageRecipient, error) {
	if req == nil {
		return nil, internalMessageV1.ErrorBadRequest("invalid parameter")
	}

	dto, err := r.repository.Create(ctx, r.client.DB, req, nil)
	if err != nil {
		r.log.Errorf("insert internal message recipient failed: %s", err.Error())
		return nil, internalMessageV1.ErrorInternalServerError("insert internal message recipient failed")
	}

	return dto, nil
}

func (r *InternalMessageRecipientRepo) Update(ctx context.Context, req *internalMessageV1.UpdateInternalMessageRecipientRequest) error {
	if req == nil || req.Data == nil {
		return internalMessageV1.ErrorBadRequest("invalid parameter")
	}
	if req.GetId() == 0 {
		return internalMessageV1.ErrorBadRequest("id is required")
	}

	// 如果不存在则创建
	if req.GetAllowMissing() {
		exist, err := r.IsExist(ctx, req.GetId())
		if err != nil {
			return err
		}
		if !exist {
			req.Data.CreatedBy = req.Data.UpdatedBy
			req.Data.UpdatedBy = nil
			_, err = r.Create(ctx, req.Data)
			return err
		}
	}

	if _, err := r.repository.UpdateWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", req.GetId()) },
	}, req.Data, req.GetUpdateMask()); err != nil {
		r.log.Errorf("update internal message recipient failed: %s", err.Error())
		return internalMessageV1.ErrorInternalServerError("update internal message recipient failed")
	}

	return nil
}

func (r *InternalMessageRecipientRepo) Delete(ctx context.Context, id uint32) error {
	if id == 0 {
		return internalMessageV1.ErrorBadRequest("invalid parameter")
	}

	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("id = ?", id) },
	}); err != nil {
		r.log.Errorf("delete internal message recipient failed: %s", err.Error())
		return internalMessageV1.ErrorInternalServerError("delete failed")
	}

	return nil
}

// MarkNotificationAsRead 将通知标记为已读
func (r *InternalMessageRecipientRepo) MarkNotificationAsRead(ctx context.Context, req *internalMessageV1.MarkNotificationAsReadRequest) error {
	if len(req.GetRecipientIds()) == 0 {
		return internalMessageV1.ErrorBadRequest("invalid parameter")
	}
	if req.GetUserId() == 0 {
		return internalMessageV1.ErrorBadRequest("invalid parameter")
	}

	now := time.Now()
	statusEntity := r.statusConverter.ToEntity(trans.Ptr(internalMessageV1.InternalMessageRecipient_READ))
	if statusEntity == nil {
		return internalMessageV1.ErrorBadRequest("invalid parameter")
	}

	err := r.client.DB.WithContext(ctx).
		Model(&models.InternalMessageRecipient{}).
		Where("id IN ? AND recipient_user_id = ? AND status <> ?", req.GetRecipientIds(), req.GetUserId(), *statusEntity).
		Updates(map[string]interface{}{
			"status":     *statusEntity,
			"read_at":    now,
			"updated_at": now,
		}).Error
	return err
}

// MarkNotificationsStatus 标记特定用户的某些或所有通知的状态
func (r *InternalMessageRecipientRepo) MarkNotificationsStatus(ctx context.Context, req *internalMessageV1.MarkNotificationsStatusRequest) error {
	if len(req.GetRecipientIds()) == 0 {
		return internalMessageV1.ErrorBadRequest("invalid parameter")
	}
	if req.GetUserId() == 0 {
		return internalMessageV1.ErrorBadRequest("invalid parameter")
	}

	now := time.Now()
	statusEntity := r.statusConverter.ToEntity(trans.Ptr(req.GetNewStatus()))
	if statusEntity == nil {
		return internalMessageV1.ErrorBadRequest("invalid parameter")
	}

	var readAt *time.Time
	var receiveAt *time.Time
	switch req.GetNewStatus() {
	case internalMessageV1.InternalMessageRecipient_READ:
		readAt = trans.Ptr(now)
	case internalMessageV1.InternalMessageRecipient_RECEIVED:
		receiveAt = trans.Ptr(now)
	}

	updates := map[string]interface{}{
		"status":     *statusEntity,
		"updated_at": now,
	}
	if readAt != nil {
		updates["read_at"] = *readAt
	}
	if receiveAt != nil {
		updates["received_at"] = *receiveAt
	}

	err := r.client.DB.WithContext(ctx).
		Model(&models.InternalMessageRecipient{}).
		Where("id IN ? AND recipient_user_id = ? AND status <> ?", req.GetRecipientIds(), req.GetUserId(), *statusEntity).
		Updates(updates).Error
	return err
}

// RevokeMessage 撤销某条消息
func (r *InternalMessageRecipientRepo) RevokeMessage(ctx context.Context, req *internalMessageV1.RevokeMessageRequest) error {
	err := r.client.DB.WithContext(ctx).
		Model(&models.InternalMessageRecipient{}).
		Where("message_id = ? AND recipient_user_id = ?", req.GetMessageId(), req.GetUserId()).
		Delete(&models.InternalMessageRecipient{}).Error
	return err
}

func (r *InternalMessageRecipientRepo) DeleteNotificationFromInbox(ctx context.Context, req *internalMessageV1.DeleteNotificationFromInboxRequest) error {
	err := r.client.DB.WithContext(ctx).
		Model(&models.InternalMessageRecipient{}).
		Where("id IN ? AND recipient_user_id = ?", req.GetRecipientIds(), req.GetUserId()).
		Delete(&models.InternalMessageRecipient{}).Error
	return err
}
