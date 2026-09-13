# 08 · 脚本系统入门：不发版扩展平台行为

> 前置：[02 章](./02-get-it-running.md)环境可用。读完你会知道：什么需求该用脚本、五类扩展点各接什么、安全模型卡在哪、以及怎么在管理页里安全地试跑一个脚本。

## 1. 先对号入座：你的需求该用什么

| 需求形态 | 用什么 |
|---|---|
| 行为参数不同 | 字典 / 策略表等配置化能力（管理页配置） |
| **逻辑需要"写代码"但不值得为此发版** | **本脚本系统** |
| 深度改流程 / 高频热路径 / UI 扩展 | Fork 主干二开 |

明确不支持：UI/页面扩展、复杂工作流编排、高 QPS 请求热路径、租户自写脚本（仅平台管理员可管理）。

## 2. 五类扩展点

| 扩展点 | 时机 | 语义 |
|---|---|---|
| 实体生命周期钩子 `before_<op>` | 变更落库前，**同步** | 可否决：`return false` / `ctx.stop("原因")` 拒绝业务写入（业务侧收到 400）；钩子自身异常 fail-open |
| 实体生命周期钩子 `after_<op>` | 变更成功后，**异步** | 只读旁路：独立 goroutine + 30s 超时 + panic 兜底 |
| 定时任务 | asynq 调度 | 脚本注册 handler，任务管理页建 `script_task` 型 PERIODIC 记录（cron + 载荷 `{handler, params}`） |
| 事件订阅/发布 | 进程内事件总线 | `eventbus.subscribe/publish`（跨实例 Resync 走 Redis pub/sub，见运维节） |
| HTTP 出站（Webhook） | 脚本内主动外呼 | 域名白名单 fail-closed、回环/云元数据硬禁、≤3 跳重定向复检、30s 超时、1MB 体积上限 |

已登记钩子实体：`user`、`tenant`、`role`、`internal_message`、`notification_channel`
（登记表在 `internal/service/script_entity_hooks.go`，新增实体一行即生效）。

## 3. 脚本形态与最小示例

入口两种形态二选一：**execute 形态**（管理页填了「挂载钩子点」字段，钩子触发时执行 `execute()`）
与**自注册形态**（钩子点留空，脚本加载时顶层代码自行注册回调/任务处理器）。

```lua
-- 自注册形态：新用户创建后做点什么（after 钩子，只读旁路）
local hook = require "kratos_hook"

hook.register("user.after_create", "新用户欢迎", function(ctx)
    local id = ctx.get("id")
    -- ...只读处理，别指望改写业务结果
    return true
end)
```

```lua
-- execute 形态：before 钩子做风控否决
function execute()
    local ctx = __get_ctx()
    local entity = ctx.get("entity")
    -- 风控判断……不合规时：
    -- ctx.stop("命中风控规则")   -- 业务侧收到 400 与此原因
    return true
end
```

Lua 与 JS 模块能力一致（`log`/`crypto`/`util`/`cache`/`eventbus`/`oss`/`http`/`hook`/`task`），
语法差异外 API 同名。**推荐 Lua**（标准库白名单沙箱）；JS（goja）无沙箱，仅平台管理员使用。

## 4. 管理页工作流

「系统管理 → 脚本管理」：

1. 新建脚本（Monaco 编辑器、Lua/JS 高亮、钩子点自动补全）；
2. **试运行**：传 JSON 键值当上下文，一次性隔离引擎执行，看成功/失败、错误详情、上下文快照、耗时——
   **先试跑再启用**是习惯，不是可选项；
3. 启用（行内开关，即时生效）；
4. 「执行日志」看每次触发的方式/版本/成败/耗时/错误，支持清理 90 天前日志。

## 5. 运维要点

- **事实源是数据库** `sys_scripts`（`version` 字段为热更新指纹）：变更后本实例即时 Resync，
  其他实例经 Redis 频道 `gowind:script:resync` 自动重同步；
- **`SCRIPT_HTTP_ALLOWED_DOMAINS`**（环境变量，逗号分隔，支持 `*.example.com`）——
  **不设置 = 全部出站拒绝**，要用 Webhook 先配它；
- **存量部署升级后**：`/admin/v1/script*` 端点不在 Api 表里，须在「接口管理 → 接口同步」触发一次
  同步（平台管理员不受影响，租户门禁依赖该表——见第 06 章闸门）；
- 所有脚本执行**串行化**（单 VM 正确性前提），每次执行落 `sys_script_logs`。

## 深读

- [script_system.md](../script_system.md) —— 五类扩展点 API、安全模型、语言/沙箱差异、运维的唯一权威说明
- [task_system.md](../task_system.md) —— 脚本任务桥的宿主系统：调度生命周期、task_payload 装载语义、系统级任务注册面

下一步：[09 · 部署上线](./09-deployment.md)。
