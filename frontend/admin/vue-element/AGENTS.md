# AGENTS.md — GoWind Admin (Vue3 + Element Plus) 脚手架开发指南

本文件定义此项目的编码约定和架构规范，供 AI Agent 在生成和修改代码时遵循。

## 项目身份

**go-wind-vue3-element-admin** — 基于 Vue 3 + Vite + TypeScript + Element Plus 的后台管理脚手架（Vue3 版 vue-element-admin）。

### 技术栈

| 类别 | 技术 |
|------|------|
| 框架 | Vue 3.5 + TypeScript 5.9 |
| 构建 | Vite 8 (Rolldown) |
| UI | Element Plus 2.x + vxe-table 4.x |
| 状态 | Pinia 3 + @tanstack/vue-query 5 |
| 路由 | vue-router 5 |
| 国际化 | vue-i18n 11 |
| CSS | Tailwind CSS 4 (`@tailwindcss/vite`) + SCSS（原 UnoCSS 已迁移，见 `src/styles/tailwind.css`） |
| 表单 | CRUD 表单用 Element Plus 原生 `:rules` + `formRef.validate()`；`vee-validate` + `zod` 作为依赖存在但 CRUD 未使用 |
| HTTP | axios（封装 gRPC-Web 风格 API） |
| 包管理 | pnpm |

### 核心目录结构

```
src/
├── api/
│   ├── generated/          # gRPC 自动生成代码（禁止手动修改）
│   ├── client.ts           # ApiClient 单例（transport 适配层）
│   └── composables/        # Composable 层：Vue Query hooks + 枚举工具函数（通过 apiClient 调用）
│       └── index.ts        # ⚠️ 新模块需在此添加 export *
├── components/Pro/         # Pro 组件库（配置化 CRUD 页面）
├── core/
│   ├── transport/rest/     # 请求客户端、PaginationQuery、makeUpdateMask、拦截器
│   ├── i18n/               # 国际化核心（$t / t）
│   ├── router/             # 路由工具函数
│   └── access/             # 权限控制
├── pages/app/              # 业务页面（按模块分目录）
├── locales/                # 翻译资源（zh-CN / en-US）
│   └── zh-CN/              # app.json, common.json, core.json, enum.json, preferences.json, routes.json, validation.json, pages/*.json
├── router/routes/modules/app/  # ⭐ 动态路由（自动扫描）
├── constants/index.ts      # 全局常量（DRAWER_WIDTH 等）
├── layouts/                # 布局组件
├── stores/                 # Pinia stores
└── styles/                 # 全局样式
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
- 翻译 key 命名：

| 分类 | 格式 | 示例 |
|------|------|------|
| 页面文本 | `pages.<module>.<field>` | `pages.position.name` |
| 枚举翻译 | `enum.<module>.<field>.<VALUE>` | `enum.position.type.REGULAR` |
| 路由标题 | `routes.<module>.<page>` | `routes.opm.position` |
| 通用文本 | `common.<category>.<key>` | `common.button.edit` |
| 弹窗标题 | `common.modal.create` / `update` | 需传 `{ moduleName }` 参数 |

### 样式
- CSS 变量使用 `--gowind-*` 前缀，避免与 Element Plus `--el-*` 冲突
- 主题色变量存储为 HSL 数值（非 hex 字符串）
- 更新 CSS 变量使用 `style.setProperty()`
- 暗黑模式下文本颜色使用 `var(--el-text-color-*)`，避免硬编码

### 路由
- 动态路由放在 `src/router/routes/modules/app/<module>.ts`
- 顶层路由使用 `Layout` 组件包裹
- `meta.title` 使用 i18n key；`meta.icon` 使用 `lucide:` 前缀
- `meta.authority` 控制权限
- 文件导出 `default` 路由数组，自动被 `import.meta.glob` 扫描

---

## API 两层架构

严格遵循分层：`generated/` + `apiClient` → `composables/`

```
gRPC 生成代码 (generated/)      ← 禁止修改
    ↓ 只导入类型（type import） + apiClient
ApiClient (client.ts)           ← 全局单例，懒加载各服务 Client
    ↓ 调用 apiClient.xxxService.Method()
Composable 层 (composables/)    ← Vue Query hooks + 枚举工具，面向组件
```

### ApiClient 单例 (`src/api/client.ts`)
- 全局唯一实例，通过 `ClientTransport` 适配 axios 请求
- 懒加载属性访问器按需创建各服务 Client（如 `apiClient.userService`、`apiClient.authenticationService`）
- 服务属性命名：驼峰服务名 + `Service` 后缀
- protobuf 重新生成后 `ApiClient` 类自动包含新服务属性

### Composable 层模板 (`src/api/composables/<module>.ts`)

```typescript
import { computed } from "vue";
import { useMutation, useQuery, type UseMutationOptions, type UseQueryOptions } from "@tanstack/vue-query";
import type {
  xxxv1_DeleteXxxRequest, xxxv1_GetXxxRequest,
  xxxv1_ListXxxResponse, xxxv1_Xxx,
} from "@/api/generated/admin/service/v1";
import { makeUpdateMask, type PaginationQuery } from "@/core/transport/rest";
import { apiClient } from "@/api/client";
import { queryClient } from "@/plugins/vue-query";
import { i18n } from "@/core/i18n";

const t = i18n.global.t;

// 列表 Hook（响应式，组件 setup 中使用）
export function useListXxxs(query: PaginationQuery, options?: UseQueryOptions<xxxv1_ListXxxResponse, Error>) {
  return useQuery({
    queryKey: ["listXxxs", query],
    queryFn: () => apiClient.xxxService.List(query.toRawParams()),
    ...options,
  });
}

// 列表查询（非 Hook，用于事件处理、手动调用）
export async function fetchListXxxs(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ["listXxxs", params],
    queryFn: () => apiClient.xxxService.List(params.toRawParams()),
    retry: 0,
  });
}

// 创建 — 注意 { data: {...} } 包裹
export function useCreateXxx(options?: UseMutationOptions<{}, Error, Record<string, any>>) {
  return useMutation({
    mutationFn: (values) => apiClient.xxxService.Create({ data: { ...values } as xxxv1_Xxx }),
    ...options,
  });
}

// 更新 — 必须使用 makeUpdateMask 生成字段掩码
export function useUpdateXxx(options?: UseMutationOptions<{}, Error, { id: number; values: Record<string, any> }>) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      apiClient.xxxService.Update({
        id,
        data: { ...values },
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

// 删除
export function useDeleteXxx(options?: UseMutationOptions<{}, Error, xxxv1_DeleteXxxRequest>) {
  return useMutation({ mutationFn: (req) => apiClient.xxxService.Delete(req), ...options });
}

// 枚举列表 — computed + i18n
export const xxxStatusList = computed(() => [
  { value: "ON", label: t("enum.xxx.status.ON") },
  { value: "OFF", label: t("enum.xxx.status.OFF") },
]);
export function xxxStatusToName(status: string) {
  const matched = xxxStatusList.value.find((item) => item.value === status);
  return matched ? matched.label : "";
}
// 语义色只走 Element Plus 的 tag type（TagType 联合类型定义在 src/api/composables/shared.ts）。
// 本端没有任何 *ToColor / *_COLOR_MAP 十六进制写法（2026-09-25 实测 0 处），
// 颜色由 effect="plain" + _dark-mode.scss 的柔化路径统一产出，别在页面里写死色值。
const XXX_STATUS_TAG_TYPE_MAP: Record<string, TagType> = { ON: "success", OFF: "info", DEFAULT: "info" };
export function xxxStatusToType(status: string): TagType {
  return XXX_STATUS_TAG_TYPE_MAP[status] || XXX_STATUS_TAG_TYPE_MAP.DEFAULT;
}
```

参照实现：`src/api/composables/internal-message.ts:251-295`（list + ToName + ToType 三件套）、
`src/api/composables/shared.ts:63-79`（全局 `statusList` / `statusToName` / `statusToType`）。
全仓此类函数 33 个，命名固定为 `xxxToName` / `xxxToType`（`internal-message.ts` 里的
`internalMessageStatusLabel` 是个历史别名，新代码别跟）。

**Composable 层规则：**
- 从 `generated/` **只导入类型**（`type` import），运行时调用通过 `apiClient`
- 列表查询直接传 `query.toRawParams()`（自动设置 `sorting/offset/limit/token/filter/filterExpr` 为 `undefined`）
- 创建 mutation 参数必须用 `{ data: {...} }` 包裹（gRPC 约定）
- 更新 mutation 必须使用 `makeUpdateMask` 生成字段掩码
- queryKey 全局唯一，格式为 `["操作名", 参数]`
- 删除 API 参数字段注意是 `ids` 还是 `id`（需查看生成类型）
- 创建文件后，在 `src/api/composables/index.ts` 中追加：`export * from "./xxx";`

---

## Pro 组件库参考

### 组件总览

| 组件 | 用途 | 导入方式 |
|------|------|----------|
| `ProPage` | 页面编排（搜索+工具栏+表格+分页） | `import ProPage from "@/components/Pro/ProPage/index.vue"` |
| `ProForm` | 动态表单 | 独立使用 |
| `ProFileSelect` | 文件选择器 | 独立使用 |

统一导出：`import { ProPage, useProModal, injectProModalApi } from "@/components/Pro"`

### ProPageConfig 关键配置

```typescript
interface ProPageConfig<T = any, Q = any> {
  engine?: "vxe" | "element";     // 表格引擎，默认 "vxe"
  rowKey?: string;                 // 行唯一标识
  search?: {
    fields?: ProFormField[];       // 搜索字段
    grid?: boolean | "left" | "right";  // 网格布局
  };
  table: {
    columns: ProTableColumn<T>[];  // 列配置（必填）
    listAction: ListAction<T, Q>;  // 列表数据请求（必填）
    deleteAction?: (ids: string) => Promise<any>;  // 删除
    toolbarRight?: Array<...>;     // 右侧工具栏（如 ["add"]）
    defaultToolbar?: Array<...>;   // 默认工具栏
    pagination?: boolean;          // 是否分页，默认 true
  };
  modal?: {
    component?: "dialog" | "drawer";
    fields: ProFormField<T>[];
    submitAction?: (data: T) => Promise<any>;
  };
}
```

### ProTableColumn 常用列配置

| 用法 | 配置 |
|------|------|
| 序号列 | `{ type: "index", label, width: 60 }` |
| 普通文本列 | `{ prop, label, minWidth }` |
| 数字列（右对齐） | `{ prop, label, width, align: "right" }` |
| 日期列 | `{ prop, label, cellType: "date", dateFormat: "YYYY-MM-DD HH:mm:ss" }` |
| 自定义插槽列 | `{ prop, label, slotName: "status" }` |
| 开关列 | `{ prop, cellType: "switch", activeValue, inactiveValue }` |
| 操作列 | `{ prop: "action", fixed: "right", cellType: "tool", buttons: [...] }` |

操作列 buttons：`{ name, label, icon: "lucide:pen-line", attrs?: { type: "danger" } }`

### ProFormField 搜索字段类型

`type` 取值：`input` | `textarea` | `select` | `radio` | `checkbox` | `switch` | `date-picker` | `input-number` | `tree-select` | `api-tree-select` | `custom`

```typescript
// 输入框
{ type: "input", label: $t("pages.xxx.name"), field: "name", attrs: { placeholder: $t("common.placeholder.input"), clearable: true } }

// 下拉（本地选项）
{ type: "select", label: $t("common.table.status"), field: "status", attrs: { clearable: true }, options: statusList.value }

// 树形选择（异步加载）
{
  type: "tree-select", label: $t("pages.xxx.org"), field: "orgId",
  attrs: { clearable: true, filterable: true, "default-expand-all": true, nodeKey: "id", props: { label: "name", value: "id", children: "children" } },
  initFn: async (item) => {
    const result = await fetchListXxx(new PaginationQuery({ formValues: { status: "ON" } }));
    item.attrs.data = result.items || [];
  },
}
```

### 组件规范
- `ElDrawer` 必须设置 `:append-to-body="true"` 和 `:destroy-on-close="true"`
- `ElDialog` 的 `appendTo` 属性是字符串选择器（非布尔值）
- `ElTreeSelect` 的 `value` 不接受 `undefined`，用 `null` 或不设初始值
- 数字列必须右对齐 (`align: "right"`)
- 抽屉宽度统一使用常量 `DRAWER_WIDTH`（来自 `@/constants`）

---

## 新建 CRUD 模块清单

创建新业务模块时，按以下顺序处理 **9 个文件**（新建 6 个、编辑 3 处既有文件；以 product 为例）：

| # | 文件 | 用途 |
|---|------|------|
| 1 | `src/api/composables/product.ts` | Vue Query hooks + 枚举（通过 apiClient 调用） |
| 2 | `src/api/composables/index.ts` | 追加 `export *` |
| 3 | `src/locales/zh-CN/pages/product.json` | 中文翻译 |
| 4 | `src/locales/en-US/pages/product.json` | 英文翻译 |
| 5 | `src/locales/zh-CN/enum.json` | 追加枚举翻译 |
| 6 | `src/locales/zh-CN/routes.json` | 追加路由标题 |
| 7 | `src/router/routes/modules/app/product.ts` | 路由配置 |
| 8 | `src/pages/app/product/index.vue` | 列表页（ProPage 配置） |
| 9 | `src/pages/app/product/product-drawer.vue` | 抽屉表单（ProModal + ElForm `:rules` + useDrawerForm，`defineExpose({ open })`） |

### 路由模板

```typescript
import type { RouteRecordRaw } from "vue-router";
import { Layout } from "@/layouts";

const routes: RouteRecordRaw[] = [
  {
    path: "/product",
    name: "ProductManagement",
    component: Layout,
    redirect: "/product/list",
    meta: {
      order: 5000,                          // 菜单排序（数字越大越靠后）
      icon: "lucide:package",               // Iconify lucide 图标
      title: "routes.product.moduleName",   // i18n key
      authority: ["sys:product_admin"],     // 权限标识
    },
    children: [
      {
        path: "list",
        name: "ProductList",
        meta: { title: "routes.product.list", authority: ["sys:product_admin"] },
        component: () => import("@/pages/app/product/index.vue"),
      },
    ],
  },
];
export default routes;
```

### 列表页骨架 (`index.vue`)

脚手架采用 **`defineExpose` + `ref` 模式**（而非 useProModal 连接模式）连接列表页与抽屉：

```vue
<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig" @add="handleAdd" @edit="handleEdit">
      <template #status="scope: any">
        <ElTag size="small" round effect="plain" :type="productStatusToType(scope.row.status)">
          {{ productStatusToName(scope.row.status) }}
        </ElTag>
      </template>
    </ProPage>
    <ProductDrawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from "vue";
import { ElTag } from "element-plus";
import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import ProductDrawer from "./product-drawer.vue";
import { fetchListProducts, useDeleteProduct, productStatusToName, productStatusToType, productStatusList } from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/core/i18n";

const { mutateAsync: deleteProduct } = useDeleteProduct();
const pageRef = ref();
const drawerRef = ref();

const pageConfig = computed<ProPageConfig>(() => ({
  skeleton: true,
  search: {
    grid: true,
    fields: [
      { type: "input", label: $t("pages.product.name"), field: "name", attrs: { placeholder: $t("common.placeholder.input"), clearable: true } },
      { type: "select", label: $t("common.table.status"), field: "status", attrs: { clearable: true }, options: productStatusList.value },
    ],
  },
  table: {
    listAction: async (query: any) => {
      const { page, pageSize, ...queryParams } = query;
      const result = await fetchListProducts(new PaginationQuery({ paging: { page: page || 1, pageSize: pageSize || 10 }, formValues: queryParams }));
      return { items: result.items || [], total: result.total || 0 };
    },
    deleteAction: async (ids: string) => { await deleteProduct({ id: ids as any }); },
    toolbarRight: ["add"],
    defaultToolbar: ["refresh", "filter"],
    columns: [
      { type: "index", label: $t("common.table.seq"), width: 60 },
      { prop: "name", label: $t("pages.product.name"), minWidth: 120 },
      { prop: "status", label: $t("common.table.status"), minWidth: 100, slotName: "status" },
      { prop: "createdAt", label: $t("common.table.createdAt"), minWidth: 160, cellType: "date", dateFormat: "YYYY-MM-DD HH:mm:ss" },
      { prop: "action", label: $t("common.table.action"), fixed: "right", width: 150, cellType: "tool", buttons: [
        { name: "edit", label: $t("common.button.edit"), icon: "lucide:pen-line" },
        { name: "delete", label: $t("common.button.delete"), icon: "lucide:trash-2", attrs: { type: "danger" } },
      ] },
    ],
  },
}));

function handleAdd() { drawerRef.value?.open({ create: true }); }
function handleEdit(row: any) { drawerRef.value?.open({ create: false, row }); }
function handleSuccess() { pageRef.value?.refresh(); }
</script>
```

### 抽屉弹窗骨架 (`product-drawer.vue`)

**默认走 `ProModal` + `useDrawerForm`**（`src/components/Pro/composables/useDrawerForm.ts`）：它托管 `visible`/`isCreate`/`currentId`/`pageLoading`/`submitLoading`/`formData`/`title`/`drawerWidth`、`open`/`close`/`resetForm`，以及 `handleSubmit`（校验 → create/update → 成功提示 → onSuccess → 关闭，失败兜底提示）。抽屉外壳统一是 `ProModal`（配 `config.component: 'drawer'`；2026-09-25 实测 33 个文件在用它，`grep -rl '<ProModal' src | wc -l` 可复测，`src/pages/` 下已无裸 `ElDrawer` 骨架），业务表单抽屉里 9 个已用 `useDrawerForm`，列表页只负责 `defineExpose({ open: drawer.open })`：

```vue
<template>
  <ProModal
    v-model:visible="drawer.visible.value"
    :title="drawer.title.value"
    :loading="drawer.pageLoading.value"
    :config="{ component: 'drawer', drawer: { size: drawer.drawerWidth, closeOnClickModal: false } }"
  >
    <ElForm ref="formRef" :model="drawer.formData" :rules="formRules" label-width="120px" class="drawer-form">
      <ElDivider content-position="left">{{ $t("common.section.basic") }}</ElDivider>
      <ElFormItem :label="$t('pages.product.name')" prop="name">
        <ElInput v-model="drawer.formData.name" :placeholder="$t('common.placeholder.input')" clearable />
      </ElFormItem>
      <!-- ...其余字段 -->
    </ElForm>
    <template #footer>
      <div class="drawer-footer">
        <ElButton @click="drawer.close">{{ $t("common.button.cancel") }}</ElButton>
        <ElButton type="primary" :loading="drawer.submitLoading.value"
          @click="drawer.handleSubmit(formRef, () => emit('success'))">
          {{ $t("common.button.confirm") }}
        </ElButton>
      </div>
    </template>
  </ProModal>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import ProModal from "@/components/Pro/ProModal/index.vue";
import { useDrawerForm } from "@/components/Pro/composables/useDrawerForm";
import { useCreateProduct, useUpdateProduct } from "@/api/composables";
import { $t } from "@/core/i18n";

const emit = defineEmits<{ success: [] }>();
const { mutateAsync: createProduct } = useCreateProduct();
const { mutateAsync: updateProduct } = useUpdateProduct();
const formRef = ref();

const drawer = useDrawerForm({
  moduleKey: "pages.product.moduleName",              // 标题自动取 common.modal.create/update
  defaults: { name: "", status: "ON" /* ... */ },
  createFn: createProduct,
  updateFn: (id, values) => updateProduct({ id, values }),
  asyncSetup: async () => { /* 可选：打开时加载下拉数据 */ },
});

// 校验走 Element Plus 原生 rules（不是 vee-validate）
const formRules = {
  name: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  status: [{ required: true, message: $t("common.validation.selectRequired"), trigger: "change" }],
};

defineExpose({ open: drawer.open });
</script>
```

**例外**：表单需要自定义转换逻辑（如 `role-drawer.vue` 的树 checkedKeys → 叶子 id 列表、多步流程）时才不用 `useDrawerForm`，改为自管 `visible`/`submitLoading` 手写 `handleSubmit`（外壳仍用 `ProModal`）。此时 catch 里要先判 `if (error !== false)` 再报错——`validate()` 校验失败时 reject 的是 `false`，否则缺字段会误弹"创建失败"。（若确实要裸用 `ElDrawer`，必须带 `:append-to-body="true"` 与 `:destroy-on-close="true"`，见上文「组件规范」。）

---

## PaginationQuery & makeUpdateMask

```typescript
import { PaginationQuery, makeUpdateMask } from "@/core/transport/rest";

new PaginationQuery({
  paging: { page: 1, pageSize: 10 },          // 分页参数
  formValues: { name: "test", status: "ON" }, // 搜索条件
  orderBy: ["-created_at"],                    // 排序（负号=降序）
});
pq.toRawParams();  // 转换为 gRPC API 参数格式

makeUpdateMask(["name", "status"]);  // → "name,status,id"（自动追加 id）
```

## 常用导入速查

```typescript
// API 层
import { apiClient } from "@/api/client";  // Composable 内部使用
import { fetchListXxx, useCreateXxx, useUpdateXxx, useDeleteXxx, xxxStatusList } from "@/api/composables";  // 组件中使用

// Pro 组件
import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";

// 工具
import { PaginationQuery, makeUpdateMask } from "@/core/transport/rest";
import { $t } from "@/core/i18n";
import { DRAWER_WIDTH } from "@/constants";

// 通用枚举（已内置 shared.ts）
import { statusList, statusToName, statusToType, enableList } from "@/api/composables";
```

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
| `defineExpose` 暴露的属性需 `.value` | 父组件通过 `ref` 直接访问暴露的方法 |
| `ElTreeSelect` value 为 undefined | 用 `null` 代替或不设初始值 |
| `IconifyIcon` 组件不显示 | 需显式导入 `import { IconifyIcon } from "@iconify/vue"` |
| 删除 API 字段名混淆 | 参数字段注意是 `ids` 还是 `id`（需查看生成类型） |

## 错误处理铁律（不吞错）

历史教训：静默失败曾让"刷新丢登录"四层叠加 bug 排查数日（每层都被上一层的话术掩盖）。**任何 catch 至少二选一**：

1. `console.error/warn` 带出**原始错误对象**（不是只打固定话术或翻译后文案）；
2. 重新抛出/向上传递。

用户可见的 ElNotification/ElMessage **不等于**日志——通知只有翻译文案，排查要靠控制台里的原始错误。合法的裸 catch 仅限：纯本地 best-effort 兜底（JSON.parse 校验、structuredClone 回退、storage 脏数据清理），且应在注释里写明兜底原因。

已知的"固定话术掩盖真因"高危点：`bootstrap.ts` 静默恢复、`use-auth.ts` 登录/MFA/登出/权限码获取——这些位置的 catch 均已改为带错误对象留痕，改动时不要回退。

## 白屏（router-view 塌空）真因与处置

**规则（照做即可，不必读下面的归档背景）**：`LayoutMain.vue` 的页面过渡**不得改回 `mode="out-in"`**（改同帧交叉淡入淡出）；`app.config.errorHandler` + `router.onError` 观测、动态导入失败的一次性整页刷新自愈、`optimizeDeps.include` 预打包**都别删**；判回归前先重启 dev server，若再现白页按「懒 chunk 解析竞态」方向排查，**不要**再归因 vite 依赖优化/HMR。

「背景（2026-09-08，归档）」已定位真因：**`<transition mode="out-in">` + vue-router 5 懒加载路由的竞态**，与 vite 依赖优化/HMR 无关（此前归因有误）。懒 chunk 在导航期间解析完成时，旧页被直接移除而新页永不挂载，且此后 router-view 整体僵死（零报错、dev/prod 均可复现，dev 因 chunk 解析慢数百倍而高发；连续导航 5~7 次必现，vue 3.5.34/3.5.42 均复现）。修复：`LayoutMain.vue` 去掉 `mode="out-in"` 改同帧交叉淡入淡出（150ms 时长不变）；顺带修复 wrapper 闭包固化 vue-router 5 slot vnode（v5 的 `Component` 是预构建 vnode 而非 v4 的组件定义，闭包缓存会渲染过期 vnode）。验证：连续 ~100 次导航零塌空 + keep-alive 状态保留正常。

仍保留的防御（别删）：`app.config.errorHandler` + `router.onError` 全局观测（[AppErrorHandler]/[RouterError] 前缀）；懒加载路由动态导入失败自动一次性整页刷新自愈；`vite:preloadError` 事件自愈刷新。重依赖（echarts/vxe-table/@tanstack/vue-query）已进 `optimizeDeps.include` 预打包。

**判回归前先重启 dev server**（多文件改动/stash 污染 HMR 状态是常见干扰源）；若再现白页，用连续菜单导航循环复现后按上述方向排查，勿再归因 vite 竞态。

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
