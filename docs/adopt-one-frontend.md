# 只保留一个前端：裁剪步骤与代价

> 读者：拿本项目做二开、只打算长期维护**一套**前端的团队。读完你会得到：删掉另外两端的完整步骤、三处会"悄悄回流"的耦合点，以及**目录所有权契约与 merge 升级 runbook**（§7——加模块只新增文件、merge 上游零冲突的操作规程）。
>
> **先说定位**：仓库里有三个前端目录，是为了让用 React 的团队和用 Vue 的团队**各取所需**，不是给某一个团队同时供两套、更不是要求你维护三套。"三端都可用"由上游维护者负责；你只需要那一个能跑起来的目录。
>
> 本文所有结论都是仓库内实测（命令 + 文件行号），不是推断。三端各自都有完整页面集（2026-09-25 实测
> 页面文件数 react 96 / vue-element 102 / vue-vben 89；复测：
> `find frontend/admin/react/src/pages -name '*.tsx' | wc -l`、
> `find frontend/admin/vue-element/src/pages -name '*.vue' | wc -l`、
> `find frontend/admin/vue-vben/src/views -name '*.vue' | wc -l`），
> 所以这不涉及"哪一端功能不全"，只涉及"你要维护几端"。

## 1. 三端互不依赖，可以放心删

| 检查项 | 实测结果 |
|---|---|
| 包根 | 各端是独立 pnpm 根 + 独立 lockfile（`frontend/admin/react/package.json`、`.../vue-element/`、`.../vue-vben/`）；仓库根、`frontend/`、`frontend/admin/` **都没有** `package.json` 或 `pnpm-workspace.yaml` → 删任一端不影响另一端安装 |
| 部署 | 各端自带 `frontend/admin/<端>/scripts/deploy/{Dockerfile,nginx.conf,build-local-docker-image.sh}`，随目录一起消失；`backend/docker-compose.yaml` 与 `backend/docker-compose.libs.yaml` 对前端**零引用** |
| CI | `.github/workflows/` 下仅有一个 docs-parity workflow（校验三语 README 行级一致），不涉及三端 → 没有写死三端的流水线要改 |
| 后端代码生成 | `make api` / `make openapi` / `make ent` 都不碰前端目录；只有 `make ts` 会（见 §3） |

唯一例外：`frontend/admin/vue-vben` 有自己的 pnpm-lock 与 node_modules（自包含单包），但它的边界在自己目录内，删它不影响别人、删别人不影响它。

## 2. 选哪一端

| 端 | 规模（页面文件 / 源码行，2026-09-25 实测） | 形态 | 门禁命令 | dev 端口 |
|---|---|---|---|---|
| react | 96 / 5.7 万（`react/src` 的 ts+tsx+css） | ProTable + DrawerForm + TanStack Query | `npm run typecheck` | 15888 |
| vue-element | 102 / 7.1 万（`vue-element/src` 的 ts+vue+样式） | ProPage 配置驱动 + vxe-table + ElForm | `npx vue-tsc --noEmit` | 15777 |
| vue-vben | 89 / 9.6 万（单包：framework 壳 3.8 万 + src 业务 5.5 万 + build-config 0.4 万） | VxeGrid + useVbenDrawer，Vben 5.x（已单包化） | `pnpm run typecheck` | 15666 |

复测（在 `frontend/admin/` 下）：
`find react/src -type f \( -name '*.ts' -o -name '*.tsx' \) | wc -l` 一类命令即可；vben 的规模要按
"你要不要连框架包一起维护"来读——留 vben 就等于收下它整个 `framework/`。

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

`backend/app/admin/service/configs/server.yaml:25-33` 的 `origins` 是逐端列的：三个演示域名 +
`localhost:15666/15777/15888`（2026-09-30 起三端 dev 端口从 5xxx 整体迁到 15xxx，旧段让给
姐妹仓，配置里也不再带顺延余量位）。删掉用不上的那些。

vite 端口被占用时会自动 +1（教程 02 第 6 节），**实际监听的端口必须进白名单**。15xxx 是本仓
专用段，姐妹仓不再来挤，但"上一个 dev server 没退干净"的自撞仍可能发生——裁剪时给保留的
那一端补上顺延位（保留 react 就补 `15889`），或真撞上时再把实际端口加进白名单。

## 4. 唯一会咬人的数据层耦合：菜单表没有"端"这个维度

机制：

- `sys_menu` 只有 `type / path / redirect / alias / name / component / meta / module` 八列（`backend/app/admin/service/internal/data/ent/schema/menu.go:32-98`，其余列来自 mixin：`parent_id` 与审计字段）。其中 `module` 是**业务模块**枚举（套餐白名单靠它过滤），**不是端标识**；`GetNavigation` 也只按角色 + 套餐白名单过滤，没有任何"哪个前端"的维度。
- 三端的「菜单同步」都写死 `mode: 'MERGE'`（`frontend/admin/react/src/api/hooks/menu.ts:201`、`frontend/admin/vue-element/src/api/composables/menu.ts:176`、`frontend/admin/vue-vben/src/api/composables/menu.ts:193`），而 MERGE 的定义就是"**数据库多出的菜单保留**"（`backend/api/protos/permission/service/v1/menu.proto:450`）。
- ⇒ 你删掉的两端此前同步进库的菜单行**不会因为删目录而消失**。

影响：

- **默认不咬**：三端 `accessMode` 默认都是 `frontend`（`frontend/admin/react/src/core/preferences/config/default.ts:5`、`frontend/admin/vue-element/src/core/preferences/config/default.ts:5`、`frontend/admin/vue-vben/.env.development:25`），侧边栏渲染的是本地路由，孤儿菜单只躺在「菜单管理」表里。
- **切到 `backend` 模式就会咬**：孤儿行会进侧边栏，点进去没有对应组件。两条处置：
  - 手工删掉孤儿行（安全）；
  - 或在菜单管理页触发 REPLACE 全量重建——**代价很大**：`menu.proto:449` 明确写了"菜单 ID 全部变化，角色-菜单授权失效"（实现见 `menu_repo.go:410` 的先 `Delete()` 再插），重建后要重新给角色授权。别顺手点。

对裁剪有利的一条性质：MERGE 按全路径匹配并**覆盖 `component`**（`menu_repo.go:483` 的 `SetNillableComponent`），
而三端同步时写的 `component` 串形状一致（都是相对 `pages/`／`views/` 的 `app/<模块>/<页>/index.vue`，
顶层布局写 `BasicLayout`；见 `react/src/api/hooks/menu.ts:150-151`、`vue-element/src/api/composables/menu.ts:112-135`、
`vue-vben/src/api/composables/menu.ts:131-160`）。所以裁剪后**让保留的那一端最后同步一次**，
库里就统一成它认识的写法。两处例外要手工处理：`IFrameView` 只有 vben 会产出（另两端认不出这个串），
以及同一路径在某一端确实没有对应页面文件时，那一行本来就是孤儿。

## 5. 文档不要改

`docs/` 里按三端口径写的表述分布很广，自己数一遍（命中数随口径浮动，别信二手数字）：

```bash
grep -roc -E 'vue-vben|vue-element|三端|15888|15777|15666' docs README*.md | grep -v ':0' | sort -t: -k2 -rn | head
```

量级参考（2026-09-25 跑上面那条命令的实际前三页结果）：
`docs/notification_domain_design.md` 68 处、`docs/design-language.md` 34 处、本文自己 29 处
（本文就是讲三端的，命中多属正常），其后 `windows-startup-guide.md` 18、`frontend_authority.md` 15、
README 三个语言版本 15/13/13、教程 02/01/04 各 11/10/8。

这些是上游口径，逐处改掉的结果是你在每次跟上游同步时都带着一堆冲突。**在 fork 里加一行说明就够了**：

> 本项目仅维护 `<你的端>` 一套前端；文档中提到的另外两端的端口、演示地址与生成目标不适用。

需要真正保持一致的只有一处：`docs/design-language.md` 是视觉权威值表，如果你改了自己这一端的颜色/圆角，改它，不用管另两端。

## 6. 机制包：下游怎么消费、怎么跟基座升级

选定一端做二开后，登录/令牌轮换/SSE/路由工厂/偏好主题这些**机制**不必从本仓拷贝——它们已抽成 npm 包（go-wind-ui 仓，changesets 自动发版）。**页面与业务永远在宿主 `src/`**，包只管"壳怎么转"。

| 端 | 依赖 | 形态 |
|---|---|---|
| react | `@go-wind/admin-react` | 常规构建包（lib + d.ts） |
| vue-element | `@go-wind/admin-core` + `@go-wind/admin-element` | 常规构建包 |
| vue-vben | `@go-wind/admin-vben` | **源码形态包**（宿主直接引其 `src/`，免 build） |

### 消费要点（按端）

**react**：`npm i @go-wind/admin-react`，bootstrap 里注入 `configureAdminRuntime({ isDev, apiBaseUrl })`，AuthGateway/路由工厂照 `frontend/admin/react/src/bootstrap.ts` 的注入清单逐项对齐。注意 `optimizeDeps.include` 需含 `'react-dom'`（包 peers 不进预构建，缺了会拿到 raw CJS 白屏）。

**vue-element**：两个包一起装；宿主合并 i18n 资源时必须保留 ns 包装层 `{ core: coreContent, ...host }`（core.json 文件名即 ns 名，平铺会丢 `core.notice.*`）；`setupI18n` 必须在 `setupRouter` **之前**；refresh 注入用纯 API 的 `refreshTokenService`（防 gateway 循环）。全清单见 `frontend/admin/vue-element/src/bootstrap.ts`。

**vue-vben**：源码形态包有三条宿主约定（缺一即断）：

1. `tsconfig.json` paths 把 `@go-wind/admin-vben/src/*` 直指 `node_modules/@go-wind/admin-vben/src/*` 子路径，且 **vue 系依赖物理映射到本仓 node_modules**（防 vue 类型双实例：RouterClassic 私有字段互斥）；
2. `vite.config.mts` 的 `optimizeDeps.exclude` 含 `@go-wind/admin-vben` 与其别名子树（被排除子树里的裸 CJS 依赖要显式 `include`）；包内代码不能用 `import.meta.glob`（vite 跳过 node_modules，glob 恒空——包内已静态枚举，宿主新增语言时在宿主侧合并）；
3. 业务热区 `framework/effects/layouts`、`framework/effects/plugins` **不在包里**，留在本仓——下游拿到的是"基座内核 + 热区源码"，改热区不跨仓、改内核走发版。

### 升级基座

```bash
pnpm update @go-wind/admin-react   # 或 admin-core / admin-element / admin-vben
```

上游发版节奏：改 go-wind-ui 对应包 → `npx changeset` → push，Release 工作流自动 publish + 打 tag。**包的 dependencies 是 publish 快照**——依赖有改必须发新版，改本地 node_modules 无效。vben 源码包升级后重跑 `typecheck`（paths 直指源码，类型错误当场暴露，这是源码形态的代价也是收益）。

## 7. 目录所有权契约与升级 runbook

> 2026-11-16 起生效。目标：二开加模块 = **纯新增文件**，merge 上游时结构性零冲突。
> 机制层（登录/令牌/SSE/路由工厂/偏好/表格）已抽包、走 §6 的 `pnpm update`，不在 git merge 路径上；
> 本章管的是仓库里剩下的部分——app 壳、演示业务页、生成产物与 locale。

### 7.1 谁拥有哪些目录

**二开拥有**（上游承诺：永不触碰、不重命名、不移动）。三端同构，命名各异：

| 内容 | react | vue-element | vue-vben |
|---|---|---|---|
| 业务页面 | `src/pages/app/<自己的模块>/` | `src/pages/app/<自己的模块>/` | `src/views/app/<自己的模块>/` |
| API hooks | `src/api/hooks/<自己的>.ts`（页面直连导入，无登记点） | `src/api/composables/<自己的>.ts` | 同 element |
| 页面文案 | `src/locales/<lang>/_modules/<自己的>.json` | `src/locales/<lang>/pages/<自己的>.json` | `src/locales/langs/<lang>/page/<自己的模块>.json` |
| 路由 | `src/router/modules/<自己的域>.tsx` | `src/router/routes/modules/app/<自己的域>.ts` | `src/router/routes/modules/app/<自己的域>.ts` |
| 路由标题 | `src/locales/<lang>/_core/routes/<自己的域>.json` | `src/locales/<lang>/routes/<自己的域>.json` | `src/locales/langs/<lang>/menu/<自己的域>.json` |

路由与 i18n 全部 `import.meta.glob` 自动注册；菜单同步由路由树自动推导，**不存在**菜单清单登记文件。

**上游拥有**（二开不碰；上游重构会小步走）：`bootstrap/app/main`、`layouts/`、偏好配置（react `src/core`、vben `preferences.ts`）、`router` 入口与 guards、`api/client.ts`、`api/generated/`（重生成产物）、locale 框架级文件（react `_core/common` 等、element `app/common/enum/preferences/validation.json`、vben `enum/ui.json`）、`components/`、`stores/`、`utils/`。

**残留缝**（双方都可能编辑的共享文件，全仓仅剩一处）：

- vue-element 的 `src/api/composables/index.ts`、vue-vben 的 `src/api/composables/index.ts`（+ `src/api/index.ts` 转出）：加模块追加一行 `export * from './<entity>';`。**追加式**——冲突是行级追加，机械解决（两边都保留）。react 端 barrel 已删（2026-11-16），无缝。

### 7.2 定制上游页：复制再改

上游演示页（如 `system/user`）是**参照实现**。要定制就复制到自己命名空间再改（如 `pages/app/my-user/`）；上游页保持可干净覆盖。直接改上游页的后果：上游每次修这页你都冲突，且上游修复要你人工 port。删掉上游演示页也可以，但上游改动它们时 merge 会出现 modify/delete 冲突，按 7.3 的第三条统一处理。

### 7.3 git merge 升级 runbook

```bash
git fetch upstream && git merge upstream/main
```

冲突三定律：

1. **lockfile 不手解**：`package.json` 手解后，删掉 `package-lock.json`/`pnpm-lock.yaml` 重装重生成——lockfile 冲突手解必错；
2. **`api/generated/` 不手解**：解完 proto 相关冲突后跑 `make ts` 重生成（确定性产物，重生成即消除冲突）；
3. **modify/delete 冲突**（你删掉的上游演示页被上游改了）：统一 keep deleted（`git rm`）。

然后：

```bash
cd frontend/admin/<端> && npm install        # 或 pnpm install，重建 lockfile
cd backend && make ts                        # 裁剪过的仓只跑保留端那条 buf 模板（见 §3 第 2 步）
# 跑你那端门禁（命令见根 AGENTS.md 三端门禁表）
```

机制层升级不走 merge，走 §6 的 `pnpm update @go-wind/admin-*`。**节奏建议**：出现 `feat(frontend)!:` 这类 breaking 提交尽快跟；平时每两周一次——长周期不 merge 是冲突的最大来源，比任何文件布局都致命。

### 7.4 上游纪律（给上游维护者）

- 机制改动只走 go-wind-ui 包发版，不在三端 `src/` 里直接改机制代码；
- 新增演示模块遵守"一模块一文件"（路由/文案/hooks 各自成文件），不新设集中式登记点；
- 残留缝文件（element/vben `composables/index.ts`）只做行追加；要重命名/移动**二开拥有**的路径，先在文档公告（这是给下游的 breaking change）；
- locale 组装的斜杠键深合并语义（element/vben 宿主 `locales/index.ts` 的 `denormalizeSlashKeys`、react 的 `routes/*.json` 合并）是宿主契约，改动须同步本文档与各端 AGENTS.md。

## 深读

- [tutorial/02 从零跑起来](./tutorial/02-get-it-running.md) —— 三端各自的启动命令与端口
- [frontend_authority.md](./frontend_authority.md) —— 路由 `frontend` / `backend` 双模式的三端实现对照
- [README.md](../README.md) 的「为什么是三套前端」（`README.md:28`） —— 三选一的能力声明
