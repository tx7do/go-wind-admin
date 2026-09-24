// Package fieldperm_test 针对 pkg/fieldperm 的 proto 消息裁剪原语做黑盒测试。
//
// 本文件覆盖 ApplyReadMask / ApplyReadMaskList / StripWriteFields：
//   - 读路径：命中隐藏集（json_name 与 proto 字段名两种拼写都要命中）的顶层
//     字段被就地清除，未命中字段原样保留——通过 protojson 序列化前后对比断言；
//   - 列表路径：对 items 逐个应用同一裁剪；
//   - 写路径：除清值外，还要从 field_mask.Paths 中剔除顶层段命中的路径
//     （"field.sub" 形式只看顶层段），未命中路径与 nil/空 mask 保持原样。
//
// 测试消息选用 identitypb.Plan（go-wind-admin/api/gen/go/identity/service/v1）：
//   - created_by：proto 名与 json 名（createdBy）拼写不同，覆盖双拼写命中路径；
//   - description / remark：两种拼写一致的字段，作为保留对照。
package fieldperm_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	identitypb "go-wind-admin/api/gen/go/identity/service/v1"
	"go-wind-admin/pkg/fieldperm"
)

// planFields 是参与测试的三个字段的 protojson 键名（均为 json_name 拼写）。
var (
	keyCreatedBy    = "createdBy"   // created_by 的 json_name（与 proto 名拼写不同）
	keyDescription  = "description" // 两种拼写一致
	keyRemark       = "remark"      // 两种拼写一致
	allPlanJsonKeys = []string{keyCreatedBy, keyDescription, keyRemark}
)

// newPopulatedPlan 构造三个字段全部填充的 Plan 消息。
func newPopulatedPlan() *identitypb.Plan {
	desc := "desc-value"
	remark := "remark-value"
	creator := uint32(42)
	return &identitypb.Plan{
		Description: &desc,
		Remark:      &remark,
		CreatedBy:   &creator,
	}
}

// marshalPlan 用 protojson 序列化消息并断言成功。
// protojson 只输出已填充字段且键名一律用 json_name，
// 因此"字段被清除"等价于"序列化输出中该键消失"。
func marshalPlan(t *testing.T, msg proto.Message) string {
	t.Helper()
	buf, err := protojson.Marshal(msg)
	require.NoError(t, err)
	return string(buf)
}

// assertKeys 断言序列化输出中各键的存在性。
func assertKeys(t *testing.T, serialized string, present []string, absent []string) {
	t.Helper()
	for _, k := range present {
		assert.Contains(t, serialized, "\""+k+"\"", "字段 %s 应保留在输出中", k)
	}
	for _, k := range absent {
		assert.NotContains(t, serialized, "\""+k+"\"", "字段 %s 应被从输出中清除", k)
	}
}

// TestApplyReadMask_HiddenSpellingVariants 隐藏集里无论放 json_name 拼写
// 还是 proto 字段名拼写，都应命中同一字段并清除之；未命中字段保留。
func TestApplyReadMask_HiddenSpellingVariants(t *testing.T) {
	tests := []struct {
		name      string
		hidden    map[string]struct{}
		absentKey string
	}{
		{name: "json spelling hit", hidden: map[string]struct{}{"createdBy": {}}, absentKey: keyCreatedBy},
		{name: "proto spelling hit", hidden: map[string]struct{}{"created_by": {}}, absentKey: keyCreatedBy},
		{name: "same-spelling field hit", hidden: map[string]struct{}{"description": {}}, absentKey: keyDescription},
		{name: "same-spelling field hit remark", hidden: map[string]struct{}{"remark": {}}, absentKey: keyRemark},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := newPopulatedPlan()
			assertKeys(t, marshalPlan(t, plan), allPlanJsonKeys, nil)

			fieldperm.ApplyReadMask(plan, tt.hidden)

			keptKeys := make([]string, 0, len(allPlanJsonKeys))
			for _, k := range allPlanJsonKeys {
				if k != tt.absentKey {
					keptKeys = append(keptKeys, k)
				}
			}
			assertKeys(t, marshalPlan(t, plan), keptKeys, []string{tt.absentKey})
		})
	}
}

// TestApplyReadMask_NonMatchingHiddenSet 未知字段名的隐藏集不清除任何字段。
func TestApplyReadMask_NonMatchingHiddenSet(t *testing.T) {
	tests := []struct {
		name   string
		hidden map[string]struct{}
	}{
		{name: "unknown field name", hidden: map[string]struct{}{"noSuchField": {}}},
		{name: "empty hidden set", hidden: map[string]struct{}{}},
		{name: "nil hidden set", hidden: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := newPopulatedPlan()
			fieldperm.ApplyReadMask(plan, tt.hidden)
			assertKeys(t, marshalPlan(t, plan), allPlanJsonKeys, nil)
		})
	}
}

// TestApplyReadMask_NilMessage nil 消息入参必须安静返回，不得 panic。
func TestApplyReadMask_NilMessage(t *testing.T) {
	assert.NotPanics(t, func() {
		fieldperm.ApplyReadMask(nil, map[string]struct{}{"createdBy": {}})
	})
}

// TestApplyReadMask_TypedNilMessageInvalid 类型化 nil（接口非 nil 但指针为 nil）
// 的反射消息无效，应安静返回，不进入字段遍历、不 panic。
func TestApplyReadMask_TypedNilMessageInvalid(t *testing.T) {
	var typedNil *identitypb.Plan
	assert.NotPanics(t, func() {
		fieldperm.ApplyReadMask(typedNil, map[string]struct{}{"createdBy": {}})
	})
}

// TestApplyReadMaskList_AppliesToEveryItem 列表裁剪必须作用于每个元素，
// 命中字段（双拼写）在所有元素上都被清除。
func TestApplyReadMaskList_AppliesToEveryItem(t *testing.T) {
	for _, spelling := range []string{"createdBy", "created_by"} {
		t.Run("spelling/"+spelling, func(t *testing.T) {
			plans := []*identitypb.Plan{newPopulatedPlan(), newPopulatedPlan(), newPopulatedPlan()}
			fieldperm.ApplyReadMaskList(plans, map[string]struct{}{spelling: {}})
			// 每个元素都必须被裁剪，无一遗漏。
			for _, plan := range plans {
				assertKeys(t, marshalPlan(t, plan), []string{keyDescription, keyRemark}, []string{keyCreatedBy})
			}
		})
	}
}

// TestApplyReadMaskList_EmptyHiddenSetNoop 空隐藏集下列表整体不动。
func TestApplyReadMaskList_EmptyHiddenSetNoop(t *testing.T) {
	plans := []*identitypb.Plan{newPopulatedPlan(), newPopulatedPlan()}
	for _, hidden := range []map[string]struct{}{nil, {}} {
		fieldperm.ApplyReadMaskList(plans, hidden)
		for _, plan := range plans {
			assertKeys(t, marshalPlan(t, plan), allPlanJsonKeys, nil)
		}
	}
}

// TestStripWriteFields_MaskPathFiltering 写路径裁剪：
// data 消息按读路径规则清值；field_mask.Paths 中顶层段命中隐藏集的路径
// （含 "field.sub" 只看顶层段的形式）被整条剔除，未命中路径保留。
// 注意路径剔除按字面量查集合：顶层段拼写必须与集合键完全一致。
func TestStripWriteFields_MaskPathFiltering(t *testing.T) {
	tests := []struct {
		name           string
		hidden         map[string]struct{}
		paths          []string
		expectedPaths  []string
		clearedDataKey string
	}{
		{
			name:           "proto spelling hit removes path with and without sub-path",
			hidden:         map[string]struct{}{"created_by": {}},
			paths:          []string{"created_by", "created_by.sub", "description", "remark.sub"},
			expectedPaths:  []string{"description", "remark.sub"},
			clearedDataKey: keyCreatedBy,
		},
		{
			name:           "same-spelling hit removes nested-top path",
			hidden:         map[string]struct{}{"description": {}},
			paths:          []string{"description", "description.deep.sub", "remark"},
			expectedPaths:  []string{"remark"},
			clearedDataKey: keyDescription,
		},
		{
			name:           "json spelling clears data but not mask path",
			hidden:         map[string]struct{}{"createdBy": {}},
			paths:          []string{"created_by", "description"},
			expectedPaths:  []string{"created_by", "description"},
			clearedDataKey: keyCreatedBy,
		},
		{
			name:           "unrelated hidden key keeps everything",
			hidden:         map[string]struct{}{"zzz": {}},
			paths:          []string{"description", "remark"},
			expectedPaths:  []string{"description", "remark"},
			clearedDataKey: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := newPopulatedPlan()
			mask := &fieldmaskpb.FieldMask{Paths: tt.paths}

			fieldperm.StripWriteFields(plan, mask, tt.hidden)

			// data 部分：命中的字段被清值，其余保留。
			keptKeys := []string{}
			for _, k := range allPlanJsonKeys {
				if k != tt.clearedDataKey {
					keptKeys = append(keptKeys, k)
				}
			}
			var absent []string
			if tt.clearedDataKey != "" {
				absent = []string{tt.clearedDataKey}
			}
			assertKeys(t, marshalPlan(t, plan), keptKeys, absent)

			// mask 部分：路径按顶层段命中情况过滤，顺序保持不变。
			assert.Equal(t, tt.expectedPaths, mask.Paths)
		})
	}
}

// TestStripWriteFields_NilAndEmptyMask nil 或空路径的 mask 不做任何剔除，
// 但 data 消息仍按隐藏集正常清值。
func TestStripWriteFields_NilAndEmptyMask(t *testing.T) {
	t.Run("nil mask", func(t *testing.T) {
		plan := newPopulatedPlan()
		var mask *fieldmaskpb.FieldMask
		fieldperm.StripWriteFields(plan, mask, map[string]struct{}{"created_by": {}})
		assertKeys(t, marshalPlan(t, plan), []string{keyDescription, keyRemark}, []string{keyCreatedBy})
		assert.Nil(t, mask)
	})
	t.Run("empty paths mask", func(t *testing.T) {
		plan := newPopulatedPlan()
		mask := &fieldmaskpb.FieldMask{}
		fieldperm.StripWriteFields(plan, mask, map[string]struct{}{"description": {}})
		assertKeys(t, marshalPlan(t, plan), []string{keyCreatedBy, keyRemark}, []string{keyDescription})
		assert.Empty(t, mask.Paths)
	})
}

// TestStripWriteFields_EmptyHiddenSetNoop 空隐藏集下 data 与 mask 均原样不动。
func TestStripWriteFields_EmptyHiddenSetNoop(t *testing.T) {
	for _, hidden := range []map[string]struct{}{nil, {}} {
		plan := newPopulatedPlan()
		mask := &fieldmaskpb.FieldMask{Paths: []string{"description", "remark"}}
		fieldperm.StripWriteFields(plan, mask, hidden)
		assertKeys(t, marshalPlan(t, plan), allPlanJsonKeys, nil)
		assert.Equal(t, []string{"description", "remark"}, mask.Paths)
	}
}

// TestApplyReadMask_ProtectedFieldsSanityBeforeMasking 前置健全性检查：
// 未裁剪前三个字段都以 json_name 拼写出现在 protojson 输出中，
// 保证后续"消失"断言确实源于裁剪而非序列化缺省。
func TestApplyReadMask_ProtectedFieldsSanityBeforeMasking(t *testing.T) {
	serialized := marshalPlan(t, newPopulatedPlan())
	for _, k := range allPlanJsonKeys {
		assert.True(t, strings.Contains(serialized, "\""+k+"\""),
			"字段 %s 初始必须存在", k)
	}
}
