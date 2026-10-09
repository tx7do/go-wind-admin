<script lang="ts" setup>
import type { VxeGridListeners, VxeGridProps } from '#/adapter/vxe-table';

import { h } from 'vue';

import { useVbenDrawer, type VbenFormProps } from '@vben/common-ui';
import { LucideFilePenLine, LucideTrash2 } from '@vben/icons';

import { notification } from 'ant-design-vue';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import TableExportButton from '#/components/TableExportButton.vue';
import {
  type identityservicev1_Plan as Plan,
  planExpiryPolicyToColor,
  planExpiryPolicyToName,
  planVersionToColor,
  planVersionToName,
  useDeletePlan,
} from '#/api';
import { $t } from '#/locales';
import { usePlanViewStore } from '#/views/app/tenant/plan/plan-view.state';

import PlanDrawer from './plan-drawer.vue';

const { mutateAsync: deletePlan } = useDeletePlan();
const planViewStore = usePlanViewStore();

const formOptions: VbenFormProps = {
  // 默认展开
  collapsed: false,
  // 控制表单是否显示折叠按钮
  showCollapseButton: false,
  // 按下回车时是否提交表单
  submitOnEnter: true,
  schema: [
    {
      component: 'Input',
      fieldName: 'name',
      label: $t('page.plan.name'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
  ],
};

const gridOptions: VxeGridProps<Plan> = {
  height: 'auto',
  stripe: true,
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
    isCurrent: true,
  },

  proxyConfig: {
    ajax: {
      query: async ({ page }, formValues) => {
        return await planViewStore.fetchPlanList(
          page.currentPage,
          page.pageSize,
          formValues,
        );
      },
    },
  },

  // 左栏分栏主表：内容列不设宽度（均分贴合容器防横向溢出），仅操作列定宽
  columns: [
    {
      title: $t('page.plan.name'),
      field: 'name',
      align: 'left',
    },
    {
      title: $t('page.plan.version'),
      field: 'version',
      slots: { default: 'version' },
    },
    {
      title: $t('page.plan.expiryPolicy'),
      field: 'expiryPolicy',
      slots: { default: 'expiryPolicy' },
    },
    {
      title: $t('page.plan.dataRetentionDays'),
      field: 'dataRetentionDays',
    },
    {
      title: $t('ui.table.description'),
      field: 'description',
    },
    {
      title: $t('ui.table.remark'),
      field: 'remark',
    },
    {
      title: $t('ui.table.createdAt'),
      field: 'createdAt',
      formatter: 'formatDateTime',
    },
    {
      title: $t('ui.table.action'),
      field: 'action',
      slots: { default: 'action' },
      width: 90,
    },
  ],
};

const gridEvents: VxeGridListeners<Plan> = {
  cellClick: ({ row }) => {
    planViewStore.setCurrentPlanId(
      typeof row.id === 'number' ? row.id : null,
    );
  },
};

const exportFetcher = (page: number, pageSize: number) =>
  planViewStore.fetchPlanList(page, pageSize, {});

const [Grid, gridApi] = useVbenVxeGrid({
  gridOptions,
  formOptions,
  gridEvents,
});

const [Drawer, drawerApi] = useVbenDrawer({
  connectedComponent: PlanDrawer,

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
    await deletePlan({ id: row.id });

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
</script>

<template>
  <Grid :table-title="$t('page.plan.planList')">
    <template #toolbar-tools>
      <a-button type="primary" @click="handleCreate">
        {{ $t('page.plan.button.create') }}
      </a-button>
          <TableExportButton :fetcher="exportFetcher" :columns="gridOptions.columns" filename="export" />
    </template>
    <template #version="{ row }">
      <a-tag :color="planVersionToColor(row.version)">
        {{ planVersionToName(row.version) }}
      </a-tag>
    </template>
    <template #expiryPolicy="{ row }">
      <a-tag :color="planExpiryPolicyToColor(row.expiryPolicy)">
        {{ planExpiryPolicyToName(row.expiryPolicy) }}
      </a-tag>
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
