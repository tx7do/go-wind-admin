package data

import (
	"context"
	"fmt"

	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	"go-wind-admin/app/admin/service/internal/data/ent"
)

// BackupRepo 负责数据库核心业务表的全量导出，供异步备份任务使用。
//
// 它把对 ent client 的直接依赖收敛在 data 层，避免 service 层（如 TaskService）
// 直接持有 *entCrud.EntClient，从而维持分层边界。
type BackupRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *bLogger.Helper
}

// NewBackupRepo 创建备份仓库。
func NewBackupRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *BackupRepo {
	return &BackupRepo{
		entClient: entClient,
		log:       ctx.NewLoggerHelper("backup/repo/admin-service"),
	}
}

// IsConfigured 返回备份所需依赖（ent client）是否就绪。
func (r *BackupRepo) IsConfigured() bool {
	return r != nil && r.entClient != nil
}

// ExportCoreTables 导出核心业务表的全量记录。
//
// 仅导出身份/权限/组织相关配置表（数据量可控、恢复价值高）；审计日志等大体量表不纳入。
//
// 任一核心表导出失败即返回 error：备份是"全量恢复"语义，缺表会造成静默还原残缺数据，
// 因此宁可不产出也不产出残缺备份。调用方据此决定是否重试。
//
// 返回的 map 键为表名，值为可被 json 序列化的 ent 实体切片。
func (r *BackupRepo) ExportCoreTables(ctx context.Context) (map[string]any, error) {
	client := r.entClient.Client()
	result := make(map[string]any)

	// 逐表导出：每张表 Query All 后，实体自带 json tag 可直接序列化
	type tableExport struct {
		name  string
		query func() (any, error)
	}

	exports := []tableExport{
		{"tenants", func() (any, error) { return client.Tenant.Query().All(ctx) }},
		{"users", func() (any, error) { return client.User.Query().All(ctx) }},
		{"roles", func() (any, error) { return client.Role.Query().All(ctx) }},
		{"permissions", func() (any, error) { return client.Permission.Query().All(ctx) }},
		{"memberships", func() (any, error) { return client.Membership.Query().All(ctx) }},
		{"org_units", func() (any, error) { return client.OrgUnit.Query().All(ctx) }},
		{"positions", func() (any, error) { return client.Position.Query().All(ctx) }},
		{"menus", func() (any, error) { return client.Menu.Query().All(ctx) }},
	}

	for _, t := range exports {
		rows, err := t.query()
		if err != nil {
			// 核心表导出失败直接返回错误，避免产出残缺备份被当作成功
			r.log.Errorf(ctx, "backup: export table %q failed: %s", t.name, err.Error())
			return nil, fmt.Errorf("export table %q failed: %w", t.name, err)
		}
		result[t.name] = rows
	}

	return result, nil
}
