package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateAgentHTTPNode(t *testing.T) {
	node := func(data map[string]any) *agentGraphNode {
		return &agentGraphNode{ID: "http_1", Type: "http", Data: data}
	}

	t.Run("缺 url 拒绝", func(t *testing.T) {
		err := validateAgentHTTPNode(node(map[string]any{"method": "GET"}))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing url")
	})

	t.Run("方法白名单外拒绝", func(t *testing.T) {
		err := validateAgentHTTPNode(node(map[string]any{"url": "https://example.com", "method": "CONNECT"}))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "disallowed method")
	})

	t.Run("method 小写归一化放行", func(t *testing.T) {
		require.NoError(t, validateAgentHTTPNode(node(map[string]any{"url": "https://example.com", "method": "post"})))
	})

	t.Run("timeout 越界拒绝", func(t *testing.T) {
		err := validateAgentHTTPNode(node(map[string]any{"url": "https://example.com", "timeoutSeconds": "120"}))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "timeoutSeconds")
	})

	t.Run("timeout 非数值拒绝", func(t *testing.T) {
		require.Error(t, validateAgentHTTPNode(node(map[string]any{"url": "https://example.com", "timeoutSeconds": "abc"})))
	})

	t.Run("缺省 method/timeout 合法", func(t *testing.T) {
		require.NoError(t, validateAgentHTTPNode(node(map[string]any{"url": "https://example.com"})))
	})
}

func TestAgentHTTPNodeRegisteredInGraphValidation(t *testing.T) {
	// http 节点进入合法类型集：含 http 节点的图能过静态校验（无环前提由通用边检查负责）
	g := &agentGraph{
		Nodes: []agentGraphNode{
			{ID: "start_1", Type: "start", Data: map[string]any{}},
			{ID: "http_1", Type: "http", Data: map[string]any{"url": "https://example.com/api"}},
			{ID: "end_1", Type: "end", Data: map[string]any{"output": "{{http_1.status}}"}},
		},
		Edges: []agentGraphEdge{
			{ID: "e1", Source: "start_1", Target: "http_1"},
			{ID: "e2", Source: "http_1", Target: "end_1"},
		},
	}
	require.NoError(t, validateAgentGraph(g), "含 http 节点的合法图应通过静态校验")
}

func TestValidateAgentOutboundURL(t *testing.T) {
	t.Run("回环拒绝", func(t *testing.T) {
		for _, raw := range []string{
			"http://127.0.0.1/x",
			"http://localhost/x",
			"http://[::1]/x",
		} {
			err := validateAgentOutboundURL(raw)
			require.Error(t, err, raw)
			assert.Contains(t, err.Error(), "blocked outbound host")
		}
	})

	t.Run("云元数据端点拒绝", func(t *testing.T) {
		err := validateAgentOutboundURL("http://169.254.169.254/latest/meta-data/")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "blocked outbound host")
	})

	t.Run("私网拒绝", func(t *testing.T) {
		for _, raw := range []string{"http://10.0.0.5/x", "http://192.168.1.1/x", "http://172.16.0.9/x"} {
			require.Error(t, validateAgentOutboundURL(raw), raw)
		}
	})

	t.Run("非 http(s) scheme 拒绝", func(t *testing.T) {
		err := validateAgentOutboundURL("ftp://example.com/file")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "scheme")
	})

	t.Run("内网后缀拒绝", func(t *testing.T) {
		err := validateAgentOutboundURL("http://api.internal/x")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "blocked outbound host")
	})

	t.Run("无 host / 解析失败拒绝", func(t *testing.T) {
		require.Error(t, validateAgentOutboundURL("not-a-url"))
		err := validateAgentOutboundURL("http://this-host-does-not-exist.invalid/x")
		require.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "resolve") || strings.Contains(err.Error(), "blocked"),
			"解析失败或被阻断均算 fail-closed，实际: %v", err)
	})
}
