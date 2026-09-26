# 只保留一个前端：裁剪步骤与代价

> 读者：拿本项目做二开、只打算长期维护**一套**前端的团队。读完你会得到：删掉另外两端的完整步骤，以及三处会"悄悄回流"的耦合点。
>
> **先说定位**：仓库里有三个前端目录，是为了让用 React 的团队和用 Vue 的团队**各取所需**，不是给某一个团队同时供两套、更不是要求你维护三套。"三端都可用"由上游维护者负责；你只需要那一个能跑起来的目录。
>
> 本文所有结论都是仓库内实测（命令 + 文件行号），不是推断。三端各自都有完整页面集（2026-09-25 实测
> 页面文件数 react 96 / vue-element 102 / vue-vben 89；复测：
> `find frontend/admin/react/src/pages -name '*.tsx' | wc -l`、
> `find frontend/admin/vue-element/src/pages -name '*.vue' | wc -l`、
> `find frontend/admin/vue-vben/apps/admin/src/views -name '*.vue' | wc -l`），
> 所以这不涉及"哪一端功能不全"，只涉及"你要维护几端"。

## 1. 三端互不依赖，可以放心删

| 检查项 | 实测结果 |
|---|---|
| 包根 | 各端是独立 pnpm 根 + 独立 lockfile（`frontend/admin/react/package.json`、`.../vue-element/`、`.../vue-vben/`）；仓库根、`frontend/`、`frontend/admin/` **都没有** `package.json` 或 `pnpm-workspace.yaml` → 删任一端不影响另一端安装 |
| 部署 | 各端自带 `frontend/admin/<端>/scripts/deploy/{Dockerfile,nginx.conf,build-local-docker-image.sh}`，随目录一起消失；`backend/docker-compose.yaml` 与 `backend/docker-compose.libs.yaml` 对前端**零引用** |
| CI | `.github/workflows/` 下仅有一个 docs-parity workflow（校验三语 README 行级一致），不涉及三端 → 没有写死三端的流水线要改 |
| 后端代码生成 | `make api` / `make openapi` / `make ent` 都不碰前端目录；只有 `make ts` 会（见 §3） |

唯一例外：`frontend/admin/vue-vben` 本身是个 monorepo（`pnpm-workspace.yaml:1-12` 含 `internal/* packages/* apps/* scripts/*`），但它的边界在自己目录内，删它不影响别人、删别人不影响它。

## 2. 选哪一端

| 端 | 规模（页面文件 / 源码行，2026-09-25 实测） | 形态 | 门禁命令 | dev 端口 |
|---|---|---|---|---|
| react | 96 / 5.7 万（`react/src` 的 ts+tsx+css） | ProTable + DrawerForm + TanStack Query | `npm run typecheck` | 5888 |
| vue-element | 102 / 7.1 万（`vue-element/src` 的 ts+vue+样式） | ProPage 配置驱动 + vxe-table + ElForm | `npx vue-tsc --noEmit` | 5777 |
| vue-vben | 89 / 9.7 万（**整个 monorepo**：apps + packages + internal；只算 `apps/admin/src` 是 4.6 万） | VxeGrid + useVbenDrawer，Vben 5.x monorepo | `pnpm run check:type` | 5666 |

复测（在 `frontend/admin/` 下）：
`find react/src -type f \( -name '*.ts' -o -name '*.tsx' \) | wc -l` 一类命令即可；vben 的规模要按
"你要不要连框架包一起维护"来读——留 vben 就等于收下它整个 `packages/`。

两条选择依据：

- **要跟上游长期同步 → 选 react**。本仓工序是所有新模块先在 react 实现（根 [`AGENTS.md`](../AGENTS.md) 「react 先行，其余移植」），另两端是移植产物，所以 react 端最早拿到新页面。
- **保留 vue-vben → 别动工具链版本**。它的 `packageManager` 钉死 `pnpm@11.18.0`、catalog 用精确版本（见 `frontend/admin/vue-vben/AGENTS.md`「工具链与已知坑」），裁剪时不要顺手升级。

## 3. 裁剪三步

### 第 1 步：删目录

删掉 `frontend/admin/<不用的两端>` 整个目录即可，无需在别处注销。

### 第 2 步：断掉生成链的回流（最容易漏）

`backend/Makefile:98-103`：

```make
ts:
	cd api && \
	buf generate --template buf.vue-vben.admin.typescript.gen.yaml && \
	buf generate --template buf.vue-element.admin.typescript.gen.yaml && \
	buf generate --template buf.react.admin.typescript.gen.yaml
```

删掉不用的那两行，并删对应的 `backend/api/buf.<端>.admin.typescript.gen.yaml`。

**漏掉这一步不会报错，只会一直脏。** 实测：把 `frontend/admin/react/src/api/generated/` 整个移走，再跑一次 `cd backend/api && buf generate --template buf.react.admin.typescript.gen.yaml` —— 退出码 0、控制台无提示，目录连同 `admin/service/v1/index.ts` 被原样建回来。三个模板同形（各自 `:3` 的 `clean: true` + 各自 `:24` 的硬编码 `out:`），所以另外两端一样。结果就是：你以为删干净了，但此后每次 `make ts` 都凭空产出一坨孤儿文件，`git status` 再也不干净。

（顺带一条：三行命令用 `&&` 串着，所以任何一端生成失败会 abort 后面两端——多留一端就多一个失败点。）

### 第 3 步：收 CORS 白名单

`backend/app/admin/service/configs/server.yaml:25-35` 的 `origins` 是逐端列的：三个演示域名 +
`localhost:5666/5777/5888/5667/5778`。删掉用不上的那些。

注意 `5667` / `5778` 与 `5666` / `5777` 成对出现：vite 端口被占用时会自动 +1（教程 02 第 6 节），
这两个就是给顺延留的余量。**react 没有这一位**——表里只有 `5888`，没有 `5889`。所以裁剪时
**给保留的那一端补两位**（含 5888 顺延用的 5889），别只留一个端口——默认端口一旦被占
（上一个 dev server 没退干净），实际端口顺延到 +1 就撞 CORS。保留的是 react 且不改配置的话，
这个坑现在就已经在：5888 被占时前端跑在 5889，而后端白名单里没有它。

## 4. 唯一会咬人的数据层耦合：菜单表没有"端"这个维度

机制：

- `sys_menu` 只有 `type / path / redirect / alias / name / component / meta / module` 八列（`backend/app/admin/service/internal/data/ent/schema/menu.go:32-98`，其余列来自 mixin：`parent_id` 与审计字段）。其中 `module` 是**业务模块**枚举（套餐白名单靠它过滤），**不是端标识**；`GetNavigation` 也只按角色 + 套餐白名单过滤，没有任何"哪个前端"的维度。
- 三端的「菜单同步」都写死 `mode: 'MERGE'`（`frontend/admin/react/src/api/hooks/menu.ts:201`、`frontend/admin/vue-element/src/api/composables/menu.ts:176`、`frontend/admin/vue-vben/apps/admin/src/api/composables/menu.ts:193`），而 MERGE 的定义就是"**数据库多出的菜单保留**"（`backend/api/protos/permission/service/v1/menu.proto:450`）。
- ⇒ 你删掉的两端此前同步进库的菜单行**不会因为删目录而消失**。

影响：

- **默认不咬**：三端 `accessMode` 默认都是 `frontend`（`frontend/admin/react/src/core/preferences/config/default.ts:5`、`frontend/admin/vue-element/src/core/preferences/config/default.ts:5`、`frontend/admin/vue-vben/apps/admin/.env.development:25`），侧边栏渲染的是本地路由，孤儿菜单只躺在「菜单管理」表里。
- **切到 `backend` 模式就会咬**：孤儿行会进侧边栏，点进去没有对应组件。两条处置：
  - 手工删掉孤儿行（安全）；
  - 或在菜单管理页触发 REPLACE 全量重建——**代价很大**：`menu.proto:449` 明确写了"菜单 ID 全部变化，角色-菜单授权失效"（实现见 `menu_repo.go:410` 的先 `Delete()` 再插），重建后要重新给角色授权。别顺手点。

对裁剪有利的一条性质：MERGE 按全路径匹配并**覆盖 `component`**（`menu_repo.go:483` 的 `SetNillableComponent`），
而三端同步时写的 `component` 串形状一致（都是相对 `pages/`／`views/` 的 `app/<模块>/<页>/index.vue`，
顶层布局写 `BasicLayout`；见 `react/src/api/hooks/menu.ts:150-151`、`vue-element/src/api/composables/menu.ts:112-135`、
`vue-vben/apps/admin/src/api/composables/menu.ts:131-160`）。所以裁剪后**让保留的那一端最后同步一次**，
库里就统一成它认识的写法。两处例外要手工处理：`IFrameView` 只有 vben 会产出（另两端认不出这个串），
以及同一路径在某一端确实没有对应页面文件时，那一行本来就是孤儿。

## 5. 文档不要改

`docs/` 里按三端口径写的表述分布很广，自己数一遍（命中数随口径浮动，别信二手数字）：

```bash
grep -roc -E 'vue-vben|vue-element|三端|5888|5777|5666' docs README*.md | grep -v ':0' | sort -t: -k2 -rn | head
```

量级参考（2026-09-25 跑上面那条命令的实际前三页结果）：
`docs/notification_domain_design.md` 68 处、`docs/design-language.md` 34 处、本文自己 29 处
（本文就是讲三端的，命中多属正常），其后 `windows-startup-guide.md` 18、`frontend_authority.md` 15、
README 三个语言版本 15/13/13、教程 02/01/04 各 11/10/8。

这些是上游口径，逐处改掉的结果是你在每次跟上游同步时都带着一堆冲突。**在 fork 里加一行说明就够了**：

> 本项目仅维护 `<你的端>` 一套前端；文档中提到的另外两端的端口、演示地址与生成目标不适用。

需要真正保持一致的只有一处：`docs/design-language.md` 是视觉权威值表，如果你改了自己这一端的颜色/圆角，改它，不用管另两端。

## 深读

- [tutorial/02 从零跑起来](./tutorial/02-get-it-running.md) —— 三端各自的启动命令与端口
- [frontend_authority.md](./frontend_authority.md) —— 路由 `frontend` / `backend` 双模式的三端实现对照
- [README.md](../README.md) 的「为什么是三套前端」（`README.md:28`） —— 三选一的能力声明
