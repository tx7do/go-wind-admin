# 任务调度系统（Task System）参考文档

> **定位**：本仓 asynq 任务调度体系的唯一权威说明——配置与启动链、任务数据模型、调度
> 生命周期、系统级常驻任务（含跨链路的到期扫描 / 审计归档 / 备份 / 站内信广播 / 通知异步派发）、
> 脚本任务桥、管理页、多租户语义与运维排障。给任务系统加新类型、排"任务没跑"的问题、或接入新调度需求前先读它。
> 脚本任务桥的脚本侧语义（处理器注册/代际清理）见 [script_system.md](./script_system.md)；
> 到期扫描与备份的业务语义分别见 [plan_billing.md](./plan_billing.md)、下文第 5.3 节。

## 1. 组件地图

```
配置      cfg.Server.Asynq（bootstrap 配置，Redis 连接）——未配置则 NewAsynqServer 返回 nil，
          整个任务子系统（含系统级任务）不启动
启动链    NewAsynqServer（internal/server/asynq_server.go）
          ├─ 固定类型订阅注册（handler 路由表）：backup / tenant_expiry_scan /
          │  audit_log_archive / broadcast_message / notification_dispatch + script_task（桥，见第 6 节）
          ├─ RegisterTaskScheduler：把调度器句柄注入 TaskService（后续所有调度动作经它）
          ├─ RegisterTaskEnqueuer：站内信服务（广播 fan-out）与通知域服务（异步派发）
          │  各自获得一次性任务入队能力；未注入时两处都退回同步路径
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
| `notification_dispatch` | `NotificationService.AsyncNotificationDispatch` | 一次性、幂等（见 5.5） |
| `script_task` | `ScriptRuntime.RunScriptTaskHandler`（经桥） | sys_tasks 型 PERIODIC，载荷带处理器名（见 6） |

订阅有两种形态：`RegisterSubscriber[T]` 的 handler 签名不带 ctx，`RegisterSubscriberWithCtx[T]`
带（`func(ctx, taskType, *T) error`）。**只有需要任务超时真正生效的那一类要用 WithCtx**：
`asynq.Timeout` 是挂在任务 ctx 上的 deadline，handler 拿不到 ctx 就等于把超时只用在"杀任务"上、
掐不断正在进行的 I/O（见 5.5 里的 SMTP 拨号）。两者的 handler 返回值都原样交给 asynq，
所以 `asynq.SkipRetry` / 归档语义在两种形态下都可用（用 `errors.Is` 判定）。

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

### 5.5 通知异步派发（`notification_dispatch`，一次性 + 台账幂等）

通知域里"调用方不需要投递结论"的那几个事件（找回密码 / 换绑验证码）在 `SendDirect` 落完台账行后
入队，SMTP 往返不再占住 HTTP；渠道测试邮件与站内信**故意保持同步**，判据与白名单见
[notification_domain_design.md](./notification_domain_design.md) §4 P2-3。要点（本任务是"幂等门不在
asynq 侧、在业务侧"的那个例外）：

- **载荷 `{delivery_id, target, title, content}`**：台账既不存正文、`target` 又是脱敏串，handler 无法从
  库里读回可投递的三元组，所以正文进载荷（代价与取舍记在通知域 §6 决策点 7）。这与 5.4 广播"载荷只带
  messageId"相反 —— 广播的本体在库里，验证码不在。
- **幂等门 = 台账状态**：handler 读回该行，`status != SENDING` 直接返回 nil（重投/归档后手工重跑都 no-op）。
  推论：**还有重试额度的失败必须留在 SENDING**，只写 `last_error` + `attempts`，否则第一次瞬态错误落了
  终态就把额度静默作废 —— 加新任务型时容易反过来写（写终态更"顺手"）。
- **选项写死在代码里**，不走 `sys_tasks.task_options`：`asynq.MaxRetry(3)`（⇒ 最多 4 次尝试，第 4 次才落
  终态 FAILED）+ `asynq.Timeout(30s)`（`pkg/mailer` 用 `DialContext`，配 `RegisterSubscriberWithCtx` 才真掐得断
  卡死的握手）。配置类错误（渠道没配/没启用/类型不对）与未注册渠道一律包 `asynq.SkipRetry`：
  重试不会让配置变好，只会把一个 OTP 载荷送进归档队列。
- **不入 `sys_tasks`**，与 5.1/5.2 同为"代码常驻"。但它确实进了 `ListTaskTypeName` 的注册面，
  所以管理页的类型下拉今天能选到它 —— 手工建一条这样的行只会在 handler 第一道守卫处报错
  （载荷缺 delivery_id），不会误发信；5.4 的 `broadcast_message` 同形。
- **运行期实测（2026-09-20，含"重试额度走完"的 4 次尝试轨迹）**：见通知域 §4 P2-3 的观测表。

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
| 通知台账一直 `SENDING` / 验证码邮件迟迟不来 | ① `server.asynq.uri` 是否配置（没配 ⇒ 通知整体退回同步投递，属预期不是故障）；② 任务有没有被同机的别的项目抢走（第 10 节"队列无命名空间"）：`asynq:{default}:retry` / `:archived` 里躺着它，而 `asynq:servers:*` 心跳里有多个进程 ⇒ 就是这个；③ 读台账的 `attempts` 与 `last_error`：`attempts=0` 且 `SKIPPED` = 入队前的配置预检就没过，是渠道配置问题，与队列无关 |

## 10. 边界与已知问题

| 项 | 现状 |
|---|---|
| typeName 路由/去重键合一 | 库层限制，跨租户同名互斥（第 8 节），库改造 TODO |
| 备份恢复流程 | 仅导出上传，无自动恢复/演练工具链；桶内对象无生命周期清理 |
| WAIT_RESULT 型 | 枚举与装载路径在，无内置消费方示范；语义同 asynq wait-result |
| 系统级任务的可见性 | 不入 sys_tasks，管理页不可见、不可停——监控只能靠服务日志（"系统级…定时任务已注册"/"expiry scan:"等前缀） |
| **asynq 队列没有命名空间** | 键形如 `asynq:{<queue>}:…`，**不带应用前缀** ⇒ "同一个 Redis DB + 同一个队列名"就是同一个队列。两个项目共库时互相抢任务，抢到的一方没有 handler 就 `handler not found` 退避重试直至归档（本机 DB 1 上实测读到过本仓 `tenant_expiry_scan` 躺在 `asynq:{default}:retry` 里，而同一 DB 里同时活着另一项目的 worker）。唯一的隔离手段是 `server.asynq.uri` 换 DB（或改 `queues` 名字），**部署时共库必须显式错开**；库层不提供"按消费者组区分"的能力 |
