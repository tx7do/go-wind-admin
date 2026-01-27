package gorm

import (
	gormCrud "github.com/tx7do/go-crud/gorm"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	gormBootstrap "github.com/tx7do/kratos-bootstrap/database/gorm"
)

// NewGormClient 创建GORM ORM数据库客户端
func NewGormClient(ctx *bootstrap.Context) (*gormCrud.Client, error) {
	l := ctx.NewLoggerHelper("gorm/data/admin-service")

	cfg := ctx.GetConfig()
	if cfg == nil || cfg.Data == nil {
		l.Fatalf("[GORM] failed getting config")
		return nil, nil
	}

	RegisterMigrateModels()

	return gormBootstrap.NewGormClient(cfg, l, nil)
}
