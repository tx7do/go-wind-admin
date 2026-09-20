# 多租户隔离机制（Tenant Isolation）参考文档

> **定位**：本仓租户隔离机制的唯一权威说明——上下文链路、HTTP 层闸门、数据层读写隔离、
> 套餐联动、覆盖边界、接入步骤与排障路径。改隔离层、Api 表、套餐门禁或给新表接租户前先读它。
> 三轴关系（接口级 authz / 租户隔离 / 数据范围）见 [data_scope_design.md](./data_scope_design.md) 第 1 节；
> 行级第二维（组织维数据范围）的完整语义也在该文档。

## 1. 机制总览

```
请求 → [认证中间件]  JWT → UserTokenPayload
                        │
                        ├─ auth.NewContext（operator，业务层读）
                        ├─ viewer.WithContext(UserViewer)   ← 隔离层的判定依据
                        └─ metadata（OperatorMetadata）
                [租户闸门]   tid>0 时：租户状态 → 到期只读 → 套餐模块白名单   ← 全 fail-closed
                [authz 引擎] 接口级授权（见 frontend_authority/教程05）
                        │
                        ▼
                service → repo → ent 隐私层（privacy.NewPolicies 合成的策略链）
                                    ├─ TenantPrivacy（库层，mixin 编译进全部带租户表）
                                    └─ TenantMutationGuardPolicy（仓内，全部带租户表冗余层）
```

| 层 | 拦什么 | 实现位置 |
|---|---|---|
| HTTP 闸门 | 租户身份下的**接口可达性**（状态/到期/套餐白名单） | `pkg/middleware/auth/auth.go`（调用方）+ `internal/data/tenant_access_checker.go`（实现） |
| ent 读隔离 | 查询行的 **tenant_id 归属** | go-crud `rule.TenantPrivacy.EvalQuery`（v0.0.55，经 `mixin.TenantID` 编译） |
| ent 写隔离 | 变更行（Update/UpdateOne/Delete/DeleteOne）的 tenant_id 归属 + tenant_id 改值防护 | 库层 `TenantPrivacy.EvalMutation`（v0.0.55 起）+ 仓内 `schema/tenant_mutation_guard.go`（全部带租户表的冗余层） |
| Create 防伪造 | 租户上下文 Create 强制覆盖 tenant_id 为 viewer.tid | 库层 `TenantPrivacy.EvalMutation` Create 分支 |

**与数据范围的组合**：试点表 `sys_positions` 的 Policy 返回 `TenantAndDataScopePolicy`
（`schema/data_scope_guard.go`，链式：租户变更防护 + 数据范围查询过滤），其余 32 张带租户表
返回裸 `TenantMutationGuardPolicy`（含 2026-09-12 补挂的 `sys_access_keys`，见第 6 节修复记录）——
全部 33 张带租户表均为库层+仓内双防线。

## 2. 租户模型与表分类

### 2.1 数据模型

| 表 | 内容 |
|---|---|
| `sys_tenants` | 租户行：`status`（ON/OFF/EXPIRED/FREEZE，仅 ON 可用）、`type`、`audit_status`、`expired_at`、套餐关联 |
| `sys_plans` | 套餐：`version`、`expiry_policy`（READONLY / BLOCK_LOGIN / FREEZE，默认 READONLY） |
| `sys_plan_modules` | 套餐 → 业务模块白名单（闸门第 3 段的判定源） |
| `sys_plan_quotas` | 套餐配额（另行核查，不参与闸门） |

平台级表（无 tenant_id，跨租户可见）：`sys_tenants`/`sys_plans` 本身、`sys_apis`、菜单、权限点/权限组、
语言等。租户表与平台表的划分**以 schema 是否装配 `mixin.TenantID[uint32]{}` 为准**，不靠命名约定。

### 2.2 带租户列的表（当前 33 张，全量盘点）

| 模块 | 表 |
|---|---|
| dict | `sys_dict_types`、`sys_dict_entries`、`sys_dict_entry_i18n` |
| file | `files`（同样没有 `sys_` 前缀） |
| internal_message | `internal_messages`、`internal_message_categories`、`internal_message_recipients`（**没有 `sys_` 前缀**，2026-09-20 按 `information_schema.tables` 核对；写探针 SQL 时别照 ent schema 名加前缀） |
| log（审计） | `sys_api_audit_logs`、`sys_data_access_audit_logs`、`sys_login_audit_logs`、`sys_operation_audit_logs`、`sys_permission_audit_logs`、`sys_policy_evaluation_logs` |
| membership | `sys_memberships`、`sys_membership_org_units`、`sys_membership_positions`、`sys_membership_roles` |
| opm | `sys_org_units`、`sys_positions`（数据范围试点，挂组合策略）、`sys_users`、`sys_user_credentials`、`sys_user_mfa_factors`、`sys_user_org_units`、`sys_user_positions`、`sys_user_roles`、`sys_roles`、`sys_role_metadata`、`sys_role_org_units`、`sys_role_permissions`、`sys_role_field_permissions` |
| system/tenant | `sys_login_policies`、`sys_tasks`、`sys_access_keys` |

> 审计日志表带 tenant_id 意味着：租户管理员只能看到本租户的审计记录；平台管理员全量。

### 2.3 mixin 与编译机制

`mixin.TenantID`（go-crud `entgo/mixin/tenant_id.go`）做两件事：

1. `Fields()` 注入 `tenant_id` 列：`Uint32`、**`Immutable()`**（ent 层禁止后续 Set 改值）、`Default(0)`、`Nillable`；
2. `Policy()` 返回 `rule.TenantPrivacy`——ent 的代码生成把 **mixin 策略与 schema 自身 Policy()
   一起**合成进 `internal/data/ent/runtime/runtime.go` 的 `privacy.NewPolicies(<mixin>, schema.X{})`
   链（`ent.Schema` 嵌入结构体提供返回 nil 的默认 `Policy()`，nil 被跳过；schema 覆写则追加进链）。

即：**挂 mixin 即获得库层隔离，无需在 schema 写任何 Policy**；schema 的 `Policy()` 返回
`TenantMutationGuardPolicy` 只是在此之上**追加**仓内第二道（见 4.3）。

## 3. 请求侧上下文链路

`pkg/middleware/auth/auth.go`（REST 链的认证中间件，装配见
`internal/server/rest_server.go` 的 `NewRestMiddleware`——链序：recovery → 请求日志 →
审计日志 → 参数校验 → selector(auth + authz)）：

1. 取 Bearer token → `accessTokenChecker.IsValidAccessToken`（JWT 验签 + 过期 + **Redis 缓存吊销核对**，
   见 [authentication.md](./authentication.md) 第 3 章）→ 失败即 401；
2. `NewContext(ctx, tokenPayload)`：业务层经 `auth.FromContext` 取 operator（`created_by` 注入源）；
3. `viewer.WithContext(ctx, appViewer.NewUserViewer(uid, tid, ouid, traceID, BuildDataScopes(...)))`：
   **隔离层判定依据**在此构建（`pkg/entgo/viewer/user_viewer.go`）；
4. OperatorMetadata 注入（`pkg/metadata`，当前 REST 配置 `WithInjectMetadata(false)` 关闭）。

ViewerContext 语义（`viewer.Context` 接口实现）：

| 实现 | tid | IsPlatform | IsSystem | 用途 |
|---|---|---|---|---|
| `UserViewer` | 令牌 tid | tid==0 | false | 请求上下文；tid==0 平台管理员全量，tid>0 租户 |
| `SystemViewer`（`pkg/entgo/viewer/system_viewer.go`） | 0 | true | true | 后台任务/闸门自查询（`NewSystemViewerContext`） |
| `NoopContext`（库） | — | — | — | 登录/找回流程（配 `privacy.DecisionContext(Allow)` 显式绕过） |

`BuildDataScopes`（同文件）把令牌 dss/dsu/旧 ds 聚合结果转成库层数据范围结构——
UNSPECIFIED 剔除、空集不兜底（交库规则 fail-closed），语义见 data_scope_design 第 3-4 节。

另有一条备用注入路径 `pkg/middleware/ent/ent.go`（从 OperatorMetadata 重建 viewer）——
**当前 REST 链未挂载**（全仓无 `ent.Server(` 引用），保留备用。

## 4. HTTP 层闸门（租户访问检查）

### 4.1 触发条件与取值

认证中间件内联调用（`auth.go` 尾段）：

- **仅 tid>0 触发**；平台管理员（tid==0）直接放行——所以"平台管理员测通"不等于"租户测通"；
- 匹配键是 **`htr.PathTemplate()`（路由模板）+ HTTP method**，不是原始 path——
  Api 表存的是 OpenAPI 文档里的路由模板（如 `/admin/v1/users/{id}`），带路径参数的请求按模板命中。

### 4.2 三段检查（`tenant_access_checker.go`，全部 fail-closed）

| 段 | 检查 | 拒绝条件 |
|---|---|---|
| 1 | 租户行 + WithPlan 预载 | 查不到 / `status != ON` → 403（OFF/EXPIRED/FREEZE 一律拒） |
| 2 | 到期只读判定 | `expired_at` 已过 **且** 套餐 `expiry_policy == READONLY` 时：非 GET/HEAD/OPTIONS → 403 |
| 3 | Api 表 `(path, method)` → `business_module` → 套餐白名单 | Api 表缺行 → 403；模块 UNSPECIFIED（未归类）→ 403；租户未挂套餐 → 拒全部业务模块；白名单 `sys_plan_modules` 计数为 0 → 403 |

枚举映射：ent `api.BusinessModule` ↔ proto `identityV1.Module`（checker 内两张 switch 表）。
白名单查询用 `SystemViewerContext`（闸门自身跨租户查询的合法通道）。

**注意**：闸门本体只实现 READONLY 档的只读降级——这是刻意的分工：BLOCK_LOGIN / FREEZE 两档
经**小时级到期扫描任务**（`AsyncTenantExpiryScan` → `EnforceExpiryPolicies`）把租户状态置
EXPIRED / FREEZE 并吊销该租户全部用户令牌，随后由本闸门第 1 段（status!=ON 拒）与登录链路的
租户状态检查接管。三档完整语义、生效时延与恢复流程见
[plan_billing.md](./plan_billing.md) 第 4 节。

### 4.3 Api 表的生成与同步（`internal/service/api_service.go`）

- `SyncApis`：`Truncate` 全表 → `syncWithOpenAPI`（解析内嵌 OpenAPI 文档 `assets.OpenApiData`，
  逐 (path, method) 生成行，`module`/`module_description` 取 OpenAPI **tag**，
  `business_module` 经 `pkg/constants/module_mapping.go` 的 `ServiceTagToBusinessModule` 映射）
  → `authorizer.ResetPolicies`（authz 策略同步重建）；
- **新服务的登记义务**：`ServiceTagToBusinessModule` 不登记新服务 tag ⇒ 其全部端点
  `business_module = UNSPECIFIED` ⇒ **所有租户（含挂了对应模块套餐的）一概 403**；
- 播种时机：Api 表仅在**空表**时启动期自动同步（count==0 守卫）；已部署实例新增端点必须在
  管理页「接口管理 → 接口同步」手动触发全量重建，否则新端点对租户 fail-closed；
- 同步后建议核对菜单（菜单播种同理见 `default_data.go`）。

## 5. 数据层隔离

### 5.1 读隔离（库层 `TenantPrivacy.EvalQuery`）

- context 缺 ViewerContext → **报错拒绝**（不是放行）；
- `IsPlatformContext || IsSystemContext` → 放行；
- 其余 → 反射调用 query 的 `Where` 注入 `tenant_id = viewer.tid`；反射签名校验失败
  （ent 升级改签名）→ **报错而非静默跳过**（防过滤悄悄消失）。

### 5.2 写隔离（两道，同谓词）

**库层 `TenantPrivacy.EvalMutation`（go-crud v0.0.55）**——对**非 Create** 变更：

- 平台/系统上下文放行；缺 viewer 拒绝；
- 经 `WhereP` 向 mutation 注入 `tenant_id = viewer.tid`，覆盖 Update/UpdateOne/Delete/DeleteOne
  全形态（`UpdateOneID`/`DeleteOneID` 的主键谓词与本谓词 AND，跨租户主键 0 行命中、返回 NotFound）；
- **tenant_id 改值防护**：mutation 里显式 Set `tenant_id` 时，仅"旧行、新值、当前 viewer 三者同租户"
  的冗余设置放行，跨租户搬迁/改他租户记录一律拒绝；无法验证（缺 `OldTenantID` 接口）时拒绝。

Create 分支：租户上下文**强制覆盖** `SetTenantID(viewer.tid)`（防伪造，优先强类型接口、
反射兜底、都不可用则报错）；平台上下文尊重代码里的显式 `SetTenantID`。

**仓内 `TenantMutationGuardPolicy`（`schema/tenant_mutation_guard.go`）**——全部 33 张带租户表
（`sys_access_keys` 于 2026-09-12 补挂，见第 6 节修复记录）经 schema `Policy()` 追加：
对非 Create 变更同样经 `WhereP` 注入租户谓词，缺 viewer 拒绝、平台/系统放行。
**历史**：该守卫诞生于库旧版本（EvalMutation 仅覆盖 Create、Update/Delete 直接放行）的时代；
**当前库版（v0.0.55）已补齐全形态**，守卫成为与库层同谓词的**冗余第二道防线**
（纵深防御——任一层回归/被移除仍兜底），文件头注释已按此更新（2026-09-12）。

### 5.3 数据范围叠加（试点）

`sys_positions` 挂 `TenantAndDataScopePolicy`：查询侧 = 租户（库层）+ 数据范围
（`rule.PermissionRule`，五档语义见 data_scope_design 第 2/4 节）；变更侧 = 租户变更防护。
其余表接入数据范围照 data_scope_design 第 4 节步骤（两列齐备 + Policy 改组合策略 + 守卫单测）。

## 6. 覆盖边界（不隔离什么）

| 路径 | 状态 | 说明 |
|---|---|---|
| ent 数据层（33 张表读写） | ✅ 隔离 | 本文档第 5 节 |
| **应用层自有存储**（Redis 缓存键、MinIO 对象路径、asynq 任务载荷） | ❌ 不自动注入 | 隔离谓词只存在于 ent privacy；这些路径必须业务侧显式携带/校验租户维度 |
| GORM 仓内路径（`internal/data/gorm/`） | ⚠️ 依赖库侧 | 仓内无独立镜像插件（grep 无 TenantIsolation）；隔离依赖 go-crud/gorm 的 viewer 机制，**未做深验证**，接 gorm 路径的租户表前先核实 |
| 跨服务出站（脚本 HTTP egress、Webhook） | ❌ | 按域名白名单管控，无租户维度（[script_system.md](./script_system.md)） |
| **租户内跨用户**（同 tenant 下 A 读/写 B 的行） | ❌ 完全不覆盖 | 隔离谓词只有 `tenant_id` 一列，"只看自己的行"必须业务侧自己钉（收件箱读/写已钉，见下） |
| 平台管理员上下文 | 放行 | 设计使然（tid==0 全量） |
| 登录/找回/闸门自查询 | 例外通道 | NoopContext+privacy.Allow（登录）/ SystemViewerContext（闸门、机器令牌交换的 AK 查询）——审计落库自身经 SystemViewer 写入 |

**"自己的行"没人钉过：收件箱越权读＋越权写（2026-09-20 实测并修）**。
`GET /admin/v1/internal-message/inbox` 的过滤条件整个来自调用方 `query` 字符串，租户隔离只保证"读不到别租户的行"，
同租户内换一个 `recipientUserId` 就能读到别人的收件记录（`title`/`content` 由父消息回填，一并带出）。
三端页面各自在前端塞 `recipientUserId`，所以这个条件从来不是服务端事实。
修法与位置见 `internal_message_recipient_repo.go` 的 `List`（非平台/非系统上下文强制
`recipient_user_id = viewer.UserID()`）与 [notification_domain_design.md](./notification_domain_design.md) §7。
**写侧同形问题同日修复**：`MarkNotificationAsRead` / `MarkNotificationsStatus` / `DeleteNotificationFromInbox`
原先一律用请求体的 `user_id` 定作用域，后果比读侧重——`DeleteNotificationFromInbox` 在 `recipient_ids`
为空时是"按用户维度整箱删除"，换一个 id 就是清空别人的收件箱。现收进同一个助手
`inboxScopedUserID`（平台/系统上下文豁免同读侧），两种落定语义与回归测试见
`internal_message_recipient_repo_sqlite_test.go` 的 `TestInternalMessageRecipientInboxWritesSqlite`：
带 `recipient_ids` 时跨用户调用缩成空操作，`recipient_ids` 为空时整箱操作缩回**调用者自己**。
**给"按人归属"的资源接入时的教训**：凡是"这张表每行属于某个用户"的路径，归属谓词要**读写两侧**都落在
repo 的查询/变更构造上，而不是指望调用方传对——三端各有一份前端，漏一份就是一个洞。

**`sys_access_keys` 守卫缺口的修复记录（2026-09-12）**：该表曾长期是 33 张带租户表中
唯一未挂仓内守卫的表（schema 无 `Policy()` 覆写）。补挂后与其他表一致（库层+仓内双防线），
并新增守卫单测 `TestTenantMutationGuardAccessKey`（`tenant_guard_test.go`）钉住
其跨租户 Update/Delete 0 行命中，防线回退会被测试捕获。当前 33 张表全部双挂。

## 7. 运维与排障

### 7.1 新表/新端点接入清单

1. **新表带租户**：schema `Mixin()` 加 `mixin.TenantID[uint32]{}`（库层隔离自动生效）+
   **同时** `Policy()` 返回 `&TenantMutationGuardPolicy{}`（与其余 32 张一致的第二道，
   见第 6 节 access_key 教训）→ `gow ent`；
2. **表同时接数据范围**：另见 data_scope_design 第 4 节（两列齐备 + 组合策略 + 守卫单测）；
3. **新端点**：部署后管理页「接口同步」；
4. **新服务（新 OpenAPI tag）**：`pkg/constants/module_mapping.go` 登记——漏登记=租户全 403。

### 7.2 租户 403 排障决策树

按序核对（全部可在管理页/库里直接查证）：

1. 令牌 tid 是否>0（平台管理员不走闸门）；
2. Api 表是否有该 (路由模板, method) 行——没有=该实例没跑「接口同步」；
3. 该行 `business_module` 是否 UNSPECIFIED——是=服务未在 `module_mapping.go` 登记；
4. 租户是否挂套餐、套餐白名单是否含该模块；
5. 租户 `status` 是否 ON；`expired_at` 是否已过且套餐 READONLY（只读降级会让写操作 403）。

### 7.3 守卫测试范式

`internal/data/ent/tenant_guard_test.go`：mockViewer（实现 `viewer.Context`）+
真实 PostgreSQL `gwa_guard_test` 库 + 迁移 + TRUNCATE 复位，矩阵断言
跨租户 Update/Delete 0 行命中/NotFound、本租户正常。当前覆盖 `sys_dict_types`
（`TestTenantMutationGuard`）与 `sys_access_keys`
（`TestTenantMutationGuardAccessKey`，2026-09-12 补挂守卫时同步新增）。
新挂守卫的表照此补测；数据范围矩阵范式见 `data_scope_guard_test.go`。

### 7.4 会话级联动

数据范围/字段黑名单随刷新令牌生效（TTL 内）、需立即生效用「在线用户 → 强制下线」
（吊销令牌对，见 [authentication.md](./authentication.md) 第 6 章）。

## 8. 已知问题与待办

| 项 | 现状 | 建议 |
|---|---|---|
| ~~`sys_access_keys` 缺仓内守卫~~ | **已修复（2026-09-12）**：schema 补挂 `Policy()` + `gow ent` 重生成 + 守卫单测钉住（第 6 节修复记录） | — |
| ~~`tenant_mutation_guard.go` 文件头注释过时~~ | **已修复（2026-09-12）**：注释已更新为"库层已全覆盖、本守卫为冗余第二道防线"并保留历史说明 | — |
| ~~`expiry_policy` 的 BLOCK_LOGIN/FREEZE 未实现~~ | **此前记载有误，实为已实现**：经小时级到期扫描任务的状态映射执行（[plan_billing.md](./plan_billing.md) 第 4 节）；闸门分工只承担 READONLY 即时降级 | FREEZE 与 EXPIRED 效果等价为已知设计现状（plan_billing 第 10 节），如需差异化再立项 |
| entql feature | 守卫与库层的 `WhereP` 注入依赖 entql 生成的 `WhereP` 方法 | schema 生成配置中必须保持 entql feature 开启，关掉=两道写隔离同时静默失效 |
| gorm 路径隔离 | 未深验证 | 接入 gorm 租户表前核实库侧行为并补测 |
