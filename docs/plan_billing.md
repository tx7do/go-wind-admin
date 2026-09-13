# 套餐与计费管控（Plan / Billing）参考文档

> **定位**：本仓套餐体系（套餐目录 / 模块白名单 / 配额）与租户订阅到期处置的唯一权威说明——
> 数据模型、管理面、到期执行全链路、配额与用量计量、租户数据清理、运维清单。
> 套餐与租户闸门（HTTP 层）的联动细节见 [tenant_isolation.md](./tenant_isolation.md) 第 4 节；
> 任务调度系统（本体系到期扫描任务的宿主）见下文第 5 节的执行注册点。

## 1. 机制总览

```
配置面    sys_plans（套餐目录：version / expiry_policy / data_retention_days / 描述）
          sys_plan_modules（套餐 → 功能模块白名单）
          sys_plan_quotas（套餐 → 配额：USER_LIMIT / STORAGE / API_CALL）
                      │ 管理页维护（三端「租户管理 → 套餐管理 / 配额管理」）
                      ▼
绑定面    sys_tenants.plan 边（plan_id 外键）+ expired_at（到期时间）
                      │
                      ▼ 执行面（两条互斥链路）
          ┌─ 模块白名单 → 租户闸门第 3 段（(路由模板,method) → Api 表模块 → 套餐白名单，fail-closed）
          ├─ READONLY 到期 → 闸门第 2 段即时只读降级（GET/HEAD/OPTIONS 之外全拒）
          └─ BLOCK_LOGIN / FREEZE 到期 → AsyncTenantExpiryScan（每小时）→ 租户状态置
             EXPIRED / FREEZE + 吊销该租户全部用户双端令牌 → 登录与闸门的"非 ON 即拒"接管
```

| 面 | 内容 | 实现位置 |
|---|---|---|
| 配置 CRUD | 套餐 / 白名单 / 配额的增删改查 | `internal/service/plan_service.go`、`plan_module_service.go`、`plan_quota_service.go` + 对应 repo |
| 绑定 | 租户创建（含管理员用户）/ 编辑的套餐绑定与到期时间 | `internal/service/tenant_service.go`（`CreateTenantWithAdminUser` 等） |
| 白名单执行 | 见 tenant_isolation §4.2 第 3 段 | `internal/data/tenant_access_checker.go` |
| READONLY 执行 | 见 tenant_isolation §4.2 第 2 段 | 同上（中间件即时，不依赖扫描） |
| BLOCK_LOGIN/FREEZE 执行 | 状态映射 + 令牌吊销 | `internal/data/tenant_usage_repo.go` `EnforceExpiryPolicies` + `internal/service/task_service.go` `AsyncTenantExpiryScan` |
| 用量计量 | 用户数 / 存储字节 / API 调用次数聚合 | `tenant_usage_repo.go` `GetUsage`（ent 与 gorm 两套同构实现） |
| 租户数据清理 | 29 张带租户表事务硬删 | `tenant_usage_repo.go` `CleanupTenantData` |

## 2. 数据模型

### 2.1 `sys_plans`（套餐目录）

| 字段 | 类型 | 说明 |
|---|---|---|
| `name` | string，**唯一索引**（可空多 NULL） | 套餐名称；同名会让订阅与配额映射歧义，故唯一 |
| `version` | enum：FREE / STANDARD / ENTERPRISE（默认 FREE） | 套餐版本标识，展示用 |
| `expiry_policy` | enum：READONLY / BLOCK_LOGIN / FREEZE（默认 READONLY） | 到期处置策略，执行语义见第 5 节 |
| `data_retention_days` | uint32 | 数据保留周期（天）——**仅存储，当前无任何执行接线** |
| `description` | string | 描述 |

边：`tenants`（反边，tenant.plan_id 外键）、`quotas`（**级联删除**：删套餐连带删其配额行）、
`modules`（**级联删除**：删套餐连带删其白名单行）。`quotas`/`modules` 边无 `Required()`
（套餐先建、白名单与配额后补，必填会在 Create 时报 missing required edge）。

### 2.2 `sys_plan_modules`（模块白名单）

| 字段 | 说明 |
|---|---|
| `module` | enum，取值即 `identityV1.Module` 的十档：DASHBOARD / OPM / SYSTEM / DICT / TENANT / PERMISSION / LOG / INTERNAL_MESSAGE / FILE / TASK |
| `plan_id` | 外键 → sys_plans（级联删除） |

闸门消费方式：`planmodule.HasPlanWith(plan.IDEQ).ModuleEQ(...)` 计数 >0 即放行
（tenant_access_checker 第 3 段，见 tenant_isolation §4.2）。

### 2.3 `sys_plan_quotas`（配额）

| 字段 | 说明 |
|---|---|
| `quota_type` | enum：USER_LIMIT / STORAGE / API_CALL |
| `quota_value` | uint64 |
| `plan_id` | 外键（级联删除） |

**当前无硬执行点**：用户创建、文件上传、API 调用路径均不检查配额——配额是
配置面 + 计量面（第 7 节），硬限制（超限拒绝）未实现。

### 2.4 `sys_tenants` 侧的订阅字段

| 字段 | 说明 |
|---|---|
| `plan` 边（`plan_id` 外键） | **权威订阅关系**——闸门、到期扫描、用量计量全部经边预载（`WithPlan`）读取 |
| `expired_at` | 到期时间；扫描任务与闸门第 2 段的判定源 |
| `status` | enum：ON / OFF / EXPIRED / FREEZE；登录与闸门对非 ON 一律拒绝（见 tenant_isolation §4.2 第 1 段） |
| `subscription_plan`（string） | 遗留展示字段，仅随租户 CRUD 持久化，无执行语义；订阅关系以 `plan` 边为准 |

## 3. 管理面

- **套餐管理**（三端 `tenant/plan` 页）：套餐 CRUD；名称唯一；版本/到期策略/描述编辑。
  删除套餐级联清空其白名单与配额行，且已绑定租户的 plan 边悬空（`WithPlan` 预载为空 →
  闸门按"无套餐"拒绝全部业务模块）——删套餐前先解绑租户。
- **配额管理**（同页/子页）：按套餐维护三类配额值。
- **租户管理**：租户 CRUD、`CreateTenantWithAdminUser`（建租户 + 租户管理员一步完成）、
  套餐绑定与 `expired_at` 维护、到期状态展示。
- 全部端点在 MODULE_TENANT 模块（`ServiceTagToBusinessModule`：TenantService/PlanService/
  PlanQuotaService → TENANT）；新部署实例记得「接口同步」（tenant_isolation §4.3）。

## 4. 到期执行链路（三档全语义）

### 4.1 READONLY（默认）

`expired_at` 已过且策略 READONLY → **闸门即时**只读降级：仅 GET/HEAD/OPTIONS 放行，
其余 403（tenant_isolation §4.2 第 2 段）。不依赖扫描任务，状态保持 ON，
续期（改 `expired_at`）后即时恢复读写。

### 4.2 BLOCK_LOGIN / FREEZE（状态映射 + 扫描任务）

```
AsyncTenantExpiryScan（系统级周期任务）
  ├─ 注册：NewAsynqServer 启动时（internal/server asynq providers），
  │        cron = "0 * * * *"（每小时整点，pkg/task/tenant_expiry.go 常量），
  │        系统级常驻——不写入 sys_tasks 表、不经任务管理页
  ▼
EnforceExpiryPolicies（tenant_usage_repo，SystemViewerContext 跨租户）
  ├─ 圈定：status==ON 且 expired_at<=now，WithPlan 预载套餐
  ├─ 无套餐 → 跳过（保持 ON；但其业务模块本就被闸门"无套餐即拒"全拒）
  ├─ BLOCK_LOGIN → status := EXPIRED
  ├─ FREEZE      → status := FREEZE
  ├─ READONLY    → 保持 ON（交给闸门即时降级）
  └─ 状态已改 → 吊销该租户全部用户令牌（admin + app 双 ClientType，
               RevokeUserToken 按 uid 删缓存键，即时生效）
  ▼
此后登录（"非 ON 租户统一文案拒绝"）与闸门第 1 段（status!=ON → 403）接管
```

**生效时延**：READONLY 即时；BLOCK_LOGIN/FREEZE 最迟 1 小时（扫描周期）。
**恢复**：扫描改写的是 `status`——续费后须人工把租户 `status` 置回 ON 并延后 `expired_at`
（下一个扫描周期不会自动恢复，ON 是圈定条件）。

### 4.3 审计与日志

扫描与状态改写全程 info/warn 日志（`expiry scan:` 前缀，含租户 ID 与执行计数）；
租户状态变更经正常 repo 路径落操作审计（操作审计中间件按 operation 名解析，
resource_type=tenant）。

## 5. 任务宿主

到期扫描是 asynq 固定分发订阅（type=`TenantExpiryScanTaskType`）的系统级周期任务，
handler 为 `TaskService.AsyncTenantExpiryScan`。它**不在** sys_tasks 表（任务管理页不可见、
不可停），删除/停用只能改代码。任务调度体系的完整机制（启动链、装载、其他系统级任务、
排障）见 [task_system.md](./task_system.md)；租户自建任务体系（sys_tasks + 任务管理页）
是另一套，`script_task` 型见 [script_system.md](./script_system.md)。

## 6. 模块白名单联动（新模块上线 checklist）

新功能模块上线的完整链（缺一环 = 租户 403）：

1. 新服务的 OpenAPI tag 在 `pkg/constants/module_mapping.go` 登记（否则 Api 表
   `business_module=UNSPECIFIED`，所有租户被拒）；
2. 部署后在管理页跑「接口同步」（Api 表全量重建）；
3. **给需要放行的套餐补 `sys_plan_modules` 白名单行**（套餐管理页）——第 1/2 步只让
   模块"可归类"，第 3 步才让"某套餐的租户"真正可达；
4. 平台管理员验收 ≠ 租户验收（平台上下文不走闸门，见 tenant_isolation §4.1）。

## 7. 配额与用量计量

### 7.1 `GetUsage`（GET `/admin/v1/tenants/{id}/usage`）

`TenantUsageRepo.GetUsage`（SystemViewerContext，跨租户合法聚合通道）：

- 套餐与配额上限：`WithPlan(WithQuotas())` 预载（plan 名 + 三类 quota_value）；
- `UserCount`：`sys_users` 按租户 COUNT；
- `StorageUsedBytes`：`sys_files.size` 按租户 SUM（ent Aggregate）；
- `ApiCallCount`：`sys_api_audit_logs` 按租户 COUNT。

ent 与 gorm 各有一套同构实现（`internal/data/tenant_usage_repo.go`、
`internal/data/gorm/tenant_usage_repo.go`），随 ORM 切换走对应路径。
用途：租户详情页的用量/配额对照展示（计量），**不做超限拦截**。

### 7.2 硬限制（未实现）

配额不做执行：USER_LIMIT 不在用户创建处校验、STORAGE 不在上传处校验、
API_CALL 无调用计数拦截。接入硬限制的天然落点是各 service 的 Create/上传入口
+ `GetUsage` 的计量函数，接入时按 (租户×类型) 查配额并拒绝——设计待立项。

## 8. 租户数据清理（`CleanupTenantData`）

POST `/admin/v1/tenants/{id}/cleanup`：

- 单事务内**硬删**该租户在全部带 `tenant_id` 业务表的数据（实现内按表逐张 Delete，
  列表对齐 ent Client 全部具备 TenantIDEQ 谓词的包，共 29 张）；
- 保留 `sys_tenants` 行，`status` 置 OFF；
- 事务提交后吊销该租户全部用户双端令牌（用户 ID 列表在事务内先收集）。

**不可逆**：清理前确认（无软删、无备份联动——备份靠 pg_backup 外部兜底）。
SystemViewerContext 通道。清理动作走租户模块端点，受租户闸门/权限面管控。

## 9. 运维注意

| 事项 | 要点 |
|---|---|
| 到期排障 | 租户突然全 403/无法登录：查 `status`（EXPIRED/FREEZE 即扫描所置 → 查套餐 expiry_policy 与 expired_at）、查白名单（第 6 节链路）、查接口同步 |
| 续费恢复 | 人工：`expired_at` 延后 + `status` 置回 ON；READONLY 档只须延后 expired_at（状态本就 ON） |
| 删套餐 | 级联清白名单/配额 + 绑定租户悬空即全业务拒——先解绑 |
| 扫描任务监控 | `expiry scan:` 日志行（每小时计数）；任务不可从管理页停 |
| 用量异常 | GetUsage 的三个计量源（用户/文件/审计日志表）即对账入口 |
| 清理操作 | 第 8 节，执行前走备份快照 |

## 10. 边界与已知问题

| 项 | 现状 |
|---|---|
| 配额硬执行 | 未实现（第 7.2 节），仅配置+计量 |
| `data_retention_days` | 仅存储，无任何消费点（未接线） |
| `subscription_plan` 字符串字段 | 遗留展示位，执行语义以 `plan` 边为准（双表示待收敛） |
| 到期扫描的圈定条件 | 仅扫 status==ON；被人工置 OFF 的过期租户不会再被扫描改状态（行为上无差异——OFF 本就全拒） |
| FREEZE 与 EXPIRED 的差异 | 当前两档效果等价（登录拒+闸门拒+令牌吊销）；如需 FREEZE 保留数据只读、EXPIRED 阻断登录之类的差异化，需在闸门按 status 细分 |
