package data

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"
	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	"google.golang.org/protobuf/types/known/timestamppb"

	accesskeyV1 "go-wind-admin/api/gen/go/access_key/service/v1"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/accesskey"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"
	appViewer "go-wind-admin/pkg/entgo/viewer"
)

// AccessKeyRepo OpenAPI 访问凭证（AK/SK）仓储。
// 租户隔离由 ent TenantPrivacy 编译策略保证；令牌交换（免鉴权流程）
// 的查询经由 SystemViewerContext 旁路租户 scope（见 service 层）。
type AccessKeyRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *bLogger.Helper
	mapper    *mapper.CopierMapper[accesskeyV1.AccessKey, ent.AccessKey]
	// 状态枚举转换（ent SwitchStatus ON/OFF <-> proto AccessKey_Status）
	statusConverter *mapper.EnumTypeConverter[accesskeyV1.AccessKey_Status, accesskey.Status]

	repository *entCrud.Repository[
		ent.AccessKeyQuery, ent.AccessKeySelect,
		ent.AccessKeyCreate, ent.AccessKeyCreateBulk,
		ent.AccessKeyUpdate, ent.AccessKeyUpdateOne,
		ent.AccessKeyDelete,
		predicate.AccessKey,
		accesskeyV1.AccessKey, ent.AccessKey,
	]
}

func tsOf(t *timestamppb.Timestamp) *time.Time {
	if t == nil || t.AsTime().IsZero() {
		return nil
	}
	u := t.AsTime()
	return &u
}

func NewAccessKeyRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *AccessKeyRepo {
	r := &AccessKeyRepo{
		log:       ctx.NewLoggerHelper("access-key/repo/admin-service"),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[accesskeyV1.AccessKey, ent.AccessKey](),
		statusConverter: mapper.NewEnumTypeConverter[accesskeyV1.AccessKey_Status, accesskey.Status](
			accesskeyV1.AccessKey_Status_name,
			accesskeyV1.AccessKey_Status_value,
		),
	}
	r.init()
	return r
}

func (r *AccessKeyRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.AccessKeyQuery, ent.AccessKeySelect,
		ent.AccessKeyCreate, ent.AccessKeyCreateBulk,
		ent.AccessKeyUpdate, ent.AccessKeyUpdateOne,
		ent.AccessKeyDelete,
		predicate.AccessKey,
		accesskeyV1.AccessKey, ent.AccessKey,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
}

func (r *AccessKeyRepo) Count(ctx context.Context, req *paginationV1.PagingRequest) (*accesskeyV1.CountAccessKeyResponse, error) {
	builder := r.entClient.Client().AccessKey.Query()

	whereSelectors, _, _ := r.repository.BuildListSelectorWithPaging(builder, req)
	if len(whereSelectors) != 0 {
		builder.Modify(whereSelectors...)
	}

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query access key count failed: %s", err.Error())
		return nil, adminV1.ErrorInternalServerError("query access key count failed")
	}

	return &accesskeyV1.CountAccessKeyResponse{
		Count: uint64(count),
	}, nil
}

func (r *AccessKeyRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*accesskeyV1.ListAccessKeyResponse, error) {
	if req == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().AccessKey.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &accesskeyV1.ListAccessKeyResponse{Total: 0, Items: nil}, nil
	}

	return &accesskeyV1.ListAccessKeyResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *AccessKeyRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().AccessKey.Query().
		Where(accesskey.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query access key exist failed: %s", err.Error())
		return false, adminV1.ErrorInternalServerError("query access key exist failed")
	}
	return exist, nil
}

func (r *AccessKeyRepo) Get(ctx context.Context, req *accesskeyV1.GetAccessKeyRequest) (*accesskeyV1.AccessKey, error) {
	if req == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().AccessKey.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *accesskeyV1.GetAccessKeyRequest_Id:
		whereCond = append(whereCond, accesskey.IDEQ(req.GetId()))
	case *accesskeyV1.GetAccessKeyRequest_AccessKey:
		whereCond = append(whereCond, accesskey.AccessKeyEQ(req.GetAccessKey()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

// Create 创建凭证。AK 与 secret 摘要由 service 层生成后传入
// （proto DTO 不携带 secret 摘要，不能走通用 DTO 映射落库）。
// 返回落库后的实体供回显（id/AK/时间戳）。
func (r *AccessKeyRepo) Create(ctx context.Context, req *accesskeyV1.CreateAccessKeyRequest, data *accesskeyV1.AccessKey, accessKey, secretHash string) (*ent.AccessKey, error) {
	builder := r.entClient.Client().AccessKey.Create().
		SetNillableName(data.Name).
		SetAccessKey(accessKey).
		SetSecretHash(secretHash).
		SetNillableExpiresAt(tsOf(data.ExpiresAt)).
		SetCreatedAt(time.Now())

	// 未指定状态时默认启用（与 ent schema 的 ON 默认一致）
	if data.Status == nil || *data.Status == accesskeyV1.AccessKey_ON {
		builder.SetStatus(accesskey.StatusOn)
	} else {
		builder.SetStatus(accesskey.StatusOff)
	}

	if data.TenantId != nil {
		builder.SetTenantID(*data.TenantId)
	}

	entity, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf(ctx, "insert access key failed: %s", err.Error())
		return nil, adminV1.ErrorInternalServerError("insert access key failed")
	}
	return entity, nil
}

func (r *AccessKeyRepo) Update(ctx context.Context, req *accesskeyV1.UpdateAccessKeyRequest) error {
	if req == nil || req.Data == nil {
		return adminV1.ErrorBadRequest("invalid parameter")
	}
	if req.GetId() == 0 {
		return adminV1.ErrorBadRequest("id is required")
	}

	builder := r.entClient.Client().AccessKey.Update()
	err := r.repository.UpdateX(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *accesskeyV1.AccessKey) {
			builder.
				SetNillableName(dto.Name).
				SetNillableUpdatedBy(dto.UpdatedBy).
				SetUpdatedAt(time.Now())

			if dto.Status != nil {
				if stEnt := r.statusConverter.ToEntity(dto.Status); stEnt != nil {
					builder.SetStatus(*stEnt)
				}
			}
			if dto.ExpiresAt != nil {
				builder.SetExpiresAt(dto.ExpiresAt.AsTime())
			}
		},
		func(s *sql.Selector) {
			s.Where(sql.EQ(accesskey.FieldID, req.GetId()))
		},
	)
	if err != nil {
		r.log.Errorf(ctx, "update access key failed: %s", err.Error())
		return adminV1.ErrorInternalServerError("update access key failed")
	}
	return nil
}

func (r *AccessKeyRepo) Delete(ctx context.Context, req *accesskeyV1.DeleteAccessKeyRequest) error {
	if req == nil {
		return adminV1.ErrorBadRequest("invalid parameter")
	}

	_, err := r.entClient.Client().AccessKey.Delete().
		Where(accesskey.IDEQ(req.GetId())).
		Exec(ctx)
	if err != nil {
		r.log.Errorf(ctx, "delete access key failed: %s", err.Error())
		return adminV1.ErrorInternalServerError("delete access key failed")
	}
	return nil
}

// UpdateSecretHash 重置密钥摘要（轮换）。旧的已签发机器令牌在自身过期前仍有效。
func (r *AccessKeyRepo) UpdateSecretHash(ctx context.Context, id uint32, secretHash string) error {
	err := r.entClient.Client().AccessKey.UpdateOneID(id).
		SetSecretHash(secretHash).
		SetUpdatedAt(time.Now()).
		Exec(ctx)
	if err != nil {
		r.log.Errorf(ctx, "update access key secret hash failed: %s", err.Error())
		return adminV1.ErrorInternalServerError("update secret failed")
	}
	return nil
}

// GetByAccessKeyBySystem 按访问键查询凭证（系统旁路视图，绕过租户 scope）——
// 仅供令牌交换（免鉴权流程）使用：交换发生时请求尚无任何 viewer。
func (r *AccessKeyRepo) GetByAccessKeyBySystem(ctx context.Context, accessKey string) (*ent.AccessKey, error) {
	svCtx := appViewer.NewSystemViewerContext(ctx)
	entity, err := r.entClient.Client().AccessKey.Query().
		Where(accesskey.AccessKeyEQ(accessKey)).
		Only(svCtx)
	if err != nil {
		return nil, err
	}
	return entity, nil
}

// TouchLastUsedBySystem 刷新最近使用时间（系统旁路；尽力而为，失败不影响交换）。
func (r *AccessKeyRepo) TouchLastUsedBySystem(ctx context.Context, id uint32) {
	svCtx := appViewer.NewSystemViewerContext(ctx)
	err := r.entClient.Client().AccessKey.UpdateOneID(id).
		SetLastUsedAt(time.Now()).
		Exec(svCtx)
	if err != nil {
		r.log.Warnf(ctx, "touch access key [%d] last_used_at failed: %s", id, err.Error())
	}
}
