// 本文件为 user_viewer.go 的补充单元测试，与既有 user_viewer_test.go（BuildDataScopes
// 的主映射矩阵）互补，补齐其余未覆盖路径：
//  1. NewUserViewer 构造的 UserViewer 各 getter 对注入身份的原样回读；
//  2. viewer.WithContext / viewer.FromContext 的用户身份往返；
//  3. 与身份注入无关的固定语义（无权限/角色、权限判定恒拒、非系统上下文、不记审计）；
//  4. BuildDataScopes 在既有矩阵之外的边界组合（显式空切片触发旧单值回退、
//     UNSPECIFIED 与有效范围混排、UNIT 类空目标集）。
package viewer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/tx7do/go-crud/viewer"

	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
)

// TestUserViewer_GettersReturnInjectedIdentity 验证注入 UserViewer 的身份字段
// （uid/tid/ouid/traceID/数据范围）必须能从对应 getter 原样取回，且平台/租户视图语义
// 严格随 tenant_id 是否为零翻转——这是租户隔离闸门的判定依据。
func TestUserViewer_GettersReturnInjectedIdentity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		uid          uint64
		tid          uint64
		ouid         uint64
		traceID      string
		dataScopes   []viewer.DataScope
		wantPlatform bool
		wantTenant   bool
	}{
		{
			name:         "tenant view tid>0",
			uid:          11,
			tid:          22,
			ouid:         33,
			traceID:      "trace-tenant-001",
			dataScopes:   []viewer.DataScope{{ScopeType: viewer.ScopeTypeAll}},
			wantPlatform: false,
			wantTenant:   true,
		},
		{
			name:         "platform view tid=0",
			uid:          1,
			tid:          0,
			ouid:         2,
			traceID:      "",
			dataScopes:   []viewer.DataScope{{ScopeType: viewer.ScopeTypeSelf}},
			wantPlatform: true,
			wantTenant:   false,
		},
		{
			name:    "multi scopes with unit targets",
			uid:     100,
			tid:     200,
			ouid:    300,
			traceID: "trace-multi",
			dataScopes: []viewer.DataScope{
				{ScopeType: viewer.ScopeTypeSelf},
				{ScopeType: viewer.ScopeTypeUnit, TargetIDs: []uint64{31, 32}},
			},
			wantPlatform: false,
			wantTenant:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			vc := NewUserViewer(tc.uid, tc.tid, tc.ouid, tc.traceID, tc.dataScopes)

			require.Equal(t, tc.uid, vc.UserID(), "UserID 必须原样回读注入值")
			require.Equal(t, tc.tid, vc.TenantID(), "TenantID 必须原样回读注入值")
			require.Equal(t, tc.ouid, vc.OrgUnitID(), "OrgUnitID 必须原样回读注入值")
			require.Equal(t, tc.traceID, vc.TraceID(), "TraceID 必须原样回读注入值")
			require.Equal(t, tc.dataScopes, vc.DataScope(), "数据范围必须原样回读注入值")
			require.Equal(t, tc.wantPlatform, vc.IsPlatformContext(),
				"IsPlatformContext 必须与 tenant_id==0 的语义一致")
			require.Equal(t, tc.wantTenant, vc.IsTenantContext(),
				"IsTenantContext 必须与 tenant_id>0 的语义一致")
		})
	}
}

// TestUserViewer_FixedSemantics 验证 UserViewer 与身份注入无关的固定语义：
// 不携带权限/角色列表（细粒度权限走 authz 链路而非 viewer）、HasPermission 恒拒、
// 永远不是系统上下文、不记审计日志。
func TestUserViewer_FixedSemantics(t *testing.T) {
	t.Parallel()

	vc := NewUserViewer(1, 2, 3, "trace-fixed", []viewer.DataScope{{ScopeType: viewer.ScopeTypeSelf}})

	require.Nil(t, vc.Permissions(), "UserViewer 不携带权限列表")
	require.Nil(t, vc.Roles(), "UserViewer 不携带角色列表")
	require.False(t, vc.IsSystemContext(), "用户身份永远不是系统上下文")
	require.False(t, vc.ShouldAudit(), "用户身份默认不记审计日志")

	for _, comb := range [][2]string{{"read", "user"}, {"", ""}} {
		require.False(t, vc.HasPermission(comb[0], comb[1]),
			"用户身份的权限判定必须恒拒 action=%q resource=%q", comb[0], comb[1])
	}
}

// TestUserViewer_ContextRoundTrip 验证 viewer.WithContext 注入的 UserViewer
// 能被 viewer.FromContext 完整取回（身份无损耗）；未注入的裸 context 取不到 viewer。
func TestUserViewer_ContextRoundTrip(t *testing.T) {
	t.Parallel()

	scopes := []viewer.DataScope{
		{ScopeType: viewer.ScopeTypeUnit, TargetIDs: []uint64{5, 6}},
	}
	ctx := viewer.WithContext(context.Background(), NewUserViewer(101, 202, 303, "trace-xyz", scopes))

	vc, ok := viewer.FromContext(ctx)
	require.True(t, ok, "WithContext 注入后必须能经 FromContext 取回")
	require.Equal(t, uint64(101), vc.UserID())
	require.Equal(t, uint64(202), vc.TenantID())
	require.Equal(t, uint64(303), vc.OrgUnitID())
	require.Equal(t, "trace-xyz", vc.TraceID())
	require.Equal(t, scopes, vc.DataScope())

	_, ok = viewer.FromContext(context.Background())
	require.False(t, ok, "未注入 viewer 的 context 不应取到 viewer")
}

// TestBuildDataScopes_EdgeCombinations 补充既有 TestBuildDataScopes 矩阵之外的
// 边界组合：空列表配 UNSPECIFIED 旧值不回退、显式空切片仍触发旧单值回退、
// UNSPECIFIED 与有效范围混排时只保留有效项、UNIT 类空目标集的映射。
// 语义与源码注释一致：UNSPECIFIED 一律剔除、不做兜底放行。
func TestBuildDataScopes_EdgeCombinations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		scopes []identityV1.DataScope
		units  []uint64
		legacy identityV1.DataScope
		want   []viewer.DataScope
	}{
		{
			name:   "nil scopes with unspecified legacy stays empty",
			scopes: nil,
			units:  nil,
			legacy: identityV1.DataScope_DATA_SCOPE_UNSPECIFIED,
			want:   []viewer.DataScope{},
		},
		{
			name:   "explicit empty slice falls back to legacy single value",
			scopes: []identityV1.DataScope{},
			units:  nil,
			legacy: identityV1.DataScope_SELF,
			want:   []viewer.DataScope{{ScopeType: viewer.ScopeTypeSelf}},
		},
		{
			name: "unspecified mixed with all keeps only all",
			scopes: []identityV1.DataScope{
				identityV1.DataScope_DATA_SCOPE_UNSPECIFIED,
				identityV1.DataScope_ALL,
			},
			units:  nil,
			legacy: identityV1.DataScope_DATA_SCOPE_UNSPECIFIED,
			want:   []viewer.DataScope{{ScopeType: viewer.ScopeTypeAll}},
		},
		{
			name: "unit and child with empty targets maps to unit with nil targets",
			scopes: []identityV1.DataScope{
				identityV1.DataScope_UNIT_AND_CHILD,
			},
			units:  nil,
			legacy: identityV1.DataScope_DATA_SCOPE_UNSPECIFIED,
			want:   []viewer.DataScope{{ScopeType: viewer.ScopeTypeUnit, TargetIDs: nil}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.want, BuildDataScopes(tc.scopes, tc.units, tc.legacy))
		})
	}
}
