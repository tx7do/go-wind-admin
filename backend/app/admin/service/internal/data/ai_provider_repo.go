package data

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/aiprovider"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"

	aiV1 "go-wind-admin/api/gen/go/ai/service/v1"
)

type AiProviderRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *bLogger.Helper

	mapper             *mapper.CopierMapper[aiV1.AiProvider, ent.AiProvider]
	modelTypeConverter *mapper.EnumTypeConverter[aiV1.AiProvider_ModelType, aiprovider.ModelType]

	repository *entCrud.Repository[
		ent.AiProviderQuery, ent.AiProviderSelect,
		ent.AiProviderCreate, ent.AiProviderCreateBulk,
		ent.AiProviderUpdate, ent.AiProviderUpdateOne,
		ent.AiProviderDelete,
		predicate.AiProvider,
		aiV1.AiProvider, ent.AiProvider,
	]
}

func NewAiProviderRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *AiProviderRepo {
	repo := &AiProviderRepo{
		log:       ctx.NewLoggerHelper("ai_provider/repo/admin-service"),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[aiV1.AiProvider, ent.AiProvider](),
		modelTypeConverter: mapper.NewEnumTypeConverter[aiV1.AiProvider_ModelType, aiprovider.ModelType](
			aiV1.AiProvider_ModelType_name, aiV1.AiProvider_ModelType_value,
		),
	}

	repo.init()

	return repo
}

func (r *AiProviderRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.AiProviderQuery, ent.AiProviderSelect,
		ent.AiProviderCreate, ent.AiProviderCreateBulk,
		ent.AiProviderUpdate, ent.AiProviderUpdateOne,
		ent.AiProviderDelete,
		predicate.AiProvider,
		aiV1.AiProvider, ent.AiProvider,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
	r.mapper.AppendConverters(r.modelTypeConverter.NewConverterPair())
}

func (r *AiProviderRepo) Count(ctx context.Context, req *paginationV1.PagingRequest) (*aiV1.CountAiProviderResponse, error) {
	builder := r.entClient.Client().AiProvider.Query()

	whereSelectors, _, _ := r.repository.BuildListSelectorWithPaging(builder, req)
	if len(whereSelectors) != 0 {
		builder.Modify(whereSelectors...)
	}

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query ai provider count failed: %s", err.Error())
		return nil, aiV1.ErrorInternalServerError("query ai provider count failed")
	}

	return &aiV1.CountAiProviderResponse{Count: uint64(count)}, nil
}

func (r *AiProviderRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*aiV1.ListAiProviderResponse, error) {
	if req == nil {
		return nil, aiV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().AiProvider.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &aiV1.ListAiProviderResponse{Total: 0, Items: nil}, nil
	}

	// 密文不出 repo：DTO 的 api_key 恒为空（写入走 service 层加密后另行传递），
	// 脱敏提示由 api_key_hint 承载。
	for _, item := range ret.Items {
		item.ApiKey = nil
	}

	return &aiV1.ListAiProviderResponse{Total: ret.Total, Items: ret.Items}, nil
}

func (r *AiProviderRepo) Get(ctx context.Context, req *aiV1.GetAiProviderRequest) (*aiV1.AiProvider, error) {
	if req == nil {
		return nil, aiV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().AiProvider.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *aiV1.GetAiProviderRequest_Id:
		whereCond = append(whereCond, aiprovider.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	dto.ApiKey = nil

	return dto, nil
}

// GetEnabledDefault 取默认可用的提供商：优先 is_default 且启用的，否则取任一启用的。
func (r *AiProviderRepo) GetEnabledDefault(ctx context.Context) (*ent.AiProvider, error) {
	entity, err := r.entClient.Client().AiProvider.Query().
		Where(aiprovider.IsEnabledEQ(true), aiprovider.IsDefaultEQ(true)).
		First(ctx)
	if err == nil {
		return entity, nil
	}
	if !ent.IsNotFound(err) {
		r.log.Errorf(ctx, "query default ai provider failed: %s", err.Error())
		return nil, aiV1.ErrorInternalServerError("query default ai provider failed")
	}

	entity, err = r.entClient.Client().AiProvider.Query().
		Where(aiprovider.IsEnabledEQ(true)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, aiV1.ErrorNotFound("no enabled ai provider")
		}
		r.log.Errorf(ctx, "query any enabled ai provider failed: %s", err.Error())
		return nil, aiV1.ErrorInternalServerError("query ai provider failed")
	}
	return entity, nil
}

// GetApiKeyDecrypted 返回 provider 的 api_key 密文（解密在 service 层做，repo 不管密钥材料）。
func (r *AiProviderRepo) GetApiKeyEncrypted(ctx context.Context, id uint64) (string, error) {
	entity, err := r.entClient.Client().AiProvider.Query().
		Where(aiprovider.IDEQ(id)).
		Select(aiprovider.FieldAPIKey).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return "", aiV1.ErrorNotFound("ai provider not found")
		}
		r.log.Errorf(ctx, "query ai provider api key failed: %s", err.Error())
		return "", aiV1.ErrorInternalServerError("query ai provider failed")
	}
	if entity.APIKey == nil {
		return "", nil
	}
	return *entity.APIKey, nil
}

func (r *AiProviderRepo) Create(ctx context.Context, req *aiV1.CreateAiProviderRequest) error {
	if req == nil || req.Data == nil {
		return aiV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.newAiProviderCreate(req.Data)

	if err := builder.Exec(ctx); err != nil {
		r.log.Errorf(ctx, "insert ai provider failed: %s", err.Error())
		return aiV1.ErrorInternalServerError("insert ai provider failed")
	}

	return nil
}

func (r *AiProviderRepo) newAiProviderCreate(data *aiV1.AiProvider) *ent.AiProviderCreate {
	builder := r.entClient.Client().AiProvider.Create().
		SetNillableName(data.Name).
		SetNillableModelType(r.modelTypeConverter.ToEntity(data.ModelType)).
		SetNillableModelName(data.ModelName).
		SetNillableBaseURL(data.BaseUrl).
		SetNillableOrganization(data.Organization).
		SetNillableAPIKey(data.ApiKey).
		SetNillableAPIKeyHint(data.ApiKeyHint).
		SetNillableLocalHost(data.LocalHost).
		SetNillableLocalPort(int32PtrToIntPtr(data.LocalPort)).
		SetNillableTimeoutSeconds(int32PtrToIntPtr(data.TimeoutSeconds)).
		SetNillableSystemPrompt(data.SystemPrompt).
		SetNillableIsDefault(data.IsDefault).
		SetNillableRemark(data.Remark).
		SetNillableIsEnabled(data.IsEnabled).
		SetNillableCreatedBy(data.CreatedBy).
		SetCreatedAt(time.Now())

	if data.Id != nil {
		builder.SetID(data.GetId())
	}

	return builder
}

func (r *AiProviderRepo) Update(ctx context.Context, req *aiV1.UpdateAiProviderRequest) error {
	if req == nil || req.Data == nil {
		return aiV1.ErrorBadRequest("invalid parameter")
	}
	if req.GetId() == 0 {
		return aiV1.ErrorBadRequest("id is required")
	}

	// 如果不存在则创建
	if req.GetAllowMissing() {
		exist, err := r.entClient.Client().AiProvider.Query().
			Where(aiprovider.IDEQ(req.GetId())).
			Exist(ctx)
		if err != nil {
			r.log.Errorf(ctx, "query ai provider exist failed: %s", err.Error())
			return aiV1.ErrorInternalServerError("query exist failed")
		}
		if !exist {
			data := req.Data
			data.Id = req.Data.Id
			return r.Create(ctx, &aiV1.CreateAiProviderRequest{Data: data})
		}
	}

	builder := r.entClient.Client().AiProvider.Update()

	if err := r.repository.UpdateX(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *aiV1.AiProvider) {
			builder.
				SetNillableName(req.Data.Name).
				SetNillableModelType(r.modelTypeConverter.ToEntity(req.Data.ModelType)).
				SetNillableModelName(req.Data.ModelName).
				SetNillableBaseURL(req.Data.BaseUrl).
				SetNillableOrganization(req.Data.Organization).
				SetNillableAPIKey(req.Data.ApiKey).
				SetNillableAPIKeyHint(req.Data.ApiKeyHint).
				SetNillableLocalHost(req.Data.LocalHost).
				SetNillableLocalPort(int32PtrToIntPtr(req.Data.LocalPort)).
				SetNillableTimeoutSeconds(int32PtrToIntPtr(req.Data.TimeoutSeconds)).
				SetNillableSystemPrompt(req.Data.SystemPrompt).
				SetNillableIsDefault(req.Data.IsDefault).
				SetNillableRemark(req.Data.Remark).
				SetNillableIsEnabled(req.Data.IsEnabled).
				SetNillableUpdatedBy(req.Data.UpdatedBy).
				SetUpdatedAt(time.Now())
		},
		func(s *sql.Selector) {
			s.Where(sql.EQ(aiprovider.FieldID, req.GetId()))
		},
	); err != nil {
		r.log.Errorf(ctx, "update ai provider failed: %s", err.Error())
		return err
	}

	return nil
}

func (r *AiProviderRepo) Delete(ctx context.Context, req *aiV1.DeleteAiProviderRequest) error {
	if req == nil {
		return aiV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().AiProvider.Delete()

	var predicates []predicate.AiProvider
	switch req.QueryBy.(type) {
	default:
	case *aiV1.DeleteAiProviderRequest_Id:
		predicates = append(predicates, aiprovider.IDEQ(req.GetId()))
	}

	if _, err := r.repository.Delete(ctx, builder, predicates...); err != nil {
		r.log.Errorf(ctx, "delete ai provider failed: %s", err.Error())
		return err
	}

	return nil
}

// int32PtrToIntPtr proto int32（可选）→ ent field.Int（可选）。
func int32PtrToIntPtr(p *int32) *int {
	if p == nil {
		return nil
	}
	v := int(*p)
	return &v
}

// GetEntityByID 取提供商实体（chat 链路解析与归属校验用）。
func (r *AiProviderRepo) GetEntityByID(ctx context.Context, id uint64) (*ent.AiProvider, error) {
	entity, err := r.entClient.Client().AiProvider.Query().
		Where(aiprovider.IDEQ(id)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, aiV1.ErrorNotFound("ai provider not found")
		}
		r.log.Errorf(ctx, "query ai provider failed: %s", err.Error())
		return nil, aiV1.ErrorInternalServerError("query ai provider failed")
	}
	return entity, nil
}
