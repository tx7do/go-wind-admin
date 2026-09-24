# AGENTS.md - GoWind React Admin 脚手架开发指南

> 本文件面向二开人员，帮助 AI 助手理解项目架构并遵循开发规范。

## 项目概述

GoWind React Admin 是基于 React 19 的企业级后台管理脚手架，采用 Ant Design v6 + ProComponents 作为 UI 框架。

## 技术栈

- **框架**: React 19 + TypeScript 6
- **UI 库**: Ant Design v6 + ProComponents 2
- **构建**: Vite 8 (SWC 编译)
- **状态管理**: Zustand 5 (persist 中间件) + TanStack React Query 5 (数据请求)
- **路由**: React Router v6
- **国际化**: i18next + react-i18next
- **样式**: Less + Tailwind v4
- **图标**: Iconify (lucide 图标集)
- **包管理**: pnpm 或 npm 均可 —— 本端 `package.json` 未声明 `packageManager`，也没有 `only-allow` 守卫（另两端强制 pnpm）；仓库三端门禁统一跑 `npm run typecheck`，用 pnpm 时对应 `pnpm typecheck`。

## 目录结构

```
src/
├── api/                    # API 层（两层架构）
│   ├── generated/          # 自动生成代码（禁止手动修改）
│   ├── client.ts           # apiClient 单例（懒加载各 Service）
│   └── hooks/              # Hooks 层 - React Query 集成
├── core/                   # 核心模块
│   ├── access/             # 权限控制
│   ├── i18n/               # 国际化
│   ├── preferences/        # 偏好设置
│   ├── router/             # 路由工厂
│   ├── storage/            # 存储工具
│   └── transport/          # HTTP 传输层
├── hooks/                  # 业务 Hooks
├── layouts/                # 布局组件（MainLayout/BlankLayout/UserLayout/IFrameLayout）
├── locales/                # 翻译资源
│   ├── zh-CN/              # 中文（_core/ + _modules/）
│   └── en-US/              # 英文
├── pages/                  # 页面组件
│   ├── app/                # 业务页面
│   └── core/               # 系统页面（错误页等）
├── router/                 # 路由配置
│   ├── config/             # 静态/错误/认证路由
│   ├── guards/             # 路由守卫
│   └── modules/            # 业务路由模块（自动导入）
├── stores/                 # Zustand Stores
├── styles/                 # 全局样式
└── utils/                  # 工具函数
```

## API 两层架构

```
Generated (自动生成类型和 Service Client) → Hooks (通过 apiClient 直调，React Query 集成)
```

`apiClient`（`src/api/client.ts`）是单例，以懒加载 getter 聚合所有 Service Client。Hooks 层直接通过 `apiClient.xxxService.Method()` 调用。

### Hooks 层模板 (`src/api/hooks/*.ts`)

```typescript
import { useMutation, useQuery, type UseMutationOptions, type UseQueryOptions } from '@tanstack/react-query';
import { apiClient } from '@/api/client';
import { type PaginationQuery, queryClient } from '@/core';

// React Hook（组件中使用）
export function useListXxx(query: PaginationQuery, options?: UseQueryOptions<...>) {
  return useQuery({
    queryKey: ['listXxx', query],
    queryFn: () => apiClient.xxxService.List(query.toRawParams()),
    ...options,
  });
}

// Fetch 方法（Store/工具函数/路由守卫中使用）
export async function fetchListXxx(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listXxx', params],
    queryFn: () => apiClient.xxxService.List(params.toRawParams()),
    retry: 0,
  });
}

// Mutation 示例
export function useCreateXxx(options?: UseMutationOptions<...>) {
  return useMutation({ mutationFn: (data) => apiClient.xxxService.Create(data), ...options });
}
```

### 使用场景

| 场景 | 方式 | 示例 |
|------|------|------|
| React 组件 | `useXxx()` | `const { data, isLoading } = useListUsers(new PaginationQuery({ paging: { page: 1, pageSize: 20 } }))` |
| React 组件（写操作） | `useXxx().mutateAsync` | `const { mutateAsync: createUser } = useCreateUser(); await createUser({ data: values });` |
| Zustand Store | `fetchXxx()` | `await fetchUser(id)` |
| 路由守卫 | `fetchXxx()` | `await fetchNavigation()` |

**命名规范**: Hooks 层 `useListXxx` / `useGetXxx` + `fetchListXxx` / `fetchXxx`。

**注意**: `useListXxx` / `useGetXxx` 返回的是 `useQuery` 结果（`data` / `isLoading`），没有 `mutateAsync`；`mutateAsync` 只属于 `useCreateXxx` / `useUpdateXxx` / `useDeleteXxx`。

## 路由系统

### 路由类型

| 路由 | 位置 | 说明 |
|------|------|------|
| 静态路由 | `router/config/static.tsx` | 主布局 + 根路由 |
| 认证路由 | `router/config/auth.tsx` | 登录/注册页 |
| 错误路由 | `router/config/error-routes.tsx` | 403/404/500 |
| 业务路由 | `router/modules/*.tsx` | 自动被 `import.meta.glob` 导入 |

### 路由配置模板

```tsx
import type { AppRouteObject } from '@/core/router/types';
import { createLazyRoute } from '@/core/router';

export const myModuleRoutes: AppRouteObject[] = [
  {
    name: 'my-module',
    path: 'my-module',
    meta: {
      title: 'routes:myModule',          // i18n 翻译键（routes 命名空间）
      icon: 'lucide:some-icon',          // Iconify 图标名
      order: 10,                         // 菜单排序
      authority: ['sys:my_module:view'], // 权限码
    },
    children: [
      {
        name: 'my-module-list',
        path: 'list',
        element: createLazyRoute(() => import('@/pages/app/my-module')),
        meta: { title: 'routes:myModuleList' },
      },
    ],
  },
];
export default myModuleRoutes;
```

### Route Meta 关键字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `title` | `string` | `'routes:xxx'` 格式的 i18n 翻译键 |
| `icon` | `string` | Iconify 图标名，如 `'lucide:layout-dashboard'` |
| `order` | `number` | 菜单排序 |
| `authority` | `string[]` | 角色码和权限码混合数组 |
| `hideInMenu` | `boolean` | 是否隐藏菜单项 |
| `hideInTab` | `boolean` | 是否隐藏标签页 |
| `keepAlive` | `boolean` | 是否缓存页面 |

### 权限模式

- `frontend`（默认）：前端路由 + `meta.authority` 过滤
- `backend`：后端返回菜单 + `pageMap` 动态匹配组件

## 权限系统

### 数据来源（分离存储）

- 角色码 → `useUserStore.userRoles`（来自 `userInfo.roles`）
- 权限码 → `useUserStore.accessCodes`（来自 `GetMyPermissionCode`）
- `meta.authority` → 角色码和权限码的**混合数组**

### 三种鉴权方式

```tsx
// 1. useAccess Hook（推荐，用于条件渲染）
import { useAccess } from '@/core/access';
const { hasAccessByCodes, hasAccessByRoles } = useAccess();
{hasAccessByCodes(['sys:user:create']) && <Button>新建</Button>}

// 2. AccessControl 组件
import { AccessControl } from '@/core/access';
<AccessControl codes={['sys:user:create']} type="code">
  <Button>新建</Button>
</AccessControl>

// 3. 非组件场景（路由生成、工具函数）
import { getAccessStatic } from '@/core/access';
const { hasAccessByCodes } = getAccessStatic();
```

### 权限码格式

`模块:资源:操作`，如 `sys:user:create`、`sys:role:update`。

### 超级管理员

拥有 `*:*:*` 角色的用户自动通过所有权限检查。

## 状态管理

### Store 列表

| Store | 文件 | 用途 | 持久化 |
|-------|------|------|--------|
| `useAuthStore` | `stores/auth.ts` | Token、登录/登出、注册 | token 持久化 |
| `useUserStore` | `stores/user.ts` | 用户信息、角色码、权限码 | userInfo 持久化 |
| `usePreferencesStore` | `core/preferences/store/` | 偏好设置 | 全部持久化到 app-preferences |

### 使用方式

```typescript
// React 组件中 — 使用 selector 精确订阅
const token = useAuthStore((s) => s.accessToken);
const isDark = usePreferencesStore((s) => s.preferences.theme.mode === 'dark');

// 非组件环境 — 使用 getState()
const token = useAuthStore.getState().accessToken;
const locale = usePreferencesStore.getState().preferences.app.locale;
```

## 国际化 (i18n)

### 命名空间

| 类别 | 目录 | 命名空间 |
|------|------|----------|
| 核心 | `_core/` | `common`, `auth`, `routes`, `editor` |
| 业务 | `_modules/` | 文件名即为命名空间（如 `user`, `role`） |

### 使用方式

```tsx
import { useI18n } from '@/core/i18n';

// 指定命名空间（推荐）
const { t } = useI18n('user');
t('username');  // 查找 user 命名空间

// 多命名空间
const { t } = useI18n(['user-detail', 'user']);

// 非组件环境
import i18n from 'i18next';
i18n.t('key', { ns: 'common' });
```

### 新增翻译

在 `src/locales/zh-CN/_modules/` 和 `src/locales/en-US/_modules/` 创建同名 JSON 文件，自动被 `import.meta.glob` 收集。

### 翻译键规则

- 插值用 `{{var}}`，**不是** `#{var}` 或 `${var}`
- 路由标题用 `'routes:xxx'` 格式
- 硬编码文本必须提取到翻译文件

## 代码风格

- **Prettier**: 单引号、分号、尾逗号 `all`、行宽 100、2 空格缩进、LF 换行
- **路径别名**: `@/` → `src/`，`#/` → `types/`
- **ESLint**: TypeScript 严格模式，React Hooks 规则强制
- **提交规范**: Conventional Commits（Husky + commitlint）

## 关键注意事项

1. **PaginationQuery 必须用 new**: `new PaginationQuery({ paging: { page, pageSize }, formValues })`（构造参数是 `paging` 对象，直接传 `{ page, pageSize }` 不会分页）
2. **非组件环境禁用 useXxx Hook**: Store/路由守卫/工具函数中只能用 `fetchXxx()` 或 `apiClient` 直调
3. **国际化插值**: `{{var}}` 而非 `#{var}`
4. **meta.title 格式**: `'routes:xxx'`
5. **权限码格式**: `模块:资源:操作`
6. **禁止修改 generated 目录**: API 类型由工具自动生成
7. **DrawerForm 用 formRef**: 没有 `useForm` 方法
8. **antd v6 变更**: 用 `items` 替代 `TabPane`，Alert 用 `title` 替代 `message`
9. **ProTable scroll.y**: 初始值必须是像素值（数字），不能是百分比
10. **角色码和权限码分离存储**: `userRoles` + `accessCodes`，不混合
11. **不要使用 `userInfo?.permissions`**: 该字段不存在
12. **错误处理铁律（不吞错）**: 任何 catch 至少二选一——`console.error/warn` 带出**原始错误对象**，或重新抛出。用户可见的 message/notification 不等于日志（只有翻译文案，排查靠控制台原始错误）。合法裸 catch 仅限纯本地 best-effort 兜底且注释写明原因。历史教训：静默吞错曾让认证链路 bug 排查数日（每层都被上一层话术掩盖）

## 新增完整功能模块清单

当需要新增一个完整的业务模块时，按以下顺序操作：

1. **翻译文件**: `src/locales/zh-CN/_modules/xxx.json` + `src/locales/en-US/_modules/xxx.json`
2. **页面组件**: `src/pages/app/xxx/index.tsx`
3. **API Hooks**: `src/api/hooks/xxx.ts`（通过 `apiClient` 直调）+ 在 `src/api/hooks/index.ts` 导出
4. **路由**: `src/router/modules/xxx.tsx`（自动导入）
5. **权限控制（可选）**: 配置 `meta.authority` 并在页面中使用 `useAccess()`
