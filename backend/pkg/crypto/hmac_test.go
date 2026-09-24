package crypto

// 本文件覆盖 HMAC 签名（hmac.go）与全局加密器初始化后的解密失败分支。
//
// 执行顺序依赖：本文件按字母序排在 payload_test.go 之前，且本文件内的
// 第一个用例必须先于本文件内的 InitGlobalEncryptor 调用执行（同文件内
// 按声明顺序运行），以覆盖"全局加密器未初始化时签名不可用"的分支。
// 若日后新增更早触发 InitGlobalEncryptor 的测试文件，请同步调整。

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSignData_UninitializedGlobalEncryptor 全局加密器未初始化
// （GOWIND_CRYPTO_KEY 未配置）时签名能力必须显式报错：签名即凭证，
// 降级为"无签名"会让签名 URL 的消费方失去校验依据；
// 校验侧同步必须整体判定失败而非放行。
func TestSignData_UninitializedGlobalEncryptor(t *testing.T) {
	requireGlobalEncryptorUnitinitialized(t)
	sig, err := SignData("payload-to-sign")
	require.Error(t, err, "未初始化时签名必须报错而非产出空签名")
	assert.Empty(t, sig)
	assert.False(t, VerifyData("payload-to-sign", "whatsoever"),
		"签名能力不可用时校验必须失败")
}

// TestSignDataVerifyData_HmacProperties 初始化后的签名/校验性质：
// 签名为小写 hex 的 HMAC-SHA256、同一输入确定性、不同输入区分、
// 校验接受自产签名且拒绝篡改签名/篡改数据/大小写变体。
func TestSignDataVerifyData_HmacProperties(t *testing.T) {
	require.NoError(t, InitGlobalEncryptor("hmac-test-key", true))

	sig1, err := SignData("payload-to-sign")
	require.NoError(t, err)
	assert.Regexp(t, `^[0-9a-f]{64}$`, sig1,
		"签名必须是 HMAC-SHA256 的小写 hex 编码（64 字符）")

	sig2, err := SignData("payload-to-sign")
	require.NoError(t, err)
	assert.Equal(t, sig1, sig2, "同一输入的签名必须确定")

	sig3, err := SignData("different-payload")
	require.NoError(t, err)
	assert.NotEqual(t, sig1, sig3, "不同输入的签名必须不同")

	assert.True(t, VerifyData("payload-to-sign", sig1), "自产签名必须校验通过")
	assert.False(t, VerifyData("tampered-payload", sig1),
		"篡改数据后必须校验失败")
	assert.False(t, VerifyData("payload-to-sign", strings.Repeat("0", 64)),
		"伪造签名必须校验失败")
	assert.False(t, VerifyData("payload-to-sign", strings.ToUpper(sig1)),
		"签名比较必须区分大小写")
	assert.False(t, VerifyData("payload-to-sign", ""),
		"空签名必须校验失败")
}

// TestDecryptPayload_EncryptedConfigUndecryptable 全局加密器已初始化时，
// 带 enc: 前缀但内容非法（非 base64）的"密文"必须走解密失败路径报错，
// 而不是被当作明文直通回传给调用方。
func TestDecryptPayload_EncryptedConfigUndecryptable(t *testing.T) {
	require.NoError(t, InitGlobalEncryptor("hmac-test-key", true))

	payload := map[string]interface{}{
		IsEncryptedKey:     true,
		EncryptedConfigKey: "enc:!!!not-base64!!!",
	}
	out, err := DecryptPayload(payload)
	require.Error(t, err, "非法 base64 的密文必须报错")
	assert.Nil(t, out)
}
