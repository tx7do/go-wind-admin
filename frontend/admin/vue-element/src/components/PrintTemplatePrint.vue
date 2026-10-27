<template>
  <ElDialog v-model="visible" :title="$t('pages.printTemplate.printTitle')" width="78%">
    <div v-if="loading" class="preview-loading">{{ $t("pages.printTemplate.rendering") }}</div>
    <ElEmpty v-else-if="templates.length === 0" :description="$t('pages.printTemplate.noTemplate')">
      <span class="empty-hint">{{ $t("pages.printTemplate.noTemplateHint") }}</span>
    </ElEmpty>
    <div v-else class="preview-body">
      <div v-if="rendering" class="preview-loading">…</div>
      <iframe ref="frameRef" title="print-preview" class="preview-frame" />
    </div>
    <template #footer>
      <ElSelect
        v-if="templates.length > 0"
        :model-value="selectedId"
        style="min-width: 220px"
        @change="handleSelect"
      >
        <ElOption
          v-for="item in templates"
          :key="item.id"
          :label="item.name || item.code"
          :value="item.id!"
        />
      </ElSelect>
      <ElButton type="primary" :disabled="templates.length === 0" @click="handlePrint">
        {{ $t("pages.printTemplate.print") }}
      </ElButton>
      <ElButton
        type="primary"
        plain
        :loading="serverPdfLoading"
        :disabled="templates.length === 0"
        @click="handleServerPdf"
      >
        {{ $t("pages.printTemplate.serverPdf") }}
      </ElButton>
    </template>
  </ElDialog>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import {
  ElButton,
  ElDialog,
  ElEmpty,
  ElMessage,
  ElOption,
  ElSelect,
} from "element-plus";
import { renderHtmlPages, browserCodeRenderer } from "@worm-vue3-print/core/browser";
import { getPaperDimensions } from "@worm-vue3-print/core/designer";
import { apiClient } from "@/api/client";

import { fetchListPrintTemplates } from "@/api/composables/print_template";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/core/i18n";

// 行打印定式（docs/print_template.md §5）：宿主列表持本组件引用，
// 行动作里 print(resourceKey, row)——模板按 resourceKey 精确匹配优先，回退通用（resourceKey 空）。
const visible = ref(false);
const loading = ref(false);
const rendering = ref(false);
const templates = ref<any[]>([]);
const selectedId = ref<number>();
const frameRef = ref<HTMLIFrameElement | null>(null);
const printData = ref<any>({});
const lastHtml = ref("");
const lastTemplate = ref<any>(null);

async function loadTemplates(resourceKey: string) {
  const resp = await fetchListPrintTemplates(
    new PaginationQuery({ paging: { page: 1, pageSize: 100 } })
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
      JSON.parse(tpl.payload || "{}"),
      printData.value as Record<string, any>,
      window.location.origin,
      browserCodeRenderer
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
    ElMessage.error((e as Error).message || $t("pages.printTemplate.renderFailed"));
  }
}

async function print(resourceKey: string, rowData: Record<string, unknown> | Record<string, unknown>[]) {
  // 单行与批量（数组）同入口：内核 renderHtmlPages 原生支持数据数组（份间分页合并单文档）
  printData.value = Array.isArray(rowData) ? rowData : rowData || {};
  visible.value = true;
  loading.value = true;
  templates.value = [];
  selectedId.value = undefined;
  try {
    const list = await loadTemplates(resourceKey);
    templates.value = list;
    if (list.length > 0) {
      selectedId.value = list[0].id;
      await renderPreview(list[0]);
    }
  } catch (e) {
    ElMessage.error((e as Error).message || $t("pages.printTemplate.renderFailed"));
  } finally {
    loading.value = false;
  }
}

function handleSelect(id: number) {
  selectedId.value = id;
  const tpl = templates.value.find((item) => item.id === id);
  if (tpl) renderPreview(tpl);
}

function handlePrint() {
  frameRef.value?.contentWindow?.print();
}

/** 服务端 PDF（M3 输出链）：当前预览 HTML → 后端 Chromium PrintToPDF → 下载 */
const serverPdfLoading = ref(false);

async function handleServerPdf() {
  if (!lastHtml.value || !lastTemplate.value) return;
  serverPdfLoading.value = true;
  try {
    let payloadTpl: any = {};
    try {
      payloadTpl = JSON.parse(lastTemplate.value.payload || "{}");
    } catch {
      /* 保底 A4 */
    }
    const dims = getPaperDimensions(payloadTpl);
    const resp = await apiClient.printTemplateService.RenderPdf({
      html: lastHtml.value,
      filename: (lastTemplate.value.code || "print") + "-" + Date.now(),
      paperWidthMm: dims.width,
      paperHeightMm: dims.height,
      landscape: false,
      background: true,
    });
    const bin = atob(resp.pdf as unknown as string);
    const bytes = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; i += 1) bytes[i] = bin.charCodeAt(i);
    const blob = new Blob([bytes], { type: "application/pdf" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = resp.filename || "print.pdf";
    a.click();
    URL.revokeObjectURL(url);
  } catch (e) {
    ElMessage.error((e as Error).message || $t("pages.printTemplate.serverPdfFailed"));
  } finally {
    serverPdfLoading.value = false;
  }
}

defineExpose({ print });
</script>

<style scoped>
.preview-body {
  position: relative;
  height: 64vh;
  background: var(--el-fill-color-light);
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

.empty-hint {
  color: var(--el-text-color-secondary);
}
</style>
