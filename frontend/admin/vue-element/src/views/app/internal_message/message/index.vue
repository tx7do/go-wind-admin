<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig" @add="handleAdd" @edit="handleEdit">
      <!-- 状态 -->
      <template #status="scope: any">
        <ElTag
          size="small"
          effect="dark"
          round
          :color="internalMessageStatusColor(scope.row.status)"
        >
          {{ internalMessageStatusLabel(scope.row.status) }}
        </ElTag>
      </template>

      <!-- 类型 -->
      <template #type="scope: any">
        <ElTag size="small" effect="dark" round :color="internalMessageTypeColor(scope.row.type)">
          {{ internalMessageTypeLabel(scope.row.type) }}
        </ElTag>
      </template>
    </ProPage>

    <!-- 抽屉 -->
    <InternalMessageDrawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import { ElTag } from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";

import {
  internalMessageStatusColor,
  internalMessageStatusLabel,
  internalMessageStatusList,
  internalMessageTypeColor,
  internalMessageTypeLabel,
  internalMessageTypeList,
  fetchListInternalMessages,
  fetchListMessageCategories,
  useDeleteInternalMessage,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/i18n";

import InternalMessageDrawer from "./internal-message-drawer.vue";

const { mutateAsync: deleteMessage } = useDeleteInternalMessage();

const pageRef = ref();
const drawerRef = ref();

const pageConfig = computed<ProPageConfig>(() => ({

  search: {
    grid: true,
    fields: [
      {
        type: "input",
        label: $t("pages.internal_message.title"),
        field: "title",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "select",
        label: $t("pages.internal_message.status"),
        field: "status",
        attrs: {
          placeholder: $t("common.placeholder.select"),
          clearable: true,
          filterable: true,
        },
        options: internalMessageStatusList.value,
      },
      {
        type: "select",
        label: $t("pages.internal_message.type"),
        field: "type",
        attrs: {
          placeholder: $t("common.placeholder.select"),
          clearable: true,
          filterable: true,
        },
        options: internalMessageTypeList.value,
      },
      {
        type: "api-tree-select",
        label: $t("pages.internal_message.categoryId"),
        field: "category_id",
        attrs: {
          placeholder: $t("common.placeholder.select"),
          clearable: true,
          filterable: true,
          nodeKey: "id",
          props: {
            label: "name",
            children: "children",
          },
          defaultExpandAll: true,
        },
        api: async () => {
          const result = await fetchListMessageCategories(
            new PaginationQuery({ formValues: { is_enabled: "true" } })
          );
          return result.items || [];
        },
      },
    ],
  },

  table: {
    listAction: async (query: any) => {
      const { page, pageSize, ...queryParams } = query;
      const result = await fetchListInternalMessages(
        new PaginationQuery({
          paging: { page: page || 1, pageSize: pageSize || 10 },
          formValues: queryParams,
        })
      );
      return { items: result.items || [], total: result.total || 0 };
    },
    deleteAction: async (ids: string) => {
      await deleteMessage({ id: ids as any });
    },
    toolbar: [],
    toolbarRight: ["add"],
    defaultToolbar: ["refresh", "exports", "filter"],
    pagination: false,
    tableAttrs: { border: true, stripe: false },
    columns: [
      { prop: "title", label: $t("pages.internal_message.title"), minWidth: 200 },
      { prop: "categoryName", label: $t("pages.internal_message.categoryName"), minWidth: 150 },
      {
        prop: "status",
        label: $t("pages.internal_message.status"),
        width: 120,
        slotName: "status",
      },
      {
        prop: "type",
        label: $t("pages.internal_message.type"),
        width: 120,
        slotName: "type",
      },
      { prop: "senderName", label: $t("pages.internal_message.senderName"), minWidth: 120 },
      {
        prop: "createdAt",
        label: $t("common.table.createdAt"),
        width: 160,
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
