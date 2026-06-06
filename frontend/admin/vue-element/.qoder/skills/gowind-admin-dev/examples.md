# GoWind Admin 代码示例

> 以下示例均来自脚手架内置模块的真实代码，二开人员可直接参考复制。

---

## 示例 1：完整 CRUD 模块（职位管理 position）

### 1.1 API 服务层

```typescript
// src/api/service/position.ts
import {
  createPositionServiceClient,
  type identityservicev1_CreatePositionRequest,
  type identityservicev1_DeletePositionRequest,
  type identityservicev1_GetPositionRequest,
  type identityservicev1_UpdatePositionRequest,
} from "@/api/generated/admin/service/v1";
import { type PaginationQuery, requestApi } from "@/core/transport/rest";

let _positionInstance: ReturnType<typeof createPositionServiceClient> | null = null;

export function getPositionService() {
  if (!_positionInstance) {
    _positionInstance = createPositionServiceClient(requestApi);
  }
  return _positionInstance;
}

export async function listPositions(query: PaginationQuery) {
  const params = query.toRawParams();
  return getPositionService().List(params);
}

export async function getPosition(request: identityservicev1_GetPositionRequest) {
  return getPositionService().Get(request);
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

### 1.2 API Composable 层

```typescript
// src/api/composables/position.ts
import { computed } from "vue";
import {
  useMutation, useQuery,
  type UseMutationOptions, type UseQueryOptions,
} from "@tanstack/vue-query";
import type {
  identityservicev1_DeletePositionRequest,
  identityservicev1_GetPositionRequest,
  identityservicev1_ListPositionResponse,
  identityservicev1_Position,
  identityservicev1_Position_Status as Position_Status,
  identityservicev1_Position_Type as Position_Type,
} from "@/api/generated/admin/service/v1";
import { makeUpdateMask, type PaginationQuery } from "@/core/transport/rest";
import {
  listPositions, getPosition, createPosition, updatePosition, deletePosition,
} from "@/api/service/position";
import { queryClient } from "@/plugins/vue-query";
import { i18n } from "@/core/i18n";

const t = i18n.global.t;

// 列表 Hook
export function useListPositions(
  query: PaginationQuery,
  options?: UseQueryOptions<identityservicev1_ListPositionResponse, Error>
) {
  return useQuery({
    queryKey: ["listPositions", query],
    queryFn: () => listPositions(query),
    ...options,
  });
}

// 非Hook列表查询
export async function fetchListPositions(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ["listPositions", params],
    queryFn: () => listPositions(params),
    retry: 0,
  });
}

// 单条查询
export function useGetPosition(
  req: identityservicev1_GetPositionRequest,
  options?: UseQueryOptions<identityservicev1_Position, Error>
) {
  return useQuery({
    queryKey: ["getPosition", req],
    queryFn: () => getPosition(req),
    ...options,
  });
}

// 创建 — 注意 { data: {...} } 包裹
export function useCreatePosition(options?: UseMutationOptions<{}, Error, Record<string, any>>) {
  return useMutation({
    mutationFn: (values) => createPosition({ data: { ...values } as identityservicev1_Position }),
    ...options,
  });
}

// 更新 — 使用 makeUpdateMask
export function useUpdatePosition(
  options?: UseMutationOptions<{}, Error, { id: number; values: Record<string, any> }>
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      updatePosition({
        id,
        data: { ...values },
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

// 删除
export function useDeletePosition(
  options?: UseMutationOptions<{}, Error, identityservicev1_DeletePositionRequest>
) {
  return useMutation({ mutationFn: (req) => deletePosition(req), ...options });
}

// ==============================
// 枚举工具函数
// ==============================

export const membershipPositionStatusList = computed(() => [
  { value: "PROBATION", label: t("enum.membershipPosition.status.PROBATION") },
  { value: "ACTIVE", label: t("enum.membershipPosition.status.ACTIVE") },
  { value: "LEAVE", label: t("enum.membershipPosition.status.LEAVE") },
  { value: "RESIGNED", label: t("enum.membershipPosition.status.RESIGNED") },
  { value: "TERMINATED", label: t("enum.membershipPosition.status.TERMINATED") },
  { value: "EXPIRED", label: t("enum.membershipPosition.status.EXPIRED") },
]);

export function membershipPositionStatusToName(status: any) {
  const values = membershipPositionStatusList.value;
  const matchedItem = values.find((item) => item.value === status);
  return matchedItem ? matchedItem.label : "";
}

const MEMBERSHIP_POSITION_STATUS_COLOR_MAP: Record<string, string> = {
  PROBATION: "#4096FF",
  ACTIVE: "#00B42A",
  LEAVE: "#FF9A2E",
  RESIGNED: "#F56C6C",
  TERMINATED: "#F53F3F",
  EXPIRED: "#909399",
  DEFAULT: "#C9CDD4",
};

export function membershipPositionStatusToColor(status: Position_Status) {
  return (
    MEMBERSHIP_POSITION_STATUS_COLOR_MAP[status as string] ||
    MEMBERSHIP_POSITION_STATUS_COLOR_MAP.DEFAULT
  );
}

// 支持明暗模式的颜色映射
const POSITION_TYPE_COLOR_THEME: Record<string, Record<string, string>> = {
  light: {
    REGULAR: "#165DFF",
    LEADER: "#722ED1",
    MANAGER: "#FF7D00",
    INTERN: "#52C41A",
    CONTRACT: "#14C9C9",
    OTHER: "#86909C",
    DEFAULT: "#C9CDD4",
  },
  dark: {
    REGULAR: "#2F77FF",
    LEADER: "#8542E7",
    MANAGER: "#FF9529",
    INTERN: "#67E037",
    CONTRACT: "#20E0E0",
    OTHER: "#9BA3AD",
    DEFAULT: "#DCE0E6",
  },
};

export function positionTypeToColor(
  positionType: Position_Type,
  theme: "dark" | "light" = "light"
): string {
  const colorMap = POSITION_TYPE_COLOR_THEME[theme];
  return colorMap[positionType as string] || colorMap.DEFAULT;
}
```

---

## 示例 2：列表页 + Drawer 弹窗（defineExpose 模式）

> 脚手架内置模块采用 **`defineExpose` + `ref` 模式**，而非 `useProModal` 连接模式。

### 2.1 列表页

```vue
<!-- src/pages/app/opm/position/index.vue -->
<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig" @add="handleAdd" @edit="handleEdit">
      <!-- 自定义插槽列 -->
      <template #type="scope: any">
        <ElTag size="small" effect="dark" round :color="positionTypeToColor(scope.row.type)">
          {{ positionTypeToName(scope.row.type) }}
        </ElTag>
      </template>
      <template #status="scope: any">
        <ElTag size="small" effect="dark" round :color="statusToColor(scope.row.status)">
          {{ statusToName(scope.row.status) }}
        </ElTag>
      </template>
    </ProPage>

    <!-- 新增/编辑抽屉 -->
    <PositionDrawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from "vue";
import { ElTag } from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import PositionDrawer from "./position-drawer.vue";

import {
  positionTypeList,
  positionTypeToColor,
  positionTypeToName,
  statusList,
  statusToColor,
  statusToName,
  fetchListOrgUnits,
  fetchListPositions,
  useDeletePosition,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/core/i18n";

const { mutateAsync: deletePosition } = useDeletePosition();

const pageRef = ref();
const drawerRef = ref();

const pageConfig = computed<ProPageConfig>(() => ({
  skeleton: true,
  search: {
    grid: true,
    fields: [
      {
        type: "input",
        label: $t("pages.position.name"),
        field: "name",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "input",
        label: $t("pages.position.code"),
        field: "code",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "select",
        label: $t("common.table.status"),
        field: "status",
        attrs: { placeholder: $t("common.placeholder.select"), clearable: true },
        options: statusList.value,
      },
      {
        type: "select",
        label: $t("pages.position.type"),
        field: "type",
        attrs: { placeholder: $t("common.placeholder.select"), clearable: true },
        options: positionTypeList.value,
      },
      {
        type: "tree-select",
        label: $t("pages.position.orgUnit"),
        field: "orgUnitId",
        attrs: {
          placeholder: $t("common.placeholder.select"),
          clearable: true,
          filterable: true,
          "default-expand-all": true,
          nodeKey: "id",
          props: { label: "name", value: "id", children: "children" },
        },
        initFn: async (item: any) => {
          try {
            const result = await fetchListOrgUnits(
              new PaginationQuery({ formValues: { status: "ON" } })
            );
            item.attrs.data = result.items || [];
          } catch (error) {
            console.error("Failed to load org unit tree:", error);
          }
        },
      },
    ],
  },
  table: {
    listAction: async (query: any) => {
      const { page, pageSize, ...queryParams } = query;
      const result = await fetchListPositions(
        new PaginationQuery({
          paging: { page: page || 1, pageSize: pageSize || 10 },
          formValues: queryParams,
        })
      );
      return { items: result.items || [], total: result.total || 0 };
    },
    deleteAction: async (ids: string) => {
      await deletePosition({ id: ids as any });
    },
    toolbar: [],
    toolbarRight: ["add"],
    defaultToolbar: ["refresh", "filter"],
    tableAttrs: { border: true, stripe: true },
    columns: [
      { type: "index", label: $t("common.table.seq"), width: 60 },
      { prop: "name", label: $t("pages.position.name"), minWidth: 120 },
      { prop: "code", label: $t("pages.position.code"), minWidth: 120 },
      { prop: "type", label: $t("pages.position.type"), minWidth: 100, slotName: "type" },
      { prop: "description", label: $t("pages.position.description"), minWidth: 150 },
      { prop: "orgUnitName", label: $t("pages.position.orgUnitName"), minWidth: 120 },
      { prop: "headcount", label: $t("pages.position.headcount"), width: 80, align: "right" },
      { prop: "status", label: $t("common.table.status"), minWidth: 100, slotName: "status" },
      { prop: "sortOrder", label: $t("common.table.sortOrder"), width: 80, align: "right" },
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

### 2.2 Drawer 弹窗

```vue
<!-- src/pages/app/opm/position/position-drawer.vue -->
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
      <ElDivider content-position="left">{{ $t("common.section.basic") }}</ElDivider>

      <ElFormItem :label="$t('pages.position.name')" prop="name">
        <ElInput v-model="formData.name" :placeholder="$t('common.placeholder.input')" clearable />
      </ElFormItem>

      <ElFormItem :label="$t('pages.position.code')" prop="code">
        <ElInput v-model="formData.code" :placeholder="$t('common.placeholder.input')" clearable />
      </ElFormItem>

      <ElFormItem :label="$t('pages.position.type')" prop="type">
        <ElSelect v-model="formData.type" :placeholder="$t('common.placeholder.select')" filterable clearable style="width: 100%">
          <ElOption v-for="item in positionTypeList" :key="item.value" :label="item.label" :value="item.value" />
        </ElSelect>
      </ElFormItem>

      <ElFormItem :label="$t('pages.position.orgUnit')" prop="orgUnitId">
        <ElTreeSelect
          v-model="formData.orgUnitId"
          :data="orgUnitTreeData"
          node-key="id"
          check-strictly
          :render-after-expand="false"
          default-expand-all
          filterable
          clearable
          :props="{ label: 'name', value: 'id', children: 'children' } as any"
          :placeholder="$t('common.placeholder.select')"
          style="width: 100%"
        />
      </ElFormItem>

      <ElFormItem :label="$t('common.table.sortOrder')" prop="sortOrder">
        <ElInputNumber v-model="formData.sortOrder" :min="1" style="width: 100%" />
      </ElFormItem>

      <ElFormItem :label="$t('common.table.status')" prop="status">
        <ElRadioGroup v-model="formData.status">
          <ElRadioButton v-for="item in statusList" :key="item.value" :value="item.value">
            {{ item.label }}
          </ElRadioButton>
        </ElRadioGroup>
      </ElFormItem>

      <ElDivider content-position="left">{{ $t("common.section.other") }}</ElDivider>

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

import {
  useCreatePosition,
  useUpdatePosition,
  fetchListOrgUnits,
  positionTypeList,
  statusList,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/core/i18n";
import { DRAWER_WIDTH } from "@/constants";

const emit = defineEmits<{ success: [] }>();

const { mutateAsync: createPosition } = useCreatePosition();
const { mutateAsync: updatePosition } = useUpdatePosition();

const visible = ref(false);
const submitLoading = ref(false);
const isCreate = ref(true);
const currentId = ref<number>();
const formRef = ref();

const formData = reactive({
  name: "",
  code: "",
  type: "REGULAR",
  orgUnitId: undefined as number | undefined,
  sortOrder: 1,
  status: "ON",
  remark: "",
});

const formRules = {
  name: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  code: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  type: [{ required: true, message: $t("common.validation.selectRequired"), trigger: "change" }],
  orgUnitId: [{ required: true, message: $t("common.validation.selectRequired"), trigger: "change" }],
  status: [{ required: true, message: $t("common.validation.selectRequired"), trigger: "change" }],
};

const title = computed(() =>
  isCreate.value
    ? $t("common.modal.create", { moduleName: $t("pages.position.moduleName") })
    : $t("common.modal.update", { moduleName: $t("pages.position.moduleName") })
);

const orgUnitTreeData = ref<any[]>([]);

async function loadOrgUnitTree() {
  try {
    const result = await fetchListOrgUnits(new PaginationQuery({ formValues: { status: "ON" } }));
    orgUnitTreeData.value = result.items || [];
  } catch (error) {
    console.error("Failed to load org unit tree:", error);
  }
}

async function open(data?: { create: boolean; row?: any }) {
  visible.value = true;
  isCreate.value = data?.create ?? true;
  currentId.value = data?.row?.id;
  resetForm();
  await loadOrgUnitTree();
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
  formData.type = "REGULAR";
  formData.orgUnitId = undefined;
  formData.sortOrder = 1;
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
      await createPosition(values);
      ElMessage.success($t("common.notification.createSuccess"));
    } else {
      await updatePosition({ id: currentId.value!, values });
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

## 示例 3：路由模块

```typescript
// src/router/routes/modules/app/opm.ts
import type { RouteRecordRaw } from "vue-router";
import { Layout } from "@/layouts";

const opm: RouteRecordRaw[] = [
  {
    path: "/opm",
    name: "OrganizationalPersonnelManagement",
    component: Layout,
    redirect: "/opm/users",
    meta: {
      order: 2001,
      icon: "lucide:users",
      title: "routes.opm.moduleName",
      keepAlive: true,
      authority: ["sys:platform_admin", "sys:tenant_manager"],
    },
    children: [
      {
        path: "org-units",
        name: "OrgUnitManagement",
        meta: {
          order: 1,
          icon: "lucide:layers",
          title: "routes.opm.orgUnit",
          authority: ["sys:platform_admin", "sys:tenant_manager"],
        },
        component: () => import("@/pages/app/opm/org_unit/index.vue"),
      },
      {
        path: "positions",
        name: "PositionManagement",
        meta: {
          order: 2,
          icon: "lucide:briefcase",
          title: "routes.opm.position",
          authority: ["sys:platform_admin", "sys:tenant_manager"],
        },
        component: () => import("@/pages/app/opm/position/index.vue"),
      },
      {
        path: "users",
        name: "UserManagement",
        meta: {
          order: 3,
          icon: "lucide:user",
          title: "routes.opm.user",
          authority: ["sys:platform_admin", "sys:tenant_manager"],
        },
        component: () => import("@/pages/app/opm/user/list/index.vue"),
      },
      {
        path: "users/detail/:id",
        name: "UserDetail",
        meta: {
          hideInMenu: true,
          title: "routes.opm.userDetail",
          authority: ["sys:platform_admin", "sys:tenant_manager"],
        },
        component: () => import("@/pages/app/opm/user/detail/index.vue"),
      },
    ],
  },
];

export default opm;
```

---

## 示例 4：国际化翻译文件

### 页面翻译 (`src/locales/zh-CN/pages/position.json`)

```json
{
  "moduleName": "职位",
  "name": "职位名称",
  "code": "职位编码",
  "type": "职位类型",
  "orgUnit": "所属组织",
  "orgUnitName": "所属组织",
  "headcount": "编制人数",
  "sortOrder": "排序",
  "status": "状态",
  "description": "职位描述",
  "remark": "备注",
  "createdAt": "创建时间"
}
```

### 枚举翻译 (`src/locales/zh-CN/enum.json` 中的 position 部分)

```json
{
  "position": {
    "type": {
      "REGULAR": "普通员工",
      "LEADER": "领导",
      "MANAGER": "经理",
      "INTERN": "实习生",
      "CONTRACT": "合同工",
      "OTHER": "其他"
    }
  }
}
```

### 路由翻译 (`src/locales/zh-CN/routes.json` 中的 opm 部分)

```json
{
  "opm": {
    "moduleName": "组织人员",
    "orgUnit": "组织架构",
    "position": "职位管理",
    "user": "用户管理",
    "userDetail": "用户详情"
  }
}
```

---

## 示例 5：通用枚举工具函数（shared.ts）

> 脚手架内置了通用枚举工具，二开人员可直接使用或参考。

```typescript
// src/api/composables/shared.ts（已内置，直接导入使用）
import { computed } from "vue";
import { $t } from "@/core/i18n";

// 开关状态列表
export const enableList = computed(() => [
  { value: "true", label: $t("enum.enable.true") },
  { value: "false", label: $t("enum.enable.false") },
]);

// 通用状态列表（ON/OFF）
export const statusList = computed(() => [
  { value: "ON", label: $t("enum.status.ON") },
  { value: "OFF", label: $t("enum.status.OFF") },
]);

export function statusToName(status: "OFF" | "ON" | undefined) {
  const values = statusList.value;
  const matchedItem = values.find((item) => item.value === status);
  return matchedItem ? matchedItem.label : "";
}

export function statusToColor(status: "OFF" | "ON" | undefined) {
  switch (status) {
    case "ON": return "#52C41A";
    case "OFF": return "#8C8C8C";
    default: return "#C9CDD4";
  }
}
```

**使用方式：**
```typescript
// 在页面组件中
import { statusList, statusToName, statusToColor } from "@/api/composables";

// 搜索栏下拉
options: statusList.value,

// 表格状态列
<ElTag :color="statusToColor(scope.row.status)">
  {{ statusToName(scope.row.status) }}
</ElTag>
```
