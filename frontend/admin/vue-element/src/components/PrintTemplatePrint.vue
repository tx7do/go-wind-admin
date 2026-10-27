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
const printData = ref<Record<string, unknown>>({});

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
      printData.value,
      window.location.origin,
      browserCodeRenderer
    );
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

async function print(resourceKey: string, rowData: Record<string, unknown>) {
  printData.value = rowData || {};
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
