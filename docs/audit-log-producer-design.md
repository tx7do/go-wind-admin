# 审计日志「生产者」补全设计

> 状态：设计稿，待评审
> 日期：2026-08-28
> 范围：补齐三类审计日志（OperationAuditLog / DataAccessAuditLog / PermissionAuditLog）的产出方，使审计记录携带「生产者」语义字段。

## 1. 背景与问题定义

### 1.1 什么是「生产者」

审计日志中的「生产者」指**标识这条审计事件由哪个业务资源/数据源/权限对象产生**的结构化字段，用于事后追溯事件归属。不同审计类型对应不同字段：

| 审计类型 | 生产者字段 | proto 定义 |
|---|---|---|
| ApiAuditLog | `api_module` | `audit/service/v1/api_audit_log.proto:101` |
| OperationAuditLog | `resource_type` / `resource_id` / `action` / `before_data` / `after_data` | `audit/service/v1/operation_audit_log.proto:72,77,82,87,92` |
| DataAccessAuditLog | `data_source` / `table_name` / `data_id` / `access_type` / `sql_text` / `sql_digest` / `affected_rows` / `db_user` | `audit/service/v1/data_access_audit_log.proto:97,102,107,112,117,122,127,172` |
| PermissionAuditLog | `target_type` / `target_id` / `target_name` / `action` / `old_value` / `new_value` | `audit/service/v1/permission_audit_log.proto:72,77,82,88,94,99` |

### 1.2 现状

| 审计类型 | 落库管道(repo/service/wire) | 采集层 | 当前产出 |
|---|---|---|---|
| ApiAuditLog | ✅ 完整 | ✅ `api_audit_log.go` Handle | ✅ 有产出 |
| LoginAuditLog | ✅ 完整 | ✅ `login_audit_log.go` Handle | ✅ 有产出 |
| OperationAuditLog | ✅ 完整 | ✅ transport 中间件 | ✅ 有产出 |
| DataAccessAuditLog | ✅ 完整 | ✅ dialect.Driver+Tx 包装器 | ✅ 有产出 |
| PermissionAuditLog | ✅ 完整 | ✅ transport 中间件 | ✅ 有产出 |

**三类审计日志的落库管道全部就绪**——repo（`Create` 全字段落库）、service（`Create` 调 `repo.Create`）、wire 装配（`wire_gen.go:112/118/120` 实例化 repo+service）都在。缺的是**采集层**和**把采集层接到管道的回调注册**。三类审计表当前为空。

### 1.3 关键约束：采集层不在 HTTP 层

`api_audit_log.go` 的 Handle 之所以能用单一 transport 中间件采集，是因为其字段全部能从 HTTP transport 直接取得（method/path/body/IP/authToken）。三类的生产者字段**不在 HTTP 层**，不能照搬：

| 类型 | 生产者字段来源 | 采集层应落在 |
|---|---|---|
| OperationAuditLog | 业务语义（哪个资源、改前改后快照） | 业务层：路由标注 + handler before/after |
| DataAccessAuditLog | 数据层执行（SQL/表名/影响行数） | 数据层：ent hooks 或 DB driver interceptor |
| PermissionAuditLog | 权限系统变更点（grant/revoke） | 权限服务内部显式 emit |

只补管道接线不补采集层，产出的日志生产者字段仍为空——等于仍「没有生产者」。

---

## 2. 已核实事实（证据索引）

### 2.1 落库管道就绪

- **repo**：`operation_audit_log_repo.go:146-178`、`data_access_audit_log_repo.go:146-182`、`permission_audit_log_repo.go:140-167` 的 `Create` 均为完整落库实现，逐一 `SetNillable...` 全字段。
- **service**：`operation_audit_log_service.go:46`、`data_access_audit_log_service.go:46`、`permission_audit_log_service.go:58` 的 `Create` 封调 `repo.Create`。
- **wire 装配**：`wire_gen.go:112/118/120` 实例化三 repo，`:113/119/121` 实例化三 service；repo provider 声明于 `data/providers/wire_set.go:87-89`。

### 2.2 admin 域 proto 刻意只读

三类 admin `i_` proto 只定义 `List`/`Get`（GET），无 `Create` HTTP 绑定：
- `i_operation_audit_log.proto:12-26`
- `i_data_access_audit_log.proto:12-26`
- `i_permission_audit_log.proto:11-25`

audit 域 proto 有 `Create` RPC 但无 HTTP 绑定，仅供进程内调用落库。这是刻意的读写分离：Create 不经 HTTP 暴露，只能由内部组件（中间件回调）触发。

### 2.3 管道接线落点（当前缺这三类）

`pkg/middleware/logging/` 当前只有 api/login 两类：

- **`options.go`**：`:10` `WriteApiLogFunc` 类型、`:11` `WriteLoginLogFunc` 类型；`:14`/`:15` `options` 字段；`:26-30`/`:32-36` `With*` Option。三类对应的类型/字段/Option 不存在。
- **`logging.go`** `Server()`：`:33`/`:34` 实例化两个中间件；`:48`/`:49` 调用两个 `Handle`。三类对应的实例化/调用不存在。
- **`rest_server.go`** `NewRestMiddleware`（`:36-43` 形参，`:50-59` 注册 `applogging.Server` 块）：当前签名只接收 `apiAuditLogRepo`/`loginAuditLogRepo`，只注册两个回调闭包。`wire_gen.go:56` 调用时传 6 实参。

> 注：`operationAuditLogRepo` 已由 `wire_gen.go:118` 构造（provider `wire_set.go:88`），当前只传给只读 `NewOperationAuditLogService`，未传入 `NewRestMiddleware`。引入 sink 不需新增 provider，仅需接线把已有 repo 接进。

### 2.4 采集层现状（三类各自）

- **路由标注（OperationAuditLog 需要）**：业务模块 proto（`i_user.proto`/`i_role.proto` 等）RPC 方法上只有 `google.api.http` 和 `redact.method_skip`，**无任何「资源类型/动作」标注**。`resource_type`/`action` 在路由层无静态来源。
- **before/after 快照（OperationAuditLog 需要）**：各 service 的 `Update` 方法行为不一致。`role_service.go:206` 取了 before 对象（兼用于业务校验），`user_service.go:445` 取的是 operator 自身而非目标对象 before。无统一 before-snapshot 机制。
- **ent hook（DataAccessAuditLog 需要）**：`Client.Use`（`ent/client.go:425-441`）签名已生成暴露，fan-out 到全部 40 实体客户端。**应用层当前注册的 hook 数为零。**
- **权限变更点（PermissionAuditLog 需要）**：权限服务的 grant/revoke/assign 变更点当前无任何 emit 调用。

### 2.5 命名坑（PermissionAuditLog 专属）

`PermissionAuditLogService` 与 `PolicyEvaluationLogService` 是两个完全独立的服务/表/proto，互不共享，仅字段名撞名导致混淆：

| | PermissionAuditLogService | PolicyEvaluationLogService |
|---|---|---|
| 文件 | `service/permission_audit_log_service.go` | `service/policy_evaluation_log_service.go` |
| proto 域 | `audit.service.v1.PermissionAuditLog` | `permission.service.v1.PolicyEvaluationLog` |
| ent 表 | `permission_audit_log` | `policy_evaluation_log` |
| repo 类型 | `*data.PermissionAuditLogRepo` | `*data.PolicyEvaluationLogRepo` |
| wire | `wire_gen.go:112-113` / `wire_set.go:87` | `wire_gen.go:114-115` / `wire_set.go:84` |

混淆源：`permission_audit_log_service.go:22` 把 `*data.PermissionAuditLogRepo` 字段命名为 `policyEvaluationLogRepo`，与 `policy_evaluation_log_service.go:22` 的同名字段（类型 `*data.PolicyEvaluationLogRepo`）撞名但类型不同。补 PermissionAuditLog 产出方前须先重命名该字段消除歧义。

---

## 3. 管道接线方案（三类通用，b1-b4）

三类共用同一套接线模式，照搬 api/login 现有结构：

**b1 — `options.go` 扩展**：为每类新增 `WriteXxxLogFunc` 类型、`options` 字段、`WithWriteXxxLogFunc` Option。落点：`options.go:10-11`(类型)/`:13-22`(字段)/`:26-36`(Option) 之后追加。

**b2 — `logging.go` 实例化与调用**：`Server()` 内 `NewXxxMiddleware(&op)` 实例化（落点 `:33/:34` 之后）+ handler 内 `xxxMiddleware.Handle(...)` 调用（落点 `:48/:49` 之后）。

**b3 — `rest_server.go` 注册闭包**：`NewRestMiddleware` 形参增加 repo（落点 `:36-43`），`applogging.Server` 块增加 `With*` Option（落点 `:50-59`），闭包内调 `xxxRepo.Create(ctx, &auditV1.CreateXxxRequest{Data: data})`。

**b4 — wire 接线**：`wire_gen.go:56` 调用 `NewRestMiddleware` 增加实参（repo 已由 `:118/120/112` 构造，不需新增 provider），重新 `wire gen`。

> 管道接线是机械工作，三类一致。真正的设计差异在采集层（第 4 节）。

---

## 4. 采集层方案（三类各异）

### 4.1 OperationAuditLog（试水类，已实施）

**目标**：记录「谁（user_id/tenant_id）对哪个资源（resource_type）做了什么动作（action）」。

#### 4.1.1 选定方案：transport 中间件 + operation 名解析

实施中确认：`resource_type`/`action` 无需 ent hook，可由 transport 中间件从 `htr.Operation()` 静态解析。kratos operation 字符串格式统一为 `/<package>.<ServiceName>/<Method>`（如 `/admin.service.v1.RoleService/Update`），已核实全 admin 命名空间（162 个 operation）均遵循此格式，无例外。解析逻辑：
- `<ServiceName>` 去除 "Service" 后缀并转小写 → `resource_type`（`RoleService`→`role`）
- `<Method>` 按下表映射 → `ActionType`：Create/BatchCreate→CREATE，Update→UPDATE，Delete→DELETE，Export→EXPORT，Import→IMPORT，Assign→ASSIGN，Unassign→UNASSIGN，其余→OTHER；非写操作（Get/List 等）返回 UNSPECIFIED 并跳过落库

此方案无 ent hook 的全局噪音/递归/时序问题，与 `api_audit_log.go` 的 post-handler 采集模式完全一致。HTTP 可得字段（ip/request_id/tenant_id/user_id/username/success/failure_reason/geo_location）复用 `utils.go` 既有包级函数。

#### 4.1.2 before_data / after_data（暂不实施）

before/after 快照需在数据 mutation 层采集，ent hook 是唯一通用来源（`m.Fields()`/`m.Field()`/`m.OldField()`，见 vendored `ent.go:270-349`）。但 ent hook 全局捕获所有实体 mutation，存在噪音/递归/时序问题（见 4.1.3）。当前版本 `before_data`/`after_data` 留空，作为后续增强——待评估 ent hook 过滤策略（实体白名单/黑名单、审计表自身短路防递归）后单独实施。

#### 4.1.3 ent hook 方案废弃原因（记录）

原设计 4.1.2 方案 B 拟用 ent hook 同时采集 resource_type/action/before/after。实施前复核发现三个问题：
1. **时序**：hook 在 handler 执行期间触发，早于 transport 中间件 post-handler，需经 context 桥接审计对象。
2. **递归**：`client.Use` 注册到全部 40 实体，审计日志表自身 Create 会触发 hook 形成递归，需按 `m.Type()` 短路。
3. **噪音**：一次业务请求级联触发多张表 mutation（如 RoleService.Update 连带 ResetPolicies 的 policy/permission 表写入），全局 hook 把它们全部记成审计事件，淹没信号。

由于 `resource_type`/`action` 已由 operation 名解析覆盖（4.1.1），ent hook 仅剩 before/after 一项收益，不足以抵消三问题，故废弃。before/after 作为独立增强项单独评估。

### 4.2 DataAccessAuditLog（已实施）

与前两类不同，DataAccessAuditLog 的生产者字段（`sql_text`/`sql_digest`/`latency_ms`/`data_source`/`access_type`/`affected_rows`）在 SQL 执行层，不在 HTTP 层。采集层落在 **dialect.Driver + Tx 包装器**，照官方 `DebugDriver`/`DebugTx`（`dialect/dialect.go:67-208`）范式实现。

#### 4.2.1 选定方案：dialect.Driver 包装器 + context 累积 + transport 中间件落库

架构分三层：
1. **driver wrapper**（`internal/data/audit_driver_wrapper.go`）：`auditDriver`/`auditTx` 实现 `dialect.Driver`/`dialect.Tx`，在 `Exec`/`Query` 前后采集 `{sql_text, sql_digest(SHA256), latency_ms, dialect, access_type(首词解析), affected_rows}`，append 进 ctx 中的 accumulator。照 `DebugDriver` 范式**同时包 Driver 和 Tx**——否则漏事务内 SQL（`ent/tx.go:324-331`）。在 `ent_client.go:35` 注入：`ent.Driver(&auditDriver{drv})`。
2. **context accumulator**（`pkg/audit/event.go`）：`AuditEvent` 类型 + `AccumulatorKey`/`SinkKey` context key。transport 中间件 pre-handler 植入 `*[]AuditEvent` 进 ctx，driver wrapper 向其 append，post-handler 取出。非 HTTP 路径（迁移）无 accumulator，不采集。
3. **transport 中间件**（`pkg/middleware/logging/data_access_audit_log.go`）：post-handler 从 ctx 取 accumulator，逐条调 `writeDataAccessAuditLogFunc`（即 `repo.Create`）落库。落库前植入 `SinkKey` 防递归标记——审计行自身的 INSERT 经 driver wrapper 时，`collect` 见到标记跳过。

`table_name`/`db_user` 需额外 SQL/DSN 解析，初版留空作后续增强。

#### 4.2.2 实施状态（已完成）

1. ✅ `pkg/audit/event.go`：公共类型与 context key（跨 internal 边界）
2. ✅ `internal/data/audit_driver_wrapper.go`：driver+tx 包装器，照 DebugDriver/DebugTx 范式
3. ✅ `ent_client.go:35`：注入 wrapper
4. ✅ 管道接线 b1-b4（options.go/logging.go/rest_server.go/wire）
5. ✅ `data_access_audit_log.go`：transport 中间件，accumulator 落库 + 防递归
6. ✅ 编译验证通过
7. ✅ e2e 验证（2026-08-28）：GET `/admin/v1/users` + `/admin/v1/roles` → `sys_data_access_audit_logs` 落库 32 条，`data_source`=`postgres`、`access_type`=`SELECT`/`INSERT`、`latency_ms`=0-50、`sql_digest`=SHA256 hex 全部正确。一次请求触发的多条 SQL（含 ent 内部关联查询）被逐条捕获，验证了单条 SQL 粒度覆盖面。表此前 0 行。
8. ✅ **sql_text 脱敏**（2026-08-28）：`pkg/audit/sqlmask.go` 实现 PostgreSQL 词法扫描器（纯 Go、零依赖——PG 的纯 Go AST 解析器均系 cgo，MySQL 方言解析器不认 `RETURNING`/`::`/双引号标识符，字面量脱敏本是词法层问题）。字符串字面量（`'…'`/`E'…'`/`$tag$…$tag$`，`''`/`\'` 转义、嵌套块注释、`$n` 占位符、双引号标识符均正确处理）与数值字面量替换为 `***`，结构保留。`sql_digest` 基于脱敏后文本计算（同构不同参 SQL 指纹一致，便于分组）；`data_masked`=true、`masking_rules`=`sql_literals:v1` 随行落库。18 个边界单测 + 指纹稳定性测试通过。附带修复：sink 防递归标记从 DataAccessAuditLog 落库时提前到整个 post-handler 审计阶段（`logging.go`），杜绝其他审计表 INSERT 进入 data_access 审计的噪音；`affected_rows`=-1 时不再落库（uint32 溢出隐患）。e2e 复验：`data_masked=t`、`sql_text` 保留 `$1` 占位符、无审计表自身 INSERT 行。
9. ⏳ `table_name`/`db_user` 增强：需 SQL 解析/DSN 解析，留空待评估

### 4.3 PermissionAuditLog（已实施）

与 OperationAuditLog 同构，采用 transport 中间件 + operation 名解析。差异仅在字段名与枚举映射：
- `target_type` 从 operation 的 ServiceName 解析（同 resource_type）
- `action` 映射到 `PermissionAuditLog_ActionType`（CREATE=5/UPDATE=3/DELETE=6/ASSIGN=7/UNASSIGN=8，与 OperationAuditLog 数值不同但按名字映射 converter 可处理）
- HTTP 可得字段：`operator_id`/`operator_name`（token）、`tenant_id`、`ip_address`、`request_id`
- schema 无 `geo_location`/`success` 字段（与 OperationAuditLog 不同），故不采集这俩
- `target_id`/`old_value`/`new_value` 留空（后续增强，需 handler 上下文）

命名坑已消除：`permission_audit_log_service.go` 的字段 `policyEvaluationLogRepo` 重命名为 `permissionAuditLogRepo`，与其实际类型 `*data.PermissionAuditLogRepo` 一致。

#### 实施状态（已完成）

1. ✅ 命名坑消除
2. ✅ 管道接线 b1-b4（options.go/logging.go/rest_server.go/wire）
3. ✅ 采集层 `permission_audit_log.go`
4. ✅ 编译验证通过
5. ✅ e2e 验证（2026-08-28）：POST/PUT/DELETE `/admin/v1/permissions` 均触发落库，`sys_permission_audit_logs` 记录 `target_type`=`permission`、`action`=`CREATE`/`UPDATE`/`DELETE`、`operator_id`=`1`、`ip_address`=`127.0.0.1`，全部正确。表此前 0 行。
6. ⏳ `target_id`/`old_value`/`new_value` 增强：待评估

---

## 5. OperationAuditLog 实施状态（已完成）

1. ✅ **管道接线 b1-b4**：`options.go` 加 `WriteOperationAuditLogFunc`（类型/字段/Option）；`logging.go` 实例化 `OperationAuditLogMiddleware` 并调 `Handle`；`rest_server.go` 形参加 `operationAuditLogRepo`、注册闭包调 `repo.Create`；wire 重新生成（`operationAuditLogRepo` 现同时注入 `NewRestMiddleware` 和 `NewOperationAuditLogService`）。
2. ✅ **采集层**：新建 `operation_audit_log.go`，post-handler 从 `htr.Operation()` 解析 `resource_type`/`action` + HTTP 可得字段，调 `writeOperationAuditLogFunc` 落库。
3. ✅ **编译验证**：`go build ./app/admin/service/...` 通过。
4. ✅ **e2e 验证（2026-08-28）**：登录后发起 `DELETE /admin/v1/roles/999`，`sys_operation_audit_logs` 表落库一条记录，`resource_type`=`role`、`action`=`DELETE`、`user_id`=`1`、`ip_address`=`127.0.0.1`、`success`=`false`（均正确采集）。生产者字段（resource_type）此前恒为 NULL，现已正确填充并持久化。验证后 `CaptchaEnabled` 已恢复 `true`，临时 DB 密码重置未还原（admin 密码仍为 `admin`，与文档一致）。
5. ⏳ **before/after 增强**：待评估 ent hook 过滤策略后单独实施。
6. ⏳ **推广评估**：验证模式后评估 4.2/4.3 可复用性。

---

## 6. 已定决策

1. **resource_type/action 来源**：transport 中间件从 `htr.Operation()` 静态解析（4.1.1），非 ent hook、非路由标注、非 path 推断。
2. **before/after 来源**：暂不实施（4.1.2），ent hook 方案因噪音/递归/时序废弃（4.1.3），作为独立增强项待评估。
3. **范围**：先试水 OperationAuditLog，验证模式后再铺开。
4. **命名坑**：动 PermissionAuditLog 前先重命名 `policyEvaluationLogRepo` 字段。
