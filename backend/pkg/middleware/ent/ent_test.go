// 本文件为 ent.go 中 Server 中间件的单元测试。
//
// 该中间件是"认证层 → 数据层"的身份桥梁：从 kratos 服务端 metadata 的
// x-md-global-operator 头里还原 OperatorMetadata（编码方式与 pkg/metadata 完全一致：
// proto marshal + RawStd base64），据此构造 UserViewer 注入下游 ctx，
// 供 Ent privacy 层做租户隔离与数据范围过滤。
// 这里锁定三条关键行为：
//  1. 合法 operator 元数据注入的 UserViewer，其 UserID/TenantID/OrgUnitID 与
//     BuildDataScopes 映射结果必须与元数据逐字段一致（含旧单值回退与 UNSPECIFIED 剔除）；
//  2. ctx 中存在有效 otel SpanContext 时 TraceID 必须透传（用于审计日志关联），否则为空；
//  3. ctx 无有效 operator 元数据（无服务端 metadata / 无该头 / 头内容非法）时
//     必须原样透传、绝不注入 viewer——下游 Ent privacy 会因缺 viewer 而 fail-closed。
package ent

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/go-kratos/kratos/v2/encoding"
	_ "github.com/go-kratos/kratos/v2/encoding/proto"
	"github.com/go-kratos/kratos/v2/metadata"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	"github.com/tx7do/go-crud/viewer"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
)

// operatorMetadataHeader 是生产链路中 operator 元数据所在的 kratos 服务端 metadata 头，
// 与 pkg/metadata 中的私有常量保持一致。
const operatorMetadataHeader = "x-md-global-operator"

// withOperatorMetadata 按 pkg/metadata 的生产编码（proto codec Marshal + RawStd base64）
// 把 OperatorMetadata 塞进 kratos 服务端 metadata，构造出中间件实际读取的 ctx。
// base 必须由调用方提供，以便叠加 otel SpanContext 等其他 context 值。
func withOperatorMetadata(t *testing.T, base context.Context, op *authenticationV1.OperatorMetadata) context.Context {
	t.Helper()

	codec := encoding.GetCodec("proto")
	require.NotNil(t, codec, "kratos proto 编解码器必须已注册")

	b, err := codec.Marshal(op)
	require.NoError(t, err, "OperatorMetadata 必须可被 proto codec 序列化")

	return metadata.NewServerContext(base, metadata.Metadata{
		operatorMetadataHeader: []string{base64.RawStdEncoding.EncodeToString(b)},
	})
}

// TestServer_InjectsUserViewerFromOperatorMetadata 表驱动验证：中间件包装后，
// 下游 handler 的 ctx 里能经 viewer.FromContext 取回 UserViewer，
// 且身份字段与数据范围映射（BuildDataScopes）与注入的 OperatorMetadata 逐字段一致。
// 覆盖聚合范围、旧单值回退、UNIT 类目标集、UNSPECIFIED 剔除等映射路径。
func TestServer_InjectsUserViewerFromOperatorMetadata(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		op           *authenticationV1.OperatorMetadata
		wantUID      uint64
		wantTID      uint64
		wantOUID     uint64
		wantScopes   []viewer.DataScope
		wantPlatform bool
		wantTenant   bool
	}{
		{
			name: "aggregated ALL scope with unit ids ignored",
			op: &authenticationV1.OperatorMetadata{
				UserId:           42,
				TenantId:         7,
				OrgUnitId:        9,
				DataScopes:       []identityV1.DataScope{identityV1.DataScope_ALL},
				DataScopeUnitIds: []uint64{7, 8},
			},
			wantUID:      42,
			wantTID:      7,
			wantOUID:     9,
			wantScopes:   []viewer.DataScope{{ScopeType: viewer.ScopeTypeAll}},
			wantPlatform: false,
			wantTenant:   true,
		},
		{
			name: "legacy single value fallback with platform view",
			op: &authenticationV1.OperatorMetadata{
				UserId:     5,
				TenantId:   0,
				OrgUnitId:  0,
				DataScope:  identityV1.DataScope_SELF,
				DataScopes: nil,
			},
			wantUID:      5,
			wantTID:      0,
			wantOUID:     0,
			wantScopes:   []viewer.DataScope{{ScopeType: viewer.ScopeTypeSelf}},
			wantPlatform: true,
			wantTenant:   false,
		},
		{
			name: "unit scope carries unit targets",
			op: &authenticationV1.OperatorMetadata{
				UserId:           1,
				TenantId:         2,
				OrgUnitId:        3,
				DataScopes:       []identityV1.DataScope{identityV1.DataScope_UNIT_ONLY},
				DataScopeUnitIds: []uint64{5},
			},
			wantUID:      1,
			wantTID:      2,
			wantOUID:     3,
			wantScopes:   []viewer.DataScope{{ScopeType: viewer.ScopeTypeUnit, TargetIDs: []uint64{5}}},
			wantPlatform: false,
			wantTenant:   true,
		},
		{
			name: "mixed scopes mapped in order with unspecified dropped",
			op: &authenticationV1.OperatorMetadata{
				UserId:           1,
				TenantId:         2,
				OrgUnitId:        3,
				DataScopes:       []identityV1.DataScope{identityV1.DataScope_DATA_SCOPE_UNSPECIFIED, identityV1.DataScope_SELF, identityV1.DataScope_UNIT_AND_CHILD},
				DataScopeUnitIds: []uint64{3},
			},
			wantUID:      1,
			wantTID:      2,
			wantOUID:     3,
			wantScopes:   []viewer.DataScope{{ScopeType: viewer.ScopeTypeSelf}, {ScopeType: viewer.ScopeTypeUnit, TargetIDs: []uint64{3}}},
			wantPlatform: false,
			wantTenant:   true,
		},
		{
			name: "all unspecified scopes yield empty scope set",
			op: &authenticationV1.OperatorMetadata{
				UserId:     1,
				TenantId:   2,
				OrgUnitId:  3,
				DataScopes: []identityV1.DataScope{identityV1.DataScope_DATA_SCOPE_UNSPECIFIED},
			},
			wantUID:      1,
			wantTID:      2,
			wantOUID:     3,
			wantScopes:   []viewer.DataScope{},
			wantPlatform: false,
			wantTenant:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var captured context.Context
			probe := func(ctx context.Context, req interface{}) (interface{}, error) {
				captured = ctx
				return req, nil
			}
			wrapped := Server()(probe)

			payload := "request-payload"
			got, err := wrapped(withOperatorMetadata(t, context.Background(), tc.op), payload)
			require.NoError(t, err)
			require.Equal(t, payload, got, "请求体必须原样透传")
			require.NotNil(t, captured, "下游 handler 必须被调用")

			vc, ok := viewer.FromContext(captured)
			require.True(t, ok, "中间件必须为下游注入 UserViewer")
			require.Equal(t, tc.wantUID, vc.UserID(), "UserID 必须来自 operator 元数据")
			require.Equal(t, tc.wantTID, vc.TenantID(), "TenantID 必须来自 operator 元数据")
			require.Equal(t, tc.wantOUID, vc.OrgUnitID(), "OrgUnitID 必须来自 operator 元数据")
			require.Equal(t, tc.wantScopes, vc.DataScope(),
				"数据范围必须与 BuildDataScopes 的映射结果一致")
			require.Equal(t, tc.wantPlatform, vc.IsPlatformContext())
			require.Equal(t, tc.wantTenant, vc.IsTenantContext())
			require.False(t, vc.IsSystemContext(), "中间件注入的永远是用户身份，不是系统身份")
			require.Equal(t, "", vc.TraceID(), "ctx 无 otel SpanContext 时 TraceID 必须为空")
		})
	}
}

// TestServer_PropagatesOtelTraceID 验证 ctx 中存在有效 otel SpanContext 时，
// 其 TraceID 必须透传到注入的 UserViewer（审计日志靠它做链路关联）。
// 这里用 trace API 直接构造非 recording 的 SpanContext，无需引入 otel SDK。
func TestServer_PropagatesOtelTraceID(t *testing.T) {
	t.Parallel()

	tid := trace.TraceID{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
	}
	sid := trace.SpanID{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18}
	sc := trace.NewSpanContext(trace.SpanContextConfig{TraceID: tid, SpanID: sid})
	require.True(t, sc.HasTraceID(), "构造的 SpanContext 必须带有效 TraceID 才能进入透传分支")

	// SpanContext 与 operator 元数据必须能同时存在于同一个 ctx
	ctx := trace.ContextWithSpanContext(context.Background(), sc)
	ctx = withOperatorMetadata(t, ctx, &authenticationV1.OperatorMetadata{
		UserId:     1,
		TenantId:   2,
		OrgUnitId:  3,
		DataScopes: []identityV1.DataScope{identityV1.DataScope_SELF},
	})

	var captured context.Context
	probe := func(ctx context.Context, req interface{}) (interface{}, error) {
		captured = ctx
		return req, nil
	}
	wrapped := Server()(probe)

	_, err := wrapped(ctx, nil)
	require.NoError(t, err)

	vc, ok := viewer.FromContext(captured)
	require.True(t, ok)
	require.Equal(t, tid.String(), vc.TraceID(),
		"有效 SpanContext 的 TraceID 必须透传到 viewer，供审计日志关联")
	require.Equal(t, uint64(1), vc.UserID(), "身份注入不因 SpanContext 存在而丢失")
}

// TestServer_PassthroughWithoutOperatorMetadata 表驱动验证：ctx 没有合法
// operator 元数据时（无服务端 metadata / metadata 无该头 / 头为空串 / 头内容非法），
// 中间件不注入 viewer、不包装 ctx、请求原样透传。
// 该行为是 Ent privacy fail-closed 的前提：缺 viewer 时数据层会拒绝而非放行。
func TestServer_PassthroughWithoutOperatorMetadata(t *testing.T) {
	t.Parallel()

	cases := map[string]context.Context{
		"no server metadata at all":      context.Background(),
		"server metadata without header": metadata.NewServerContext(context.Background(), metadata.Metadata{"x-md-global-other": []string{"1"}}),
		"operator header empty string":   metadata.NewServerContext(context.Background(), metadata.Metadata{operatorMetadataHeader: []string{""}}),
		"operator header invalid base64": metadata.NewServerContext(context.Background(), metadata.Metadata{operatorMetadataHeader: []string{"!!!not-base64!!!"}}),
	}

	for name, ctx := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var captured context.Context
			called := false
			probe := func(ctx context.Context, req interface{}) (interface{}, error) {
				captured = ctx
				called = true
				return req, nil
			}
			wrapped := Server()(probe)

			payload := "passthrough-payload"
			got, err := wrapped(ctx, payload)
			require.NoError(t, err)
			require.Equal(t, payload, got, "请求体必须原样透传")
			require.True(t, called, "下游 handler 必须被调用")
			require.True(t, captured == ctx, "ctx 必须原样透传，不得被包装")

			vc, ok := viewer.FromContext(captured)
			require.False(t, ok, "无有效 operator 元数据时不得注入 viewer")
			require.Nil(t, vc)
		})
	}
}
