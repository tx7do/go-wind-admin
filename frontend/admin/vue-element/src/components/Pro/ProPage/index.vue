<template>
  <div class="pro-page">
    <!-- 搜索区 -->
    <ProSearch
      v-if="searchVisible && config.search?.fields?.length"
      :fields="config.search.fields"
      :colon="config.search.colon"
      :is-expandable="config.search.isExpandable"
      :show-number="config.search.showNumber"
      :grid="config.search.grid"
      @search="handleSearch"
      @reset="handleReset"
    />

    <!-- 表格区 -->
    <ElCard shadow="never" class="pro-page__table-card">
      <!-- 工具栏 -->
      <ProToolbar
        :left-buttons="leftButtons"
        :right-buttons="rightButtons"
        :default-toolbar="defaultToolbarButtons"
        :perm-prefix="config.permPrefix"
        @button-click="handleToolbarClick"
        @refresh="handleRefresh"
        @search="toggleSearch"
        @export="handleExport"
        @import="handleImport"
      >
        <template #left><slot name="toolbar-left" /></template>
        <template #right><slot name="toolbar-right" /></template>
      </ProToolbar>

      <!-- 表格 -->
      <ProTable
        ref="tableRef"
        :columns="config.table.columns"
        :data="tableData"
        :loading="tableState.loading.value"
        :row-key="rowKey"
        :table="config.table.tableAttrs"
        :pagination="tableState.showPagination"
        :total="tableState.pagination.total"
        :current-page="tableState.pagination.currentPage"
        :page-size="tableState.pagination.pageSize"
        :page-sizes="tableState.pagination.pageSizes"
        @selection-change="tableState.handleSelectionChange"
        @modify="handleModify"
        @operate="handleOperate"
        @current-change="handlePageChange"
        @size-change="handleSizeChange"
      >
        <!-- 透传自定义列插槽 -->
        <template v-for="(_, name) in $slots" :key="name" #[name]="slotProps">
          <slot :name="name" v-bind="slotProps" />
        </template>
      </ProTable>
    </ElCard>

    <!-- 弹窗 -->
    <ProModal
      v-if="config.modal"
      v-model:visible="modalState.visible.value"
      :mode="modalState.mode.value"
      :config="modalConfig"
      :form-data="modalState.formData as any"
      @submit="handleModalSubmit"
    >
      <template v-for="(_, name) in modalSlots" :key="name" #[name]="slotProps">
        <slot :name="name" v-bind="slotProps" />
      </template>
    </ProModal>
  </div>
</template>

<script setup lang="ts" generic="T extends Record<string, any>, Q extends Record<string, any>">
import { reactive, computed, ref, useSlots } from "vue";
import { ElMessage, ElMessageBox, ElCard } from "element-plus";
import { useI18n } from "@/i18n";

import ProSearch from "../ProSearch/index.vue";
import ProTable from "../ProTable/index.vue";
import ProModal from "../ProModal/index.vue";
import ProToolbar from "../ProToolbar/index.vue";

import { useTableState } from "../composables/useTableState";
import { useModalState } from "../composables/useModalState";

import type { ToolbarButton } from "../ProToolbar/types";
import type { ProPageConfig, ToolsButton, ToolbarRight } from "./types";

const props = defineProps<{ config: ProPageConfig<T, Q> }>();
const emit = defineEmits<{
  search: [params: Q];
  reset: [params: Q];
  add: [];
  edit: [row: T];
  view: [row: T];
  delete: [ids: string];
  operate: [data: { name: string; row: T; $index: number }];
  toolbar: [name: string];
}>();

const { t } = useI18n();
const slots = useSlots();

// === 基础状态 ===
const rowKey = props.config.rowKey ?? "id";
const searchParams = reactive<Q>({} as Q);
const searchVisible = ref(true);

const tableState = useTableState<T, Q>({
  indexAction: props.config.table.listAction as any,
  rowKey,
  pagination: props.config.table.pagination,
  request: props.config.table.request,
});

const modalState = useModalState<T>(rowKey);
const tableRef = ref<any>(null);

const tableData = computed(() => tableState.data.value);

// === 区分表格插槽和弹窗插槽 ===
const tableColumnProps = computed(() =>
  props.config.table.columns
    .filter((c) => c.cellType === "custom" || c.slotName)
    .map((c) => c.slotName ?? c.prop)
);
const modalSlots = computed(() => {
  const result: Record<string, any> = {};
  for (const name of Object.keys(slots)) {
    if (!tableColumnProps.value.includes(name as any)) {
      result[name] = slots[name];
    }
  }
  return result;
});

// === 工具栏按钮转换（参考CURD createToolbar） ===
const builtinButtons: Record<string, { textKey: string; attrs: Record<string, any> }> = {
  add: { textKey: "common.button.add", attrs: { icon: "plus", type: "success" } },
  delete: {
    textKey: "common.button.delete",
    attrs: { icon: "delete", type: "danger" },
  },
  import: { textKey: "common.button.import", attrs: { icon: "upload" } },
  export: { textKey: "common.button.export", attrs: { icon: "download" } },
};

function toToolbarButtons(
  items: Array<string | ToolsButton> | undefined,
  defaultAttrs: Record<string, any> = {}
): ToolbarButton[] {
  if (!items?.length) return [];
  return items.map((item) => {
    if (typeof item === "string") {
      const cfg = builtinButtons[item];
      return {
        name: item,
        text: cfg ? t(cfg.textKey) : item,
        type: item as ToolbarButton["type"],
        icon: cfg?.attrs?.icon,
        attrs: { ...defaultAttrs, ...cfg?.attrs },
      };
    }
    return {
      name: item.name,
      text: item.text ?? (item.textKey ? t(item.textKey) : item.name),
      attrs: { ...defaultAttrs, ...item.attrs },
      perm: item.perm,
      visible: item.visible,
    };
  });
}

const leftButtons = computed(() => toToolbarButtons(props.config.table.toolbar));
const rightButtons = computed(() => toToolbarButtons(props.config.table.toolbarRight));
const defaultToolbarButtons = computed(() => {
  const dt = props.config.table.defaultToolbar;
  if (!dt?.length)
    return ["refresh", "filter", "search"] as (
      | "refresh"
      | "filter"
      | "search"
      | "imports"
      | "exports"
    )[];
  return dt
    .filter((item): item is ToolbarRight => typeof item === "string")
    .map((item) => item as "refresh" | "filter" | "search" | "exports" | "imports");
});

// === 搜索处理 ===
function handleSearch(params: Q) {
  Object.keys(searchParams).forEach((k) => delete (searchParams as any)[k]);
  Object.assign(searchParams, params);
  tableState.fetch(searchParams, true);
  emit("search", { ...searchParams } as Q);
}

function handleReset(params: Q) {
  Object.keys(searchParams).forEach((k) => delete (searchParams as any)[k]);
  Object.assign(searchParams, params);
  tableState.fetch(searchParams, true);
  emit("reset", { ...searchParams } as Q);
}

function toggleSearch() {
  searchVisible.value = !searchVisible.value;
}

// === 工具栏事件 ===
function handleToolbarClick(name: string) {
  switch (name) {
    case "add":
      modalState.open("add");
      emit("add");
      break;
    case "delete":
      handleBatchDelete();
      break;
    case "export":
      handleExport();
      break;
    case "import":
      handleImport();
      break;
    default:
      emit("toolbar", name);
      break;
  }
}

function handleRefresh() {
  tableState.fetch(searchParams);
}

// === 表格事件 ===
function handleModify(data: { row: T; field: string; value: any }) {
  if (props.config.table.modifyAction) {
    props.config.table.modifyAction({
      [rowKey]: (data.row as any)[rowKey],
      field: data.field,
      value: data.value,
    });
  }
}

function handleOperate(data: { name: string; row: T; $index: number }) {
  switch (data.name) {
    case "edit":
      modalState.open("edit", data.row);
      emit("edit", data.row);
      break;
    case "view":
      modalState.open("view", data.row);
      emit("view", data.row);
      break;
    case "delete":
      handleDelete(data.row);
      break;
    default:
      emit("operate", data);
      break;
  }
}

// === 删除（参考CURD PageContent） ===
async function handleBatchDelete() {
  const ids = tableState.getSelectionIds().join(",");
  if (!ids) {
    ElMessage.warning(t("pages.curd.message.selectDeleteItems"));
    return;
  }
  await ElMessageBox.confirm(t("pages.curd.message.confirmDelete"), t("common.title.confirm"), {
    confirmButtonText: t("common.button.confirm"),
    cancelButtonText: t("common.button.cancel"),
    type: "warning",
  });
  await props.config.table.deleteAction?.(ids);
  ElMessage.success(t("pages.curd.message.deleteSuccess"));
  tableState.fetch(searchParams, true);
}

async function handleDelete(row: T) {
  const id = String((row as any)[rowKey]);
  await ElMessageBox.confirm(t("pages.curd.message.confirmDelete"), t("common.title.confirm"), {
    confirmButtonText: t("common.button.confirm"),
    cancelButtonText: t("common.button.cancel"),
    type: "warning",
  });
  await props.config.table.deleteAction?.(id);
  ElMessage.success(t("pages.curd.message.deleteSuccess"));
  tableState.fetch(searchParams, true);
}

// === 导入导出（参考CURD PageContent） ===
async function handleExport() {
  if (!props.config.table.exportAction) {
    ElMessage.warning("未配置导出功能");
    return;
  }
  const response = await props.config.table.exportAction({ ...searchParams } as Q);
  if (response?.data) {
    const fileData = response.data;
    const disposition = response.headers?.["content-disposition"] ?? "";
    const fileName = disposition
      ? decodeURI(disposition.split(";")[1]?.split("=")[1] ?? "export.xlsx")
      : "export.xlsx";
    saveXlsx(fileData, fileName);
  }
}

function handleImport() {
  if (!props.config.table.importAction) {
    ElMessage.warning("未配置导入功能");
    return;
  }
  const input = document.createElement("input");
  input.type = "file";
  input.accept = ".xlsx,.xls";
  input.onchange = async (e: Event) => {
    const file = (e.target as HTMLInputElement).files?.[0];
    if (file) {
      await props.config.table.importAction!(file);
      ElMessage.success(t("pages.curd.message.importSuccess"));
      tableState.fetch(searchParams, true);
    }
  };
  input.click();
}

// === 分页 ===
function handlePageChange(page: number) {
  tableState.pagination.currentPage = page;
  tableState.fetch(searchParams);
}

function handleSizeChange(size: number) {
  tableState.pagination.pageSize = size;
  tableState.fetch(searchParams, true);
}

// === 弹窗配置映射 ===
const modalConfig = computed(() => {
  if (!props.config.modal) return {} as any;
  return {
    component: props.config.modal.component,
    dialog: props.config.modal.dialog,
    drawer: props.config.modal.drawer,
    form: props.config.modal.form,
    colon: props.config.modal.colon,
    rowKey: rowKey,
    fields: props.config.modal.fields ?? [],
    submitAction: props.config.modal.submitAction,
    beforeSubmit: props.config.modal.beforeSubmit,
  };
});

function handleModalSubmit() {
  tableState.fetch(searchParams, true);
}

// === 工具函数 ===
function saveXlsx(fileData: any, fileName: string) {
  const fileType =
    "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet;charset=utf-8";
  const blob = new Blob([fileData], { type: fileType });
  const url = window.URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = fileName;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  window.URL.revokeObjectURL(url);
}

// 初始化加载数据
tableState.fetch(searchParams);

defineExpose({
  tableRef,
  tableState,
  modalState,
  searchParams,
  refresh: () => tableState.fetch(searchParams),
});
</script>

<style scoped>
.pro-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
  height: 100%;
}

.pro-page__table-card {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
</style>
