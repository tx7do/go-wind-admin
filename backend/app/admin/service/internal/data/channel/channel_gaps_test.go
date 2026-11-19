// 本文件补 channel 包既有测试未覆盖的分支：
//   - sender.go 的 Registry：注册/命中/同渠道覆盖/未注册返回 (nil, false)；
//   - settings_crypto.go 的边界：空 store 解密、未知类型、坏 JSON、
//     敏感字段留空且无旧值时 Merge 的 required 兜底、Sanitize 的空串入参。
// （SendRequest/SettingsSender 的拨号路径受包级 settingsHTTPClient 的 SSRF 开关
//   在 init 时定格的限制，本文件只测不拨号路径。）
package channel

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	notificationV1 "go-wind-admin/api/gen/go/notification/service/v1"
)

// stubSender 可识别实例的 Sender 桩。
type stubSender struct {
	channel notificationV1.Channel
}

func (s *stubSender) Channel() notificationV1.Channel { return s.channel }
func (s *stubSender) Send(context.Context, *SendRequest) (*SendReceipt, error) {
	return &SendReceipt{ChannelID: 1, ProviderID: "stub"}, nil
}

func TestRegistry_RegisterAndLookup(t *testing.T) {
	r := NewRegistry()

	// 未注册：ok=false，返回 nil。
	got, ok := r.Sender(notificationV1.Channel_EMAIL)
	require.False(t, ok)
	require.Nil(t, got)

	email := &stubSender{channel: notificationV1.Channel_EMAIL}
	sms := &stubSender{channel: notificationV1.Channel_SMS}
	r.Register(email)
	r.Register(sms)

	got, ok = r.Sender(notificationV1.Channel_EMAIL)
	require.True(t, ok)
	require.Same(t, email, got)

	got, ok = r.Sender(notificationV1.Channel_SMS)
	require.True(t, ok)
	require.Same(t, sms, got)
}

func TestRegistry_SameChannelOverrides(t *testing.T) {
	r := NewRegistry()
	first := &stubSender{channel: notificationV1.Channel_WEBHOOK}
	second := &stubSender{channel: notificationV1.Channel_WEBHOOK}

	r.Register(first)
	r.Register(second)

	got, ok := r.Sender(notificationV1.Channel_WEBHOOK)
	require.True(t, ok)
	require.Same(t, second, got, "re-registering a channel must replace the previous sender")
}

// --- settings_crypto 边界 ---

func TestDecryptSettings_Edges(t *testing.T) {
	// 空 store：空 map、不报错（渠道行 settings 列允许 NULL/空串）。
	got, err := DecryptSettings("TELEGRAM", "")
	require.NoError(t, err)
	require.Empty(t, got)

	// 未知类型：报错（schema 不存在）。
	_, err = DecryptSettings("NO_SUCH_TYPE", `{"a":"b"}`)
	require.Error(t, err)

	// 坏 JSON：报错。
	_, err = DecryptSettings("TELEGRAM", "not-json")
	require.Error(t, err)
}

func TestSanitizeSettingsForDTO_Edges(t *testing.T) {
	// 空串入 → 空串出。
	require.Empty(t, SanitizeSettingsForDTO("TELEGRAM", ""))

	// 坏 JSON → 空串（不 panic）。
	require.Empty(t, SanitizeSettingsForDTO("TELEGRAM", "not-json"))

	// 未知类型 → 原样返回（schema nil 分支）。
	raw := `{"botToken":"x"}`
	require.Equal(t, raw, SanitizeSettingsForDTO("NO_SUCH_TYPE", raw))
}

func TestNormalizeSettingsForStore_Edges(t *testing.T) {
	// 未知类型。
	_, err := NormalizeSettingsForStore("NO_SUCH_TYPE", `{"a":"b"}`)
	require.Error(t, err)

	// 空串入参。
	_, err = NormalizeSettingsForStore("TELEGRAM", "")
	require.Error(t, err)

	// JSON 数组解不进 map。
	_, err = NormalizeSettingsForStore("TELEGRAM", `[1,2]`)
	require.Error(t, err)

	// required 字段纯空白被拦（TrimSpace 语义）。
	_, err = NormalizeSettingsForStore("TELEGRAM", `{"botToken":"   "}`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Bot Token")
}

func TestMergeSettingsForStore_Edges(t *testing.T) {
	// 未知类型。
	_, err := MergeSettingsForStore("NO_SUCH_TYPE", `{"botToken":"old"}`, `{"botToken":"new"}`)
	require.Error(t, err)

	// 空新值：报错。
	_, err = MergeSettingsForStore("TELEGRAM", `{"botToken":"old"}`, "")
	require.Error(t, err)

	// 敏感字段留空 + 无旧值：required 兜底报错。
	_, err = MergeSettingsForStore("TELEGRAM", "", `{"apiBase":"https://x"}`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Bot Token")

	// 敏感字段留空 + 有旧值：保留旧密文，普通字段更新。
	old, err := NormalizeSettingsForStore("TELEGRAM", `{"botToken":"123:OLD","apiBase":"https://old"}`)
	require.NoError(t, err)
	merged, err := MergeSettingsForStore("TELEGRAM", old, `{"botToken":"","apiBase":"https://new"}`)
	require.NoError(t, err)
	dec, err := DecryptSettings("TELEGRAM", merged)
	require.NoError(t, err)
	require.Equal(t, "123:OLD", dec["botToken"], "empty sensitive value must keep the stored secret")
	require.Equal(t, "https://new", dec["apiBase"])
}
