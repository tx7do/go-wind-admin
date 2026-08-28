package data

import (
	"context"
	"errors"
	"time"

	"entgo.io/ent/dialect/sql"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/timeutil"
	"github.com/tx7do/go-utils/trans"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/internalmessage"
	"go-wind-admin/app/admin/service/internal/data/ent/internalmessagerecipient"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"

	internalMessageV1 "go-wind-admin/api/gen/go/internal_message/service/v1"
)

type InternalMessageRecipientRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *bLogger.Helper

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
		r.log.Errorf(ctx, "query count failed: %s", err.Error())
		return 0, internalMessageV1.ErrorInternalServerError("query count failed")
	}

	return count, nil
}

func (r *InternalMessageRecipientRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().InternalMessageRecipient.Query().
		Where(internalmessagerecipient.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query exist failed: %s", err.Error())
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
		r.log.Errorf(ctx, "insert internal message recipient failed: %s", err.Error())
		return nil, internalMessageV1.ErrorInternalServerError("insert internal message recipient failed")
	}

	return r.mapper.ToDTO(entity), nil
}

// recipientInsertBatchSize 批量插入收件记录的单批行数。
// 控制单条 INSERT 语句的大小，避免超过 max_allowed_packet 或单事务过大。
const recipientInsertBatchSize = 500

// CreateBulk 批量插入收件记录，超出单批上限时自动分批。
// 使用 ON CONFLICT (message_id, recipient_user_id) DO NOTHING：
// (message_id, recipient_user_id) 有唯一约束，asynq 广播任务重试时若某批已落库，
// 冲突行会被忽略而非报错回滚，保证幂等。upsert 模式下数据库不返回实体，
// 故本方法不返回创建结果，调用方（广播 handler）按入参数视为已投递并逐条推送 SSE。
// 部分批次因非冲突错误失败时跳过该批继续后续批次，聚合错误返回。
func (r *InternalMessageRecipientRepo) CreateBulk(ctx context.Context, reqs []*internalMessageV1.InternalMessageRecipient) error {
	if len(reqs) == 0 {
		return nil
	}

	var errs error
	for start := 0; start < len(reqs); start += recipientInsertBatchSize {
		end := min(start+recipientInsertBatchSize, len(reqs))

		builders := make([]*ent.InternalMessageRecipientCreate, 0, end-start)
		for _, req := range reqs[start:end] {
			builders = append(builders, r.entClient.Client().InternalMessageRecipient.Create().
				SetNillableTenantID(req.TenantId).
				SetNillableMessageID(req.MessageId).
				SetNillableRecipientUserID(req.RecipientUserId).
				SetNillableStatus(r.statusConverter.ToEntity(req.Status)).
				SetNillableReceivedAt(timeutil.TimestamppbToTime(req.ReceivedAt)).
				SetNillableReadAt(timeutil.TimestamppbToTime(req.ReadAt)).
				SetCreatedAt(time.Now()))
		}

		// 唯一约束 (message_id, recipient_user_id) 冲突时忽略该行，不更新任何字段。
		// 见 schema/internal_message_recipient.go 的 Unique index 与 ent feature sql/upsert。
		err := r.entClient.Client().InternalMessageRecipient.CreateBulk(builders...).
			OnConflict(
				sql.ResolveWithIgnore(),
				sql.ConflictColumns(internalmessagerecipient.FieldMessageID, internalmessagerecipient.FieldRecipientUserID),
			).
			Exec(ctx)
		if err != nil {
			r.log.Errorf(ctx, "bulk insert internal message recipients failed: %s", err.Error())
			errs = errors.Join(errs, err)
		}
	}

	return errs
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

	builder := r.entClient.Client().InternalMessageRecipient.Update()
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

		r.log.Errorf(ctx, "delete one data failed: %s", err.Error())

		return internalMessageV1.ErrorInternalServerError("delete failed")
	}

	return nil
}

// MarkNotificationAsRead 将通知标记为已读。
// recipient_ids 为空表示"标记该用户全部未读"——前端"全部已读"入口只加载了当前页数据，
// 无法枚举全部 id，由服务端按用户维度兜底（status <> READ 的守卫保证已读记录不被重写）。
func (r *InternalMessageRecipientRepo) MarkNotificationAsRead(ctx context.Context, req *internalMessageV1.MarkNotificationAsReadRequest) error {
	if req.GetUserId() == 0 {
		return internalMessageV1.ErrorBadRequest("invalid parameter")
	}

	now := time.Now()
	builder := r.entClient.Client().InternalMessageRecipient.Update().
		Where(
			internalmessagerecipient.RecipientUserIDEQ(req.GetUserId()),
			internalmessagerecipient.StatusNEQ(internalmessagerecipient.StatusRead),
		)
	if len(req.GetRecipientIds()) > 0 {
		builder = builder.Where(internalmessagerecipient.IDIn(req.GetRecipientIds()...))
	}
	_, err := builder.
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

// RevokeMessageWithMessage 撤销消息。
// user_id == 0 为全局撤销：同一事务内删除消息本体与全部收件记录——两表各自删除无共享事务时，
// 半成功会留下指向已删除消息的幽灵收件记录；user_id > 0 为单用户撤销：仅删除该用户的收件记录，
// 消息本体与其他收件人不受影响（单条语句，无需事务）。
func (r *InternalMessageRecipientRepo) RevokeMessageWithMessage(ctx context.Context, req *internalMessageV1.RevokeMessageRequest) (err error) {
	if req == nil || req.GetMessageId() == 0 {
		return internalMessageV1.ErrorBadRequest("invalid parameter")
	}

	if req.GetUserId() > 0 {
		if _, err = r.entClient.Client().InternalMessageRecipient.Delete().
			Where(
				internalmessagerecipient.MessageIDEQ(req.GetMessageId()),
				internalmessagerecipient.RecipientUserIDEQ(req.GetUserId()),
			).
			Exec(ctx); err != nil {
			r.log.Errorf(ctx, "revoke recipients failed: %s", err.Error())
			return internalMessageV1.ErrorInternalServerError("revoke recipients failed")
		}
		return nil
	}

	var tx *ent.Tx
	tx, err = r.entClient.Client().Tx(ctx)
	if err != nil {
		r.log.Errorf(ctx, "start transaction failed: %s", err.Error())
		return internalMessageV1.ErrorInternalServerError("start transaction failed")
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				r.log.Errorf(ctx, "transaction rollback failed: %s", rollbackErr.Error())
			}
			return
		}
		if commitErr := tx.Commit(); commitErr != nil {
			r.log.Errorf(ctx, "transaction commit failed: %s", commitErr.Error())
			err = internalMessageV1.ErrorInternalServerError("transaction commit failed")
		}
	}()

	var deleted int
	if deleted, err = tx.InternalMessage.Delete().
		Where(internalmessage.IDEQ(req.GetMessageId())).
		Exec(ctx); err != nil {
		r.log.Errorf(ctx, "delete message failed: %s", err.Error())
		return internalMessageV1.ErrorInternalServerError("delete message failed")
	}
	if deleted == 0 {
		err = internalMessageV1.ErrorNotFound("internal message not found")
		return err
	}

	if _, err = tx.InternalMessageRecipient.Delete().
		Where(internalmessagerecipient.MessageIDEQ(req.GetMessageId())).
		Exec(ctx); err != nil {
		r.log.Errorf(ctx, "revoke recipients failed: %s", err.Error())
		return internalMessageV1.ErrorInternalServerError("revoke recipients failed")
	}

	return nil
}

// DeleteNotificationFromInbox 删除用户收件箱中的通知记录。
// recipient_ids 为空表示清空该用户收件箱（前端"清空"入口无法枚举全部ID，
// 与 MarkNotificationAsRead 的空 ids 语义保持一致）。
func (r *InternalMessageRecipientRepo) DeleteNotificationFromInbox(ctx context.Context, req *internalMessageV1.DeleteNotificationFromInboxRequest) error {
	if req.GetUserId() == 0 {
		return internalMessageV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().InternalMessageRecipient.Delete().
		Where(internalmessagerecipient.RecipientUserIDEQ(req.GetUserId()))
	if len(req.GetRecipientIds()) > 0 {
		builder = builder.Where(internalmessagerecipient.IDIn(req.GetRecipientIds()...))
	}
	_, err := builder.Exec(ctx)
	return err
}
