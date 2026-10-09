<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { h } from 'vue';

import { Page } from '@vben/common-ui';
import { $t } from '@vben/locales';
import { LucideFilePenLine, LucidePlus, LucideTrash2 } from '@vben/icons';

import { message } from 'ant-design-vue';

import { useVbenDrawer } from '@vben/common-ui';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import { PaginationQuery, deleteAiProvider, fetchListAiProviders } from '#/api';
import type { aiservicev1_AiProvider as AiProvider } from '#/api/generated/admin/service/v1';

import AiProviderDrawer from './ai-provider-drawer.vue';

const formOptions = {
  collapsed: false,
  showCollapseButton: false,
  submitOnEnter: true,
  schema: [
    {
      component: 'Input',
      fieldName: 'name',
      label: $t('page.aiProvider.name'),
      componentProps: { allowClear: true },
    },
  ],
};

const gridOptions: VxeGridProps<AiProvider> = {
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
      query: async ({ page }) => {
        return await fetchListAiProviders(
          new PaginationQuery({
            paging: { page: page.currentPage, pageSize: page.pageSize },
          }),
        );
      },
    },
  },
  columns: [
    { title: $t('ui.table.seq'), type: 'seq', width: 50 },
    { title: $t('page.aiProvider.name'), field: 'name', minWidth: 140 },
    {
      title: $t('page.aiProvider.modelType'),
      field: 'modelType',
      width: 110,
      slots: { default: 'modelType' },
    },
    { title: $t('page.aiProvider.modelName'), field: 'modelName', minWidth: 150 },
    {
      title: $t('page.aiProvider.endpoint'),
      field: 'endpoint',
      minWidth: 200,
      slots: { default: 'endpoint' },
      showOverflow: 'tooltip',
    },
    {
      title: $t('page.aiProvider.apiKey'),
      field: 'apiKeyHint',
      width: 120,
      slots: { default: 'apiKeyHint' },
    },
    { title: $t('page.aiProvider.timeoutSeconds'), field: 'timeoutSeconds', width: 110 },
    {
      title: $t('page.aiProvider.isEnabled'),
      field: 'isEnabled',
      width: 80,
      slots: { default: 'isEnabled' },
    },
    { title: $t('page.aiProvider.remark'), field: 'remark', minWidth: 120 },
    {
      title: $t('ui.table.action'),
      field: 'action',
      fixed: 'right',
      slots: { default: 'action' },
      width: 130,
    },
  ],
};

const [Grid, gridApi] = useVbenVxeGrid({ gridOptions, formOptions });

// connectedComponent 模式：抽屉关闭即刷新列表（vben 惯例，不用 invalidateQueries）
const [Drawer, drawerApi] = useVbenDrawer({
  connectedComponent: AiProviderDrawer,
  onOpenChange(isOpen) {
    if (!isOpen) gridApi.reload();
  },
});

function handleAdd() {
  drawerApi.setData({ create: true });
  drawerApi.open();
}

function handleEdit(row: AiProvider) {
  drawerApi.setData({ create: false, row });
  drawerApi.open();
}

async function handleDelete(row: AiProvider) {
  try {
    await deleteAiProvider(row.id!);
    await gridApi.reload();
  } catch (error) {
    console.error('delete ai provider failed:', error);
    message.error(error instanceof Error ? error.message : 'delete failed');
  }
}

function modelTypeLabel(type?: string): string {
  if (type === 'CLOUD') return $t('page.aiProvider.modelTypeCloud');
  if (type === 'LOCAL') return $t('page.aiProvider.modelTypeLocal');
  return type || '-';
}

function modelTypeColor(type?: string): string {
  if (type === 'CLOUD') return 'processing';
  if (type === 'LOCAL') return 'success';
  return 'default';
}

function endpointText(row: AiProvider): string {
  if (row.modelType === 'CLOUD') return row.baseUrl || '-';
  if (row.modelType === 'LOCAL') {
    return `${row.localHost || 'localhost'}:${row.localPort ?? 11434}`;
  }
  return '-';
}
</script>

<template>
  <Page auto-content-height>
    <Grid :table-title="$t('page.aiProvider.moduleName')">
      <template #toolbar-tools>
        <a-button type="primary" class="mr-2" @click="handleAdd">
          <template #icon>
            <LucidePlus />
          </template>
          {{ $t('page.aiProvider.name') }}
        </a-button>
      </template>
      <template #modelType="{ row }">
        <a-tag v-if="row.modelType" :color="modelTypeColor(row.modelType)">
          {{ modelTypeLabel(row.modelType) }}
        </a-tag>
        <template v-else>-</template>
      </template>
      <template #endpoint="{ row }">{{ endpointText(row) }}</template>
      <template #apiKeyHint="{ row }">
        <span v-if="row.apiKeyHint">{{ row.apiKeyHint }}</span>
        <a-tag v-else>{{ $t('page.aiProvider.notSet') }}</a-tag>
      </template>
      <template #isEnabled="{ row }">
        <a-tag v-if="row.isEnabled" color="success">{{ $t('page.aiProvider.enabledOn') }}</a-tag>
        <a-tag v-else>{{ $t('page.aiProvider.enabledOff') }}</a-tag>
      </template>
      <template #action="{ row }">
        <a-button
          type="link"
          size="small"
          :icon="h(LucideFilePenLine)"
          @click.stop="handleEdit(row)"
        />
        <a-popconfirm
          :cancel-text="$t('ui.button.cancel')"
          :ok-text="$t('ui.button.ok')"
          :title="$t('ui.text.do_you_want_delete', { moduleName: $t('page.aiProvider.moduleName') })"
          @confirm="handleDelete(row)"
        >
          <a-button danger type="link" size="small" :icon="h(LucideTrash2)" />
        </a-popconfirm>
      </template>
    </Grid>
    <Drawer />
  </Page>
</template>
