<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { h, ref } from 'vue';

import { Page, useVbenDrawer } from '@vben/common-ui';
import { $t } from '@vben/locales';
import { Icon as Iconify } from '@iconify/vue';
import { LucideFilePenLine, LucideTrash2 } from '@vben/icons';
import TableActions from '#/components/TableActions.vue';
import type { TableAction } from '#/components/TableActions.vue';

import { message } from 'ant-design-vue';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import { PaginationQuery, deleteAiKnowledgeBase, fetchListAiKnowledgeBases } from '#/api';
import type { aiservicev1_AiKnowledgeBase as AiKnowledgeBase } from '#/api/generated/admin/service/v1';

import AiKnowledgeDrawer from './ai-knowledge-drawer.vue';
import DocsModal from './docs-modal.vue';
import TableAddButton from '#/components/TableAddButton.vue';
import TableExportButton from '#/components/TableExportButton.vue';

const [BaseDrawer, baseDrawerApi] = useVbenDrawer({
  connectedComponent: AiKnowledgeDrawer,
  onOpenChange(isOpen) {
    if (!isOpen) gridApi.reload();
  },
});

const docsVisible = ref(false);
const docsTarget = ref<AiKnowledgeBase>();

const formOptions = {
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
      width: 200,
    },
  ],
};

const exportFetcher = (page: number, pageSize: number) =>
  fetchListAiKnowledgeBases(new PaginationQuery({ paging: { page, pageSize } }));

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

/* 操作列描述符：文档/编辑/删除三钮全行内，删除 Popconfirm 确认 */
function buildRowActions(): TableAction[] {
  return [
    {
      key: 'docs',
      label: $t('page.aiKnowledge.docs'),
      icon: h(Iconify, { icon: 'lucide:file-text' }),
      onClick: handleDocs,
    },
    {
      key: 'edit',
      label: $t('common.edit'),
      icon: h(LucideFilePenLine),
      onClick: handleEdit,
    },
    {
      key: 'delete',
      label: $t('common.delete'),
      icon: h(LucideTrash2),
      danger: true,
      confirm: {
        title: $t('ui.text.do_you_want_delete', {
          moduleName: $t('page.aiKnowledge.moduleName'),
        }),
      },
      onClick: handleDelete,
    },
  ];
}
</script>

<template>
  <Page auto-content-height>
    <Grid :table-title="$t('page.aiKnowledge.moduleName')">
      <template #toolbar-tools>
        <TableExportButton :fetcher="exportFetcher" :columns="gridOptions.columns" filename="ai-knowledge-bases" />
        <TableAddButton @click="handleAdd" />
      </template>
      <template #action="{ row }">
        <TableActions :row="row" :actions="buildRowActions()" />
      </template>
    </Grid>
    <BaseDrawer />
    <DocsModal v-model:visible="docsVisible" :base="docsTarget" />
  </Page>
</template>
