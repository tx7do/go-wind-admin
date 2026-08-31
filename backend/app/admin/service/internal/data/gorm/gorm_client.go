//go:build gorm_backend
// +build gorm_backend

package gorm

import (
	"context"
	"fmt"

	gormCrud "github.com/tx7do/go-crud/gorm"
	windLog "github.com/tx7do/go-wind/log"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	conf "github.com/tx7do/kratos-bootstrap/api/gen/go/conf/v1"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
)

// NewGormClient 创建GORM ORM数据库客户端。
//
// 不走 kratos-bootstrap/database/gorm 的同名构造:其 v0.1.6 把 kratos *log.Helper
// 直接传给要求 go-wind/log.Logger(ctx 风格)的 go-crud/gorm,两上游接口错位无法
// 编译;这里在仓库内复制其配置装配,日志经 helperLogger 适配。上游对齐后可整体
// 换回 gormBootstrap.NewGormClient 并删除适配器。
func NewGormClient(ctx *bootstrap.Context) (*gormCrud.Client, error) {
	l := ctx.NewLoggerHelper("gorm/data/admin-service")

	cfg := ctx.GetConfig()
	if cfg == nil || cfg.Data == nil || cfg.Data.Database == nil {
		l.Warnf(nil, "[GORM] database config is nil")
		return nil, nil
	}

	RegisterMigrateModels()

	return newGormClientFromConf(cfg, helperLogger{h: l})
}

// newGormClientFromConf 按 Bootstrap 配置构建 go-crud/gorm 客户端
// (与 kratos-bootstrap/database/gorm v0.1.6 的装配逻辑一致)。
func newGormClientFromConf(cfg *conf.Bootstrap, logger windLog.Logger) (*gormCrud.Client, error) {
	var options []gormCrud.Option

	options = append(options, gormCrud.WithLogger(logger))

	if cfg.Data.Database.GetDriver() != "" {
		options = append(options, gormCrud.WithDriverName(cfg.Data.Database.GetDriver()))
	}
	if cfg.Data.Database.GetSource() != "" {
		options = append(options, gormCrud.WithDSN(cfg.Data.Database.GetSource()))
	}

	options = append(options,
		gormCrud.WithEnableMigrate(cfg.Data.Database.GetMigrate()),
		gormCrud.WithEnableTrace(cfg.Data.Database.GetEnableTrace()),
		gormCrud.WithEnableMetrics(cfg.Data.Database.GetEnableMetrics()),
	)

	if cfg.Data.Database.MaxIdleConnections != nil {
		options = append(options, gormCrud.WithMaxIdleConns(int(cfg.Data.Database.GetMaxIdleConnections())))
	}
	if cfg.Data.Database.MaxOpenConnections != nil {
		options = append(options, gormCrud.WithMaxOpenConns(int(cfg.Data.Database.GetMaxOpenConnections())))
	}
	if cfg.Data.Database.ConnectionMaxLifetime != nil {
		options = append(options, gormCrud.WithConnMaxLifetime(cfg.Data.Database.GetConnectionMaxLifetime().AsDuration()))
	}

	db, err := gormCrud.NewClient(options...)
	if err != nil {
		return nil, err
	}
	return db, nil
}

// helperLogger 把 kratos-bootstrap 的 *logger.Helper 适配为 go-crud 要求的
// go-wind/log.Logger(后者多一个 Enabled,且 With 返回类型不同,无法直接满足)。
// 上游两库接口对齐后可整体移除。
type helperLogger struct {
	h *bLogger.Helper
}

func (w helperLogger) Debug(ctx context.Context, msg string, keyvals ...any) {
	w.h.Debugf(ctx, "%s", sprintKV(msg, keyvals))
}

func (w helperLogger) Info(ctx context.Context, msg string, keyvals ...any) {
	w.h.Infof(ctx, "%s", sprintKV(msg, keyvals))
}

func (w helperLogger) Warn(ctx context.Context, msg string, keyvals ...any) {
	w.h.Warnf(ctx, "%s", sprintKV(msg, keyvals))
}

func (w helperLogger) Error(ctx context.Context, msg string, keyvals ...any) {
	w.h.Errorf(ctx, "%s", sprintKV(msg, keyvals))
}

// Enabled 始终放行:bLogger.Helper 无级别查询,过滤交由底层日志器。
func (w helperLogger) Enabled(windLog.Level) bool { return true }

func (w helperLogger) With(keyvals ...any) windLog.Logger {
	return helperLogger{h: &bLogger.Helper{Logger: w.h.With(keyvals...)}}
}

func sprintKV(msg string, keyvals []any) string {
	if len(keyvals) == 0 {
		return msg
	}
	return fmt.Sprint(append([]any{msg}, keyvals...)...)
}
