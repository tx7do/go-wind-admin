# API 开发规约文档

本文档描述了项目中 API 层的架构设计、使用规范和最佳实践。

---

## 📋 目录

- [架构概览](#架构概览)
- [目录结构](#目录结构)
- [两层架构详解](#两层架构详解)
- [使用指南](#使用指南)
- [开发规范](#开发规范)
- [最佳实践](#最佳实践)
- [常见问题](#常见问题)

---

## 架构概览

本项目采用**两层 API 架构**，通过 `apiClient` 单例直连调用：

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
               │ 通过 apiClient 调用
┌──────────────▼──────────────────────────┐
│      apiClient 单例 (src/api/client.ts)   │
│  - 懒加载各 Service Client               │
│  - 统一 Token 注入、错误拦截、自动刷新    │
└──────────────┬──────────────────────────┘
               │ 调用 Generated Code
┌──────────────▼──────────────────────────┐
│   Generated API Code (自动生成代码)       │
│  - createXxxServiceClient()             │
│  - createApiClient()                    │
│  - TypeScript Types                     │
└─────────────────────────────────────────┘
```

### 核心原则

1. **职责分离**：Hooks 层负责 React Query 集成，apiClient 负责传输层
2. **单向依赖**：Hooks → apiClient → Generated，不反向依赖
3. **类型安全**：全程 TypeScript 类型支持
4. **环境隔离**：区分 React 组件环境和非 React 环境

---

## 目录结构

```
src/api/
├── generated/                    # 自动生成的 API 代码（不要手动修改）
│   └── admin/service/v1/
│       └── index.ts              # 所有 API 类型、Service Client 工厂、ApiClient 类
│
├── client.ts                     # apiClient 单例（适配 transport，懒加载 Service）
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

## 两层架构详解

### 1️⃣ apiClient 单例（`src/api/client.ts`）

**位置**: `src/api/client.ts`

**职责**:

- 将已有的 `requestApi`（基于 axios）适配为 `ClientTransport` 接口
- 通过 `createApiClient(transport)` 创建聚合型 `apiClient`
- `apiClient` 以懒加载 getter 的形式暴露所有 Service Client
- 保留 Token 注入、错误拦截、自动刷新等全部已有逻辑

**ApiClient 结构**:

```typescript
// apiClient 聚合了所有 Service Client，通过懒加载 getter 访问
export const apiClient = createApiClient(transport);

// 使用方式 — 每个 getter 首次访问时创建对应 Service Client
apiClient.userService.List(params);       // 用户服务
apiClient.roleService.List(params);       // 角色服务
apiClient.authenticationService.Login(req); // 认证服务
// ... 更多 Service
```

---

### 2️⃣ Hooks 层（React Query 集成层）

**位置**: `src/api/hooks/*.ts`

**职责**:

- 通过 `apiClient` 直接调用 Service 方法
- 提供 `useMutation` / `useQuery` 封装
- 同时提供非 Hook 的 `fetchXxx()` 方法
- 配置 React Query 选项（重试、缓存等）

**标准模板**:

```typescript
import { useMutation, useQuery, type UseMutationOptions, type UseQueryOptions } from '@tanstack/react-query';
import {
  type xxxv1_Item,
  type xxxv1_GetItemRequest,
  type xxxv1_ListItemResponse,
  type xxxv1_CreateItemRequest,
  type xxxv1_DeleteItemRequest,
} from '@/api/generated/admin/service/v1';
import { makeUpdateMask, type PaginationQuery } from '@/core';
import { queryClient } from '@/core';
import { apiClient } from '@/api/client';

// ==============================
// 列表查询（useQuery + fetch）
// ==============================
export function useListItems(
  query: PaginationQuery,
  options?: UseQueryOptions<xxxv1_ListItemResponse, Error>,
) {
  return useQuery({
    queryKey: ['listItems', query],
    queryFn: () => apiClient.itemService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListItems(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listItems', params],
    queryFn: () => apiClient.itemService.List(params.toRawParams()),
    retry: 0,
  });
}

// ==============================
// 获取详情
// ==============================
export function useGetItem(
  req: xxxv1_GetItemRequest,
  options?: UseQueryOptions<xxxv1_Item, Error>,
) {
  return useQuery({
    queryKey: ['getItem', req],
    queryFn: () => apiClient.itemService.Get(req),
    ...options,
  });
}

// ==============================
// 创建（Mutation）
// ==============================
export function useCreateItem(
  options?: UseMutationOptions<{}, Error, xxxv1_CreateItemRequest>,
) {
  return useMutation({
    mutationFn: (data) => apiClient.itemService.Create(data),
    ...options,
  });
}

// ==============================
// 更新（Mutation，含 updateMask）
// ==============================
export function useUpdateItem(
  options?: UseMutationOptions<{}, Error, { id: number; values: Record<string, any> }>,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      apiClient.itemService.Update({
        id,
        data: { ...values },
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

// ==============================
// 删除（Mutation）
// ==============================
export function useDeleteItem(
  options?: UseMutationOptions<{}, Error, xxxv1_DeleteItemRequest>,
) {
  return useMutation({
    mutationFn: (req) => apiClient.itemService.Delete(req),
    ...options,
  });
}
```

**关键要点**:

- ✅ 直接通过 `apiClient.xxxService.Method()` 调用
- ✅ 同时提供 `useXxx()` Hook 和 `fetchXxx()` 方法
- ✅ Hook 用于 React 组件，享受 React Query 的缓存、重试等功能
- ✅ `fetchXxx()` 用于 Zustand Store、工具函数等非 React 环境
- ✅ `fetchXxx()` 设置 `retry: 0`，避免意外重试
- ✅ 使用有意义的 `queryKey`，便于缓存管理

---

### 3️⃣ Generated 层（自动生成代码）

**位置**: `src/api/generated/admin/service/v1/`

**说明**:

- ⚠️ **此目录由工具自动生成，不要手动修改**
- 包含所有 API 的 TypeScript 类型定义
- 包含所有 Service Client 的工厂函数（`createXxxServiceClient`）
- 包含聚合型 `ApiClient` 类和 `createApiClient` 工厂
- 当后端 API 变更时，重新生成此目录

---

## 使用指南

### 场景 1：在 React 组件中使用

```tsx
import { useListItems, useGetItem, useCreateItem } from '@/api/hooks/item';

function ItemList() {
  // 使用 Hook 获取数据
  const { data, isLoading } = useListItems(new PaginationQuery({ page: 1, pageSize: 10 }));
  const createMutation = useCreateItem();

  const handleCreate = async () => {
    await createMutation.mutateAsync({ data: { name: 'new item' } });
  };

  return <div>...</div>;
}
```

### 场景 2：在 Zustand Store 中使用

```typescript
import { create } from 'zustand';
import { fetchListItems } from '@/api/hooks/item';
import { PaginationQuery } from '@/core';

export const useItemStore = create<{
  items: any[];
  loadItems: () => Promise<void>;
}>((set) => ({
  items: [],
  loadItems: async () => {
    // 使用 fetch 方法（非 Hook）
    const result = await fetchListItems(new PaginationQuery({ page: 1, pageSize: 20 }));
    set({ items: result.items ?? [] });
  },
}));
```

### 场景 3：在路由守卫中使用

```typescript
import { fetchNavigation, fetchMyPermissionCode } from '@/api/hooks/admin-portal';

export async function initRoutes() {
  // 使用 fetch 方法（非 Hook）
  const navigation = await fetchNavigation();
  const permissions = await fetchMyPermissionCode();
  // ...
}
```

### 场景 4：直接使用 apiClient（特殊场景）

当 Hooks 层未封装某个方法，或需要直接调用时：

```typescript
import { apiClient } from '@/api/client';

// 直接调用 apiClient（任何时候都可用）
const result = await apiClient.userService.List(params);
```

---

## 开发规范

### 1. 命名规范

#### Hooks 层命名

```typescript
// React Hooks
export function useListXxx() { ... }
export function useGetXxx() { ... }
export function useCreateXxx() { ... }
export function useUpdateXxx() { ... }
export function useDeleteXxx() { ... }

// Fetch 方法（非 Hook，供 Store/路由守卫调用）
export async function fetchListXxx() { ... }
export async function fetchXxx() { ... }
```

### 2. 类型规范

```typescript
// ✅ 正确：使用生成的类型
import type { identityservicev1_User } from '@/api/generated/admin/service/v1';

// ✅ 正确：Hook 中使用 UseQueryOptions / UseMutationOptions
export function useGetUser(
  req: identityservicev1_GetUserRequest,
  options?: UseQueryOptions<identityservicev1_User, Error>,
) {
  return useQuery({
    queryKey: ['getUser', req],
    queryFn: () => apiClient.userService.Get(req),
    ...options,
  });
}

// ❌ 错误：使用 any
export function useGetUser(req: any, options?: any) { ... }
```

---

## 最佳实践

### 1. 选择合适的调用方式

| 使用场景            | 推荐方式            | 示例                              |
|-----------------|-----------------|--------------------------------|
| React 组件中       | `useXxx()` Hook | `const { data } = useListUsers(query)` |
| Zustand Store 中 | `fetchXxx()` 方法 | `await fetchUser(id)`           |
| 路由守卫中           | `fetchXxx()` 方法 | `await fetchNavigation()`       |
| 工具函数中           | `fetchXxx()` 方法 | `await fetchMyPermissionCode()` |
| Hooks 未封装的方法   | `apiClient` 直调 | `await apiClient.xxxService.Method()` |

### 2. 利用 React Query 的缓存失效

```typescript
export function useCreateUser() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data) => apiClient.userService.Create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['listUsers'] });
    },
  });
}
```

---

## 常见问题

### Q1: 为什么没有 Service 层？

**A**: 项目使用 `apiClient` 单例替代了独立的 Service 层。`apiClient` 内部以懒加载方式创建各 Service Client，Hooks 层直接通过 `apiClient.xxxService.Method()` 调用，减少了中间层，代码更简洁。

### Q2: 什么时候用 `useXxx()`，什么时候用 `fetchXxx()`？

**A**:

- **`useXxx()`**：在 React 组件中使用，享受 React Query 的所有功能
- **`fetchXxx()`**：在非 React 环境（Store、工具函数、路由守卫）中使用

### Q3: 如何添加新的 API 模块？

**A**: 遵循以下步骤：

1. **生成 API 代码**（后端提供 proto 文件后）
   ```bash
   # 本端 package.json 没有 generate:api 脚本；TS 代码由后端 Makefile 的 ts 目标生成
   # （backend/api/buf.react.admin.typescript.gen.yaml 输出到 src/api/generated/）
   cd backend && make ts
   ```

2. **创建 Hooks 文件** (`src/api/hooks/xxx.ts`)
   ```typescript
   import { useMutation, useQuery, type UseMutationOptions, type UseQueryOptions } from '@tanstack/react-query';
   import { type PaginationQuery, queryClient } from '@/core';
   import { apiClient } from '@/api/client';

   export function useListXxx(
     query: PaginationQuery,
     options?: UseQueryOptions<...>,
   ) {
     return useQuery({
       queryKey: ['listXxx', query],
       queryFn: () => apiClient.xxxService.List(query.toRawParams()),
       ...options,
     });
   }

   export async function fetchListXxx(params: PaginationQuery) {
     return queryClient.fetchQuery({
       queryKey: ['listXxx', params],
       queryFn: () => apiClient.xxxService.List(params.toRawParams()),
       retry: 0,
     });
   }
   ```

3. **导出模块** (`src/api/hooks/index.ts`)
   ```typescript
   export * from './xxx';
   ```

### Q4: 如何处理分页查询？

**A**: 使用 `PaginationQuery` 类型：

```typescript
// Hooks 层直接转换
export function useListUsers(
  query: PaginationQuery,
  options?: UseQueryOptions<...>,
) {
  return useQuery({
    queryKey: ['listUsers', query],
    queryFn: () => apiClient.userService.List(query.toRawParams()),
    ...options,
  });
}
```

---

## 附录

### 相关资源

- [TanStack Query 官方文档](https://tanstack.com/query/latest)
- [Zustand 官方文档](https://zustand-demo.pmnd.rs/)
- [TypeScript 官方文档](https://www.typescriptlang.org/docs/)
