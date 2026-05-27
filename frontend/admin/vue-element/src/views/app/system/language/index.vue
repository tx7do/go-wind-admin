<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig" @add="handleAdd" @edit="handleEdit">
      <!-- 是否启用 -->
      <template #isEnabled="scope: any">
        <ElTag size="small" effect="dark" round :color="enableBoolToColor(scope.row.isEnabled)">
          {{ enableBoolToName(scope.row.isEnabled) }}
        </ElTag>
      </template>

      <!-- 是否默认 -->
      <template #isDefault="scope: any">
        <ElTag size="small" effect="dark" round :color="enableBoolToColor(scope.row.isDefault)">
          {{ enableBoolToName(scope.row.isDefault) }}
        </ElTag>
      </template>
    </ProPage>

    <!-- 新增/编辑抽屉 -->
    <LanguageDrawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import { ElTag } from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import LanguageDrawer from "./language-drawer.vue";

import {
  enableBoolToColor,
  enableBoolToName,
  fetchListLanguages,
  useDeleteLanguage,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/i18n";

const { mutateAsync: deleteLanguage } = useDeleteLanguage();

const pageRef = ref();
const drawerRef = ref();

const pageConfig = computed<ProPageConfig>(() => ({

  search: {
    grid: true,
    fields: [
      {
        type: "input",
        label: $t("pages.language.languageName"),
        field: "languageName",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "input",
        label: $t("pages.language.languageCode"),
        field: "languageCode",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
    ],
  },

  table: {
    listAction: async (query: any) => {
      const { page, pageSize, ...queryParams } = query;
      const result = await fetchListLanguages(
        new PaginationQuery({
          paging: { page: page || 1, pageSize: pageSize || 10 },
          formValues: queryParams,
        })
      );
      return { items: result.items || [], total: result.total || 0 };
    },
    deleteAction: async (ids: string) => {
      await deleteLanguage({ id: ids as any });
    },
    toolbar: [],
    toolbarRight: ["add"],
    defaultToolbar: ["refresh", "exports", "filter"],
    tableAttrs: { border: true, stripe: false },
    columns: [
      { type: "index", label: $t("common.table.seq"), width: 60 },
      { prop: "nativeName", label: $t("pages.language.nativeName"), minWidth: 120, fixed: "left" },
      { prop: "languageName", label: $t("pages.language.languageName"), minWidth: 120 },
      { prop: "languageCode", label: $t("pages.language.languageCode"), minWidth: 120 },
      {
        prop: "isEnabled",
        label: $t("pages.language.isEnabled"),
        width: 100,
        slotName: "isEnabled",
      },
      {
        prop: "isDefault",
        label: $t("pages.language.isDefault"),
        width: 100,
        slotName: "isDefault",
      },
      { prop: "sortOrder", label: $t("common.table.sortOrder"), width: 100 },
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
