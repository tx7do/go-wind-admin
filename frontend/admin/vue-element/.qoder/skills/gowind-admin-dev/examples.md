# GoWind Admin 代码示例

## 示例 1：完整的 CRUD 模块（职位管理）

### API 服务层

```typescript
// src/api/service/position.ts
import {
  createPositionServiceClient,
  type identityservicev1_CreatePositionRequest,
  type identityservicev1_DeletePositionRequest,
  type identityservicev1_UpdatePositionRequest,
} from "@/api/generated/admin/service/v1";
import { type PaginationQuery, requestApi } from "@/core/transport/rest";

let _instance: ReturnType<typeof createPositionServiceClient> | null = null;

export function getPositionService() {
  if (!_instance) _instance = createPositionServiceClient(requestApi);
  return _instance;
}

export async function listPositions(query: PaginationQuery) {
  const params = query.toRawParams();
  return getPositionService().List({
    ...params,
    sorting: undefined, offset: undefined, limit: undefined,
    token: undefined, filter: undefined, filterExpr: undefined,
  });
}

export async function createPosition(request: identityservicev1_CreatePositionRequest) {
  return getPositionService().Create(request);
}

export async function updatePosition(request: identityservicev1_UpdatePositionRequest) {
  return getPositionService().Update(request);
}

export async function deletePosition(request: identityservicev1_DeletePositionRequest) {
  return getPositionService().Delete(request);
}
```

### API Composable 层

```typescript
// src/api/composables/position.ts
import { computed } from "vue";
import { useMutation, useQuery, type UseMutationOptions, type UseQueryOptions } from "@tanstack/vue-query";
import type {
  identityservicev1_Position,
  identityservicev1_ListPositionResponse,
  identityservicev1_DeletePositionRequest,
} from "@/api/generated/admin/service/v1";
import { makeUpdateMask, type PaginationQuery } from "@/core/transport/rest";
import { listPositions, createPosition, updatePosition, deletePosition } from "@/api/service/position";
import { queryClient } from "@/plugins/vue-query";
import { i18n } from "@/core/i18n";

const t = i18n.global.t;

export function useListPositions(query: PaginationQuery, options?: UseQueryOptions<identityservicev1_ListPositionResponse, Error>) {
  return useQuery({ queryKey: ["listPositions", query], queryFn: () => listPositions(query), ...options });
}

export async function fetchListPositions(params: PaginationQuery) {
  return queryClient.fetchQuery({ queryKey: ["listPositions", params], queryFn: () => listPositions(params), retry: 0 });
}

export function useCreatePosition(options?: UseMutationOptions<{}, Error, Record<string, any>>) {
  return useMutation({
    mutationFn: (values) => createPosition({ data: { ...values } as identityservicev1_Position }),
    ...options,
  });
}

export function useUpdatePosition(options?: UseMutationOptions<{}, Error, { id: number; values: Record<string, any> }>) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      updatePosition({ id, data: { ...values }, updateMask: makeUpdateMask(Object.keys(values ?? {})) }),
    ...options,
  });
}

export function useDeletePosition(options?: UseMutationOptions<{}, Error, identityservicev1_DeletePositionRequest>) {
  return useMutation({ mutationFn: (req) => deletePosition(req), ...options });
}

// 枚举
export const positionTypeList = computed(() => [
  { value: "REGULAR", label: t("enum.position.type.REGULAR") },
  { value: "LEADER", label: t("enum.position.type.LEADER") },
  { value: "MANAGER", label: t("enum.position.type.MANAGER") },
]);
```

---

## 示例 2：列表页 + Drawer 弹窗（useProModal 模式）

### 列表页

```vue
<!-- src/pages/app/opm/position/index.vue -->
<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig" @add="handleAdd" @edit="handleEdit">
      <template #status="scope: any">
        <ElTag size="small" effect="dark" round :color="statusToColor(scope.row.status)">
          {{ statusToName(scope.row.status) }}
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
import PositionDrawer from "./position-drawer.vue";
import { fetchListPositions, useDeletePosition, positionTypeList } from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/core/i18n";

const { mutateAsync: deletePosition } = useDeletePosition();
const pageRef = ref();

const [ConnectedDrawer, modalApi] = useProModal({
  connectedComponent: PositionDrawer,
  onOpenChange(isOpen) {
    if (!isOpen) pageRef.value?.refresh();
  },
});

const pageConfig = computed<ProPageConfig>(() => ({
  search: {
    fields: [
      { type: "input", label: $t("pages.position.name"), field: "name", attrs: { clearable: true } },
      { type: "select", label: $t("pages.position.type"), field: "type",
        options: positionTypeList.value, attrs: { clearable: true } },
    ],
    grid: true,
  },
  table: {
    listAction: async (query: any) => {
      const { page, pageSize, ...queryParams } = query;
      const result = await fetchListPositions(
        new PaginationQuery({ paging: { page: page || 1, pageSize: pageSize || 10 }, formValues: queryParams })
      );
      return { items: result.items || [], total: result.total || 0 };
    },
    deleteAction: async (ids: string) => { await deletePosition({ id: ids as any }); },
    toolbarRight: ["add"],
    defaultToolbar: ["refresh", "filter"],
    columns: [
      { type: "index", label: $t("common.table.seq"), width: 60 },
      { prop: "name", label: $t("pages.position.name"), minWidth: 120 },
      { prop: "remark", label: $t("common.table.remark"), minWidth: 150 },
      {
        prop: "action", label: $t("common.table.action"), fixed: "right", width: 150, cellType: "tool",
        buttons: [
          { name: "edit", label: $t("common.button.edit") },
          { name: "delete", label: $t("common.button.delete"), attrs: { type: "danger" } },
        ],
      },
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

### Drawer 弹窗

```vue
<!-- src/pages/app/opm/position/position-drawer.vue -->
<template>
  <ElDrawer
    v-model="visible"
    :title="title"
    size="600px"
    :close-on-click-modal="false"
    :append-to-body="true"
    :destroy-on-close="true"
    @close="handleClose"
  >
    <ElForm :model="formData" label-width="120px">
      <ElFormItem :label="$t('pages.position.name')" required>
        <ElInput v-model="formData.name" :placeholder="$t('common.placeholder.input')" clearable />
      </ElFormItem>
      <ElFormItem :label="$t('common.table.remark')">
        <ElInput v-model="formData.remark" type="textarea" :rows="3" />
      </ElFormItem>
    </ElForm>

    <template #footer>
      <div class="drawer-footer">
        <ElButton @click="handleClose">{{ $t("common.button.cancel") }}</ElButton>
        <ElButton type="primary" :loading="loading" @click="handleSubmit">
          {{ $t("common.button.confirm") }}
        </ElButton>
      </div>
    </template>
  </ElDrawer>
</template>

<script lang="ts" setup>
import { computed, ref, watch } from "vue";
import { ElMessage } from "element-plus";
import { useCreatePosition, useUpdatePosition } from "@/api/composables";
import { $t } from "@/core/i18n";
import { injectProModalApi } from "@/components/Pro";

const modalApi = injectProModalApi();
const data = computed(() => modalApi.getData<{ create?: boolean; row?: any }>());
const isCreate = computed(() => !!data.value.create);
const visible = computed({
  get: () => modalApi.store.isOpen,
  set: (v) => { if (!v) modalApi.close(); },
});

const { mutateAsync: createMut } = useCreatePosition();
const { mutateAsync: updateMut } = useUpdatePosition();
const loading = ref(false);

const formData = ref({ name: "", remark: "" });
const title = computed(() =>
  isCreate.value
    ? $t("common.modal.create", { moduleName: $t("pages.position.moduleName") })
    : $t("common.modal.update", { moduleName: $t("pages.position.moduleName") })
);

watch(visible, (val) => {
  if (val) {
    if (!isCreate.value && data.value.row) {
      formData.value = { name: data.value.row.name || "", remark: data.value.row.remark || "" };
    } else {
      formData.value = { name: "", remark: "" };
    }
  }
});

function handleClose() { modalApi.close(); }

async function handleSubmit() {
  try {
    loading.value = true;
    if (isCreate.value) {
      await createMut(formData.value);
      ElMessage.success($t("common.notification.create_success"));
    } else {
      await updateMut({ id: data.value.row!.id, values: formData.value });
      ElMessage.success($t("common.notification.update_success"));
    }
    modalApi.close();
  } catch (error) {
    console.error("Submit error:", error);
  } finally {
    loading.value = false;
  }
}
</script>

<style scoped>
.drawer-footer { display: flex; gap: 8px; justify-content: flex-end; }
</style>
```

---

## 示例 3：路由模块

```typescript
// src/router/routes/modules/app/opm.ts
import type { RouteRecordRaw } from "vue-router";
import { Layout } from "@/layouts";

const routes: RouteRecordRaw[] = [
  {
    path: "/opm",
    name: "OpmManagement",
    component: Layout,
    redirect: "/opm/org",
    meta: {
      order: 4000,
      icon: "lucide:building",
      title: "routes.opm.moduleName",
      authority: ["sys:admin"],
    },
    children: [
      {
        path: "position",
        name: "PositionManagement",
        meta: {
          order: 2,
          icon: "lucide:briefcase",
          title: "routes.opm.position",
          authority: ["sys:admin"],
        },
        component: () => import("@/pages/app/opm/position/index.vue"),
      },
    ],
  },
];

export default routes;
```

---

## 示例 4：国际化翻译文件

### 页面翻译 (`src/locales/zh-CN/pages/position.json`)

```json
{
  "moduleName": "职位管理",
  "name": "职位名称",
  "type": "职位类型",
  "notification": {
    "create_success": "职位创建成功",
    "update_success": "职位更新成功",
    "delete_success": "职位删除成功"
  }
}
```

### 枚举翻译 (`src/locales/zh-CN/enum.json` 中追加)

```json
{
  "position": {
    "type": {
      "REGULAR": "正式",
      "LEADER": "负责人",
      "MANAGER": "经理",
      "INTERN": "实习生",
      "CONTRACT": "合同工"
    }
  }
}
```

### 路由翻译 (`src/locales/zh-CN/routes.json` 中追加)

```json
{
  "opm": {
    "moduleName": "组织人员",
    "position": "职位管理"
  }
}
```
