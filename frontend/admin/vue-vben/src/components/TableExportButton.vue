<script lang="ts" setup>
import { ref } from 'vue';

import { $t } from '@vben/locales';

import { message } from 'ant-design-vue';

import { downloadTableFile, type TableExportFormat } from '#/utils/csv';

/**
 * 列表导出按钮（CSV / Excel 下拉）：内置分页聚合（默认上限 1 万行），
 * 按列定义（title/field）生成文件并触发下载。
 * fetcher 契约：(page, pageSize) => Promise<{ items, total }>；
 * 页面侧用一行箭头把 PaginationQuery 拼好（如需 formValues 在闭包里带上）。
 */
const props = withDefaults(
  defineProps<{
    columns?: any[];
    fetcher: (
      page: number,
      pageSize: number,
    ) => Promise<{ items?: any[]; total?: number }>;
    filename?: string;
    maxRows?: number;
  }>(),
  { filename: 'export', maxRows: 10_000 },
);

const emit = defineEmits<{
  exported: [count: number];
}>();

const exporting = ref(false);

function exportColumns() {
  return (props.columns || [])
    .filter((c) => c.field && c.title)
    .map((c) => ({ title: c.title as string, key: c.field as string }));
}

async function handleMenuClick(info: { key: string | number }) {
  const format = String(info.key) as TableExportFormat;
  if (format !== 'csv' && format !== 'xlsx') return;

  const cols = exportColumns();
  if (cols.length === 0) {
    message.error($t('ui.export.noColumns'));
    return;
  }

  exporting.value = true;
  try {
    const rows: any[] = [];
    const pageSize = 1000;
    for (let p = 1; rows.length < props.maxRows; p++) {
      const resp = await props.fetcher(p, pageSize);
      const items = resp.items ?? [];
      rows.push(...items);
      if (items.length < pageSize) break;
    }
    const sliced = rows.slice(0, props.maxRows);
    await downloadTableFile(
      format,
      `${props.filename}-${Date.now()}`,
      cols,
      sliced,
    );
    message.success($t('ui.export.success', { count: sliced.length }));
    emit('exported', sliced.length);
  } catch (err: any) {
    message.error(`${$t('ui.export.failed')}：${err?.message ?? ''}`);
  } finally {
    exporting.value = false;
  }
}
</script>

<template>
  <a-dropdown>
    <a-button class="mr-2" :loading="exporting">
      {{ $t('ui.export.title') }}
    </a-button>
    <template #overlay>
      <a-menu @click="handleMenuClick">
        <a-menu-item key="csv">{{ $t('ui.export.csv') }}</a-menu-item>
        <a-menu-item key="xlsx">{{ $t('ui.export.xlsx') }}</a-menu-item>
      </a-menu>
    </template>
  </a-dropdown>
</template>
