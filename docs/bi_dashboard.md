# BI 仪表盘（设计器 + 大屏渲染）

BI 分析能力：可视化设计器编排图表（柱状/折线/饼图/仪表盘等）、跨图联动、下钻、主题预设，
仪表盘/大屏持久化到数据库并按租户隔离。

设计器与渲染内核来自 **[go-wind-ui](https://github.com/tx7do/go-wind-ui)**（`@go-wind/bi-core` +
`@go-wind/bi-vue` / `@go-wind/bi-react`，已发布 npm）；本仓是它的宿主——持久化、权限、菜单、
多租户都在这一层。库的选型与设计文档：go-wind-ui 仓 `docs/bi-designer.md`。

## 定位边界

- **schema 即 JSON**：payload 列存设计器产出的整份 schema（数据集 + 图表布局 + 主题 +
  联动/下钻声明），服务端**不解析、不校验内部结构**，只透传——schema 演进只发生在库里
  （含 schemaVersion 迁移），后端零改动。打印模板（print_template）同口径。
- **数据集两类**：`static`（内嵌行，设计期录入）与 `api`（运行时走 resolvers 回调取数）。
  resolvers 是宿主注入的取数函数表（键 = dataset.id），schema 与渲染器不认识任何 HTTP 客户端；
  本仓当前三端只消费 static 演示数据，接真实业务时在宿主层注入 resolver。
- **库与宿主**：图型/布局/主题/联动/下钻/撤销栈全在 `@go-wind/bi-*`；本仓只管存取与展示。

## 数据模型（bi_dashboards）

| 列 | 说明 |
|---|---|
| `name` / `code` | 名称 / 编码（`(code, tenant_id)` 唯一；业务按编码取） |
| `kind` | `DASHBOARD`（管理页仪表盘）/ `SCREEN`（大屏）/ `TEMPLATE`（可复用模板） |
| `remark` | 备注 |
| `payload` | 设计器 schema JSON 原文（MySQL `json` / Postgres `jsonb`） |
| `chart_count` / `theme_key` | 冗余列，列表页直显免解析 JSON |
| `thumbnail` | 缩略图 URL（设计器导出 PNG 后可上传，预留） |
| `is_built_in` | 系统内置（禁删，可复制派生） |
| `is_public` | 租户内共享（false 仅创建者可见） |
| `version` | 乐观锁：更新必须回传读到的版本，成功后 +1 |
| `is_enabled` / `sort_order` | 启用开关 / 排序 |
| `TenantID` / `OperatorID` / `TimeAt` mixin | 租户隔离 + 操作人 + 时间戳 |

> **乐观锁语义**：设计器是整份覆盖保存，多标签/多人同时编辑同一仪表盘时，
> 回传读到的 `version` 走 CAS（`WHERE id AND version`），0 命中区分“行已删(404)”与
> “被他人改过(409)”，前端提示一次并自动刷新版本继续后续编辑。

## API（admin BFF）

| Method | Path | 说明 |
|---|---|---|
| GET | `/admin/v1/bi-dashboards` | 分页列表（contains 搜索 name/code，布尔列传 EQ） |
| GET | `/admin/v1/bi-dashboards/{id}` | 详情 |
| POST | `/admin/v1/bi-dashboards` | 创建 |
| PUT | `/admin/v1/bi-dashboards/{id}` | 更新（`data.version` 回传触发乐观锁） |
| DELETE | `/admin/v1/bi-dashboards/{id}` | 删除（内置行 400） |
| POST | `/admin/v1/bi-dashboards/{id}:copy` | 复制派生（副本恒非内置，名称/编码缺省自动派生） |

内置种子：`builtin.bi.basic`（基础仪表盘模板，kind=TEMPLATE），启动期按编码缺一补一、不覆盖。
已部署实例新增端点须在管理页「接口同步」重建 Api 表（go-crud 租户闸门 fail-closed）。

## 三端入口（react / vue-element / vue-vben 同构）

| 页面 | 路由 | 说明 |
|---|---|---|
| 仪表盘 | `/bi/dashboard` | 渲染态：无 id 读本地草稿/演示；`?id=N` 渲染库中仪表盘 |
| 设计器 | `/bi/designer` | 无 id：本地草稿（`bi:schema:default`）；`?id=N`：加载库中仪表盘，画布改动防抖自动保存回写（乐观锁），「预览」↔ 仪表盘互跳 |
| 仪表盘管理 | `/bi/manage` | 列表（搜索/分页/启用/内置标记/图表数/主题）+ 全屏设计器抽屉 + 复制/删除 |

菜单：三端统一「BI 分析 → 仪表盘 / 设计器 / 仪表盘管理」。BI 路由为三端**前端路由自带**，
不走后端种子菜单；权限 code：`sys:platform_admin` / `sys:tenant_manager`（与 ele 一致）。

## 主题跟随语义（重要）

- schema **不设 `theme.key`** → auto：渲染器监听 `html.dark` 自动跟随宿主亮暗
  （vue 端组件响应式更新，ECharts 轴文字/图例色同步重绘）；
- 设 `theme.key` → 钉死预设（大屏投屏需要钉死时在设计器主题下拉里选）；
- 本仓演示数据与内置模板均不钉死——宿主切亮暗时仪表盘跟随变化是默认预期。

> 卡片等 chrome 样式：bi-react 卡片视觉全在 `styles.css`（含 `html.dark` 分支），
> 不允许 inline 写死视觉值（inline 优先级会压死暗色分支，教训见 go-wind-ui 4a9d8a7）。

## 全屏

仪表盘页与设计器页都提供页级全屏按钮（元素级 `requestFullscreen`，非整个文档）：
ele 圆形 SVG 图标钮 / vben 框架 `VbenIconButton` + Maximize/Minimize 图标 / react antd 图标钮。
进入全屏时脚本把页面底色刷成当前主题布局底色（fullscreen 元素默认透出 UA 黑底）。

## 宿主集成注意（go-wind-ui 消费侧）

- `@go-wind/bi-*` 的 dist 在 vite 里是普通预构建包，无需 optimizeDeps 排除；
  **换库版本后要清 `node_modules/.vite` 并重启 dev server**，否则 vite 预构建缓存不读新 dist。
- 当前三端 node_modules 里的 `@go-wind/bi-*` dist 是**本地构建手工同步**的
  （npm token 恢复前 0.2.4/0.3.0 未发布）：`pnpm install` 会还原成 registry 旧版，
  需重拷 `go-wind-ui/packages/<pkg>/dist` 或等发布。
- 设计器挂载即 emit 一次 schema 快照（宿主据此拿到待保存初值）；`initialSchema` 是
  **挂载时快照**，换目标必须换 key 强制重挂载（react 桥接壳尤其注意）。
- 自动保存基线必须用「首帧 emit 定基线」：`normalizeSchema` 补默认值后画布规范化
  JSON 与服务端原文永不相等，拿原文当基线挡不住回显误保存。

## 已知边界 / 后续

- 独立设计器无 id 时仍是 localStorage 草稿（`bi:schema:default`），换设备/换人不可见；
  若草稿模式要彻底退场，把默认入口改为 `/bi/manage` 的“新建”即可。
- 缩略图列预留未实现（设计器导出 PNG 上传后即可挂列表预览）。
- 版本历史（每次保存留快照）未做；`version` 已就位，后续可加 revisions 表。
- 数据集 resolver 注入与业务绑定（api 型数据集）是宿主职责，演示期未接。

## 验证过的链路

- 后端 e2e：Create → CAS 更新（v1→v2）→ stale 写 409 → Copy 200 → 内置禁删 400 → Delete 200。
- 三端：列表 3 行、抽屉编辑保存（version 递增实证）、`/bi/dashboard?id=N` 渲染库中行、
  设计器主题切换自动保存、打开 6 秒无回显误保存、全屏进入/退出正常。
