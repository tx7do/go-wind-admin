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

// DistributionRow 统计分布行（与 data.DistributionRow 结构一致，供 service 层使用）。
type DistributionRow struct {
	Action string `sql:"action"`
	Status string `sql:"status"`
	Count  int    `sql:"count"`
}

// TrendRow 登录趋势按日分桶后的结果项（与 data.TrendRow 结构一致，供 service 层使用）。
type TrendRow struct {
	Date  string
	Count int
}

// DashboardRepo gorm 脚手架桩：ent 侧聚合 4 张表（User/Role/LoginAuditLog/OperationAuditLog）
// 的 GroupBy/Aggregate/Scan 统计，go-crud/gorm 无对应原语，且无单一 gorm model。
// 见 data/dashboard_repo.go。
type DashboardRepo struct {
	log *log.Helper
}

func NewDashboardRepo(ctx *bootstrap.Context) *DashboardRepo {
	return &DashboardRepo{
		log: ctx.NewLoggerHelper("dashboard/gorm-repo/admin-service"),
	}
}

func (r *DashboardRepo) CountActiveUsers(ctx context.Context) (int, error) {
	return 0, nil
}

func (r *DashboardRepo) CountRoles(ctx context.Context) (int, error) {
	return 0, nil
}

func (r *DashboardRepo) CountTodayLogins(ctx context.Context) (int, error) {
	return 0, nil
}

func (r *DashboardRepo) CountTodayOperations(ctx context.Context) (int, error) {
	return 0, nil
}

func (r *DashboardRepo) LoginTrend(ctx context.Context, days int) ([]TrendRow, error) {
	return []TrendRow{}, nil
}

func (r *DashboardRepo) OperationActionDistribution(ctx context.Context) ([]DistributionRow, error) {
	return []DistributionRow{}, nil
}

func (r *DashboardRepo) LoginStatusDistribution(ctx context.Context) ([]DistributionRow, error) {
	return []DistributionRow{}, nil
}
