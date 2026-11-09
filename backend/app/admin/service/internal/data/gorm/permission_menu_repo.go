//go:build gorm_backend
// +build gorm_backend

package gorm

import (
	"context"
	"strings"
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

// PermissionMenuRepo 权限-菜单授权绑定仓储。
//
// 绑定存菜单全路径（full_path）快照而非 menu_id：全路径是菜单的身份键（菜单
// 同步 MERGE 按它保号、REPLACE 重建后同路径照常命中），授权不再随菜单 id
// 重建而漂移——与 sys_permission_apis 的 (path, method) 端点快照同构。
// data 栈同构镜像，改这里须同步 data/permission_menu_repo.go。
type PermissionMenuRepo struct {
	client     *gormCrud.Client
	log        *bLogger.Helper
	mapper     *mapper.CopierMapper[permissionV1.PermissionMenu, models.PermissionMenu]
	repository *gormCrud.Repository[permissionV1.PermissionMenu, models.PermissionMenu]
}

func NewPermissionMenuRepo(
	ctx *bootstrap.Context,
	client *gormCrud.Client,
) *PermissionMenuRepo {
	repo := &PermissionMenuRepo{
		log:    ctx.NewLoggerHelper("permission-menu/gorm-repo/admin-service"),
		client: client,
		mapper: mapper.NewCopierMapper[permissionV1.PermissionMenu, models.PermissionMenu](),
	}

	repo.init()

	return repo
}

func (r *PermissionMenuRepo) init() {
	r.repository = gormCrud.NewRepository[permissionV1.PermissionMenu, models.PermissionMenu](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
}

// joinMenuFullPathGorm 菜单全路径拼接：与 data 栈 menu_repo.joinMenuFullPath 逐字
// 一致（身份键两侧同源，规则漂移=绑定反查错位）。gorm 包与 data 包不互引，故复制。
// 注意**绝对路径分支**：path 以 "/" 开头时直接脱离父链（如 /dashboard 下的
// /analytics、/workspace 是顶层绝对路由），漏掉会把子菜单拼成 //parent//child。
func joinMenuFullPathGorm(parentFullPath, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return parentFullPath
	}
	if strings.HasPrefix(path, "/") {
		return path
	}
	if parentFullPath == "" || parentFullPath == "/" {
		return "/" + path
	}
	return strings.TrimRight(parentFullPath, "/") + "/" + path
}

// buildMenuFullPathIDIndex 全量菜单行树遍历，产出 id→全路径映射。
func (r *PermissionMenuRepo) buildMenuFullPathIDIndex(ctx context.Context) (map[uint64]string, error) {
	var rows []*models.Menu
	if err := r.client.DB.WithContext(ctx).Model(&models.Menu{}).Find(&rows).Error; err != nil {
		r.log.Errorf(ctx, "query existing menus failed: %s", err.Error())
		return nil, permissionV1.ErrorInternalServerError("query existing menus failed")
	}

	childrenOf := make(map[uint64][]*models.Menu, len(rows))
	var roots []*models.Menu
	for _, row := range rows {
		// models.Menu 经 mixin.Tree64 内嵌 ParentID64，父指针经字段提升直接可达
		if row.ParentID == nil || *row.ParentID == 0 {
			roots = append(roots, row)
		} else {
			childrenOf[*row.ParentID] = append(childrenOf[*row.ParentID], row)
		}
	}

	byID := make(map[uint64]string, len(rows))
	visited := make(map[uint64]bool, len(rows))
	var walk func(rows []*models.Menu, parentFullPath string)
	walk = func(rows []*models.Menu, parentFullPath string) {
		for _, row := range rows {
			if visited[row.ID] {
				continue // 脏数据成环防护
			}
			visited[row.ID] = true
			fullPath := joinMenuFullPathGorm(parentFullPath, derefStrPtr(row.Path))
			byID[row.ID] = fullPath
			walk(childrenOf[row.ID], fullPath)
		}
	}
	walk(roots, "")

	return byID, nil
}

func derefStrPtr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// resolveFullPaths 把菜单引用（id 列表）解析成全路径身份快照；解析不到随 missing
// 返回由调用方整体失败（不吞错铁律）。
func (r *PermissionMenuRepo) resolveFullPaths(ctx context.Context, menuIDs []uint64) (fullPaths []string, missing []uint64, err error) {
	if len(menuIDs) == 0 {
		return nil, nil, nil
	}

	byID, err := r.buildMenuFullPathIDIndex(ctx)
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
func (r *PermissionMenuRepo) ListMenuFullPaths(ctx context.Context, permissionIDs []uint64) ([]string, error) {
	if len(permissionIDs) == 0 {
		return nil, nil
	}

	var rows []*models.PermissionMenu
	if err := r.client.DB.WithContext(ctx).Model(&models.PermissionMenu{}).
		Where("permission_id IN ?", permissionIDs).
		Where("full_path IS NOT NULL AND full_path <> ''").
		Find(&rows).Error; err != nil {
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

// ListMenuIDs 全路径快照反解回当前 sys_menus id（管理页勾选框回显）；
// 路径已不存在的快照解析不出 id，防御性跳过。
func (r *PermissionMenuRepo) ListMenuIDs(ctx context.Context, permissionIDs []uint64) ([]uint64, error) {
	paths, err := r.ListMenuFullPaths(ctx, permissionIDs)
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, nil
	}

	byID, err := r.buildMenuFullPathIDIndex(ctx)
	if err != nil {
		return nil, err
	}
	pathToID := make(map[string]uint64, len(byID))
	for id, p := range byID {
		pathToID[p] = id
	}

	ids := make([]uint64, 0, len(paths))
	seen := make(map[uint64]struct{}, len(paths))
	for _, p := range paths {
		if id, ok := pathToID[p]; ok {
			if _, dup := seen[id]; dup {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// AssignMenus 给权限分配菜单：入参是菜单引用（id 列表），落库前一次性解析成
// 全路径快照；提交了已不存在的 id → 整体 400 带缺失明细。空列表 = 清空授权。
func (r *PermissionMenuRepo) AssignMenus(
	ctx context.Context,
	permissionID uint64,
	menuIDs []uint64,
) error {
	fullPaths, missing, err := r.resolveFullPaths(ctx, menuIDs)
	if err != nil {
		return err
	}
	if len(missing) > 0 {
		return permissionV1.ErrorBadRequest("menus not found: %v", missing)
	}

	return r.client.DB.WithContext(ctx).Transaction(func(tx *gormDB.DB) error {
		if err := r.cleanNotExistMenusWithClient(ctx, tx, permissionID, fullPaths); err != nil {
			// 清理失效关联失败不阻断分配（仅告警），与 data 栈一致
			r.log.Warnf(ctx, "clean not exist menus for permission [%d] failed: %s", permissionID, err.Error())
		}
		return r.assignFullPathsWithClient(ctx, tx, permissionID, fullPaths)
	})
}

// AssignMenusWithTx 给权限分配菜单（全路径快照入参，供已有事务的调用方复用）。
func (r *PermissionMenuRepo) AssignMenusWithTx(
	ctx context.Context,
	tx *gormDB.DB,
	permissionID uint64,
	fullPaths []string,
) error {
	return r.assignFullPathsWithClient(ctx, tx, permissionID, fullPaths)
}

func (r *PermissionMenuRepo) assignFullPathsWithClient(
	ctx context.Context,
	tx *gormDB.DB,
	permissionID uint64,
	fullPaths []string,
) error {
	if len(fullPaths) == 0 {
		return nil
	}

	now := time.Now()

	for _, p := range fullPaths {
		// 冲突即更新：依赖 (permission_id, full_path) 唯一索引
		res := tx.Exec(
			`INSERT INTO sys_permission_menus (permission_id, full_path, created_at, updated_at)
VALUES (?, ?, ?, ?)
ON CONFLICT (permission_id, full_path) DO UPDATE SET updated_at = EXCLUDED.updated_at`,
			permissionID, p, now, now,
		)
		if res.Error != nil {
			r.log.Errorf(ctx, "assign permission menu [%s] failed: %s", p, res.Error.Error())
			return permissionV1.ErrorInternalServerError("assign permission menus failed")
		}
	}

	return nil
}

// cleanNotExistMenusWithClient 清理该权限下不在全路径集合内的绑定快照；
// 空集合 = 清空该权限全部菜单授权。
func (r *PermissionMenuRepo) cleanNotExistMenusWithClient(
	ctx context.Context,
	tx *gormDB.DB,
	permissionID uint64,
	fullPaths []string,
) error {
	db := tx.Where("permission_id = ?", permissionID)
	if len(fullPaths) > 0 {
		notIn := tx.Where("1 = 1")
		for _, p := range fullPaths {
			notIn = notIn.Where("full_path <> ?", p)
		}
		db = db.Where("(?)", notIn)
	}
	if err := db.Delete(&models.PermissionMenu{}).Error; err != nil {
		r.log.Errorf(ctx, "clean not exist permission menus failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("clean not exists permission menus failed")
	}
	return nil
}

// DeleteByFullPaths 按菜单全路径级联清理授权绑定：菜单行（含子树）被删除时调用，
// 消灭悬空绑定。菜单同步 REPLACE 的清空重建不走这里——重建后同路径新行照常命中。
func (r *PermissionMenuRepo) DeleteByFullPaths(ctx context.Context, tx *gormDB.DB, fullPaths []string) error {
	if len(fullPaths) == 0 {
		return nil
	}

	if err := tx.WithContext(ctx).Where("full_path IN ?", fullPaths).
		Delete(&models.PermissionMenu{}).Error; err != nil {
		r.log.Errorf(ctx, "delete permission menus by full paths failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("delete permission menus by full paths failed")
	}
	return nil
}

// TruncateExcept 清空绑定表但保留指定权限的绑定（由调用方按 code 前缀解析）；
// 为空 = 全清。
func (r *PermissionMenuRepo) TruncateExcept(ctx context.Context, keepPermissionIDs []uint64) error {
	db := r.client.DB.WithContext(ctx)
	if len(keepPermissionIDs) > 0 {
		db = db.Where("permission_id NOT IN ?", keepPermissionIDs)
	} else {
		db = db.Where("1 = 1")
	}
	if err := db.Delete(&models.PermissionMenu{}).Error; err != nil {
		r.log.Errorf(ctx, "failed to truncate permission menu table: %s", err.Error())
		return permissionV1.ErrorInternalServerError("truncate failed")
	}
	return nil
}

// Delete 删除权限关联的菜单
func (r *PermissionMenuRepo) Delete(ctx context.Context, permissionID uint64) error {
	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("permission_id = ?", permissionID) },
	}); err != nil {
		r.log.Errorf(ctx, "delete permission menus by permission id failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("delete failed")
	}
	return nil
}

func (r *PermissionMenuRepo) DeleteByPermissionIDs(ctx context.Context, permissionIDs []uint64) error {
	if _, err := r.repository.DeleteWithFilters(ctx, r.client.DB, []func(*gormDB.DB) *gormDB.DB{
		func(db *gormDB.DB) *gormDB.DB { return db.Where("permission_id IN ?", permissionIDs) },
	}); err != nil {
		r.log.Errorf(ctx, "delete permission menus by permission ids failed: %s", err.Error())
		return permissionV1.ErrorInternalServerError("delete permission menus by permission ids failed")
	}
	return nil
}
