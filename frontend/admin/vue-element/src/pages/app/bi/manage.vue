<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage
      ref="pageRef"
      :config="pageConfig"
      @add="handleAdd"
      @edit="handleEdit"
      @operate="handleOperate"
    >
      <!-- 种类 -->
      <template #kind="scope: any">
        <ElTag size="small" round :type="kindTagType(scope.row.kind)">
          {{ kindLabel(scope.row.kind) }}
        </ElTag>
      </template>

      <!-- 启用状态 -->
      <template #isEnabled="scope: any">
        <ElTag size="small" round :type="scope.row.isEnabled ? 'success' : 'info'">
          {{ scope.row.isEnabled ? $t("pages.biDashboard.enabledTag") : $t("pages.biDashboard.disabledTag") }}
        </ElTag>
      </template>

      <!-- 内置标记 -->
      <template #isBuiltIn="scope: any">
        <ElTag size="small" effect="plain" round :type="scope.row.isBuiltIn ? 'warning' : 'info'">
          {{ scope.row.isBuiltIn ? $t("pages.biDashboard.builtInTag") : $t("pages.biDashboard.customTag") }}
        </ElTag>
      </template>
    </ProPage>

    <!-- 设计器抽屉（create/edit 两态） -->
    <DesignerDialog ref="designerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
/**
 * BI 仪表盘管理页（ele 端）：设计器 schema 载体列表；
 * 「设计」进全屏抽屉画布；内置行禁删可复制派生。
 */
import { computed, ref } from "vue";
import { ElTag } from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import DesignerDialog from "./designer-dialog.vue";

import {
  fetchListBiDashboards,
  useCopyBiDashboard,
  useDeleteBiDashboard,
} from "@/api/composables/bi_dashboard";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/core/i18n";

const { mutateAsync: deleteDashboard } = useDeleteBiDashboard();
const { mutateAsync: copyDashboard } = useCopyBiDashboard();

const pageRef = ref();
const designerRef = ref();

const KIND_MAP: Record<string, { label: string; type: "primary" | "warning" | "success" }> = {
  DASHBOARD: { label: "pages.biDashboard.kindDashboard", type: "primary" },
  SCREEN: { label: "pages.biDashboard.kindScreen", type: "warning" },
  TEMPLATE: { label: "pages.biDashboard.kindTemplate", type: "success" },
};

function kindLabel(kind?: string) {
  const meta = KIND_MAP[kind ?? ""] ?? KIND_MAP.DASHBOARD;
  return $t(meta.label);
}

function kindTagType(kind?: string) {
  return (KIND_MAP[kind ?? ""] ?? KIND_MAP.DASHBOARD).type;
}

const pageConfig = computed<ProPageConfig>(() => ({
  skeleton: true,
  search: {
    grid: true,
    fields: [
      {
        type: "input",
        label: $t("pages.biDashboard.name"),
        field: "name",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "input",
        label: $t("pages.biDashboard.code"),
        field: "code",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
    ],
  },

  table: {
    listAction: async (query: any) => {
      const { page, pageSize, ...queryParams } = query;
      const result = await fetchListBiDashboards(
        new PaginationQuery({
          paging: { page: page || 1, pageSize: pageSize || 10 },
          formValues: queryParams,
        })
      );
      return { items: result.items || [], total: result.total || 0 };
    },
    deleteAction: async (ids: string) => {
      await deleteDashboard({ id: ids as any });
    },
    toolbar: [],
    toolbarRight: ["add"],
    defaultToolbar: ["exports", "refresh", "zoom", "filter"],
    tableAttrs: { border: true, stripe: true },
    emptyActionText: "common.button.add",
    columns: [
      { type: "index", label: $t("common.table.seq"), width: 60 },
      { prop: "name", label: $t("pages.biDashboard.name"), minWidth: 170 },
      { prop: "code", label: $t("pages.biDashboard.code"), minWidth: 180 },
      { prop: "kind", label: $t("pages.biDashboard.kind"), width: 90, slotName: "kind" },
      { prop: "chartCount", label: $t("pages.biDashboard.chartCount"), width: 80 },
      { prop: "themeKey", label: $t("pages.biDashboard.themeKey"), width: 100 },
      {
        prop: "isEnabled",
        label: $t("pages.biDashboard.isEnabled"),
        width: 80,
        slotName: "isEnabled",
      },
      {
        prop: "isBuiltIn",
        label: $t("pages.biDashboard.isBuiltIn"),
        width: 80,
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
        width: 210,
        cellType: "tool",
        buttons: [
          { name: "design", label: $t("pages.biDashboard.openDesigner"), icon: "lucide:pencil-ruler" },
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
  designerRef.value?.open({});
}

function handleEdit(row: any) {
  designerRef.value?.open({ row });
}

/** ProPage 对未登记的 tool 按钮名走 operate 透传 */
function handleOperate(data: { name: string; row: any }) {
  if (data.name === "copy" && data.row?.id) {
    copyDashboard({ id: data.row.id }).then(() => {
      pageRef.value?.refresh();
    });
  }
  // design 与 edit 同义（更醒目的入口）
  if (data.name === "design") {
    designerRef.value?.open({ row: data.row });
  }
}

function handleSuccess() {
  pageRef.value?.refresh();
}
</script>
