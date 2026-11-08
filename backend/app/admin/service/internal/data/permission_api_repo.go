package data

import (
	"context"
	"time"

	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/api"
	"go-wind-admin/app/admin/service/internal/data/ent/permissionapi"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"

	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
)

// Endpoint API 端点身份（path + method）——授权绑定的执行键。
//
// 与 authorizer 策略、租户闸门的裁决键同形：绑定存这个快照后，端点的删除重建
// （sys_apis id 漂移）不再影响授权有效性；要"变更身份"只能删旧建新，而删除会
// 级联清理绑定、重建需要重新授权，语义始终是明账。
type Endpoint struct {
	Path   string
	Method string
}

func (e Endpoint) key() string {
	return e.Path + "\x00" + e.Method
}

type PermissionApiRepo struct {
	log       *bLogger.Helper
	entClient *entCrud.EntClient[*ent.Client]
}

func NewPermissionApiRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *PermissionApiRepo {
	return &PermissionApiRepo{
		log:       ctx.NewLoggerHelper("permission-api/repo/admin-service"),
		entClient: entClient,
	}
}

// resolveEndpoints 把前端提交的 api 引用（id 列表）一次性解析成端点身份快照。
// 解析不到的 id 说明引用已过期（并发同步删除等）：随 missing 返回，由调用方
// 整体失败而非静默丢弃（不吞错铁律——静默丢绑定 = 授权悄悄缩水）。
func (r *PermissionApiRepo) resolveEndpoints(ctx context.Context, apiIDs []uint32) (endpoints []Endpoint, missing []uint32, err error) {
	if len(apiIDs) == 0 {
		return nil, nil, nil
	}

	apis, err := r.entClient.Client().Api.Query().
		Where(api.IDIn(apiIDs...)).
		All(ctx)
	if err != nil {
		r.log.Errorf(ctx, "query apis for endpoint resolve failed: %s", err.Error())
		return nil, nil, permissionV1.ErrorInternalServerError("query apis failed")
	}

	byID := make(map[uint32]Endpoint, len(apis))
	for _, a := range apis {
		if a == nil || a.Path == nil || a.Method == nil || *a.Path == "" || *a.Method == "" {
			// 缺身份的行不是合法端点（脏数据），按缺失处理
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

	rows, err := r.entClient.Client().PermissionApi.Query().
		Where(
			permissionapi.PermissionIDIn(permissionIDs...),
			permissionapi.PathNotNil(),
			permissionapi.PathNEQ(""),
			permissionapi.MethodNotNil(),
			permissionapi.MethodNEQ(""),
		).
		Select(permissionapi.FieldPath, permissionapi.FieldMethod).
		All(ctx)
	if err != nil {
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

// ListApiIDs 列出权限关联的 API 资源 ID 列表：端点快照反解回当前 sys_apis 的 id，
// 供管理页勾选框回显。端点已被同步删除的快照解析不出 id，防御性跳过
// （正常情况下端点删除会级联清绑定，此处只为并发窗口兜底）。
func (r *PermissionApiRepo) ListApiIDs(ctx context.Context, permissionIDs []uint32) ([]uint32, error) {
	endpoints, err := r.ListEndpoints(ctx, permissionIDs)
	if err != nil {
		return nil, err
	}
	if len(endpoints) == 0 {
		return nil, nil
	}

	ors := make([]predicate.Api, 0, len(endpoints))
	for _, e := range endpoints {
		ors = append(ors, api.And(api.PathEQ(e.Path), api.MethodEQ(e.Method)))
	}

	apis, err := r.entClient.Client().Api.Query().
		Where(api.Or(ors...)).
		All(ctx)
	if err != nil {
		r.log.Errorf(ctx, "resolve api ids by endpoints failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("resolve api ids by endpoints failed")
	}

	byEndpoint := make(map[string]uint32, len(apis))
	for _, a := range apis {
		if a == nil || a.Path == nil || a.Method == nil {
			continue
		}
		// 端点在库中唯一（(path,method) 唯一索引），一组键只对应一行
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

// AssignApis 给权限分配API资源：入参是前端语义的 api 引用（id 列表），落库前
// 一次性解析成 (path, method) 快照——绑定从此与 sys_apis 的 id 解耦。
// 提交了已不存在的 id → 整体 400 带缺失明细，不静默丢弃（不吞错铁律）。
// 空列表 = 清空该权限的全部接口授权。
func (r *PermissionApiRepo) AssignApis(
	ctx context.Context,
	permissionID uint32,
	apiIDs []uint32,
) (err error) {
	endpoints, missing, err := r.resolveEndpoints(ctx, apiIDs)
	if err != nil {
		return err
	}
	if len(missing) > 0 {
		return permissionV1.ErrorBadRequest("apis not found: %v", missing)
	}

	var tx *ent.Tx
	tx, err = r.entClient.Client().Tx(ctx)
	if err != nil {
		r.log.Errorf(ctx, "start transaction failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("start transaction failed")
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				r.log.Errorf(context.Background(), "transaction rollback failed: %s", rollbackErr.Error())
			}
			return
		}
		if commitErr := tx.Commit(); commitErr != nil {
			r.log.Errorf(context.Background(), "transaction commit failed: %s", commitErr.Error())
			err = permissionV1.ErrorInternalServerError("transaction commit failed")
		}
	}()

	// 清理失效关联失败不阻断分配（仅告警）：CleanNotExist 出错时仍执行 Assign，
	// 由 (permission, path, method) 唯一约束兜底重复数据
	if err = r.cleanNotExistApisWithClient(ctx, tx.Client(), permissionID, endpoints); err != nil {
		r.log.Warnf(context.Background(), "clean not exist apis for permission [%d] failed: %s", permissionID, err.Error())
	}

	return r.assignEndpointsWithTx(ctx, tx, permissionID, endpoints)
}

// AssignApisWithTx 给权限分配API资源（端点快照入参，供已有事务的调用方复用）。
func (r *PermissionApiRepo) AssignApisWithTx(
	ctx context.Context,
	tx *ent.Tx,
	permissionID uint32,
	endpoints []Endpoint,
) error {
	return r.assignEndpointsWithTx(ctx, tx, permissionID, endpoints)
}

func (r *PermissionApiRepo) assignEndpointsWithTx(
	ctx context.Context,
	tx *ent.Tx,
	permissionID uint32,
	endpoints []Endpoint,
) error {
	if len(endpoints) == 0 {
		return nil
	}

	now := time.Now()

	for _, e := range endpoints {
		pm := tx.PermissionApi.
			Create().
			SetPermissionID(permissionID).
			SetPath(e.Path).
			SetMethod(e.Method).
			SetCreatedAt(now).
			OnConflictColumns(
				permissionapi.FieldPermissionID,
				permissionapi.FieldPath,
				permissionapi.FieldMethod,
			).
			UpdateNewValues().
			SetUpdatedAt(now)
		if err := pm.Exec(ctx); err != nil {
			r.log.Errorf(ctx, "assign permission api [%s %s] failed: %s", e.Method, e.Path, err.Error())
			return permissionV1.ErrorInternalServerError("assign permission apis failed")
		}
	}

	return nil
}

// cleanNotExistApisWithClient 清理该权限下不在端点集合内的绑定快照；
// 空集合 = 清空该权限全部接口授权。client 形参使其同时可用于 tx 内。
func (r *PermissionApiRepo) cleanNotExistApisWithClient(
	ctx context.Context,
	client *ent.Client,
	permissionID uint32,
	endpoints []Endpoint,
) error {
	builder := client.PermissionApi.Delete().Where(permissionapi.PermissionIDEQ(permissionID))

	if len(endpoints) > 0 {
		ors := make([]predicate.PermissionApi, 0, len(endpoints))
		for _, e := range endpoints {
			ors = append(ors, permissionapi.And(
				permissionapi.PathEQ(e.Path),
				permissionapi.MethodEQ(e.Method),
			))
		}
		builder = builder.Where(permissionapi.Not(permissionapi.Or(ors...)))
	}

	if _, err := builder.Exec(ctx); err != nil {
		r.log.Errorf(ctx, "clean not exist permission apis failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("clean not exists permission apis failed")
	}
	return nil
}

// DeleteByEndpoints 按 (path, method) 级联清理绑定快照：API 行被删除/同步移除时
// 调用，消灭孤儿绑定。client 形参使其可传入 SyncMerge 的 tx client。
func (r *PermissionApiRepo) DeleteByEndpoints(ctx context.Context, client *ent.Client, path, method string) error {
	if path == "" || method == "" {
		return nil
	}

	if _, err := client.PermissionApi.Delete().
		Where(
			permissionapi.PathEQ(path),
			permissionapi.MethodEQ(method),
		).
		Exec(ctx); err != nil {
		r.log.Errorf(ctx, "delete permission apis by endpoint [%s %s] failed: %s", method, path, err.Error())
		return permissionV1.ErrorInternalServerError("delete permission apis by endpoint failed")
	}
	return nil
}

// Truncate 清空绑定表（保留种子权限 1/2/3 的绑定）——权限同步重建流程专用。
func (r *PermissionApiRepo) Truncate(ctx context.Context) error {
	builder := r.entClient.Client().PermissionApi.Delete().
		Where(
			permissionapi.PermissionIDNotIn(1, 2, 3),
		)

	if _, err := builder.Exec(ctx); err != nil {
		r.log.Errorf(ctx, "failed to truncate permission api table: %s", err.Error())
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
		r.log.Errorf(ctx, "delete permission apis by permission id failed: %s", err.Error())
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
		r.log.Errorf(ctx, "delete permission apis by permission ids failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("delete permission apis by permission ids failed")
	}
	return nil
}
