# 审计日志采集层（Audit Log Producers）参考文档

> **定位**：六类审计日志采集机制的唯一权威说明——采集层架构、生产者字段、脱敏与指纹、
> 防递归、管道接线点位、管理页与归档、设计沿革与留空字段。
> 改审计采集、加新日志字段、排"审计没落库"前先读它。归档任务的调度语义见
> [task_system.md](./task_system.md) 第 5.2 节；等保口径见根 [README.md](../README.md) 安全章节。
>
> **状态**：设计定稿于 2026-08-28，三类型补全（操作 / 数据访问 / 权限变更）连同策略评估
> 包装均已实施并通过 e2e（各节内注日期）。本文档已由设计稿重组为参考形态；
> 方案比选过程保留于第 7 节设计沿革。

## 1. 机制总览

| 日志类型 | 采集层 | 落库管道 |
|---|---|---|
| ApiAuditLog（API 调用审计） | transport 中间件（`api_audit_log.go` Handle） | repo/service/wiring 全链（先于本文档存在） |
| LoginAuditLog（登录审计） | transport 中间件（`login_audit_log.go` Handle） | 同上 |
| OperationAuditLog（操作审计） | transport 中间件（`operation_audit_log.go`，2026-08-28 接通） | 同上（本文档补齐） |
| DataAccessAuditLog（数据访问审计） | dialect.Driver + Tx 包装器（`audit_driver_wrapper.go`，2026-08-28 接通） | 同上 |
| PermissionAuditLog（权限变更审计） | transport 中间件（`permission_audit_log.go`，2026-08-28 接通） | 同上 |
| PolicyEvaluationLog（策略评估日志） | authz 引擎包装器 `newEvalLoggingEngine`（每次 IsAuthorized 埋点 + trace_id + 评估上下文快照） | 同上 |

共同管道形态：采集层构造审计 DTO → 经注册回调（`Write*LogFunc`，见第 8 节）→
repo `Create` 全字段落库。全部采集在 **post-handler** 阶段执行（请求已完成，审计不影响业务时序）。

读写分离约束：六类的 admin `i_` proto 只定义 List/Get（GET），**Create 无 HTTP 绑定**——
审计写入只能由进程内组件（中间件/driver 包装器回调）触发，不经 HTTP 暴露；
audit 域 proto 有 Create RPC 但同样无 HTTP 绑定。

## 2. 采集层机制

### 2.1 transport 中间件类（api / login / operation / permission）

HTTP 可得字段（IP、UA、auth token、路径、成败、归属地等）由 `pkg/middleware/logging/utils.go`
的既有包级函数提取，四类中间件复用。

**operation 名解析**（operation / permission 两类的 `resource_type`/`target_type` 与 `action` 来源）：

- kratos operation 字符串全命名空间统一为 `/<package>.<ServiceName>/<Method>`——
  设计期核查全部 162 个 admin 命名空间 operation 均遵循此格式，无例外；
- `<ServiceName>` 去 "Service" 后缀转小写 → `resource_type` / `target_type`（`RoleService`→`role`）；
- `<Method>` → ActionType 映射：Create/BatchCreate→CREATE、Update→UPDATE、Delete→DELETE、
  Export→EXPORT、Import→IMPORT、Assign→ASSIGN、Unassign→UNASSIGN、其余→OTHER；
  **非写操作（Get/List 等）解析为 UNSPECIFIED 并跳过落库**（读操作不产生操作/权限审计）。

**permission 与 operation 的差异**：字段名与目标枚举不同——`target_type` 同源解析，
`action` 映射到 `PermissionAuditLog_ActionType`（数值：CREATE=5 / UPDATE=3 / DELETE=6 /
ASSIGN=7 / UNASSIGN=8，与 OperationAuditLog 数值不同、按名字映射由 converter 处理）；
schema 无 `geo_location`/`success` 字段，故不采集这两项。

命名陷阱（已消除）：`permission_audit_log_service.go` 的 repo 字段曾命名
`policyEvaluationLogRepo`，与 `policy_evaluation_log_service.go` 的同名字段撞名异型，
补产出方前先重命名为 `permissionAuditLogRepo` 与其实际类型一致——后续维护者勿改回。

### 2.2 driver 包装器类（data access）

数据访问审计的生产者字段（`sql_text`/`sql_digest`/`latency_ms`/`data_source`/`access_type`/
`affected_rows`）只在 SQL 执行层可得，HTTP 层不可见。架构分三层：

1. **driver/tx 包装器**（`internal/data/audit_driver_wrapper.go`）：`auditDriver`/`auditTx`
   实现 `dialect.Driver`/`dialect.Tx`，照官方 `DebugDriver`/`DebugTx` 范式**同时包 Driver
   与 Tx**（漏包 Tx 会漏事务内 SQL）。`Exec`/`Query` 前后采集
   {sql_text, sql_digest(SHA256), latency_ms, dialect, access_type(SQL 首词解析), affected_rows}
   append 进 context 累积器。注入点 `ent_client.go`：
   `ent.Driver(&auditDriver{drv})`。
2. **context 累积器**（`pkg/audit/event.go`）：`AuditEvent` 类型 +
   `AccumulatorKey`/`SinkKey` context key。中间件 pre-handler 植入 `*[]AuditEvent`，
   包装器向其 append，post-handler 取出。非 HTTP 路径（如迁移）无累积器，不采集。
3. **transport 中间件**（`data_access_audit_log.go`）：post-handler 取累积器逐条调
   `writeDataAccessAuditLogFunc`（即 repo.Create）落库。落库前植入 `SinkKey` 防递归标记。

`affected_rows == -1` 时跳过落库（uint32 溓出隐患）。

## 3. 生产者字段索引（proto 定义点）

| 审计类型 | 生产者字段 | proto 定义 |
|---|---|---|
| ApiAuditLog | `api_module` | `audit/service/v1/api_audit_log.proto:101` |
| OperationAuditLog | `resource_type` / `resource_id` / `action` / `before_data` / `after_data` | `audit/service/v1/operation_audit_log.proto:72,77,82,87,92` |
| DataAccessAuditLog | `data_source` / `table_name` / `data_id` / `access_type` / `sql_text` / `sql_digest` / `affected_rows` / `db_user` | `audit/service/v1/data_access_audit_log.proto:97,102,107,112,117,122,127,172` |
| PermissionAuditLog | `target_type` / `target_id` / `target_name` / `action` / `old_value` / `new_value` | `audit/service/v1/permission_audit_log.proto:72,77,82,88,94,99` |

字段语义边界见第 9 节（哪些字段当前留空）。

## 4. 脱敏与指纹（`pkg/audit/sqlmask.go`）

- PostgreSQL **词法扫描器**（纯 Go、零依赖——PG 的纯 Go AST 解析器均系 cgo，
  MySQL 方言解析器不认 `RETURNING`/`::`/双引号标识符；字面量脱敏本是词法层问题）；
- 字符串字面量（`'…'`/`E'…'`/`$tag$…$tag$`，含 `''`/`\'` 转义、嵌套块注释、`$n` 占位符、
  双引号标识符）与数值字面量替换为 `***`，**结构保留**（含占位符 `$1` 原样保留）；
- `sql_digest` 基于**脱敏后文本**计算 SHA256——同构不同参 SQL 指纹一致，便于分组统计；
- `data_masked=true`、`masking_rules="sql_literals:v1"` 随行落库；
- 验证：18 个边界单测 + 指纹稳定性测试（2026-08-28）；e2e 复验确认 `data_masked=t`、
  无审计表自身 INSERT 行混入。

## 5. 防递归（Sink 标记）

审计行自身的 INSERT 同样经过 driver 包装器——若不短路，落库动作会再生成审计事件，
无限递归。机制：**post-handler 审计阶段开始时**即向 context 植入 `SinkKey` 标记
（覆盖全部审计中间件，而非仅 data access 落库时刻），包装器 `collect` 见标记即跳过。
这同时杜绝其他审计表 INSERT 的行为噪音混入数据访问审计。

## 6. 管理页与归档

- **查询页**：六类各有独立页面（三端齐备，当前形态为列表 + 筛选 + CSV 导出，
  导出按筛选分页聚合、上限 1 万行）。
- **归档**：系统级 asynq 周期任务（每日 03:30，`audit_log_archive`）：超保留期行导出
  JSONL 归档文件后从库删除。调度/注册机制见 [task_system.md](./task_system.md) 第 5.2 节；
  参数：`AUDIT_ARCHIVE_DIR`（默认 `./data/audit-archive`）、`AUDIT_RETENTION_DAYS`
  （默认 180，库内留存天数）。
- 留存策略（库内 180 天 + 归档文件留痕）为等保 ≥6 个月要求的落地形态。

## 7. 设计沿革（方案比选记录，2026-08-28）

1. **resource_type / action 来源**：候选有路由标注（proto 注解）、ent hook、operation 名解析。
   选定 **transport 中间件 + `htr.Operation()` 静态解析**（§2.1）——路由层无静态来源
   （业务 proto 当时仅有 `google.api.http` 与 `redact.method_skip` 注解，无资源/动作标注）。
2. **ent hook 方案废弃**（原拟同时采集 before/after 快照）：三问题——
   ① 时序（hook 在 handler 执行期触发，早于 post-handler，需 context 桥接）；
   ② 递归（`client.Use` 注册到全部实体，审计表自身 Create 会触发 hook 形成递归）；
   ③ 噪音（一次业务请求级联多表 mutation——如 RoleService.Update 连带
   ResetPolicies 的 policy/permission 写入——全局 hook 会全部记成审计事件，淹没信号）。
   因 resource_type/action 已由 operation 名解析覆盖，hook 仅剩 before/after 一项收益，
   不足以抵消三问题，废弃。before/after 作为独立增强项待评估（§9）。
3. **data access 选 driver 包装器**：生产者字段只在 SQL 执行层可得，照官方
   DebugDriver/DebugTx 范式包 Driver+Tx 是唯一全覆盖（含事务）形态。
4. **已定决策**：resource_type/action 走 operation 名解析（非 hook/非标注/非 path 推断）；
   before/after 暂不实施；范围先试水 OperationAuditLog 验证模式，后铺开 4.2/4.3；
   动 PermissionAuditLog 前先消除命名陷阱。
   ——实施结果：4.2 / 4.3 与试水项均已落地并 e2e（§1 状态列）。

## 8. 管道接线点位（当前 wiring）

| 环节 | 位置 |
|---|---|
| 回调类型/字段/Option | `pkg/middleware/logging/options.go`（五类 `Write*LogFunc`） |
| 中间件实例化与 Handle 调用 | `pkg/middleware/logging/logging.go` |
| 回调闭包注册（闭包内调各 repo.Create） | `internal/server/rest_server.go` `NewRestMiddleware` 的 `applogging.Server(With*…)` 块 |
| 策略评估包装引擎 | `authz.Server(newEvalLoggingEngine(authorizer.Engine(), policyEvaluationLogRepo))`（同文件） |
| repo 实例化 | `cmd/server/wiring_ent.go`（手写 DI；五处 `data.New*AuditLogRepo`） |

> 历史注：本仓 DI 已弃用 Wire 改手写 `wiring_*.go`；本文档早稿中的 `wire_gen.go`/
> `wire_set.go` 行号引用即彼时装配点，现状以上表为准。

## 9. 边界与留空字段

| 字段 | 状态 |
|---|---|
| `before_data` / `after_data`（操作审计改前/后快照） | 留空待评估——需 ent hook 过滤策略（白名单/审计表短路），见 §7.2 |
| `target_id` / `old_value` / `new_value`（权限审计目标与值变化） | 留空待评估——需 handler 上下文来源 |
| `table_name` / `db_user`（数据访问审计表名/库用户） | 留空待评估——需 SQL/DSN 解析 |
| 非写操作的审计 | 设计即跳过（Get/List 等不产生操作/权限审计行） |
| 六类查询页的详情形态 | 当前为列表 + 筛选 + 导出，无详情抽屉 |
