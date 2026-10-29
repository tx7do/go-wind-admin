<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { h, ref } from 'vue';

import { Page, type VbenFormProps } from '@vben/common-ui';
import { LucideCopy, LucidePencil, LucideTrash2 } from '@vben/icons';

import { notification } from 'ant-design-vue';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import { PaginationQuery } from '#/transport/rest';

import {
  fetchListBiDashboards,
  useCopyBiDashboard,
  useDeleteBiDashboard,
} from '#/api/composables/bi_dashboard';
import { type biservicev1_BiDashboard as BiDashboard } from '#/api/generated/admin/service/v1';
import { $t } from '#/locales';
import TableActions from '#/components/TableActions.vue';
import type { TableAction } from '#/components/TableActions.vue';
import TableAddButton from '#/components/TableAddButton.vue';
import TableExportButton from '#/components/TableExportButton.vue';

import BiDesignerDrawer from './bi-designer-drawer.vue';

const { mutateAsync: deleteDashboard } = useDeleteBiDashboard();
const { mutateAsync: copyDashboard } = useCopyBiDashboard();

const formOptions: VbenFormProps = {
  collapsed: false,
  showCollapseButton: false,
  submitOnEnter: true,
  schema: [
    {
      component: 'Input',
      fieldName: 'name',
      label: $t('page.bi.name'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Input',
      fieldName: 'code',
      label: $t('page.bi.code'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
  ],
};

const KIND_COLORS: Record<string, string> = {
  DASHBOARD: 'processing',
  SCREEN: 'purple',
  TEMPLATE: 'gold',
};

const gridOptions: VxeGridProps<BiDashboard> = {
  toolbarConfig: {
    custom: true,
    export: true,
    refresh: true,
    zoom: true,
  },
  exportConfig: {},
  pagerConfig: {},
  rowConfig: {
    isHover: true,
  },
  height: 'auto',
  stripe: true,

  proxyConfig: {
    ajax: {
      query: async ({ page }, formValues) => {
        return await fetchListBiDashboards(
          new PaginationQuery({
            paging: { page: page.currentPage, pageSize: page.pageSize },
            formValues,
          }),
        );
      },
    },
  },

  columns: [
    { title: $t('page.bi.name'), field: 'name', minWidth: 170 },
    { title: $t('page.bi.code'), field: 'code', minWidth: 180 },
    {
      title: $t('page.bi.kind'),
      field: 'kind',
      width: 90,
      slots: { default: 'kind' },
    },
    { title: $t('page.bi.chartCount'), field: 'chartCount', width: 80 },
    { title: $t('page.bi.themeKey'), field: 'themeKey', width: 100 },
    {
      title: $t('page.bi.isEnabled'),
      field: 'isEnabled',
      width: 80,
      slots: { default: 'isEnabled' },
    },
    {
      title: $t('page.bi.isBuiltIn'),
      field: 'isBuiltIn',
      width: 80,
      slots: { default: 'isBuiltIn' },
    },
    {
      title: $t('ui.table.createdAt'),
      field: 'createdAt',
      formatter: 'formatDateTime',
      width: 140,
    },
    {
      title: $t('ui.table.action'),
      field: 'action',
      fixed: 'right',
      slots: { default: 'action' },
      width: 190,
    },
  ],
};

const exportFetcher = (page: number, pageSize: number) =>
  fetchListBiDashboards(new PaginationQuery({ paging: { page, pageSize } }));

const [Grid, gridApi] = useVbenVxeGrid({ gridOptions, formOptions });

const designerRef = ref();

function handleSuccess() {
  gridApi.reload();
}

/* 创建 */
function handleCreate() {
  designerRef.value?.open({});
}

/* 编辑（= 设计） */
function handleEdit(row: any) {
  designerRef.value?.open({ row });
}

/* 复制派生：服务端整份复制成新行 */
async function handleCopy(row: any) {
  try {
    await copyDashboard({ id: row.id });
    notification.success({ message: $t('page.bi.copySuccess') });
    await gridApi.reload();
  } catch (error) {
    console.error('copy bi dashboard failed:', error);
    notification.error({ message: $t('page.bi.copyFailed') });
  }
}

/* 删除 */
async function handleDelete(row: any) {
  try {
    await deleteDashboard({ id: row.id });

    notification.success({
      message: $t('ui.notification.delete_success'),
    });

    await gridApi.reload();
  } catch (error) {
    console.error('delete bi dashboard failed:', error);
    notification.error({
      message: $t('ui.notification.delete_failed'),
    });
  }
}

/* 操作列描述符：设计/复制行内、删除 Popconfirm 确认（内置行剔除） */
function buildRowActions(): TableAction[] {
  return [
    {
      key: 'design',
      label: $t('page.bi.openDesigner'),
      icon: h(LucidePencil),
      onClick: handleEdit,
    },
    {
      key: 'copy',
      label: $t('common.copy'),
      icon: h(LucideCopy),
      onClick: handleCopy,
    },
    {
      key: 'delete',
      label: $t('common.delete'),
      icon: h(LucideTrash2),
      danger: true,
      hidden: true, // 占位：具体行在渲染闭包里按 isBuiltIn 计算
      confirm: {
        title: $t('ui.text.do_you_want_delete', {
          moduleName: $t('page.bi.moduleName'),
        }),
      },
      onClick: handleDelete,
    },
  ];
}

/* 每行独立描述符：内置行剔除删除 */
function rowActions(row: BiDashboard): TableAction[] {
  return buildRowActions()
    .map((a) =>
      a.key === 'delete' ? { ...a, hidden: !!row.isBuiltIn } : a,
    );
}
</script>

<template>
  <Page auto-content-height>
    <Grid :table-title="$t('page.bi.moduleName')">
      <template #toolbar-tools>
        <TableExportButton :fetcher="exportFetcher" :columns="gridOptions.columns" filename="bi-dashboards" />
        <TableAddButton @click="handleCreate" />
      </template>
      <template #kind="{ row }">
        <a-tag :color="KIND_COLORS[row.kind] || 'default'">
          {{ row.kind === 'SCREEN' ? $t('page.bi.kindScreen') : row.kind === 'TEMPLATE' ? $t('page.bi.kindTemplate') : $t('page.bi.kindDashboard') }}
        </a-tag>
      </template>
      <template #isEnabled="{ row }">
        <a-tag :color="row.isEnabled ? 'green' : 'default'">
          {{ row.isEnabled ? $t('page.bi.enabledTag') : $t('page.bi.disabledTag') }}
        </a-tag>
      </template>
      <template #isBuiltIn="{ row }">
        <a-tag :color="row.isBuiltIn ? 'gold' : 'default'">
          {{ row.isBuiltIn ? $t('page.bi.builtInTag') : $t('page.bi.customTag') }}
        </a-tag>
      </template>
      <template #action="{ row }">
        <TableActions :row="row" :actions="rowActions(row)" />
      </template>
    </Grid>
    <BiDesignerDrawer ref="designerRef" @success="handleSuccess" />
  </Page>
</template>
