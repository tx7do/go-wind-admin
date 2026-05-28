# CLAUDE.md — GoWind Admin 项目指令

本文件为 Claude Code 提供项目级开发规范和约定。所有代码生成和修改必须遵循以下规则。

## 项目概述

**go-wind-vue3-element-admin** — 基于 Vue 3 + Vite + TypeScript + Element Plus 的后台管理模板（Vue3 版 vue-element-admin）。

### 技术栈

Vue 3.5 / TypeScript 5.9 / Vite 8 / Element Plus 2.x / vxe-table 4.x / Pinia 3 / @tanstack/vue-query 5 / vue-router 5 / vue-i18n 11 / UnoCSS + SCSS / vee-validate + zod / axios（封装 gRPC-Web 风格 API）

### 核心目录结构

```
src/
├── api/
│   ├── generated/          # gRPC 自动生成代码（禁止手动修改）
│   ├── service/            # 服务层：封装 gRPC 客户端调用
│   └── composables/        # Composable 层：Vue Query hooks + 枚举工具函数
├── components/Pro/         # Pro 组件库（配置化 CRUD 页面）
├── core/
│   ├── transport/rest/     # 请求客户端、PaginationQuery、拦截器
│   ├── i18n/               # 国际化核心
│   ├── router/             # 路由工具函数
│   └── access/             # 权限控制
├── pages/app/              # 业务页面（按模块分目录）
├── locales/                # 翻译资源（zh-CN / en-US）
├── router/routes/modules/  # 动态路由模块
├── layouts/                # 布局组件
└── stores/                 # Pinia stores
```

---

## 编码规范

### TypeScript
- 接口命名**不使用 I 前缀**（`User` 而非 `IUser`）
- 路由类型使用 `RouteRecordRaw`（非 `AppRoute`）
- 路由数组 map+sort 后需类型断言为 `RouteRecordRaw[]`
- 不使用 `isFunction` 做类型窄化，用 `typeof fn === "function"`
- `defineExpose` 暴露的属性直接访问，无需 `.value`

### 国际化（强制）
- 所有用户可见文本必须通过 `$t()` / `t()` 国际化，**禁止硬编码中文**
- `$t()` 用于模板和 `computed`（响应式场景）
- `t()` 用于 composable 顶层（非响应式场景）
- 翻译 key 命名：`pages.<module>.<field>` / `enum.<module>.<field>.<VALUE>` / `routes.<module>.<page>` / `common.<category>.<key>`

### API 三层架构

严格遵循分层：`generated/` → `service/` → `composables/`

#### 服务层 (`src/api/service/`)
- 使用单例模式创建 gRPC 服务客户端
- 列表查询使用 `PaginationQuery.toRawParams()` 并清除无用字段（`sorting`, `offset`, `limit`, `token`, `filter`, `filterExpr`）
- 在 `src/api/service/index.ts` 中添加 `export *`

#### Composable 层 (`src/api/composables/`)
- **禁止直接依赖 gRPC 生成代码的实现细节**，只导入类型
- 每个 composable 文件导出：`use*` Hook + `fetch*` 非 Hook 函数 + 枚举工具
- queryKey 格式：`["操作名", 参数]`，全局唯一
- **创建 mutation 的参数必须用 `{ data: {...} }` 包裹**（gRPC 约定）
- **更新 mutation 必须使用 `makeUpdateMask` 生成字段掩码**
- 枚举列表用 `computed(() => [...])` + i18n `t()` 标签
- 在 `src/api/composables/index.ts` 中添加 `export *`

### 组件规范
- `ElDrawer` 必须设置 `:append-to-body="true"`
- `ElDialog` 的 `appendTo` 属性是字符串选择器（非布尔值）
- `ElTreeSelect` 的 `value` 不接受 `undefined`，用 `null` 代替
- 暗黑模式下文本颜色使用 `var(--el-text-color-*)` CSS 变量，避免硬编码

### 样式规范
- CSS 变量使用 `--gowind-*` 前缀，避免与 Element Plus `--el-*` 冲突
- 主题色变量存储为 HSL 数值（非 hex 字符串）
- 更新 CSS 变量使用 `style.setProperty()`

### 路由
- 动态路由放在 `src/router/routes/modules/app/<module>.ts`
- 顶层路由使用 `Layout` 组件包裹
- `meta.title` 使用 i18n key
- `meta.icon` 使用 `lucide:` 前缀
- `meta.authority` 控制权限
- 文件导出 `default` 路由数组

---

## 新建 CRUD 模块清单

创建新业务模块时，按以下顺序生成文件：

1. **`src/api/service/<module>.ts`** — gRPC 服务客户端封装
2. **`src/api/composables/<module>.ts`** — Vue Query hooks + 枚举工具
3. **`src/api/service/index.ts`** — 添加 `export *`
4. **`src/api/composables/index.ts`** — 添加 `export *`
5. **`src/locales/zh-CN/pages/<module>.json`** — 中文翻译
6. **`src/locales/en-US/pages/<module>.json`** — 英文翻译
7. **`src/locales/zh-CN/enum.json`** — 追加枚举翻译
8. **`src/locales/zh-CN/routes.json`** — 追加路由标题
9. **`src/router/routes/modules/app/<module>.ts`** — 路由配置
10. **`src/pages/app/<module>/<module>/index.vue`** — 列表页（ProPage 配置）
11. **`src/pages/app/<module>/<module>/<module>-drawer.vue`** — 弹窗组件（useProModal 模式）

---

## 常见陷阱

| 陷阱 | 正确做法 |
|------|----------|
| gRPC 创建接口直接传对象 | 必须用 `{ data: {...} }` 包裹 |
| `isFunction` 做类型窄化 | 使用 `typeof fn === "function"` |
| PowerShell 中用 `&&` 连接命令 | 使用分号 `;` |
| `ElLink` underline 传 boolean | 已废弃 boolean，使用字符串 |
| vue-i18n `$te` 传 ns 选项对象 | `$te` 不支持 ns 选项，用完整 key |
| `ElTreeSelect` value 传 undefined | 使用 `null` 代替 |

---

## 构建命令

```bash
pnpm dev              # 启动开发服务器
pnpm build            # 类型检查 + 生产构建
pnpm build-only       # 仅构建（不检查类型）
pnpm type-check       # TypeScript 类型检查
pnpm lint             # ESLint + Prettier + Stylelint
pnpm commit           # Git 提交（cz-git 交互式）
```

- Node 版本：`^20.19.0 || >=22.12.0`
- 包管理器：仅 pnpm（preinstall 强制检查）
- Git 提交：Conventional Commits 规范
