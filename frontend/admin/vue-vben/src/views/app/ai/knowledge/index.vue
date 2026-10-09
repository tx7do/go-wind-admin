<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { h, ref } from 'vue';

import { Page, useVbenDrawer } from '@vben/common-ui';
import { $t } from '@vben/locales';
import { LucideFilePenLine, LucidePlus, LucideTrash2 } from '@vben/icons';
import { Icon as Iconify } from '@iconify/vue';

import { message } from 'ant-design-vue';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import { PaginationQuery, deleteAiKnowledgeBase, fetchListAiKnowledgeBases } from '#/api';
import type { aiservicev1_AiKnowledgeBase as AiKnowledgeBase } from '#/api/generated/admin/service/v1';

import AiKnowledgeDrawer from './ai-knowledge-drawer.vue';
import DocsModal from './docs-modal.vue';

const [BaseDrawer, baseDrawerApi] = useVbenDrawer({
  connectedComponent: AiKnowledgeDrawer,
  onOpenChange(isOpen) {
    if (!isOpen) gridApi.reload();
  },
});

const docsVisible = ref(false);
const docsTarget = ref<AiKnowledgeBase>();

const formOptions = {
  collapsed: false,
  showCollapseButton: false,
  submitOnEnter: true,
  schema: [
    {
      component: 'Input',
      fieldName: 'name',
      label: $t('page.aiKnowledge.name'),
      componentProps: { allowClear: true },
    },
  ],
};

const gridOptions: VxeGridProps<AiKnowledgeBase> = {
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
        return await fetchListAiKnowledgeBases(
          new PaginationQuery({
            paging: { page: page.currentPage, pageSize: page.pageSize },
          }),
        );
      },
    },
  },
  columns: [
    { title: $t('ui.table.seq'), type: 'seq', width: 50 },
    { title: $t('page.aiKnowledge.name'), field: 'name', minWidth: 160 },
    { title: $t('page.aiKnowledge.description'), field: 'description', minWidth: 180 },
    { title: $t('page.aiKnowledge.embeddingModel'), field: 'embeddingModel', minWidth: 180 },
    { title: $t('page.aiKnowledge.docCount'), field: 'docCount', width: 90 },
    {
      title: $t('ui.table.action'),
      field: 'action',
      fixed: 'right',
      slots: { default: 'action' },
      width: 170,
    },
  ],
};

const [Grid, gridApi] = useVbenVxeGrid({ gridOptions, formOptions });

function handleAdd() {
  baseDrawerApi.setData({ create: true });
  baseDrawerApi.open();
}

function handleEdit(row: AiKnowledgeBase) {
  baseDrawerApi.setData({ create: false, row });
  baseDrawerApi.open();
}

async function handleDelete(row: AiKnowledgeBase) {
  try {
    await deleteAiKnowledgeBase(row.id!);
    await gridApi.reload();
  } catch (error) {
    console.error('delete ai knowledge base failed:', error);
    message.error(error instanceof Error ? error.message : 'delete failed');
  }
}

function handleDocs(row: AiKnowledgeBase) {
  docsTarget.value = row;
  docsVisible.value = true;
}
</script>

<template>
  <Page auto-content-height>
    <Grid :table-title="$t('page.aiKnowledge.moduleName')">
      <template #toolbar-tools>
        <a-button class="mr-2" type="primary" @click="handleAdd">
          <template #icon>
            <LucidePlus />
          </template>
          {{ $t('page.aiKnowledge.name') }}
        </a-button>
      </template>
      <template #action="{ row }">
        <a-button type="link" size="small" @click.stop="handleDocs(row)">
          <Iconify icon="lucide:file-text" />
        </a-button>
        <a-button type="link" size="small" :icon="h(LucideFilePenLine)" @click.stop="handleEdit(row)" />
        <a-popconfirm
          :cancel-text="$t('ui.button.cancel')"
          :ok-text="$t('ui.button.ok')"
          :title="$t('ui.text.do_you_want_delete', { moduleName: $t('page.aiKnowledge.moduleName') })"
          @confirm="handleDelete(row)"
        >
          <a-button danger type="link" size="small" :icon="h(LucideTrash2)" />
        </a-popconfirm>
      </template>
    </Grid>
    <BaseDrawer />
    <DocsModal v-model:visible="docsVisible" :base="docsTarget" />
  </Page>
</template>
