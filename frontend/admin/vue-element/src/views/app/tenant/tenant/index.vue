<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig" @add="handleAdd" @edit="handleEdit">
      <!-- 类型 -->
      <template #type="scope: any">
        <ElTag size="small" effect="dark" round :color="tenantTypeToColor(scope.row.type)">
          {{ tenantTypeToName(scope.row.type) }}
        </ElTag>
      </template>

      <!-- 审核状态 -->
      <template #auditStatus="scope: any">
        <ElTag
          size="small"
          effect="dark"
          round
          :color="tenantAuditStatusToColor(scope.row.auditStatus)"
        >
          {{ tenantAuditStatusToName(scope.row.auditStatus) }}
        </ElTag>
      </template>

      <!-- 状态 -->
      <template #status="scope: any">
        <ElTag size="small" effect="dark" round :color="tenantStatusToColor(scope.row.status)">
          {{ tenantStatusToName(scope.row.status) }}
        </ElTag>
      </template>
    </ProPage>

    <!-- 弹窗组件自动连接 -->
    <ConnectedDrawer />
  </div>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import { ElTag } from "element-plus";
import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import { useProModal } from "@/components/Pro";
import TenantDrawer from "./tenant-drawer.vue";

import {
  tenantAuditStatusList,
  tenantAuditStatusToColor,
  tenantAuditStatusToName,
  tenantStatusList,
  tenantStatusToColor,
  tenantStatusToName,
  tenantTypeList,
  tenantTypeToColor,
  tenantTypeToName,
  fetchListTenants,
  useDeleteTenant,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/i18n";

const { mutateAsync: deleteTenant } = useDeleteTenant();

const pageRef = ref();

// useProModal 连接 TenantDrawer 组件
const [ConnectedDrawer, modalApi] = useProModal({
  connectedComponent: TenantDrawer,
  onOpenChange(isOpen) {
    if (!isOpen) pageRef.value?.refresh();
  },
});

// === 页面配置 ===
const pageConfig = computed<ProPageConfig>(() => ({

  // 搜索配置
  search: {
    fields: [
      {
        type: "input",
        label: $t("pages.tenant.name"),
        field: "name",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "input",
        label: $t("pages.tenant.code"),
        field: "code",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "select",
        label: $t("pages.tenant.type"),
        field: "type",
        attrs: { placeholder: $t("common.placeholder.select"), clearable: true },
        options: tenantTypeList.value,
      },
      {
        type: "select",
        label: $t("pages.tenant.auditStatus"),
        field: "auditStatus",
        attrs: { placeholder: $t("common.placeholder.select"), clearable: true },
        options: tenantAuditStatusList.value,
      },
      {
        type: "select",
        label: $t("common.table.status"),
        field: "status",
        attrs: { placeholder: $t("common.placeholder.select"), clearable: true },
        options: tenantStatusList.value,
      },
    ],
    grid: true,
  },

  // 表格配置
  table: {
    listAction: async (query: any) => {
      const { page, pageSize, ...queryParams } = query;
      const result = await fetchListTenants(
        new PaginationQuery({
          paging: { page: page || 1, pageSize: pageSize || 10 },
          formValues: queryParams,
        })
      );
      return { items: result.items || [], total: result.total || 0 };
    },
    deleteAction: async (ids: string) => {
      await deleteTenant({ id: ids as any });
    },
    toolbar: [],
    toolbarRight: ["add"],
    defaultToolbar: ["refresh", "exports", "imports", "filter"],
    tableAttrs: { border: true, stripe: false },
    columns: [
      { type: "index", label: $t("common.table.seq"), width: 60 },
      { prop: "name", label: $t("pages.tenant.name"), minWidth: 120 },
      { prop: "code", label: $t("pages.tenant.code"), minWidth: 120 },
      { prop: "adminUserName", label: $t("pages.tenant.adminUserName"), minWidth: 120 },
      { prop: "type", label: $t("pages.tenant.type"), minWidth: 100, slotName: "type" },
      {
        prop: "auditStatus",
        label: $t("pages.tenant.auditStatus"),
        minWidth: 100,
        slotName: "auditStatus",
      },
      { prop: "status", label: $t("common.table.status"), minWidth: 100, slotName: "status" },
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
          { name: "edit", label: $t("common.button.edit") },
          { name: "delete", label: $t("common.button.delete"), attrs: { type: "danger" } },
        ],
      },
    ],
  },
}));

// === 事件处理 ===
function handleAdd() {
  modalApi.open({ create: true });
}

function handleEdit(row: any) {
  modalApi.open({ create: false, row });
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
