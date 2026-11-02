// Package ai 的客户端工厂测试。
//
// NewClient 只做"表行 → 客户端"的翻译，内部 openai.Client 的 config 字段不可导出，
// 无法断言 BaseURL/APIKey 落位；这里能钉住的是调用方依赖的契约面：
//   - nil 配置必须报错（service 层传空行的兜底路径）；
//   - ModelType 不在 {1=LOCAL, 2=CLOUD} 内必须报错——proto 枚举零值 UNSPECIFIED(0)
//     走到这里就是配置行坏了，静默返回一个可用客户端会把错误吞到第一次对话才爆；
//   - 云端/本地两条合法分支都能构造出非 nil 客户端（上游对空 host/port 有默认值）。
//
// 超时语义（注入无超时 http.Client 以免流式被 30s 腰斩）在上游插件内部，无法从
// 返回值断言；该语义的守护靠 docs/ai_module.md 与本文件注释，回归时人工核查。
package ai

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestNewClient_NilConfig 配置行缺失时必须立即报错，
// 不能返回一个"看起来能用"的客户端把问题推迟到对话期。
func TestNewClient_NilConfig(t *testing.T) {
	client, err := NewClient(nil, "sk-test")
	require.Error(t, err, "nil 配置应返回错误")
	require.Nil(t, client, "出错时不应返回客户端实例")
	require.Contains(t, err.Error(), "config is nil")
}

// TestNewClient_UnsupportedModelType ModelType 零值（UNSPECIFIED=0）与越界值
// 都必须报错：provider 行的 model_type 坏了要在构造期暴露，而不是吞成默认客户端。
func TestNewClient_UnsupportedModelType(t *testing.T) {
	for _, mt := range []int32{0, 3, -1} {
		client, err := NewClient(&ClientConfig{ModelType: mt}, "sk-test")
		require.Error(t, err, "ModelType=%d 应返回错误", mt)
		require.Nil(t, client)
		require.Contains(t, err.Error(), "unsupported ai model type",
			"错误信息要能让人直接定位到 model_type 字段")
	}
}

// TestNewClient_Cloud 云端分支：合法配置构造成功。
// apiKey 必须由调用方解密后传入（空 key 构造出的客户端调用必 401），
// 本层不做密钥校验，这里只钉住"给了 key 就能构造"。
func TestNewClient_Cloud(t *testing.T) {
	client, err := NewClient(&ClientConfig{
		ModelType: int32(ModelTypeCloud),
		BaseUrl:   "https://llm.example.com/v1",
	}, "sk-decrypted-key")
	require.NoError(t, err, "云端合法配置应构造成功")
	require.NotNil(t, client)

	// 空 apiKey 也允许构造（上游按 key 拼 header，构造期不校验）——
	// 归属校验与解密在 service 层完成后才进来，本层不重复把关。
	client, err = NewClient(&ClientConfig{ModelType: int32(ModelTypeCloud)}, "")
	require.NoError(t, err, "云端空 key 构造期不拦（校验职责在 service 层）")
	require.NotNil(t, client)
}

// TestNewClient_Local 本地分支：Ollama 的 host/port 缺省由上游兜底
// （localhost:11434），本层透传即可；显式给出也照常构造。
func TestNewClient_Local(t *testing.T) {
	client, err := NewClient(&ClientConfig{ModelType: int32(ModelTypeLocal)}, "")
	require.NoError(t, err, "本地缺省 host/port 应由上游兜底构造成功")
	require.NotNil(t, client)

	client, err = NewClient(&ClientConfig{
		ModelType: int32(ModelTypeLocal),
		LocalHost: "192.168.1.10",
		LocalPort: 11435,
	}, "")
	require.NoError(t, err, "本地显式 host/port 应构造成功")
	require.NotNil(t, client)
}
