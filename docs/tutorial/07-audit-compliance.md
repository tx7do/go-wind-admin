# 07 · 审计与等保能力的使用面

> 前置：[01 章](./01-architecture-overview.md)。读完你会知道：平台内置了哪些审计/身份鉴别能力、它们各自记录什么、留存与归档怎么运转、以及管理页在哪配置——做到"会用、会查、知道边界"。

## 1. 六类审计日志

| 日志 | 记什么 | 采集层 |
|---|---|---|
| 登录审计 | 登录尝试：成败、IP、归属地、失败原因 | transport 中间件 |
| 操作审计 | 谁对哪个资源做了什么写操作（resource_type / action） | transport 中间件（从 kratos operation 名静态解析） |
| API 审计 | 每次 API 调用的 method/path/耗时/IP | transport 中间件 |
| 数据访问审计 | SQL 级：sql_text（**词法脱敏后**）、SHA256 指纹、影响行数、时延 | dialect.Driver + Tx 包装器（ent 执行层） |
| 权限变更审计 | 权限对象的 grant/revoke/assign 变更 | transport 中间件 |
| 策略评估日志 | 每次 authz 判定的策略匹配过程与 trace_id | authz 引擎包装器 |

设计要点（为什么分两种采集层）：HTTP 可得字段（IP、路径、操作者）在 transport 中间件就能拿到；
SQL 级字段只有数据库执行层知道，所以数据访问审计走 driver 包装器（照 ent 官方 DebugDriver
范式），在事务内也逐条捕获。防递归：审计行自身的 INSERT 带跳过标记，不会记进审计。
`sql_text` 的脱敏是 PostgreSQL 词法扫描器（纯 Go 实现）：字符串/数值字面量替换为 `***`、
结构保留，指纹基于脱敏文本——同构不同参的 SQL 指纹一致，便于分组统计。
全部实现细节与字段清单见 [audit-log-producer-design.md](../audit-log-producer-design.md)。

## 2. 留存与归档（等保 180 天要求）

- 库内留存默认 **180 天**（`AUDIT_RETENTION_DAYS` 可调）；
- asynq 每日定时归档：超期数据导出 **JSONL 归档文件**留痕，库内清理——留存与库瘦身两不误；
- 归档任务与备份（`scripts/backup/pg_backup.sh`，默认保留 30 份自动轮换）是两条独立链路，别混。

## 3. 身份鉴别三件套（管理页可调）

| 能力 | 说明 | 配置位置 |
|---|---|---|
| 口令复杂度 | ≥8 位、小写/大写/数字/符号四类取三 | 参数管理（平台参数，启动播种） |
| 历史口令复用 | 默认禁止复用近 3 条历史口令 | 参数管理 |
| 口令有效期 | 默认 90 天到期强制换新 | 参数管理 |
| TOTP MFA | 可选绑定，登录第二因子 | 个人中心 / 用户管理 |
| 登录限流 | IP + 用户名双维度 Redis 限流；可配置登录限制策略（时段/IP 段） | 登录策略管理页 |

三件套阈值经「参数管理」调整——**内置参数启动时播种**（空表守卫），已部署实例改参数立即生效，
不需要改代码或环境变量（这三项早先走环境变量，该路径已废弃：生产代码里 `os.Getenv` 只剩
`AUDIT_RETENTION_DAYS` / `AUDIT_ARCHIVE_DIR`（§2 的归档）与 JWT / 加密密钥几处，口令策略已不从环境读）。

## 4. 管理页在哪

六类审计日志各有独立查询页（三端齐备，当前形态为列表 + 筛选 + 导出）。

**把一次操作的日志串起来，靠的是 `request_id` 而不是 `trace_id`。** 三端的请求客户端给每个请求注入
`X-Request-ID`（如 `frontend/admin/react/src/core/transport/rest/request-client.ts:143`），API / 登录 /
操作 / 权限变更 / 数据访问五类都把它原样落库，同一个请求号即可跨表捞回这次点击产生的全部行。

> **`trace_id` 这一列基本是空的，别按它排障。** 本仓没有接链路追踪：中间件链里没有 kratos
> `tracing.Server()`（`app/admin/service/internal/server/rest_server.go` 的 `ms` 里查不到），
> `backend/docker-compose.yaml` 的 jaeger 段整段是注释状态（86 行起）。本机库实测（2026-09-25，
> 4.2 万行审计数据）：`sys_api_audit_logs` / `sys_login_audit_logs` / `sys_operation_audit_logs` /
> `sys_data_access_audit_logs` 四类的 `trace_id` **全为 NULL**（0 / 38,655 行）。
> 唯一有值的是 `sys_policy_evaluation_logs.trace_id`（2,887 / 3,193 行），但它也不是链路 ID——
> `traceIDFromHeader` 优先读 `traceparent`，读不到就退到 `X-Request-Id`
> （`app/admin/service/internal/server/policy_eval_logging_engine.go:136`），而三端发的正是
> `X-Request-ID`：这 2,887 个值里有 2,882 个能直接 join 上 `sys_api_audit_logs.request_id`。
> 也就是说它存的仍是那个请求号，只是列名不同。真要上链路，补 OTel 采集端 + `tracing.Server()`
> 即可，字段已预留。

参数管理 / 登录策略 / 在线用户（强制下线）在系统管理对应的管理页。

## 5. 边界与诚实声明

- 等保 2.0 除技术要求外还有管理制度、物理环境、人员组织等**非软件范畴**的内容——本平台
  覆盖的是技术措施部分，不能替代完整测评流程；
- `before_data`/`after_data`（操作审计的改前改后快照）与 `db_user`（数据访问审计）**留空待增强**——
  当前版本没有这几块，查不到不是配置问题。本机库实测（2026-09-25）：
  `sys_operation_audit_logs` 209 行的 before/after 全空，`sys_data_access_audit_logs` 34,369 行的
  `db_user` 全空（仓储层 `SetNillableDbUser` 有透传，但没有任何生产者给它赋值）。
  同表的 `table_name` / `data_category` **已经在采集**（从脱敏 SQL 里抽表名、首表映射数据分类，
  `backend/pkg/middleware/logging/data_access_audit_log.go:89-92`），实测两列 34,369 / 34,369 全有值——
  旧文档把这三列一起说成"待增强"是错的；
- authz 引擎默认 noop（第 05 章）——策略评估日志在 noop 下记录的是放行决策，切了引擎才有
  实质拦截记录。

## 深读

- [audit-log-producer-design.md](../audit-log-producer-design.md) —— 六类日志的采集层设计、字段索引、实施状态（唯一权威）
- [sys_config.md](../sys_config.md) —— 平台参数（口令策略三件套阈值的存放处）：缓存读取器、多实例失效广播、内置参数补种
- 根 [README.md](../../README.md)「安全与等保合规」—— 能力矩阵总表（对外口径）
- [backend_deploy.md](../backend_deploy.md) —— 生产部署时与审计相关的配置（留存天数、备份）

下一步：[08 · 脚本系统入门](./08-script-system.md)。
