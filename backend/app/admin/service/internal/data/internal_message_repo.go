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

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/internalmessage"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"

	internalMessageV1 "go-wind-admin/api/gen/go/internal_message/service/v1"
)

type InternalMessageRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper          *mapper.CopierMapper[internalMessageV1.InternalMessage, ent.InternalMessage]
	statusConverter *mapper.EnumTypeConverter[internalMessageV1.InternalMessage_Status, internalmessage.Status]
	typeConverter   *mapper.EnumTypeConverter[internalMessageV1.InternalMessage_Type, internalmessage.Type]

	repository *entCrud.Repository[
		ent.InternalMessageQuery, ent.InternalMessageSelect,
		ent.InternalMessageCreate, ent.InternalMessageCreateBulk,
		ent.InternalMessageUpdate, ent.InternalMessageUpdateOne,
		ent.InternalMessageDelete,
		predicate.InternalMessage,
		internalMessageV1.InternalMessage, ent.InternalMessage,
	]
}

func NewInternalMessageRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *InternalMessageRepo {
	repo := &InternalMessageRepo{
		log:             ctx.NewLoggerHelper("internal-message/repo/admin-service"),
		entClient:       entClient,
		mapper:          mapper.NewCopierMapper[internalMessageV1.InternalMessage, ent.InternalMessage](),
		statusConverter: mapper.NewEnumTypeConverter[internalMessageV1.InternalMessage_Status, internalmessage.Status](internalMessageV1.InternalMessage_Status_name, internalMessageV1.InternalMessage_Status_value),
		typeConverter:   mapper.NewEnumTypeConverter[internalMessageV1.InternalMessage_Type, internalmessage.Type](internalMessageV1.InternalMessage_Type_name, internalMessageV1.InternalMessage_Type_value),
	}

	repo.init()

	return repo
}

func (r *InternalMessageRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.InternalMessageQuery, ent.InternalMessageSelect,
		ent.InternalMessageCreate, ent.InternalMessageCreateBulk,
		ent.InternalMessageUpdate, ent.InternalMessageUpdateOne,
		ent.InternalMessageDelete,
		predicate.InternalMessage,
		internalMessageV1.InternalMessage, ent.InternalMessage,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
	r.mapper.AppendConverters(r.typeConverter.NewConverterPair())
}

func (r *InternalMessageRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().InternalMessage.Query()
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

func (r *InternalMessageRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*internalMessageV1.ListInternalMessageResponse, error) {
	if req == nil {
		return nil, internalMessageV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().InternalMessage.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &internalMessageV1.ListInternalMessageResponse{Total: 0, Items: nil}, nil
	}

	return &internalMessageV1.ListInternalMessageResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *InternalMessageRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().InternalMessage.Query().
		Where(internalmessage.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, internalMessageV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *InternalMessageRepo) Get(ctx context.Context, req *internalMessageV1.GetInternalMessageRequest) (*internalMessageV1.InternalMessage, error) {
	if req == nil {
		return nil, internalMessageV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().InternalMessage.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *internalMessageV1.GetInternalMessageRequest_Id:
		whereCond = append(whereCond, internalmessage.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

func (r *InternalMessageRepo) Create(ctx context.Context, req *internalMessageV1.CreateInternalMessageRequest) (*internalMessageV1.InternalMessage, error) {
	if req == nil || req.Data == nil {
		return nil, internalMessageV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().InternalMessage.Create().
		SetNillableTenantID(req.Data.TenantId).
		SetNillableTitle(req.Data.Title).
		SetNillableContent(req.Data.Content).
		SetSenderID(req.Data.GetSenderId()).
		SetNillableCategoryID(req.Data.CategoryId).
		SetNillableStatus(r.statusConverter.ToEntity(req.Data.Status)).
		SetNillableType(r.typeConverter.ToEntity(req.Data.Type)).
		SetNillableCreatedBy(req.Data.CreatedBy).
		SetNillableCreatedAt(timeutil.TimestamppbToTime(req.Data.CreatedAt))

	if req.Data.CreatedAt == nil {
		builder.SetCreatedAt(time.Now())
	}

	if req.Data.Id != nil {
		builder.SetID(req.GetData().GetId())
	}

	var err error
	var entity *ent.InternalMessage
	if entity, err = builder.Save(ctx); err != nil {
		r.log.Errorf("insert internal message failed: %s", err.Error())
		return nil, internalMessageV1.ErrorInternalServerError("insert internal message failed")
	}

	return r.mapper.ToDTO(entity), nil
}

func (r *InternalMessageRepo) Update(ctx context.Context, req *internalMessageV1.UpdateInternalMessageRequest) error {
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
			createReq := &internalMessageV1.CreateInternalMessageRequest{Data: req.Data}
			createReq.Data.CreatedBy = createReq.Data.UpdatedBy
			createReq.Data.UpdatedBy = nil
			_, err = r.Create(ctx, createReq)
			return err
		}
	}

	builder := r.entClient.Client().Debug().InternalMessage.Update()
	err := r.repository.UpdateX(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *internalMessageV1.InternalMessage) {
			builder.
				SetNillableTitle(req.Data.Title).
				SetNillableContent(req.Data.Content).
				SetNillableSenderID(req.Data.SenderId).
				SetNillableCategoryID(req.Data.CategoryId).
				SetNillableStatus(r.statusConverter.ToEntity(req.Data.Status)).
				SetNillableType(r.typeConverter.ToEntity(req.Data.Type)).
				SetNillableUpdatedBy(req.Data.UpdatedBy).
				SetNillableUpdatedAt(timeutil.TimestamppbToTime(req.Data.UpdatedAt))

			if req.Data.UpdatedAt == nil {
				builder.SetUpdatedAt(time.Now())
			}
		},
		func(s *sql.Selector) {
			s.Where(sql.EQ(internalmessage.FieldID, req.GetId()))
		},
	)

	return err
}

func (r *InternalMessageRepo) Delete(ctx context.Context, id uint32) error {
	if id == 0 {
		return internalMessageV1.ErrorBadRequest("invalid parameter")
	}

	if err := r.entClient.Client().InternalMessage.DeleteOneID(id).Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return internalMessageV1.ErrorNotFound("internal message not found")
		}

		r.log.Errorf("delete one data failed: %s", err.Error())

		return internalMessageV1.ErrorInternalServerError("delete failed")
	}

	return nil
}
