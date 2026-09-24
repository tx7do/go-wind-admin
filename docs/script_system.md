# 脚本系统（脚本级插件系统）

GoWind Admin 内置一套以 **Lua / JavaScript** 为载体的脚本级插件系统：平台管理员可以在
不改主版本代码的前提下，编写轻量脚本来扩展平台行为——响应业务事件、注册定时任务、
对外推送 Webhook 等。脚本以数据库为唯一事实源，通过管理界面增删改查，变更即时生效。

- 后端内核：`backend/pkg/scripting`（语言无关编排器 + 各语言适配器）
- 平台运行时：`backend/app/admin/service/internal/service`（`script_runtime.go` 多语言实例/数据库加载/任务桥，`script_entity_hooks.go` 实体钩子）
- 管理面：`系统管理 → 脚本管理`（三端齐备：React / vue-element / vue-vben）

## 定位与边界

脚本系统填补「配置化扩展」与「改代码二开」之间的中间层：

| 需求形态 | 用什么 |
|---|---|
| 行为参数不同 | 字典 / 策略表等配置化能力 |
| **逻辑需要"写代码"但不值得为此发版** | **本脚本系统** |
| 深度改流程 / 高频热路径 / UI 扩展 | Fork 主干二开 |

明确不支持：UI / 页面扩展、复杂工作流编排、高 QPS 请求热路径、租户自写脚本
（当前仅平台管理员可管理脚本）。

## 五类扩展点

### 1. 实体生命周期钩子（before / after）

平台在实体的写路径上暴露同步/异步钩子点，命名 `<entity>.before_<op>` / `<entity>.after_<op>`
（op = create / update / delete）。已登记实体：`user`、`tenant`、`role`、
`internal_message`、`notification_channel`（见 `backend/app/admin/service/internal/service/script_entity_hooks.go` 的 `EntityHooksMapping`，
新增实体登记一行即生效）。

| 类型 | 时机 | 语义 |
|---|---|---|
| `before` | 变更落库前，**同步** | 可否决：脚本 `return false` 或 `ctx.stop("原因")` 拒绝业务写入，业务侧收到 400 与否决原因；钩子自身异常 fail-open（不阻断业务） |
| `after` | 变更成功后，**异步** | 只读旁路：独立 goroutine + 30s 超时 + panic 兜底，失败只记日志不影响业务 |

**「该钩子点上没挂脚本」不是失败**：`EntityHooksMapping` 登记的实体 × 三类操作，远多于实际挂过脚本的
钩子点，每次映射实体变更都会问到若干空钩子点——两侧入口（`InvokeEntityHookVeto` / `InvokeEntityHook`）对
"无挂载"一律返回 nil，不打日志、不落执行记录。真出错的仍照报：ERROR 日志指名脚本名
（`script entity hook <点> failed: script '<名>' failed: …`）、`sys_script_logs` 落一条
`trigger_type=hook, success=false`，脚本本身加载失败另有 Resync 阶段按脚本名逐条的 ERROR。

上下文：`ctx.get("entity")`（实体类型）、`ctx.get("op")`、`ctx.get("id")`（实体 ID，update/delete 可得）。

### 2. 定时任务

脚本顶层调用 `task.register_handler(name, description, fn, opts)` 注册处理器（**当前仅 Lua 侧可实现**，
JS 的 `task` 模块为空占位，见「脚本编写约定」），再在「任务管理」新建一条调度记录即可周期执行：

- 任务记录：`type = PERIODIC`，`typeName = "script_task"`，`cron_spec` 任意合法 cron，
  `task_payload`：`{"handler": "<处理器名>", "params": {...}}`
- `opts`（可选）：`optional`（参数默认值表）、`required`（必填参数）、`timeout_secs`（默认 30）、
  `max_retries`（默认 2）、`priority`（默认 5，普通优先级）

执行链：asynq 调度 → 固定分发订阅（`script_task`）→ 按载荷 `handler` 字段分发到对应语言引擎 →
参数合并默认值、校验必填后以单一 params 表调用。脚本删除/禁用后，其处理器在下次
Resync 时按代际自动清理。

### 3. 事件订阅与发布

进程内事件总线（Redis pub/sub 用于跨实例 Resync 通知，见下文「运维」）：

- Lua：`local eventbus = require "kratos_eventbus"` → `subscribe / subscribe_async / subscribe_once / publish`
- JS：全局 `eventbus.subscribe(type, fn) / eventbus.publish(type, data)`

### 4. HTTP 出站（Webhook）

- Lua：`local http = require "kratos_http"`；JS：全局 `http`
- API：`http.get(url)`、`http.post(url, body, contentType?)`、
  `http.request(url, {method, headers, body})` → `{status, body, headers}`

安全护栏（fail-closed）：

- **域名白名单**：环境变量 `SCRIPT_HTTP_ALLOWED_DOMAINS`（逗号分隔，
  支持 `*.example.com` 通配一级子域）；未设置 = 全部出站拒绝
- **环回 / 云元数据地址硬禁**：仅拦截精确命中的 5 个字面量
  （`localhost`、`127.0.0.1`、`0.0.0.0`、`::1`、`169.254.169.254`，见
  `backend/pkg/scripting/api/module_http.go` 的 `checkAllowedURL`），且是 host 字符串比较而非 IP 段判断。
  **已知边界**：私有网段不在拦截范围内——白名单里写入 `127.0.0.2`、`10.x.x.x`、`192.168.x.x`、
  云厂商元数据备用地址（如阿里云 `100.100.100.200`）等仍会放行。因此白名单本身即安全边界，
  配置时只填确实需要的外网域名，不要填内网 IP。
- 重定向逐跳复检白名单（≤3 跳）；单请求超时 ≤30s；请求 / 响应体 ≤1MB
- 脚本不可覆盖 `Host` / `User-Agent` 头

### 5. 试运行（TestRun）

管理页行内「试运行」：已保存脚本按 id 在**一次性隔离引擎**中执行（不污染常驻
引擎的 VM 与注册），支持传入上下文初始数据（JSON 键值对），返回执行后的上下文快照。
也可直接运行未保存的草稿。

> **耗时看「执行日志」，不看试运行弹窗**：响应体 `TestRunScriptResponse.duration_ms` 有定义但
> 服务端从未赋值（`backend/app/admin/service/internal/service/script_service.go` 的 `TestRun` 只回填
> `Success` / `Error` / `Context`），三端弹窗读 `result.durationMs ?? 0` 因而恒显示 `0ms`。
> 同一次试运行会以 `trigger_type=test_run` 落一条执行日志
> （`backend/app/admin/service/internal/service/script_runtime.go` 的 `logExecution`），需要耗时就去「执行日志」查。
> 注意该列是 `time.Since(started).Milliseconds()` 的整数毫秒：跑不满 1ms 的脚本仍会记成 `0`
> （本机库 5 条 hook 日志实测全为 0，即此原因），要看量级请挑有循环或 I/O 的脚本试。

## 脚本编写约定

入口有两种形态（二选一，不要混用）：

**A. execute 形态** —— 脚本填写「挂载钩子点」字段时使用：
钩子触发时执行脚本的 `execute()`。

```lua
local log = require "kratos_logger"

function execute()
    local ctx = __get_ctx()                 -- 执行上下文
    local entity = ctx.get("entity")        -- 读取钩子数据
    ctx.set("result", "ok")                 -- 写回上下文（试运行可见）
    -- ctx.stop("拒绝原因")                  -- before 钩子中：否决业务写入
    return true                             -- false 表示失败/中止
end
```

**B. 自注册形态** —— 「挂载钩子点」留空时使用：脚本加载时顶层代码执行一次，
自行注册回调或任务处理器（改动脚本后由 Resync 重新加载）。

```lua
local hook = require "kratos_hook"
local task = require "task"

hook.register("user.after_create", "新用户欢迎", function(ctx)
    local id = ctx.get("id")
    -- ...
    return true
end)

task.register_handler("daily_cleanup", "每日清理", function(params)
    -- params 含任务载荷 params 与可选默认值合并后的结果
    return true
end, { optional = { older_than = 86400 }, timeout_secs = 60 })
```

JavaScript 与 Lua 的模块能力一致（`log` / `crypto` / `util` / `cache` / `eventbus` / `oss` /
`http` / `hook` + `__get_ctx / __set_ctx / __stop`），语法差异外 API 同名。

> **唯一例外：`task` 是 Lua 独有的。** JS 侧的 `task` 模块只是一个**空表占位、无实现**
> （`backend/pkg/scripting/runtime_javascript.go:100-104` 注册的是 `map[string]any{}`，注释即写明
> in-script 任务注册 API 尚未实现），其上没有任何 `register_handler` 可用。
> 任务处理器当前只能用 Lua 编写；asynq 任务桥与执行链路本身与语言无关，已通。

### 语言与沙箱

| 语言 | 引擎 | 沙箱 | 说明 |
|---|---|---|---|
| Lua（推荐） | gopher-lua | 标准库白名单：base / load / table / string / math / coroutine；禁用 os / io / debug | 默认语言 |
| JavaScript | goja | 无沙箱 | 仅限平台管理员使用 |

安全设计要点：所有脚本执行串行化（单 VM 正确性前提）；执行入口探活用哨兵法
（库限制：`GetGlobal` 读函数返回 nil）；业务依赖（Redis / OSS / EventBus）注入后
显式重放模块绑定。每次执行落 `sys_script_logs`（触发方式 / 版本 / 成败 / 耗时 / 错误），
管理页「执行日志」可查，支持清理 90 天前日志。

## 管理面

`系统管理 → 脚本管理`：

- 脚本 CRUD、启停（行内开关，即时生效）、按名称 / 语言 / 钩子点搜索
- 编辑抽屉：Monaco 源码编辑器（Lua / JS 高亮）、语言切换骨架（未编辑时自动换，
  已编辑则确认后切换）、钩子点自动补全
- 试运行：输入 JSON 键值 → 查看成功 / 失败、错误详情、上下文快照（耗时见「执行日志」，见上文 TestRun 注）
- 钩子点概览：当前已注册的全部钩子点与挂载数、引擎支持的语言
- 执行日志：触发方式 / 结果筛选、错误详情、清理 90 天前日志

## 运维要点

- **存储**：脚本唯一事实源是数据库 `sys_scripts` 表（version 字段为热更新指纹）；
  变更后本实例即时 Resync，其他实例经 Redis 频道 `gowind:script:resync` 收到通知自动重同步。
- **环境变量**：`SCRIPT_HTTP_ALLOWED_DOMAINS`（HTTP 出站白名单，未设置即全部拒绝）。
- **部署**：全新部署会自动建表；存量部署升级后需在「系统管理 → API 管理」触发一次
  同步，使 `/admin/v1/script*` 端点进入 Api 表（平台管理员不受影响，租户门禁依赖该表）。
- **多语言扩展**：语言由 go-scripts 适配器注册表动态发现（当前 lua / javascript）；
  新语言实现 RuntimeBinder 即可接入，无需改编排层。
