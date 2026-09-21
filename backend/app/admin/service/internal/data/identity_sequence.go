package data

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
)

// 显式 ID 写入的表：菜单与接口的种子/重建都自带 id（sys_menus 1..74、sys_apis 1..N），
// 而 PostgreSQL 的 identity 序列不会因显式 ID 插入而前进，于是序列停在 1。
const (
	identityTableMenus = "sys_menus"
	identityTableApis  = "sys_apis"
)

// alignIdentitySequenceSQL 把 <table>_id_seq 推到 max(id)+1，只前进不回退。
//
// 第二项是序列自己该到的位置：PG 里 is_called=true 表示 last_value 已发放、下一个值
// 是 last_value+1；is_called=false（含本语句刚 setval 过的状态）下一个值就是
// last_value 本身。少了这个分支，每次重启都会白烧一个 id。
//
// 保住序列进度而不是只看 max(id)：菜单被硬删后 max(id) 会回落，只按 max(id)+1 置值
// 会把已发放过的 id 重新发出去，让 sys_permission_menus 里尚未清理的引用指向一行新菜单。
//
// %[1]s 只接受本文件的常量表名，永不拼接用户输入。
const alignIdentitySequenceSQL = `SELECT setval(
    '%[1]s_id_seq',
    GREATEST(
        (SELECT COALESCE(MAX(id), 0) + 1 FROM "%[1]s"),
        (SELECT last_value + CASE WHEN is_called THEN 1 ELSE 0 END FROM "%[1]s_id_seq")
    ),
    false
)`

// isPostgresDriver 判定是否需要序列对齐。MySQL / SQLite 的自增列在插入显式更大
// ID 时会自行抬高，只有 PG 的 identity/serial 会留在原地。
// ent 支持 "postgres"(lib/pq) 与 "pgx"(jackc/pgx) 两种方言，配置里写哪个都算 PG。
func isPostgresDriver(driver string) bool {
	return driver == "postgres" || driver == "pgx"
}

// driverNameOf 读取配置里的数据库驱动名（与 server_monitor_repo 同一来源）。
func driverNameOf(ctx *bootstrap.Context) string {
	if ctx == nil {
		return ""
	}
	cfg := ctx.GetConfig()
	if cfg == nil || cfg.Data == nil || cfg.Data.Database == nil {
		return ""
	}
	return cfg.Data.Database.GetDriver()
}

// alignIdentitySequence 把一张表的 id 序列对齐到"下一个未使用的值"。
//
// 幂等：重复执行结果相同。非 PG 驱动为 no-op。失败只返回错误由调用方记录——
// 这是一次启动期自愈，不该阻断服务启动。
//
// 多副本部署下两个实例同时执行，各自算出的目标值都 ≥ max(id)+1 且都只会推进，
// 最坏情况写入同一个值，不会倒退。
func alignIdentitySequence(ctx context.Context, db *sql.DB, driver, table string) error {
	if !isPostgresDriver(driver) {
		return nil
	}
	if db == nil {
		return fmt.Errorf("align identity sequence for %s: database handle is nil", table)
	}

	if _, err := db.ExecContext(ctx, fmt.Sprintf(alignIdentitySequenceSQL, table)); err != nil {
		return fmt.Errorf("align identity sequence for %s: %w", table, err)
	}

	return nil
}

// AlignIdentitySequence 对齐 sys_menus 的 id 序列。
// 种子（显式 ID）跑完之后调用，否则「新建菜单」与菜单同步的新增分支必撞 sys_menus_pkey。
func (r *MenuRepo) AlignIdentitySequence(ctx context.Context) error {
	return alignIdentitySequence(ctx, r.entClient.DB(), r.driverName, identityTableMenus)
}

// AlignIdentitySequence 对齐 sys_apis 的 id 序列。
// 接口同步是 truncate + 按显式 ID 重建，每跑一次都要重新对齐。
func (r *ApiRepo) AlignIdentitySequence(ctx context.Context) error {
	return alignIdentitySequence(ctx, r.entClient.DB(), r.driverName, identityTableApis)
}
