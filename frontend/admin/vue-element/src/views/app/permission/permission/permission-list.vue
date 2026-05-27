<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig" @add="handleAdd" @edit="handleEdit" @toolbar="handleToolbar">
      <!-- 状态 -->
      <template #status="scope: any">
        <ElTag size="small" effect="dark" round :color="statusToColor(scope.row.status)">
          {{ statusToName(scope.row.status) }}
        </ElTag>
      </template>
    </ProPage>

    <!-- 新增/编辑抽屉 -->
    <PermissionDrawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ref, watch, onMounted } from "vue";
import { ElTag, ElMessage, ElMessageBox } from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import PermissionDrawer from "./permission-drawer.vue";

import { statusList, statusToColor, statusToName, useDeletePermission } from "@/api/composables";
import { $t } from "@/i18n";
import { usePermissionViewStore } from "@/views/app/permission/permission/permission-view.state";

const { mutateAsync: deletePermission } = useDeletePermission();
const permissionViewStore = usePermissionViewStore();

const pageRef = ref();
const drawerRef = ref();

// 监听分组切换,自动刷新权限列表
watch(
  () => permissionViewStore.needReloadPermissionList,
  (needReload) => {
    if (needReload && pageRef.value) {
      pageRef.value.refresh();
      permissionViewStore.needReloadPermissionList = false;
    }
  }
);

// 初始化时加载一次
onMounted(() => {
  if (permissionViewStore.currentGroupId) {
    permissionViewStore.needReloadPermissionList = true;
  }
});

const pageConfig: ProPageConfig = {

  search: {
    fields: [
      {
        type: "input",
        label: $t("pages.permission.name"),
        field: "name",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "input",
        label: $t("pages.permission.code"),
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
    ],
  },

  table: {
    listAction: async (query: any) => {
      const { page, pageSize, ...queryParams } = query;
      const result = await permissionViewStore.fetchPermissionList(
        permissionViewStore.currentGroupId,
        page || 1,
        pageSize || 10,
        queryParams
      );
      return { items: result.items || [], total: result.total || 0 };
    },
    deleteAction: async (ids: string) => {
      await deletePermission({ id: ids as any });
    },
    toolbar: [],
    toolbarRight: ["add"],
    defaultToolbar: ["refresh", "filter"],
    tableAttrs: { border: true, stripe: true, height: "auto" },
    columns: [
      {
        prop: "name",
        label: $t("pages.permission.name"),
        minWidth: 120,
        align: "left",
      },
      {
        prop: "code",
        label: $t("pages.permission.code"),
        minWidth: 120,
        align: "left",
      },
      {
        prop: "groupName",
        label: $t("pages.permission.groupName"),
        minWidth: 120,
      },
      {
        prop: "status",
        label: $t("common.table.status"),
        width: 90,
        slotName: "status",
      },
      {
        prop: "action",
        label: $t("common.table.action"),
        fixed: "right",
        width: 150,
        cellType: "tool",
        buttons: [
          { name: "edit", text: $t("common.button.edit") },
          { name: "delete", text: $t("common.button.delete"), attrs: { type: "danger" } },
        ],
      },
    ],
  },
};

function handleAdd() {
  drawerRef.value?.open({ create: true });
}

function handleEdit(row: any) {
  drawerRef.value?.open({ create: false, row });
}

async function handleToolbar(name: string) {
  if (name === "syncPermissions") {
    try {
      await ElMessageBox.confirm(
        $t("common.message.confirmSyncPermissions", {
          moduleName: $t("pages.permission.moduleName"),
        }),
        $t("common.title.warning"),
        {
          confirmButtonText: $t("common.button.confirm"),
          cancelButtonText: $t("common.button.cancel"),
          type: "warning",
        }
      );
      permissionViewStore.reloadGroupList();
      ElMessage.success($t("common.notification.syncSuccess"));
    } catch (error) {
      if (error !== "cancel") {
        ElMessage.error($t("common.notification.syncFailed"));
      }
    }
  }
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
