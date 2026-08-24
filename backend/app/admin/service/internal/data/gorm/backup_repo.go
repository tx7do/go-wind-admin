//go:build gorm_backend
// +build gorm_backend

// Package gorm 中的仓储是 ent 仓储的平行 gorm 镜像，作为"ent 为主力、gorm 为备选"脚手架的完整代码。
// 这些仓储为死代码：不接入 wire、不被 service 引用；采用者需要时自行装配。
package gorm

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
)

// BackupRepo gorm 脚手架桩：ent 侧跨 8 张表（Tenant/User/Role/Permission/Membership/OrgUnit/Position/Menu）
// 的全表导出，无单一 gorm model，go-crud/gorm 无对应原语。见 data/backup_repo.go。
type BackupRepo struct {
	log *log.Helper
}

func NewBackupRepo(ctx *bootstrap.Context) *BackupRepo {
	return &BackupRepo{
		log: ctx.NewLoggerHelper("backup/gorm-repo/admin-service"),
	}
}

func (r *BackupRepo) IsConfigured() bool {
	return false
}

func (r *BackupRepo) ExportCoreTables(ctx context.Context) (map[string]any, error) {
	return nil, nil
}
