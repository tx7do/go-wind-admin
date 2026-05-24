# API 开发规约文档

本文档描述了项目中 API 层的架构设计、使用规范和最佳实践。

---

## 📋 目录

- [架构概览](#架构概览)
- [目录结构](#目录结构)
- [三层架构详解](#三层架构详解)
- [使用指南](#使用指南)
- [开发规范](#开发规范)
- [最佳实践](#最佳实践)
- [常见问题](#常见问题)

---

## 架构概览

本项目采用**三层 API 架构**，清晰分离关注点：

```
┌─────────────────────────────────────────┐
│         React Components (UI层)          │
└──────────────┬──────────────────────────┘
               │ 使用 Hooks
┌──────────────▼──────────────────────────┐
│      API Hooks Layer (React Query层)     │
│  - useXxx() - React Hooks               │
│  - fetchXxx() - 非 Hook 方法             │
└──────────────┬──────────────────────────┘
               │ 调用 Service
┌──────────────▼──────────────────────────┐
│      API Service Layer (服务封装层)       │
│  - getXxxService() - Service Client     │
│  - listXxx(), getXxx() - 纯函数          │
└──────────────┬──────────────────────────┘
               │ 调用 Generated Code
┌──────────────▼──────────────────────────┐
│   Generated API Code (自动生成代码)       │
│  - createXxxServiceClient()             │
│  - TypeScript Types                     │
└─────────────────────────────────────────┘
```

### 核心原则

1. **职责分离**：每层只负责自己的职责
2. **单向依赖**：上层依赖下层，不反向依赖
3. **类型安全**：全程 TypeScript 类型支持
4. **环境隔离**：区分 React 组件环境和非 React 环境

---

## 目录结构

```
src/api/
├── generated/                    # 自动生成的 API 代码（不要手动修改）
│   └── admin/service/v1/
│       ├── index.ts              # 所有 API 类型和 Client 工厂
│
├── service/                      # Service 层 - 服务封装
│   ├── auth.ts                   # 认证服务
│   ├── user.ts                   # 用户管理服务
│   ├── index.ts                  # 统一导出
│   └── ...
│
├── hooks/                        # Hooks 层 - React Query 集成
│   ├── auth.ts                   # 认证相关 Hooks
│   ├── user.ts                   # 用户管理 Hooks
│   ├── index.ts                  # 统一导出
│   └── ...
│
└── index.ts                      # API 模块总入口
```

---

## 三层架构详解

### 1️⃣ Service 层（服务封装层）

**位置**: `src/api/service/*.ts`

**职责**:

- 封装生成的 API Client
- 提供单例模式的 Service Client
- 导出纯函数式的 API 调用方法
- 处理请求参数转换（如分页参数）

**标准模板**:

```typescript
import {
  createUserServiceClient,
  type identityservicev1_User,
  type identityservicev1_GetUserRequest,
  type identityservicev1_ListUserResponse,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery, requestApi } from '@/core';

// 单例模式：缓存 Service Client 实例
let _instance: ReturnType<typeof createUserServiceClient> | null = null;

/**
 * 获取用户服务单例（延迟初始化）
 */
export function getUserService() {
  if (!_instance) {
    _instance = createUserServiceClient(requestApi);
  }
  return _instance;
}

// ==============================
// 用户管理 API
// ==============================

/**
 * 获取用户列表
 */
export async function listUsers(query: PaginationQuery) {
  const params = query.toRawParams();
  return getUserService().List(params);
}

/**
 * 获取单个用户
 */
export async function getUser(request: identityservicev1_GetUserRequest) {
  return getUserService().Get(request);
}

/**
 * 创建用户
 */
export async function createUser(data: identityservicev1_User) {
  return getUserService().Create({ data });
}

/**
 * 更新用户
 */
export async function updateUser(request: identityservicev1_UpdateUserRequest) {
  return getUserService().Update(request);
}

/**
 * 删除用户
 */
export async function deleteUser(request: { id: number }) {
  return getUserService().Delete(request);
}
```

**关键要点**:

- ✅ 使用单例模式缓存 Service Client
- ✅ 导出 `getXxxService()` 函数供外部使用
- ✅ 导出纯函数式 API 方法（`async/await`）
- ✅ 处理分页参数转换（`query.toRawParams()`）
- ❌ 不要在 Service 层使用 React Hooks
- ❌ 不要直接暴露 Service Client 给 UI 层

---

### 2️⃣ Hooks 层（React Query 集成层）

**位置**: `src/api/hooks/*.ts`

**职责**:

- 将 Service 层的函数包装为 React Hooks
- 提供 `useMutation` / `useQuery` 封装
- 同时提供非 Hook 的 `fetchXxx()` 方法
- 配置 React Query 选项（重试、缓存等）

**标准模板**:

```typescript
import { useMutation, type UseMutationOptions } from '@tanstack/react-query';
import {
  type identityservicev1_User,
  type identityservicev1_GetUserRequest,
  type identityservicev1_ListUserResponse,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery } from '@/core';
import { listUsers, getUser, createUser, updateUser, deleteUser } from '@/api/service/user';
import { queryClient } from '@/core';

// ==============================
// React Hooks（用于 React 组件中）
// ==============================

/**
 * 获取用户列表 Hook
 */
export function useListUsers(
  options?: UseMutationOptions<identityservicev1_ListUserResponse, Error, PaginationQuery>,
) {
  return useMutation({
    mutationFn: (query) => listUsers(query),
    ...options,
  });
}

/**
 * 获取单个用户 Hook
 */
export function useGetUser(
  options?: UseMutationOptions<identityservicev1_User, Error, identityservicev1_GetUserRequest>,
) {
  return useMutation({
    mutationFn: (req) => getUser(req),
    ...options,
  });
}

/**
 * 创建用户 Hook
 */
export function useCreateUser(
  options?: UseMutationOptions<{}, Error, identityservicev1_User>,
) {
  return useMutation({
    mutationFn: (data) => createUser(data),
    ...options,
  });
}

// ==============================
// Fetch 方法（用于 Store / 非 React 环境）
// ==============================

/**
 * 获取用户列表【给 Store / 外部调用】不带 Hook 的方法
 */
export async function fetchListUsers(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listUsers', params],
    queryFn: () => listUsers(params),
    retry: 0,
  });
}

/**
 * 获取单个用户【给 Store / 外部调用】不带 Hook 的方法
 */
export async function fetchUser(params: identityservicev1_GetUserRequest) {
  return queryClient.fetchQuery({
    queryKey: ['getUser', params],
    queryFn: () => getUser(params),
    retry: 0,
  });
}
```

**关键要点**:

- ✅ 同时提供 `useXxx()` Hook 和 `fetchXxx()` 方法
- ✅ Hook 用于 React 组件，享受 React Query 的缓存、重试等功能
- ✅ `fetchXxx()` 用于 Zustand Store、工具函数等非 React 环境
- ✅ `fetchXxx()` 设置 `retry: 0`，避免意外重试
- ✅ 使用有意义的 `queryKey`，便于缓存管理
- ❌ 不要在 Hooks 层直接创建 Service Client
- ❌ 不要在 Hooks 层直接使用 `requestApi`

---

### 3️⃣ Generated 层（自动生成代码）

**位置**: `src/api/generated/admin/service/v1/`

**说明**:

- ⚠️ **此目录由工具自动生成，不要手动修改**
- 包含所有 API 的 TypeScript 类型定义
- 包含所有 Service Client 的工厂函数
- 当后端 API 变更时，重新生成此目录

**主要导出**:

```typescript
// 类型定义
export type identityservicev1_User = { ... }
export type identityservicev1_GetUserRequest = { ... }

// Service Client 工厂
export function createUserServiceClient(requestApi: RequestApi): UserServiceClient

export function createMenuServiceClient(requestApi: RequestApi): MenuServiceClient

// ... 更多 Service Client
```

---

## 使用指南

### 场景 1：在 React 组件中使用

```tsx
import { useListUsers, useGetUser, useCreateUser } from '@/api/hooks/user';

function UserList() {
  // 使用 Hook 获取数据
  const listUsersMutation = useListUsers();
  const getUserMutation = useGetUser();
  const createUserMutation = useCreateUser();

  // 加载用户列表
  const handleLoadUsers = async () => {
    const result = await listUsersMutation.mutateAsync({
      page: 1,
      pageSize: 10,
    });
    console.log('用户列表:', result);
  };

  // 获取单个用户
  const handleGetUser = async (userId: number) => {
    const user = await getUserMutation.mutateAsync({ id: userId });
    console.log('用户详情:', user);
  };

  // 创建用户
  const handleCreateUser = async () => {
    await createUserMutation.mutateAsync({
      username: 'newuser',
      email: 'newuser@example.com',
    });
  };

  return <div>...</div>;
}
```

### 场景 2：在 Zustand Store 中使用

```typescript
import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { fetchLogin, fetchUserProfile } from '@/api/hooks/auth';
import { fetchListUsers } from '@/api/hooks/user';

interface AuthState {
  token: string | null;
  userInfo: any | null;
  users: any[];
  login: (username: string, password: string) => Promise<void>;
  logout: () => void;
  loadUsers: () => Promise<void>;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      token: null,
      userInfo: null,
      users: [],

      // 登录：使用 fetch 方法（非 Hook）
      login: async (username, password) => {
        try {
          // 调用登录 API
          const loginResult = await fetchLogin({
            username,
            password,
            grant_type: 'password',
          });

          // 获取用户信息
          const userInfo = await fetchUserProfile();

          set({
            token: loginResult.access_token,
            userInfo,
          });
        } catch (error) {
          console.error('登录失败:', error);
          throw error;
        }
      },

      // 登出
      logout: () => {
        set({ token: null, userInfo: null });
      },

      // 加载用户列表：使用 fetch 方法
      loadUsers: async () => {
        const result = await fetchListUsers({
          page: 1,
          pageSize: 20,
        });
        set({ users: result.items || [] });
      },
    }),
    {
      name: 'auth-storage',
    }
  )
);
```

### 场景 3：在路由守卫中使用

```typescript
// core/router/guards/auth-guard.ts
import { fetchNavigation, fetchMyPermissionCode } from '@/api/hooks/admin-portal';

export async function checkRoutePermission(routePath: string): Promise<boolean> {
  try {
    // 获取用户权限码
    const permissions = await fetchMyPermissionCode();

    // 检查是否有权限访问该路由
    return permissions.includes(routePath);
  } catch (error) {
    console.error('权限检查失败:', error);
    return false;
  }
}

export async function generateDynamicRoutes() {
  try {
    // 获取导航菜单（包含路由信息）
    const navigation = await fetchNavigation();

    // 转换为 React Router 路由配置
    return transformToRoutes(navigation);
  } catch (error) {
    console.error('生成路由失败:', error);
    return [];
  }
}
```

### 场景 4：在工具函数中使用

```typescript
// utils/permission.ts
import { fetchMyPermissionCode } from '@/api/hooks/admin-portal';

/**
 * 检查用户是否有指定权限
 */
export async function hasPermission(permissionCode: string): Promise<boolean> {
  const permissions = await fetchMyPermissionCode();
  return permissions.includes(permissionCode);
}

/**
 * 检查用户是否有任一权限
 */
export async function hasAnyPermission(permissionCodes: string[]): Promise<boolean> {
  const permissions = await fetchMyPermissionCode();
  return permissionCodes.some(code => permissions.includes(code));
}

/**
 * 检查用户是否有所有权限
 */
export async function hasAllPermissions(permissionCodes: string[]): Promise<boolean> {
  const permissions = await fetchMyPermissionCode();
  return permissionCodes.every(code => permissions.includes(code));
}
```

---

## 开发规范

### 1. 命名规范

#### Service 层命名

```typescript
// Service Client 获取函数
export function getUserService() { ...
}

export function getMenuService() { ...
}

// API 方法命名
export async function listXxx() { ...
}      // 列表查询
export async function getXxx() { ...
}       // 获取单个
export async function createXxx() { ...
}    // 创建
export async function updateXxx() { ...
}    // 更新
export async function deleteXxx() { ...
}    // 删除
export async function batchXxx() { ...
}     // 批量操作
```

#### Hooks 层命名

```typescript
// React Hooks
export function useListXxx() { ...
}

export function useGetXxx() { ...
}

export function useCreateXxx() { ...
}

export function useUpdateXxx() { ...
}

export function useDeleteXxx() { ...
}

// Fetch 方法
export async function fetchListXxx() { ...
}

export async function fetchXxx() { ...
}
```

### 2. 类型规范

```typescript
// ✅ 正确：使用生成的类型
import type { identityservicev1_User } from '@/api/generated/admin/service/v1';

// ✅ 正确：明确标注返回类型
export async function getUser(id: number): Promise<identityservicev1_User> {
  return getUserService().Get({ id });
}

// ✅ 正确：Hook 中使用 UseMutationOptions
export function useGetUser(
  options?: UseMutationOptions<identityservicev1_User, Error, number>,
) {
  return useMutation({
    mutationFn: (id) => getUser(id),
    ...options,
  });
}

// ❌ 错误：使用 any
export async function getUser(id: any): Promise<any> { ...
}

// ❌ 错误：缺少类型标注
export function useGetUser(options?) { ...
}
```

### 3. 错误处理规范

```typescript
// ✅ Service 层：让错误自然抛出
export async function getUser(id: number) {
  return getUserService().Get({ id });  // 错误会向上传播
}

// ✅ Hooks 层：通过 React Query 的 onError 处理
export function useGetUser(options?: UseMutationOptions<

...>)
{
  return useMutation({
    mutationFn: (id) => getUser(id),
    onError: (error) => {
      console.error('获取用户失败:', error);
      message.error('获取用户信息失败');
    },
    ...options,
  });
}

// ✅ Fetch 方法：由调用方决定如何处理
export async function fetchUser(id: number) {
  return queryClient.fetchQuery({
    queryKey: ['user', id],
    queryFn: () => getUser(id),
    retry: 0,  // 不自动重试
  });
}

// 调用方处理错误
try {
  const user = await fetchUser(1);
} catch (error) {
  message.error('获取用户失败');
}
```

### 4. 注释规范

```typescript
/**
 * 获取用户服务单例（延迟初始化）
 */
export function getUserService() { ...
}

/**
 * 获取用户列表
 * @param query 分页查询参数
 * @returns 用户列表响应
 */
export async function listUsers(query: PaginationQuery) { ...
}

/**
 * 获取用户列表 Hook
 * @param options React Query 配置选项
 * @returns Mutation 对象
 */
export function useListUsers(
  options?: UseMutationOptions<

...>,
)
{ ...
}

// ==============================================
// 获取用户列表 【给 Store / 外部调用】不带 Hook 的方法
// ==============================================
export async function fetchListUsers(params: PaginationQuery) { ...
}
```

---

## 最佳实践

### 1. 选择合适的调用方式

| 使用场景            | 推荐方式            | 示例                              |
|-----------------|-----------------|---------------------------------|
| React 组件中       | `useXxx()` Hook | `const mutation = useGetUser()` |
| Zustand Store 中 | `fetchXxx()` 方法 | `await fetchUser(id)`           |
| 路由守卫中           | `fetchXxx()` 方法 | `await fetchNavigation()`       |
| 工具函数中           | `fetchXxx()` 方法 | `await fetchMyPermissionCode()` |
| 非 React 环境      | `fetchXxx()` 方法 | `await fetchLogin(credentials)` |

### 2. 合理配置 React Query 选项

```typescript
// ✅ 列表查询：启用缓存
export function useListUsers(options?: UseMutationOptions<

...>)
{
  return useMutation({
    mutationFn: (query) => listUsers(query),
    gcTime: 5 * 60 * 1000,  // 缓存 5 分钟
    ...options,
  });
}

// ✅ 详情查询：较短缓存时间
export function useGetUser(options?: UseMutationOptions<

...>)
{
  return useMutation({
    mutationFn: (id) => getUser(id),
    gcTime: 2 * 60 * 1000,  // 缓存 2 分钟
    ...options,
  });
}

// ✅ 创建/更新/删除：不缓存
export function useCreateUser(options?: UseMutationOptions<

...>)
{
  return useMutation({
    mutationFn: (data) => createUser(data),
    gcTime: 0,  // 不缓存
    ...options,
  });
}
```

### 3. 利用 React Query 的缓存失效

```typescript
export function useCreateUser() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data) => createUser(data),
    onSuccess: () => {
      // 创建成功后，使列表缓存失效
      queryClient.invalidateQueries({ queryKey: ['listUsers'] });
      message.success('创建用户成功');
    },
  });
}

export function useUpdateUser() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data) => updateUser(data),
    onSuccess: (_, variables) => {
      // 更新成功后，使相关缓存失效
      queryClient.invalidateQueries({ queryKey: ['listUsers'] });
      queryClient.invalidateQueries({ queryKey: ['getUser', variables.id] });
      message.success('更新用户成功');
    },
  });
}
```

### 4. 统一错误处理

```typescript
// 在应用入口处配置全局错误处理
import { QueryClient } from '@tanstack/react-query';

export const queryClient = new QueryClient({
  defaultOptions: {
    mutations: {
      onError: (error) => {
        // 全局错误处理
        if (error.message.includes('401')) {
          // Token 过期，跳转登录
          window.location.href = '/login';
        } else if (error.message.includes('403')) {
          message.error('没有权限执行此操作');
        } else {
          message.error('操作失败，请稍后重试');
        }
      },
    },
  },
});
```

### 5. 性能优化建议

```typescript
// ✅ 使用 optimistic updates 提升用户体验
export function useUpdateUser() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data) => updateUser(data),
    onMutate: async (newData) => {
      // 取消正在进行的查询
      await queryClient.cancelQueries({ queryKey: ['getUser', newData.id] });

      // 保存之前的值
      const previousUser = queryClient.getQueryData(['getUser', newData.id]);

      // 乐观更新
      queryClient.setQueryData(['getUser', newData.id], newData);

      return { previousUser };
    },
    onError: (err, newData, context) => {
      // 出错时回滚
      queryClient.setQueryData(['getUser', newData.id], context?.previousUser);
    },
    onSettled: () => {
      // 最终使缓存失效
      queryClient.invalidateQueries({ queryKey: ['getUser'] });
    },
  });
}
```

---

## 常见问题

### Q1: 为什么需要 Service 层和 Hooks 层？

**A**:

- **Service 层**：封装纯函数式 API 调用，可在任何环境中使用（React、Node.js、测试等）
- **Hooks 层**：集成 React Query，提供声明式数据获取、缓存、重试等功能
- **分离好处**：职责清晰、易于测试、可复用性强

### Q2: 什么时候用 `useXxx()`，什么时候用 `fetchXxx()`？

**A**:

- **`useXxx()`**：在 React 组件中使用，享受 React Query 的所有功能
- **`fetchXxx()`**：在非 React 环境（Store、工具函数、路由守卫）中使用

### Q3: 如何添加新的 API 模块？

**A**: 遵循以下步骤：

1. **生成 API 代码**（后端提供 proto 文件后）
   ```bash
   npm run generate:api
   ```

2. **创建 Service 文件** (`src/api/service/xxx.ts`)
   ```typescript
   import { createXxxServiceClient } from '@/api/generated/admin/service/v1';
   import { requestApi } from '@/core';
   
   let _instance: ReturnType<typeof createXxxServiceClient> | null = null;
   
   export function getXxxService() {
     if (!_instance) {
       _instance = createXxxServiceClient(requestApi);
     }
     return _instance;
   }
   
   export async function listXxx(query: PaginationQuery) { ... }
   export async function getXxx(id: number) { ... }
   ```

3. **创建 Hooks 文件** (`src/api/hooks/xxx.ts`)
   ```typescript
   import { useMutation } from '@tanstack/react-query';
   import { listXxx, getXxx } from '@/api/service/xxx';
   import { queryClient } from '@/core';
   
   export function useListXxx(options?) {
     return useMutation({ mutationFn: (q) => listXxx(q), ...options });
   }
   
   export async function fetchListXxx(params) {
     return queryClient.fetchQuery({
       queryKey: ['listXxx', params],
       queryFn: () => listXxx(params),
       retry: 0,
     });
   }
   ```

4. **导出模块** (`src/api/service/index.ts` 和 `src/api/hooks/index.ts`)
   ```typescript
   export * from './xxx';
   ```

### Q4: 如何处理分页查询？

**A**: 使用 `PaginationQuery` 类型：

```typescript
import { type PaginationQuery } from '@/core';

// Service 层
export async function listUsers(query: PaginationQuery) {
  const params = query.toRawParams();  // 转换为原始参数
  return getUserService().List(params);
}

// Hooks 层
export function useListUsers(options?) {
  return useMutation({
    mutationFn: (query: PaginationQuery) => listUsers(query),
    ...options,
  });
}

// 使用
const pagination = new PaginationQuery({ page: 1, pageSize: 10 });
const result = await listUsers(pagination);
```

### Q5: 如何调试 API 调用？

**A**:

1. **使用 React Query Devtools**
   ```typescript
   // App.tsx
   import { ReactQueryDevtools } from '@tanstack/react-query-devtools';
   
   function App() {
     return (
       <>
         <YourApp />
         <ReactQueryDevtools initialIsOpen={false} />
       </>
     );
   }
   ```

2. **查看网络请求**
    - 浏览器开发者工具 → Network 标签
    - 过滤 XHR/Fetch 请求

3. **日志输出**
   ```typescript
   export async function listUsers(query: PaginationQuery) {
     console.log('[API] listUsers called with:', query);
     const result = await getUserService().List(query.toRawParams());
     console.log('[API] listUsers result:', result);
     return result;
   }
   ```

---

## 附录

### 相关资源

- [TanStack Query 官方文档](https://tanstack.com/query/latest)
- [Zustand 官方文档](https://zustand-demo.pmnd.rs/)
- [TypeScript 官方文档](https://www.typescriptlang.org/docs/)

### 项目特定资源

- Core 模块文档：`src/core/README.md`
- 状态管理规范：`src/stores/README.md`
- 路由系统文档：`src/router/README.md`
