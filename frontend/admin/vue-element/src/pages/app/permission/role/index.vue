<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig" @add="handleAdd" @edit="handleEdit">
      <!-- 状态 -->
      <template #status="scope: any">
        <ElTag size="small" effect="plain" round :type="statusToType(scope.row.status)">
          {{ statusToName(scope.row.status) }}
        </ElTag>
      </template>

      <!-- 数据权限范围 -->
      <template #dataScope="scope: any">
        <ElTag size="small" round :type="dataScopeToType(scope.row.dataScope)">
          {{ roleDataScopeToName(scope.row.dataScope) }}
        </ElTag>
      </template>
    </ProPage>

    <!-- 新增/编辑抽屉 -->
    <RoleDrawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";
import { ElTag } from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import type { ProFormField } from "@/components/Pro/ProForm/types";
import RoleDrawer from "./role-drawer.vue";

import {
  statusList,
  statusToType,
  statusToName,
  dataScopeToType,
  roleDataScopeToName,
  fetchListRoles,
  useDeleteRole,
} from "@/api/composables";
import { PaginationQuery } from "@go-wind/admin-core";
import { $t, useAppUserStore } from "@go-wind/admin-element";

const { mutateAsync: deleteRole } = useDeleteRole();

// 角色值（code）仅平台管理员可见/可搜——其余用户的角色码由系统生成，
// 页面不再展示与维护（issue #14）。
const userStore = useAppUserStore();
const isPlatformAdmin = computed(() =>
  (userStore.userInfo?.roles ?? []).includes("platform:admin"),
);

const pageRef = ref();
const drawerRef = ref();

const pageConfig = computed<ProPageConfig>(() => ({
  skeleton: true,
  search: {
    grid: true,
    fields: [
      {
        type: "input",
        label: $t("pages.role.name"),
        field: "name",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      ...(isPlatformAdmin.value
        ? [
            {
              type: "input",
              label: $t("pages.role.code"),
              field: "code",
              attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
            },
          ] satisfies ProFormField[]
        : []),
      ],
    },

  table: {
    exportFilename: "roles",
    listAction: async (query: any) => {
      const { page, pageSize, ...queryParams } = query;
      const result = await fetchListRoles(
        new PaginationQuery({
          paging: { page: page || 1, pageSize: pageSize || 10 },
          formValues: queryParams,
        })
      );
      return { items: result.items || [], total: result.total || 0 };
    },
    deleteAction: async (ids: string) => {
      await deleteRole({ id: ids as any });
    },
    toolbar: [],
    toolbarRight: ["add"],
    defaultToolbar: ["exports", "refresh", "zoom", "filter"],
    tableAttrs: { border: true, stripe: false },
    columns: [
      { type: "index", label: $t("common.table.seq"), width: 60 },
      { prop: "name", label: $t("pages.role.name"), minWidth: 120 },
      ...(isPlatformAdmin.value
        ? [{ prop: "code", label: $t("pages.role.code"), minWidth: 120 }]
        : []),
      { prop: "parentName", label: $t("pages.role.parentName"), minWidth: 120 },
      { prop: "tenantName", label: $t("pages.role.tenantName"), minWidth: 120 },
      {
        prop: "status",
        label: $t("common.table.status"),
        minWidth: 100,
        slotName: "status",
      },
      {
        prop: "dataScope",
        label: $t("pages.role.dataScope"),
        minWidth: 100,
        slotName: "dataScope",
      },
      { prop: "description", label: $t("common.table.description"), minWidth: 150 },
      {
        prop: "createdAt",
        label: $t("common.table.createdAt"),
        minWidth: 160,
        cellType: "date",
        dateFormat: "YYYY-MM-DD HH:mm:ss",
      },
      {
        prop: "action",
        label: $t("common.table.action"),
        fixed: "right",
        width: 150,
        cellType: "tool",
        buttons: [
          { name: "edit", label: $t("common.button.edit"), icon: "lucide:pen-line" },
          {
            name: "delete",
            label: $t("common.button.delete"),
            icon: "lucide:trash-2",
            attrs: { type: "danger" },
          },
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
