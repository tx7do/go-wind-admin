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
    <div ref="contentRef" class="pro-page__content">
      <!-- 工具栏 -->
      <ProToolbar
        :left-buttons="leftButtons"
        :right-buttons="rightButtons"
        :default-toolbar="defaultToolbarButtons"
        :columns="filterColumns"
        @button-click="handleToolbarClick"
        @refresh="handleRefresh"
        @search="toggleSearch"
        @export="openExportsModal"
        @import="openImportsModal"
        @zoom="handleZoom"
        @filter-change="handleFilterChange"
      >
        <template #left><slot name="toolbar-left" /></template>
        <template #before-tools><slot name="toolbar-before-tools" /></template>
        <template v-if="!!$slots['toolbar-filter']" #filter>
          <slot name="toolbar-filter" />
        </template>
        <template #right><slot name="toolbar-right" /></template>
      </ProToolbar>

      <!-- 表格 -->
      <ProTable
        ref="tableRef"
        :engine="config.engine"
        :table-id="tableId"
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
        @row-click="handleRowClick"
      >
        <!-- 透传自定义列插槽 -->
        <template v-for="(_, name) in $slots" :key="name" #[name]="slotProps">
          <slot :name="name" v-bind="slotProps" />
        </template>
      </ProTable>
    </div>

    <!-- 导出弹窗 -->
    <ExportModal
      ref="exportModalRef"
      :columns="exportableColumns"
      :selection-data="selectionData"
      :table-data="tableData"
      :exports-action="config.table.exportsAction"
      :search-params="searchParams"
      :default-filename="config.exportFilename"
    />

    <!-- 导入弹窗 -->
    <ImportModal
      ref="importModalRef"
      :imports-action="config.table.importsAction"
      :import-action="config.table.importAction"
      :import-template="config.table.importTemplate"
      @success="handleRefresh"
    />

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
import { ElMessage, ElMessageBox } from "element-plus";
import { useRoute } from "vue-router";
import { useI18n } from "@/i18n";

import ProSearch from "../ProSearch/index.vue";
import ProTable from "../ProTable/index.vue";
import ProModal from "../ProModal/index.vue";
import ProToolbar from "../ProToolbar/index.vue";
import ExportModal from "./ExportModal.vue";
import ImportModal from "./ImportModal.vue";

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
  "row-click": [row: T];
}>();

const { t } = useI18n();
const slots = useSlots();
const route = useRoute();

// === 基础状态 ===
const rowKey = props.config.rowKey ?? "id";
// 自动生成 tableId：优先用 config.tableId，否则用路由路径
const tableId =
  props.config.tableId ?? (route.path.replace(/\//g, "_").replace(/^_/, "") || undefined);
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
const contentRef = ref<HTMLElement | null>(null);
const exportModalRef = ref<InstanceType<typeof ExportModal>>();
const importModalRef = ref<InstanceType<typeof ImportModal>>();

const tableData = computed(() => tableState.data.value);

// === 选中数据（用于导出） ===
const selectionData = computed(() => tableRef.value?.getSelectionRows?.() ?? []);

// === filter 列显隐 ===
// vxe-table 引擎由 customConfig 接管列显隐；el-table 引擎使用 checkbox
const filterColumns = computed(() => {
  const cols = tableRef.value?.resolvedColumns;
  if (!cols) return [];
  return cols.filter(
    (col: any) => col.prop && col.label && col.type !== "selection" && col.type !== "index"
  );
});

// === 可导出的列（有 prop 和 label 的普通列） ===
const exportableColumns = computed(() =>
  props.config.table.columns.filter((col: any) => col.prop && col.label && !col.type)
);

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

// === 工具栏按钮转换 ===
const builtinButtons: Record<
  string,
  { textKey: string; icon: string; attrs: Record<string, any> }
> = {
  add: { textKey: "common.button.add", icon: "plus", attrs: { type: "success" } },
  delete: {
    textKey: "common.button.delete",
    icon: "delete",
    attrs: { type: "danger" },
  },
  import: { textKey: "common.button.import", icon: "upload", attrs: {} },
  export: { textKey: "common.button.export", icon: "download", attrs: {} },
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
        icon: cfg?.icon,
        attrs: { ...defaultAttrs, ...cfg?.attrs },
      };
    }
    return {
      name: item.name,
      text: item.text ?? (item.textKey ? t(item.textKey) : item.name),
      attrs: { ...defaultAttrs, ...item.attrs },
      auth: item.auth,
      visible: item.visible,
    };
  });
}

const leftButtons = computed(() => toToolbarButtons(props.config.table.toolbar));
const rightButtons = computed(() => toToolbarButtons(props.config.table.toolbarRight));
const defaultToolbarButtons = computed(() => {
  const dt = props.config.table.defaultToolbar;
  if (!dt?.length) return ["refresh", "filter", "search"] as Array<ToolbarRight | ToolsButton>;
  return dt;
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

// === 筛选处理（filter 由 ProToolbar 内部 ElPopover + columns checkbox 处理） ===
function handleFilterChange(columns: any[]) {
  // 列配置已通过响应式更新，这里可以做额外处理
  // 比如保存到 localStorage 或后端
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
      openExportsModal();
      break;
    case "import":
      openImportsModal();
      break;
    case "refresh":
      handleRefresh();
      break;
    case "search":
      toggleSearch();
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

// === 删除 ===
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
    lockScroll: false,
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
    lockScroll: false,
  });
  await props.config.table.deleteAction?.(id);
  ElMessage.success(t("pages.curd.message.deleteSuccess"));
  tableState.fetch(searchParams, true);
}

// === 全屏缩放 ===
function handleZoom(isFullscreen: boolean) {
  const el = contentRef.value;
  if (!el) return;
  if (isFullscreen) {
    el.requestFullscreen?.();
  } else {
    document.exitFullscreen?.();
  }
}

function openExportsModal() {
  exportModalRef.value?.open();
}

function openImportsModal() {
  importModalRef.value?.open();
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

// === 行点击 ===
function handleRowClick(row: T) {
  emit("row-click", row);
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
    rowKey,
    fields: props.config.modal.fields ?? [],
    submitAction: props.config.modal.submitAction,
    beforeSubmit: props.config.modal.beforeSubmit,
  };
});

function handleModalSubmit() {
  tableState.fetch(searchParams, true);
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

<style scoped lang="scss">
.pro-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
  height: 100%;

  &__content {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-height: 0;
    overflow: hidden;
    padding: 12px;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
    box-shadow: 0 1px 3px 0 rgb(0 0 0 / 5%);
  }
}
</style>
