# 脚本引擎与多语言插件系统指南

本文档说明 `pkg/scripting` 的**多语言脚本插件架构**（Lua + JavaScript）与**多来源加载 / 热更新**能力。

## 架构概览

`pkg/scripting` 采用 **Hook 编排器 + 语言适配器** 架构，编排器核心**零语言耦合**：

```
┌───────────────────────────────────────────────────────┐
│  Engine (Hook 编排器, 完全语言无关)                      │
│  ├─ Hook 注册 / 优先级 / 链式执行 / 回调管理              │
│  ├─ 执行上下文 (Context)                                │
│  ├─ 持有 gsEngine.Engine (go-scripts 引擎接口)           │
│  └─ 持有 RuntimeBinder (语言适配器接口)                  │
│       │                                                 │
│       ├── LuaBinder     ← runtime_lua.go (沙箱)         │
│       ├── JSBinder      ← runtime_javascript.go (goja)  │
│       └── 未来: PythonBinder 等                           │
└───────────────────────────────────────────────────────┘
```

**核心抽象（runtime.go）**：
- `RuntimeBinder`：把业务 API / hook.register / 执行上下文注入到特定语言 VM
- `ScriptCallback`：语言无关的回调（Lua 用 `lua.LFunction`，JS 用 goja Value）
- `execCtxHolder`：执行上下文持有器

**切换语言只需改 `Config.EngineType`**，编排器通过 `RuntimeBinder` 接口面向抽象，核心逻辑零语言耦合。

## 多语言切换

```go
import gsEngine "github.com/tx7do/go-scripts"

// Lua（默认）
config := lua.DefaultConfig() // EngineType = gsEngine.LuaType
engine := lua.NewEngine(config, logger)

// JavaScript
config := lua.DefaultConfig()
config.EngineType = gsEngine.JavaScriptType
engine := lua.NewEngine(config, logger)
```

两门语言可**同进程共存**（各自独立的编排器实例）。

### 语言能力对照

| 引擎 | 实现 | 沙箱 | 业务 API | hook.register | 回调调用 |
|------|------|------|---------|--------------|---------|
| **Lua** | gopher-lua | ✅ `SetOpenLibs` 白名单 | ✅ 全部 8 模块 | ✅ | ✅ 直接 LState.PCall |
| **JavaScript** | goja | ⚠️ 无内置沙箱 | ✅ logger/crypto/util | ✅ | ✅ RegisterGlobal+CallFunction |
| Python | gpython | — | ❌ stub 限制 | ❌ | 暂不支持（上游限制）|

### 为何现在能直接用 go-scripts/lua？

go-scripts v0.0.7 新增了 **RuntimeHook** 机制，解决了之前阻碍直接采用的两个核心问题：

| 能力 | 机制 | 说明 |
|------|------|------|
| ✅ **业务 API 注入** | `AddRuntimeHook` | VM 创建后、Load/Execute 前运行，可注入 Redis/EventBus/OSS/Crypto 等 8 个业务模块 |
| ✅ **池隔离** | `recordBusinessGlobal` + `ClearGlobals` | 业务全局变量被追踪，归还池前清除，引擎实例间不泄漏 |
| ✅ **Hook 自注册** | RuntimeHook 注册 `hook.register` Go 函数 | 脚本可调用 `hook.register(name, fn)` 将回调交还 Go 侧 |

**沙箱已启用**（go-scripts/lua v0.0.8+）：通过 `SetOpenLibs` 仅开启安全标准库，
禁用 `os`/`io`/`debug`/`load` 等危险库（命令注入 / 文件系统 / 反射绕过防护）。
安全库白名单：base / load / table / string / math / coroutine。详见 `applySandbox`。

| 库 | 状态 | 原因 |
|----|------|------|
| base | ✅ 开启 | 基础函数（print/pairs/error/require） |
| load (package) | ✅ 开启 | require / module loaders，模块系统必需 |
| table | ✅ 开启 | 表操作 |
| string | ✅ 开启 | 字符串处理 |
| math | ✅ 开启 | 数学运算 |
| coroutine | ✅ 开启 | 协程 |
| **os** | ❌ 禁用 | `os.execute` 命令注入 / `os.remove` 文件删除 / `os.getenv` 环境泄漏 |
| **io** | ❌ 禁用 | 读写任意文件 |
| **debug** | ❌ 禁用 | 反射调试，可绕过元表保护 |

### 架构关键点

- **引擎单 VM**：go-scripts/lua 引擎生命周期内持有单个 VM（statePool.Borrow），回调引用在 Close 前持续有效
- **LoadString vs ExecuteString**：go-scripts 的 `LoadString` 仅编译不执行；本项目 `LoadScriptString` 内部用 `ExecuteString` 以确保 `hook.register` 立即触发
- **模块注册语义**：`engine.RegisterModule(name, loader)` 调用 loader 时传入 name 参数，loader 须执行 `PreloadModule`（见 `preloadAdapter`）

## 多语言切换

通过 `ScriptEngineFactory` 注入引擎实现：

```go
import gsEngine "github.com/tx7do/go-scripts"

// 默认使用 go-scripts/lua（经 init() 自动注册工厂）
engine := lua.NewEngine(lua.DefaultConfig(), logger)

// 自定义引擎工厂（切换语言）
lua.SetEngineFactory(func(config *lua.Config, logger log.Logger) (gsEngine.Engine, error) {
    return gsEngine.NewScriptEngine(config.EngineType) // JavaScriptType 等
})

// 或单次指定
config := lua.DefaultConfig()
config.EngineType = gsEngine.LuaType // 当前支持 lua，未来扩展 javascript 等
```

### 脚本编写约定（适配 go-scripts）

执行上下文通过 RuntimeHook 注入的全局函数访问（非参数）：

```lua
local log = require "kratos_logger"
local hook = require "kratos_hook"

-- 入口函数：execute() 无参数，上下文经 __get_ctx() 获取
function execute()
    local ctx = __get_ctx()
    local input = ctx.get("input")
    ctx.set("result", "processed: " .. input)
    -- ctx.stop("reason")  -- 中止 hook 链
    return true  -- 返回 false 也表示中止
end
```

> 注：hook.register 注册的回调函数仍以 `function(ctx)` 形式接收上下文参数（回调路径直接传递 ctx 表）。

## 脚本来源 (Source)

脚本不再只能从文件系统加载，支持多种来源，统一实现 `source.Reader` 接口：

| 来源 | 类型 | 热更新 | 适用场景 |
|------|------|--------|---------|
| **FileSource** | 本地文件 | ✅ mtime 轮询 | 开发调试 |
| **DBSource** | 数据库 | ✅ 轮询/hash | 生产（管理后台 UI 维护脚本） |
| **MemSource** | 内存 | ✅ channel | 单测、动态推送 |
| go-scripts 其他 | S3/etcd/Redis/HTTP | ✅ 各自机制 | 分布式部署 |

### 文件来源

```go
engine := lua.NewEngine(config, logger)

// 从目录自动加载（向后兼容）
engine.LoadScriptsFromDir(ctx, "./scripts")

// 或绑定 FileSource（支持搜索路径 + 热更新）
src := lua.NewFileSource("./scripts", "./scripts/hooks")
engine.SetSource(src)

// 按 key 加载
engine.LoadScript(ctx, "on_login.lua")

// 启用热更新（文件变更自动重新加载）
engine.WatchScript(ctx, "on_login.lua")
```

### 数据库来源（核心收益）

脚本存储在数据库表中，通过管理后台 UI 增删改，无需重启服务：

```go
// 在 app 层 Wire 装配时注入真正的 DB 查询函数（避免 pkg → data 循环依赖）
dbSource := lua.NewDBSource(
    func(ctx context.Context, key string) (string, error) {
        // 从数据库按 name/id 查询脚本源码
        return scriptRepo.GetSourceByName(ctx, key)
    },
    lua.WithScriptHasher(func(ctx context.Context, key string) (string, error) {
        // 高效变更检测（返回 updated_at 或版本号）
        return scriptRepo.GetUpdatedAt(ctx, key)
    }),
    lua.WithPollInterval(5*time.Second), // 热更新轮询间隔
)
defer dbSource.Close()

engine.SetSource(dbSource)

// 加载并监听
engine.LoadScript(ctx, "user_registered")
engine.WatchScript(ctx, "user_registered") // DB 变更自动重载
```

**DBSource 特性**：
- 内存缓存：避免每次 Load 都查库
- `Invalidate(key)` / `InvalidateAll()`：主动失效缓存
- 热更新：`hasher` 优先（高效），否则比对源码字符串

### 缓存层（远程源推荐）

对远程源（DB/S3/etcd）可用 go-scripts 的 `CachedSource` 包裹，减少 IO：

```go
import gsSource "github.com/tx7do/go-scripts/source"

cached, _ := gsSource.NewCachedSource(dbSource, gsSource.WithTTL(5*time.Minute))
engine.SetSource(cached)
```

## 引擎接口能力

go-scripts/lua 引擎实现完整的 `Engine` 接口（7 个能力子接口）：

| 能力 | 方法 | 说明 |
|------|------|------|
| 生命周期 | `Init` / `Close` / `IsInitialized` | 引擎初始化与释放 |
| 脚本加载 | `Load` / `LoadMulti` / `LoadString` | 从 Source 或内联加载（仅编译） |
| 脚本执行 | `Execute` / `ExecuteFromKey` / `ExecuteString` | 执行并返回结果 |
| 全局访问 | `RegisterGlobal` / `GetGlobal` | 全局变量读写 |
| 函数注册 | `RegisterFunction` / `CallFunction` | 宿主函数注册与调用 |
| 模块注册 | `RegisterModule` | Lua 模块（preload 风格 loader） |
| 热更新 | `StartWatch` / `StopWatch` | 脚本变更自动重载 |
| **RuntimeHook** | `AddRuntimeHook` | **业务 API / hook.register / 执行上下文注入** |

## Lua ↔ JavaScript 脚本编写对照

两门语言的脚本 API 完全一致（同一套业务模块、hook.register、执行上下文）：

| 操作 | Lua | JavaScript |
|------|-----|-----------|
| 入口函数 | `function execute() ... end` | `function execute() { ... }` |
| 获取上下文 | `local ctx = __get_ctx()` | `const ctx = __get_ctx()` |
| 读取数据 | `ctx.get("key")` | `ctx.get("key")` |
| 写入数据 | `ctx.set("key", val)` | `ctx.set("key", val)` |
| 中止 | `ctx.stop("reason")` 或 `return false` | `ctx.stop("reason")` 或 `return false` |
| 注册 hook | `hook.register("name", "desc", fn)` | `hook.register("name", "desc", fn)` |
| 日志 | `require "kratos_logger"; log.info(...)` | `log.info(...)`（直接全局对象）|
| 加密 | `require "kratos_crypto"; crypto.hash_sha256(...)` | `crypto.hash_sha256(...)`（全局）|

### Lua 示例

```lua
local log = require "kratos_logger"
local hook = require "kratos_hook"

hook.register("on_login", "用户登录钩子", function(ctx)
    local user = ctx.get("user")
    log.info("User logged in: " .. tostring(user))
    ctx.set("processed", true)
    return true
end)
```

### JavaScript 示例

```javascript
hook.register("on_login", "用户登录钩子", function(ctx) {
    var user = ctx.get("user");
    log.info("User logged in: " + user);
    ctx.set("processed", true);
    return true;
});
```

## 向后兼容

编排器 API 保持兼容：
- `NewEngine(config, logger)` — 创建编排器（默认 Lua）
- `ExecuteHook(ctx, hookName, execCtx)` — Hook 执行
- `AddScript` / `RegisterHook` / `RegisterCallback` — Hook 管理
- `LoadScriptsFromDir` / `LoadScriptFile` / `LoadScriptString` — 脚本加载
- `SetRedis` / `SetEventBus` / `SetOSS` — 业务依赖注入

## 文件结构

```
pkg/scripting/
├── engine.go              # Hook 编排器（完全语言无关，零 Lua 依赖）
├── runtime.go             # 语言无关抽象层（ScriptCallback/RuntimeBinder/RuntimeDeps）
├── runtime_lua.go         # Lua 适配器（LuaBinder + luaCallback + 沙箱）
├── runtime_javascript.go  # JS 适配器（JSBinder + jsCallback）
├── source_file.go         # FileSource（包装 go-scripts FileSource）
├── source_db.go           # DBSource（数据库 + 热更新）
├── script.go / context.go # 数据结构（语言无关）
├── hook/registry.go       # Hook 系统（语言无关）
├── api/                   # 业务 API 模块
│   ├── module.go          #   ModuleDef（语言无关，供 JS 用）
│   └── *.go               #   Loader*（Lua 专用）+ Module*（语言无关）
└── internal/convert/      # Lua ↔ Go 转换
```

