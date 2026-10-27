<script lang="ts" setup>
import { ref } from 'vue';

import { $t } from '@vben/locales';

import { Modal, Select, notification } from 'ant-design-vue';

import {
  fetchListPrintTemplates,
} from '#/api/composables/print_template';
import { PaginationQuery } from '#/transport/rest';
import { apiClient } from '#/api/client';
import { renderHtmlPages, browserCodeRenderer } from '@worm-vue3-print/core/browser';
import { getPaperDimensions } from '@worm-vue3-print/core/designer';

// 行打印定式（docs/print_template.md §5）：宿主列表持本组件引用，
// 行动作里 print(resourceKey, row)——模板按 resourceKey 精确匹配优先，回退通用（resourceKey 空）。
const visible = ref(false);
const loading = ref(false);
const rendering = ref(false);
const templates = ref<any[]>([]);
const selectedId = ref<number>();
const frameRef = ref<HTMLIFrameElement | null>(null);
const printData = ref<Record<string, unknown>>({});
const lastHtml = ref('');
const lastTemplate = ref<any>(null);
const serverPdfLoading = ref(false);

async function loadTemplates(resourceKey: string) {
  const resp = await fetchListPrintTemplates(
    new PaginationQuery({ paging: { page: 1, pageSize: 100 } }),
  );
  const enabled = (resp.items || []).filter((item) => item.isEnabled !== false);
  const exact = enabled.filter((item) => item.resourceKey === resourceKey);
  const generic = enabled.filter((item) => !item.resourceKey);
  return [...exact, ...generic];
}

async function renderPreview(tpl: any) {
  rendering.value = true;
  try {
    const result = await renderHtmlPages(
      JSON.parse(tpl.payload || '{}'),
      printData.value,
      window.location.origin,
      browserCodeRenderer,
    );
    lastHtml.value = result.html;
    lastTemplate.value = tpl;
    window.setTimeout(() => {
      const doc = frameRef.value?.contentWindow?.document;
      if (doc) {
        doc.open();
        doc.write(result.html);
        doc.close();
      }
      rendering.value = false;
    }, 80);
  } catch (e) {
    rendering.value = false;
    notification.error({ message: (e as Error).message || $t('page.printTemplate.renderFailed') });
  }
}

async function print(resourceKey: string, rowData: Record<string, unknown>) {
  printData.value = rowData || {};
  visible.value = true;
  loading.value = true;
  templates.value = [];
  selectedId.value = undefined;
  try {
    const list = await loadTemplates(resourceKey);
    templates.value = list;
    const first = list[0];
    if (first) {
      selectedId.value = first.id;
      await renderPreview(first);
    }
  } catch (e) {
    notification.error({ message: (e as Error).message || $t('page.printTemplate.renderFailed') });
  } finally {
    loading.value = false;
  }
}

function handleSelect(value: unknown) {
  const id = Number(value);
  selectedId.value = id;
  const tpl = templates.value.find((item) => item.id === id);
  if (tpl) renderPreview(tpl);
}

function handlePrint() {
  frameRef.value?.contentWindow?.print();
}

/** 服务端 PDF（M3 输出链）：当前预览 HTML → 后端 Chromium PrintToPDF → 下载 */
async function handleServerPdf() {
  if (!lastHtml.value || !lastTemplate.value) return;
  serverPdfLoading.value = true;
  try {
    let payloadTpl: any = {};
    try {
      payloadTpl = JSON.parse(lastTemplate.value.payload || '{}');
    } catch {
      // 保底 A4
    }
    const dims = getPaperDimensions(payloadTpl);
    const resp = await apiClient.printTemplateService.RenderPdf({
      html: lastHtml.value,
      filename: (lastTemplate.value.code || 'print') + '-' + Date.now(),
      paperWidthMm: dims.width,
      paperHeightMm: dims.height,
      landscape: false,
      background: true,
    });
    const bin = atob(resp.pdf as unknown as string);
    const bytes = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; i += 1) bytes[i] = bin.charCodeAt(i);
    const blob = new Blob([bytes], { type: 'application/pdf' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = resp.filename || 'print.pdf';
    a.click();
    URL.revokeObjectURL(url);
  } catch (e) {
    notification.error({ message: (e as Error).message || $t('page.printTemplate.serverPdfFailed') });
  } finally {
    serverPdfLoading.value = false;
  }
}

defineExpose({ print });
</script>

<template>
  <Modal v-model:open="visible" :title="$t('page.printTemplate.printTitle')" width="78%">
    <div v-if="loading" class="preview-loading">{{ $t('page.printTemplate.rendering') }}</div>
    <div v-else-if="templates.length === 0" class="preview-empty">
      {{ $t('page.printTemplate.noTemplate') }}
      <div class="preview-empty-hint">{{ $t('page.printTemplate.noTemplateHint') }}</div>
    </div>
    <div v-else class="preview-body">
      <div v-if="rendering" class="preview-loading">…</div>
      <iframe ref="frameRef" title="print-preview" class="preview-frame" />
    </div>
    <template #footer>
      <Select
        v-if="templates.length > 0"
        :value="selectedId"
        style="min-width: 220px"
        :options="templates.map((item) => ({ value: item.id, label: item.name || item.code }))"
        @change="handleSelect"
      />
      <a-button type="primary" :disabled="templates.length === 0" @click="handlePrint">
        {{ $t('page.printTemplate.print') }}
      </a-button>
      <a-button
        type="primary"
        ghost
        :loading="serverPdfLoading"
        :disabled="templates.length === 0"
        @click="handleServerPdf"
      >
        {{ $t('page.printTemplate.serverPdf') }}
      </a-button>
    </template>
  </Modal>
</template>

<style scoped>
.preview-body {
  position: relative;
  height: 64vh;
  background: #f5f5f5;
  overflow: auto;
}

.preview-loading {
  padding: 24px;
  text-align: center;
}

.preview-frame {
  width: 100%;
  height: 64vh;
  border: none;
}

.preview-empty {
  padding: 32px;
  text-align: center;
}

.preview-empty-hint {
  margin-top: 8px;
  color: #999;
  font-size: 12px;
}
</style>
