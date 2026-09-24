package data

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newAuditLogArchiveRepoSqlite 用 enttest helper 构造一个可直接做归档的
// AuditLogArchiveRepo，逐字段复刻 NewAuditLogArchiveRepo（log 换 NopLogger，
// client 取测试 client 的底层 ent client；该 repo 无 init()）。
func newAuditLogArchiveRepoSqlite(t *testing.T) *AuditLogArchiveRepo {
	t.Helper()
	entClient := enttest.NewEntClientForTest(t)
	return &AuditLogArchiveRepo{
		log:    bLogger.NewHelper(bLogger.NopLogger()),
		client: entClient.Client(),
	}
}

// seedOneRowPerAuditTable 往六张审计表各直插一行（created_at 由 ent 默认取当前时间），
// 返回各表直插后的行数（应全为 1）。
func seedOneRowPerAuditTable(t *testing.T, client *ent.Client, ctx context.Context, suffix string) {
	t.Helper()
	require.NoError(t, client.ApiAuditLog.Create().
		SetCreatedAt(time.Now()).
		SetNillableUsername(trans.Ptr("archive_user_"+suffix)).
		Exec(ctx), "直插 sys_api_audit_logs 应成功")
	require.NoError(t, client.LoginAuditLog.Create().
		SetCreatedAt(time.Now()).
		SetNillableUsername(trans.Ptr("archive_user_"+suffix)).
		Exec(ctx), "直插 sys_login_audit_logs 应成功")
	require.NoError(t, client.OperationAuditLog.Create().
		SetCreatedAt(time.Now()).
		SetNillableResourceType(trans.Ptr("archive_res_"+suffix)).
		Exec(ctx), "直插 sys_operation_audit_logs 应成功")
	require.NoError(t, client.PermissionAuditLog.Create().
		SetCreatedAt(time.Now()).
		SetIPAddress("10.9.9.9").
		SetRequestID("req-archive-"+suffix).
		SetReason("archive reason "+suffix).
		SetNillableTargetType(trans.Ptr("archive_tgt_"+suffix)).
		Exec(ctx), "直插 sys_permission_audit_logs 应成功")
	require.NoError(t, client.DataAccessAuditLog.Create().
		SetCreatedAt(time.Now()).
		SetNillableDataSource(trans.Ptr("archive_ds_"+suffix)).
		Exec(ctx), "直插 sys_data_access_audit_logs 应成功")
	require.NoError(t, client.PolicyEvaluationLog.Create().
		SetCreatedAt(time.Now()).
		SetUserID(1).
		SetMembershipID(1).
		SetPermissionID(1).
		SetIPAddress("10.9.9.9").
		SetNillableRequestPath(trans.Ptr("/archive/"+suffix)).
		Exec(ctx), "直插 sys_policy_evaluation_logs 应成功")
}

// auditTableCounts 直查六张审计表的当前行数。
func auditTableCounts(t *testing.T, client *ent.Client, ctx context.Context) map[string]int {
	t.Helper()
	counts := map[string]int{}
	var err error
	counts["sys_api_audit_logs"], err = client.ApiAuditLog.Query().Count(ctx)
	require.NoError(t, err)
	counts["sys_login_audit_logs"], err = client.LoginAuditLog.Query().Count(ctx)
	require.NoError(t, err)
	counts["sys_operation_audit_logs"], err = client.OperationAuditLog.Query().Count(ctx)
	require.NoError(t, err)
	counts["sys_permission_audit_logs"], err = client.PermissionAuditLog.Query().Count(ctx)
	require.NoError(t, err)
	counts["sys_data_access_audit_logs"], err = client.DataAccessAuditLog.Query().Count(ctx)
	require.NoError(t, err)
	counts["sys_policy_evaluation_logs"], err = client.PolicyEvaluationLog.Query().Count(ctx)
	require.NoError(t, err)
	return counts
}

// TestAuditLogArchiveRepoSqlite_ArchiveExpiredBeforeNow 验证：
// before 取未来时间时，六张审计表的行（created_at 为当前时间，均早于 before）
// 全部导出为 JSONL 后从库内删除，返回各表归档行数；落盘文件逐表一份、
// 行数与导出一致；库内六表计数归零。
func TestAuditLogArchiveRepoSqlite_ArchiveExpiredBeforeNow(t *testing.T) {
	repo := newAuditLogArchiveRepoSqlite(t)
	client := repo.client
	ctx := enttest.NewSystemViewerCtx(context.Background())

	seedOneRowPerAuditTable(t, client, ctx, "future")
	require.Equal(t, map[string]int{
		"sys_api_audit_logs":         1,
		"sys_login_audit_logs":       1,
		"sys_operation_audit_logs":   1,
		"sys_permission_audit_logs":  1,
		"sys_data_access_audit_logs": 1,
		"sys_policy_evaluation_logs": 1,
	}, auditTableCounts(t, client, ctx), "六张审计表应各含 1 行")

	outDir := filepath.Join(t.TempDir(), "archive")
	results, err := repo.ArchiveExpired(ctx, time.Now().Add(1*time.Hour), outDir)
	require.NoError(t, err, "归档应成功")
	require.NoError(t, err, "归档应成功")
	require.Equal(t, map[string]int{
		"sys_api_audit_logs":         1,
		"sys_login_audit_logs":       1,
		"sys_operation_audit_logs":   1,
		"sys_permission_audit_logs":  1,
		"sys_data_access_audit_logs": 1,
		"sys_policy_evaluation_logs": 1,
	}, results, "六张表各归档 1 行")

	require.Equal(t, map[string]int{
		"sys_api_audit_logs":         0,
		"sys_login_audit_logs":       0,
		"sys_operation_audit_logs":   0,
		"sys_permission_audit_logs":  0,
		"sys_data_access_audit_logs": 0,
		"sys_policy_evaluation_logs": 0,
	}, auditTableCounts(t, client, ctx), "归档后六张表计数应归零")

	// 落盘校验：每张表一份 JSONL，各含 1 行合法 JSON。
	// 文件名为 <表名>-<stamp>.jsonl（stamp 形如 20060102-150405，自带连字符）。
	entries, err := os.ReadDir(outDir)
	require.NoError(t, err)
	files := map[string]string{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		files[e.Name()] = filepath.Join(outDir, e.Name())
	}
	require.Len(t, files, 6, "应落盘 6 份 JSONL 文件")
	tables := map[string]bool{}
	for name, path := range files {
		table := ""
		for _, tbl := range []string{
			"sys_api_audit_logs", "sys_login_audit_logs", "sys_operation_audit_logs",
			"sys_permission_audit_logs", "sys_data_access_audit_logs", "sys_policy_evaluation_logs",
		} {
			if strings.HasPrefix(name, tbl+"-") {
				table = tbl
				break
			}
		}
		require.NotEmpty(t, table, "文件 %s 应以六张审计表之一的表名为前缀", name)
		tables[table] = true
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
		require.Len(t, lines, 1, "文件 %s 应恰含 1 行（导出行数与删除行数一致）", name)
		require.True(t, json.Valid([]byte(lines[0])), "归档行应为合法 JSON")
	}
	require.Len(t, tables, 6, "六张审计表应各落盘一份")
}

// TestAuditLogArchiveRepoSqlite_ArchiveExpiredAfterNow 验证：
// before 取过去时间时，无行满足 created_at < before，
// 不归档、不落盘、库内行保留，返回空 map。
func TestAuditLogArchiveRepoSqlite_ArchiveExpiredAfterNow(t *testing.T) {
	repo := newAuditLogArchiveRepoSqlite(t)
	client := repo.client
	ctx := enttest.NewSystemViewerCtx(context.Background())

	seedOneRowPerAuditTable(t, client, ctx, "past")

	outDir := filepath.Join(t.TempDir(), "archive")
	results, err := repo.ArchiveExpired(ctx, time.Now().Add(-24*time.Hour), outDir)
	require.NoError(t, err, "空跑归档应成功")
	require.Empty(t, results, "过去时间阈值应无归档行")

	require.Equal(t, map[string]int{
		"sys_api_audit_logs":         1,
		"sys_login_audit_logs":       1,
		"sys_operation_audit_logs":   1,
		"sys_permission_audit_logs":  1,
		"sys_data_access_audit_logs": 1,
		"sys_policy_evaluation_logs": 1,
	}, auditTableCounts(t, client, ctx), "空跑后六张表行数应保留")

	entries, err := os.ReadDir(outDir)
	require.NoError(t, err)
	jsonlCount := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".jsonl") {
			jsonlCount++
		}
	}
	require.Zero(t, jsonlCount, "空跑不应落盘任何 JSONL 文件")
}
