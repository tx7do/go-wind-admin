# 任务调度系统（Task System）参考文档

> **定位**：本仓 asynq 任务调度体系的唯一权威说明——配置与启动链、任务数据模型、调度
> 生命周期、系统级常驻任务（含跨链路的到期扫描 / 审计归档 / 备份 / 站内信广播）、脚本任务桥、
> 管理页、多租户语义与运维排障。给任务系统加新类型、排"任务没跑"的问题、或接入新调度需求前先读它。
> 脚本任务桥的脚本侧语义（处理器注册/代际清理）见 [script_system.md](./script_system.md)；
> 到期扫描与备份的业务语义分别见 [plan_billing.md](./plan_billing.md)、下文第 5.3 节。

## 1. 组件地图

```
配置      cfg.Server.Asynq（bootstrap 配置，Redis 连接）——未配置则 NewAsynqServer 返回 nil，
          整个任务子系统（含系统级任务）不启动
启动链    NewAsynqServer（internal/server/asynq_server.go）
          ├─ 固定类型订阅注册（handler 路由表）：backup / tenant_expiry_scan /
          │  audit_log_archive / broadcast_message + script_task（桥，见第 6 节）
          ├─ RegisterTaskScheduler：把调度器句柄注入 TaskService（后续所有调度动作经它）
          ├─ RegisterTaskEnqueuer：站内信服务获得一次性任务入队能力（广播 fan-out）
          └─ StartAllTask（SystemViewer 上下文）：装载 sys_tasks + 重注册系统级 cron
数据      sys_tasks（任务表，带租户列）+ task_options（asynq 选项映射）
执行      asynq 调度器（Redis 队列，周期任务 cron entry + 一次性任务队列）
管理页    三端「系统管理 → 任务管理」（system/task）：任务 CRUD + 行内启停
```

## 2. 配置与启动链

- `configs/server.yaml` 的 `server.asynq.uri`（Redis 连接串）；缺失/未配置 →
  `NewAsynqServer` 直接 `return nil, nil`，任务子系统整体不可用，`hasScheduler()` 全链路
  nil 防护（ListTaskTypeName 返回空表、ControlTask 拒绝、StartAllTask 空转）。
- 订阅注册（启动期一次性，asynq 的 mux 拒绝 Start 之后注册 handler——这是脚本桥采用
  固定分发类型的根因，见第 6 节）：

| 固定类型常量 | handler | 形态 |
|---|---|---|
| `backup` | `TaskService.AsyncBackup` | 一次性（见 5.3） |
| `tenant_expiry_scan` | `TaskService.AsyncTenantExpiryScan` | 系统级 cron，每小时（见 5.1） |
| `audit_log_archive` | `TaskService.AsyncAuditLogArchive` | 系统级 cron，每日 03:30（见 5.2） |
| `broadcast_message` | `InternalMessageService.AsyncBroadcastMessage` | 一次性、幂等（见 5.4） |
| `script_task` | `ScriptRuntime.RunScriptTaskHandler`（经桥） | sys_tasks 型 PERIODIC，载荷带处理器名（见 6） |

## 3. 任务数据模型（`sys_tasks`）

| 字段 | 说明 |
|---|---|
| `type` | enum：PERIODIC / DELAY / WAIT_RESULT（默认 PERIODIC）——决定 startTask 走 NewPeriodicTask / NewTask / NewWaitResultTask 三条装载路径 |
| `type_name` | 任务执行类型名——**既是 asynq 的 handler 路由键又是调度项去重键**（全局命名空间，见第 8 节） |
| `cron_spec` | cron 表达式（仅 PERIODIC 有意义） |
| `task_payload` | JSON 字符串载荷，装载时 `json.Unmarshal` 到 handler 注册的载荷结构（nil 兜底为空对象，asynq 拒绝 nil message） |
| `task_options` | JSON，映射 asynq 选项：`max_retry`→MaxRetry、`timeout`/`deadline`/`process_in`/`process_at`→对应 asynq 同名项、`unique_ttl`→Unique、`retention`→Retention、`group`/`task_id`→Group/TaskID |
| `enable` | 启停位：false 的任务只落库不进调度器；Create/Update 后按当前值决定是否装载 |
| `tenant_id` / 审计列 | mixin 装配（任务表带租户维度，管理页查询按租户隔离） |

## 4. 调度生命周期（TaskService）

- **Create（H8 守卫）**：`type_name` 必须已在调度器注册（`TaskTypeExists`），否则 400
  "task type is not registered"——防止无 handler 的幽灵任务（每个 cron tick 报错）。
  落库后按 `enable` 决定是否装载；装载失败**不掩盖**（DB 已建但任务不会跑，显式报错）。
- **Update**：改 `cron_spec`/`payload`/`options` 时先 `stopTask` 再 `startTask`（旧 entry
  注销、新 entry 重建）；`type`/`type_name` 变更需走删除重建。
- **ControlTask**：按 `type_name` 定位任务，行内 Start / Stop / Restart；未知控制类型显式拒绝。
- **全局控制**：`StopAllTask` = `RemoveAllPeriodicTask`（**清掉包括系统级在内的全部周期项**）；
  `StartAllTask`/`RestartAllTask`（先 Remove 再 Start）= 重新装载 sys_tasks + **末尾重注册
  系统级 cron**——顺序保证重启路径下系统任务必然恢复（见 5 节）。
- **startTask**：PERIODIC → `NewPeriodicTask(cron, typeName, payload, opts...)`；
  DELAY → `NewTask`（一次性入队）；WAIT_RESULT → `NewWaitResultTask`。
- **ListTaskTypeName**：返回调度器当前已注册的处理器类型名（管理页新建对话框的类型下拉源）。

## 5. 系统级常驻任务

### 5.1 租户到期扫描（`tenant_expiry_scan`）

cron `0 * * * *`（`pkg/task/tenant_expiry.go`）。handler 语义完整见
[plan_billing.md](./plan_billing.md) 第 4 节（状态映射 + 令牌吊销 + READONLY 分工）。
不写 sys_tasks——规避 typeName 去重（第 8 节），注册点固定在 `startAllTask` 末尾。

### 5.2 审计日志归档（`audit_log_archive`）

cron `30 3 * * *`。把超过保留期的六类审计行导出 JSONL 归档文件后从库中删除
（库瘦身 + ≥6 个月留痕的等保平衡）。环境变量：`AUDIT_ARCHIVE_DIR`（默认
`./data/audit-archive`）、`AUDIT_RETENTION_DAYS`（默认 180）。采集与字段语义见
[audit-log-producer-design.md](./audit-log-producer-design.md)。

### 5.3 备份（`backup`，一次性）

`AsyncBackup`：SystemViewer 全量导出核心表 → JSON 序列化 → gzip → 上传 MinIO 桶
`backups`（对象名 `<日期>/<名称>-<时间>.json.gz`）。**这是应用级逻辑备份**，
与 `scripts/backup/pg_backup.sh`（pg_dump 物理备份，30 份轮换）互补、互不替代。
桶内对象的保留/清理策略当前无自动化（见第 10 节）。

### 5.4 站内信广播（`broadcast_message`，一次性 + 幂等）

`InternalMessageService.SendMessage` 落库消息后入队；handler 按 messageId 分页拉取收件人
批量插投递记录。断点恢复（进程重启后未完成投递自动重试）+ 幂等
（(message_id, recipient_user_id) 唯一约束 + CreateBulk `ON CONFLICT DO NOTHING`）。
载荷只带 messageId（消息本体留在库里，载荷不带大字段）。语义属站内信子系统。

## 6. 脚本任务桥（`script_task`）

asynq mux 的"Start 后不能注册 handler"约束 vs 脚本处理器运行期动态增删——故启动期注册
**一个固定分发类型**，处理器名放载荷 `handler` 字段，由 `ScriptRuntime.RunScriptTaskHandler`
按名分发。sys_tasks 行写法：`type=PERIODIC`、`type_name="script_task"`、`task_payload=
{"handler":"<名>","params":{...}}`。脚本删除/禁用后处理器在 Resync 时按代际清理。
完整 API 与安全模型见 [script_system.md](./script_system.md) 第 2 节。

## 7. 管理页（三端 `system/task`）

任务列表（启停位、类型、cron、载荷只读展示）+ 新建/编辑抽屉（类型下拉 = ListTaskTypeName
实时注册面、cron、payload JSON、选项）+ 行内启停/重启。全部经 `/admin/v1/tasks*` 端点
（MODULE_TASK），受权限面与租户闸门管控（新部署记得「接口同步」，见
[tenant_isolation.md](./tenant_isolation.md) §4.3）。

## 8. 多租户语义与调度器限制

- `sys_tasks` 带租户列：管理页的列表/详情查询按租户隔离（ent 隐私层，见 tenant_isolation 第 5 节）；
  `StartAllTask` 用 **SystemViewer** 跨租户装载全部任务（调度装载必须全量）。
- **typeName 全局命名空间限制**：调度器用 typeName 既做路由又做调度项去重键（`entryIDs[typeName]`
  单条目）。跨租户同名 PERIODIC 任务会互相覆盖产生"无法注销的孤儿 entry"——`startAllTask`
  按 typeName 去重、**只调度首个**并告警跳过。彻底隔离需调度器支持"路由类型/调度键分离"
  （库层改造，TODO）。实务约束：**任务 typeName 需全局唯一**（含租户维度前缀）。

## 9. 运维与排障

| 症状 | 核对顺序 |
|---|---|
| 任务没跑 | ① `enable` 位；② typeName 是否在注册面（ListTaskTypeName / 启动日志"系统级…已注册"）；③ H8 拒绝创建的报错（未注册类型）；④ 调度器是否配置（`server.asynq.uri`，未配置整个子系统静默缺失）；⑤ 同名任务被 startAllTask 去重跳过的告警日志 |
| StopAllTask 后系统任务还在 | 正常——StartAllTask 末尾自动重注册；要彻底停系统任务只能停服务或改代码 |
| 备份对象 | MinIO `backups` 桶，日期分层对象名；恢复 = 下载 JSON 反序列化（当前无自动恢复流程） |
| 归档目录/保留期 | `AUDIT_ARCHIVE_DIR` / `AUDIT_RETENTION_DAYS`（改后下个 03:30 周期生效） |
| 改了 cron 没生效 | Update 走 stop→start 重装载；确认后看管理页行内状态与调度器日志 |

## 10. 边界与已知问题

| 项 | 现状 |
|---|---|
| typeName 路由/去重键合一 | 库层限制，跨租户同名互斥（第 8 节），库改造 TODO |
| 备份恢复流程 | 仅导出上传，无自动恢复/演练工具链；桶内对象无生命周期清理 |
| WAIT_RESULT 型 | 枚举与装载路径在，无内置消费方示范；语义同 asynq wait-result |
| 系统级任务的可见性 | 不入 sys_tasks，管理页不可见、不可停——监控只能靠服务日志（"系统级…定时任务已注册"/"expiry scan:"等前缀） |
