# AGENTS.md — go-wind-admin Monorepo 开发指南（AI Agent 入口）

本文件是 monorepo 总入口：定位布局、指路各端规范、声明全仓铁律。**深入开发前必读对应端的 AGENTS.md**。

## 仓库布局

```
backend/                    Go + Kratos + Ent（DI 手写装配 wiring_*.go，已弃用 Wire；HTTP :7788，SSE 网关 :7789）
frontend/admin/
├── react/                  React 19 + antd 6 + ProComponents + TanStack Query + zustand
├── vue-element/            Vue 3 + Element Plus + vxe-table + TanStack vue-query + Pinia
└── vue-vben/               Vben Admin 5.x monorepo（apps/admin + packages/*）+ Ant Design Vue
docs/                       文档体系（总入口 docs/README.md：教程层 docs/tutorial/ + 参考层专题文档）
```

## 三端门禁（必须保持全绿）

| 端 | 命令（在各自目录下） | dev 端口 |
|---|---|---|
| react | `npm run typecheck` | 5888 |
| vue-element | `npx vue-tsc --noEmit`（或 `npm run type-check`） | 5777 |
| vue-vben | `pnpm run check:type` | 5666 |

2026-09-07 起三端 typecheck 全部 0 错误。**门禁出现任何新报错，一律当作自己引入的 bug 修复**，不存在"可忽略的既有错误"。改完代码先跑门禁再声称完成。

## 全仓铁律

1. **不吞错**：任何 catch 至少二选一——`console.error/warn` 带出**原始错误对象**，或重新抛出。用户可见的通知/Message ≠ 日志（只有翻译文案）。合法裸 catch 仅限纯本地 best-effort 兜底且注释写明原因。历史教训：认证链路静默吞错曾让 bug 排查耗时数日。
2. **vue-vben 工具链版本已钉死**（catalog 精确版本 + packageManager 匹配本机 pnpm）：禁止改回 `^` 范围、禁止顺手升级 vue/typescript/vue-tsc/pnpm。原因与升级流程见 `frontend/admin/vue-vben/AGENTS.md`「工具链与已知坑」。
3. **搜索条件一律 contains 而非 EQ**、ID 类字段不进模糊搜索；CRUD 请求体必须包 `{ data: {...} }`——细节见 `.zcode/skills/add-crud-module/SKILL.md`。

## 开发策略：react 先行，其余移植

新功能/新模块以 **react 端为行为基准先实现**，验证通过后再移植到 vue-element / vue-vben。移植是"有参照的翻译"，远比三端并行首创便宜；vue-vben 框架变体语料薄，直接首创容易产出框架级错误（详见其 AGENTS.md）。

**CRUD 模块**：使用 `/add-crud-module` skill（后端 + 前端端到端流程）。

**代码生成器**：配套工具 [go-wind-toolkit/gowind-uiapp](https://github.com/tx7do/go-wind-toolkit/tree/main/gowind-uiapp)（桌面 GUI + CLI，从数据库表/SQL 生成前后端代码，含简单表单）。CLI（`gowind-cli`）非交互、JSON 输出，适合 Agent 调用。工具产物仍须按本仓铁律与约定验收补齐（`{ data: {...} }` 包裹、contains 搜索、已部署实例的新端点走管理页「接口同步」登记进 Api 表、`make ts` 生成三端 TS 等）。

> 系统默认数据（admin 用户、角色、菜单、权限、语言等）由服务启动时在 Go 侧自动播种（`pkg/constants/default_data.go` + 各 service 的 count==0 守卫；Api 表仅在空表时于启动期自动同步，**已部署实例新增端点须在管理页「接口同步」手动触发全量重建**，否则租户闸门 fail-closed 403），**不要**找 SQL 种子脚本，`backend/sql/` 下只剩演示数据。

## 后端任务：gow 优先，make 兜底

后端的运行与代码生成统一走 [gow CLI](https://github.com/tx7do/go-wind-toolkit/tree/main/gowind)（安装：`go install github.com/tx7do/go-wind-toolkit/gowind/cmd/gow@latest`），在 `backend/` 下执行。**接手后不要上来就用 Makefile 的 make 命令**：Windows 无原生 make、嵌套 Makefile 需要跨目录 cd，而 gow 自动发现 `app/*/service`，行为一致：

| 任务 | gow 命令 |
|---|---|
| 运行服务 | `gow run admin` |
| Ent 生成 | `gow ent`（全部服务）/ `gow ent admin` |
| Proto / API Go 代码 | `gow api` |
| 从数据库表生成 CRUD | `gow generate`（DSN 驱动；`--proto-only` 仅出 proto） |

项目已弃用 Wire（DI 为手写 `wiring_*.go`），不要运行 `gow wire`，也不要在新代码里引入 wire 依赖。

gow 未覆盖的任务（三端 TS 生成 `make ts`、OpenAPI `make openapi`、`make test/lint` 等）才退回 Makefile（`backend/` 根目录执行）。

## 本地验证要点

- 后端起在 `:7788`（`gow run admin`；启动方式见 `docs/windows-startup-guide.md` / `docs/backend_deploy.md`）；前端 dev 端口见上表，代理已配置好 API 转发。
- 登录账号：全新环境播种为 `admin / Abcd@1234`（`pkg/constants/default_data.go` 的 `DefaultUserPassword`）；本机库现状为 `admin / admin`（历史 e2e 改密残留，以本机实际为准）。图形验证码的答案可在 Redis 中按 `gowind:captcha:<captchaId>` 直接读取，便于自动化验证。
- vue-element 在 dev 下若见 router-view 塌空/白屏：先重启 dev server 再下结论（vite 依赖优化竞态已做遏制与自愈，见其 AGENTS.md「dev 白屏处置」）。

## 文档索引

- **文档总入口（两层索引：教程层 + 参考层）**：`docs/README.md`；渐进教程系列（面向采用者的 9 章学习路径）在 `docs/tutorial/`
- 各端规范：`frontend/admin/{react,vue-element,vue-vben}/AGENTS.md`
- 后端：`docs/backend_project_struct.md`、`docs/backend_deploy.md`、`docs/audit-log-producer-design.md`
- 前端权限模型：`docs/frontend_authority.md`
- 查询/分页规则：`docs/list_query_rule.md`
- 脚本系统：`docs/script_system.md`（Lua/JS 脚本级插件：钩子点/定时任务/HTTP 出站/安全模型；改钩子点或模块先读它）
- 认证与令牌链路：`docs/authentication.md`（登录全流程/令牌与刷新轮换/MFA/限流策略/会话吊销/已知问题；改登录、令牌、刷新、MFA、限流或登录策略前先读它）
- 多租户隔离：`docs/tenant_isolation.md`（上下文链路/HTTP 闸门/数据层读写隔离/套餐联动/覆盖边界与排障；改隔离层、Api 表、套餐门禁或给新表接租户前先读它）
- 套餐与计费管控：`docs/plan_billing.md`（三档到期策略全链路/模块白名单/配额与用量计量/租户数据清理；改套餐、配额、到期处置或排租户 403 前先读它）
- 任务调度系统：`docs/task_system.md`（配置与启动链/任务数据模型/调度生命周期/系统级常驻任务/脚本任务桥/多租户语义与排障；加任务类型、排"任务没跑"或接新调度需求前先读它）
- 数据权限范围：`docs/data_scope_design.md`（角色级行数据范围：语义/聚合/接入步骤/运维边界；新表接入数据范围或改聚合规则先读它）
- 设计语言规范：`docs/design-language.md`（三端视觉唯一权威值表，改颜色/圆角/布局尺寸先改这里再同步三端）
