package api

import (
	"time"

	"github.com/go-kratos/kratos/v2/log"
	lua "github.com/yuin/gopher-lua"
)

// RegisterUtilAPI registers utility functions for Lua scripts
// Provides common utilities like sleep, time, etc.
func RegisterUtilAPI(L *lua.LState, logger *log.Helper) {
	// Register as requireable module
	L.PreloadModule("kratos_util", LoaderUtil(logger))

	if logger != nil {
		logger.Debug("Registered Lua util API")
	}
}

// LoaderUtil 返回 util 模块（kratos_util）的 loader，供 go-scripts 引擎 RegisterModule 使用。
func LoaderUtil(logger *log.Helper) lua.LGFunction {
	return func(L *lua.LState) int {
		// Create util module
		utilModule := L.NewTable()

		// util.sleep(seconds)
		// Sleep for the specified number of seconds (can be fractional)
		utilModule.RawSetString("sleep", L.NewFunction(func(L *lua.LState) int {
			seconds := L.CheckNumber(1)
			duration := time.Duration(float64(seconds) * float64(time.Second))

			if logger != nil {
				logger.Debugf("Lua sleep: %v", duration)
			}

			time.Sleep(duration)
			return 0
		}))

		// util.time()
		// Returns current Unix timestamp
		utilModule.RawSetString("time", L.NewFunction(func(L *lua.LState) int {
			L.Push(lua.LNumber(time.Now().Unix()))
			return 1
		}))

		// util.timestamp()
		// Returns current Unix timestamp in milliseconds
		utilModule.RawSetString("timestamp", L.NewFunction(func(L *lua.LState) int {
			L.Push(lua.LNumber(time.Now().UnixMilli()))
			return 1
		}))

		// util.date(format)
		// Returns formatted date string (default: RFC3339)
		utilModule.RawSetString("date", L.NewFunction(func(L *lua.LState) int {
			format := L.OptString(1, time.RFC3339)
			L.Push(lua.LString(time.Now().Format(format)))
			return 1
		}))

		L.Push(utilModule)
		return 1
	}
}
