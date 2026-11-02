// 等保口令策略（历史口令检查）的测试。
//
// 两层各盯各的：
//   - appendPasswordHistory 纯函数：历史列表的追加/截断/异键保留语义——
//     extra_info 是 JSONB 里的共享地盘，写坏一个键就可能毁掉同槽位的其他业务数据；
//   - checkPasswordHistory（白盒，走 ConfigRepo + bcrypt）：新口令命中历史即拒、
//     各类"历史不可用"形态一律跳过不阻塞主流程（历史检查是增强项，不是单点故障源）。
package data

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"

	"go-wind-admin/pkg/constants"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/enttest"
	"go-wind-admin/app/admin/service/internal/data/ent/sysconfig"
	"go-wind-admin/app/admin/service/internal/data/ent/usercredential"
)

// passwordHistoryExtra 组装一段含历史列表的 extra_info JSON。
func passwordHistoryExtra(hashes ...string) *string {
	items := make([]any, 0, len(hashes))
	for _, h := range hashes {
		items = append(items, h)
	}
	return extraInfoJSON(map[string]any{"password_history": items})
}

// extraInfoJSON 把 map 序列化成 extra_info 字符串指针。
func extraInfoJSON(m map[string]any) *string {
	b, err := json.Marshal(m)
	if err != nil {
		panic(err) // 测试夹具是纯数据 map，marshal 失败即测试代码自身错误
	}
	s := string(b)
	return &s
}

// mustParseExtra 把 extra_info 字符串解析回 map（断言合法性）。
func mustParseExtra(tb testing.TB, s string) map[string]any {
	tb.Helper()
	var m map[string]any
	require.NoError(tb, json.Unmarshal([]byte(s), &m), "extra_info 应是合法 JSON")
	return m
}

// TestAppendPasswordHistory_KeepRecentN 验证"只保留最近 limit 条"：
// 超出后裁掉最旧的（FIFO），新口令哈希总在末尾。
func TestAppendPasswordHistory_KeepRecentN(t *testing.T) {
	out := appendPasswordHistory(
		passwordHistoryExtra("h1", "h2", "h3"),
		"h4", 3,
	)
	require.NotNil(t, out)
	extra := mustParseExtra(t, *out)
	require.Equal(t, []any{"h2", "h3", "h4"}, extra["password_history"],
		"超过 limit 应裁掉最旧的 h1，新哈希追加在末尾")
}

// TestAppendPasswordHistory_EmptyExtra 验证从零建历史：
// 原本没有 extra_info（首次改密）时新建结构、只含本次旧哈希。
func TestAppendPasswordHistory_EmptyExtra(t *testing.T) {
	out := appendPasswordHistory(nil, "h1", 3)
	require.NotNil(t, out)
	extra := mustParseExtra(t, *out)
	require.Equal(t, []any{"h1"}, extra["password_history"])

	// 空 extra_info 字符串同义
	out = appendPasswordHistory(trans.Ptr(""), "h2", 3)
	require.NotNil(t, out)
	extra = mustParseExtra(t, *out)
	require.Equal(t, []any{"h2"}, extra["password_history"])
}

// TestAppendPasswordHistory_PreservesOtherKeys 验证异键保留：
// extra_info 里已有其他业务键（如 mfa 元数据）时，历史追加不得破坏它们。
func TestAppendPasswordHistory_PreservesOtherKeys(t *testing.T) {
	in := extraInfoJSON(map[string]any{
		"password_history": []any{"h1"},
		"other_business":   "keep-me",
	})
	out := appendPasswordHistory(in, "h2", 3)
	require.NotNil(t, out)
	extra := mustParseExtra(t, *out)
	require.Equal(t, "keep-me", extra["other_business"], "无关键应原样保留")
	require.Equal(t, []any{"h1", "h2"}, extra["password_history"])
}

// TestAppendPasswordHistory_CorruptedExtra 验证解析失败兜底：
// extra_info 非历史格式（存了别的业务数据/坏 JSON）时以新结构重建——
// 历史让位，但不 panic、不阻塞改密主流程。
func TestAppendPasswordHistory_CorruptedExtra(t *testing.T) {
	out := appendPasswordHistory(trans.Ptr("not-a-json"), "h1", 3)
	require.NotNil(t, out)
	extra := mustParseExtra(t, *out)
	require.Equal(t, []any{"h1"}, extra["password_history"])
}

// TestAppendPasswordHistory_Disabled 验证 limit<=0 关闭：原样返回，不动 extra_info。
func TestAppendPasswordHistory_Disabled(t *testing.T) {
	in := passwordHistoryExtra("h1")
	require.Same(t, in, appendPasswordHistory(in, "h2", 0),
		"limit<=0 应原样返回（历史检查已关闭）")
	require.Same(t, in, appendPasswordHistory(in, "h2", -1))
}

// TestAppendPasswordHistory_SkipsGarbageEntries 验证脏条目容错：
// 历史数组里的非字符串/空串条目在重建时被剔除，不污染后续比对。
func TestAppendPasswordHistory_SkipsGarbageEntries(t *testing.T) {
	in := extraInfoJSON(map[string]any{
		"password_history": []any{"h1", 42, nil, "", "h2"},
	})
	out := appendPasswordHistory(in, "h3", 10)
	require.NotNil(t, out)
	extra := mustParseExtra(t, *out)
	require.Equal(t, []any{"h1", "h2", "h3"}, extra["password_history"],
		"非字符串与空串条目应被剔除")
}

// TestCheckPasswordHistory_RejectsReusedPassword 走真实 bcrypt 验证重用拒绝：
// 历史里存着旧口令的 bcrypt 哈希，新明文与其相同即拒（等保核心语义）。
func TestCheckPasswordHistory_RejectsReusedPassword(t *testing.T) {
	repo := newPasswordPolicyRepoForTest(t)
	ctx := enttest.NewSystemContext(context.Background())

	crypto := NewPasswordCrypto()
	oldHash, err := crypto.Encrypt("OldPass@123")
	require.NoError(t, err, "构造历史哈希失败")

	passwordType := usercredential.CredentialTypePasswordHash
	entity := &ent.UserCredential{
		CredentialType: &passwordType,
		ExtraInfo:      passwordHistoryExtra(oldHash),
	}

	err = repo.checkPasswordHistory(ctx, entity, "OldPass@123")
	require.Error(t, err, "新口令与历史口令相同应被拒绝")
	require.Contains(t, err.Error(), "must differ from recent passwords")

	// 换一个不同的新口令：放行
	require.NoError(t, repo.checkPasswordHistory(ctx, entity, "NewPass@456"),
		"不同的新口令应放行")
}

// TestCheckPasswordHistory_SkipsUnavailableHistory 验证各类"历史不可用"形态：
// 实体缺失 / 类型缺失 / 非口令类型凭证 / extra_info 为空 / 坏 JSON / 历史列表为空——
// 一律跳过（返回 nil），历史检查阻塞不了改密主流程。
func TestCheckPasswordHistory_SkipsUnavailableHistory(t *testing.T) {
	repo := newPasswordPolicyRepoForTest(t)
	ctx := enttest.NewSystemContext(context.Background())
	passwordType := usercredential.CredentialTypePasswordHash

	cases := []struct {
		name   string
		entity *ent.UserCredential
	}{
		{"实体为 nil", nil},
		{"credential_type 缺失", &ent.UserCredential{ExtraInfo: passwordHistoryExtra("h")}},
		{"非口令类型凭证", &ent.UserCredential{
			CredentialType: trans.Ptr(usercredential.CredentialTypeOauthToken),
			ExtraInfo:      passwordHistoryExtra("h"),
		}},
		{"extra_info 为空", &ent.UserCredential{CredentialType: &passwordType}},
		{"extra_info 坏 JSON", &ent.UserCredential{
			CredentialType: &passwordType,
			ExtraInfo:      trans.Ptr("<<corrupted>>"),
		}},
		{"历史列表为空", &ent.UserCredential{
			CredentialType: &passwordType,
			ExtraInfo:      passwordHistoryExtra(),
		}},
	}
	for _, tc := range cases {
		require.NoError(t, repo.checkPasswordHistory(ctx, tc.entity, "Whatever@123"),
			"形态[%s]应跳过检查而非报错", tc.name)
	}
}

// TestCheckPasswordHistory_DisabledByConfig 验证平台参数关闭路径：
// sys_config 里 historyCount=0 时跳过检查（参数管理页可调，<=0 关闭）。
// getCachedEntry 带缓存，播种必须在首次读取之前（本测试 fresh 建库先行播种）。
func TestCheckPasswordHistory_DisabledByConfig(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	ctx := enttest.NewSystemContext(context.Background())

	// 播种 historyCount=0（int 型参数）
	vt := sysconfig.ValueTypeInt
	err = entClient.Client().SysConfig.Create().
		SetKey(constants.ConfigKeyPasswordHistoryCount).
		SetName("口令历史保留条数").
		SetValue("0").
		SetNillableValueType(&vt).
		Exec(ctx)
	require.NoError(t, err, "播种 historyCount=0 失败")

	repo := NewUserCredentialRepoForTest(entClient, NewPasswordCrypto(), NewConfigRepoForTest(entClient, rdb))

	passwordType := usercredential.CredentialTypePasswordHash
	entity := &ent.UserCredential{
		CredentialType: &passwordType,
		ExtraInfo:      passwordHistoryExtra("anything"),
	}
	require.NoError(t, repo.checkPasswordHistory(ctx, entity, "Whatever@123"),
		"historyCount=0 时应跳过历史检查")
}

// --- helpers ---

// newPasswordPolicyRepoForTest 组装 checkPasswordHistory 依赖的真实仓储
// （enttest SQLite + miniredis 版 ConfigRepo + bcrypt）。
func newPasswordPolicyRepoForTest(t *testing.T) *UserCredentialRepo {
	t.Helper()
	entClient := enttest.NewEntClientForTest(t)

	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	return NewUserCredentialRepoForTest(entClient, NewPasswordCrypto(), NewConfigRepoForTest(entClient, rdb))
}
