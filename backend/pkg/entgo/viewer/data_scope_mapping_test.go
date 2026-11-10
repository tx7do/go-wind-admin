package viewer

import (
	"testing"

	"github.com/tx7do/go-crud/viewer"

	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
)

func one(t *testing.T, typ viewer.ScopeType, targets []int64) []viewer.DataScope {
	t.Helper()
	return []viewer.DataScope{{ScopeType: typ, TargetIDs: targets}}
}

// TestBuildDataScopes 验证令牌聚合数据范围到库层执行结构的映射矩阵：
// ALL/SELF/UNIT_* 映射、UNSPECIFIED 剔除、旧单值回退、混合并集。
func TestBuildDataScopes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		scopes  []identityV1.DataScope
		units   []int64
		legacy  identityV1.DataScope
		wantLen int
		want    []viewer.DataScope
	}{
		{
			name:    "all",
			scopes:  []identityV1.DataScope{identityV1.DataScope_ALL},
			wantLen: 1,
			want:    one(t, viewer.ScopeTypeAll, nil),
		},
		{
			name:    "self",
			scopes:  []identityV1.DataScope{identityV1.DataScope_SELF},
			wantLen: 1,
			want:    one(t, viewer.ScopeTypeSelf, nil),
		},
		{
			name:    "unit_only carries targets",
			scopes:  []identityV1.DataScope{identityV1.DataScope_UNIT_ONLY},
			units:   []int64{5},
			wantLen: 1,
			want:    one(t, viewer.ScopeTypeUnit, []int64{5}),
		},
		{
			name:    "selected_units carries targets",
			scopes:  []identityV1.DataScope{identityV1.DataScope_SELECTED_UNITS},
			units:   []int64{1, 2},
			wantLen: 1,
			want:    one(t, viewer.ScopeTypeUnit, []int64{1, 2}),
		},
		{
			name:    "unspecified dropped leaves empty",
			scopes:  []identityV1.DataScope{identityV1.DataScope_DATA_SCOPE_UNSPECIFIED},
			units:   []int64{9},
			wantLen: 0,
			want:    []viewer.DataScope{},
		},
		{
			name:    "legacy single value fallback",
			scopes:  nil,
			legacy:  identityV1.DataScope_ALL,
			wantLen: 1,
			want:    one(t, viewer.ScopeTypeAll, nil),
		},
		{
			name: "mixed union",
			scopes: []identityV1.DataScope{
				identityV1.DataScope_SELF,
				identityV1.DataScope_UNIT_AND_CHILD,
			},
			units:   []int64{3},
			wantLen: 2,
			want: []viewer.DataScope{
				{ScopeType: viewer.ScopeTypeSelf},
				{ScopeType: viewer.ScopeTypeUnit, TargetIDs: []int64{3}},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := BuildDataScopes(tc.scopes, tc.units, tc.legacy)
			if len(got) != tc.wantLen {
				t.Fatalf("len = %d, want %d", len(got), tc.wantLen)
			}
			for i := range tc.want {
				if got[i].ScopeType != tc.want[i].ScopeType {
					t.Fatalf("scope[%d].type = %v, want %v", i, got[i].ScopeType, tc.want[i].ScopeType)
				}
				if len(got[i].TargetIDs) != len(tc.want[i].TargetIDs) {
					t.Fatalf("scope[%d].targets len = %d, want %d", i, len(got[i].TargetIDs), len(tc.want[i].TargetIDs))
				}
			}
		})
	}
}

// TestBuildDataScopes_EdgeCombinations 补充既有 TestBuildDataScopes 矩阵之外的
// 边界组合：空列表配 UNSPECIFIED 旧值不回退、显式空切片仍触发旧单值回退、
// UNSPECIFIED 与有效范围混排时只保留有效项、UNIT 类空目标集的映射。
// 语义与源码注释一致：UNSPECIFIED 一律剔除、不做兜底放行。
func TestBuildDataScopes_EdgeCombinations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		scopes  []identityV1.DataScope
		units   []int64
		legacy  identityV1.DataScope
		wantLen int
		want    []viewer.DataScope
	}{
		{
			name:    "nil scopes with unspecified legacy stays empty",
			scopes:  nil,
			units:   nil,
			legacy:  identityV1.DataScope_DATA_SCOPE_UNSPECIFIED,
			wantLen: 0,
			want:    []viewer.DataScope{},
		},
		{
			name:    "explicit empty slice falls back to legacy single value",
			scopes:  []identityV1.DataScope{},
			units:   nil,
			legacy:  identityV1.DataScope_SELF,
			wantLen: 1,
			want:    one(t, viewer.ScopeTypeSelf, nil),
		},
		{
			name: "unspecified mixed with all keeps only all",
			scopes: []identityV1.DataScope{
				identityV1.DataScope_DATA_SCOPE_UNSPECIFIED,
				identityV1.DataScope_ALL,
			},
			units:   nil,
			legacy:  identityV1.DataScope_DATA_SCOPE_UNSPECIFIED,
			wantLen: 1,
			want:    one(t, viewer.ScopeTypeAll, nil),
		},
		{
			name:    "unit and child with empty targets maps to unit with nil targets",
			scopes:  []identityV1.DataScope{identityV1.DataScope_UNIT_AND_CHILD},
			units:   nil,
			legacy:  identityV1.DataScope_DATA_SCOPE_UNSPECIFIED,
			wantLen: 1,
			want:    one(t, viewer.ScopeTypeUnit, nil),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := BuildDataScopes(tc.scopes, tc.units, tc.legacy)
			if len(got) != tc.wantLen {
				t.Fatalf("len = %d, want %d", len(got), tc.wantLen)
			}
			for i := range tc.want {
				if got[i].ScopeType != tc.want[i].ScopeType {
					t.Fatalf("scope[%d].type = %v, want %v", i, got[i].ScopeType, tc.want[i].ScopeType)
				}
				if len(got[i].TargetIDs) != len(tc.want[i].TargetIDs) {
					t.Fatalf("scope[%d].targets len = %d, want %d", i, len(got[i].TargetIDs), len(tc.want[i].TargetIDs))
				}
			}
		})
	}
}
