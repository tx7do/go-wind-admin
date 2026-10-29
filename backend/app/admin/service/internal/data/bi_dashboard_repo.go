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

	biV1 "go-wind-admin/api/gen/go/bi/service/v1"
	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/bidashboard"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"
)

type BiDashboardRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *bLogger.Helper

	mapper        *mapper.CopierMapper[biV1.BiDashboard, ent.BiDashboard]
	kindConverter *mapper.EnumTypeConverter[biV1.BiDashboardKind, bidashboard.Kind]

	repository *entCrud.Repository[
		ent.BiDashboardQuery, ent.BiDashboardSelect,
		ent.BiDashboardCreate, ent.BiDashboardCreateBulk,
		ent.BiDashboardUpdate, ent.BiDashboardUpdateOne,
		ent.BiDashboardDelete,
		predicate.BiDashboard,
		biV1.BiDashboard, ent.BiDashboard,
	]
}

func NewBiDashboardRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *BiDashboardRepo {
	repo := &BiDashboardRepo{
		log:       ctx.NewLoggerHelper("bi-dashboard/repo/admin-service"),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[biV1.BiDashboard, ent.BiDashboard](),
		kindConverter: mapper.NewEnumTypeConverter[biV1.BiDashboardKind, bidashboard.Kind](
			biV1.BiDashboardKind_name, biV1.BiDashboardKind_value,
		),
	}

	repo.init()

	return repo
}

func (r *BiDashboardRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.BiDashboardQuery, ent.BiDashboardSelect,
		ent.BiDashboardCreate, ent.BiDashboardCreateBulk,
		ent.BiDashboardUpdate, ent.BiDashboardUpdateOne,
		ent.BiDashboardDelete,
		predicate.BiDashboard,
		biV1.BiDashboard, ent.BiDashboard,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
	r.mapper.AppendConverters(r.kindConverter.NewConverterPair())
}

func (r *BiDashboardRepo) Count(ctx context.Context, req *paginationV1.PagingRequest) (uint64, error) {
	builder := r.entClient.Client().BiDashboard.Query()

	whereSelectors, _, _ := r.repository.BuildListSelectorWithPaging(builder, req)
	if len(whereSelectors) != 0 {
		builder.Modify(whereSelectors...)
	}

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query bi dashboard count failed: %s", err.Error())
		return 0, adminV1.ErrorInternalServerError("query bi dashboard count failed")
	}

	return uint64(count), nil
}

func (r *BiDashboardRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*biV1.ListBiDashboardResponse, error) {
	if req == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().BiDashboard.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &biV1.ListBiDashboardResponse{Total: 0, Items: nil}, nil
	}

	return &biV1.ListBiDashboardResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *BiDashboardRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().BiDashboard.Query().
		Where(bidashboard.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query exist failed: %s", err.Error())
		return false, adminV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *BiDashboardRepo) Get(ctx context.Context, req *biV1.GetBiDashboardRequest) (*biV1.BiDashboard, error) {
	if req == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().BiDashboard.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *biV1.GetBiDashboardRequest_Id:
		whereCond = append(whereCond, bidashboard.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

func (r *BiDashboardRepo) Create(ctx context.Context, req *biV1.CreateBiDashboardRequest) error {
	if req == nil || req.Data == nil {
		return adminV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.newBiDashboardCreate(req.Data)

	if err := builder.Exec(ctx); err != nil {
		r.log.Errorf(ctx, "insert bi dashboard failed: %s", err.Error())
		if ent.IsConstraintError(err) {
			return adminV1.ErrorBadRequest("bi dashboard code already exists")
		}
		return adminV1.ErrorInternalServerError("insert bi dashboard failed")
	}

	return nil
}

func (r *BiDashboardRepo) newBiDashboardCreate(dto *biV1.BiDashboard) *ent.BiDashboardCreate {
	builder := r.entClient.Client().BiDashboard.Create().
		SetNillableName(dto.Name).
		SetNillableCode(dto.Code).
		SetNillableRemark(dto.Remark).
		SetNillablePayload(dto.Payload).
		SetNillableChartCount(dto.ChartCount).
		SetNillableThemeKey(dto.ThemeKey).
		SetNillableThumbnail(dto.Thumbnail).
		SetNillableIsBuiltIn(dto.IsBuiltIn).
		SetNillableIsPublic(dto.IsPublic).
		SetNillableIsEnabled(dto.IsEnabled).
		SetNillableSortOrder(dto.SortOrder).
		SetNillableCreatedBy(dto.CreatedBy).
		SetCreatedAt(time.Now())

	if dto.Kind != nil {
		builder.SetKind(bidashboard.Kind(dto.GetKind().String()))
	}
	if dto.Version != nil {
		builder.SetVersion(dto.GetVersion())
	}
	if dto.Id != nil {
		builder.SetID(dto.GetId())
	}

	return builder
}

func (r *BiDashboardRepo) Update(ctx context.Context, req *biV1.UpdateBiDashboardRequest) error {
	if req == nil || req.Data == nil {
		return adminV1.ErrorBadRequest("invalid parameter")
	}
	if req.GetId() == 0 {
		return adminV1.ErrorBadRequest("id is required")
	}

	// 如果不存在则创建
	if req.GetAllowMissing() {
		exist, err := r.IsExist(ctx, req.GetId())
		if err != nil {
			return err
		}
		if !exist {
			createReq := &biV1.CreateBiDashboardRequest{Data: req.Data}
			createReq.Data.CreatedBy = createReq.Data.UpdatedBy
			createReq.Data.UpdatedBy = nil
			return r.Create(ctx, createReq)
		}
	}

	// 乐观锁：调用方在 data.version 回传了读到的版本时按 CAS 更新。
	// 设计器是整份覆盖保存，多标签/多人同时编辑靠它挡住 stale 写；
	// 未传 version 的调用（列表页轻量改字段）走普通更新，但版本号照样 +1，
	// 让持有旧版本的编辑器下一次保存时能发现冲突。
	if req.Data.Version != nil {
		return r.updateWithVersionCAS(ctx, req)
	}

	return r.updatePlain(ctx, req)
}

// updateWithVersionCAS 带版本校验的更新：命中 (id, version) 才写入并把版本号 +1。
// 只应用调用方显式给出的字段（SetNillable），未给字段保持原值。
func (r *BiDashboardRepo) updateWithVersionCAS(ctx context.Context, req *biV1.UpdateBiDashboardRequest) error {
	builder := r.entClient.Client().BiDashboard.Update().
		Where(
			bidashboard.IDEQ(req.GetId()),
			bidashboard.VersionEQ(req.Data.GetVersion()),
		).
		SetNillableName(req.Data.Name).
		SetNillableCode(req.Data.Code).
		SetNillableRemark(req.Data.Remark).
		SetNillablePayload(req.Data.Payload).
		SetNillableChartCount(req.Data.ChartCount).
		SetNillableThemeKey(req.Data.ThemeKey).
		SetNillableThumbnail(req.Data.Thumbnail).
		SetNillableIsBuiltIn(req.Data.IsBuiltIn).
		SetNillableIsPublic(req.Data.IsPublic).
		SetNillableIsEnabled(req.Data.IsEnabled).
		SetNillableSortOrder(req.Data.SortOrder).
		SetNillableUpdatedBy(req.Data.UpdatedBy).
		AddVersion(1).
		SetUpdatedAt(time.Now())

	if req.Data.Kind != nil {
		builder.SetKind(bidashboard.Kind(req.Data.GetKind().String()))
	}

	affected, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf(ctx, "update bi dashboard (cas) failed: %s", err.Error())
		if ent.IsConstraintError(err) {
			return adminV1.ErrorBadRequest("bi dashboard code already exists")
		}
		return adminV1.ErrorInternalServerError("update bi dashboard failed")
	}
	if affected == 0 {
		// 两种可能：行已被删 / 版本号过期（他人已保存）。分别给出可区分的提示。
		exist, existErr := r.IsExist(ctx, req.GetId())
		if existErr != nil {
			return existErr
		}
		if !exist {
			return adminV1.ErrorNotFound("bi dashboard not found")
		}
		return adminV1.ErrorConflict("bi dashboard has been modified by another session, reload and retry")
	}

	return nil
}

func (r *BiDashboardRepo) updatePlain(ctx context.Context, req *biV1.UpdateBiDashboardRequest) error {
	builder := r.entClient.Client().BiDashboard.Update()
	err := r.repository.UpdateX(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *biV1.BiDashboard) {
			builder.
				SetNillableName(req.Data.Name).
				SetNillableCode(req.Data.Code).
				SetNillableRemark(req.Data.Remark).
				SetNillablePayload(req.Data.Payload).
				SetNillableChartCount(req.Data.ChartCount).
				SetNillableThemeKey(req.Data.ThemeKey).
				SetNillableThumbnail(req.Data.Thumbnail).
				SetNillableIsBuiltIn(req.Data.IsBuiltIn).
				SetNillableIsPublic(req.Data.IsPublic).
				SetNillableIsEnabled(req.Data.IsEnabled).
				SetNillableSortOrder(req.Data.SortOrder).
				SetNillableUpdatedBy(req.Data.UpdatedBy).
				AddVersion(1).
				SetUpdatedAt(time.Now())

			if req.Data.Kind != nil {
				builder.SetKind(bidashboard.Kind(req.Data.GetKind().String()))
			}
		},
		func(s *sql.Selector) {
			s.Where(sql.EQ(bidashboard.FieldID, req.GetId()))
		},
	)
	if err != nil {
		r.log.Errorf(ctx, "update bi dashboard failed: %s", err.Error())
		if ent.IsConstraintError(err) {
			return adminV1.ErrorBadRequest("bi dashboard code already exists")
		}
		return adminV1.ErrorInternalServerError("update bi dashboard failed")
	}

	return nil
}

func (r *BiDashboardRepo) Delete(ctx context.Context, req *biV1.DeleteBiDashboardRequest) error {
	if req == nil {
		return adminV1.ErrorBadRequest("invalid parameter")
	}

	// 删除前先取行：内置仪表盘禁删守卫
	entity, err := r.entClient.Client().BiDashboard.Query().
		Where(bidashboard.IDEQ(req.GetId())).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			// 幂等删除：目标不存在视为已删除
			return nil
		}
		r.log.Errorf(ctx, "query bi dashboard before delete failed: %s", err.Error())
		return adminV1.ErrorInternalServerError("query bi dashboard failed")
	}
	if entity.IsBuiltIn != nil && *entity.IsBuiltIn {
		return adminV1.ErrorBadRequest("built-in bi dashboard cannot be deleted")
	}

	builder := r.entClient.Client().BiDashboard.Delete()

	_, err = r.repository.Delete(ctx, builder, func(s *sql.Selector) {
		s.Where(sql.EQ(bidashboard.FieldID, req.GetId()))
	})
	if err != nil {
		r.log.Errorf(ctx, "delete bi dashboard failed: %s", err.Error())
		return adminV1.ErrorInternalServerError("delete bi dashboard failed")
	}

	return nil
}

// SeedDefaults 按编码缺一补一地播种内置仪表盘（启动期）：
// 编码不存在的按给定默认值创建，已存在的跳过不覆盖。
// 与 sys_configs / print_templates 的 SeedDefaults 同款语义。
// 内置行落平台上下文（tid=0）；租户侧使用走"复制内置模板派生本租户行"。
func (r *BiDashboardRepo) SeedDefaults(ctx context.Context, defaults []*biV1.BiDashboard) error {
	for _, item := range defaults {
		if item == nil || item.GetCode() == "" {
			continue
		}
		exists, err := r.entClient.Client().BiDashboard.Query().
			Where(bidashboard.CodeEQ(item.GetCode())).
			Exist(ctx)
		if err != nil {
			r.log.Errorf(ctx, "seed bi dashboard %q: exists query failed: %s", item.GetCode(), err.Error())
			return err
		}
		if exists {
			continue
		}
		if err := r.newBiDashboardCreate(item).Exec(ctx); err != nil {
			r.log.Errorf(ctx, "seed bi dashboard %q: insert failed: %s", item.GetCode(), err.Error())
			return err
		}
	}
	return nil
}
