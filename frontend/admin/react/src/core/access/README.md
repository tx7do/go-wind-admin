# 权限管理模块 (Access Control) - React

## 概述

本模块提供了完整的 React 权限管理解决方案，对齐 Vue 版 `core/access`，支持**前端权限控制**和**后端权限控制**两种模式。

### 核心特性

- **双模式支持**：前端路由权限 / 后端动态路由
- **多维度权限**：角色权限 + 权限码权限
- **多层级控制**：路由级别 + 菜单级别 + 页面级别 + 按钮级别
- **三种使用方式**：Hook（条件渲染）、组件（`<AccessControl>`）、静态函数（非组件场景）
- **超级管理员**：拥有 `*:*:*` 角色的用户自动通过所有检查

---

## 模块结构

```
src/core/access/
├── index.ts               # 统一导出
├── constants.ts           # 常量（ROLE_ROOT）
├── use-access.ts          # useAccess Hook + getAccessStatic
├── access-control.tsx     # <AccessControl> 组件
└── README.md              # 本文档
```

---

## 权限数据来源

| 来源 | API | 存储位置 | 说明 |
|------|-----|----------|------|
| 角色码 | `GET /admin/v1/me` → `userInfo.roles` | `useUserStore.userRoles` | 用户的角色列表 |
| 权限码 | `GET /admin/v1/perm-codes` → `codes` | `useUserStore.accessCodes` | 用户的权限码列表 |
| UI 绑定 | `GET /admin/v1/routes` → `meta.authority` | `RouteMeta.authority` | 路由/菜单的权限要求 |

> **注意**：`meta.authority` 是角色码和权限码的**混合数组**，鉴权时同时匹配两个来源。

---

## 使用方法

### 1. 路由级别权限控制

在路由 `meta.authority` 中配置权限要求：

```typescript
// src/router/modules/permissions.tsx
{
  name: 'permission',
  path: 'permission',
  meta: {
    title: 'routes:permission',
    authority: ['sys:platform_admin', 'sys:tenant_manager'],
  },
  children: [
    {
      name: 'permission-codes',
      path: 'codes',
      meta: {
        title: 'routes:permission-codes',
        authority: ['sys:platform_admin'],
      },
      element: createLazyRoute(() => import('@/pages/app/permission/permission')),
    },
  ],
}
```

**工作原理**：路由生成阶段通过 `generateRoutesByFrontend` 过滤，无权限的路由不会被注册。

---

### 2. 按钮级别权限控制

#### 方式一：Hook + 条件渲染（推荐）

对应 Vue 的 `v-access:code` 指令，React 用条件渲染替代：

```tsx
import { useAccess } from '@/core/access';

const UserPage = () => {
  const { hasAccessByCodes, hasAccessByRoles } = useAccess();

  return (
    <>
      {/* 基于权限码 */}
      {hasAccessByCodes(['sys:user:create']) && <Button>新建</Button>}
      {hasAccessByCodes(['sys:user:update']) && <Button>编辑</Button>}
      {hasAccessByCodes(['sys:user:delete']) && <Button danger>删除</Button>}

      {/* 基于角色码 */}
      {hasAccessByRoles(['admin']) && <Button>系统设置</Button>}
    </>
  );
};
```

#### 方式二：AccessControl 组件

对应 Vue 的 `<AccessControl>` 组件：

```tsx
import { AccessControl } from '@/core/access';

// 基于权限码
<AccessControl codes={['sys:user:create']} type="code">
  <Button>新建用户</Button>
</AccessControl>

// 基于角色码
<AccessControl codes={['admin']} type="role">
  <Button>管理员操作</Button>
</AccessControl>

// 混合匹配（meta.authority 模式）
<AccessControl codes={['admin', 'sys:user:export']} type="authority">
  <Button>导出</Button>
</AccessControl>

// 自定义 fallback
<AccessControl codes={['sys:user:delete']} type="code" fallback={<span>无权限</span>}>
  <Button danger>删除</Button>
</AccessControl>
```

#### 方式三：非组件场景（路由生成、工具函数）

```typescript
import { getAccessStatic } from '@/core/access';

// 在路由生成等非 React 场景
const { getAllPermissions, hasAccessByCodes } = getAccessStatic();
const permissions = getAllPermissions();

if (hasAccessByCodes(['sys:user:create'])) {
  // ...
}
```

---

## API 参考

### `useAccess()` Hook

权限判断的核心 Hook，用于 React 组件内部：

```typescript
const {
  userRoles,            // string[] - 当前用户的角色码列表
  accessCodes,          // string[] - 当前用户的权限码列表
  hasAccessByRoles,     // (roles: string[]) => boolean
  hasAccessByCodes,     // (codes: string[]) => boolean
  hasAccessByAuthority, // (authority: string[] | undefined) => boolean
  getAllPermissions,    // () => string[]
  isSuperAdmin,         // () => boolean
} = useAccess();
```

| 方法 | 参数 | 说明 |
|------|------|------|
| `hasAccessByRoles(roles)` | 角色码数组 | 用户角色中包含任一指定角色即通过 |
| `hasAccessByCodes(codes)` | 权限码数组 | 用户权限码中包含任一指定码即通过 |
| `hasAccessByAuthority(authority)` | authority 数组 | 同时匹配角色码和权限码 |
| `getAllPermissions()` | - | 返回角色码+权限码的合并列表 |
| `isSuperAdmin()` | - | 是否拥有 `*:*:*` 超级管理员角色 |

### `<AccessControl>` 组件

| 属性 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `codes` | `string[]` | - | 所需权限码/角色码列表 |
| `type` | `'code' \| 'role' \| 'authority'` | `'code'` | 鉴权维度 |
| `fallback` | `ReactNode` | `null` | 无权限时的替代内容 |
| `children` | `ReactNode` | - | 有权限时渲染的内容 |

### `getAccessStatic()` 静态方法

与 `useAccess()` 返回值相同，用于路由生成、工具函数等非 React 组件场景。

### `ROLE_ROOT` 常量

```typescript
import { ROLE_ROOT } from '@/core/access';
// 值为 '*:*:*'，拥有此角色的用户自动通过所有权限检查
```

---

## 权限流程

```
用户登录
  → useAuthStore 存储 token
  → router/index.tsx useEffect 触发
    → useAuth().getUserPermissionCodes()
      → fetchUserInfo()        → userInfo.roles[]  写入 useUserStore.userRoles
      → fetchMyPermissionCode() → codes[]           写入 useUserStore.accessCodes
    → getAccessStatic().getAllPermissions()
    → createAccessibleRouter({ permissions })
      → generateRoutesByFrontend: hasPermission 检查 meta.authority
    → 路由树生成完成
  → MainLayout / PageContainer
    → useAccess() 提供鉴权方法
    → 菜单过滤 / 页面权限 / 按钮权限
```

---

## 存储结构

```typescript
// useUserStore (Zustand + persist)
interface UserState {
  userInfo: BasicUserInfo | null;  // 用户信息（含 roles 字段）
  userRoles: string[];              // 角色码（来自 userInfo.roles）
  accessCodes: string[];            // 权限码（来自 GetMyPermissionCode）
}
```

---

## 常见问题

### Q1: 如何调试权限？

```typescript
import { useAccess } from '@/core/access';

const { userRoles, accessCodes, isSuperAdmin } = useAccess();
console.log('角色码:', userRoles);
console.log('权限码:', accessCodes);
console.log('超级管理员:', isSuperAdmin());
```

### Q2: 权限变更后如何刷新？

```typescript
import { useAuth } from '@/hooks/useAuth';

const auth = useAuth();
await auth.getUserPermissionCodes(); // 重新获取并更新 store
// Zustand 响应式更新，组件自动重渲染
```

### Q3: 如何实现"与"逻辑（需同时拥有多个权限）？

```tsx
const { hasAccessByCodes } = useAccess();

const canAdvanced = hasAccessByCodes(['sys:user:create'])
  && hasAccessByCodes(['sys:user:approve']);

return canAdvanced ? <Button>高级操作</Button> : null;
```

### Q4: 如何在非组件环境中检查权限？

```typescript
import { getAccessStatic } from '@/core/access';

const { hasAccessByCodes } = getAccessStatic();
if (hasAccessByCodes(['sys:user:create'])) {
  // 执行操作
}
```

---

## 权限前缀规范

```
模块:资源:操作
例如：
- sys:user:create    系统模块-用户资源-创建操作
- sys:user:update    系统模块-用户资源-更新操作
- sys:user:delete    系统模块-用户资源-删除操作
- sys:user:view      系统模块-用户资源-查看操作
- sys:user:export    系统模块-用户资源-导出操作
```

---

## 注意事项

1. **权限缓存**：`userRoles` 和 `accessCodes` 不持久化（`partialize` 排除），页面刷新后通过路由守卫重新获取
2. **超级管理员**：拥有 `*:*:*` 角色的用户自动通过所有检查，无需逐项匹配
3. **性能**：鉴权使用 `Set` 数据结构，查询复杂度 O(1)
4. **安全性**：前端权限控制仅控制 UI 展示，真正的权限验证在后端进行

---

## 相关文件

- [路由生成](../router/generators/generate-routes-frontend.ts) - 前端路由权限过滤
- [菜单过滤](../router/utils/menu.ts) - 菜单权限过滤
- [路由守卫](../router/guards/AuthGuard.tsx) - 认证守卫
- [用户 Store](../../stores/user.ts) - 权限数据存储
- [认证 Hook](../../hooks/useAuth.ts) - 权限码获取
