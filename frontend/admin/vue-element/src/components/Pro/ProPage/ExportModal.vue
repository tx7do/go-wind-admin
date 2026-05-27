<template>
  <ElDialog
    v-model="state.visible"
    :title="t('pages.curd.export.title')"
    width="700px"
    align-center
    append-to-body
    :lock-scroll="false"
    @close="handleClose"
  >
    <ElScrollbar max-height="70vh">
      <ElForm
        ref="formRef"
        :model="state.form"
        :rules="formRules"
        label-width="120px"
        label-position="left"
        style="padding-right: var(--el-dialog-padding-primary)"
      >
        <ElFormItem :label="t('pages.curd.export.filename')" prop="filename">
          <ElInput
            v-model="state.form.filename"
            :placeholder="t('pages.curd.export.filenamePlaceholder')"
            clearable
          />
        </ElFormItem>

        <ElFormItem :label="t('pages.curd.export.fileType')" prop="fileType">
          <ElSelect v-model="state.form.fileType">
            <ElOption
              v-for="type in fileTypes"
              :key="type.value"
              :label="type.label"
              :value="type.value"
            />
          </ElSelect>
        </ElFormItem>

        <ElFormItem :label="t('pages.curd.export.dataScope')" prop="origin">
          <ElSelect v-model="state.form.origin">
            <ElOption :label="t('pages.curd.export.originOptions.current')" value="current" />
            <ElOption
              :label="t('pages.curd.export.originOptions.selected')"
              value="selected"
              :disabled="selectionData.length === 0"
            />
          </ElSelect>
        </ElFormItem>

        <ElFormItem :label="t('pages.curd.export.fields')" prop="fields">
          <div class="fields-selector">
            <div class="fields-header">
              <ElCheckbox
                :model-value="isAllFieldsSelected"
                :indeterminate="isIndeterminate"
                @change="handleToggleAllFields"
              >
                {{ t("pages.curd.export.allFields") }}
              </ElCheckbox>
            </div>
            <div class="fields-list">
              <ElCheckboxGroup v-model="state.form.fields">
                <ElCheckbox
                  v-for="col in exportableColumns"
                  :key="col.prop"
                  :value="col.prop"
                  :label="col.label"
                />
              </ElCheckboxGroup>
            </div>
          </div>
        </ElFormItem>

        <ElFormItem :label="t('pages.curd.export.params')" prop="params">
          <div class="params-group">
            <ElCheckbox v-model="state.form.showHeader">
              {{ t("pages.curd.export.paramsOptions.header") }}
            </ElCheckbox>
            <ElCheckbox v-model="state.form.showFooter">
              {{ t("pages.curd.export.paramsOptions.footer") }}
            </ElCheckbox>
            <ElCheckbox v-model="state.form.showRawData">
              {{ t("pages.curd.export.paramsOptions.rawData") }}
            </ElCheckbox>
          </div>
          <div class="params-group">
            <ElCheckbox v-model="state.form.showGroupHeader">
              {{ t("pages.curd.export.paramsOptions.groupHeader") }}
            </ElCheckbox>
            <ElCheckbox v-model="state.form.mergeCells">
              {{ t("pages.curd.export.paramsOptions.mergeCells") }}
            </ElCheckbox>
            <ElCheckbox v-model="state.form.showStyle">
              {{ t("pages.curd.export.paramsOptions.style") }}
            </ElCheckbox>
          </div>
        </ElFormItem>
      </ElForm>
    </ElScrollbar>

    <template #footer>
      <ElButton @click="handleClose">{{ t("common.button.cancel") }}</ElButton>
      <ElButton type="primary" @click="handleSubmit">
        {{ t("pages.curd.export.export") }}
      </ElButton>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
import { reactive, ref, computed, nextTick } from "vue";
import {
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
} from "element-plus";
import type { FormInstance, FormRules } from "element-plus";
import { useThrottleFn } from "@vueuse/core";
import { useI18n } from "@/i18n";
import type { ProTableColumn } from "../ProTable/types";

const props = defineProps<{
  columns: ProTableColumn[];
  selectionData: Record<string, any>[];
  tableData: Record<string, any>[];
  exportsAction?: (queryParams: any) => Promise<any[]>;
  searchParams?: Record<string, any>;
  defaultFilename?: string;
}>();

const { t } = useI18n();

const formRef = ref<FormInstance>();

// 可导出的列（有 prop 和 label 的普通列）
const exportableColumns = computed(() =>
  props.columns.filter((col: any) => col.prop && col.label && !col.type)
);

const defaultFields = computed(() =>
  exportableColumns.value.map((col: any) => col.prop).filter(Boolean)
);

const fileTypes = computed(() => [
  { label: t("pages.curd.export.fileTypeOptions.csv"), value: "csv" },
  { label: t("pages.curd.export.fileTypeOptions.html"), value: "html" },
  { label: t("pages.curd.export.fileTypeOptions.xml"), value: "xml" },
  { label: t("pages.curd.export.fileTypeOptions.txt"), value: "txt" },
]);

const state = reactive<{
  visible: boolean;
  form: {
    filename: string;
    fileType: "csv" | "html" | "xml" | "txt";
    fields: string[];
    origin: "current" | "selected";
    showHeader: boolean;
    showFooter: boolean;
    showRawData: boolean;
    showGroupHeader: boolean;
    mergeCells: boolean;
    showStyle: boolean;
  };
}>({
  visible: false,
  form: {
    filename: "",
    fileType: "csv",
    fields: [],
    origin: "current",
    showHeader: true,
    showFooter: false,
    showRawData: false,
    showGroupHeader: false,
    mergeCells: false,
    showStyle: false,
  },
});

const formRules: FormRules = {
  fields: [{ required: true, message: t("pages.curd.message.selectFields") }],
  origin: [{ required: true, message: t("pages.curd.message.selectOrigin") }],
};

// 全部字段是否选中
const isAllFieldsSelected = computed(
  () =>
    exportableColumns.value.length > 0 &&
    state.form.fields.length === exportableColumns.value.length
);

// 是否部分选中
const isIndeterminate = computed(
  () => state.form.fields.length > 0 && state.form.fields.length < exportableColumns.value.length
);

function handleToggleAllFields(checked: boolean | string | number) {
  if (checked) {
    state.form.fields = exportableColumns.value.map((col: any) => col.prop);
  } else {
    state.form.fields = [];
  }
}

function open() {
  state.form.fields = [...defaultFields.value];
  state.visible = true;
}

function handleClose() {
  state.visible = false;
  nextTick(() => formRef.value?.clearValidate());
}

const handleSubmit = useThrottleFn(() => {
  formRef.value?.validate((valid: boolean) => {
    if (!valid) return;
    doExport();
    handleClose();
  });
}, 3000);

function doExport() {
  const filename = state.form.filename || props.defaultFilename || "export";
  const rows = state.form.origin === "selected" ? props.selectionData : (props.tableData ?? []);

  // 获取选中的字段
  const selectedFields = exportableColumns.value.filter(
    (col: any) => col.prop && col.label && state.form.fields.includes(col.prop)
  );

  // 根据文件类型生成不同格式
  switch (state.form.fileType) {
    case "csv":
      exportCSV(filename, selectedFields, rows);
      break;
    case "html":
      exportHTML(filename, selectedFields, rows);
      break;
    case "xml":
      exportXML(filename, selectedFields, rows);
      break;
    case "txt":
      exportTXT(filename, selectedFields, rows);
      break;
  }
}

// 导出 CSV
function exportCSV(filename: string, fields: any[], rows: any[]) {
  const headers = fields.map((col) => col.label);
  const keys = fields.map((col) => col.prop);

  let csvContent = headers.join(",") + "\n";
  rows.forEach((row) => {
    const values = keys.map((key) => {
      const value = row[key] ?? "";
      return `"${String(value).replace(/"/g, '""')}"`;
    });
    csvContent += values.join(",") + "\n";
  });

  saveFile(csvContent, `${filename}.csv`, "text/csv;charset=utf-8");
}

// 导出 HTML
function exportHTML(filename: string, fields: any[], rows: any[]) {
  const headers = fields.map((col) => col.label);
  const keys = fields.map((col) => col.prop);

  let htmlContent = `<html><head><meta charset="UTF-8"></head><body><table border="1">\n`;
  htmlContent += "<thead><tr>" + headers.map((h) => `<th>${h}</th>`).join("") + "</tr></thead>\n";
  htmlContent += "<tbody>\n";
  rows.forEach((row) => {
    htmlContent += "<tr>" + keys.map((key) => `<td>${row[key] ?? ""}</td>`).join("") + "</tr>\n";
  });
  htmlContent += "</tbody></table></body></html>";

  saveFile(htmlContent, `${filename}.html`, "text/html;charset=utf-8");
}

// 导出 XML
function exportXML(filename: string, fields: any[], rows: any[]) {
  const keys = fields.map((col) => col.prop);

  let xmlContent = '<?xml version="1.0" encoding="UTF-8"?>\n<data>\n';
  rows.forEach((row) => {
    xmlContent += "  <row>\n";
    keys.forEach((key) => {
      xmlContent += `    <${key}>${escapeXml(row[key] ?? "")}</${key}>\n`;
    });
    xmlContent += "  </row>\n";
  });
  xmlContent += "</data>";

  saveFile(xmlContent, `${filename}.xml`, "application/xml;charset=utf-8");
}

// 导出 TXT
function exportTXT(filename: string, fields: any[], rows: any[]) {
  const headers = fields.map((col) => col.label);
  const keys = fields.map((col) => col.prop);

  let txtContent = headers.join("\t") + "\n";
  rows.forEach((row) => {
    const values = keys.map((key) => String(row[key] ?? "").replace(/\t/g, "    "));
    txtContent += values.join("\t") + "\n";
  });

  saveFile(txtContent, `${filename}.txt`, "text/plain;charset=utf-8");
}

// XML 转义
function escapeXml(str: string): string {
  return String(str)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&apos;");
}

// 通用保存方法
function saveFile(content: string, filename: string, mimeType: string) {
  const blob = new Blob([content], { type: mimeType });
  const url = window.URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  window.URL.revokeObjectURL(url);
}

defineExpose({ open });
</script>

<style scoped>
.fields-selector {
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
}

.fields-header {
  padding: 8px 12px;
  background: var(--el-fill-color-light);
  border-bottom: 1px solid var(--el-border-color);
}

.fields-list {
  max-height: 200px;
  overflow-y: auto;
  padding: 8px 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.params-group {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  margin-bottom: 8px;
}

.params-group:last-child {
  margin-bottom: 0;
}
</style>
