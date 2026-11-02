package service

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/pkg/mailtext"
)

func digestStatsFixture() *data.AuditDigestStats {
	return &data.AuditDigestStats{
		Total:  7,
		Failed: 2,
		Users:  3,
		TopActions: []data.AuditActionCount{
			{Action: "CREATE", Count: 5},
			{Action: "DELETE", Count: 2},
		},
	}
}

// 文案表两语言都必须完整、且真实分叉（防"翻译"只是复制中文）；未识别语言与
// 未标注一样回落中文，标签按归一化命中。
func TestDigestCopyTableBothLocales(t *testing.T) {
	zh := digestCopyOf(mailtext.LocaleZhCN)
	en := digestCopyOf(mailtext.LocaleEnUS)
	for _, cp := range []digestCopy{zh, en} {
		for _, f := range []string{
			cp.title, cp.dateLine, cp.totalLine, cp.failedLine, cp.usersLine,
			cp.actionsHead, cp.actionLine, cp.plainDigest, cp.plainAction,
			cp.plainJoiner, cp.systemPrompt,
		} {
			require.NotEmpty(t, f)
		}
	}
	require.NotEqual(t, zh, en)
	// digestCopyOf 只认规范化标签：canonical 键命中，其余（未知、非规范化
	// 变体）一律回落中文。BCP-47 变体归一是 LocaleOfTag 的职责（下方单测），
	// 投递路径先归一再查表。
	require.Equal(t, zh, digestCopyOf("zh-CN"))
	require.Equal(t, en, digestCopyOf("en-US"))
	require.Equal(t, zh, digestCopyOf("xx"))
	require.Equal(t, zh, digestCopyOf("zh"))
	require.Equal(t, zh, digestCopyOf("en"))
}

func TestLocaleOfTagNormalization(t *testing.T) {
	for _, c := range []struct {
		tag  string
		want mailtext.Locale
		ok   bool
	}{
		{"zh-CN", mailtext.LocaleZhCN, true},
		{"zh", mailtext.LocaleZhCN, true},
		{"ZH-CN", mailtext.LocaleZhCN, true},
		{"en-US", mailtext.LocaleEnUS, true},
		{"en", mailtext.LocaleEnUS, true},
		{"fr", "", false},
		{"", "", false},
	} {
		got, ok := mailtext.LocaleOfTag(c.tag)
		require.Equal(t, c.ok, ok, c.tag)
		if c.ok {
			require.Equal(t, c.want, got, c.tag)
		} else {
			require.Equal(t, mailtext.LocaleZhCN, got, c.tag)
		}
	}
}

func TestDigestTitleLocalized(t *testing.T) {
	require.Equal(t, "AI 审计日报（2026-10-02）", digestTitle(mailtext.LocaleZhCN, "2026-10-02"))
	require.Equal(t, "AI Audit Digest (2026-10-02)", digestTitle(mailtext.LocaleEnUS, "2026-10-02"))
}

func TestDigestFactsLocalized(t *testing.T) {
	stats := digestStatsFixture()
	require.Equal(t,
		"日期：2026-10-02\n操作总数：7\n失败操作数：2\n涉及用户数：3\n动作分布：\n  - CREATE: 5 次\n  - DELETE: 2 次\n",
		digestFacts(mailtext.LocaleZhCN, "2026-10-02", stats))
	require.Equal(t,
		"Date: 2026-10-02\nTotal operations: 7\nFailed operations: 2\nUsers involved: 3\nAction distribution:\n  - CREATE: 5\n  - DELETE: 2\n",
		digestFacts(mailtext.LocaleEnUS, "2026-10-02", stats))
}

func TestBuildPlainDigestLocalized(t *testing.T) {
	stats := digestStatsFixture()
	require.Equal(t,
		"昨日（2026-10-02）操作审计：总数 7，失败 2，涉及用户 3。\n动作分布：CREATE 5 次，DELETE 2 次",
		buildPlainDigest(mailtext.LocaleZhCN, "2026-10-02", stats))
	require.Equal(t,
		"Yesterday (2026-10-02) operation audit: total 7, failed 2, users involved 3.\nAction distribution: CREATE 5 times, DELETE 2 times",
		buildPlainDigest(mailtext.LocaleEnUS, "2026-10-02", stats))
}
