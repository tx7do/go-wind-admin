package api

// 本文件针对 logger.go 做单元测试：
//   - convertForFormat：float64 整数值转 int、其余（小数/Inf/NaN/非数值）原样返回的直测表；
//   - RegisterLogger / LoaderLogger：经 require "kratos_logger" 加载模块，
//     通过记录型 logger 断言 info/warn/error/debug 与四个格式化变体的
//     等级归属、消息内容，以及 Lua number（float64）经 convertForFormat 转为 int 后的格式化结果。
//
// 记录型 logger 实现项目统一 Logger 接口，仅按调用顺序记录 (等级, 消息)，供断言。

import (
	"context"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	lua "github.com/yuin/gopher-lua"
)

// recLogEntry 记录一次日志调用的等级与消息。
type recLogEntry struct {
	level string
	msg   string
}

// recordingLogger 记录全部日志调用的 Logger 实现（测试专用，非并发安全）。
type recordingLogger struct {
	entries []recLogEntry
}

func (r *recordingLogger) Debug(_ context.Context, msg string, _ ...any) {
	r.entries = append(r.entries, recLogEntry{"debug", msg})
}

func (r *recordingLogger) Info(_ context.Context, msg string, _ ...any) {
	r.entries = append(r.entries, recLogEntry{"info", msg})
}

func (r *recordingLogger) Warn(_ context.Context, msg string, _ ...any) {
	r.entries = append(r.entries, recLogEntry{"warn", msg})
}

func (r *recordingLogger) Error(_ context.Context, msg string, _ ...any) {
	r.entries = append(r.entries, recLogEntry{"error", msg})
}

func (r *recordingLogger) With(_ ...any) bLogger.Logger { return r }

// snapshot 返回已记录条目的拷贝。
func (r *recordingLogger) snapshot() []recLogEntry {
	out := make([]recLogEntry, len(r.entries))
	copy(out, r.entries)
	return out
}

// TestConvertForFormat 直测 float64 整数 → int 的转换与其余类型的透传。
// NaN/Inf 与小数必须保持 float64 原样（整数值判定为 Floor 相等且非 Inf/NaN）。
func TestConvertForFormat(t *testing.T) {
	t.Run("whole float converts to int", func(t *testing.T) {
		require.Equal(t, 42, convertForFormat(float64(42)))
		require.Equal(t, -3, convertForFormat(float64(-3)))
		require.Equal(t, 0, convertForFormat(float64(0)))
	})
	t.Run("fractional float passes through", func(t *testing.T) {
		require.Equal(t, float64(3.5), convertForFormat(float64(3.5)))
		require.Equal(t, float64(-0.25), convertForFormat(float64(-0.25)))
	})
	t.Run("Inf and NaN pass through", func(t *testing.T) {
		inf := convertForFormat(math.Inf(1))
		require.IsType(t, float64(0), inf)
		require.True(t, math.IsInf(inf.(float64), 1))
		nan := convertForFormat(math.NaN())
		require.IsType(t, float64(0), nan)
		require.True(t, math.IsNaN(nan.(float64)))
	})
	t.Run("non float values pass through unchanged", func(t *testing.T) {
		require.Equal(t, "str", convertForFormat("str"))
		require.Equal(t, 7, convertForFormat(7))
		require.Equal(t, true, convertForFormat(true))
		require.Equal(t, nil, convertForFormat(nil))
	})
}

// TestLoaderLogger_NonFormatLevels 四个非格式化日志函数经 Lua 模块调用后，
// 记录型 logger 应按顺序收到对应等级与原样消息。
func TestLoaderLogger_NonFormatLevels(t *testing.T) {
	rec := &recordingLogger{}
	L := lua.NewState()
	defer L.Close()
	RegisterLogger(L, bLogger.NewHelper(rec))

	err := L.DoString(`
		local log = require "kratos_logger"
		log.info("m-info")
		log.warn("m-warn")
		log.error("m-error")
		log.debug("m-debug")
	`)
	require.NoError(t, err)
	require.Equal(t, []recLogEntry{
		{"info", "m-info"},
		{"warn", "m-warn"},
		{"error", "m-error"},
		{"debug", "m-debug"},
	}, rec.snapshot())
}

// TestLoaderLogger_FormatArgsConversion 格式化变体：Lua number 以 float64 进入转换层，
// 整数值应被转为 int 使 %d 可用；小数保持 float64 由 %v 输出；字符串原样输出。
// 该测试同时证明消息确经 Helper.Infof 完成格式化。
func TestLoaderLogger_FormatArgsConversion(t *testing.T) {
	rec := &recordingLogger{}
	L := lua.NewState()
	defer L.Close()
	RegisterLogger(L, bLogger.NewHelper(rec))

	err := L.DoString(`
		local log = require "kratos_logger"
		log.infof("n=%d", 3)
		log.warnf("a=%d b=%d", 1, 2)
		log.errorf("f=%v", 3.5)
		log.debugf("s=%s!", "txt")
		log.infof("plain-no-args")
	`)
	require.NoError(t, err)
	require.Equal(t, []recLogEntry{
		{"info", "n=3"},
		{"warn", "a=1 b=2"},
		{"error", "f=3.5"},
		{"debug", "s=txt!"},
		{"info", "plain-no-args"},
	}, rec.snapshot())
}
