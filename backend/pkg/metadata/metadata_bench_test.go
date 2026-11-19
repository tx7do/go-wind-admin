// 本文件补 pkg/metadata 的性能基准与 NewContext 错误分支。
// encode/decode（proto marshal + base64）是认证中间件每请求两次的热路径，
// 既有测试只验正确性，未给性能基线。
package metadata

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
)

func benchOperatorInfo() *authenticationV1.OperatorMetadata {
	return &authenticationV1.OperatorMetadata{
		UserId:    42,
		TenantId:  3,
		OrgUnitId: 9,
	}
}

// TestNewContext_NilInfo 现状钉住：nil OperatorMetadata 被 proto codec 编码为空
// （不报错）。NewContext 写入的是 client 侧 metadata，而 FromContext 读取的是
// server 侧——纯 background 上下文没有 server metadata，命中 ErrNoMetadata。
func TestNewContext_NilInfo(t *testing.T) {
	ctx, err := NewContext(context.Background(), nil)
	require.NoError(t, err, "proto codec encodes a nil message as empty bytes")
	got, err := FromContext(ctx)
	require.Nil(t, got)
	require.Equal(t, ErrNoMetadata, err)
}

// --- 基准 ---

func BenchmarkEncodeOperatorMetadata(b *testing.B) {
	info := benchOperatorInfo()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := encodeOperatorMetadata(info); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecodeOperatorMetadata(b *testing.B) {
	info := benchOperatorInfo()
	enc, err := encodeOperatorMetadata(info)
	require.NoError(b, err)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := decodeOperatorMetadata(enc); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkNewContext_Append 认证中间件每请求的写入成本（proto marshal + base64 + md 追加）。
// FromContext 读的是 server 侧 metadata（与 NewContext 写的 client 侧不同源，
// 不能在同侧往返），读取成本由 BenchmarkDecodeOperatorMetadata 单独覆盖。
func BenchmarkNewContext_Append(b *testing.B) {
	info := benchOperatorInfo()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := NewContext(context.Background(), info); err != nil {
			b.Fatal(err)
		}
	}
}
