<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { Page, useVbenDrawer } from '@vben/common-ui';
import {
  LucideCirclePlay,
  LucideFilePenLine,
  LucideHistory,
  LucideNetwork,
  LucideRotateCcw,
  LucideTrash2,
} from '@vben/icons';
import { $t } from '@vben/locales';

import { h } from 'vue';

import { Modal, notification } from 'ant-design-vue';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import {
  deleteAiAgentApp,
  fetchListAiAgentApps,
} from '#/api';
import TableActions from '#/components/TableActions.vue';
import type { TableAction } from '#/components/TableActions.vue';
import TableAddButton from '#/components/TableAddButton.vue';
import TableExportButton from '#/components/TableExportButton.vue';
import type {
  aiservicev1_AiAgentApp as AiAgentApp,
} from '#/api/generated/admin/service/v1';
import AgentCanvas from './agent-canvas.vue';
import AgentEditDrawer from './agent-edit-drawer.vue';
import AgentRevisionsDrawer from './agent-revisions-drawer.vue';
import AgentRunDrawer from './agent-run-drawer.vue';
import AgentRunsDrawer from './agent-runs-drawer.vue';

// 抽屉们：画布（全屏编辑器）/ 编辑（表单）/ 试运行（输入表单+运行）/ 运行台账（只读列表）/ 版本历史
const [CanvasDrawer, canvasDrawerApi] = useVbenDrawer({ connectedComponent: AgentCanvas });
const [EditDrawer, editDrawerApi] = useVbenDrawer({ connectedComponent: AgentEditDrawer });
const [RunDrawer, runDrawerApi] = useVbenDrawer({ connectedComponent: AgentRunDrawer });
const [RunsDrawer, runsDrawerApi] = useVbenDrawer({ connectedComponent: AgentRunsDrawer });
const [RevisionsDrawer, revisionsDrawerApi] = useVbenDrawer({ connectedComponent: AgentRevisionsDrawer });

const gridOptions: VxeGridProps<AiAgentApp> = {
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
        return await fetchListAiAgentApps({
          page: page.currentPage,
          pageSize: page.pageSize,
        });
      },
    },
  },
  columns: [
    { title: $t('ui.table.seq'), type: 'seq', width: 50 },
    { title: $t('page.aiAgent.name'), field: 'name', minWidth: 160 },
    {
      title: $t('page.aiAgent.description'),
      field: 'description',
      minWidth: 200,
      showOverflow: 'tooltip',
    },
    {
      title: $t('page.aiAgent.enabled'),
      field: 'enabled',
      width: 90,
      slots: { default: 'enabled' },
    },
    {
      title: $t('page.aiAgent.updatedAt'),
      field: 'updatedAt',
      width: 170,
    },
    {
      title: $t('ui.table.action'),
      field: 'action',
      fixed: 'right',
      slots: { default: 'action' },
      width: 250,
    },
  ],
};

const exportFetcher = (page: number, pageSize: number) =>
  fetchListAiAgentApps({ page, pageSize });

const [Grid, gridApi] = useVbenVxeGrid({ gridOptions });

function openCreate() {
  editDrawerApi.setState({ title: $t('page.aiAgent.create') });
  editDrawerApi.setData({ create: true });
  editDrawerApi.open();
}

function openCanvas(row: AiAgentApp) {
  canvasDrawerApi.setData({ row });
  canvasDrawerApi.open();
}

function openEdit(row: AiAgentApp) {
  editDrawerApi.setState({ title: $t('page.aiAgent.edit') });
  editDrawerApi.setData({ create: false, row });
  editDrawerApi.open();
}

function openRun(row: AiAgentApp) {
  runDrawerApi.setData({ row });
  runDrawerApi.open();
}

function openRuns(row: AiAgentApp) {
  runsDrawerApi.setData({ row });
  runsDrawerApi.open();
}

function openRevisions(row: AiAgentApp) {
  revisionsDrawerApi.setData({ row });
  revisionsDrawerApi.open();
}

/* 「更多」下拉里的删除走 Modal 确认（下拉里放不下 Popconfirm） */
function confirmDelete(row: AiAgentApp) {
  Modal.confirm({
    title: $t('page.aiAgent.deleteConfirm'),
    okText: $t('ui.button.ok'),
    cancelText: $t('ui.button.cancel'),
    okType: 'danger',
    onOk: () => handleDelete(row),
  });
}

/* 操作列描述符：画布/编辑/测试运行行内，运行台账/删除溢出进「更多」。
   confirmDelete 内部已弹 Modal.confirm，不再挂 confirm 免双重确认 */
function buildRowActions(): TableAction[] {
  return [
    {
      key: 'canvas',
      label: $t('page.aiAgent.openCanvas'),
      icon: h(LucideNetwork),
      onClick: (row) => openCanvas(row),
    },
    {
      key: 'edit',
      label: $t('common.edit'),
      icon: h(LucideFilePenLine),
      onClick: (row) => openEdit(row),
    },
    {
      key: 'runs',
      label: $t('page.aiAgent.runsTitle'),
      icon: h(LucideHistory),
      onClick: (row) => openRuns(row),
    },
    {
      key: 'revisions',
      label: $t('page.aiAgent.openRevisions'),
      icon: h(LucideRotateCcw),
      onClick: (row) => openRevisions(row),
    },
    {
      key: 'test-run',
      label: $t('page.aiAgent.testRun'),
      icon: h(LucideCirclePlay),
      onClick: (row) => openRun(row),
    },
    {
      key: 'delete',
      label: $t('common.delete'),
      icon: h(LucideTrash2),
      danger: true,
      onClick: (row) => confirmDelete(row),
    },
  ];
}

async function handleDelete(row: AiAgentApp) {
  try {
    await deleteAiAgentApp(row.id!);
    notification.success({ message: $t('ui.notification.delete_success') });
    await gridApi.reload();
  } catch (error: any) {
    console.error('[ai-agent] delete failed', error);
    notification.error({ message: error?.message || $t('ui.notification.delete_failed') });
  }
}
</script>

<template>
  <Page auto-content-height>
    <Grid :table-title="$t('page.aiAgent.moduleName')">
      <template #toolbar-tools>
        <TableExportButton :fetcher="exportFetcher" :columns="gridOptions.columns" filename="ai-agent-apps" />
        <TableAddButton @click="openCreate" />
      </template>
      <template #enabled="{ row }">
        <a-tag v-if="row.enabled" color="success">
          {{ $t('page.aiAgent.enabledOn') }}
        </a-tag>
        <a-tag v-else>{{ $t('page.aiAgent.enabledOff') }}</a-tag>
      </template>
      <template #action="{ row }">
        <TableActions :row="row" :actions="buildRowActions()" />
      </template>
    </Grid>
    <CanvasDrawer />
    <EditDrawer />
    <RunDrawer />
    <RunsDrawer />
    <RevisionsDrawer />
  </Page>
</template>
