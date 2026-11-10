package data

import (
	"context"
	"time"

	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/permissionmenu"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"

	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
)

// PermissionMenuRepo 权限-菜单授权绑定仓储。
//
// 绑定存菜单全路径（full_path）快照而非 menu_id：全路径是菜单的身份键（菜单
// 同步 MERGE 按 it 保号、REPLACE 重建后同路径照常命中），授权不再随菜单 id
// 重建而漂移——与 sys_permission_apis 的 (path, method) 端点快照同构。
// 线格式不变：前端照旧传 menu_ids，落库前一次性解析成全路径快照。
type PermissionMenuRepo struct {
	log       *bLogger.Helper
	entClient *entCrud.EntClient[*ent.Client]
}

func NewPermissionMenuRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *PermissionMenuRepo {
	return &PermissionMenuRepo{
		log:       ctx.NewLoggerHelper("permission-menu/repo/admin-service"),
		entClient: entClient,
	}
}

// menuRepoForIndex 构造仅用于全路径索引的 MenuRepo（绕开 wiring 依赖环：
// MenuRepo 反过来持有本 repo 做删除级联）。索引构建是无状态查询，共享安全。
func (r *PermissionMenuRepo) menuRepoForIndex(log *bLogger.Helper) *MenuRepo {
	return &MenuRepo{log: log, entClient: r.entClient}
}

// resolveFullPaths 把前端提交的菜单引用（id 列表）一次性解析成全路径身份快照。
// 解析不到的 id 随 missing 返回，由调用方整体失败而非静默丢弃（不吞错铁律）。
func (r *PermissionMenuRepo) resolveFullPaths(ctx context.Context, menuIDs []int64) (fullPaths []string, missing []int64, err error) {
	if len(menuIDs) == 0 {
		return nil, nil, nil
	}

	_, byID, _, err := r.menuRepoForIndex(r.log).buildMenuFullPathIndex(ctx, r.entClient.Client())
	if err != nil {
		return nil, nil, err
	}

	seen := make(map[string]struct{}, len(menuIDs))
	for _, id := range menuIDs {
		p, ok := byID[id]
		if !ok || p == "" {
			missing = append(missing, id)
			continue
		}
		if _, dup := seen[p]; dup {
			continue
		}
		seen[p] = struct{}{}
		fullPaths = append(fullPaths, p)
	}

	return fullPaths, missing, nil
}

// ListMenuFullPaths 列出权限关联的菜单全路径快照（绑定表即真相，不回查 sys_menus）。
func (r *PermissionMenuRepo) ListMenuFullPaths(ctx context.Context, permissionIDs []int64) ([]string, error) {
	if len(permissionIDs) == 0 {
		return nil, nil
	}

	rows, err := r.entClient.Client().PermissionMenu.Query().
		Where(
			permissionmenu.PermissionIDIn(permissionIDs...),
			permissionmenu.FullPathNotNil(),
			permissionmenu.FullPathNEQ(""),
		).
		Select(permissionmenu.FieldFullPath).
		All(ctx)
	if err != nil {
		r.log.Errorf(ctx, "list permission menu full paths failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("list permission menu full paths failed")
	}

	paths := make([]string, 0, len(rows))
	for _, row := range rows {
		if row == nil || row.FullPath == nil {
			continue
		}
		paths = append(paths, *row.FullPath)
	}
	return paths, nil
}

// ListMenuIDs 列出权限关联的菜单 ID 列表：全路径快照反解回当前 sys_menus 的 id，
// 供管理页勾选框回显。路径已不存在的快照解析不出 id，防御性跳过（正常情况下
// 菜单删除会级联清绑定；菜单同步 REPLACE 重建后同路径新行照常命中）。
func (r *PermissionMenuRepo) ListMenuIDs(ctx context.Context, permissionIDs []int64) ([]int64, error) {
	paths, err := r.ListMenuFullPaths(ctx, permissionIDs)
	if err != nil {
		return nil, err
	}
	return r.menuRepoForIndex(r.log).ResolveMenuIDsByFullPaths(ctx, paths)
}

// AssignMenus 给权限分配菜单：入参是前端语义的菜单引用（id 列表），落库前一次性
// 解析成全路径快照。提交了已不存在的 id → 整体 400 带缺失明细，不静默丢弃。
// 空列表 = 清空该权限的全部菜单授权。
func (r *PermissionMenuRepo) AssignMenus(
	ctx context.Context,
	permissionID int64,
	menuIDs []int64,
) (err error) {
	fullPaths, missing, err := r.resolveFullPaths(ctx, menuIDs)
	if err != nil {
		return err
	}
	if len(missing) > 0 {
		return permissionV1.ErrorBadRequest("menus not found: %v", missing)
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

	// 清理失效关联失败不阻断分配（仅告警），理由同 CleanNotExistApis：
	// 由 (permission_id, full_path) 唯一约束兜底重复数据
	if err = r.cleanNotExistMenusWithClient(ctx, tx.Client(), permissionID, fullPaths); err != nil {
		r.log.Warnf(context.Background(), "clean not exist menus for permission [%d] failed: %s", permissionID, err.Error())
	}

	return r.assignFullPathsWithTx(ctx, tx, permissionID, fullPaths)
}

// AssignMenusWithTx 给权限分配菜单（全路径快照入参，供已有事务的调用方复用）。
func (r *PermissionMenuRepo) AssignMenusWithTx(
	ctx context.Context,
	tx *ent.Tx,
	permissionID int64,
	fullPaths []string,
) error {
	return r.assignFullPathsWithTx(ctx, tx, permissionID, fullPaths)
}

func (r *PermissionMenuRepo) assignFullPathsWithTx(
	ctx context.Context,
	tx *ent.Tx,
	permissionID int64,
	fullPaths []string,
) error {
	if len(fullPaths) == 0 {
		return nil
	}

	now := time.Now()

	for _, p := range fullPaths {
		pm := tx.PermissionMenu.
			Create().
			SetPermissionID(permissionID).
			SetFullPath(p).
			SetCreatedAt(now).
			OnConflictColumns(
				permissionmenu.FieldPermissionID,
				permissionmenu.FieldFullPath,
			).
			UpdateNewValues().
			SetUpdatedAt(now)
		if err := pm.Exec(ctx); err != nil {
			r.log.Errorf(ctx, "assign permission menu [%s] failed: %s", p, err.Error())
			return permissionV1.ErrorInternalServerError("assign permission menus failed")
		}
	}

	return nil
}

// cleanNotExistMenusWithClient 清理该权限下不在全路径集合内的绑定快照；
// 空集合 = 清空该权限全部菜单授权。client 形参使其同时可用于 tx 内。
func (r *PermissionMenuRepo) cleanNotExistMenusWithClient(
	ctx context.Context,
	client *ent.Client,
	permissionID int64,
	fullPaths []string,
) error {
	builder := client.PermissionMenu.Delete().Where(permissionmenu.PermissionIDEQ(permissionID))

	if len(fullPaths) > 0 {
		ors := make([]predicate.PermissionMenu, 0, len(fullPaths))
		for _, p := range fullPaths {
			ors = append(ors, permissionmenu.FullPathEQ(p))
		}
		builder = builder.Where(permissionmenu.Not(permissionmenu.Or(ors...)))
	}

	if _, err := builder.Exec(ctx); err != nil {
		r.log.Errorf(ctx, "clean not exist permission menus failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("clean not exists permission menus failed")
	}
	return nil
}

// UpdateFullPaths 菜单移动/改名后按"旧全路径 → 新全路径"改写绑定快照：
// 绑定以全路径为身份键，菜单行的全路径变了绑定必须随行，否则角色授权静默
// 失效（DeleteByFullPaths 的更新侧对称语义）。整个改写单事务执行，撞
// (permission_id, full_path) 唯一约束即整体回滚并报错（明账，不静默吞）。
func (r *PermissionMenuRepo) UpdateFullPaths(ctx context.Context, rewrites map[string]string) error {
	if len(rewrites) == 0 {
		return nil
	}

	var tx *ent.Tx
	var err error
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

	for oldPath, newPath := range rewrites {
		if uerr := tx.PermissionMenu.Update().
			Where(permissionmenu.FullPathEQ(oldPath)).
			SetFullPath(newPath).
			Exec(ctx); uerr != nil {
			r.log.Errorf(ctx, "rewrite permission menu full path [%s -> %s] failed: %s", oldPath, newPath, uerr.Error())
			err = permissionV1.ErrorInternalServerError("update permission menu full paths failed")
			return err
		}
	}

	return nil
}

// DeleteByFullPaths 按菜单全路径级联清理授权绑定：菜单行（含子树）被删除时
// 调用，消灭悬空绑定。注意：菜单同步 REPLACE 的清空重建**不走这里**——
// 重建后同路径新行照常命中快照，授权存续正是快照绑定的意义所在。
func (r *PermissionMenuRepo) DeleteByFullPaths(ctx context.Context, client *ent.Client, fullPaths []string) error {
	if len(fullPaths) == 0 {
		return nil
	}

	if _, err := client.PermissionMenu.Delete().
		Where(
			permissionmenu.FullPathIn(fullPaths...),
		).
		Exec(ctx); err != nil {
		r.log.Errorf(ctx, "delete permission menus by full paths failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("delete permission menus by full paths failed")
	}
	return nil
}

// TruncateExcept 清空绑定表但保留指定权限（按 code 前缀解析出的系统种子权限）
// 的绑定。keepPermissionIDs 为空 = 全清（权限表整体重建场景）。
func (r *PermissionMenuRepo) TruncateExcept(ctx context.Context, keepPermissionIDs []int64) error {
	builder := r.entClient.Client().PermissionMenu.Delete()
	if len(keepPermissionIDs) > 0 {
		builder = builder.Where(permissionmenu.PermissionIDNotIn(keepPermissionIDs...))
	}

	if _, err := builder.Exec(ctx); err != nil {
		r.log.Errorf(ctx, "failed to truncate permission menu table: %s", err.Error())
		return permissionV1.ErrorInternalServerError("truncate failed")
	}

	return nil
}

// Delete 删除权限关联的菜单
func (r *PermissionMenuRepo) Delete(ctx context.Context, permissionID int64) error {
	if _, err := r.entClient.Client().PermissionMenu.Delete().
		Where(
			permissionmenu.PermissionIDEQ(permissionID),
		).
		Exec(ctx); err != nil {
		r.log.Errorf(ctx, "failed to delete permission menu by permission id: %s", err.Error())
		return permissionV1.ErrorInternalServerError("delete failed")
	}

	return nil
}

func (r *PermissionMenuRepo) DeleteByPermissionIDs(ctx context.Context, permissionIDs []int64) error {
	return r.DeleteByPermissionIDsOnClient(ctx, r.entClient.Client(), permissionIDs)
}

// DeleteByPermissionIDsOnClient DeleteByPermissionIDs 的 client 形参变体：
// 权限行删除+级联清理同事务执行时传事务 client。
func (r *PermissionMenuRepo) DeleteByPermissionIDsOnClient(ctx context.Context, client *ent.Client, permissionIDs []int64) error {
	if len(permissionIDs) == 0 {
		return nil
	}
	if _, err := client.PermissionMenu.Delete().
		Where(
			permissionmenu.PermissionIDIn(permissionIDs...),
		).
		Exec(ctx); err != nil {
		r.log.Errorf(ctx, "delete permission menus by permission ids failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("delete permission menus by permission ids failed")
	}
	return nil
}
