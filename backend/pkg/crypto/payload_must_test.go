package crypto

// 本文件覆盖任务载荷（payload.go）的边界分支：
// 序列化失败的载荷必须报错、伪造的"已加密"标记必须被拒、
// 非法密文/非法 JSON 的解密路径必须报错，以及 Must* 辅助函数在
// 成功与失败（panic）路径上的行为。
//
// 这些用例对全局加密器的初始化状态不敏感（未初始化与已初始化时
// 走到的分支不同但断言结果一致），无执行顺序要求。

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEncryptPayload_MarshalFailure 含不可 JSON 序列化值（函数）的载荷
// 必须报错，而不是产出半加密的残缺载荷。
func TestEncryptPayload_MarshalFailure(t *testing.T) {
	payload := map[string]interface{}{
		"host":           "imap.example.com",
		"unserializable": func() {},
	}
	out, err := EncryptPayload(payload)
	require.Error(t, err, "含不可序列化值的载荷必须报错")
	assert.Nil(t, out)
}

// TestDecryptPayload_MarkerFalsePassthrough 显式标记 IsEncryptedKey=false
// 的载荷按明文原样返回（与缺省标记同一语义，兼容历史数据）。
func TestDecryptPayload_MarkerFalsePassthrough(t *testing.T) {
	payload := map[string]interface{}{
		IsEncryptedKey: false,
		"host":         "imap.example.com",
		"port":         993,
	}
	out, err := DecryptPayload(payload)
	require.NoError(t, err)
	assert.Equal(t, payload, out, "标记为未加密的载荷必须原样返回")
}

// TestDecryptPayload_InvalidEncryptedConfigType 标记为已加密但密文字段
// 类型非法（非字符串）的载荷必须报错：这是任务队列里的数据完整性
// 错误，静默直通会把内部结构当业务数据使用。
func TestDecryptPayload_InvalidEncryptedConfigType(t *testing.T) {
	payload := map[string]interface{}{
		IsEncryptedKey:     true,
		EncryptedConfigKey: 42,
	}
	out, err := DecryptPayload(payload)
	require.Error(t, err, "非字符串形式的密文字段必须报错")
	assert.Nil(t, out)
}

// TestDecryptPayload_GarbageConfigUnmarshalFailure 密文内容不是合法 JSON
// 时必须报错（无论加密能力启用与否，直通或解密后都不是合法载荷）。
func TestDecryptPayload_GarbageConfigUnmarshalFailure(t *testing.T) {
	payload := map[string]interface{}{
		IsEncryptedKey:     true,
		EncryptedConfigKey: "definitely not json at all",
	}
	out, err := DecryptPayload(payload)
	require.Error(t, err, "非 JSON 的载荷内容必须报错")
	assert.Nil(t, out)
}

// TestMustEncryptPayload_Success 成功路径：不 panic，结果为带
// EncryptedConfigKey 键的封装载荷，路由元数据保留。
func TestMustEncryptPayload_Success(t *testing.T) {
	payload := map[string]interface{}{
		"task_id":   123,
		"task_type": "email_processor",
	}
	out := MustEncryptPayload(payload)
	require.NotNil(t, out)
	assert.Contains(t, out, EncryptedConfigKey, "封装结果必须含密文槽位")
	assert.Equal(t, 123, out["task_id"], "task_id 路由元数据必须保留")
	assert.Equal(t, "email_processor", out["task_type"], "task_type 路由元数据必须保留")
}

// TestMustEncryptPayload_PanicsOnMarshalFailure 不可序列化载荷上
// MustEncryptPayload 必须把错误以 panic 暴露。
func TestMustEncryptPayload_PanicsOnMarshalFailure(t *testing.T) {
	assert.Panics(t, func() {
		_ = MustEncryptPayload(map[string]interface{}{"bad": func() {}})
	}, "不可序列化载荷必须 panic")
}

// TestMustDecryptPayload_Success 成功路径：未加密载荷直通返回，不 panic。
func TestMustDecryptPayload_Success(t *testing.T) {
	payload := map[string]interface{}{
		IsEncryptedKey: false,
		"keep":         1,
	}
	out := MustDecryptPayload(payload)
	assert.Equal(t, payload, out)
}

// TestMustDecryptPayload_PanicsOnUndecryptableConfig 非法 JSON 的
// "密文"上 MustDecryptPayload 必须 panic。
func TestMustDecryptPayload_PanicsOnUndecryptableConfig(t *testing.T) {
	assert.Panics(t, func() {
		_ = MustDecryptPayload(map[string]interface{}{
			IsEncryptedKey:     true,
			EncryptedConfigKey: "still not json",
		})
	}, "非法 JSON 的载荷必须 panic")
}
