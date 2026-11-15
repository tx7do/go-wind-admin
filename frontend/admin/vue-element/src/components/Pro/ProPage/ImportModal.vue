<template>
  <ElDialog
    v-model="state.visible"
    :title="t('pages.curd.import.title')"
    width="600px"
    align-center
    append-to-body
    :lock-scroll="false"
    @close="handleClose"
  >
    <ElScrollbar max-height="60vh">
      <ElForm
        ref="formRef"
        :model="state"
        :rules="formRules"
        style="padding-right: var(--el-dialog-padding-primary)"
      >
        <ElFormItem :label="t('pages.curd.import.file')" prop="files">
          <ElUpload
            ref="uploadRef"
            v-model:file-list="state.files"
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
                  v-if="importTemplate"
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
      <ElButton type="primary" :disabled="state.files.length === 0" @click="handleSubmit">
        {{ t("common.button.confirm") }}
      </ElButton>
      <ElButton @click="handleClose">{{ t("common.button.cancel") }}</ElButton>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
import { reactive, ref, nextTick } from "vue";
import {
  ElMessage,
  ElDialog,
  ElForm,
  ElFormItem,
  ElScrollbar,
  ElButton,
  ElUpload,
  ElIcon,
  ElLink,
} from "element-plus";
import { UploadFilled } from "@element-plus/icons-vue";
import type { FormInstance, FormRules, UploadInstance, UploadRawFile } from "element-plus";
import { genFileId } from "element-plus";
import { useThrottleFn } from "@vueuse/core";
import ExcelJS from "exceljs";
import { useI18n } from "@go-wind/admin-element";

const props = defineProps<{
  importsAction?: (data: Record<string, any>[]) => Promise<any>;
  importAction?: (file: File) => Promise<any>;
  importTemplate?: string | (() => Promise<any>);
}>();

const emit = defineEmits<{
  success: [];
}>();

const { t } = useI18n();

const formRef = ref<FormInstance>();
const uploadRef = ref<UploadInstance>();

const state = reactive<{
  visible: boolean;
  files: any[];
}>({
  visible: false,
  files: [],
});

const formRules: FormRules = {
  files: [{ required: true, message: t("pages.curd.message.selectFile") }],
};

function open() {
  state.files = [];
  state.visible = true;
}

function handleClose() {
  state.visible = false;
  nextTick(() => formRef.value?.clearValidate());
}

function handleFileExceed(files: File[]) {
  uploadRef.value!.clearFiles();
  const file = files[0] as UploadRawFile;
  file.uid = genFileId();
  uploadRef.value!.handleStart(file);
}

function handleDownloadTemplate() {
  const tpl = props.importTemplate;
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

const handleSubmit = useThrottleFn(() => {
  formRef.value?.validate((valid: boolean) => {
    if (!valid) return;
    doImport();
  });
}, 3000);

async function doImport() {
  const file = state.files[0]?.raw as File;
  if (!file) return;

  // 如果配置了 importsAction（批量导入），解析 Excel 后传后端
  if (props.importsAction) {
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
        props.importsAction!(data)
          .then(() => {
            ElMessage.success(t("pages.curd.message.importSuccess"));
            handleClose();
            emit("success");
          })
          .catch((err: any) => {
            ElMessage.error(
              `${t("pages.curd.message.importFailed")}: ${err?.message ?? ""}`
            );
          });
      })
      .catch(() => {
        ElMessage.error(t("pages.curd.message.readFileFailed"));
      });
    };
    return;
  }

  // 否则使用 importAction（文件直接上传）
  if (!props.importAction) {
    ElMessage.warning(t("pages.curd.message.noImportAction"));
    return;
  }
  await props.importAction(file);
  ElMessage.success(t("pages.curd.message.importSuccess"));
  handleClose();
  emit("success");
}

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

defineExpose({ open });
</script>
