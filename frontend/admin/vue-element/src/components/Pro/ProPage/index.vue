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
        :perm-prefix="config.permPrefix"
        @button-click="handleToolbarClick"
        @refresh="handleRefresh"
        @search="toggleSearch"
        @export="openExportsModal"
        @import="openImportsModal"
        @zoom="handleZoom"
      >
        <template #left><slot name="toolbar-left" /></template>
        <template #before-tools><slot name="toolbar-before-tools" /></template>
        <template v-if="!!$slots['toolbar-filter']" #filter><slot name="toolbar-filter" /></template>
        <template #right><slot name="toolbar-right" /></template>
      </ProToolbar>

      <!-- 表格 -->
      <ProTable
        ref="tableRef"
        :engine="config.engine"
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
    </div>

    <!-- 导出弹窗 -->
    <ElDialog
      v-model="exportsState.visible"
      :title="t('pages.curd.export.title')"
      width="600px"
      align-center
      @close="closeExportsModal"
    >
      <ElScrollbar max-height="60vh">
        <ElForm
          ref="exportsFormRef"
          :model="exportsState.form"
          :rules="exportsFormRules"
          style="padding-right: var(--el-dialog-padding-primary)"
        >
          <ElFormItem :label="t('pages.curd.export.filename')" prop="filename">
            <ElInput v-model="exportsState.form.filename" clearable />
          </ElFormItem>
          <ElFormItem :label="t('pages.curd.export.sheetname')" prop="sheetname">
            <ElInput v-model="exportsState.form.sheetname" clearable />
          </ElFormItem>
          <ElFormItem :label="t('pages.curd.export.origin')" prop="origin">
            <ElSelect v-model="exportsState.form.origin">
              <ElOption :label="t('pages.curd.export.originOptions.current')" value="current" />
              <ElOption
                :label="t('pages.curd.export.originOptions.selected')"
                value="selected"
                :disabled="selectionData.length === 0"
              />
              <ElOption
                :label="t('pages.curd.export.originOptions.remote')"
                value="remote"
                :disabled="!config.table.exportsAction"
              />
            </ElSelect>
          </ElFormItem>
          <ElFormItem :label="t('pages.curd.export.fields')" prop="fields">
            <ElCheckboxGroup v-model="exportsState.form.fields">
              <ElCheckbox
                v-for="col in exportableColumns"
                :key="col.prop"
                :value="col.prop"
                :label="col.label"
              />
            </ElCheckboxGroup>
          </ElFormItem>
        </ElForm>
      </ElScrollbar>
      <template #footer>
        <ElButton type="primary" @click="submitExports">
          {{ t("common.button.confirm") }}
        </ElButton>
        <ElButton @click="closeExportsModal">{{ t("common.button.cancel") }}</ElButton>
      </template>
    </ElDialog>

    <!-- 导入弹窗 -->
    <ElDialog
      v-model="importsState.visible"
      :title="t('pages.curd.import.title')"
      width="600px"
      align-center
      @close="closeImportsModal"
    >
      <ElScrollbar max-height="60vh">
        <ElForm
          ref="importsFormRef"
          :model="importsState"
          :rules="importsFormRules"
          style="padding-right: var(--el-dialog-padding-primary)"
        >
          <ElFormItem :label="t('pages.curd.import.file')" prop="files">
            <ElUpload
              ref="uploadRef"
              v-model:file-list="importsState.files"
              class="w-full"
              accept="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet, application/vnd.ms-excel"
              :drag="true"
              :limit="1"
              :auto-upload="false"
              :on-exceed="handleFileExceed"
            >
              <ElIcon class="el-icon--upload"><UploadFilled /></ElIcon>
              <div class="el-upload__text">
                <span>{{ t("pages.curd.import.dragText") }}</span>
                <em>{{ t("pages.curd.import.clickText") }}</em>
              </div>
              <template #tip>
                <div class="el-upload__tip">
                  {{ t("pages.curd.import.fileTypeTip") }}
                  <ElLink
                    v-if="config.table.importTemplate"
                    type="primary"
                    icon="download"
                    underline="never"
                    @click="handleDownloadTemplate"
                  >
                    {{ t("pages.curd.import.downloadTemplate") }}
                  </ElLink>
                </div>
              </template>
            </ElUpload>
          </ElFormItem>
        </ElForm>
      </ElScrollbar>
      <template #footer>
        <ElButton
          type="primary"
          :disabled="importsState.files.length === 0"
          @click="submitImports"
        >
          {{ t("common.button.confirm") }}
        </ElButton>
        <ElButton @click="closeImportsModal">{{ t("common.button.cancel") }}</ElButton>
      </template>
    </ElDialog>

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
import { reactive, computed, ref, nextTick, useSlots } from "vue";
import {
  ElMessage,
  ElMessageBox,
  ElDialog,
  ElForm,
  ElFormItem,
  ElInput,
  ElSelect,
  ElOption,
  ElScrollbar,
  ElButton,
  ElCheckbox,
  ElCheckboxGroup,
  ElUpload,
  ElIcon,
  ElLink,
} from "element-plus";
import { UploadFilled } from "@element-plus/icons-vue";
import { useThrottleFn } from "@vueuse/core";
import type { FormInstance, FormRules, UploadInstance, UploadRawFile } from "element-plus";
import { genFileId } from "element-plus";
import ExcelJS from "exceljs";
import { useI18n } from "@/i18n";

import ProSearch from "../ProSearch/index.vue";
import ProTable from "../ProTable/index.vue";
import ProModal from "../ProModal/index.vue";
import ProToolbar from "../ProToolbar/index.vue";

import { useTableState } from "../composables/useTableState";
import { useModalState } from "../composables/useModalState";

import type { ToolbarButton } from "../ProToolbar/types";
import type {
  ProPageConfig,
  ToolsButton,
  ToolbarRight,
} from "./types";

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
const contentRef = ref<HTMLElement | null>(null);
const exportsFormRef = ref<FormInstance>();
const importsFormRef = ref<FormInstance>();
const uploadRef = ref<UploadInstance>();

const tableData = computed(() => tableState.data.value);

// === 选中数据（用于导出） ===
const selectionData = computed(() => tableRef.value?.getSelectionRows?.() ?? []);

// === filter 列显隐 ===
// vxe-table 引擎由 customConfig 接管列显隐；el-table 引擎使用 checkbox
const filterColumns = computed(() => {
  const cols = tableRef.value?.resolvedColumns;
  if (!cols) return [];
  return cols.filter(
    (col: any) =>
      col.prop && col.label && col.type !== "selection" && col.type !== "index",
  );
});

// === 可导出的列（有 prop 和 label 的普通列） ===
const exportableColumns = computed(() =>
  props.config.table.columns.filter(
    (col: any) => col.prop && col.label && !col.type,
  ),
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

// === 工具栏按钮转换（参考CURD createToolbar） ===
const builtinButtons: Record<string, { textKey: string; icon: string; attrs: Record<string, any> }> = {
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
    return ["refresh", "filter", "search"] as Array<ToolbarRight | ToolsButton>;
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

// === 导出弹窗（参考 CURD ExcelJS） ===
const defaultExportFields = exportableColumns.value.map((col: any) => col.prop);
const exportsState = reactive<{
  visible: boolean;
  form: {
    filename: string;
    sheetname: string;
    fields: string[];
    origin: "current" | "selected" | "remote";
  };
}>({
  visible: false,
  form: {
    filename: "",
    sheetname: "",
    fields: [...defaultExportFields],
    origin: "current",
  },
});
const exportsFormRules: FormRules = {
  fields: [{ required: true, message: t("pages.curd.message.selectFields") }],
  origin: [{ required: true, message: t("pages.curd.message.selectOrigin") }],
};

function openExportsModal() {
  // 重置字段为最新列
  exportsState.form.fields = exportableColumns.value.map((col: any) => col.prop);
  exportsState.visible = true;
}

function closeExportsModal() {
  exportsState.visible = false;
  nextTick(() => exportsFormRef.value?.clearValidate());
}

const submitExports = useThrottleFn(() => {
  exportsFormRef.value?.validate((valid: boolean) => {
    if (!valid) return;
    doExport();
    closeExportsModal();
  });
}, 3000);

function doExport() {
  const filename = exportsState.form.filename || props.config.permPrefix || "export";
  const sheetname = exportsState.form.sheetname || "sheet";
  const workbook = new ExcelJS.Workbook();
  const worksheet = workbook.addWorksheet(sheetname);
  const excelCols: Partial<ExcelJS.Column>[] = [];
  exportableColumns.value.forEach((col: any) => {
    if (col.prop && col.label && exportsState.form.fields.includes(col.prop)) {
      excelCols.push({ header: col.label, key: col.prop });
    }
  });
  worksheet.columns = excelCols;

  if (exportsState.form.origin === "remote") {
    if (!props.config.table.exportsAction) {
      ElMessage.error(t("pages.curd.message.noExportsAction"));
      return;
    }
    props.config.table.exportsAction({ ...searchParams } as Q).then((data) => {
      worksheet.addRows(data);
      workbook.xlsx.writeBuffer().then((buffer) => saveXlsx(buffer, filename));
    });
  } else {
    const rows =
      exportsState.form.origin === "selected"
        ? (tableRef.value?.getSelectionRows?.() ?? [])
        : (tableData.value ?? []);
    worksheet.addRows(rows);
    workbook.xlsx.writeBuffer().then((buffer) => saveXlsx(buffer, filename));
  }
}

// === 导入弹窗（参考 CURD ExcelJS） ===
const importsState = reactive<{
  visible: boolean;
  files: any[];
}>({
  visible: false,
  files: [],
});
const importsFormRules: FormRules = {
  files: [{ required: true, message: t("pages.curd.message.selectFile") }],
};

function openImportsModal() {
  importsState.files = [];
  importsState.visible = true;
}

function closeImportsModal() {
  importsState.visible = false;
  nextTick(() => importsFormRef.value?.clearValidate());
}

function handleFileExceed(files: File[]) {
  uploadRef.value!.clearFiles();
  const file = files[0] as UploadRawFile;
  file.uid = genFileId();
  uploadRef.value!.handleStart(file);
}

function handleDownloadTemplate() {
  const tpl = props.config.table.importTemplate;
  if (typeof tpl === "string") {
    window.open(tpl);
  } else if (typeof tpl === "function") {
    tpl().then((res: any) => {
      const disposition = res.headers?.["content-disposition"] ?? "";
      const name = disposition
        ? decodeURI(disposition.split(";")[1]?.split("=")[1] ?? "template.xlsx")
        : "template.xlsx";
      saveXlsx(res.data, name);
    });
  }
}

const submitImports = useThrottleFn(() => {
  importsFormRef.value?.validate((valid: boolean) => {
    if (!valid) return;
    doImport();
  });
}, 3000);

async function doImport() {
  const file = importsState.files[0]?.raw as File;
  if (!file) return;

  // 如果配置了 importsAction（批量导入），解析 Excel 后传后端
  if (props.config.table.importsAction) {
    const workbook = new ExcelJS.Workbook();
    const reader = new FileReader();
    reader.readAsArrayBuffer(file);
    reader.onload = (ev) => {
      const result = ev.target?.result as ArrayBuffer;
      if (!result) {
        ElMessage.error(t("pages.curd.message.readFileFailed"));
        return;
      }
      workbook.xlsx.load(result).then((wb) => {
        const ws = wb.getWorksheet(1);
        if (!ws) return;
        const fields: any[] = [];
        ws.getRow(1).eachCell((cell) => fields.push(cell.value));
        const data: Record<string, any>[] = [];
        for (let i = 2; i <= ws.rowCount; i++) {
          const row = ws.getRow(i);
          const rowData: Record<string, any> = {};
          row.eachCell((cell, colNumber) => {
            rowData[fields[colNumber - 1]] = cell.value;
          });
          data.push(rowData);
        }
        if (data.length === 0) {
          ElMessage.error(t("pages.curd.message.noDataParsed"));
          return;
        }
        props.config.table.importsAction!(data).then(() => {
          ElMessage.success(t("pages.curd.message.importSuccess"));
          closeImportsModal();
          handleRefresh();
        });
      });
    };
    return;
  }

  // 否则使用 importAction（文件直接上传）
  if (!props.config.table.importAction) {
    ElMessage.warning(t("pages.curd.message.noImportAction"));
    return;
  }
  await props.config.table.importAction(file);
  ElMessage.success(t("pages.curd.message.importSuccess"));
  closeImportsModal();
  handleRefresh();
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
    overflow: auto;
    padding: 8px 12px;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color);
    border-radius: 4px;
  }
}
</style>
