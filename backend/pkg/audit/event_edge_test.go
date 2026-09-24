// 本文件补 pkg/audit 的事件上下文与脱敏词法边角分支：
//
//	event.go 四个 context 辅助函数（AccumulatorKey/SinkKey/FromContext/IsSinking）
//	此前零覆盖——它们是采集管道（wrapper→accumulator→落库侧防递归）的键位约定；
//	sqlmask 的未闭合字符串/未闭合引号标识符/嵌套与未闭合块注释/美元符号边角/
//	十六进制数值等词法分支。
package audit

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func contextBackground() context.Context { return context.Background() }
func contextWithValue(ctx context.Context, k, v any) context.Context {
	return context.WithValue(ctx, k, v)
}

// TestEventContextKeys accumulator/sink 键位约定与取回语义：
// 值经 context.WithValue 植入后 FromContext/IsSinking 必须按原语义取回；
// 无植入时 FromContext 不命中、IsSinking 为 false。
func TestEventContextKeys(t *testing.T) {
	require.NotNil(t, AccumulatorKey())
	require.NotNil(t, SinkKey())

	events := make([]AuditEvent, 0)
	ctx := contextWithValue(contextBackground(), AccumulatorKey(), &events)
	got, ok := FromContext(ctx)
	require.True(t, ok, "植入 accumulator 后 FromContext 应命中")
	require.Same(t, &events, got)

	_, ok = FromContext(contextBackground())
	require.False(t, ok, "未植入时 FromContext 不得命中")

	require.False(t, IsSinking(contextBackground()), "未植入 sink 标记时 IsSinking 应为 false")
	sinkCtx := contextWithValue(contextBackground(), SinkKey(), true)
	require.True(t, IsSinking(sinkCtx), "植入 sink 标记后 IsSinking 应为 true")
}

// TestMaskSQSLexerEdges 脱敏词法的边角分支：
// 未闭合字符串脱敏到末尾、引号标识符内的 "" 转义与未闭合标识符原样保留、
// 嵌套与未闭合块注释原样保留、行注释原样保留、$1 处于串末尾仍按占位符保留、
// 孤立 $ 原样单字节、美元引用整体脱敏、十六进制数值按数值脱敏。
func TestMaskSQLLexerEdges(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "未闭合字符串脱敏到末尾",
			in:   "WHERE a = 'abc",
			want: "WHERE a = ***",
		},
		{
			name: "引号标识符内双引号转义原样保留",
			in:   `SELECT "a""b" FROM t`,
			want: `SELECT "a""b" FROM t`,
		},
		{
			name: "未闭合引号标识符原样保留到末尾",
			in:   `SELECT "unterminated`,
			want: `SELECT "unterminated`,
		},
		{
			name: "嵌套块注释原样保留（其中数字仍脱敏）",
			in:   "SELECT /* a /* b */ c */ 1",
			want: "SELECT /* a /* b */ c */ ***",
		},
		{
			name: "未闭合块注释原样保留到末尾（前导数字仍脱敏）",
			in:   "SELECT 1 /* xx",
			want: "SELECT *** /* xx",
		},
		{
			name: "行注释原样保留（前导数字仍脱敏）",
			in:   "SELECT 1 -- trailing comment",
			want: "SELECT *** -- trailing comment",
		},
		{
			name: "占位符处于串末尾",
			in:   "WHERE id = $1",
			want: "WHERE id = $1",
		},
		{
			name: "孤立美元符号原样保留",
			in:   "a $ b",
			want: "a $ b",
		},
		{
			name: "美元引用整体脱敏",
			in:   "SELECT $tag$secret body$tag$ FROM t",
			want: "SELECT *** FROM t",
		},
		{
			name: "空标签美元引用整体脱敏",
			in:   "SELECT $$secret$$ FROM t",
			want: "SELECT *** FROM t",
		},
		{
			name: "十六进制数值脱敏",
			in:   "WHERE a = 0xFF",
			want: "WHERE a = ***",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, MaskSQL(tc.in))
		})
	}
}
