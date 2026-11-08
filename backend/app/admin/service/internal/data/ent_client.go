package data

import (
	"context"
	"fmt"

	"entgo.io/ent/dialect/sql"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/lib/pq"

	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	entBootstrap "github.com/tx7do/kratos-bootstrap/database/ent"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/migrate"
	_ "go-wind-admin/app/admin/service/internal/data/ent/runtime"
)

// NewEntClient 创建Ent ORM数据库客户端
func NewEntClient(ctx *bootstrap.Context) (*entCrud.EntClient[*ent.Client], func(), error) {
	l := ctx.NewLoggerHelper("ent/data/admin-service")

	cfg := ctx.GetConfig()
	if cfg == nil || cfg.Data == nil {
		l.Errorf(context.Background(), "[ENT] failed getting config")
		panic("[ENT] failed getting config")
	}

	cli, err := entBootstrap.NewEntClient(cfg, func(drv *sql.Driver) *ent.Client {
		client := ent.NewClient(
			ent.Driver(&auditDriver{drv}),
			ent.Log(func(a ...any) {
				l.Debug(context.Background(), fmt.Sprint(a...))
			}),
		)
		if client == nil {
			l.Errorf(context.Background(), "[ENT] failed creating ent client")
			panic("[ENT] failed creating ent client")
		}

		// run the auto migration tool
		if cfg.Data.Database.GetMigrate() {
			// API 授权绑定键迁移 (api_id → (path,method) 快照)：见 api_endpoint_migrate.go。
			// 预去重必须在 Schema.Create 之前——sys_apis 新唯一键 (path,method) 建索引
			// 会被旧键形态下的重复行卡死启动；回填摘列在 Schema.Create 之后（列已建、
			// 旧列尚在）。除预去重外均 best-effort：失败 WARN 不阻断启动。
			if db := drv.DB(); db != nil {
				if removed, dErr := DedupeSysApisByIdentity(ctx.Context(), db, drv.Dialect()); dErr != nil {
					l.Errorf(context.Background(), "[ENT] sys_apis endpoint dedupe failed: %v", dErr)
				} else if removed > 0 {
					l.Infof(context.Background(), "[ENT] sys_apis 按 (path,method) 去重 %d 行（保留每组最小 id）", removed)
				}
			}

			if err := client.Schema.Create(ctx.Context(), migrate.WithForeignKeys(true)); err != nil {
				l.Errorf(context.Background(), "[ENT] failed creating schema resources: %v", err)
				panic("[ENT] failed creating schema resources")
			}

			if db := drv.DB(); db != nil {
				if err := MigratePermissionApiEndpoints(ctx.Context(), db, drv.Dialect()); err != nil {
					// 不阻断：回填失败的绑定 path 为空，authorizer 装载时跳过、
					// 权限页重新勾选保存即自愈；旧列残留不影响新键运行。
					l.Errorf(context.Background(), "[ENT] permission api endpoint backfill failed: %v", err)
				}
			}
		}

		return client
	})
	if err != nil {
		l.Errorf(context.Background(), "[ENT] failed creating ent client: %v", err)
		panic("[ENT] failed creating ent client")
	}

	return cli, func() {
		if cleanErr := cli.Close(); cleanErr != nil {
			l.Errorf(context.Background(), "[ENT] failed closing ent client: %v", cleanErr)
		}
	}, nil
}
