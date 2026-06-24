package api

import (
	"math"

	"github.com/go-kratos/kratos/v2/log"
	lua "github.com/yuin/gopher-lua"

	"go-wind-admin/pkg/scripting/internal/convert"
)

// convertForFormat converts values to be format-string friendly.
// Specifically, converts float64 to int if it's a whole number,
// since Lua numbers are always float64 but Go format %d expects int.
func convertForFormat(val interface{}) interface{} {
	if f, ok := val.(float64); ok {
		// If the float is a whole number, convert to int for %d compatibility
		if f == math.Floor(f) && !math.IsInf(f, 0) && !math.IsNaN(f) {
			return int(f)
		}
	}
	return val
}

// RegisterLogger registers the logger API for Lua as a requireable module
// RegisterLogger registers the logger API for Lua as a requireable module
func RegisterLogger(L *lua.LState, logger *log.Helper) {
	// Register in package.preload so it can be required
	L.PreloadModule("kratos_logger", LoaderLogger(logger))
}

// LoaderLogger 返回 logger 模块（kratos_logger）的 loader，供 go-scripts 引擎 RegisterModule 使用。
func LoaderLogger(logger *log.Helper) lua.LGFunction {
	return func(L *lua.LState) int {
		// Create log module
		logModule := L.NewTable()

		// log.info(message)
		logModule.RawSetString("info", L.NewFunction(func(L *lua.LState) int {
			msg := L.CheckString(1)
			logger.Info(msg)
			return 0
		}))

		// log.warn(message)
		logModule.RawSetString("warn", L.NewFunction(func(L *lua.LState) int {
			msg := L.CheckString(1)
			logger.Warn(msg)
			return 0
		}))

		// log.error(message)
		logModule.RawSetString("error", L.NewFunction(func(L *lua.LState) int {
			msg := L.CheckString(1)
			logger.Error(msg)
			return 0
		}))

		// log.debug(message)
		logModule.RawSetString("debug", L.NewFunction(func(L *lua.LState) int {
			msg := L.CheckString(1)
			logger.Debug(msg)
			return 0
		}))

		// log.infof(format, args...)
		logModule.RawSetString("infof", L.NewFunction(func(L *lua.LState) int {
			format := L.CheckString(1)
			args := make([]interface{}, L.GetTop()-1)
			for i := 2; i <= L.GetTop(); i++ {
				args[i-2] = convertForFormat(convert.ToGoValue(L.Get(i)))
			}
			logger.Infof(format, args...)
			return 0
		}))

		// log.errorf(format, args...)
		logModule.RawSetString("errorf", L.NewFunction(func(L *lua.LState) int {
			format := L.CheckString(1)
			args := make([]interface{}, L.GetTop()-1)
			for i := 2; i <= L.GetTop(); i++ {
				args[i-2] = convertForFormat(convert.ToGoValue(L.Get(i)))
			}
			logger.Errorf(format, args...)
			return 0
		}))

		// log.warnf(format, args...)
		logModule.RawSetString("warnf", L.NewFunction(func(L *lua.LState) int {
			format := L.CheckString(1)
			args := make([]interface{}, L.GetTop()-1)
			for i := 2; i <= L.GetTop(); i++ {
				args[i-2] = convertForFormat(convert.ToGoValue(L.Get(i)))
			}
			logger.Warnf(format, args...)
			return 0
		}))

		// log.debugf(format, args...)
		logModule.RawSetString("debugf", L.NewFunction(func(L *lua.LState) int {
			format := L.CheckString(1)
			args := make([]interface{}, L.GetTop()-1)
			for i := 2; i <= L.GetTop(); i++ {
				args[i-2] = convertForFormat(convert.ToGoValue(L.Get(i)))
			}
			logger.Debugf(format, args...)
			return 0
		}))

		L.Push(logModule)
		return 1
	}
}
