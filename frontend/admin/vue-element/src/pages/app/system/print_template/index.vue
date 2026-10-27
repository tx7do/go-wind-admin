<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage
      ref="pageRef"
      :config="pageConfig"
      @add="handleAdd"
      @edit="handleEdit"
      @operate="handleOperate"
    >
      <!-- 纸张方向 -->
      <template #orientation="scope: any">
        {{ scope.row.orientation === "landscape" ? $t("pages.printTemplate.landscape") : $t("pages.printTemplate.portrait") }}
      </template>

      <!-- 启用状态 -->
      <template #isEnabled="scope: any">
        <ElTag size="small" round :type="scope.row.isEnabled ? 'success' : 'info'">
          {{ scope.row.isEnabled ? $t("pages.printTemplate.enabledTag") : $t("pages.printTemplate.disabledTag") }}
        </ElTag>
      </template>

      <!-- 内置标记 -->
      <template #isBuiltIn="scope: any">
        <ElTag size="small" effect="plain" round :type="scope.row.isBuiltIn ? 'warning' : 'info'">
          {{ scope.row.isBuiltIn ? $t("pages.printTemplate.builtInTag") : $t("pages.printTemplate.customTag") }}
        </ElTag>
      </template>
    </ProPage>

    <!-- 设计器对话框（create/edit/copy 三态） -->
    <DesignerDialog ref="designerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";
import { ElTag } from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import DesignerDialog from "./designer-dialog.vue";

import {
  fetchListPrintTemplates,
  useDeletePrintTemplate,
} from "@/api/composables/print_template";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/core/i18n";

const { mutateAsync: deleteTemplate } = useDeletePrintTemplate();

const pageRef = ref();
const designerRef = ref();

const pageConfig = computed<ProPageConfig>(() => ({
  skeleton: true,
  search: {
    grid: true,
    fields: [
      {
        type: "input",
        label: $t("pages.printTemplate.name"),
        field: "name",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "input",
        label: $t("pages.printTemplate.code"),
        field: "code",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
    ],
  },

  table: {
    listAction: async (query: any) => {
      const { page, pageSize, ...queryParams } = query;
      const result = await fetchListPrintTemplates(
        new PaginationQuery({
          paging: { page: page || 1, pageSize: pageSize || 10 },
          formValues: queryParams,
        })
      );
      return { items: result.items || [], total: result.total || 0 };
    },
    deleteAction: async (ids: string) => {
      await deleteTemplate({ id: ids as any });
    },
    toolbar: [],
    toolbarRight: ["add"],
    defaultToolbar: ["exports", "refresh", "zoom", "filter"],
    tableAttrs: { border: true, stripe: true },
    emptyActionText: "common.button.add",
    columns: [
      { type: "index", label: $t("common.table.seq"), width: 60 },
      { prop: "name", label: $t("pages.printTemplate.name"), minWidth: 180 },
      { prop: "code", label: $t("pages.printTemplate.code"), minWidth: 190 },
      { prop: "resourceKey", label: $t("pages.printTemplate.resourceKey"), minWidth: 140, showOverflow: "title" },
      { prop: "paperSize", label: $t("pages.printTemplate.paperSize"), width: 100 },
      {
        prop: "orientation",
        label: $t("pages.printTemplate.orientation"),
        width: 100,
        slotName: "orientation",
      },
      {
        prop: "isEnabled",
        label: $t("pages.printTemplate.isEnabled"),
        width: 90,
        slotName: "isEnabled",
      },
      {
        prop: "isBuiltIn",
        label: $t("pages.printTemplate.isBuiltIn"),
        width: 90,
        slotName: "isBuiltIn",
      },
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
        width: 190,
        cellType: "tool",
        buttons: [
          { name: "edit", label: $t("common.button.edit"), icon: "lucide:pen-line" },
          { name: "copy", label: $t("common.button.copy"), icon: "lucide:copy" },
          {
            name: "delete",
            label: $t("common.button.delete"),
            icon: "lucide:trash-2",
            attrs: { type: "danger" },
            visible: (row: any) => !row.isBuiltIn,
          },
        ],
      },
    ],
  },
}));

function handleAdd() {
  designerRef.value?.open({ create: true });
}

function handleEdit(row: any) {
  designerRef.value?.open({ row });
}

/** ProPage 对未登记的 tool 按钮名走 operate 透传——copy 在此接设计器（复制派生走新建） */
function handleOperate(data: { name: string; row: any }) {
  if (data.name === "copy") {
    designerRef.value?.open({ copy: true, row: data.row });
  }
}

function handleSuccess() {
  pageRef.value?.refresh();
}
</script>
