<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { h, watch } from 'vue';

import { useVbenDrawer } from '@vben/common-ui';
import { LucideFilePenLine, LucideTrash2 } from '@vben/icons';

import { notification } from 'ant-design-vue';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import TableExportButton from '#/components/TableExportButton.vue';
import { type identityservicev1_PlanQuota as PlanQuota } from '#/api';
import { planQuotaTypeToName, useDeletePlanQuota } from '#/api';
import { $t } from '#/locales';
import { usePlanViewStore } from '#/views/app/tenant/plan/plan-view.state';

import PlanQuotaDrawer from './plan-quota-drawer.vue';

const { mutateAsync: deletePlanQuota } = useDeletePlanQuota();
const planViewStore = usePlanViewStore();

const gridOptions: VxeGridProps<PlanQuota> = {
  stripe: true,
  height: 'auto',

  toolbarConfig: {
    custom: false,
    export: true,
    import: true,
    refresh: true,
    zoom: false,
  },
  exportConfig: {},
  pagerConfig: {},
  rowConfig: {
    isHover: true,
    isCurrent: false,
  },

  proxyConfig: {
    ajax: {
      query: async ({ page }, formValues) => {
        return await planViewStore.fetchPlanQuotaList(
          planViewStore.currentPlanId,
          page.currentPage,
          page.pageSize,
          formValues,
        );
      },
    },
  },

  columns: [
    {
      title: $t('page.plan.quotaType'),
      field: 'quotaType',
      slots: { default: 'quotaType' },
      minWidth: 95,
    },
    {
      title: $t('page.plan.quotaValue'),
      field: 'quotaValue',
      minWidth: 95,
    },
    {
      title: $t('ui.table.createdAt'),
      field: 'createdAt',
      minWidth: 95,
      formatter: 'formatDateTime',
    },
    {
      title: $t('ui.table.action'),
      field: 'action',
      fixed: 'right',
      slots: { default: 'action' },
      minWidth: 90,
    },
  ],
};

const exportFetcher = (page: number, pageSize: number) =>
  planViewStore.fetchPlanQuotaList(planViewStore.currentPlanId, page, pageSize, {});

const [Grid, gridApi] = useVbenVxeGrid({ gridOptions });

const [Drawer, drawerApi] = useVbenDrawer({
  connectedComponent: PlanQuotaDrawer,

  onOpenChange(isOpen: boolean) {
    if (!isOpen) {
      gridApi.reload();
    }
  },
});

/* 打开模态窗口 */
function openDrawer(create: boolean, row?: any) {
  drawerApi.setData({
    create,
    row,
  });

  drawerApi.open();
}

/* 创建 */
function handleCreate() {
  openDrawer(true);
}

/* 编辑 */
function handleEdit(row: any) {
  openDrawer(false, row);
}

/* 删除 */
async function handleDelete(row: any) {
  try {
    await deletePlanQuota({ id: row.id });

    notification.success({
      message: $t('ui.notification.delete_success'),
    });

    await gridApi.reload();
  } catch {
    notification.error({
      message: $t('ui.notification.delete_failed'),
    });
  }
}

watch(
  () => planViewStore.currentPlanId,
  () => {
    gridApi.reload();
  },
);
</script>

<template>
  <Grid :table-title="$t('page.plan.planQuotaList')">
    <template #toolbar-tools>
      <a-button type="primary" @click="handleCreate">
        {{ $t('page.plan.button.create') }}
      </a-button>
          <TableExportButton :fetcher="exportFetcher" :columns="gridOptions.columns" filename="export" />
    </template>
    <template #quotaType="{ row }">
      {{ planQuotaTypeToName(row.quotaType) }}
    </template>
    <template #action="{ row }">
      <a-button
        type="link"
        :icon="h(LucideFilePenLine)"
        @click.stop="handleEdit(row)"
      />
      <a-popconfirm
        :cancel-text="$t('ui.button.cancel')"
        :ok-text="$t('ui.button.ok')"
        :title="
          $t('ui.text.do_you_want_delete', {
            moduleName: $t('page.plan.moduleName'),
          })
        "
        @confirm="handleDelete(row)"
      >
        <a-button danger type="link" :icon="h(LucideTrash2)" />
      </a-popconfirm>
    </template>
  </Grid>
  <Drawer />
</template>
