<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig" @add="handleAdd" @edit="handleEdit">
      <!-- 状态 -->
      <template #isEnabled="scope: any">
        <ElTag size="small" effect="dark" round :color="enableBoolToColor(scope.row.isEnabled)">
          {{ enableBoolToName(scope.row.isEnabled) }}
        </ElTag>
      </template>
    </ProPage>

    <!-- 抽屉 -->
    <InternalMessageCategoryDrawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import { ElTag } from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";

import {
  enableBoolToColor,
  enableBoolToName,
  fetchListMessageCategories,
  useDeleteMessageCategory,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/i18n";

import InternalMessageCategoryDrawer from "./internal-message-category-drawer.vue";

const { mutateAsync: deleteMessageCategory } = useDeleteMessageCategory();

const pageRef = ref();
const drawerRef = ref();

const pageConfig = computed<ProPageConfig>(() => ({

  search: {
    grid: true,
    fields: [
      {
        type: "input",
        label: $t("pages.internal_message_category.name"),
        field: "name",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "input",
        label: $t("pages.internal_message_category.code"),
        field: "code",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
    ],
  },

  table: {
    listAction: async (query: any) => {
      const { page, pageSize, ...queryParams } = query;
      const result = await fetchListMessageCategories(
        new PaginationQuery({
          paging: { page: page || 1, pageSize: pageSize || 10 },
          formValues: queryParams,
        })
      );
      return { items: result.items || [], total: result.total || 0 };
    },
    deleteAction: async (ids: string) => {
      await deleteMessageCategory({ id: ids as any });
    },
    toolbar: [],
    toolbarRight: ["add"],
    defaultToolbar: ["refresh", "exports", "filter"],
    pagination: false,
    tableAttrs: { border: true, stripe: false },
    columns: [
      { prop: "name", label: $t("pages.internal_message_category.name"), minWidth: 150 },
      { prop: "code", label: $t("pages.internal_message_category.code"), minWidth: 150 },
      { prop: "sortOrder", label: $t("common.table.sortOrder"), width: 70 },
      {
        prop: "isEnabled",
        label: $t("common.table.status"),
        width: 95,
        slotName: "isEnabled",
      },
      {
        prop: "createdAt",
        label: $t("common.table.createdAt"),
        width: 160,
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

function handleAdd() {
  drawerRef.value?.open();
}

function handleEdit(row: any) {
  drawerRef.value?.open(row);
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
