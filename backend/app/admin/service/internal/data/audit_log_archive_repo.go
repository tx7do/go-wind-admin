package data

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	apiauditlog "go-wind-admin/app/admin/service/internal/data/ent/apiauditlog"
	"go-wind-admin/app/admin/service/internal/data/ent"
	dataaccessauditlog "go-wind-admin/app/admin/service/internal/data/ent/dataaccessauditlog"
	loginauditlog "go-wind-admin/app/admin/service/internal/data/ent/loginauditlog"
	operationauditlog "go-wind-admin/app/admin/service/internal/data/ent/operationauditlog"
	permissionauditlog "go-wind-admin/app/admin/service/internal/data/ent/permissionauditlog"
	policyevaluationlog "go-wind-admin/app/admin/service/internal/data/ent/policyevaluationlog"

	appViewer "go-wind-admin/pkg/entgo/viewer"
)

// AuditLogArchiveRepo 审计日志归档：把超过保留期的审计行导出为本地 JSONL
// 文件后删除，满足等保"日志留存 ≥6 个月"——库内保存在保留期内，超期数据
// 落盘归档（文件保留不受库删除影响），库瘦身与留痕兼得。
type AuditLogArchiveRepo struct {
	log    *bLogger.Helper
	client *ent.Client
}

func NewAuditLogArchiveRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *AuditLogArchiveRepo {
	return &AuditLogArchiveRepo{
		log:    ctx.NewLoggerHelper("audit-log-archive/repo/admin-service"),
		client: entClient.Client(),
	}
}

// archiveBatch 单批大小：控制单事务/单文件的行数上限。
const archiveBatch = 5000

// ArchiveExpired 归档所有审计表中 created_at < before 的行。
// 返回各表归档行数。导出成功才删除；单表失败跳过并记日志，不影响其他表。
func (r *AuditLogArchiveRepo) ArchiveExpired(ctx context.Context, before time.Time, outDir string) (map[string]int, error) {
	ctx = appViewer.NewSystemViewerContext(ctx)
	client := r.client
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		return nil, fmt.Errorf("create archive dir: %w", err)
	}

	results := make(map[string]int)

	type tableJob struct {
		name  string
		query func() (rows []any, ids []uint32, err error)
		del   func(ids []uint32) (int, error)
	}

	jobs := []tableJob{
		{
			name: "sys_api_audit_logs",
			query: func() ([]any, []uint32, error) {
				rows, err := client.ApiAuditLog.Query().
					Where(apiauditlog.CreatedAtLT(before)).Limit(archiveBatch).All(ctx)
				if err != nil {
					return nil, nil, err
				}
				ids := make([]uint32, 0, len(rows))
				out := make([]any, 0, len(rows))
				for _, e := range rows {
					ids = append(ids, e.ID)
					out = append(out, e)
				}
				return out, ids, nil
			},
			del: func(ids []uint32) (int, error) {
				return client.ApiAuditLog.Delete().Where(apiauditlog.IDIn(ids...)).Exec(ctx)
			},
		},
		{
			name: "sys_login_audit_logs",
			query: func() ([]any, []uint32, error) {
				rows, err := client.LoginAuditLog.Query().
					Where(loginauditlog.CreatedAtLT(before)).Limit(archiveBatch).All(ctx)
				if err != nil {
					return nil, nil, err
				}
				ids := make([]uint32, 0, len(rows))
				out := make([]any, 0, len(rows))
				for _, e := range rows {
					ids = append(ids, e.ID)
					out = append(out, e)
				}
				return out, ids, nil
			},
			del: func(ids []uint32) (int, error) {
				return client.LoginAuditLog.Delete().Where(loginauditlog.IDIn(ids...)).Exec(ctx)
			},
		},
		{
			name: "sys_operation_audit_logs",
			query: func() ([]any, []uint32, error) {
				rows, err := client.OperationAuditLog.Query().
					Where(operationauditlog.CreatedAtLT(before)).Limit(archiveBatch).All(ctx)
				if err != nil {
					return nil, nil, err
				}
				ids := make([]uint32, 0, len(rows))
				out := make([]any, 0, len(rows))
				for _, e := range rows {
					ids = append(ids, e.ID)
					out = append(out, e)
				}
				return out, ids, nil
			},
			del: func(ids []uint32) (int, error) {
				return client.OperationAuditLog.Delete().Where(operationauditlog.IDIn(ids...)).Exec(ctx)
			},
		},
		{
			name: "sys_permission_audit_logs",
			query: func() ([]any, []uint32, error) {
				rows, err := client.PermissionAuditLog.Query().
					Where(permissionauditlog.CreatedAtLT(before)).Limit(archiveBatch).All(ctx)
				if err != nil {
					return nil, nil, err
				}
				ids := make([]uint32, 0, len(rows))
				out := make([]any, 0, len(rows))
				for _, e := range rows {
					ids = append(ids, e.ID)
					out = append(out, e)
				}
				return out, ids, nil
			},
			del: func(ids []uint32) (int, error) {
				return client.PermissionAuditLog.Delete().Where(permissionauditlog.IDIn(ids...)).Exec(ctx)
			},
		},
		{
			name: "sys_data_access_audit_logs",
			query: func() ([]any, []uint32, error) {
				rows, err := client.DataAccessAuditLog.Query().
					Where(dataaccessauditlog.CreatedAtLT(before)).Limit(archiveBatch).All(ctx)
				if err != nil {
					return nil, nil, err
				}
				ids := make([]uint32, 0, len(rows))
				out := make([]any, 0, len(rows))
				for _, e := range rows {
					ids = append(ids, e.ID)
					out = append(out, e)
				}
				return out, ids, nil
			},
			del: func(ids []uint32) (int, error) {
				return client.DataAccessAuditLog.Delete().Where(dataaccessauditlog.IDIn(ids...)).Exec(ctx)
			},
		},
		{
			name: "sys_policy_evaluation_logs",
			query: func() ([]any, []uint32, error) {
				rows, err := client.PolicyEvaluationLog.Query().
					Where(policyevaluationlog.CreatedAtLT(before)).Limit(archiveBatch).All(ctx)
				if err != nil {
					return nil, nil, err
				}
				ids := make([]uint32, 0, len(rows))
				out := make([]any, 0, len(rows))
				for _, e := range rows {
					ids = append(ids, e.ID)
					out = append(out, e)
				}
				return out, ids, nil
			},
			del: func(ids []uint32) (int, error) {
				return client.PolicyEvaluationLog.Delete().Where(policyevaluationlog.IDIn(ids...)).Exec(ctx)
			},
		},
	}

	stamp := time.Now().Format("20060102-150405")
	for _, job := range jobs {
		total := 0
		for {
			rows, ids, err := job.query()
			if err != nil {
				r.log.Errorf(ctx, "archive %s: query: %s", job.name, err.Error())
				break
			}
			if len(rows) == 0 {
				break
			}

			path := filepath.Join(outDir, fmt.Sprintf("%s-%s.jsonl", job.name, stamp))
			f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o640)
			if err != nil {
				r.log.Errorf(ctx, "archive %s: open file: %s", job.name, err.Error())
				break
			}
			werr := func() error {
				defer f.Close()
				enc := json.NewEncoder(f)
				for _, row := range rows {
					if err := enc.Encode(row); err != nil {
						return err
					}
				}
				return f.Sync()
			}()
			if werr != nil {
				r.log.Errorf(ctx, "archive %s: write: %s", job.name, werr.Error())
				break
			}

			// 导出成功才删除；删除行数应与导出一致（同批 id）。
			if _, derr := job.del(ids); derr != nil {
				r.log.Errorf(ctx, "archive %s: delete: %s（文件已落盘，重跑将产生重复行）", job.name, derr.Error())
				break
			}
			total += len(rows)
			if len(rows) < archiveBatch {
				break
			}
		}
		if total > 0 {
			results[job.name] = total
		}
	}

	return results, nil
}
