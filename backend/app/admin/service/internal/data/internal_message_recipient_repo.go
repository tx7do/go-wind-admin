package data

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/timeutil"
	"github.com/tx7do/go-utils/trans"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/internalmessagerecipient"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"

	internalMessageV1 "go-wind-admin/api/gen/go/internal_message/service/v1"
)

type InternalMessageRecipientRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper          *mapper.CopierMapper[internalMessageV1.InternalMessageRecipient, ent.InternalMessageRecipient]
	statusConverter *mapper.EnumTypeConverter[internalMessageV1.InternalMessageRecipient_Status, internalmessagerecipient.Status]

	repository *entCrud.Repository[
		ent.InternalMessageRecipientQuery, ent.InternalMessageRecipientSelect,
		ent.InternalMessageRecipientCreate, ent.InternalMessageRecipientCreateBulk,
		ent.InternalMessageRecipientUpdate, ent.InternalMessageRecipientUpdateOne,
		ent.InternalMessageRecipientDelete,
		predicate.InternalMessageRecipient,
		internalMessageV1.InternalMessageRecipient, ent.InternalMessageRecipient,
	]
}

func NewInternalMessageRecipientRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *InternalMessageRecipientRepo {
	repo := &InternalMessageRecipientRepo{
		log:             ctx.NewLoggerHelper("internal-message-recipient/repo/admin-service"),
		entClient:       entClient,
		mapper:          mapper.NewCopierMapper[internalMessageV1.InternalMessageRecipient, ent.InternalMessageRecipient](),
		statusConverter: mapper.NewEnumTypeConverter[internalMessageV1.InternalMessageRecipient_Status, internalmessagerecipient.Status](internalMessageV1.InternalMessageRecipient_Status_name, internalMessageV1.InternalMessageRecipient_Status_value),
	}

	repo.init()

	return repo
}

func (r *InternalMessageRecipientRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.InternalMessageRecipientQuery, ent.InternalMessageRecipientSelect,
		ent.InternalMessageRecipientCreate, ent.InternalMessageRecipientCreateBulk,
		ent.InternalMessageRecipientUpdate, ent.InternalMessageRecipientUpdateOne,
		ent.InternalMessageRecipientDelete,
		predicate.InternalMessageRecipient,
		internalMessageV1.InternalMessageRecipient, ent.InternalMessageRecipient,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
}

func (r *InternalMessageRecipientRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().InternalMessageRecipient.Query()
	if len(whereCond) != 0 {
		builder.Modify(whereCond...)
	}

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf("query count failed: %s", err.Error())
		return 0, internalMessageV1.ErrorInternalServerError("query count failed")
	}

	return count, nil
}

func (r *InternalMessageRecipientRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().InternalMessageRecipient.Query().
		Where(internalmessagerecipient.IDEQ(id)).
		Exist(ctx)
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

	builder := r.entClient.Client().InternalMessageRecipient.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
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

	builder := r.entClient.Client().InternalMessageRecipient.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *internalMessageV1.GetInternalMessageRecipientRequest_Id:
		whereCond = append(whereCond, internalmessagerecipient.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

func (r *InternalMessageRecipientRepo) Create(ctx context.Context, req *internalMessageV1.InternalMessageRecipient) (*internalMessageV1.InternalMessageRecipient, error) {
	if req == nil {
		return nil, internalMessageV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().InternalMessageRecipient.Create().
		SetNillableTenantID(req.TenantId).
		SetNillableMessageID(req.MessageId).
		SetNillableRecipientUserID(req.RecipientUserId).
		SetNillableStatus(r.statusConverter.ToEntity(req.Status)).
		SetNillableReceivedAt(timeutil.TimestamppbToTime(req.ReceivedAt)).
		SetNillableReadAt(timeutil.TimestamppbToTime(req.ReadAt)).
		SetCreatedAt(time.Now())

	var err error
	var entity *ent.InternalMessageRecipient
	if entity, err = builder.Save(ctx); err != nil {
		r.log.Errorf("insert internal message recipient failed: %s", err.Error())
		return nil, internalMessageV1.ErrorInternalServerError("insert internal message recipient failed")
	}

	return r.mapper.ToDTO(entity), nil
}

func (r *InternalMessageRecipientRepo) Update(ctx context.Context, req *internalMessageV1.UpdateInternalMessageRecipientRequest) error {
	if req == nil || req.Data == nil {
		return internalMessageV1.ErrorBadRequest("invalid parameter")
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

	builder := r.entClient.Client().Debug().InternalMessageRecipient.Update()
	err := r.repository.UpdateX(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *internalMessageV1.InternalMessageRecipient) {
			builder.
				SetNillableMessageID(req.Data.MessageId).
				SetNillableRecipientUserID(req.Data.RecipientUserId).
				SetNillableStatus(r.statusConverter.ToEntity(req.Data.Status)).
				SetNillableReceivedAt(timeutil.TimestamppbToTime(req.Data.ReceivedAt)).
				SetNillableReadAt(timeutil.TimestamppbToTime(req.Data.ReadAt)).
				SetUpdatedAt(time.Now())
		},
		func(s *sql.Selector) {
			s.Where(sql.EQ(internalmessagerecipient.FieldID, req.GetId()))
		},
	)

	return err
}

func (r *InternalMessageRecipientRepo) Delete(ctx context.Context, id uint32) error {
	if id == 0 {
		return internalMessageV1.ErrorBadRequest("invalid parameter")
	}

	if err := r.entClient.Client().InternalMessageRecipient.DeleteOneID(id).Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return internalMessageV1.ErrorNotFound("internal message recipient not found")
		}

		r.log.Errorf("delete one data failed: %s", err.Error())

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
	_, err := r.entClient.Client().InternalMessageRecipient.Update().
		Where(
			internalmessagerecipient.IDIn(req.GetRecipientIds()...),
			internalmessagerecipient.RecipientUserIDEQ(req.GetUserId()),
			internalmessagerecipient.StatusNEQ(internalmessagerecipient.StatusRead),
		).
		SetStatus(internalmessagerecipient.StatusRead).
		SetNillableReadAt(trans.Ptr(now)).
		SetNillableUpdatedAt(trans.Ptr(now)).
		Save(ctx)
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
	var readAt *time.Time
	var receiveAt *time.Time
	switch req.GetNewStatus() {
	case internalMessageV1.InternalMessageRecipient_READ:
		readAt = trans.Ptr(now)
	case internalMessageV1.InternalMessageRecipient_RECEIVED:
		receiveAt = trans.Ptr(now)
	}

	_, err := r.entClient.Client().InternalMessageRecipient.Update().
		Where(
			internalmessagerecipient.IDIn(req.GetRecipientIds()...),
			internalmessagerecipient.RecipientUserIDEQ(req.GetUserId()),
			internalmessagerecipient.StatusNEQ(*r.statusConverter.ToEntity(trans.Ptr(req.GetNewStatus()))),
		).
		SetNillableStatus(r.statusConverter.ToEntity(trans.Ptr(req.GetNewStatus()))).
		SetNillableReadAt(readAt).
		SetNillableReceivedAt(receiveAt).
		SetNillableUpdatedAt(trans.Ptr(now)).
		Save(ctx)
	return err
}

// RevokeMessage 撤销某条消息
func (r *InternalMessageRecipientRepo) RevokeMessage(ctx context.Context, req *internalMessageV1.RevokeMessageRequest) error {
	_, err := r.entClient.Client().InternalMessageRecipient.Delete().
		Where(
			internalmessagerecipient.MessageIDEQ(req.GetMessageId()),
			internalmessagerecipient.RecipientUserIDEQ(req.GetUserId()),
		).
		Exec(ctx)
	return err
}

func (r *InternalMessageRecipientRepo) DeleteNotificationFromInbox(ctx context.Context, req *internalMessageV1.DeleteNotificationFromInboxRequest) error {
	_, err := r.entClient.Client().InternalMessageRecipient.Delete().
		Where(
			internalmessagerecipient.IDIn(req.GetRecipientIds()...),
			internalmessagerecipient.RecipientUserIDEQ(req.GetUserId()),
		).
		Exec(ctx)
	return err
}
