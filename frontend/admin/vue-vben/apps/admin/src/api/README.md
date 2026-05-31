# api 模块

> 面向 admin 业务逻辑开发者的使用指南

## 模块定位

API 层采用 **三层架构**，将后端 gRPC/HTTP 接口封装为前端可用的 Vue Query hooks：

```
generated/           ← 第 1 层：protobuf 自动生成的 TypeScript 类型 + Service Client 工厂
  （由 protoc-gen-typescript-http 生成，不要手动编辑）

service/             ← 第 2 层：Service 层，调用 generated Client，封装纯异步函数
  每个 service 文件 = 一个 gRPC Service 的单例封装

composables/         ← 第 3 层：Vue Query hooks 层，面向业务组件的最终 API
  use*   = 组件内使用的 Vue Query hook（useQuery / useMutation）
  fetch* = 组件外使用的 Promise 方法（Store、路由守卫等）
  枚举工具 = 各模块的状态/颜色映射函数
```

---

## 目录结构

```
src/api/
├── index.ts                        # 统一导出入口
├── generated/
│   └── admin/service/v1/           # protobuf 自动生成的代码（勿手动修改）
│       └── index.ts                # 所有类型 + createXxxServiceClient 工厂
├── service/                        # Service 层（一个文件对应一个后端 Service）
│   ├── auth.ts                     # 认证服务
│   ├── user.ts                     # 用户管理
│   ├── role.ts                     # 角色管理
│   ├── permission.ts               # 权限管理
│   ├── permission-group.ts         # 权限组管理
│   ├── org-unit.ts                 # 组织单元
│   ├── position.ts                 # 岗位管理
│   ├── menu.ts                     # 菜单管理
│   ├── api.ts                      # API 管理
│   ├── dict.ts                     # 字典管理（类型 + 条目）
│   ├── file.ts                     # 文件管理
│   ├── file-transfer.ts            # 文件传输
│   ├── tenant.ts                   # 租户管理
│   ├── task.ts                     # 异步任务
│   ├── login-policy.ts             # 登录策略
│   ├── language.ts                 # 语言管理
│   ├── admin-portal.ts             # 管理门户（导航等）
│   ├── user-profile.ts             # 用户个人资料
│   ├── internal-message.ts         # 内部消息
│   ├── login-audit-log.ts          # 登录审计日志
│   ├── api-audit-log.ts            # API 审计日志
│   ├── operation-audit-log.ts      # 操作审计日志
│   ├── data-access-audit-log.ts    # 数据访问审计日志
│   ├── permission-audit-log.ts     # 权限审计日志
│   └── policy-evaluation-log.ts    # 策略评估日志
└── composables/                    # Vue Query hooks 层（与 service 一一对应）
    ├── shared.ts                   # 通用枚举工具（enable/status/成功失败等）
    ├── auth.ts                     # 认证 hooks
    ├── user.ts                     # 用户 hooks + 用户枚举工具
    ├── ...其余同 service 目录
    └── index.ts                    # 统一导出
```

---

## 快速开始

### 在组件中查询数据

```vue
<script setup lang="ts">
import { useListUsers } from "@/api";
import { PaginationQuery } from "#/transport/rest";

const query = new PaginationQuery({
  paging: { page: 1, pageSize: 20 },
  formValues: { status: "NORMAL" },
});

const { data, isLoading, error } = useListUsers(query);
</script>

<template>
  <div v-if="isLoading">加载中...</div>
  <div v-else-if="error">{{ error.message }}</div>
  <div v-else>{{ data }}</div>
</template>
```

### 在组件中执行写操作

```vue
<script setup lang="ts">
import { useCreateUser } from "@/api";

const { mutateAsync, isPending } = useCreateUser();

async function handleSubmit(formValues: Record<string, any>) {
  await mutateAsync({ data: formValues, password: "default123" });
}
</script>
```

### 在 Store / 路由守卫等非组件上下文中调用

```ts
import { fetchListUsers, fetchUser } from "@/api";

// fetch* 函数不依赖 Vue 组件上下文，可在任何地方使用
const users = await fetchListUsers(new PaginationQuery({ paging: { page: 1, pageSize: 10 } }));
const user = await fetchUser({ id: 1 });
```

---

## 三层架构详解

### 第 1 层：generated（自动生成）

由 `protoc-gen-typescript-http` 根据 protobuf 定义生成，包含：

- **类型定义**：`permissionservicev1_Role`、`identityservicev1_User` 等
- **请求/响应类型**：`permissionservicev1_ListRoleResponse`、`identityservicev1_CreateUserRequest` 等
- **Service Client 工厂**：`createUserServiceClient(requestApi)` 等

```ts
// 典型导入（composables 和 service 层使用）
import type {
  identityservicev1_User,
  identityservicev1_ListUserResponse,
  createUserServiceClient,
} from "#/api/generated/admin/service/v1";
```

**命名规则**：`{service}v1_{MessageName}`
- `identityservicev1_` — 用户/身份相关
- `permissionservicev1_` — 权限/角色/菜单相关
- `dictservicev1_` — 字典相关
- `authenticationservicev1_` — 认证相关
- 以此类推

### 第 2 层：service

每个文件对应一个后端 gRPC Service，提供：
- **延迟初始化的单例 Client**
- **纯异步函数**（可直接 `await` 调用）

```ts
// service/user.ts 的典型结构
import { createUserServiceClient } from "#/api/generated/admin/service/v1";
import { requestApi } from "#/transport/rest";

let _instance: ReturnType<typeof createUserServiceClient> | null = null;

export function getUserService() {
  if (!_instance) {
    _instance = createUserServiceClient(requestApi);
  }
  return _instance;
}

export async function listUsers(query: PaginationQuery) {
  return getUserService().List(query.toRawParams());
}
```

**何时直接用 service 层**：当你在 composables 层构建 hook 时，或在非 Vue 上下文不需要缓存时。

### 第 3 层：composables

面向业务组件的最终 API 层，提供三种导出：

#### `use*` — Vue Query hooks（组件内使用）

```ts
// Query hooks（读取数据）
const { data, isLoading, refetch } = useListUsers(query);
const { data } = useGetUser({ id: 1 });

// Mutation hooks（写操作）
const { mutateAsync, isPending } = useCreateUser();
const { mutateAsync } = useUpdateUser();
const { mutateAsync } = useDeleteUser();
```

#### `fetch*` — Promise 方法（Store / 路由守卫等外部调用）

```ts
// 不依赖组件 setup 上下文，内部通过 queryClient.fetchQuery 实现
const users = await fetchListUsers(query);
const user = await fetchUser({ id: 1 });
```

#### 枚举工具 — 状态映射函数

```ts
import { userStatusToColor, userStatusToName, genderToName } from "@/api";

const color = userStatusToColor("NORMAL");  // "#4096FF"
const label = userStatusToName("NORMAL");   // "正常"
```

---

## 核心 API 模式

### 分页查询

所有列表查询统一使用 `PaginationQuery`：

```ts
import { PaginationQuery } from "#/transport/rest";

const query = new PaginationQuery({
  paging: { page: 1, pageSize: 20 },         // 分页参数
  formValues: { status: "NORMAL", name: "张" }, // 搜索条件（自动过滤空值）
  orderBy: ["-created_at"],                     // 排序（"-"前缀 = 降序）
  fieldMask: "id,name,status",                  // 只返回指定字段
});

// 在 hooks 中使用
const { data } = useListUsers(query);
```

**`PaginationQuery` 参数说明：**

| 参数 | 类型 | 说明 |
|---|---|---|
| `paging` | `{ page?, pageSize? }` | 页码和每页条数 |
| `formValues` | `Record<string, unknown>` | 搜索条件，自动序列化为 JSON filter |
| `orderBy` | `string[]` | 排序字段，默认 `["-created_at"]` |
| `fieldMask` | `string \| string[]` | 只返回指定字段（field mask） |
| `isTenantUser` | `boolean` | 是否为租户用户（自动清理 tenant_id 字段） |

### 创建操作

```ts
const { mutateAsync } = useCreateUser();

await mutateAsync({
  data: { name: "张三", email: "zhangsan@example.com", ... },
  password: "initialPassword",
});
```

### 更新操作（自动生成 updateMask）

```ts
const { mutateAsync } = useUpdateUser();

await mutateAsync({
  id: 1,
  values: { name: "李四", email: "lisi@example.com" },  // 只传需要更新的字段
});
// 内部自动生成 updateMask: "name,email,id"
```

### 删除操作

```ts
// 单个删除
const { mutateAsync } = useDeleteUser();
await mutateAsync({ id: 1 });

// 批量删除（某些模块支持）
const { mutateAsync } = useDeleteDictEntry();
await mutateAsync({ ids: [1, 2, 3] });
```

---

## 模块速查表

### 认证与门户

| 模块 | composable | service | 说明 |
|---|---|---|---|
| 认证 | `auth.ts` | `auth.ts` | 登录、登出、注册、验证码、刷新Token |
| 管理门户 | `admin-portal.ts` | `admin-portal.ts` | 导航菜单获取 |

### 用户与身份

| 模块 | composable | service | 说明 |
|---|---|---|---|
| 用户管理 | `user.ts` | `user.ts` | 用户 CRUD + 枚举（状态/性别） |
| 用户资料 | `user-profile.ts` | `user-profile.ts` | 当前用户个人资料 |
| 租户管理 | `tenant.ts` | `tenant.ts` | 租户 CRUD |

### 组织架构

| 模块 | composable | service | 说明 |
|---|---|---|---|
| 组织单元 | `org-unit.ts` | `org-unit.ts` | 部门树管理 |
| 岗位管理 | `position.ts` | `position.ts` | 岗位 CRUD |

### 权限管理

| 模块 | composable | service | 说明 |
|---|---|---|---|
| 权限 | `permission.ts` | `permission.ts` | 权限 CRUD |
| 权限组 | `permission-group.ts` | `permission-group.ts` | 权限组 CRUD |
| 角色 | `role.ts` | `role.ts` | 角色 CRUD |
| 菜单 | `menu.ts` | `menu.ts` | 菜单 CRUD |
| API | `api.ts` | `api.ts` | API 管理 |

### 系统管理

| 模块 | composable | service | 说明 |
|---|---|---|---|
| 字典 | `dict.ts` | `dict.ts` | 字典类型 + 字典条目 |
| 文件 | `file.ts` | `file.ts` | 文件上传下载 |
| 文件传输 | `file-transfer.ts` | `file-transfer.ts` | 文件传输任务 |
| 异步任务 | `task.ts` | `task.ts` | 后台任务管理 |
| 登录策略 | `login-policy.ts` | `login-policy.ts` | 登录策略配置 |
| 语言 | `language.ts` | `language.ts` | 语言管理 |

### 审计日志

| 模块 | composable | service | 说明 |
|---|---|---|---|
| 登录日志 | `login-audit-log.ts` | `login-audit-log.ts` | 登录审计 |
| API 日志 | `api-audit-log.ts` | `api-audit-log.ts` | API 调用审计 |
| 操作日志 | `operation-audit-log.ts` | `operation-audit-log.ts` | 操作审计 |
| 数据访问日志 | `data-access-audit-log.ts` | `data-access-audit-log.ts` | 数据访问审计 |
| 权限日志 | `permission-audit-log.ts` | `permission-audit-log.ts` | 权限变更审计 |
| 策略评估日志 | `policy-evaluation-log.ts` | `policy-evaluation-log.ts` | 策略评估日志 |

### 消息

| 模块 | composable | service | 说明 |
|---|---|---|---|
| 内部消息 | `internal-message.ts` | `internal-message.ts` | 站内信 |

---

## 通用枚举工具（shared.ts）

提供全局通用的状态映射函数，可直接在模板中使用：

```ts
import { enableBoolToColor, enableBoolToName, statusToColor, statusToName } from "@/api";

// 启用/禁用
enableBoolToColor(true);    // "#52C41A"
enableBoolToName(false);    // "禁用"

// ON/OFF 状态
statusToColor("ON");        // "#52C41A"
statusToName("OFF");        // "已关闭"

// HTTP 方法列表
import { methodList } from "@/api";
// [{ value: "GET", label: "GET" }, { value: "POST", ... }]
```

---

## 如何新增一个业务模块的 API

假设后端新增了 `NotificationService`：

### 第 1 步：确认 generated 层已有类型

确保 protobuf 已重新生成，`#/api/generated/admin/service/v1` 中包含：
- `notificationservicev1_*` 类型
- `createNotificationServiceClient` 工厂函数

### 第 2 步：创建 service 层

新建 `src/api/service/notification.ts`：

```ts
import { createNotificationServiceClient } from "#/api/generated/admin/service/v1";
import { type PaginationQuery, requestApi } from "#/transport/rest";

let _instance: ReturnType<typeof createNotificationServiceClient> | null = null;

function getNotificationService() {
  if (!_instance) {
    _instance = createNotificationServiceClient(requestApi);
  }
  return _instance;
}

export async function listNotifications(query: PaginationQuery) {
  return getNotificationService().List(query.toRawParams());
}

export async function createNotification(request: any) {
  return getNotificationService().Create(request);
}
```

### 第 3 步：创建 composables 层

新建 `src/api/composables/notification.ts`：

```ts
import { useMutation, useQuery } from "@tanstack/vue-query";
import { type PaginationQuery, makeUpdateMask } from "#/transport/rest";
import { listNotifications, createNotification } from "@/api/service/notification";
import { queryClient } from "@/plugins/vue-query";

// 组件内使用
export function useListNotifications(query: PaginationQuery) {
  return useQuery({
    queryKey: ["listNotifications", query],
    queryFn: () => listNotifications(query),
  });
}

// 组件外使用
export async function fetchListNotifications(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ["listNotifications", params],
    queryFn: () => listNotifications(params),
    retry: 0,
  });
}

// 写操作
export function useCreateNotification() {
  return useMutation({
    mutationFn: (values: Record<string, any>) => createNotification({ data: { ...values } }),
  });
}
```

### 第 4 步：注册导出

在 `service/index.ts` 和 `composables/index.ts` 中各加一行：

```ts
export * from "./notification";
```

---

## Vue Query 配置

全局 QueryClient 配置（`src/plugins/vue-query.ts`）：

| 配置项 | 值 | 说明 |
|---|---|---|
| `staleTime` | `60_000` (60s) | 数据在 60 秒内视为新鲜，不会重新请求 |
| `retry` | `false` | 请求失败不自动重试 |
| `refetchOnWindowFocus` | `false` | 窗口聚焦时不自动刷新 |
| `refetchOnReconnect` | `false` | 网络重连时不自动刷新 |

### Query Key 约定

| 模式 | 示例 | 说明 |
|---|---|---|
| 列表 | `["listUsers", query]` | query 变化时自动重新请求 |
| 详情 | `["getUser", { id: 1 }]` | 按参数区分缓存 |
| 操作 | `["login"]` | Mutation key（用于手动触发） |

---

## 数据流

```
组件 → useListUsers(query)
         │
         ├─ Vue Query 检查缓存
         │    ├─ 命中且未过期 → 返回缓存数据
         │    └─ 未命中 → 调用 queryFn
         │
         └─ queryFn → listUsers(query)
                       │
                       └─ service 层 → UserService.List(params)
                                       │
                                       └─ gRPC Client → requestApi({ path, method, body })
                                                          │
                                                          └─ RequestClient (axios)
                                                              ├─ 注入 Token
                                                              ├─ 注入 Request-ID
                                                              ├─ 注入 Accept-Language
                                                              ├─ 发送 HTTP 请求
                                                              └─ 响应拦截（401 → 刷新Token）
```

---

## 注意事项

1. **不要修改 generated 目录** — 由 protobuf 工具链自动生成，修改会被覆盖
2. **组件内用 `use*`，组件外用 `fetch*`** — `use*` 依赖 Vue setup 上下文，在 Store/路由守卫中使用会报错
3. **更新操作只传变化的字段** — `useUpdate*` 内部会自动生成 `updateMask`，只传需要修改的字段即可
4. **composables 不要直接依赖 generated 代码** — 通过 service 层间接使用，保持依赖方向清晰
5. **Query Key 是响应式的** — 传入 `useQuery` 的参数如果包含 `ref/reactive/computed`，变化时会自动重新请求
6. **错误处理** — Vue Query 的 `error` 已包含错误信息，无需手动 try/catch；`mutation` 的错误通过 `onError` 回调处理
