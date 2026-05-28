---
name: gowind-admin-dev
description: GoWind Admin (Vue3 + Element Plus) 后台管理脚手架开发指南。帮助开发者按照项目规范创建 CRUD 模块、编写 API 层（service/composable）、使用 Pro 组件库构建页面、配置路由与国际化。适用于在此脚手架上开发新业务模块、生成标准 CRUD 页面、或理解项目架构约定。
---

# GoWind Admin 开发指南

## 项目概览

**go-wind-vue3-element-admin** — 基于 Vue 3 + Vite + TypeScript + Element Plus 的后台管理模板。

### 技术栈

| 类别 | 技术 |
|------|------|
| 框架 | Vue 3.5 + TypeScript 5.9 |
| 构建 | Vite 8 (Rolldown) |
| UI | Element Plus 2.x + vxe-table 4.x |
| 状态 | Pinia 3 + @tanstack/vue-query 5 |
| 路由 | vue-router 5 |
| 国际化 | vue-i18n 11 |
| CSS | UnoCSS + SCSS |
| 表单 | vee-validate + zod |
| HTTP | axios (封装 gRPC-Web 风格 API) |

### 核心架构

```
src/
├── api/
│   ├── generated/          # gRPC 自动生成代码（勿手动修改）
│   ├── service/            # 服务层：封装 gRPC 客户端调用
│   └── composables/        # Composable 层：Vue Query hooks + 枚举工具
├── components/Pro/         # Pro 组件库（配置化 CRUD）
├── core/
│   ├── transport/rest/     # 请求客户端、PaginationQuery、拦截器
│   ├── i18n/               # 国际化核心
│   ├── router/             # 路由工具
│   └── access/             # 权限控制
├── pages/app/              # 业务页面（按模块分目录）
├── locales/                # 翻译资源（zh-CN / en-US）
├── router/routes/modules/  # 动态路由模块
├── layouts/                # 布局组件
└── stores/                 # Pinia stores
```

---

## 新建 CRUD 模块（开发流程）

创建一个新业务模块需要以下文件，按顺序操作：

### 1. API 服务层 (`src/api/service/<module>.ts`)

```typescript
import { createXxxServiceClient } from "@/api/generated/admin/service/v1";
import { type PaginationQuery, requestApi } from "@/core/transport/rest";

let _instance: ReturnType<typeof createXxxServiceClient> | null = null;

export function getXxxService() {
  if (!_instance) {
    _instance = createXxxServiceClient(requestApi);
  }
  return _instance;
}

export async function listXxx(query: PaginationQuery) {
  const params = query.toRawParams();
  return getXxxService().List({ ...params, sorting: undefined, offset: undefined, limit: undefined, token: undefined, filter: undefined, filterExpr: undefined });
}

export async function createXxx(request: XxxCreateRequest) {
  return getXxxService().Create(request);
}

export async function updateXxx(request: XxxUpdateRequest) {
  return getXxxService().Update(request);
}

export async function deleteXxx(request: { id: number }) {
  return getXxxService().Delete(request);
}
```

**关键规则：**
- 使用单例模式创建 gRPC 服务客户端
- 列表查询使用 `PaginationQuery.toRawParams()` 转换参数
- 创建接口的参数必须用 `{ data: 实际数据 }` 包裹

### 2. API Composable 层 (`src/api/composables/<module>.ts`)

```typescript
import { computed } from "vue";
import { useQuery, useMutation, type UseQueryOptions, type UseMutationOptions } from "@tanstack/vue-query";
import { makeUpdateMask, type PaginationQuery } from "@/core/transport/rest";
import { listXxx, createXxx, updateXxx, deleteXxx } from "@/api/service/xxx";
import { queryClient } from "@/plugins/vue-query";
import { i18n } from "@/core/i18n";

const t = i18n.global.t;

// 列表查询 Hook
export function useListXxx(query: PaginationQuery, options?: UseQueryOptions<XxxListResponse, Error>) {
  return useQuery({ queryKey: ["listXxx", query], queryFn: () => listXxx(query), ...options });
}

// 列表查询（非 Hook，用于 Store/外部调用）
export async function fetchListXxx(params: PaginationQuery) {
  return queryClient.fetchQuery({ queryKey: ["listXxx", params], queryFn: () => listXxx(params), retry: 0 });
}

// 创建
export function useCreateXxx(options?: UseMutationOptions<{}, Error, Record<string, any>>) {
  return useMutation({
    mutationFn: (values) => createXxx({ data: { ...values } as XxxType }),
    ...options,
  });
}

// 更新（使用 makeUpdateMask 生成字段掩码）
export function useUpdateXxx(options?: UseMutationOptions<{}, Error, { id: number; values: Record<string, any> }>) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      updateXxx({ id, data: { ...values }, updateMask: makeUpdateMask(Object.keys(values ?? {})) }),
    ...options,
  });
}

// 删除
export function useDeleteXxx(options?: UseMutationOptions<{}, Error, { id: number }>) {
  return useMutation({ mutationFn: (req) => deleteXxx(req), ...options });
}

// 枚举列表（computed + i18n）
export const xxxStatusList = computed(() => [
  { value: "ACTIVE", label: t("enum.xxx.status.ACTIVE") },
  { value: "INACTIVE", label: t("enum.xxx.status.INACTIVE") },
]);

// 枚举工具函数
export function xxxStatusToName(status: string) {
  return xxxStatusList.value.find((item) => item.value === status)?.label ?? "";
}
```

**关键规则：**
- 每个 composable 文件导出：`use*` Hook + `fetch*` 非 Hook 函数 + 枚举工具
- queryKey 格式：`["操作名", 参数]`
- 创建 mutation 的参数用 `{ data: {...} }` 包裹
- 更新 mutation 必须使用 `makeUpdateMask` 生成字段掩码
- 枚举列表用 `computed(() => [...])` + i18n `t()` 标签
- 在 `src/api/composables/index.ts` 中添加 `export * from "./xxx"`

### 3. 国际化 (`src/locales/zh-CN/pages/xxx.json` + `src/locales/en-US/pages/xxx.json`)

```json
{
  "moduleName": "模块名称",
  "name": "名称",
  "code": "编码",
  "notification": {
    "create_success": "创建成功"
  }
}
```

- 在 `src/locales/zh-CN/enum.json` 中添加枚举翻译
- 在 `src/locales/zh-CN/routes.json` 中添加路由标题
- 翻译文件按页面粒度拆分，放在 `pages/` 目录下

### 4. 路由 (`src/router/routes/modules/app/<module>.ts`)

```typescript
import type { RouteRecordRaw } from "vue-router";
import { Layout } from "@/layouts";

const routes: RouteRecordRaw[] = [
  {
    path: "/xxx",
    name: "XxxManagement",
    component: Layout,
    redirect: "/xxx/list",
    meta: {
      order: 3000,
      icon: "lucide:xxx-icon",
      title: "routes.xxx.moduleName",
      authority: ["sys:xxx_admin"],
    },
    children: [
      {
        path: "list",
        name: "XxxList",
        meta: {
          order: 1,
          icon: "lucide:list",
          title: "routes.xxx.list",
          authority: ["sys:xxx_admin"],
        },
        component: () => import("@/pages/app/xxx/xxx/index.vue"),
      },
    ],
  },
];

export default routes;
```

**关键规则：**
- 使用 `RouteRecordRaw` 类型（非 `AppRoute`）
- 顶层路由使用 `Layout` 组件包裹
- `meta.title` 使用 i18n key（`routes.xxx.xxx`）
- `meta.authority` 控制权限
- `meta.icon` 使用 `lucide:` 前缀的 UnoCSS 图标
- 文件导出 `default` 路由数组，自动被 `import.meta.glob` 扫描

### 5. 页面组件 (`src/pages/app/xxx/xxx/index.vue`)

```vue
<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig" @add="handleAdd" @edit="handleEdit">
      <template #status="scope: any">
        <ElTag size="small" effect="dark" round :color="xxxStatusToColor(scope.row.status)">
          {{ xxxStatusToName(scope.row.status) }}
        </ElTag>
      </template>
    </ProPage>
    <ConnectedDrawer />
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from "vue";
import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import { useProModal } from "@/components/Pro";
import XxxDrawer from "./xxx-drawer.vue";
import { fetchListXxx, useDeleteXxx, xxxStatusToName, xxxStatusToColor } from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/core/i18n";

const { mutateAsync: deleteXxx } = useDeleteXxx();
const pageRef = ref();

// useProModal 连接 Drawer 组件
const [ConnectedDrawer, modalApi] = useProModal({
  connectedComponent: XxxDrawer,
  onOpenChange(isOpen) {
    if (!isOpen) pageRef.value?.refresh();
  },
});

const pageConfig = computed<ProPageConfig>(() => ({
  search: {
    fields: [
      { type: "input", label: $t("pages.xxx.name"), field: "name", attrs: { placeholder: $t("common.placeholder.input"), clearable: true } },
    ],
    grid: true,
  },
  table: {
    listAction: async (query: any) => {
      const { page, pageSize, ...queryParams } = query;
      const result = await fetchListXxx(new PaginationQuery({ paging: { page: page || 1, pageSize: pageSize || 10 }, formValues: queryParams }));
      return { items: result.items || [], total: result.total || 0 };
    },
    deleteAction: async (ids: string) => { await deleteXxx({ id: ids as any }); },
    toolbarRight: ["add"],
    defaultToolbar: ["refresh", "filter"],
    columns: [
      { type: "index", label: $t("common.table.seq"), width: 60 },
      { prop: "name", label: $t("pages.xxx.name"), minWidth: 120 },
      { prop: "status", label: $t("common.table.status"), minWidth: 100, slotName: "status" },
      { prop: "createdAt", label: $t("common.table.createdAt"), minWidth: 160, cellType: "date", dateFormat: "YYYY-MM-DD HH:mm:ss" },
      { prop: "action", label: $t("common.table.action"), fixed: "right", width: 150, cellType: "tool", buttons: [
        { name: "edit", label: $t("common.button.edit") },
        { name: "delete", label: $t("common.button.delete"), attrs: { type: "danger" } },
      ] },
    ],
  },
}));

function handleAdd() { modalApi.open({ create: true }); }
function handleEdit(row: any) { modalApi.open({ create: false, row }); }
</script>

<style lang="scss" scoped>
.app-container { padding: 20px; width: 100%; min-width: 0; flex-shrink: 0; }
</style>
```

### 6. 弹窗组件 (`src/pages/app/xxx/xxx/xxx-drawer.vue`)

使用 `useProModal` + `connectedComponent` 模式：

- 列表页通过 `useProModal({ connectedComponent: Drawer })` 创建弹窗连接
- 弹窗组件通过 `injectProModalApi()` 获取 modalApi
- `modalApi.getData()` 获取传入数据，`modalApi.open()` / `modalApi.close()` 控制开关
- `ElDrawer` 必须设置 `append-to-body` 和 `destroy-on-close`

详见 [reference.md](reference.md) 和 [examples.md](examples.md)。

---

## 编码规范

### 通用规范
- TypeScript 接口命名**不使用 I 前缀**（如 `User` 而非 `IUser`）
- 所有用户可见文本必须通过 `$t()` / `t()` 国际化，禁止硬编码中文
- 使用 `$t()` 用于组件模板和 `computed` 中，`t()` 用于非响应式场景
- 路由类型使用 `RouteRecordRaw`
- 包管理器使用 **pnpm**

### API 层规范
- **composables 禁止直接依赖 gRPC 生成代码的实现细节**，只导入类型
- 创建类 API 调用时请求参数必须用 `{ data: 实际数据 }` 包裹
- 更新操作必须使用 `makeUpdateMask` 生成字段掩码
- 删除 API 参数字段注意是 `ids` 还是 `id`（需查看生成类型）
- queryKey 全局唯一，格式为 `["操作名", 参数]`

### 组件规范
- `ElDrawer` 必须设置 `:append-to-body="true"`
- `ElDialog` 的 `appendTo` 属性是字符串选择器（非布尔值）
- `ElTreeSelect` 的 `value` 不接受 `undefined`，用 `null` 代替
- 暗黑模式下文本颜色使用 Element Plus CSS 变量（`var(--el-text-color-*)`），避免硬编码
- 自定义主题色需同步生成 Element Plus CSS 变量

### 样式规范
- CSS 变量使用 `--gowind-*` 前缀，避免与 Element Plus `--el-*` 冲突
- 主题色变量存储为 HSL 数值（非 hex 字符串）
- 更新 CSS 变量使用 `style.setProperty()`，避免被覆盖
- 菜单折叠时文字通过 `overflow: hidden` 隐藏

---

## 常见陷阱

| 陷阱 | 正确做法 |
|------|----------|
| gRPC 创建接口直接传对象 | 必须用 `{ data: {...} }` 包裹 |
| `isFunction` 做类型窄化 | 使用 `typeof fn === "function"` |
| PowerShell 中用 `&&` 连接命令 | 使用分号 `;` |
| `ElLink` underline 传 boolean | 已废弃 boolean，使用字符串 |
| vue-i18n `$te` 传 ns 选项对象 | `$te` 不支持 ns 选项，用完整 key |
| 路由数组 map+sort 后类型丢失 | 断言为 `RouteRecordRaw[]` |
| `defineExpose` 暴露的属性通过 ref 访问不到 | 直接访问，无需 `.value` |

更多详细参考请查阅 [reference.md](reference.md) 和 [examples.md](examples.md)。
