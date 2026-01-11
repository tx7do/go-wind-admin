package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/permissionapi"

	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
)

type PermissionApiRepo struct {
	log       *log.Helper
	entClient *entCrud.EntClient[*ent.Client]
}

func NewPermissionApiRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *PermissionApiRepo {
	return &PermissionApiRepo{
		log:       ctx.NewLoggerHelper("permission-api/repo/admin-service"),
		entClient: entClient,
	}
}

// CleanApis 清理权限的所有API资源
func (r *PermissionApiRepo) CleanApis(
	ctx context.Context,
	tx *ent.Tx,
	permissionIDs []uint32,
) error {
	if _, err := tx.PermissionApi.Delete().
		Where(
			permissionapi.PermissionIDIn(permissionIDs...),
		).
		Exec(ctx); err != nil {
		r.log.Errorf("delete old permission apis failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("delete old permission apis failed")
	}
	return nil
}

// CleanNotExistApis 清理权限中不存在的API资源
func (r *PermissionApiRepo) CleanNotExistApis(
	ctx context.Context,
	tx *ent.Tx,
	permissionID uint32,
	apiIDs []uint32,
) error {
	if _, err := tx.PermissionApi.Delete().
		Where(
			permissionapi.APIIDNotIn(apiIDs...),
			permissionapi.PermissionIDEQ(permissionID),
		).
		Exec(ctx); err != nil {
		r.log.Errorf("clean not exists permission apis failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("clean not exists permission apis failed")
	}
	return nil
}

// AssignApis 给权限分配API资源
func (r *PermissionApiRepo) AssignApis(
	ctx context.Context,
	permissionID uint32,
	apiIDs []uint32,
) (err error) {
	var tx *ent.Tx
	tx, err = r.entClient.Client().Tx(ctx)
	if err != nil {
		r.log.Errorf("start transaction failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("start transaction failed")
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				r.log.Errorf("transaction rollback failed: %s", rollbackErr.Error())
			}
			return
		}
		if commitErr := tx.Commit(); commitErr != nil {
			r.log.Errorf("transaction commit failed: %s", commitErr.Error())
			err = permissionV1.ErrorInternalServerError("transaction commit failed")
		}
	}()

	if err = r.CleanNotExistApis(ctx, tx, permissionID, apiIDs); err != nil {

	}

	return r.AssignApisWithTx(ctx, tx, permissionID, apiIDs)
}

// AssignApisWithTx 给权限分配API资源
func (r *PermissionApiRepo) AssignApisWithTx(
	ctx context.Context,
	tx *ent.Tx,
	permissionID uint32,
	apis []uint32,
) error {
	if len(apis) == 0 {
		return nil
	}

	now := time.Now()

	for _, apiID := range apis {
		pm := tx.PermissionApi.
			Create().
			SetPermissionID(permissionID).
			SetAPIID(apiID).
			SetCreatedAt(now).
			OnConflictColumns(
				permissionapi.FieldPermissionID,
				permissionapi.FieldAPIID,
			).
			UpdateNewValues().
			SetUpdatedAt(now)
		if err := pm.Exec(ctx); err != nil {
			r.log.Errorf("assign permission apis failed: %s", err.Error())
			return permissionV1.ErrorInternalServerError("assign permission apis failed")
		}
	}

	return nil
}

// ListApiIDs 列出权限关联的API资源ID列表
func (r *PermissionApiRepo) ListApiIDs(ctx context.Context, permissionIDs []uint32) ([]uint32, error) {
	q := r.entClient.Client().PermissionApi.
		Query().
		Where(
			permissionapi.PermissionIDIn(permissionIDs...),
		)

	intIDs, err := q.
		Select(permissionapi.FieldAPIID).
		Ints(ctx)
	if err != nil {
		r.log.Errorf("list permission apis by permission id failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("list permission apis by permission id failed")
	}

	ids := make([]uint32, len(intIDs))
	for i, v := range intIDs {
		ids[i] = uint32(v)
	}
	return ids, nil
}

// Truncate 清空表数据
func (r *PermissionApiRepo) Truncate(ctx context.Context) error {
	builder := r.entClient.Client().PermissionApi.Delete().
		Where(
			permissionapi.PermissionIDNotIn(1, 2, 3),
		)

	if _, err := builder.Exec(ctx); err != nil {
		r.log.Errorf("failed to truncate permission api table: %s", err.Error())
		return permissionV1.ErrorInternalServerError("truncate failed")
	}

	return nil
}

// Delete 删除权限关联的API资源
func (r *PermissionApiRepo) Delete(ctx context.Context, permissionID uint32) error {
	if _, err := r.entClient.Client().PermissionApi.Delete().
		Where(
			permissionapi.PermissionIDEQ(permissionID),
		).
		Exec(ctx); err != nil {
		r.log.Errorf("delete permission apis by permission id failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("delete permission apis by permission id failed")
	}
	return nil
}

func (r *PermissionApiRepo) DeleteByPermissionIDs(ctx context.Context, permissionIDs []uint32) error {
	if _, err := r.entClient.Client().PermissionApi.Delete().
		Where(
			permissionapi.PermissionIDIn(permissionIDs...),
		).
		Exec(ctx); err != nil {
		r.log.Errorf("delete permission apis by permission ids failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("delete permission apis by permission ids failed")
	}
	return nil
}

// AssignApi 给权限分配API资源
func (r *PermissionApiRepo) AssignApi(ctx context.Context, permissionID uint32, apiID uint32) error {
	now := time.Now()
	pm := r.entClient.Client().PermissionApi.
		Create().
		SetPermissionID(permissionID).
		SetAPIID(apiID).
		SetCreatedAt(now).
		OnConflictColumns(
			permissionapi.FieldPermissionID,
			permissionapi.FieldAPIID,
		).
		UpdateNewValues().
		SetUpdatedAt(now)
	if err := pm.Exec(ctx); err != nil {
		return permissionV1.ErrorInternalServerError("assign permission api failed")
	}

	return nil
}
