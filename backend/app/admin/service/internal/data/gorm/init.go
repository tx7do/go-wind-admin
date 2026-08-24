//go:build gorm_backend
// +build gorm_backend

package gorm

import (
	"github.com/tx7do/go-crud/gorm"

	"go-wind-admin/app/admin/service/internal/data/gorm/models"
)

// RegisterMigrateModels 注册全部 GORM 模型以供 AutoMigrate。
//
// 仅在 NewGormClient（即运行时切换选中 gorm 后端）内部调用，不保留全局 init()。
// 这里不再保留全局 init()——init 会在包导入时无条件执行，包括 ent 后端构建时，从而给
// ent 二进制注册一堆用不到的 gorm 迁移模型（副作用污染）。
//
// 模型清单与 internal/data/gorm/models 一一对应，以 ent schema 为单一事实源
// （字段/列名/类型/可空性取自 ent 生成的实体代码）。
func RegisterMigrateModels() {
	gorm.RegisterMigrateModels(
		&models.Api{},
		&models.ApiAuditLog{},
		&models.DataAccessAuditLog{},
		&models.DictEntry{},
		&models.DictEntryI18n{},
		&models.DictType{},
		&models.File{},
		&models.InternalMessage{},
		&models.InternalMessageCategory{},
		&models.InternalMessageRecipient{},
		&models.Language{},
		&models.LoginAuditLog{},
		&models.LoginPolicy{},
		&models.Membership{},
		&models.MembershipOrgUnit{},
		&models.MembershipPosition{},
		&models.MembershipRole{},
		&models.Menu{},
		&models.OperationAuditLog{},
		&models.OrgUnit{},
		&models.Permission{},
		&models.PermissionApi{},
		&models.PermissionAuditLog{},
		&models.PermissionGroup{},
		&models.PermissionMenu{},
		&models.PermissionPolicy{},
		&models.Plan{},
		&models.PlanModule{},
		&models.PlanQuota{},
		&models.PolicyEvaluationLog{},
		&models.Position{},
		&models.Role{},
		&models.RoleMetadata{},
		&models.RolePermission{},
		&models.Task{},
		&models.Tenant{},
		&models.User{},
		&models.UserCredential{},
		&models.UserOrgUnit{},
		&models.UserPosition{},
		&models.UserRole{},
	)
}
