<script lang="ts" setup>
import { ref } from 'vue';

import { $t } from '@vben/locales';

import { message } from 'ant-design-vue';

import { generateImportTemplate, runImportFile, type ImportField } from '#/utils/import';

/**
 * 通用 Excel 导入弹窗：客户端模板下载 + xlsx 解析 + 逐行落库。
 * 落库经调用方注入的 createRow（页面既有 create 变体），
 * { data } 包裹/校验/租户隔离/审计全走既有链路。
 */
const props = defineProps<{
  /** 可导入字段（label 即模板表头） */
  fields: ImportField[];
  open: boolean;
  /** 单行落库动作（页面既有 create mutation 的 mutateAsync） */
  createRow: (values: Record<string, any>) => Promise<any>;
}>();

const emit = defineEmits<{
  close: [];
  success: [imported: number];
}>();

const file = ref<File | null>(null);
const submitting = ref(false);
const fileList = ref<any[]>([]);

function beforeUpload(f: any) {
  file.value = f as File;
  fileList.value = [f];
  return false; // 手动控制：选择即停，提交时统一解析
}

function handleRemove() {
  file.value = null;
  fileList.value = [];
}

function reset() {
  file.value = null;
  fileList.value = [];
}

function handleCancel() {
  reset();
  emit('close');
}

async function handleDownloadTemplate() {
  const { data } = await generateImportTemplate(props.fields);
  const blob = new Blob([data], {
    type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
  });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = 'import-template.xlsx';
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
}

async function handleOk() {
  if (!file.value) return;
  submitting.value = true;
  try {
    const imported = await runImportFile(file.value, props.fields, props.createRow);
    message.success($t('ui.import.importSuccess', { count: imported }));
    reset();
    emit('success', imported);
    emit('close');
  } catch (err: any) {
    message.error(`${$t('ui.import.importFailed')}：${err?.message ?? ''}`);
  } finally {
    submitting.value = false;
  }
}
</script>

<template>
  <a-modal
    :open="open"
    :title="$t('ui.import.title')"
    :ok-button-props="{ disabled: !file || submitting }"
    :confirm-loading="submitting"
    :ok-text="$t('ui.button.ok')"
    :cancel-text="$t('ui.button.cancel')"
    @ok="handleOk"
    @cancel="handleCancel"
  >
    <a-upload-dragger
      accept=".xlsx"
      :max-count="1"
      :before-upload="beforeUpload"
      :file-list="fileList"
      @remove="handleRemove"
    >
      <p class="ant-upload-text">{{ $t('ui.import.uploadText') }}</p>
      <p class="ant-upload-hint">{{ $t('ui.import.uploadHint') }}</p>
    </a-upload-dragger>
    <a-button type="link" class="mt-2 px-0" @click="handleDownloadTemplate">
      {{ $t('ui.import.downloadTemplate') }}
    </a-button>
  </a-modal>
</template>
