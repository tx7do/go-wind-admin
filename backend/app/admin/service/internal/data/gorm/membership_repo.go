//go:build gorm_backend
// +build gorm_backend

package gorm

import (
	"context"
	"errors"
	"time"

	gormDB "gorm.io/gorm"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	gormCrud "github.com/tx7do/go-crud/gorm"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	"go-wind-admin/app/admin/service/internal/data/gorm/models"

	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
)

type MembershipRepo struct {
	client          *gormCrud.Client
	log             *log.Helper
	mapper          *mapper.CopierMapper[identityV1.Membership, models.Membership]
	repository      *gormCrud.Repository[identityV1.Membership, models.Membership]
	statusConverter *mapper.EnumTypeConverter[identityV1.Membership_Status, string]
}

func NewMembershipRepo(
	ctx *bootstrap.Context,
	client *gormCrud.Client,
) *MembershipRepo {
	repo := &MembershipRepo{
		log:             ctx.NewLoggerHelper("membership/gorm-repo/admin-service"),
		client:          client,
		mapper:          mapper.NewCopierMapper[identityV1.Membership, models.Membership](),
		statusConverter: mapper.NewEnumTypeConverter[identityV1.Membership_Status, string](identityV1.Membership_Status_name, identityV1.Membership_Status_value),
	}

	repo.init()

	return repo
}

func (r *MembershipRepo) init() {
	r.repository = gormCrud.NewRepository[identityV1.Membership, models.Membership](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
}

func (r *MembershipRepo) SetUserOrgUnitID(ctx context.Context, userID uint32, orgUnitID uint32) error {
	var val any
	if orgUnitID == 0 {
		val = nil
	} else {
		val = orgUnitID
	}
	if err := r.client.DB.WithContext(ctx).Model(&models.Membership{}).
		Where("user_id = ?", userID).
		Update("org_unit_id", val).Error; err != nil {
		r.log.Errorf("update membership org_unit_id failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("update membership org_unit_id failed")
	}
	return nil
}

func (r *MembershipRepo) SetUserRoleID(ctx context.Context, userID uint32, roleID uint32) error {
	var val any
	if roleID == 0 {
		val = nil
	} else {
		val = roleID
	}
	if err := r.client.DB.WithContext(ctx).Model(&models.Membership{}).
		Where("user_id = ?", userID).
		Update("role_id", val).Error; err != nil {
		r.log.Errorf("update membership role_id failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("update membership role_id failed")
	}
	return nil
}

func (r *MembershipRepo) SetUserPositionID(ctx context.Context, userID uint32, positionID uint32) error {
	var val any
	if positionID == 0 {
		val = nil
	} else {
		val = positionID
	}
	if err := r.client.DB.WithContext(ctx).Model(&models.Membership{}).
		Where("user_id = ?", userID).
		Update("position_id", val).Error; err != nil {
		r.log.Errorf("update membership position_id failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("update membership position_id failed")
	}
	return nil
}

func (r *MembershipRepo) SetUserStatus(ctx context.Context, userID uint32, status *identityV1.Membership_Status) error {
	var val any
	if status == nil {
		val = nil
	} else {
		s := r.statusConverter.ToEntity(status)
		val = *s
	}
	if err := r.client.DB.WithContext(ctx).Model(&models.Membership{}).
		Where("user_id = ?", userID).
		Update("status", val).Error; err != nil {
		r.log.Errorf("update membership status failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("update membership status failed")
	}
	return nil
}

func (r *MembershipRepo) SetUserEndAt(ctx context.Context, userID uint32, endAt *time.Time) error {
	var val any
	if endAt == nil {
		val = nil
	} else {
		val = *endAt
	}
	if err := r.client.DB.WithContext(ctx).Model(&models.Membership{}).
		Where("user_id = ?", userID).
		Update("end_at", val).Error; err != nil {
		r.log.Errorf("update membership end_at failed: %s", err.Error())
		return identityV1.ErrorInternalServerError("update membership end_at failed")
	}
	return nil
}

func (r *MembershipRepo) GetMembershipByUserTenant(ctx context.Context, userID, tenantID uint32) (*identityV1.Membership, error) {
	now := time.Now()
	var entity models.Membership
	if err := r.client.DB.WithContext(ctx).Model(&models.Membership{}).
		Where("user_id = ?", userID).
		Where("tenant_id = ?", tenantID).
		Where("end_at IS NULL OR end_at > ?", now).
		First(&entity).Error; err != nil {
		if errors.Is(err, gormDB.ErrRecordNotFound) {
			return nil, identityV1.ErrorNotFound("membership not found")
		}
		r.log.Errorf("get membership failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("get membership failed")
	}
	dto := r.mapper.ToDTO(&entity)
	return dto, nil
}

func (r *MembershipRepo) GetUserActiveMemberships(ctx context.Context, userID uint32) ([]*identityV1.Membership, error) {
	now := time.Now()
	var entities []*models.Membership
	if err := r.client.DB.WithContext(ctx).Model(&models.Membership{}).
		Where("user_id = ?", userID).
		Where("end_at IS NULL OR end_at > ?", now).
		Find(&entities).Error; err != nil {
		r.log.Errorf("get user active memberships failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("get user active memberships failed")
	}
	dtos := make([]*identityV1.Membership, 0, len(entities))
	for _, entity := range entities {
		dtos = append(dtos, r.mapper.ToDTO(entity))
	}
	return dtos, nil
}

func (r *MembershipRepo) GetMembershipIDByUserID(ctx context.Context, userID uint32) (uint32, error) {
	now := time.Now()
	var entity models.Membership
	if err := r.client.DB.WithContext(ctx).Model(&models.Membership{}).
		Where("user_id = ?", userID).
		Where("end_at IS NULL OR end_at > ?", now).
		Select("id").
		First(&entity).Error; err != nil {
		if errors.Is(err, gormDB.ErrRecordNotFound) {
			return 0, identityV1.ErrorNotFound("membership not found")
		}
		r.log.Errorf("get membership failed: %s", err.Error())
		return 0, identityV1.ErrorInternalServerError("get membership failed")
	}
	return entity.ID, nil
}

func (r *MembershipRepo) ListUserIDs(ctx context.Context, membershipID uint32, excludeExpired bool) ([]uint32, error) {
	q := r.client.DB.WithContext(ctx).Model(&models.Membership{}).
		Where("id = ?", membershipID)
	if excludeExpired {
		now := time.Now()
		q = q.Where("end_at IS NULL OR end_at > ?", now)
	}
	var ids []uint32
	if err := q.Pluck("user_id", &ids).Error; err != nil {
		r.log.Errorf("query user ids by membership id failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("query user ids by membership id failed")
	}
	return ids, nil
}

func (r *MembershipRepo) ListUserIDsByMembershipIDs(ctx context.Context, membershipIDs []uint32, excludeExpired bool) ([]uint32, error) {
	q := r.client.DB.WithContext(ctx).Model(&models.Membership{}).
		Where("id IN ?", membershipIDs)
	if excludeExpired {
		now := time.Now()
		q = q.Where("end_at IS NULL OR end_at > ?", now)
	}
	var ids []uint32
	if err := q.Pluck("user_id", &ids).Error; err != nil {
		r.log.Errorf("query user ids by membership ids failed: %s", err.Error())
		return nil, identityV1.ErrorInternalServerError("query user ids by membership ids failed")
	}
	return ids, nil
}

// === 以下方法为 gorm scaffold 桩：依赖 ent 跨仓储事务/委托/edge，go-crud/gorm 无对应原语 ===

func (r *MembershipRepo) AssignTenantMembershipWith(ctx context.Context, userID uint32, tenantID uint32) error {
	return identityV1.ErrorInternalServerError("gorm scaffold: AssignTenantMembershipWith not implemented — ent *ent.Tx cross-repo transaction has no go-crud/gorm primitive; see data/membership_repo.go")
}

func (r *MembershipRepo) AssignTenantMembershipWithTx(ctx context.Context, tx any, userID uint32, tenantID uint32) error {
	return identityV1.ErrorInternalServerError("gorm scaffold: AssignTenantMembershipWithTx not implemented — ent *ent.Tx cross-repo transaction has no go-crud/gorm primitive; see data/membership_repo.go")
}

func (r *MembershipRepo) AssignMembershipRoles(ctx context.Context, membershipID uint32, datas any) error {
	return identityV1.ErrorInternalServerError("gorm scaffold: AssignMembershipRoles not implemented — ent *ent.Tx cross-repo bridge-table delegation has no go-crud/gorm primitive; see data/membership_repo.go")
}

func (r *MembershipRepo) AssignMembershipPositions(ctx context.Context, membershipID uint32, datas any) error {
	return identityV1.ErrorInternalServerError("gorm scaffold: AssignMembershipPositions not implemented — ent *ent.Tx cross-repo bridge-table delegation has no go-crud/gorm primitive; see data/membership_repo.go")
}

func (r *MembershipRepo) AssignMembershipOrgUnits(ctx context.Context, membershipID uint32, datas any) error {
	return identityV1.ErrorInternalServerError("gorm scaffold: AssignMembershipOrgUnits not implemented — ent *ent.Tx cross-repo bridge-table delegation has no go-crud/gorm primitive; see data/membership_repo.go")
}

func (r *MembershipRepo) GetMembershipID(ctx context.Context, userID uint32, tenantID uint32) (uint32, error) {
	return 0, identityV1.ErrorInternalServerError("gorm scaffold: GetMembershipID not implemented — ent *ent.Tx transaction has no go-crud/gorm primitive; see data/membership_repo.go")
}

func (r *MembershipRepo) ListMembershipRoleIDs(ctx context.Context, membershipID uint32, excludeExpired bool) ([]uint32, error) {
	return nil, identityV1.ErrorInternalServerError("gorm scaffold: ListMembershipRoleIDs not implemented — ent *ent.Tx cross-repo bridge-table delegation has no go-crud/gorm primitive; see data/membership_repo.go")
}

func (r *MembershipRepo) GetRoleIDsByMembership(ctx context.Context, membershipID uint32) ([]uint32, error) {
	return nil, identityV1.ErrorInternalServerError("gorm scaffold: GetRoleIDsByMembership not implemented — ent cross-repo bridge-table delegation has no go-crud/gorm primitive; see data/membership_repo.go")
}

func (r *MembershipRepo) ListMembershipOrgUnitIDs(ctx context.Context, membershipID uint32, excludeExpired bool) ([]uint32, error) {
	return nil, identityV1.ErrorInternalServerError("gorm scaffold: ListMembershipOrgUnitIDs not implemented — ent *ent.Tx cross-repo bridge-table delegation has no go-crud/gorm primitive; see data/membership_repo.go")
}

func (r *MembershipRepo) ListMembershipPositionIDs(ctx context.Context, membershipID uint32, excludeExpired bool) ([]uint32, error) {
	return nil, identityV1.ErrorInternalServerError("gorm scaffold: ListMembershipPositionIDs not implemented — ent *ent.Tx cross-repo bridge-table delegation has no go-crud/gorm primitive; see data/membership_repo.go")
}

func (r *MembershipRepo) ListMembershipRelationIDs(ctx context.Context, membershipID uint32) (roleIDs []uint32, positionIDs []uint32, orgUnitIDs []uint32, err error) {
	return nil, nil, nil, identityV1.ErrorInternalServerError("gorm scaffold: ListMembershipRelationIDs not implemented — ent cross-repo bridge-table delegation has no go-crud/gorm primitive; see data/membership_repo.go")
}

func (r *MembershipRepo) ListUserIDsByOrgUnitID(ctx context.Context, orgUnitID uint32, excludeExpired bool) ([]uint32, error) {
	return nil, identityV1.ErrorInternalServerError("gorm scaffold: ListUserIDsByOrgUnitID not implemented — ent cross-repo bridge-table delegation has no go-crud/gorm primitive; see data/membership_repo.go")
}

func (r *MembershipRepo) ListUserIDsByOrgUnitIDs(ctx context.Context, orgUnitIDs []uint32, excludeExpired bool) ([]uint32, error) {
	return nil, identityV1.ErrorInternalServerError("gorm scaffold: ListUserIDsByOrgUnitIDs not implemented — ent cross-repo bridge-table delegation has no go-crud/gorm primitive; see data/membership_repo.go")
}

func (r *MembershipRepo) ListUserIDsByPositionID(ctx context.Context, positionID uint32, excludeExpired bool) ([]uint32, error) {
	return nil, identityV1.ErrorInternalServerError("gorm scaffold: ListUserIDsByPositionID not implemented — ent cross-repo bridge-table delegation has no go-crud/gorm primitive; see data/membership_repo.go")
}

func (r *MembershipRepo) ListUserIDsByPositionIDs(ctx context.Context, positionIDs []uint32, excludeExpired bool) ([]uint32, error) {
	return nil, identityV1.ErrorInternalServerError("gorm scaffold: ListUserIDsByPositionIDs not implemented — ent cross-repo bridge-table delegation has no go-crud/gorm primitive; see data/membership_repo.go")
}

func (r *MembershipRepo) ListUserIDsByRoleID(ctx context.Context, roleID uint32, excludeExpired bool) ([]uint32, error) {
	return nil, identityV1.ErrorInternalServerError("gorm scaffold: ListUserIDsByRoleID not implemented — ent cross-repo bridge-table delegation has no go-crud/gorm primitive; see data/membership_repo.go")
}

func (r *MembershipRepo) ListUserIDsByRoleIDs(ctx context.Context, roleIDs []uint32, excludeExpired bool) ([]uint32, error) {
	return nil, identityV1.ErrorInternalServerError("gorm scaffold: ListUserIDsByRoleIDs not implemented — ent cross-repo bridge-table delegation has no go-crud/gorm primitive; see data/membership_repo.go")
}

func (r *MembershipRepo) CleanRelationsByUserID(ctx context.Context, tx any, userID uint32) error {
	return identityV1.ErrorInternalServerError("gorm scaffold: CleanRelationsByUserID not implemented — ent *ent.Tx cross-repo bridge-table delegation has no go-crud/gorm primitive; see data/membership_repo.go")
}
