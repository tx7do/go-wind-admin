# 06 · 多租户与行级隔离

> 前置：[05 章](./05-permission-model.md)（三轴正交的框架）。读完你会理解：租户上下文怎么进请求、行级隔离在数据层怎么强制、部署侧的 Api 表闸门为什么是 fail-closed、以及隔离的**覆盖边界**——出了边界要自己负责。

## 1. 租户模型

- 每个带租户维度的表有 `tenant_id` 列（ent `mixin.TenantID[uint32]{}` 装配，见第 03 章）；
- 请求进入后端后，认证中间件（`pkg/middleware/auth`）从已验签的令牌构建
  **ViewerContext**（租户 ID + 平台/系统上下文标志，实现在 `pkg/entgo/viewer`），
  作为隔离层判定依据。平台管理员上下文（tid=0）跨租户可见；
- 租户与其套餐的绑定由平台管理员在「租户管理 / 套餐管理」页维护；租户侧用户看到的数据范围由本章所述各层自动裁剪。

## 2. HTTP 层闸门：Api 表 `(path, method)`

每个租户请求在业务逻辑之前，按（**路由模板**，method）查 **Api 表**（接口注册表，含模块归属）：

- 命中且该租户**套餐允许该模块** → 放行到业务；
- 未命中（表里没这行）或套餐不允许 → **fail-closed 403**。

关键运维事实：

- Api 表**仅在空表时**由启动期从内嵌 OpenAPI 文档自动同步；**已部署实例新增端点后必须在
  管理页「接口管理 → 接口同步」手动触发全量重建**——忘了这步，新端点对租户就是 403；
- 「平台管理员测通了」≠「租户测通了」：平台上下文不走这道闸门，验收要拿租户账号测。

## 3. 数据层：读与写各一条防线

**库层（`mixin.TenantID` 编译进每张租户表）**：go-crud 的 `TenantPrivacy` 同时覆盖
读与全部变更形态——查询自动注入 `tenant_id = <viewer tid>`；Update/UpdateOne/Delete/
DeleteOne 的 WHERE 恒含租户谓词（跨租户变更 0 行命中、单键操作返回 NotFound）；
Create 强制覆盖 tenant_id 防伪造；显式改 tenant_id 值仅放行"同租户冗余设置"。
编译进生成代码而非运行时 hook，业务侧无法绕过。

**仓内冗余层（`schema/tenant_mutation_guard.go` 的 `TenantMutationGuardPolicy`，全部带租户表挂载）**：
与库层同谓词的第二道防线（纵深防御）——它诞生于库旧版只覆盖 Create 的时代，
当前库版（v0.0.55）已补齐全形态，两层叠加。守卫行为由
`tenant_guard_test.go`（含 `sys_access_keys` 用例）钉住。

组合策略：`TenantAndDataScopePolicy`（`schema/data_scope_guard.go`）把租户变更防护与
数据范围查询过滤链在一起——任一子策略拒绝即整体拒绝。

**接新表**：`Mixin()` 加 `mixin.TenantID[uint32]{}`（库层全形态隔离自动生效），
且 `Policy()` 返回 `&TenantMutationGuardPolicy{}`——现有 33 张租户表**无一例外**都挂了
（其中岗位表 `position.go:188` 返的是组合策略 `TenantAndDataScopePolicy{}`，见第 4 节）。
完整步骤与守卫单测范式见参考文档第 7 节。

## 4. 数据范围（行级第二维）

同一行隔离层里还有组织维度的过滤（角色五档：全部/本部门/本部门及以下/自定义/仅本人），
由令牌 `dss`/`dsu` claim 承载聚合结果，查询时注入 `created_by = uid` 或
`org_unit_id IN (targets)` 谓词。**V1 仅试点岗位表**，其余表逐表 opt-in 接入。
五档语义、聚合规则、新表接入步骤、fail-closed 退化（单元集超 256 / 空集整体拒绝）：
全部见 [data_scope_design.md](../data_scope_design.md)——这是唯一权威，接入前通读。

## 5. 覆盖边界（重要）

行级隔离覆盖的是 **ent 数据层**。以下路径**不自动注入租户谓词**，业务侧必须自己携带并校验租户维度：

- 应用层自有缓存（Redis key 构造）；
- 对象存储路径（OSS / MinIO 对象 key）；
- 异步任务载荷（asynq 任务体）；
- 任何绕过 ent 的直连 SQL / GORM 路径。

接入新存储或新队列时，先想清楚租户维度怎么进 key / 路径 / 载荷，再写代码。

## 6. 套餐联动（plan / billing）

租户绑定的套餐决定两件事：可用**模块白名单**（闸门第 3 段，与 Api 表模块归类联动）与
**到期处置**——三档：READONLY 即时只读降级（闸门第 2 段）；BLOCK_LOGIN / FREEZE 经
小时级系统扫描任务把租户状态置 EXPIRED / FREEZE 并吊销该租户全部用户令牌，随后由
"非 ON 即拒"路径接管。套餐与配额的管理页在「套餐管理 / 配额管理」。运维侧注意：
租户报 403 或只读，先查套餐模块白名单与到期时间，再查接口同步。全链语义、生效时延、
续费恢复流程见套餐专文。

## 深读

- [tenant_isolation.md](../tenant_isolation.md) —— 租户隔离唯一权威：上下文链路、HTTP 闸门、数据层读写隔离全形态、覆盖边界（含 `sys_access_keys` 曾缺冗余层守卫、2026-09-13 补挂的修复记录）、接入步骤、租户 403 排障决策树
- [plan_billing.md](../plan_billing.md) —— 套餐与计费管控唯一权威：三档到期策略全链路、模块白名单上线 checklist、配额与用量计量、租户数据清理
- [data_scope_design.md](../data_scope_design.md) —— 数据范围唯一权威（五档/聚合/接入/运维/边界）
- [frontend_authority.md](../frontend_authority.md) —— 字段级权限（字段轴，与本章行级轴正交）
- `backend/app/admin/service/internal/data/ent/schema/tenant_mutation_guard.go`、
  `data_scope_guard.go` —— 仓内守卫与组合策略源码（文件头注释含守卫的历史定位与库 v0.0.55 现状）
- 根 [AGENTS.md](../../AGENTS.md)「仓库布局」节的 Api 表播种说明 —— 接口同步义务的出处

下一步：[07 · 审计与等保](./07-audit-compliance.md)。
