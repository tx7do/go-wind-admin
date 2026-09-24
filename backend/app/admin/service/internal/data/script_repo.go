package data

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	scriptV1 "go-wind-admin/api/gen/go/script/service/v1"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"
	"go-wind-admin/app/admin/service/internal/data/ent/script"
)

type ScriptRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *bLogger.Helper

	mapper            *mapper.CopierMapper[scriptV1.Script, ent.Script]
	languageConverter *mapper.EnumTypeConverter[scriptV1.Language, script.Language]

	repository *entCrud.Repository[
		ent.ScriptQuery, ent.ScriptSelect,
		ent.ScriptCreate, ent.ScriptCreateBulk,
		ent.ScriptUpdate, ent.ScriptUpdateOne,
		ent.ScriptDelete,
		predicate.Script,
		scriptV1.Script, ent.Script,
	]
}

func NewScriptRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *ScriptRepo {
	repo := &ScriptRepo{
		log:       ctx.NewLoggerHelper("script/repo/admin-service"),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[scriptV1.Script, ent.Script](),
		languageConverter: mapper.NewEnumTypeConverter[scriptV1.Language, script.Language](
			scriptV1.Language_name, scriptV1.Language_value,
		),
	}

	repo.init()

	return repo
}

func (r *ScriptRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.ScriptQuery, ent.ScriptSelect,
		ent.ScriptCreate, ent.ScriptCreateBulk,
		ent.ScriptUpdate, ent.ScriptUpdateOne,
		ent.ScriptDelete,
		predicate.Script,
		scriptV1.Script, ent.Script,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.languageConverter.NewConverterPair())
}

func (r *ScriptRepo) Count(ctx context.Context, req *paginationV1.PagingRequest) (*scriptV1.CountScriptsResponse, error) {
	builder := r.entClient.Client().Script.Query()

	whereSelectors, _, _ := r.repository.BuildListSelectorWithPaging(builder, req)
	if len(whereSelectors) != 0 {
		builder.Modify(whereSelectors...)
	}

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query script count failed: %s", err.Error())
		return nil, scriptV1.ErrorInternalServerError("query script count failed")
	}

	return &scriptV1.CountScriptsResponse{
		Count: uint64(count),
	}, nil
}

func (r *ScriptRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*scriptV1.ListScriptsResponse, error) {
	if req == nil {
		return nil, scriptV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Script.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &scriptV1.ListScriptsResponse{Total: 0, Items: nil}, nil
	}

	return &scriptV1.ListScriptsResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *ScriptRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().Script.Query().
		Where(script.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query script exist failed: %s", err.Error())
		return false, scriptV1.ErrorInternalServerError("query script exist failed")
	}
	return exist, nil
}

func (r *ScriptRepo) IsNameExist(ctx context.Context, name string, excludeID uint32) (bool, error) {
	query := r.entClient.Client().Script.Query().Where(script.NameEQ(name))
	if excludeID > 0 {
		query = query.Where(script.IDNEQ(excludeID))
	}
	exist, err := query.Exist(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query script name exist failed: %s", err.Error())
		return false, scriptV1.ErrorInternalServerError("query script name exist failed")
	}
	return exist, nil
}

func (r *ScriptRepo) Get(ctx context.Context, req *scriptV1.GetScriptRequest) (*scriptV1.Script, error) {
	if req == nil {
		return nil, scriptV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Script.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	case *scriptV1.GetScriptRequest_Name:
		whereCond = append(whereCond, script.NameEQ(req.GetName()))
	default:
		whereCond = append(whereCond, script.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, nil
}

// GetByNameByName 按唯一名取脚本（内部用，返回 DTO）。
func (r *ScriptRepo) GetByName(ctx context.Context, name string) (*scriptV1.Script, error) {
	entity, err := r.entClient.Client().Script.Query().
		Where(script.NameEQ(name)).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return r.mapper.ToDTO(entity), nil
}

func (r *ScriptRepo) Create(ctx context.Context, req *scriptV1.CreateScriptRequest) error {
	if req == nil || req.Data == nil {
		return scriptV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.newScriptCreate(req.Data)

	if err := builder.Exec(ctx); err != nil {
		r.log.Errorf(ctx, "insert script failed: %s", err.Error())
		return scriptV1.ErrorInternalServerError("insert script failed")
	}

	return nil
}

func (r *ScriptRepo) newScriptCreate(dto *scriptV1.Script) *ent.ScriptCreate {
	builder := r.entClient.Client().Script.Create().
		SetNillableName(dto.Name).
		SetNillableHookPoint(dto.HookPoint).
		SetNillablePriority(dto.Priority).
		SetNillableDescription(dto.Description).
		SetNillableCritical(dto.Critical).
		SetNillableVersion(dto.Version).
		SetNillableIsEnabled(dto.IsEnabled).
		SetNillableCreatedBy(dto.CreatedBy).
		SetCreatedAt(time.Now())

	// language：nil 留空走 ent 默认 LUA
	if dto.Language != nil {
		builder.SetNillableLanguage(r.languageConverter.ToEntity(dto.Language))
	}

	if dto.Source != nil {
		builder.SetSource(dto.GetSource())
	}

	if dto.Id != nil {
		builder.SetID(dto.GetId())
	}

	return builder
}

func (r *ScriptRepo) Update(ctx context.Context, req *scriptV1.UpdateScriptRequest) error {
	if req == nil || req.Data == nil {
		return scriptV1.ErrorBadRequest("invalid parameter")
	}
	if req.GetId() == 0 {
		return scriptV1.ErrorBadRequest("id is required")
	}

	// 如果不存在则创建
	if req.GetAllowMissing() {
		exist, err := r.IsExist(ctx, req.GetId())
		if err != nil {
			return err
		}
		if !exist {
			createReq := &scriptV1.CreateScriptRequest{Data: req.Data}
			createReq.Data.CreatedBy = createReq.Data.UpdatedBy
			createReq.Data.UpdatedBy = nil
			return r.Create(ctx, createReq)
		}
	}

	builder := r.entClient.Client().Script.Update()
	err := r.repository.UpdateX(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *scriptV1.Script) {
			builder.
				SetNillableName(dto.Name).
				SetNillableHookPoint(dto.HookPoint).
				SetNillablePriority(dto.Priority).
				SetNillableDescription(dto.Description).
				SetNillableCritical(dto.Critical).
				SetNillableIsEnabled(dto.IsEnabled).
				SetNillableUpdatedBy(dto.UpdatedBy).
				SetUpdatedAt(time.Now())

			// version 每次更新自增（热更新指纹）
			builder.AddVersion(1)

			if dto.Language != nil {
				builder.SetNillableLanguage(r.languageConverter.ToEntity(dto.Language))
			}

			if dto.Source != nil {
				builder.SetSource(dto.GetSource())
			}
		},
		func(s *sql.Selector) {
			s.Where(sql.EQ(script.FieldID, req.GetId()))
		},
	)
	if err != nil {
		r.log.Errorf(ctx, "update script failed: %s", err.Error())
		return scriptV1.ErrorInternalServerError("update script failed")
	}

	return nil
}

func (r *ScriptRepo) Delete(ctx context.Context, req *scriptV1.DeleteScriptRequest) error {
	if req == nil {
		return scriptV1.ErrorBadRequest("invalid parameter")
	}
	if len(req.GetIds()) == 0 {
		return scriptV1.ErrorBadRequest("ids is required")
	}

	if _, err := r.entClient.Client().Script.Delete().
		Where(script.IDIn(req.GetIds()...)).
		Exec(ctx); err != nil {
		r.log.Errorf(ctx, "delete script failed: %s", err.Error())
		return scriptV1.ErrorInternalServerError("delete script failed")
	}

	return nil
}

// ListEnabledScripts 返回全部已启用脚本（脚本运行时启动加载/全量重同步用）。
func (r *ScriptRepo) ListEnabledScripts(ctx context.Context) ([]*scriptV1.Script, error) {
	entities, err := r.entClient.Client().Script.Query().
		Where(script.IsEnabledEQ(true)).
		All(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query enabled scripts failed: %s", err.Error())
		return nil, scriptV1.ErrorInternalServerError("query enabled scripts failed")
	}

	dtos := make([]*scriptV1.Script, 0, len(entities))
	for _, entity := range entities {
		dtos = append(dtos, r.mapper.ToDTO(entity))
	}
	return dtos, nil
}

// GetSourceByName 按唯一名取脚本源码（DBSource 加载回调）。
func (r *ScriptRepo) GetSourceByName(ctx context.Context, name string) (string, error) {
	entity, err := r.entClient.Client().Script.Query().
		Where(script.NameEQ(name)).
		Only(ctx)
	if err != nil {
		return "", err
	}
	if entity.Source == nil {
		return "", nil
	}
	return *entity.Source, nil
}

// GetVersionByName 按唯一名返回脚本版本指纹（DBSource 热更新变更检测回调）。
func (r *ScriptRepo) GetVersionByName(ctx context.Context, name string) (string, error) {
	entity, err := r.entClient.Client().Script.Query().
		Where(script.NameEQ(name)).
		Only(ctx)
	if err != nil {
		return "", err
	}
	// Version 为 nillable 列（*uint32）：此前 fmt.Sprintf("%d", entity.Version)
	// 格式化的是指针地址而非版本号，DBSource 热更新的变更检测指纹因此失效。
	if entity.Version == nil {
		return "", nil
	}
	return fmt.Sprintf("%d", *entity.Version), nil
}
