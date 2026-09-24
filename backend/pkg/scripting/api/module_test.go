package api

// 本文件针对 module.go 做单元测试：
//   - convertForFormatVal / convertForFormatArgs：float64 整数值转 int 的直测表
//     （Inf/NaN/小数/非数值必须原样透传）；
//   - ModuleLogger：模块定义的结构（名称、八个函数键、函数类型）与调用行为
//     （经由记录型 logger 断言各等级归属、消息内容，以及格式化参数经
//     convertForFormatArgs 转换后的结果）。

import (
	"math"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
)

// TestConvertForFormatVal 直测 float64 整数 → int 的转换与其余类型的透传。
func TestConvertForFormatVal(t *testing.T) {
	t.Run("whole float converts to int", func(t *testing.T) {
		require.Equal(t, 42, convertForFormatVal(float64(42)))
		require.Equal(t, -3, convertForFormatVal(float64(-3)))
	})
	t.Run("fractional float passes through", func(t *testing.T) {
		require.Equal(t, float64(3.5), convertForFormatVal(float64(3.5)))
	})
	t.Run("Inf and NaN pass through", func(t *testing.T) {
		inf := convertForFormatVal(math.Inf(-1))
		require.IsType(t, float64(0), inf)
		require.True(t, math.IsInf(inf.(float64), -1))
		nan := convertForFormatVal(math.NaN())
		require.IsType(t, float64(0), nan)
		require.True(t, math.IsNaN(nan.(float64)))
	})
	t.Run("non float values pass through unchanged", func(t *testing.T) {
		require.Equal(t, "str", convertForFormatVal("str"))
		require.Equal(t, true, convertForFormatVal(true))
		require.Equal(t, nil, convertForFormatVal(nil))
	})
}

// TestConvertForFormatArgs 直测 variadic 参数批量转换：
// 整数 float 转为 int、其余元素原样保留，空切片保持为空。
func TestConvertForFormatArgs(t *testing.T) {
	t.Run("mixed types converted element-wise", func(t *testing.T) {
		got := convertForFormatArgs([]any{float64(1), "a", float64(2.5), true})
		require.Equal(t, int(1), got[0])
		require.Equal(t, "a", got[1])
		require.Equal(t, float64(2.5), got[2])
		require.Equal(t, true, got[3])
		require.Len(t, got, 4)
	})
	t.Run("empty slice stays empty", func(t *testing.T) {
		got := convertForFormatArgs([]any{})
		require.Empty(t, got)
	})
	t.Run("whole floats all converted", func(t *testing.T) {
		got := convertForFormatArgs([]any{float64(-7), float64(100)})
		require.Equal(t, int(-7), got[0])
		require.Equal(t, int(100), got[1])
	})
}

// TestModuleLogger_Structure 校验模块定义：名称为 "log"，包含且仅包含八个日志函数键。
func TestModuleLogger_Structure(t *testing.T) {
	rec := &recordingLogger{}
	mod := ModuleLogger(bLogger.NewHelper(rec))

	require.Equal(t, "log", mod.Name)
	require.Len(t, mod.Funcs, 8)
	for _, name := range []string{"info", "warn", "error", "debug", "infof", "warnf", "errorf", "debugf"} {
		fn, ok := mod.Funcs[name]
		require.True(t, ok, "ModuleLogger should expose %q", name)
		require.Equal(t, reflect.Func, reflect.TypeOf(fn).Kind(), "Funcs[%q] should be a function", name)
	}
}

// TestModuleLogger_Invocation 调用八个函数并经记录型 logger 断言：
// 非格式化函数按等级透传消息；格式化函数中整数 float 被转为 int（%d 输出），
// 小数与 NaN 保持 float64（%v 输出 3.5 / NaN）。
func TestModuleLogger_Invocation(t *testing.T) {
	rec := &recordingLogger{}
	mod := ModuleLogger(bLogger.NewHelper(rec))

	info, ok := mod.Funcs["info"].(func(string))
	require.True(t, ok)
	info("go-info")
	warn, ok := mod.Funcs["warn"].(func(string))
	require.True(t, ok)
	warn("go-warn")
	errFn, ok := mod.Funcs["error"].(func(string))
	require.True(t, ok)
	errFn("go-error")
	dbg, ok := mod.Funcs["debug"].(func(string))
	require.True(t, ok)
	dbg("go-debug")

	infof, ok := mod.Funcs["infof"].(func(string, ...any))
	require.True(t, ok)
	infof("i=%d", float64(3))
	warnf, ok := mod.Funcs["warnf"].(func(string, ...any))
	require.True(t, ok)
	warnf("w=%s%d", "x", float64(9))
	errorf, ok := mod.Funcs["errorf"].(func(string, ...any))
	require.True(t, ok)
	errorf("e=%v", float64(2.5))
	debugf, ok := mod.Funcs["debugf"].(func(string, ...any))
	require.True(t, ok)
	debugf("d=%v", math.NaN())

	require.Equal(t, []recLogEntry{
		{"info", "go-info"},
		{"warn", "go-warn"},
		{"error", "go-error"},
		{"debug", "go-debug"},
		{"info", "i=3"},
		{"warn", "w=x9"},
		{"error", "e=2.5"},
		{"debug", "d=NaN"},
	}, rec.snapshot())
}
