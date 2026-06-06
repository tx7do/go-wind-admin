---
name: gowind-admin-dev
description: GoWind Admin (Vue3 + Element Plus) 后台管理脚手架二开指南。帮助二开人员理解脚手架架构约定，按照标准流程创建新业务 CRUD 模块、接入 gRPC API、使用 Pro 组件库构建页面、配置路由与国际化。所有示例均来自脚手架内置模块的真实代码。
---

# GoWind Admin 二开发指南

> 本文档面向 **基于 GoWind Admin 脚手架进行二次开发** 的开发者。如果你要新建一个业务模块，请按以下流程操作。

## 项目概览

**go-wind-vue3-element-admin** — 基于 Vue 3 + Vite + TypeScript + Element Plus 的后台管理脚手架。

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

### 目录结构（二开人员关注点）

```
src/
├── api/
│   ├── generated/          # gRPC 自动生成代码（禁止手动修改）
│   ├── service/            # 服务层：封装 gRPC 客户端单例
│   │   └── index.ts        # ⚠️ 新模块需在此添加 export *
│   └── composables/        # Composable 层：Vue Query hooks + 枚举工具
│       └── index.ts        # ⚠️ 新模块需在此添加 export *
├── components/Pro/         # Pro 组件库（配置化 CRUD）
├── core/
│   ├── transport/rest/     # 请求客户端、PaginationQuery、makeUpdateMask
│   ├── i18n/               # 国际化核心（$t / t）
│   ├── router/             # 路由工具
│   └── access/             # 权限控制
├── pages/app/              # ⭐ 业务页面（按模块分目录）
├── locales/                # 翻译资源（zh-CN / en-US）
│   ├── zh-CN/
│   │   ├── common.json     # 通用文本（按钮、表单、表格等）
│   │   ├── enum.json       # ⚠️ 新枚举翻译在此追加
│   │   ├── routes.json     # ⚠️ 新路由标题在此追加
│   │   └── pages/          # ⚠️ 每个模块一个 JSON 文件
│   └── en-US/              # 英文翻译（同结构）
├── router/routes/modules/app/  # ⭐ 动态路由（自动扫描）
├── constants/index.ts      # 全局常量（DRAWER_WIDTH 等）
└── styles/                 # 全局样式
```

---

## 新建业务模块（完整流程）

> 以创建一个 **「产品管理 (product)」** 模块为例，需创建以下文件（按顺序）：

### 总览：9 个文件

| # | 文件 | 用途 |
|---|------|------|
| 1 | `src/api/service/product.ts` | gRPC 客户端封装 |
| 2 | `src/api/service/index.ts` | 追加 `export *` |
| 3 | `src/api/composables/product.ts` | Vue Query hooks + 枚举 |
| 4 | `src/api/composables/index.ts` | 追加 `export *` |
| 5 | `src/locales/zh-CN/pages/product.json` | 页面翻译 |
| 6 | `src/locales/en-US/pages/product.json` | 英文翻译 |
| 7 | `src/locales/zh-CN/enum.json` | 追加枚举翻译 |
| 8 | `src/locales/zh-CN/routes.json` | 追加路由标题 |
| 9 | `src/router/routes/modules/app/product.ts` | 路由配置 |
| 10 | `src/pages/app/product/index.vue` | 列表页 |
| 11 | `src/pages/app/product/product-drawer.vue` | 新增/编辑抽屉 |

---

### 步骤 1：API 服务层 (`src/api/service/product.ts`)

从 `generated/` 中找到对应的 gRPC Client 工厂函数和请求类型：

```typescript
// src/api/service/product.ts
import {
  createProductServiceClient,
  type productv1_CreateProductRequest,
  type productv1_DeleteProductRequest,
  type productv1_GetProductRequest,
  type productv1_UpdateProductRequest,
} from "@/api/generated/admin/service/v1";
import { type PaginationQuery, requestApi } from "@/core/transport/rest";

// 单例模式
let _instance: ReturnType<typeof createProductServiceClient> | null = null;

export function getProductService() {
  if (!_instance) {
    _instance = createProductServiceClient(requestApi);
  }
  return _instance;
}

// 列表查询 — PaginationQuery.toRawParams() 自动处理分页、排序、过滤
export async function listProducts(query: PaginationQuery) {
  const params = query.toRawParams();
  return getProductService().List(params);
}

// 单条查询
export async function getProduct(request: productv1_GetProductRequest) {
  return getProductService().Get(request);
}

// 创建
export async function createProduct(request: productv1_CreateProductRequest) {
  return getProductService().Create(request);
}

// 更新
export async function updateProduct(request: productv1_UpdateProductRequest) {
  return getProductService().Update(request);
}

// 删除
export async function deleteProduct(request: productv1_DeleteProductRequest) {
  return getProductService().Delete(request);
}
```

**⚠️ 重要规则：**
- 单例模式创建 gRPC 客户端（闭包变量 + 懒初始化）
- 列表查询直接传 `query.toRawParams()`，它会自动设置 `sorting/offset/limit/token/filter/filterExpr` 为 `undefined`
- 创建文件后，在 `src/api/service/index.ts` 中追加：`export * from "./product";`

---

### 步骤 2：API Composable 层 (`src/api/composables/product.ts`)

基于 service 层封装 Vue Query hooks：

```typescript
// src/api/composables/product.ts
import { computed } from "vue";
import {
  useMutation, useQuery,
  type UseMutationOptions, type UseQueryOptions,
} from "@tanstack/vue-query";
import type {
  productv1_DeleteProductRequest,
  productv1_GetProductRequest,
  productv1_ListProductResponse,
  productv1_Product,
} from "@/api/generated/admin/service/v1";
import { makeUpdateMask, type PaginationQuery } from "@/core/transport/rest";
import {
  listProducts, getProduct, createProduct, updateProduct, deleteProduct,
} from "@/api/service/product";
import { queryClient } from "@/plugins/vue-query";
import { i18n } from "@/core/i18n";

const t = i18n.global.t;

// ==============================
// Vue Query Hooks
// ==============================

// 列表查询 Hook（响应式，组件 setup 中使用）
export function useListProducts(
  query: PaginationQuery,
  options?: UseQueryOptions<productv1_ListProductResponse, Error>
) {
  return useQuery({
    queryKey: ["listProducts", query],
    queryFn: () => listProducts(query),
    ...options,
  });
}

// 列表查询（非 Hook，用于事件处理、手动调用）
export async function fetchListProducts(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ["listProducts", params],
    queryFn: () => listProducts(params),
    retry: 0,
  });
}

// 创建 — 注意 { data: {...} } 包裹
export function useCreateProduct(options?: UseMutationOptions<{}, Error, Record<string, any>>) {
  return useMutation({
    mutationFn: (values) => createProduct({ data: { ...values } as productv1_Product }),
    ...options,
  });
}

// 更新 — 必须使用 makeUpdateMask 生成字段掩码
export function useUpdateProduct(
  options?: UseMutationOptions<{}, Error, { id: number; values: Record<string, any> }>
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      updateProduct({
        id,
        data: { ...values },
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

// 删除
export function useDeleteProduct(
  options?: UseMutationOptions<{}, Error, productv1_DeleteProductRequest>
) {
  return useMutation({ mutationFn: (req) => deleteProduct(req), ...options });
}

// ==============================
// 枚举工具函数
// ==============================

// 枚举列表 — computed + i18n
export const productStatusList = computed(() => [
  { value: "ON", label: t("enum.status.ON") },
  { value: "OFF", label: t("enum.status.OFF") },
]);

// 枚举值 → 显示名称
export function productStatusToName(status: string) {
  const values = productStatusList.value;
  const matchedItem = values.find((item) => item.value === status);
  return matchedItem ? matchedItem.label : "";
}

// 枚举值 → 颜色（支持明暗模式）
const PRODUCT_STATUS_COLOR_MAP: Record<string, string> = {
  ON: "#52C41A",
  OFF: "#8C8C8C",
  DEFAULT: "#C9CDD4",
};
export function productStatusToColor(status: string) {
  return PRODUCT_STATUS_COLOR_MAP[status] || PRODUCT_STATUS_COLOR_MAP.DEFAULT;
}
```

**⚠️ 重要规则：**
- composable 层**禁止直接依赖 gRPC 实现细节**，只导入类型（`type` import）
- 创建 mutation 参数必须用 `{ data: {...} }` 包裹
- 更新 mutation 必须使用 `makeUpdateMask` 生成字段掩码
- 枚举列表用 `computed(() => [...])` + `t()` 标签
- 创建文件后，在 `src/api/composables/index.ts` 中追加：`export * from "./product";`

---

### 步骤 3：国际化翻译

#### 3.1 页面翻译 (`src/locales/zh-CN/pages/product.json`)

```json
{
  "moduleName": "产品",
  "name": "产品名称",
  "code": "产品编码",
  "category": "产品分类",
  "price": "价格",
  "status": "状态",
  "description": "产品描述",
  "remark": "备注"
}
```

#### 3.2 英文翻译 (`src/locales/en-US/pages/product.json`)

```json
{
  "moduleName": "Product",
  "name": "Product Name",
  "code": "Product Code",
  "category": "Category",
  "price": "Price",
  "status": "Status",
  "description": "Description",
  "remark": "Remark"
}
```

#### 3.3 枚举翻译（在 `src/locales/zh-CN/enum.json` 中追加）

```json
{
  "product": {
    "status": {
      "ON": "上架",
      "OFF": "下架"
    }
  }
}
```

#### 3.4 路由标题（在 `src/locales/zh-CN/routes.json` 中追加）

```json
{
  "product": {
    "moduleName": "产品管理",
    "list": "产品列表"
  }
}
```

---

### 步骤 4：路由 (`src/router/routes/modules/app/product.ts`)

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
      icon: "lucide:package",               // UnoCSS lucide 图标
      title: "routes.product.moduleName",   // i18n key
      authority: ["sys:product_admin"],     // 权限标识
    },
    children: [
      {
        path: "list",
        name: "ProductList",
        meta: {
          order: 1,
          icon: "lucide:list",
          title: "routes.product.list",
          authority: ["sys:product_admin"],
        },
        component: () => import("@/pages/app/product/index.vue"),
      },
    ],
  },
];

export default routes;
```

**⚠️ 路由规则：**
- 类型使用 `RouteRecordRaw`（不是 `AppRoute`）
- 顶层路由使用 `Layout` 组件包裹
- `meta.title` 使用 i18n key
- `meta.icon` 使用 `lucide:` 前缀
- 文件导出 `default` 路由数组，自动被 `import.meta.glob` 扫描

---

### 步骤 5：列表页 (`src/pages/app/product/index.vue`)

```vue
<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig" @add="handleAdd" @edit="handleEdit">
      <!-- 自定义插槽：状态列 -->
      <template #status="scope: any">
        <ElTag size="small" effect="dark" round :color="productStatusToColor(scope.row.status)">
          {{ productStatusToName(scope.row.status) }}
        </ElTag>
      </template>
    </ProPage>

    <!-- 新增/编辑抽屉 -->
    <ProductDrawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from "vue";
import { ElTag } from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import ProductDrawer from "./product-drawer.vue";

import {
  fetchListProducts,
  useDeleteProduct,
  productStatusToName,
  productStatusToColor,
  productStatusList,
} from "@/api/composables";
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
      {
        type: "input",
        label: $t("pages.product.name"),
        field: "name",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "select",
        label: $t("common.table.status"),
        field: "status",
        attrs: { placeholder: $t("common.placeholder.select"), clearable: true },
        options: productStatusList.value,
      },
    ],
  },
  table: {
    listAction: async (query: any) => {
      const { page, pageSize, ...queryParams } = query;
      const result = await fetchListProducts(
        new PaginationQuery({
          paging: { page: page || 1, pageSize: pageSize || 10 },
          formValues: queryParams,
        })
      );
      return { items: result.items || [], total: result.total || 0 };
    },
    deleteAction: async (ids: string) => {
      await deleteProduct({ id: ids as any });
    },
    toolbar: [],
    toolbarRight: ["add"],
    defaultToolbar: ["refresh", "filter"],
    columns: [
      { type: "index", label: $t("common.table.seq"), width: 60 },
      { prop: "name", label: $t("pages.product.name"), minWidth: 120 },
      { prop: "code", label: $t("pages.product.code"), minWidth: 120 },
      { prop: "price", label: $t("pages.product.price"), minWidth: 100, align: "right" },
      {
        prop: "status",
        label: $t("common.table.status"),
        minWidth: 100,
        slotName: "status",
      },
      {
        prop: "createdAt",
        label: $t("common.table.createdAt"),
        minWidth: 160,
        cellType: "date",
        dateFormat: "YYYY-MM-DD HH:mm:ss",
      },
      { prop: "remark", label: $t("common.table.remark"), minWidth: 150 },
      {
        prop: "action",
        label: $t("common.table.action"),
        fixed: "right",
        width: 150,
        cellType: "tool",
        buttons: [
          { name: "edit", label: $t("common.button.edit"), icon: "lucide:pen-line" },
          { name: "delete", label: $t("common.button.delete"), icon: "lucide:trash-2", attrs: { type: "danger" } },
        ],
      },
    ],
  },
}));

function handleAdd() {
  drawerRef.value?.open({ create: true });
}

function handleEdit(row: any) {
  drawerRef.value?.open({ create: false, row });
}

function handleSuccess() {
  pageRef.value?.refresh();
}
</script>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
  width: 100%;
  min-width: 0;
  flex-shrink: 0;
}
</style>
```

---

### 步骤 6：新增/编辑抽屉 (`src/pages/app/product/product-drawer.vue`)

```vue
<template>
  <ElDrawer
    v-model="visible"
    :title="title"
    :size="DRAWER_WIDTH"
    :close-on-click-modal="false"
    :append-to-body="true"
    :destroy-on-close="true"
    @close="handleClose"
  >
    <ElForm ref="formRef" :model="formData" :rules="formRules" label-width="120px" class="drawer-form">
      <ElFormItem :label="$t('pages.product.name')" prop="name">
        <ElInput v-model="formData.name" :placeholder="$t('common.placeholder.input')" clearable />
      </ElFormItem>

      <ElFormItem :label="$t('pages.product.code')" prop="code">
        <ElInput v-model="formData.code" :placeholder="$t('common.placeholder.input')" clearable />
      </ElFormItem>

      <ElFormItem :label="$t('pages.product.price')" prop="price">
        <ElInputNumber v-model="formData.price" :min="0" style="width: 100%" />
      </ElFormItem>

      <ElFormItem :label="$t('common.table.status')" prop="status">
        <ElRadioGroup v-model="formData.status">
          <ElRadioButton v-for="item in productStatusList" :key="item.value" :value="item.value">
            {{ item.label }}
          </ElRadioButton>
        </ElRadioGroup>
      </ElFormItem>

      <ElFormItem :label="$t('common.table.remark')" prop="remark">
        <ElInput v-model="formData.remark" type="textarea" :rows="3" :placeholder="$t('common.placeholder.input')" />
      </ElFormItem>
    </ElForm>

    <template #footer>
      <div class="drawer-footer">
        <ElButton @click="handleClose">{{ $t("common.button.cancel") }}</ElButton>
        <ElButton type="primary" :loading="submitLoading" @click="handleSubmit">
          {{ $t("common.button.confirm") }}
        </ElButton>
      </div>
    </template>
  </ElDrawer>
</template>

<script lang="ts" setup>
import { computed, reactive, ref } from "vue";
import { ElMessage } from "element-plus";

import { useCreateProduct, useUpdateProduct, productStatusList } from "@/api/composables";
import { $t } from "@/core/i18n";
import { DRAWER_WIDTH } from "@/constants";

const emit = defineEmits<{ success: [] }>();

const { mutateAsync: createProduct } = useCreateProduct();
const { mutateAsync: updateProduct } = useUpdateProduct();

const visible = ref(false);
const submitLoading = ref(false);
const isCreate = ref(true);
const currentId = ref<number>();
const formRef = ref();

const formData = reactive({
  name: "",
  code: "",
  price: 0,
  status: "ON",
  remark: "",
});

const formRules = {
  name: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  code: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  status: [{ required: true, message: $t("common.validation.selectRequired"), trigger: "change" }],
};

const title = computed(() =>
  isCreate.value
    ? $t("common.modal.create", { moduleName: $t("pages.product.moduleName") })
    : $t("common.modal.update", { moduleName: $t("pages.product.moduleName") })
);

async function open(data?: { create: boolean; row?: any }) {
  visible.value = true;
  isCreate.value = data?.create ?? true;
  currentId.value = data?.row?.id;
  resetForm();
  if (!isCreate.value && data?.row) {
    Object.assign(formData, data.row);
  }
}

function handleClose() {
  visible.value = false;
  resetForm();
}

function resetForm() {
  formData.name = "";
  formData.code = "";
  formData.price = 0;
  formData.status = "ON";
  formData.remark = "";
  formRef.value?.clearValidate();
}

async function handleSubmit() {
  if (!formRef.value) return;
  try {
    await formRef.value.validate();
    submitLoading.value = true;
    const values = { ...formData };
    if (isCreate.value) {
      await createProduct(values);
      ElMessage.success($t("common.notification.createSuccess"));
    } else {
      await updateProduct({ id: currentId.value!, values });
      ElMessage.success($t("common.notification.updateSuccess"));
    }
    emit("success");
    handleClose();
  } catch (error) {
    if (error !== false) {
      ElMessage.error(
        isCreate.value ? $t("common.notification.createFailed") : $t("common.notification.updateFailed")
      );
    }
  } finally {
    submitLoading.value = false;
  }
}

defineExpose({ open });
</script>

<style lang="scss" scoped>
.drawer-form { padding-right: 10px; }
.drawer-footer { display: flex; justify-content: flex-end; gap: 10px; }
</style>
```

---

## 编码规范速查

### 通用规范
- TypeScript 接口**不使用 I 前缀**（`User` 而非 `IUser`）
- 所有用户可见文本必须通过 `$t()` / `t()` 国际化，**禁止硬编码中文**
- `$t()` 用于模板和 `computed` 中；`t()` 用于 composable 顶层（非响应式）
- 路由类型使用 `RouteRecordRaw`
- 包管理器使用 **pnpm**
- Git 提交信息遵循 Conventional Commits 规范

### API 层规范
- composable 层**禁止直接依赖 gRPC 生成代码的实现细节**，只导入类型
- 创建 mutation 参数用 `{ data: {...} }` 包裹
- 更新 mutation 必须使用 `makeUpdateMask` 生成字段掩码
- queryKey 全局唯一，格式为 `["操作名", 参数]`
- 删除 API 参数字段注意是 `ids` 还是 `id`（需查看生成类型）

### 组件规范
- `ElDrawer` 必须设置 `:append-to-body="true"` 和 `:destroy-on-close="true"`
- `ElDialog` 的 `appendTo` 属性是字符串选择器（非布尔值）
- `ElTreeSelect` 的 `value` 不接受 `undefined`，用 `null` 或不设初始值
- 暗黑模式下文本颜色使用 `var(--el-text-color-*)`，避免硬编码
- 数字列必须右对齐 (`align: "right"`)
- 抽屉宽度统一使用常量 `DRAWER_WIDTH`（来自 `@/constants`）

### 样式规范
- CSS 变量使用 `--gowind-*` 前缀，避免与 Element Plus `--el-*` 冲突
- 主题色变量存储为 HSL 数值（非 hex 字符串）
- 更新 CSS 变量使用 `style.setProperty()`

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

更多详细参考请查阅 [reference.md](reference.md) 和 [examples.md](examples.md)。
