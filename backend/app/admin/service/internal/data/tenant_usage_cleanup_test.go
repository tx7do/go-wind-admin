package data

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

// expectedTenantCleanupTables 是 ent.Client/Tx 中全部拥有 TenantIDEQ 谓词的
// 实体（ent 客户端字段名）。CleanupTenantData 的 deleteFns 必须逐一覆盖：
// 租户清理漏一张表 = 该表数据跨租户残留（套餐清理欠账曾因此少删 4 张表，
// 后来 AI/BI/打印模块落地又静默涨到 14 张）。功能语义（删数据/保租户/置 OFF/
// 不误删他租户）由 TestTenantUsageRepoSqlite_CleanupTenantData 覆盖；
// 本测试钉"表清单完整性"这个静态性质——新增带 tenant_id 的表时，把新表同时
// 补进 deleteFns 和这里，否则本测试失败。
var expectedTenantCleanupTables = []string{
	"AccessKey",
	"AiAgentApp", "AiAgentRun", "AiChunk", "AiConversation", "AiDoc",
	"AiKnowledgeBase", "AiMessage", "AiUsageLog",
	"ApiAuditLog", "BiDashboard", "BiDashboardRevision", "DataAccessAuditLog",
	"DictEntry", "DictEntryI18n", "DictType", "File",
	"InternalMessage", "InternalMessageCategory", "InternalMessageRecipient",
	"LoginAuditLog", "LoginPolicy",
	"Membership", "MembershipOrgUnit", "MembershipPosition", "MembershipRole",
	"OperationAuditLog", "OrgUnit", "PermissionAuditLog", "PolicyEvaluationLog",
	"Position", "PrintTemplate",
	"Role", "RoleFieldPermission", "RoleMetadata", "RoleOrgUnit", "RolePermission",
	"Task", "User", "UserCredential", "UserMfaFactor", "UserOrgUnit",
	"UserPosition", "UserRole",
}

// TestTenantCleanupCoversAllTenantTables 源扫描守卫：从 tenant_usage_repo.go
// 提取 CleanupTenantData 里全部 tx.<X>.Delete() 调用，与带 tenant_id 的实体
// 全集比对，漏一张即失败。
func TestTenantCleanupCoversAllTenantTables(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller 应能取到本测试文件路径")
	repoFile := filepath.Join(filepath.Dir(thisFile), "tenant_usage_repo.go")
	src, err := os.ReadFile(repoFile)
	require.NoError(t, err, "应能读取 tenant_usage_repo.go 源码")

	re := regexp.MustCompile(`tx\.(\w+)\.Delete\(\)`)
	matches := re.FindAllStringSubmatch(string(src), -1)
	require.NotEmpty(t, matches, "应能在源码中匹配到 tx.<X>.Delete() 调用")

	got := make([]string, 0, len(matches))
	for _, m := range matches {
		got = append(got, m[1])
	}
	sort.Strings(got)
	got = uniqueStrings(got)

	want := append([]string(nil), expectedTenantCleanupTables...)
	sort.Strings(want)

	require.ElementsMatch(t, want, got,
		"CleanupTenantData 的删表清单与带 tenant_id 的实体全集不一致——"+
			"新增带 tenant_id 的表必须同步进 deleteFns 与 expectedTenantCleanupTables")
}

func uniqueStrings(in []string) []string {
	out := in[:0]
	var prev string
	for i, s := range in {
		if i == 0 || s != prev {
			out = append(out, s)
		}
		prev = s
	}
	return out
}
