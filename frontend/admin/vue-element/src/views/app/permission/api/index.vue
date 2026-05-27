<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig" @add="handleAdd" @edit="handleEdit" @toolbar="handleToolbar" />

    <!-- 新增/编辑抽屉 -->
    <ApiDrawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig, ToolsButton } from "@/components/Pro/ProPage/types";
import ApiDrawer from "./api-drawer.vue";

import { methodList, fetchListApis, useDeleteApi, useSyncApisApi } from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/i18n";

const { mutateAsync: deleteApi } = useDeleteApi();
const { mutateAsync: syncApis } = useSyncApisApi();

const pageRef = ref();
const drawerRef = ref();

const pageConfig = computed<ProPageConfig>(() => ({

  search: {
    grid: true,
    fields: [
      {
        type: "select",
        label: $t("pages.api.method"),
        field: "method",
        attrs: {
          placeholder: $t("common.placeholder.select"),
          clearable: true,
          filterable: true,
        },
        options: methodList,
      },
      {
        type: "input",
        label: $t("pages.api.module"),
        field: "module",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "input",
        label: $t("pages.api.path"),
        field: "path",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
    ],
  },

  table: {
    listAction: async (query: any) => {
      const { page, pageSize, ...queryParams } = query;
      const result = await fetchListApis(
        new PaginationQuery({
          paging: { page: page || 1, pageSize: pageSize || 10 },
          formValues: queryParams,
          orderBy: ["path"],
        })
      );
      return { items: result.items || [], total: result.total || 0 };
    },
    deleteAction: async (ids: string) => {
      await deleteApi({ id: ids as any });
    },
    toolbar: [],
    toolbarRight: [
      {
        name: "sync",
        label: $t("pages.api.button.sync"),
        attrs: { type: "danger", icon: "refresh" },
      } as ToolsButton,
      "add",
    ],
    defaultToolbar: ["refresh", "exports", "filter"],
    tableAttrs: { border: true, stripe: false },
    columns: [
      { type: "index", label: $t("common.table.seq"), width: 60 },
      { prop: "description", label: $t("common.table.description"), minWidth: 150 },
      { prop: "path", label: $t("pages.api.path"), minWidth: 200 },
      { prop: "method", label: $t("pages.api.method"), width: 100 },
      { prop: "module", label: $t("pages.api.module"), minWidth: 120 },
      { prop: "moduleDescription", label: $t("pages.api.moduleDescription"), minWidth: 150 },
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
          { name: "edit", label: $t("common.button.edit") },
          { name: "delete", label: $t("common.button.delete"), attrs: { type: "danger" } },
        ],
      },
    ],
  },
}));

function handleAdd() {
  drawerRef.value?.open();
}

function handleEdit(row: any) {
  drawerRef.value?.open(row);
}

function handleToolbar(name: string) {
  if (name === "sync") {
    ElMessageBox.confirm(
      $t("common.confirm.do_you_want_sync", { moduleName: $t("pages.api.moduleName") }),
      $t("common.title.confirm"),
      {
        confirmButtonText: $t("common.button.confirm"),
        cancelButtonText: $t("common.button.cancel"),
        type: "warning",
      }
    ).then(async () => {
      try {
        await syncApis();
        ElMessage.success($t("pages.api.notification.sync_success"));
        pageRef.value?.refresh();
      } catch {
        ElMessage.error($t("pages.api.notification.sync_failed"));
      }
    });
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
