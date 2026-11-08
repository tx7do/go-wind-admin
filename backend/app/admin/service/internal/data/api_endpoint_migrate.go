package data

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
)

// API 授权绑定键从 api_id 迁移到 (path, method) 快照的一次性数据迁移。
//
// 背景：sys_permission_apis 旧版按 api_id 绑定，而「接口同步」曾以 Truncate +
// 顺序重编号的方式全量重建 sys_apis——每次同步所有 API id 洗牌，绑定随之漂移
// （授权悄悄换到别的端点 / 失效）。新版绑定存 (path, method) 快照，与 authorizer
// 策略、租户闸门的裁决键同形，天然免疫端点删除重建。
//
// 两个入口（均由 ent_client.go 的建库回调按序调用，仅当 migrate 开启时执行）：
//  1. DedupeSysApisByIdentity —— ent Schema.Create **之前**跑：老唯一键是
//     (module, path, method, scope)，同一路径+方法在不同 module 下可重复；
//     新唯一键 (path, method) 建索引时会被脏数据卡死启动，先按同键去重（保留最小 id）。
//  2. MigratePermissionApiEndpoints —— Schema.Create **之后**跑：此时 path/method
//     列已建、旧 api_id 列尚在（ent 自动迁移从不删列），按 api_id→sys_apis 回填
//     绑定快照，随后尽力摘除旧列与旧索引。
//
// 除预去重外全部 best-effort：失败只打 WARN 不阻断启动（温和迁移原则，见
// security-fail-closed-breaks-dev-flow 的既定裁决——迁移挂掉会断开发测试链路）。
// 历史绑定可能已被历次重编号漂污，回填只保底不修复真相；上线后在权限管理页
// 核对一遍（重新勾选保存即以当前端点为准自愈）。

// endpointDialect 标识目标数据库方言（drv.Dialect() 的返回值）。
type endpointDialect string

const (
	dialectPostgres endpointDialect = "postgres"
	dialectSQLite   endpointDialect = "sqlite3"
	dialectMySQL    endpointDialect = "mysql"
)

// DedupeSysApisByIdentity 预去重：同 (path, method) 的 sys_apis 行只保留最小 id。
// 必须在新唯一索引创建前执行，否则索引建立失败会 panic 掉整个启动。
func DedupeSysApisByIdentity(ctx context.Context, db *sql.DB, dialect string) (removed int64, err error) {
	if db == nil {
		return 0, nil
	}
	// NULL/空身份行不是任何端点，不参与去重（GROUP BY 会把 NULL 归成一组，
	// 直接对全表 NOT IN 会把它们误删）。
	const q = `DELETE FROM sys_apis
WHERE path IS NOT NULL AND path <> '' AND method IS NOT NULL AND method <> ''
  AND id NOT IN (
    SELECT MIN(id) FROM sys_apis
    WHERE path IS NOT NULL AND path <> '' AND method IS NOT NULL AND method <> ''
    GROUP BY path, method
  )`

	res, err := db.ExecContext(ctx, q)
	if err != nil {
		return 0, fmt.Errorf("dedupe sys_apis by (path,method): %w", err)
	}
	removed, _ = res.RowsAffected()
	return removed, nil
}

// MigratePermissionApiEndpoints 回填 sys_permission_apis 的 (path, method) 快照
// 并摘除旧 api_id 列与旧索引。幂等：无待回填行且旧列已摘时零操作。
func MigratePermissionApiEndpoints(ctx context.Context, db *sql.DB, dialect string) error {
	if db == nil {
		return nil
	}

	// —— 1. 旧 api_id 列还在才需要回填；不在（全新安装或已迁移）直接跳过 ——
	hasLegacy, err := columnExists(ctx, db, dialect, "sys_permission_apis", "api_id")
	if err != nil {
		return fmt.Errorf("probe legacy api_id column: %w", err)
	}
	if hasLegacy {
		if err = backfillPermissionApiEndpoints(ctx, db, dialect); err != nil {
			return err
		}
		if err = dropLegacyPermissionApiColumn(ctx, db, dialect); err != nil {
			return err
		}
	}

	// —— 2. 摘除 ent 自动迁移不会删的历史索引（旧列已随上面摘除时其索引一并消失，
	//       但 PG 的 DROP INDEX IF EXISTS 幂等，重复执行无害）——
	for _, idx := range []string{
		"idx_sys_api_res_module_path_method_scope", // 旧唯一键 (module,path,method,scope)
		"idx_sys_api_res_path_method",              // 旧非唯一 (path,method)，新唯一索引已换名
		"uix_perm_api_permission_api_id",           // 旧绑定唯一键 (permission_id,api_id)
		"idx_perm_api_api_id",                      // 旧绑定单列索引
	} {
		if err = dropIndexIfExists(ctx, db, dialect, idx); err != nil {
			return err
		}
	}

	return nil
}

// backfillPermissionApiEndpoints 按 api_id→sys_apis 把旧绑定回填成端点快照；
// api_id 已不在 sys_apis 的孤儿绑定（历史重编号漂污的残留）直接删除。
func backfillPermissionApiEndpoints(ctx context.Context, db *sql.DB, dialect string) error {
	// 占位符方言：PG 驱动（lib/pq / pgx stdlib）只认 $N，不认问号
	ph := func(i int) string {
		if endpointDialect(dialect) == dialectPostgres {
			return "$" + strconv.Itoa(i)
		}
		return "?"
	}

	// 可回填：绑定行 + 其 api_id 在 sys_apis 有主
	rows, err := db.QueryContext(ctx, `
SELECT pa.id, a.path, a.method
FROM sys_permission_apis pa
JOIN sys_apis a ON pa.api_id = a.id
WHERE pa.path IS NULL OR pa.path = '' OR pa.method IS NULL OR pa.method = ''`)
	if err != nil {
		return fmt.Errorf("query bindings to backfill: %w", err)
	}

	type pair struct {
		id           int64
		path, method string
	}
	var todo []pair
	for rows.Next() {
		var p pair
		if err = rows.Scan(&p.id, &p.path, &p.method); err != nil {
			_ = rows.Close()
			return fmt.Errorf("scan binding row: %w", err)
		}
		todo = append(todo, p)
	}
	if err = rows.Err(); err != nil {
		_ = rows.Close()
		return fmt.Errorf("iterate bindings: %w", err)
	}
	_ = rows.Close()

	for _, p := range todo {
		if _, err = db.ExecContext(ctx,
			`UPDATE sys_permission_apis SET path = `+ph(1)+`, method = `+ph(2)+` WHERE id = `+ph(3),
			p.path, p.method, p.id,
		); err != nil {
			return fmt.Errorf("backfill binding id=%d: %w", p.id, err)
		}
	}

	// 孤儿：api_id 无主的旧绑定，快照无从谈起，删掉（授权语义上它们早已死条目）
	res, err := db.ExecContext(ctx, `
DELETE FROM sys_permission_apis
WHERE (path IS NULL OR path = '')
  AND api_id IS NOT NULL
  AND NOT EXISTS (SELECT 1 FROM sys_apis a WHERE a.id = sys_permission_apis.api_id)`)
	if err != nil {
		return fmt.Errorf("delete orphan bindings: %w", err)
	}
	_ = res // 孤儿删除结果仅计数量；数量本身不值得单独 WARN（回填主体已完成）

	return nil
}

// dropLegacyPermissionApiColumn 摘除旧 api_id 列。SQLite 必须先摘引用它的索引
// （否则 DROP COLUMN 报错），PG 的 DROP COLUMN 会自动连带索引，先 DROP INDEX 也幂等。
func dropLegacyPermissionApiColumn(ctx context.Context, db *sql.DB, dialect string) error {
	for _, idx := range []string{"uix_perm_api_permission_api_id", "idx_perm_api_api_id"} {
		if _, err := db.ExecContext(ctx, "DROP INDEX IF EXISTS "+idx); err != nil {
			// MySQL 不支持 DROP INDEX IF EXISTS 等方言差异：摘列失败留待人工，
			// 不阻断启动
			return fmt.Errorf("drop legacy index %s: %w", idx, err)
		}
	}

	switch endpointDialect(dialect) {
	case dialectPostgres:
		if _, err := db.ExecContext(ctx, `ALTER TABLE sys_permission_apis DROP COLUMN IF EXISTS api_id`); err != nil {
			return fmt.Errorf("drop legacy api_id column: %w", err)
		}
	case dialectSQLite:
		// SQLite 无 IF EXISTS 形态，列存在性已在调用方确认过
		if _, err := db.ExecContext(ctx, `ALTER TABLE sys_permission_apis DROP COLUMN api_id`); err != nil {
			return fmt.Errorf("drop legacy api_id column: %w", err)
		}
	case dialectMySQL:
		// MySQL 8 的 DROP COLUMN 不支持 IF EXISTS：能走到这里说明列在，直接摘
		if _, err := db.ExecContext(ctx, `ALTER TABLE sys_permission_apis DROP COLUMN api_id`); err != nil {
			return fmt.Errorf("drop legacy api_id column: %w", err)
		}
	}
	return nil
}

// dropIndexIfExists 尽力摘除历史遗留索引；方言差异（如 MySQL 不认 IF EXISTS）
// 只 WARN 不阻断。
func dropIndexIfExists(ctx context.Context, db *sql.DB, dialect string, name string) error {
	if _, err := db.ExecContext(ctx, "DROP INDEX IF EXISTS "+name); err != nil {
		// MySQL 上失败属预期（语法不支持），其余方言失败也不值得断启动
		_ = dialect
		return nil
	}
	return nil
}

// columnExists 探测列是否存在（PG / SQLite / MySQL 三方言）。
func columnExists(ctx context.Context, db *sql.DB, dialect string, table, column string) (bool, error) {
	switch endpointDialect(dialect) {
	case dialectPostgres:
		const q = `SELECT 1 FROM information_schema.columns
WHERE table_schema = current_schema() AND table_name = $1 AND column_name = $2`
		var one int
		err := db.QueryRowContext(ctx, q, table, column).Scan(&one)
		if err == sql.ErrNoRows {
			return false, nil
		}
		return err == nil, err

	case dialectSQLite:
		// pragma 表值函数接受绑定参数（SQLite ≥3.16）
		const q = `SELECT 1 FROM pragma_table_info(?) WHERE name = ?`
		var one int
		err := db.QueryRowContext(ctx, q, table, column).Scan(&one)
		if err == sql.ErrNoRows {
			return false, nil
		}
		return err == nil, err

	case dialectMySQL:
		const q = `SELECT 1 FROM information_schema.columns
WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?`
		var one int
		err := db.QueryRowContext(ctx, q, table, column).Scan(&one)
		if err == sql.ErrNoRows {
			return false, nil
		}
		return err == nil, err
	}
	return false, fmt.Errorf("unsupported dialect: %s", dialect)
}
