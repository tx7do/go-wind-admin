package models

import (
	"testing"

	gormsqlite "github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestAllModelsMigrate 验证 RegisterMigrateModels 中注册的全部 GORM model 都能在
// SQLite 内存库上完成 AutoMigrate。这是以 ent 为基准完善 gorm model 后的验收闸门：
// 任何字段/mixin/标签定义错误都会在迁移阶段暴露。
func TestAllModelsMigrate(t *testing.T) {
	db, err := gorm.Open(gormsqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		SkipDefaultTransaction: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})

	migrated := allMigrateModelsForTest()
	require.NotEmpty(t, migrated, "model list must not be empty")

	for _, m := range migrated {
		name := m.(interface{ TableName() string }).TableName()
		t.Run(name, func(t *testing.T) {
			require.NoError(t, db.AutoMigrate(m), "AutoMigrate failed for %s", name)
		})
	}
}

// allMigrateModelsForTest 返回与 RegisterMigrateModels（data/gorm/init.go）完全一致的
// 模型列表，集中维护以避免两处漂移。
func allMigrateModelsForTest() []interface{} {
	return []interface{}{
		&Api{}, &ApiAuditLog{}, &DataAccessAuditLog{},
		&DictEntry{}, &DictEntryI18n{}, &DictType{},
		&File{}, &InternalMessage{}, &InternalMessageCategory{}, &InternalMessageRecipient{},
		&Language{}, &LoginAuditLog{}, &LoginPolicy{},
		&Membership{}, &MembershipOrgUnit{}, &MembershipPosition{}, &MembershipRole{},
		&Menu{}, &OperationAuditLog{}, &OrgUnit{},
		&Permission{}, &PermissionApi{}, &PermissionAuditLog{}, &PermissionGroup{},
		&PermissionMenu{}, &PermissionPolicy{},
		&Plan{}, &PlanModule{}, &PlanQuota{},
		&PolicyEvaluationLog{}, &Position{}, &Role{}, &RoleMetadata{}, &RolePermission{},
		&Task{}, &Tenant{}, &User{}, &UserCredential{}, &UserMfaFactor{},
		&UserOrgUnit{}, &UserPosition{}, &UserRole{},
	}
}
