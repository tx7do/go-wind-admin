//go:build gorm_backend
// +build gorm_backend

package gorm

import (
	"context"
	"time"

	gormDB "gorm.io/gorm"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	gormCrud "github.com/tx7do/go-crud/gorm"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	"go-wind-admin/app/admin/service/internal/data/gorm/models"

	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
)

// Endpoint API 端点身份（path + method）——授权绑定的执行键。
// data 包同构类型的 gorm 镜像：绑定存快照后，端点删除重建（sys_apis id 漂移）
// 不再影响授权有效性。两栈语义必须保持一致，改这里须同步 data/permission_api_repo.go。
type Endpoint struct {
	Path   string
	Method string
}

func (e Endpoint) key() string {
	return e.Path + "\x00" + e.Method
}

type PermissionApiRepo struct {
	client     *gormCrud.Client
	log        *bLogger.Helper
	mapper     *mapper.CopierMapper[permissionV1.PermissionApi, models.PermissionApi]
	repository *gormCrud.Repository[permissionV1.PermissionApi, models.PermissionApi]
}

func NewPermissionApiRepo(
	ctx *bootstrap.Context,
	client *gormCrud.Client,
) *PermissionApiRepo {
	repo := &PermissionApiRepo{
		log:    ctx.NewLoggerHelper("permission-api/gorm-repo/admin-service"),
		client: client,
		mapper: mapper.NewCopierMapper[permissionV1.PermissionApi, models.PermissionApi](),
	}

	repo.init()

	return repo
}

func (r *PermissionApiRepo) init() {
	r.repository = gormCrud.NewRepository[permissionV1.PermissionApi, models.PermissionApi](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
}

// resolveEndpoints 把 api 引用（id 列表）解析成端点身份快照；解析不到随 missing
// 返回由调用方整体失败（不吞错铁律）。与 data 栈同名方法语义一致。
func (r *PermissionApiRepo) resolveEndpoints(ctx context.Context, apiIDs []uint32) (endpoints []Endpoint, missing []uint32, err error) {
	if len(apiIDs) == 0 {
		return nil, nil, nil
	}

	var apis []*models.Api
	if err = r.client.DB.WithContext(ctx).Model(&models.Api{}).Where("id IN ?", apiIDs).Find(&apis).Error; err != nil {
		r.log.Errorf(ctx, "query apis for endpoint resolve failed: %s", err.Error())
		return nil, nil, permissionV1.ErrorInternalServerError("query apis failed")
	}

	byID := make(map[uint32]Endpoint, len(apis))
	for _, a := range apis {
		if a == nil || a.Path == nil || a.Method == nil || *a.Path == "" || *a.Method == "" {
			continue
		}
		byID[a.ID] = Endpoint{Path: *a.Path, Method: *a.Method}
	}

	seen := make(map[string]struct{}, len(apiIDs))
	for _, id := range apiIDs {
		e, ok := byID[id]
		if !ok {
			missing = append(missing, id)
			continue
		}
		if _, dup := seen[e.key()]; dup {
			continue
		}
		seen[e.key()] = struct{}{}
		endpoints = append(endpoints, e)
	}

	return endpoints, missing, nil
}

// ListEndpoints 列出权限关联的端点快照（绑定表即真相，不回查 sys_apis）。
func (r *PermissionApiRepo) ListEndpoints(ctx context.Context, permissionIDs []uint32) ([]Endpoint, error) {
	if len(permissionIDs) == 0 {
		return nil, nil
	}

	var rows []*models.PermissionApi
	if err := r.client.DB.WithContext(ctx).Model(&models.PermissionApi{}).
		Where("permission_id IN ?", permissionIDs).
		Where("path IS NOT NULL AND path <> '' AND method IS NOT NULL AND method <> ''").
		Find(&rows).Error; err != nil {
		r.log.Errorf(ctx, "list permission api endpoints failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("list permission api endpoints failed")
	}

	endpoints := make([]Endpoint, 0, len(rows))
	for _, row := range rows {
		if row == nil || row.Path == nil || row.Method == nil {
			continue
		}
		endpoints = append(endpoints, Endpoint{Path: *row.Path, Method: *row.Method})
	}
	return endpoints, nil
}

// ListApiIDs 端点快照反解回当前 sys_apis id（管理页勾选框回显）；
// 端点已删除的快照解析不出 id，防御性跳过。
func (r *PermissionApiRepo) ListApiIDs(ctx context.Context, permissionIDs []uint32) ([]uint32, error) {
	endpoints, err := r.ListEndpoints(ctx, permissionIDs)
	if err != nil {
		return nil, err
	}
	if len(endpoints) == 0 {
		return nil, nil
	}

	or := r.client.DB.WithContext(ctx).Model(&models.Api{})
	for i, e := range endpoints {
		scope := r.client.DB.WithContext(ctx).Where("path = ? AND method = ?", e.Path, e.Method)
		if i == 0 {
			or = or.Where("(?)", scope)
		} else {
			or = or.Or("(?)", scope)
		}
	}

	var apis []*models.Api
	if err := or.Find(&apis).Error; err != nil {
		r.log.Errorf(ctx, "resolve api ids by endpoints failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("resolve api ids by endpoints failed")
	}

	byEndpoint := make(map[string]uint32, len(apis))
	for _, a := range apis {
		if a == nil || a.Path == nil || a.Method == nil {
			continue
		}
		byEndpoint[(Endpoint{Path: *a.Path, Method: *a.Method}).key()] = a.ID
	}

	ids := make([]uint32, 0, len(endpoints))
	idSeen := make(map[uint32]struct{}, len(endpoints))
	for _, e := range endpoints {
		id, ok := byEndpoint[e.key()]
		if !ok {
			continue
		}
		if _, dup := idSeen[id]; dup {
			continue
		}
		idSeen[id] = struct{}{}
		ids = append(ids, id)
	}

	return ids, nil
}

// AssignApis 给权限分配API资源：入参是 api 引用（id 列表），落库前一次性解析成
// (path, method) 快照；提交了已不存在的 id → 整体 400 带缺失明细。
// 空列表 = 清空该权限的全部接口授权。
func (r *PermissionApiRepo) AssignApis(
	ctx context.Context,
	permissionID uint32,
	apiIDs []uint32,
) error {
	endpoints, missing, err := r.resolveEndpoints(ctx, apiIDs)
	if err != nil {
		return err
	}
	if len(missing) > 0 {
		return permissionV1.ErrorBadRequest("apis not found: %v", missing)
	}

	return r.client.DB.WithContext(ctx).Transaction(func(tx *gormDB.DB) error {
		if err := r.cleanNotExistApisWithClient(ctx, tx, permissionID, endpoints); err != nil {
			// 清理失效关联失败不阻断分配（仅告警），与 data 栈一致
			r.log.Warnf(ctx, "clean not exist apis for permission [%d] failed: %s", permissionID, err.Error())
		}
		return r.assignEndpointsWithClient(ctx, tx, permissionID, endpoints)
	})
}

// AssignApisWithTx 给权限分配API资源（端点快照入参，供已有事务的调用方复用）。
func (r *PermissionApiRepo) AssignApisWithTx(
	ctx context.Context,
	tx *gormDB.DB,
	permissionID uint32,
	endpoints []Endpoint,
) error {
	return r.assignEndpointsWithClient(ctx, tx, permissionID, endpoints)
}

func (r *PermissionApiRepo) assignEndpointsWithClient(
	ctx context.Context,
	tx *gormDB.DB,
	permissionID uint32,
	endpoints []Endpoint,
) error {
	if len(endpoints) == 0 {
		return nil
	}

	now := time.Now()

	for _, e := range endpoints {
		// 冲突即更新：依赖 (permission_id, path, method) 唯一索引
		//（PostgreSQL ON CONFLICT；SQLite 同语法；其余方言退化为先删后插）
		res := tx.Exec(
			`INSERT INTO sys_permission_apis (permission_id, path, method, created_at, updated_at)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT (permission_id, path, method) DO UPDATE SET updated_at = EXCLUDED.updated_at`,
			permissionID, e.Path, e.Method, now, now,
		)
		if res.Error != nil {
			r.log.Errorf(ctx, "assign permission api [%s %s] failed: %s", e.Method, e.Path, res.Error.Error())
			return permissionV1.ErrorInternalServerError("assign permission apis failed")
		}
	}

	return nil
}

// cleanNotExistApisWithClient 清理该权限下不在端点集合内的绑定快照；
// 空集合 = 清空该权限全部接口授权。
func (r *PermissionApiRepo) cleanNotExistApisWithClient(
	ctx context.Context,
	tx *gormDB.DB,
	permissionID uint32,
	endpoints []Endpoint,
) error {
	db := tx.Where("permission_id = ?", permissionID)
	if len(endpoints) > 0 {
		notIn := tx.Where("1 = 1")
		for _, e := range endpoints {
			notIn = notIn.Where("NOT (path = ? AND method = ?)", e.Path, e.Method)
		}
		db = db.Where("(?)", notIn)
	}
	if err := db.Delete(&models.PermissionApi{}).Error; err != nil {
		r.log.Errorf(ctx, "clean not exist permission apis failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("clean not exists permission apis failed")
	}
	return nil
}

// DeleteByEndpoints 按 (path, method) 级联清理绑定快照：API 行被删除/同步移除时
// 调用，消灭孤儿绑定。tx 形参使其可传入 SyncMerge 的事务。
func (r *PermissionApiRepo) DeleteByEndpoints(ctx context.Context, tx *gormDB.DB, path, method string) error {
	if path == "" || method == "" {
		return nil
	}

	if err := tx.WithContext(ctx).Where("path = ? AND method = ?", path, method).
		Delete(&models.PermissionApi{}).Error; err != nil {
		r.log.Errorf(ctx, "delete permission apis by endpoint [%s %s] failed: %s", method, path, err.Error())
		return permissionV1.ErrorInternalServerError("delete permission apis by endpoint failed")
	}
	return nil
}

// Truncate 清空绑定表（保留种子权限 1/2/3 的绑定）——权限同步重建流程专用。
func (r *PermissionApiRepo) Truncate(ctx context.Context) error {
	if err := r.client.DB.WithContext(ctx).
		Where("permission_id NOT IN ?", []uint32{1, 2, 3}).
		Delete(&models.PermissionApi{}).Error; err != nil {
		r.log.Errorf(ctx, "failed to truncate permission api table: %s", err.Error())
		return permissionV1.ErrorInternalServerError("truncate failed")
	}

	return nil
}

// Delete 删除权限关联的API资源
func (r *PermissionApiRepo) Delete(ctx context.Context, permissionID uint32) error {
	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("permission_id = ?", permissionID) },
	}); err != nil {
		r.log.Errorf(ctx, "delete permission apis by permission id failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("delete permission apis by permission id failed")
	}
	return nil
}

func (r *PermissionApiRepo) DeleteByPermissionIDs(ctx context.Context, permissionIDs []uint32) error {
	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("permission_id IN ?", permissionIDs) },
	}); err != nil {
		r.log.Errorf(ctx, "delete permission apis by permission ids failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("delete permission apis by permission ids failed")
	}
	return nil
}
