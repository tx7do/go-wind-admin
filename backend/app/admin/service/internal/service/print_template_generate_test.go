package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func genTemplate(t *testing.T, payload string) (string, []string) {
	t.Helper()
	clean, fields, err := sanitizeGeneratedTemplate(payload)
	require.NoError(t, err)
	return clean, fields
}

func TestSanitizeGeneratedTemplate_AcceptsValid(t *testing.T) {
	payload := `{"unit":"mm","paperSize":"A4","orientation":"portrait","schemaVersion":1,` +
		`"margins":{"top":10,"right":10,"bottom":10,"left":10},` +
		`"header":{"height":10,"elements":[]},"footer":{"height":10,"elements":[]},` +
		`"watermark":{},"guides":[],"name":"出库单","elements":[` +
		`{"id":"el-title","type":"text","options":{"left":55,"top":10,"width":100,"height":12,` +
		`"formatter":"出库单 {orderNo}","fontSize":16},"printElementType":{"type":"text","title":"文本"}}]}`

	clean, fields := genTemplate(t, payload)

	var out map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(clean), &out))
	assert.Equal(t, "A4", out["paperSize"])
	assert.InDelta(t, 1, out["schemaVersion"], 0.001)
	assert.Equal(t, []string{"orderNo"}, fields, "应提取 formatter 里的 {field} 引用")
}

func TestSanitizeGeneratedTemplate_RejectsStructuralViolations(t *testing.T) {
	cases := []struct {
		name    string
		payload string
		wantErr string
	}{
		{
			name:    "未知顶层键",
			payload: `{"unit":"mm","hacker":"x"}`,
			wantErr: "unexpected top-level key",
		},
		{
			name:    "单位不是毫米",
			payload: `{"unit":"pt"}`,
			wantErr: "unit must be",
		},
		{
			name:    "未知纸张",
			payload: `{"unit":"mm","paperSize":"A9"}`,
			wantErr: "unsupported paperSize",
		},
		{
			name:    "方向非法",
			payload: `{"unit":"mm","orientation":"diagonal"}`,
			wantErr: "orientation",
		},
		{
			name:    "未知元素类型（含 html 注入面）",
			payload: `{"unit":"mm","elements":[{"id":"e1","type":"html","options":{}}]}`,
			wantErr: "disallowed type",
		},
		{
			name:    "元素未知键",
			payload: `{"unit":"mm","elements":[{"id":"e1","type":"text","onload":"evil()","options":{}}]}`,
			wantErr: "unexpected key",
		},
		{
			name:    "元素缺 id",
			payload: `{"unit":"mm","elements":[{"type":"text","options":{}}]}`,
			wantErr: "missing id",
		},
		{
			name:    "元素缺 options",
			payload: `{"unit":"mm","elements":[{"id":"e1","type":"text"}]}`,
			wantErr: "missing options",
		},
		{
			name:    "margins 非数值",
			payload: `{"unit":"mm","margins":{"top":"<script>"}}`,
			wantErr: "margins.top",
		},
		{
			name:    "非 JSON",
			payload: `not json at all`,
			wantErr: "not valid json",
		},
		{
			name:    "空串",
			payload: `   `,
			wantErr: "empty",
		},
		{
			name:    "尾随数据",
			payload: `{"unit":"mm"} {"unit":"mm"}`,
			wantErr: "trailing data",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := sanitizeGeneratedTemplate(tc.payload)
			require.Error(t, err, "结构性越界必须拒绝")
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

func TestSanitizeGeneratedTemplate_Clamps(t *testing.T) {
	// 超宽页边距钳到 50；元素几何钳到 2000；name 超长截断
	payload := `{"unit":"mm","margins":{"top":500,"right":-1,"bottom":10,"left":10},` +
		`"name":"` + strings.Repeat("长", 300) + `","elements":[` +
		`{"id":"e1","type":"rect","options":{"left":99999,"top":5,"width":50,"height":50}}]}`
	clean, _, err := sanitizeGeneratedTemplate(payload)
	require.NoError(t, err, "几何越界按钳制收敛而非拒绝")

	var out map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(clean), &out))
	margins := out["margins"].(map[string]interface{})
	assert.InDelta(t, 50, margins["top"], 0.001, "超限边距钳到 50")
	assert.InDelta(t, 0, margins["right"], 0.001, "负边距钳到 0")
	assert.InDelta(t, 10, margins["bottom"], 0.001, "合法边距保持不变")

	assert.Len(t, []rune(out["name"].(string)), 200, "name 按 rune 截断到 200")

	elem := out["elements"].([]interface{})[0].(map[string]interface{})
	opts := elem["options"].(map[string]interface{})
	assert.InDelta(t, 2000, opts["left"], 0.001, "超限几何钳到 2000")
	assert.InDelta(t, 5, opts["top"], 0.001, "合法几何保持不变")
}

func TestSanitizeGeneratedTemplate_Limits(t *testing.T) {
	// 元素数超限
	var elems []string
	for i := 0; i < printGenMaxElements+1; i++ {
		elems = append(elems, `{"id":"e","type":"text","options":{}}`)
	}
	_, _, err := sanitizeGeneratedTemplate(`{"unit":"mm","elements":[` + strings.Join(elems, ",") + `]}`)
	require.Error(t, err, "元素数超限必须拒绝")
	assert.Contains(t, err.Error(), "too many elements")

	// 嵌套过深
	deep := strings.Repeat(`{"a":`, printGenMaxJSONDepth+2) + "1" + strings.Repeat(`}`, printGenMaxJSONDepth+2)
	_, _, err = sanitizeGeneratedTemplate(`{"unit":"mm","watermark":` + deep + `}`)
	require.Error(t, err, "嵌套过深必须拒绝")
	assert.Contains(t, err.Error(), "nesting too deep")
}

func TestSanitizeGeneratedTemplate_FieldRefs(t *testing.T) {
	payload := `{"unit":"mm","elements":[` +
		`{"id":"a","type":"text","options":{"formatter":"{orderNo} / {customerName}"}},` +
		`{"id":"b","type":"barcode","options":{"formatter":"{orderNo}"}}]}`

	clean, fields := genTemplate(t, payload)
	assert.Equal(t, []string{"customerName", "orderNo"}, fields, "引用集合去重且有序")
	assert.Contains(t, clean, "barcode", "条码类型在白名单内应保留")

	// 表达式形态不放行（AI 生成面只收标识符占位符）
	_, _, err := sanitizeGeneratedTemplate(`{"unit":"mm","elements":[{"id":"a","type":"text","options":{"formatter":"{a.b.c}"}}]}`)
	require.NoError(t, err, "{a.b.c} 只是没被识别为引用，不应整体拒绝")
}

func TestExtractTemplatePayload(t *testing.T) {
	assert.Equal(t, `{"a":1}`, extractTemplatePayload("```json\n{\"a\":1}\n```"))
	assert.Equal(t, `{"a":1}`, extractTemplatePayload("好的，这是模板：\n{\"a\":1}\n以上。"))
	assert.Equal(t, `{"a":1}`, extractTemplatePayload(`{"a":1}`))
}
