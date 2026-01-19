package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/timeutil"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"
	"go-wind-admin/app/admin/service/internal/data/ent/rolemetadata"

	userV1 "go-wind-admin/api/gen/go/user/service/v1"
)

type RoleMetadataRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper              *mapper.CopierMapper[userV1.RoleMetadata, ent.RoleMetadata]
	syncPolicyConverter *mapper.EnumTypeConverter[userV1.RoleMetadata_SyncPolicy, rolemetadata.SyncPolicy]
	scopeConverter      *mapper.EnumTypeConverter[userV1.RoleMetadata_Scope, rolemetadata.Scope]

	repository *entCrud.Repository[
		ent.RoleMetadataQuery, ent.RoleMetadataSelect,
		ent.RoleMetadataCreate, ent.RoleMetadataCreateBulk,
		ent.RoleMetadataUpdate, ent.RoleMetadataUpdateOne,
		ent.RoleMetadataDelete,
		predicate.RoleMetadata,
		userV1.RoleMetadata, ent.RoleMetadata,
	]
}

func NewRoleMetadataRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
) *RoleMetadataRepo {
	repo := &RoleMetadataRepo{
		log:       ctx.NewLoggerHelper("role-metadata/repo/admin-service"),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[userV1.RoleMetadata, ent.RoleMetadata](),
		syncPolicyConverter: mapper.NewEnumTypeConverter[userV1.RoleMetadata_SyncPolicy, rolemetadata.SyncPolicy](
			userV1.RoleMetadata_SyncPolicy_name, userV1.RoleMetadata_SyncPolicy_value,
		),
		scopeConverter: mapper.NewEnumTypeConverter[userV1.RoleMetadata_Scope, rolemetadata.Scope](
			userV1.RoleMetadata_Scope_name, userV1.RoleMetadata_Scope_value,
		),
	}

	repo.init()

	return repo
}

func (r *RoleMetadataRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.RoleMetadataQuery, ent.RoleMetadataSelect,
		ent.RoleMetadataCreate, ent.RoleMetadataCreateBulk,
		ent.RoleMetadataUpdate, ent.RoleMetadataUpdateOne,
		ent.RoleMetadataDelete,
		predicate.RoleMetadata,
		userV1.RoleMetadata, ent.RoleMetadata,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.syncPolicyConverter.NewConverterPair())
	r.mapper.AppendConverters(r.scopeConverter.NewConverterPair())
}

func (r *RoleMetadataRepo) Create(ctx context.Context, tx *ent.Tx, data *userV1.RoleMetadata) error {
	err := tx.RoleMetadata.Create().
		SetNillableTenantID(data.TenantId).
		SetRoleID(data.GetRoleId()).
		SetNillableIsTemplate(data.IsTemplate).
		SetNillableTemplateFor(data.TemplateFor).
		SetNillableTemplateVersion(data.TemplateVersion).
		SetNillableLastSyncedVersion(data.LastSyncedVersion).
		SetNillableLastSyncedAt(timeutil.TimestamppbToTime(data.LastSyncedAt)).
		SetNillableSyncPolicy(r.syncPolicyConverter.ToEntity(data.SyncPolicy)).
		SetNillableScope(r.scopeConverter.ToEntity(data.Scope)).
		SetCustomOverrides(data.CustomOverrides).
		SetCreatedAt(time.Now()).
		SetNillableCreatedBy(data.CreatedBy).
		Exec(ctx)
	return err
}

// Upsert 插入或更新角色元数据
func (r *RoleMetadataRepo) Upsert(ctx context.Context, data *userV1.RoleMetadata) error {
	now := time.Now()
	builder := r.entClient.Client().RoleMetadata.Create().
		SetNillableTenantID(data.TenantId).
		SetRoleID(data.GetRoleId()).
		SetNillableIsTemplate(data.IsTemplate).
		SetNillableTemplateFor(data.TemplateFor).
		SetNillableTemplateVersion(data.TemplateVersion).
		SetNillableLastSyncedVersion(data.LastSyncedVersion).
		SetNillableLastSyncedAt(timeutil.TimestamppbToTime(data.LastSyncedAt)).
		SetNillableSyncPolicy(r.syncPolicyConverter.ToEntity(data.SyncPolicy)).
		SetNillableScope(r.scopeConverter.ToEntity(data.Scope)).
		SetCustomOverrides(data.CustomOverrides).
		SetCreatedAt(now).
		SetNillableCreatedBy(data.CreatedBy).
		OnConflictColumns(
			rolemetadata.FieldRoleID,
		).
		AddTemplateVersion(1).
		SetUpdatedAt(now).
		SetUpdatedBy(data.GetUpdatedBy())

	if data.LastSyncedAt != nil {
		builder.SetLastSyncedAt(*timeutil.TimestamppbToTime(data.LastSyncedAt))
	}
	if data.SyncPolicy != nil {
		builder.SetSyncPolicy(*r.syncPolicyConverter.ToEntity(data.SyncPolicy))
	}
	if data.Scope != nil {
		builder.SetScope(*r.scopeConverter.ToEntity(data.Scope))
	}
	if data.CustomOverrides != nil {
		builder.SetCustomOverrides(data.GetCustomOverrides())
	}

	err := builder.Exec(ctx)
	return err
}

// Get 获取角色元数据
func (r *RoleMetadataRepo) Get(ctx context.Context, roleID uint32) (*userV1.RoleMetadata, error) {
	rm, err := r.entClient.Client().RoleMetadata.Query().
		Where(
			rolemetadata.RoleIDEQ(roleID),
		).
		Only(ctx)
	if err != nil {
		return nil, err
	}

	dto := r.mapper.ToDTO(rm)
	return dto, nil
}

// IsExistByRoleID 判断角色元数据是否存在
func (r *RoleMetadataRepo) IsExistByRoleID(ctx context.Context, roleID uint32) (bool, error) {
	count, err := r.entClient.Client().RoleMetadata.Query().
		Where(
			rolemetadata.RoleIDEQ(roleID),
		).
		Count(ctx)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// IsTemplateRole 判断角色是否是模版角色
func (r *RoleMetadataRepo) IsTemplateRole(ctx context.Context, roleID uint32) (bool, error) {
	rm, err := r.entClient.Client().RoleMetadata.Query().
		Where(
			rolemetadata.RoleIDEQ(roleID),
		).
		Only(ctx)
	if err != nil {
		return false, err
	}
	var isTemplate bool
	if rm.IsTemplate != nil {
		isTemplate = *rm.IsTemplate
	}
	return isTemplate, nil
}

// UpgradeTemplateVersion 升级模版版本号
func (r *RoleMetadataRepo) UpgradeTemplateVersion(ctx context.Context, tx *ent.Tx, roleID uint32) error {
	err := tx.RoleMetadata.Update().
		Where(
			rolemetadata.RoleIDEQ(roleID),
			rolemetadata.IsTemplateEQ(true),
		).
		AddTemplateVersion(1).
		SetUpdatedAt(time.Now()).
		Exec(ctx)
	if err != nil {
		return err
	}

	return err
}
