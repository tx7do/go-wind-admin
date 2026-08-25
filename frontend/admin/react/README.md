# GoWind React Admin

基于 React 19 + Ant Design 6 的后台管理系统前端，提供权限管理、组织架构、审计日志、国际化、主题切换等开箱即用的企业级能力。

## 技术栈

| 类别 | 技术 | 说明 |
|------|------|------|
| 构建工具 | Vite 8 + @vitejs/plugin-react | 快速 HMR，ESM 原生构建 |
| 核心框架 | React 19 + TypeScript 6 | 严格模式，路径别名 `@/` |
| UI 组件 | Ant Design 6 + ProComponents | 主题 Token，暗色模式 |
| 路由 | react-router-dom v6 | 前端/后端双模式动态路由 + 权限守卫 |
| 状态管理 | Zustand 5 | 轻量，支持 persist 中间件 |
| 数据请求 | Axios + TanStack Query 5 | 统一拦截封装，接口缓存/自动状态管理 |
| 国际化 | i18next + react-i18next | 命名空间分离，静态全量预加载 |
| 样式 | UnoCSS + Less + AntD Token | 原子化 CSS + 主题变量 |
| 图表 | ECharts 6 + echarts-for-react | 数据可视化 |
| 富文本 | Tiptap 3 | 可扩展的富文本编辑器 |
| 工程化 | ESLint + Prettier + Husky + Commitlint | 代码质量 + Git 提交规范 |

---

## 快速开始

### 环境要求

- Node.js >= 18
- pnpm（项目使用 pnpm 管理）

### 安装与运行

```bash
# 安装依赖
pnpm install

# 启动开发服务器（默认端口 7000）
pnpm dev

# 类型检查
pnpm typecheck

# 代码检查
pnpm lint

# 构建生产包
pnpm build

# 预览生产包
pnpm preview
```

### 环境变量

| 文件 | 用途 |
|------|------|
| `.env` | 公共配置（应用标题、命名空间、AES 密钥等） |
| `.env.development` | 开发环境（API 地址 `localhost:7788`，代理配置） |
| `.env.production` | 生产环境（API 地址 `api.demo.admin.gowind.cloud`） |

关键变量：

| 变量 | 说明 | 示例 |
|------|------|------|
| `VITE_API_URL` | API 基础地址 | `http://localhost:7788` |
| `VITE_SSE_URL` | SSE 推送地址 | `http://localhost:7789/events` |
| `VITE_SERVER_PORT` | 开发服务器端口 | `5888` |
| `VITE_PROXY` | 开发代理配置 | `[["/admin", "http://127.0.0.1:7788/"]]` |
| `VITE_APP_TITLE` | 应用标题 | `GoWind Admin` |
| `VITE_MOCK` | 是否启用 Mock | `false` |

---

## 项目结构

```
src/
├── api/                        # API 层
│   ├── generated/              # 后端 Proto 生成的类型定义
│   ├── service/                # 服务函数（对 axios 的直接封装）
│   ├── hooks/                  # React Query Hooks（组件内使用）
│   └── index.ts
│
├── core/                       # 核心模块（通用、可复用）
│   ├── access/                 # 权限控制（useAccess / AccessControl）
│   ├── i18n/                   # 国际化（useI18n / 语言检测 / 格式化工具）
│   ├── preferences/            # 偏好设置（主题/布局/语言/存储到 localStorage）
│   ├── router/                 # 路由引擎（路由生成 / 菜单生成 / 类型定义）
│   ├── storage/                # 增强存储（TTL / 驱逐策略 / 批量操作 / 跨 Tab 同步）
│   └── transport/              # 网络传输
│       ├── rest/               # REST 客户端（Axios 封装 + 拦截器）
│       └── sse/                # SSE 客户端（服务端推送）
│
├── router/                     # 业务路由配置
│   ├── config/                 # 静态路由 / 认证路由 / 错误路由
│   ├── guards/                 # AuthGuard / GuestGuard
│   ├── modules/                # 业务路由模块（自动导入）
│   └── index.tsx               # AppRouter 入口
│
├── stores/                     # 全局状态（Zustand）
│   ├── auth.ts                 # 认证状态（token / refreshToken / forceLogout）
│   ├── user.ts                 # 用户信息（userInfo / roles / permissions）
│   ├── tabs.ts                 # 标签页状态
│   └── pageRefresh.ts          # 页面刷新控制
│
├── hooks/                      # 业务 Hooks
│   ├── useAuth.ts              # 认证统一入口（登录/登出/权限码获取）
│   ├── useDictCache.ts         # 字典缓存
│   ├── useProTableScrollY.ts   # ProTable 动态高度
│   ├── useTokenRefresh.ts      # Token 定时刷新
│   └── ...
│
├── locales/                    # 翻译资源
│   ├── zh-CN/                  # 中文（_core/ 核心 + _modules/ 业务模块）
│   └── en-US/                  # 英文（结构同上）
│
├── layouts/                    # 布局组件
│   ├── MainLayout/             # 主布局（侧边栏 + 顶栏 + 标签页 + 内容区）
│   ├── UserLayout/             # 用户布局（登录/注册）
│   ├── BlankLayout/            # 空白布局（错误页）
│   └── IFrameLayout/           # iframe 内嵌布局
│
├── pages/                      # 页面组件
│   ├── app/                    # 业务页面
│   │   ├── dashboard/          # 仪表盘
│   │   ├── permission/         # 权限管理（角色/权限点/权限组）
│   │   ├── opm/                # 组织人员（组织架构/职位/用户）
│   │   ├── tenant/             # 租户管理
│   │   ├── internal-message/   # 内部消息
│   │   ├── log/                # 审计日志（登录/API/操作/数据访问/权限）
│   │   └── system/             # 系统管理（菜单/API/字典/文件/任务/策略/语言）
│   └── core/                   # 核心页面
│       ├── auth/               # 登录 / 注册
│       └── error/              # 401 / 403 / 500 / 404 等
│
├── components/                 # 公共组件
│   ├── bussiness/AuthLayout/   # 认证页布局组件
│   └── common/                 # 通用组件（Editor / Loading / PageContainer）
│
├── utils/                      # 工具函数（树操作 / 颜色 / 日期 / 加密等）
├── styles/                     # 全局样式（CSS 覆盖 / 暗色模式 / 骨架屏）
├── config/constants.ts         # 全局常量
├── App.tsx                     # 根组件
├── main.tsx                    # 入口（bootstrap → render）
└── bootstrap.ts                # 初始化（i18n + RequestClient 配置）
```

---

## 核心模块文档

每个核心模块都有详细的 README，涵盖 API、使用方法和注意事项：

| 模块 | 文档 | 说明 |
|------|------|------|
| 权限控制 | [core/access/README.md](src/core/access/README.md) | useAccess Hook、AccessControl 组件、静态鉴权 |
| 路由系统 | [core/router/README.md](src/core/router/README.md) | 前端/后端路由模式、RouteMeta、菜单生成、懒加载 |
| 国际化 | [core/i18n/README.md](src/core/i18n/README.md) | 命名空间规范、useI18n、格式化工具、翻译键约定 |
| 偏好设置 | [core/preferences/README.md](src/core/preferences/README.md) | 主题/布局配置、ThemeProvider、持久化机制 |
| 存储管理 | [core/storage/README.md](src/core/storage/README.md) | TTL 过期、缓存驱逐、批量操作、多标签页同步 |

---

## 开发规范

### 路径别名

| 别名 | 指向 | 用途 |
|------|------|------|
| `@/` | `src/` | 业务代码导入 |
| `#/` | `types/` | 全局类型定义 |

```typescript
import { useAuth } from '@/hooks/useAuth';
import type { AppRouteObject } from '@/core/router';
```

### 新增业务页面流程

1. **创建页面组件**：`src/pages/app/<module>/<page>/index.tsx`
2. **添加路由配置**：在 `src/router/modules/` 下对应文件中添加路由（使用 `createLazyRoute` 懒加载）
3. **添加翻译文件**：在 `src/locales/zh-CN/_modules/` 和 `en-US/_modules/` 下添加对应 JSON
4. **添加 API Hook**（如需）：在 `src/api/hooks/` 和 `src/api/service/` 下封装

### 代码提交规范

项目使用 Husky + Commitlint 强制 Git 提交规范：

```bash
# 提交信息格式
<type>(<scope>): <subject>

# 示例
feat(user): 新增用户管理页面
fix(auth): 修复 token 过期后重定向丢失问题
docs(router): 补充路由模块文档
refactor(storage): 重构缓存驱逐策略
```

**Pre-commit** 自动执行 `lint-staged`（ESLint + Prettier）。

### 命名约定

| 类型 | 规范 | 示例 |
|------|------|------|
| 组件文件 | PascalCase | `UserDrawer.tsx` |
| 工具/Hook 文件 | camelCase | `useDictCache.ts` |
| 路由模块文件 | kebab-case | `internal-message.tsx` |
| 翻译 JSON | kebab-case | `login-audit-log.json` |
| Store 文件 | camelCase | `auth.ts` |
| 样式文件 | kebab-case / camelCase | `AuthLayout.style.less` |
| i18n 命名空间 | JSON 文件名 | `user`、`permission`、`routes` |

---

## 关键架构

### 应用启动流程

```
main.tsx
  → bootstrap()
    → initI18n(locale)              // 初始化 i18next
    → RequestClient.init(...)       // 配置 Axios（getToken / refreshToken / 错误处理）
  → render()
    → QueryClientProvider           // TanStack Query
    → ThemeProvider                 // AntD 主题 + 暗色模式 + 水印 + 国际化
    → AntdApp                       // AntD 上下文（message / notification / modal）
    → App
      → useLocaleSync()            // 偏好设置 ↔ i18n 双向同步
      → AppRouter
        → useEffect(isAuthenticated)
          → getUserPermissionCodes() // 获取权限码
          → fetchAllDictEntries()    // 预加载字典
          → createAccessibleRouter() // 根据模式生成路由 + 菜单
        → <RouterProvider />        // 渲染路由
```

### 认证流程

- **登录**：`useAuth().login()` → 存储 token → 路由守卫放行 → 跳转原页面
- **Token 失效**：401 响应 → `forceLogout()` → 清除 token → 跳转登录页（携带 redirect）
- **Token 刷新**：`useTokenRefresh` Hook 定时检查，过期前自动调用 refreshToken
- **登出**：`useAuth().logout()` → 调用后端 /logout → 清除状态 → 跳转登录页

### 权限体系

- **权限码**：`GET /admin/v1/perm-codes` → `useUserStore.accessCodes`
- **角色码**：`GET /admin/v1/me` → `useUserStore.userRoles`
- **路由权限**：`meta.authority` 数组（角色码 + 权限码混合）
- **鉴权方式**：`useAccess()` Hook（条件渲染）、`<AccessControl>` 组件、`getAccessStatic()` 静态方法
- **超级管理员**：拥有 `*:*:*` 角色自动通过所有权限检查

### API 调用模式

```typescript
// 1. Service 层：封装 Axios 请求（src/api/service/）
export const listUsers = (params: PaginationQuery) =>
  requestClient.get('/admin/v1/users', { params });

// 2. Hook 层：封装 React Query（src/api/hooks/）
export function useListUsers(query: PaginationQuery) {
  return useQuery({
    queryKey: ['listUsers', query],
    queryFn: () => listUsers(query),
  });
}

// 3. 页面中使用
const { data, isLoading } = useListUsers({ page: 1, pageSize: 20 });
```

### 路由模式

| 模式 | 偏好设置 | 路由来源 | 说明 |
|------|----------|----------|------|
| 前端模式 | `accessMode: 'frontend'` | `src/router/modules/` | 静态定义 + 权限过滤（默认） |
| 后端模式 | `accessMode: 'backend'` | `GET /admin/v1/routes` | API 返回路由树 + 组件映射 |

通过偏好设置面板切换（通用 → 权限模式），刷新后生效。

### 状态管理

| Store | 文件 | 持久化 | 说明 |
|-------|------|--------|------|
| `useAuthStore` | `stores/auth.ts` | accessToken 持久化 | 认证状态、token、登录登出 |
| `useUserStore` | `stores/user.ts` | 不持久化 | 用户信息、角色码、权限码 |
| `useTabsStore` | `stores/tabs.ts` | 标签页数据持久化 | 标签页管理 |
| `usePreferencesStore` | `core/preferences/store/` | preferences 持久化 | 偏好设置 |
| `usePageRefreshStore` | `stores/pageRefresh.ts` | 不持久化 | 页面刷新控制 |

---

## 功能模块

### 已实现功能

- **仪表盘**：数据概览、图表展示
- **权限管理**：角色管理、权限点管理、权限组管理
- **组织人员**：组织架构管理、职位管理、用户管理（含用户详情）
- **租户管理**：多租户隔离、租户成员管理
- **内部消息**：消息列表、消息分类、收件箱
- **日志审计**：登录审计、API 审计、操作审计、数据访问审计、权限审计
- **系统管理**：菜单管理、API 管理、字典管理、文件管理、任务管理、登录策略、语言管理
- **个人中心**：用户资料编辑
- **偏好设置**：主题色 / 暗色模式 / 布局方式 / 侧边栏 / 标签页 / 快捷键
- **认证**：登录、注册、Token 刷新、登录过期处理

---

## 常见问题

### Q: 开发环境启动后接口 404？

检查 `.env.development` 中的 `VITE_PROXY` 配置是否指向正确的后端地址，确保后端服务已在 `7788` 端口启动。

### Q: 修改代码后页面没有更新？

Vite 的 HMR 通常能自动刷新。如果遇到状态残留，尝试手动刷新页面。路由级别改动（如新增路由模块）可能需要重启开发服务器。

### Q: 如何切换暗色/亮色模式？

点击顶栏右侧的主题切换图标，或打开偏好设置面板（顶栏齿轮图标）→ 外观 → 主题模式。支持 `light` / `dark` / `auto`（跟随系统）。

### Q: 如何新增一个后端接口的 API 调用？

1. 在 `src/api/service/` 下创建 service 函数（使用 `requestClient`）
2. 在 `src/api/hooks/` 下创建对应的 React Query Hook
3. 在页面组件中 import Hook 使用

### Q: 生产构建如何分析包体积？

设置 `.env` 中 `VITE_ANALYZE=true`，然后执行 `pnpm build`，构建完成后会自动生成可视化分析报告。

---

## 相关链接

- [Vite 文档](https://vitejs.dev/)
- [React 文档](https://react.dev/)
- [Ant Design 6](https://ant.design/)
- [TanStack Query](https://tanstack.com/query/)
- [Zustand](https://zustand.docs.pmnd.rs/)
- [i18next](https://www.i18next.com/)
