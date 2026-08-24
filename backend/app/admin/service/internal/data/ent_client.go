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
			ent.Driver(drv),
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
			if err := client.Schema.Create(ctx.Context(), migrate.WithForeignKeys(true)); err != nil {
				l.Errorf(context.Background(), "[ENT] failed creating schema resources: %v", err)
				panic("[ENT] failed creating schema resources")
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
