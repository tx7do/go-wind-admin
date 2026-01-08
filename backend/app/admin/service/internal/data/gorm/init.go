package gorm

import (
	"github.com/tx7do/go-crud/gorm"

	"go-wind-admin/app/admin/service/internal/data/gorm/models"
)

func init() {
	RegisterMigrateModels()
}

// RegisterMigrateModels registers all GORM models for migration.
func RegisterMigrateModels() {
	gorm.RegisterMigrateModels(
		&models.AdminLoginLog{},
		&models.AdminLoginRestriction{},
		&models.AdminOperationLog{},
		&models.Api{},
		&models.Department{},
		&models.DictEntry{},
		&models.DictType{},
		&models.File{},
		&models.InternalMessage{},
		&models.InternalMessageCategory{},
		&models.InternalMessageRecipient{},
		&models.Language{},
		&models.Menu{},
		&models.Organization{},
		&models.Position{},
		&models.Role{},
		&models.RoleApi{},
		&models.RoleDept{},
		&models.RoleMenu{},
		&models.RoleOrg{},
		&models.RolePosition{},
		&models.Task{},
		&models.Tenant{},
		&models.User{},
		&models.UserCredential{},
		&models.UserPosition{},
		&models.UserRole{},
	)
}
