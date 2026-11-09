<script lang="ts" setup>
import type { VxeGridListeners, VxeGridProps } from '#/adapter/vxe-table';

import { h, watch } from 'vue';

import { useVbenDrawer, type VbenFormProps } from '@vben/common-ui';
import { LucideFilePenLine, LucideTrash2 } from '@vben/icons';
import { isEqual } from '@vben/utils';

import { notification } from 'ant-design-vue';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import TableAddButton from '#/components/TableAddButton.vue';
import TableExportButton from '#/components/TableExportButton.vue';
import {
  statusList,
  statusToColor,
  statusToName,
  useDeletePermissionGroup,
} from '#/api';
import { type permissionservicev1_PermissionGroup as PermissionGroup } from '#/api';
import { $t } from '#/locales';
import TableActions from '#/components/TableActions.vue';
import type { TableAction } from '#/components/TableActions.vue';
import { usePermissionViewStore } from '#/views/app/permission/permission/permission-view.state';

import PermissionGroupDrawer from './permission-group-drawer.vue';

const permissionViewStore = usePermissionViewStore();
const { mutateAsync: deletePermissionGroup } = useDeletePermissionGroup();

const formOptions: VbenFormProps = {
  // 默认展开
  collapsed: false,
  // 控制表单是否显示折叠按钮
  showCollapseButton: false,
  // 按下回车时是否提交表单
  submitOnEnter: true,
  // 左栏窄容器：默认 md:grid-cols-3 视口断点会把三个字段挤成细条，固定一行两字段
  wrapperClass: 'grid-cols-1 sm:grid-cols-2',
  schema: [
    {
      component: 'Input',
      fieldName: 'name',
      label: $t('page.permissionGroup.name'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Input',
      fieldName: 'module',
      label: $t('page.permissionGroup.module'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Select',
      fieldName: 'status',
      label: $t('ui.table.status'),
      componentProps: {
        options: statusList,
        placeholder: $t('ui.placeholder.select'),
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        allowClear: true,
        showSearch: true,
      },
    },
  ],
};

const gridOptions: VxeGridProps<PermissionGroup> = {
  // 分组面板工具栏只留刷新圆钮：导出有独立「导出全部」文字按钮，全屏/列设置冗余
  toolbarConfig: {
    export: false,
    refresh: true,
    zoom: false,
    custom: false,
  },
  pagerConfig: {
    enabled: false,
  },
  rowConfig: {
    isHover: true,
    isCurrent: true,
  },
  treeConfig: {
    parentField: 'parentId',
    rowField: 'id',
    expandAll: true,
  },

  height: 'auto',

  proxyConfig: {
    ajax: {
      query: async ({ page }, formValues) => {

        return await permissionViewStore.fetchGroupList(
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
      title: $t('page.permissionGroup.name'),
      field: 'name',
      align: 'left',
      treeNode: true,
    },
    {
      title: $t('page.permissionGroup.module'),
      field: 'module',
      align: 'left',
    },
    {
      title: $t('ui.table.status'),
      field: 'status',
      slots: { default: 'status' },
    },
    {
      title: $t('ui.table.action'),
      field: 'action',
      slots: { default: 'action' },
      width: 140,
    },
  ],
};

const gridEvents: VxeGridListeners<PermissionGroup> = {
  cellClick: ({ row }) => {
    permissionViewStore.setCurrentGroupId(
      typeof row.id === 'string' && row.id !== '' ? row.id : '',
    );
  },
};

const exportFetcher = (page: number, pageSize: number) =>
  permissionViewStore.fetchGroupList(page, pageSize, {});

const [Grid, gridApi] = useVbenVxeGrid({
  gridOptions,
  gridEvents,
  formOptions,
});

const [Drawer, drawerApi] = useVbenDrawer({
  // 连接抽离的组件
  connectedComponent: PermissionGroupDrawer,

  onOpenChange(isOpen: boolean) {
    if (!isOpen) {
      // 关闭时，重载表格数据
      gridApi.reload();
    }
  },
});

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
    await deletePermissionGroup({ id: row.id });

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

const expandAll = () => {
  gridApi.grid?.setAllTreeExpand(true);
};

const collapseAll = () => {
  gridApi.grid?.setAllTreeExpand(false);
};

watch(
  () => permissionViewStore.needReloadGroupList,
  (newValues, oldValue) => {
    if (isEqual(newValues, oldValue) || !newValues) {
      return;
    }
    permissionViewStore.needReloadGroupList = false;
    gridApi.reload();
  },
);

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
          moduleName: $t('page.permissionGroup.moduleName'),
        }),
      },
      onClick: handleDelete,
    },
  ];
}
</script>

<template>
  <Grid :table-title="$t('page.permissionGroup.moduleName')">
    <template #toolbar-tools>
      <a-button class="mr-2" @click="expandAll">
        {{ $t('ui.tree.expand_all') }}
      </a-button>
      <a-button class="mr-2" @click="collapseAll">
        {{ $t('ui.tree.collapse_all') }}
      </a-button>
          <TableExportButton :fetcher="exportFetcher" :columns="gridOptions.columns" filename="export" />
          <TableAddButton @click="handleCreate" />
    </template>
    <template #status="{ row }">
      <a-tag :color="statusToColor(row.status)">
        {{ statusToName(row.status) }}
      </a-tag>
    </template>
    <template #action="{ row }">
      <TableActions :row="row" :actions="buildRowActions()" />
    </template>
  </Grid>
  <Drawer />
</template>
