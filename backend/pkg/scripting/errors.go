package scripting

// 本文件原承载自研 LuaEngine 的哨兵错误与适配器。
// 切换到 go-scripts/lua 后，引擎错误由 go-scripts 提供（ErrLuaEngineNotInitialized 等），
// hook 引擎适配器移至 engine_runtime.go，故本文件仅保留包级注释占位。
//
// 如未来需要 pkg/scripting 自有的错误类型，可在此定义。
