# 脚本引擎与多来源加载指南

本文档说明 `pkg/lua` 升级后的**多语言脚本引擎架构**与**多来源加载 / 热更新**能力。

## 架构概览

升级后的 `pkg/lua` 采用 **Hook 编排器 + 脚本引擎** 双层架构：

```
┌───────────────────────────────────────────────┐
│  Engine (Hook 编排器, 语言无关)                  │
│  ├─ Hook 注册 / 优先级 / 链式执行 / 回调管理       │
│  ├─ 执行上下文 (Context)                         │
│  └─ 持有 gsEngine.Engine (脚本引擎接口)           │
│       │                                         │
│       ├── LuaEngine   ← 当前实现 (安全沙箱)       │
│       ├── JSEngine    ← 未来 (goja)             │
│       └── PythonEngine ← 未来 (gpython)         │
└───────────────────────────────────────────────┘
```

**关键点**：`LuaEngine` 实现了 [go-scripts](https://github.com/tx7do/go-scripts) 的 `Engine` 接口。
未来添加 JS/Python 只需新增一个实现同一接口的引擎，编排器代码无需改动。

### 为什么不直接用 go-scripts/lua？

不使用 `go-scripts/lua` 子模块有三层原因（按重要性排序）：

| 原因 | 能否绕过 | 说明 |
|------|---------|------|
| **业务 API 无注入点** ⭐ | 难 | VM 创建（`virtualMachine.init()`）写死了加载哪些库，没有钩子点注入 Redis/EventBus/OSS/Crypto 等 8 个业务模块，且需每个 VM 都注册 |
| **Hook 自注册** | 不可绕过 | 本项目脚本从 Lua 侧 `hook.register(name, fn)` 自注册回调，go-scripts 是 Go 单向注册，无此能力 |
| **OpenLibs 安全** | 可（fork 改一行） | VM 全开 `os.execute`/`io`/`loadfile` 等，对插件系统是安全漏洞 |

> 另有次要因素：`go-scripts/lua` 的 go.mod 拖入 gopher-lua-libs（含 aws-sdk）、prometheus 等重依赖，
> 会膨胀本项目 go.mod。

**结论**：业务 API 注入 + Hook 自注册的改造成本 ≈ 自研 LuaEngine。
自研实现 go-scripts.Engine 接口，既拿到统一接口红利（多语言切换、source、热更新），
又保留沙箱与业务集成的完全控制权。

## 多语言切换

通过 `ScriptEngineFactory` 注入引擎实现：

```go
import gsEngine "github.com/tx7do/go-scripts"

// 默认使用 Lua
engine := lua.NewEngine(lua.DefaultConfig(), logger)

// 自定义引擎工厂（切换语言）
lua.SetEngineFactory(func(config *lua.Config, logger log.Logger) (gsEngine.Engine, error) {
    // 返回你的引擎实现（需满足 gsEngine.Engine 接口）
    return myJSEngine.New(config, logger), nil
})

// 或单次指定
config := lua.DefaultConfig()
config.EngineType = lua.EngineTypeLua // 当前支持 lua，未来扩展 javascript 等
```

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

`LuaEngine` 实现 go-scripts 的完整 `Engine` 接口（7 个能力子接口）：

| 能力 | 方法 | 说明 |
|------|------|------|
| 生命周期 | `Init` / `Close` / `IsInitialized` | 引擎初始化与释放 |
| 脚本加载 | `Load` / `LoadMulti` / `LoadString` | 从 Source 或内联加载 |
| 脚本执行 | `Execute` / `ExecuteFromKey` / `ExecuteString` | 执行并返回结果 |
| 全局访问 | `RegisterGlobal` / `GetGlobal` | 全局变量读写 |
| 函数注册 | `RegisterFunction` / `CallFunction` | 宿主函数注册与调用 |
| 模块注册 | `RegisterModule` | Lua 模块（require） |
| 热更新 | `StartWatch` / `StopWatch` | 脚本变更自动重载 |

> **池一致性**：`RegisterFunction` / `RegisterGlobal` / `RegisterModule` 会记录到全局注册表，
> 新建 VM 自动重放，并应用到当前池中所有 VM，保证池内状态一致。

## 向后兼容

原有 API 保持兼容：
- `NewEngine(config, logger)` — 创建编排器（默认 Lua）
- `ExecuteHook(ctx, hookName, execCtx)` — Hook 执行
- `AddScript` / `RegisterHook` / `RegisterCallback` — Hook 管理
- `LoadScriptsFromDir` / `LoadScriptFile` / `LoadScriptString` — 脚本加载
- `SetRedis` / `SetEventBus` / `SetOSS` — 业务依赖注入

## 文件结构

```
pkg/lua/
├── engine.go              # Hook 编排器（语言无关）
├── engine_lua.go          # LuaEngine（实现 go-scripts.Engine 接口）
├── vm_pool.go             # VM 池（借鉴 go-scripts EnginePool 健壮性）
├── source_file.go         # FileSource（包装 go-scripts FileSource）
├── source_db.go           # DBSource（数据库 + 热更新）
├── loader_compat.go       # 向后兼容的目录遍历辅助
├── errors.go              # 哨兵错误 + 类型转换辅助
├── script.go / context.go # 数据结构（不变）
├── hook/registry.go       # Hook 系统（不变）
├── api/                   # 8 个业务 API 模块（不变）
└── internal/convert/      # Lua ↔ Go 转换（不变）
```
