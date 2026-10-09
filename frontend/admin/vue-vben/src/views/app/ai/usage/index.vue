<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { computed, onMounted, ref } from 'vue';

import { Page } from '@vben/common-ui';
import { $t } from '@vben/locales';
import { dateUtil } from '@vben/utils';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import { apiClient, PaginationQuery, serverExportFile } from '#/api';
import { message } from 'ant-design-vue';

// 当前搜索条件（服务端导出透传用，与 Grid proxy 查询同源——"导出的就是当前搜索看到的"）
const latestFormValuesRef = ref<Record<string, unknown>>({});

const monthTokens = ref(0);
const monthCalls = ref(0);
const quotaConfigured = ref(false);
const quotaLimit = ref(0);

// 汇总卡数值是"文字"不是"填充"，取值必须随主题走 --metric-* 档
//（定义在 packages/styles/src/antd/index.css，与 react / vue-element 端同名变量同值）：
// 原来直接写 hsl(var(--primary)) 与 tailwind 的 cyan-400/violet-400，实测浅色下 1.81:1、
// 2.72:1，连 24px 大字号的 3.0:1 下限都够不到（2026-09-28 全页面扫测）。
const metricItems = computed(() => [
  {
    color: 'var(--metric-blue)',
    label: $t('page.aiUsage.monthTokens'),
    value: monthTokens.value.toLocaleString(),
  },
  {
    color: 'var(--metric-cyan)',
    label: $t('page.aiUsage.monthCalls'),
    value: monthCalls.value.toLocaleString(),
  },
  {
    color: 'var(--metric-violet)',
    label: $t('page.aiUsage.quota'),
    value: quotaConfigured.value ? quotaLimit.value.toLocaleString() : '∞',
  },
]);

async function loadSummary() {
  try {
    const resp = await apiClient.aiUsageLogService.GetUsageSummary({});
    monthTokens.value = resp.monthTokens ?? 0;
    monthCalls.value = resp.monthCalls ?? 0;
    quotaConfigured.value = resp.quotaConfigured ?? false;
    quotaLimit.value = resp.quotaLimit ?? 0;
  } catch (error) {
    console.error('load usage summary failed:', error);
  }
}

/** 流水列表 fetcher：List + 分页参数（Grid proxy 复用） */
async function fetchListAiUsageLogs(query: PaginationQuery) {
  return apiClient.aiUsageLogService.List(query.toRawParams());
}

// 服务端全量导出：当前搜索条件经 PaginationQuery 同源序列化透传，
// 导出的即当前搜索看到的；行范围由后端 viewer 语义决定（租户=本租户）。
async function handleServerExport() {
  try {
    await serverExportFile(
      'admin/v1/ai/usage-logs:export',
      new PaginationQuery({ formValues: latestFormValuesRef.value }),
    );
    message.success($t('page.auditExport.serverSuccess'));
  } catch (error: any) {
    // 原始错误必须留在控制台：用户可见的那句翻译不包含服务端的原因
    console.error('[export] server-side export failed', error);
    message.error(error?.message || $t('page.auditExport.serverFailed'));
  }
}

function handleExportMenu(info: { key: string | number }) {
  if (String(info.key) === 'server-xlsx') {
    handleServerExport();
    return;
  }
}

const formOptions = {
  collapsed: false,
  showCollapseButton: false,
  submitOnEnter: true,
  schema: [
    {
      component: 'Input',
      fieldName: 'modelName',
      label: $t('page.aiUsage.model'),
      componentProps: { allowClear: true },
    },
  ],
};

const gridOptions: VxeGridProps<Record<string, any>> = {
  toolbarConfig: {
    custom: true,
    refresh: true,
    zoom: true,
  },
  height: 'auto',
  pagerConfig: {},
  rowConfig: { isHover: true, keyField: 'id' },
  stripe: true,
  proxyConfig: {
    ajax: {
      query: async ({ page }, formValues) => {
        latestFormValuesRef.value =
          formValues && Object.keys(formValues).length > 0 ? { ...formValues } : {};
        return await fetchListAiUsageLogs(
          new PaginationQuery({
            paging: { page: page.currentPage, pageSize: page.pageSize },
            // formValues 字符串值由 PaginationQuery 统一转 __contains（搜索铁律）
            formValues:
              formValues && Object.keys(formValues).length > 0
                ? { ...formValues }
                : undefined,
          }),
        );
      },
    },
  },
  columns: [
    { title: $t('ui.table.seq'), type: 'seq', width: 50 },
    { title: $t('page.aiUsage.model'), field: 'modelName', minWidth: 180 },
    {
      title: $t('page.aiUsage.promptTokens'),
      field: 'promptTokens',
      width: 130,
      formatter: ({ cellValue }) => Number(cellValue ?? 0).toLocaleString(),
    },
    {
      title: $t('page.aiUsage.completionTokens'),
      field: 'completionTokens',
      width: 140,
      formatter: ({ cellValue }) => Number(cellValue ?? 0).toLocaleString(),
    },
    {
      title: $t('page.aiUsage.totalTokens'),
      field: 'totalTokens',
      width: 110,
      formatter: ({ cellValue }) => Number(cellValue ?? 0).toLocaleString(),
    },
    {
      title: $t('page.aiUsage.duration'),
      field: 'durationMs',
      width: 110,
      formatter: ({ cellValue }) => `${Number(cellValue ?? 0)} ms`,
    },
    {
      title: $t('page.aiUsage.time'),
      field: 'createdAt',
      width: 180,
      formatter: ({ cellValue }) =>
        cellValue ? dateUtil(cellValue).format('YYYY-MM-DD HH:mm:ss') : '',
    },
  ],
};

const [Grid] = useVbenVxeGrid({ gridOptions, formOptions });

onMounted(() => {
  loadSummary();
});
</script>

<template>
  <Page auto-content-height>
    <div class="flex h-full flex-col gap-4">
      <!-- 汇总卡 -->
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <div
          v-for="m in metricItems"
          :key="m.label"
          class="rounded-xl border border-solid border-border bg-card p-4"
        >
          <div class="mb-2 text-sm text-muted-foreground">{{ m.label }}</div>
          <div class="text-2xl font-semibold tabular-nums" :style="{ color: m.color }">
            {{ m.value }}
          </div>
        </div>
      </div>

      <!-- 流水列表：vxe Grid 统一搜索/服务端分页约定（与其余管理页一致） -->
      <div class="min-h-0 flex-1">
        <Grid>
          <template #toolbar-tools>
            <a-dropdown class="mr-2">
              <a-button>{{ $t('ui.button.exportAll') }}</a-button>
              <template #overlay>
                <a-menu @click="handleExportMenu">
                  <a-menu-item key="server-xlsx">{{
                    $t('page.auditExport.serverFull')
                  }}</a-menu-item>
                </a-menu>
              </template>
            </a-dropdown>
          </template>
        </Grid>
      </div>
    </div>
  </Page>
</template>
