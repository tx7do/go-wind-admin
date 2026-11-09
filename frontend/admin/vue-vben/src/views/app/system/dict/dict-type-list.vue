<script lang="ts" setup>
import type { VxeGridListeners, VxeGridProps } from '#/adapter/vxe-table';

import { h } from 'vue';

import { useVbenDrawer, type VbenFormProps } from '@vben/common-ui';
import { LucideFilePenLine, LucideTrash2 } from '@vben/icons';

import { notification } from 'ant-design-vue';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import TableAddButton from '#/components/TableAddButton.vue';
import TableExportButton from '#/components/TableExportButton.vue';
import { type dictservicev1_DictType as DictType } from '#/api';
import { enableBoolToColor, enableBoolToName, useDeleteDictType } from '#/api';
import { $t } from '#/locales';
import TableActions from '#/components/TableActions.vue';
import type { TableAction } from '#/components/TableActions.vue';
import { useDictViewStore } from '#/views/app/system/dict/dict-view.state';

import DictTypeDrawer from './dict-type-drawer.vue';

const { mutateAsync: deleteDictType } = useDeleteDictType();
const dictViewStore = useDictViewStore();

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
      fieldName: 'type_code',
      label: $t('page.dict.typeCode'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
  ],
};

const gridOptions: VxeGridProps<DictType> = {
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
        return await dictViewStore.fetchTypeList(
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
      title: $t('page.dict.typeName'),
      field: 'typeName',
      align: 'left',
    },
    {
      title: $t('page.dict.typeCode'),
      field: 'typeCode',
      align: 'left',
    },
    {
      title: $t('ui.table.status'),
      field: 'isEnabled',
      slots: { default: 'isEnabled' },
    },
    {
      title: $t('ui.table.action'),
      field: 'action',
      slots: { default: 'action' },
      width: 140,
    },
  ],
};

const gridEvents: VxeGridListeners<DictType> = {
  // cellDblclick: ({ row }) => {
  //   dictViewStore.setCurrentMain(typeof row.id === 'string' && row.id !== '' ? row.id : '');
  // },
  cellClick: ({ row }) => {
    dictViewStore.setCurrentTypeId(
      typeof row.id === 'string' && row.id !== '' ? row.id : '',
    );
  },
};

const exportFetcher = (page: number, pageSize: number) =>
  dictViewStore.fetchTypeList(page, pageSize, {});

const [Grid, gridApi] = useVbenVxeGrid({
  gridOptions,
  formOptions,
  gridEvents,
});

const [Drawer, drawerApi] = useVbenDrawer({
  connectedComponent: DictTypeDrawer,

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
    await deleteDictType({ ids: [row.id] });

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

/* 操作列描述符：编辑行内、删除 Popconfirm 确认（≤3 个全行内，定式由 TableActions 统一执行） */
function buildRowActions(): TableAction[] {
  return [
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
          moduleName: $t('page.dict.moduleName'),
        }),
      },
      onClick: handleDelete,
    },
  ];
}
</script>

<template>
  <Grid :table-title="$t('page.dict.dictTypeList')">
    <template #toolbar-tools>
          <TableExportButton :fetcher="exportFetcher" :columns="gridOptions.columns" filename="export" />
          <TableAddButton @click="handleCreate" />
    </template>
    <template #isEnabled="{ row }">
      <a-tag :color="enableBoolToColor(row.isEnabled)">
        {{ enableBoolToName(row.isEnabled) }}
      </a-tag>
    </template>
    <template #action="{ row }">
      <TableActions :row="row" :actions="buildRowActions()" />
    </template>
  </Grid>
  <Drawer />
</template>
