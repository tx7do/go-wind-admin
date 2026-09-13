# 01 · 架构全景：一次请求的完整路径

> 前置：无。读完你会知道：代码都在哪、一个请求从浏览器到数据库经过了什么、有哪些横切系统在起作用。

## 1. Monorepo 布局

```
backend/                    Go 后端（kratos 微服务框架 + ent ORM）
├── api/                    proto 定义与生成物（protos/ 手写，gen/ 生成）
├── app/admin/service/      admin 服务本体（cmd/server 入口、configs 配置、internal/ 实现）
├── pkg/                    跨服务公共包（bootstrap、middleware、audit、scripting…）
├── scripts/                部署与环境脚本（docker/ deploy/ env/ backup/）
frontend/admin/
├── react/                  React 19 + antd 6 + ProComponents + TanStack Query + zustand
├── vue-element/            Vue 3 + Element Plus + vxe-table + TanStack vue-query + Pinia
└── vue-vben/               Vben Admin 5.x monorepo（apps/admin + packages/*）+ Ant Design Vue
docs/                       本文档体系（教程层 + 参考层，见 [docs/README.md](../README.md)）
```

三套前端功能同构、技术栈各异，共用同一套后端契约（第 3 章展开）。后端目录的逐项说明见
[backend_project_struct.md](../backend_project_struct.md)——它是目录职责的权威参考。

**三个前端端口约定（本地开发）**：react `5888`、vue-element `5777`、vue-vben `5666`
（`apps/admin/.env.development` 的 `VITE_PORT`；若端口被占用 vite 会自动顺延，启动日志里有实际端口）。

## 2. 一次请求的完整路径

以"登录后的管理员在用户管理页点了一次查询"为例：

```
浏览器
  │  GET /admin/v1/users?...（后端路由本身带 /admin/v1 前缀；前端经 TanStack Query / vue-query 发起）
  ▼
开发态接线（三端两种形态，由各端 .env.development 与 vite 配置决定）：
  │  react —— baseURL 为相对路径，请求打到 dev server，由 vite 代理按 /admin
  │           前缀转发到 7788（避免跨域）
  │  vue-element / vue-vben —— baseURL 直接指向 http://localhost:7788，
  │           开发态跨域直连（后端 CORS 放行）
  ▼
Kratos HTTP transport（:7788）
  │  1. 路由匹配：由 BFF proto 的 google.api.http 注解生成的路由表
  │  2. 中间件链：JWT 验签与认证上下文注入、操作/API 审计日志、请求日志
  │  3. 生成的 handler：Register*HTTPServer 系列函数，解参、调 service 方法
  ▼
Service 层（internal/service/）
  │  业务逻辑薄层：从认证上下文取 operator、注入审计字段、委托 repo
  ▼
Repo 层（internal/data/）
  │  DTO↔entity 映射（CopierMapper + 字段转换器）；Update 走 FilterByFieldMask
  ▼
ent 隐私层（internal/data/ent/，代码生成）
  │  租户隔离（TenantPrivacy）、数据范围谓词、字段黑名单裁剪
  ▼
PostgreSQL / MySQL
  │
  ▼（响应沿原路返回：entity→DTO 转换、字段级裁剪、protojson 序列化）
```

关键认知：

- **service/repo 的 CRUD 骨架高度模式化**——第 4 章你会照着样例亲手写一遍；
- **横切关注点不在业务代码里**：认证在中间件、行级隔离在 ent 隐私层、审计在中间件与 driver 包装器——业务代码"看不见"它们，但每条路径都在其覆盖下；
- **DTO 与 entity 是两套类型**：proto 生成的 DTO 只在 service/repo 边界出现，库里存的是 ent entity，映射靠显式注册的转换器（时间、枚举），漏注册就是静默零值。

另有两条独立通道：

| 通道 | 端口 | 用途 |
|---|---|---|
| SSE transport | :7789 `/events` | 服务端推送（站内信、通知）；streamID 用 userId 标识，多设备 fan-out |
| asynq worker | Redis | 异步任务（定时任务、审计归档、广播扇出），与 HTTP 服务同进程启动 |

## 3. 横切系统地图

| 系统 | 一句话 | 实现位置 | 深读 |
|---|---|---|---|
| 认证 | JWT（access 内存 + refresh HttpOnly Cookie），登录口令应用层 AES 加密传输 | `pkg/middleware/auth`、`internal/service` 认证服务 | 第 5 章 |
| 授权（接口级） | authz 引擎判定"该用户能否调该接口"，策略存 DB 可热更新 | `pkg/middleware/auth` 引擎接入 | 第 5 章 |
| 租户隔离 | 行级 `tenant_id` 谓词，读写在数据层强制；HTTP 层另有 Api 表 `(path,method)` 闸门 | ent 隐私层 + `pkg/middleware` | 第 6 章 |
| 数据范围 | 角色级行过滤（五档），令牌承载聚合结果 | ent 隐私层（逐表 opt-in） | [data_scope_design.md](../data_scope_design.md) |
| 字段级权限 | 角色黑名单字段，服务端响应裁剪 + 前端隐藏 | 响应包装器 + 三端 access store | [frontend_authority.md](../frontend_authority.md) |
| 审计日志 | 六类日志全覆盖，含 SQL 级 data access 采集与词法脱敏 | `pkg/middleware/logging`、driver 包装器 | [audit-log-producer-design.md](../audit-log-producer-design.md) |
| 异步任务 | asynq 调度（cron/周期），Redis 队列 | `internal/service` 任务桥 | [task_system.md](../task_system.md) |
| 脚本系统 | Lua/JS 脚本级插件：实体钩子/定时任务/事件/HTTP 出站 | `pkg/scripting` + 管理页 | [script_system.md](../script_system.md) |
| 对象存储 | MinIO（S3 兼容），预签名上传下载，元数据落库 | `internal/data` 文件模块 | [backend_file_upload.md](../backend_file_upload.md) |

## 4. 生成代码与手写代码的边界

这个仓库大量依赖代码生成。**分不清这条边界，改错文件是新人最常见的翻车点：**

| 目录 | 产生方式 | 能否手改 |
|---|---|---|
| `backend/api/gen/go/**` | `gow api` 从 proto 生成 | ❌ 改 proto 重新生成 |
| `backend/app/admin/service/internal/data/ent/**`（除 `schema/`） | `gow ent` 从 schema 生成 | ❌ 改 schema 重新生成 |
| 三端 `api/generated/**` | `make ts`（后端）从 BFF proto 生成 | ❌ |
| `backend/app/admin/service/internal/data/ent/schema/**` | 手写 | ✅ |
| `backend/api/protos/**` | 手写 proto | ✅ |
| `internal/service/`、`internal/data/*_repo.go` | 手写（模式化样板） | ✅ |
| `cmd/server/wiring_*.go` | 手写 DI 装配（项目已弃用 Wire） | ✅ |

生成链路的完整机制在[第 3 章](./03-codegen-chain.md)，实战在第 4 章。

## 5. 本章小结

- 三前端共用一套 proto 契约生成的客户端，后端是唯一事实源；
- 请求路径：前端代理 → transport 中间件（认证/审计）→ service → repo → ent 隐私层 → DB；
- 横切系统与业务代码解耦，但每一层都有明确职责，动对应层前读对应参考文档；
- 先分清生成物与手写物，再动手改代码。

下一步：[02 · 从零跑起来](./02-get-it-running.md)。
