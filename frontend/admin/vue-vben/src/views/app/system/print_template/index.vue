<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { h } from 'vue';

import { Page, useVbenDrawer, type VbenFormProps } from '@vben/common-ui';
import { LucideCopy, LucideFilePenLine, LucideTrash2 } from '@vben/icons';

import { notification } from 'ant-design-vue';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import {
  fetchListPrintTemplates,
  PaginationQuery,
  useDeletePrintTemplate,
} from '#/api';
import { type printservicev1_PrintTemplate as PrintTemplate } from '#/api';
import { $t } from '#/locales';
import TableActions from '#/components/TableActions.vue';
import type { TableAction } from '#/components/TableActions.vue';
import TableAddButton from '#/components/TableAddButton.vue';
import TableExportButton from '#/components/TableExportButton.vue';

import DesignerDrawer from './designer-drawer.vue';

const { mutateAsync: deleteTemplate } = useDeletePrintTemplate();

const formOptions: VbenFormProps = {
  collapsed: false,
  showCollapseButton: false,
  submitOnEnter: true,
  schema: [
    {
      component: 'Input',
      fieldName: 'name',
      label: $t('page.printTemplate.name'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Input',
      fieldName: 'code',
      label: $t('page.printTemplate.code'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
  ],
};

const gridOptions: VxeGridProps<PrintTemplate> = {
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
        return await fetchListPrintTemplates(
          new PaginationQuery({
            paging: { page: page.currentPage, pageSize: page.pageSize },
            formValues,
          }),
        );
      },
    },
  },

  columns: [
    { title: $t('page.printTemplate.name'), field: 'name', minWidth: 180 },
    { title: $t('page.printTemplate.code'), field: 'code', minWidth: 190 },
    {
      title: $t('page.printTemplate.resourceKey'),
      field: 'resourceKey',
      minWidth: 130,
      showOverflow: 'title',
    },
    { title: $t('page.printTemplate.paperSize'), field: 'paperSize', width: 100 },
    {
      title: $t('page.printTemplate.orientation'),
      field: 'orientation',
      width: 100,
      slots: { default: 'orientation' },
    },
    {
      title: $t('page.printTemplate.isEnabled'),
      field: 'isEnabled',
      width: 90,
      slots: { default: 'isEnabled' },
    },
    {
      title: $t('page.printTemplate.isBuiltIn'),
      field: 'isBuiltIn',
      width: 90,
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
      width: 180,
    },
  ],
};

const exportFetcher = (page: number, pageSize: number) =>
  fetchListPrintTemplates(new PaginationQuery({ paging: { page, pageSize } }));

const [Grid, gridApi] = useVbenVxeGrid({ gridOptions, formOptions });

const [Drawer, drawerApi] = useVbenDrawer({
  connectedComponent: DesignerDrawer,

  onOpenChange(isOpen: boolean) {
    if (!isOpen) {
      gridApi.reload();
    }
  },
});

function openDrawer(options: { create?: boolean; copy?: boolean; row?: any }) {
  drawerApi.setData(options);
  drawerApi.open();
}

/* 创建 */
function handleCreate() {
  openDrawer({ create: true });
}

/* 编辑 */
function handleEdit(row: any) {
  openDrawer({ row });
}

/* 复制派生（预填，保存走新建） */
function handleCopy(row: any) {
  openDrawer({ copy: true, row });
}

/* 删除 */
async function handleDelete(row: any) {
  try {
    await deleteTemplate({ id: row.id });

    notification.success({
      message: $t('ui.notification.delete_success'),
    });

    await gridApi.reload();
  } catch (error) {
    console.error('delete print template failed:', error);
    notification.error({
      message: $t('ui.notification.delete_failed'),
    });
  }
}

/* 操作列描述符：编辑/复制行内、删除 Popconfirm 确认（内置行 hidden 剔除） */
function buildRowActions(): TableAction[] {
  return [
    {
      key: 'edit',
      label: $t('common.edit'),
      icon: h(LucideFilePenLine),
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
      hidden: true, // 占位：具体行在渲染闭包里按 isBuiltIn 计算（见模板内联）
      confirm: {
        title: $t('ui.text.do_you_want_delete', {
          moduleName: $t('page.printTemplate.moduleName'),
        }),
      },
      onClick: handleDelete,
    },
  ];
}

/* 每行独立描述符：内置行剔除删除 */
function rowActions(row: PrintTemplate): TableAction[] {
  return buildRowActions()
    .map((a) =>
      a.key === 'delete' ? { ...a, hidden: !!row.isBuiltIn } : a,
    );
}
</script>

<template>
  <Page auto-content-height>
    <Grid :table-title="$t('menu.system.printTemplates')">
      <template #toolbar-tools>
        <TableExportButton :fetcher="exportFetcher" :columns="gridOptions.columns" filename="print-templates" />
        <TableAddButton @click="handleCreate" />
      </template>
      <template #orientation="{ row }">
        {{ row.orientation === 'landscape' ? $t('page.printTemplate.landscape') : $t('page.printTemplate.portrait') }}
      </template>
      <template #isEnabled="{ row }">
        <a-tag :color="row.isEnabled ? 'green' : 'default'">
          {{ row.isEnabled ? $t('page.printTemplate.enabledTag') : $t('page.printTemplate.disabledTag') }}
        </a-tag>
      </template>
      <template #isBuiltIn="{ row }">
        <a-tag :color="row.isBuiltIn ? 'gold' : 'default'">
          {{ row.isBuiltIn ? $t('page.printTemplate.builtInTag') : $t('page.printTemplate.customTag') }}
        </a-tag>
      </template>
      <template #action="{ row }">
        <TableActions :row="row" :actions="rowActions(row)" />
      </template>
    </Grid>
    <Drawer />
  </Page>
</template>
