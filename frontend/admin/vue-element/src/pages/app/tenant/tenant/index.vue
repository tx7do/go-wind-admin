<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage
      ref="pageRef"
      :config="pageConfig"
      @add="handleAdd"
      @edit="handleEdit"
      @toolbar="handleToolbar"
      @operate="handleOperate"
    >
      <!-- 类型 -->
      <template #type="scope: any">
        <ElTag size="small" round :type="tenantTypeToType(scope.row.type)">
          {{ tenantTypeToName(scope.row.type) }}
        </ElTag>
      </template>

      <!-- 审核状态 -->
      <template #auditStatus="scope: any">
        <ElTag size="small" round :type="tenantAuditStatusToType(scope.row.auditStatus)">
          {{ tenantAuditStatusToName(scope.row.auditStatus) }}
        </ElTag>
      </template>

      <!-- 状态 -->
      <template #status="scope: any">
        <ElTag size="small" round :type="tenantStatusToType(scope.row.status)">
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
import { ElMessage, ElMessageBox, ElTag } from "element-plus";
import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig, ToolsButton } from "@/components/Pro/ProPage/types";
import { useProModal } from "@/components/Pro";
import TenantDrawer from "./tenant-drawer.vue";

import {
  tenantAuditStatusList,
  tenantAuditStatusToType,
  tenantAuditStatusToName,
  tenantStatusList,
  tenantStatusToType,
  tenantStatusToName,
  tenantTypeList,
  tenantTypeToType,
  tenantTypeToName,
  fetchListTenants,
  fetchTrashTenants,
  useDeleteTenant,
  usePurgeTenant,
  useRestoreTenant,
} from "@/api/composables";
import { PaginationQuery } from "@go-wind/admin-core";
import { $t } from "@go-wind/admin-element";

const { mutateAsync: deleteTenant } = useDeleteTenant();
const { mutateAsync: restoreTenant } = useRestoreTenant();
const { mutateAsync: purgeTenant } = usePurgeTenant();

const pageRef = ref();

// 回收站二态（issue #3）：active=在营名录 / trash=已删除（停车）租户
const trashMode = ref(false);

// useProModal 连接 TenantDrawer 组件
const [ConnectedDrawer, modalApi] = useProModal({
  connectedComponent: TenantDrawer,
  onOpenChange(isOpen) {
    if (!isOpen) pageRef.value?.refresh();
  },
});

// === 页面配置 ===
const pageConfig = computed<ProPageConfig>(() => ({
  skeleton: true,
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
      const fetcher = trashMode.value ? fetchTrashTenants : fetchListTenants;
      const result = await fetcher(
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
    toolbar: [
      {
        name: "toggleTrash",
        label: trashMode.value
          ? $t("pages.tenant.activeTenants")
          : $t("pages.tenant.trash"),
        icon: "lucide:trash-2",
      } as ToolsButton,
    ],
    toolbarRight: trashMode.value ? [] : ["add"],
    defaultToolbar: ["exports", "imports", "refresh", "zoom", "filter"],
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
      { prop: "memberCount", label: $t("pages.tenant.memberCount"), minWidth: 80 },
      { prop: "remark", label: $t("common.table.remark"), minWidth: 150 },
      {
        prop: "action",
        label: $t("common.table.action"),
        fixed: "right",
        width: 150,
        cellType: "tool",
        buttons:
          trashMode.value
            ? [
                {
                  name: "restore",
                  label: $t("pages.tenant.restore"),
                  icon: "lucide:archive-restore",
                  attrs: { type: "success" },
                },
                {
                  name: "purge",
                  label: $t("pages.tenant.purge"),
                  icon: "lucide:trash-2",
                  attrs: { type: "danger" },
                },
              ]
            : [
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

// === 事件处理 ===
function handleAdd() {
  modalApi.open({ create: true });
}

function handleEdit(row: any) {
  modalApi.open({ create: false, row });
}

async function handleToolbar(name: string) {
  if (name !== "toggleTrash") return;
  trashMode.value = !trashMode.value;
  pageRef.value?.refresh();
}

async function handleOperate(data: { name: string; row: any }) {
  const { name, row } = data;
  if (!row?.id) return;
  if (name === "restore") {
    await ElMessageBox.confirm(
      $t("pages.tenant.restoreConfirm", {
        moduleName: $t("pages.tenant.moduleName"),
      }),
      $t("common.title.confirm"),
      { type: "warning", lockScroll: false }
    ).catch(() => Promise.reject("cancel"));
    try {
      await restoreTenant({ id: row.id });
      ElMessage.success($t("pages.tenant.restoreSuccess"));
    } catch (error: any) {
      ElMessage.error(error?.message || $t("common.message.operationFailed"));
    }
    pageRef.value?.refresh();
  } else if (name === "purge") {
    await ElMessageBox.confirm(
      $t("pages.tenant.purgeConfirmDesc"),
      $t("pages.tenant.purgeConfirmTitle"),
      { type: "warning", lockScroll: false }
    ).catch(() => Promise.reject("cancel"));
    try {
      await purgeTenant({ id: row.id });
      ElMessage.success($t("pages.tenant.purgeSuccess"));
    } catch (error: any) {
      ElMessage.error(error?.message || $t("common.message.operationFailed"));
    }
    pageRef.value?.refresh();
  }
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
