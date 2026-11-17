<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

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
  useDeletePermission,
  useSyncPermissions,
} from '#/api';
import { type permissionservicev1_Permission as Permission } from '#/api';
import { $t } from '#/locales';
import TableActions from '#/components/TableActions.vue';
import type { TableAction } from '#/components/TableActions.vue';
import { usePermissionViewStore } from '#/views/app/permission/permission/permission-view.state';

import PermissionDrawer from './permission-drawer.vue';

const { mutateAsync: deletePermission } = useDeletePermission();
const { mutateAsync: syncPermissions, isPending: isSyncing } = useSyncPermissions();
const permissionViewStore = usePermissionViewStore();

const formOptions: VbenFormProps = {
  // 默认展开
  // 控制表单是否显示折叠按钮
  showCollapseButton: false,
  // 按下回车时是否提交表单
  submitOnEnter: true,
  schema: [
    {
      component: 'Input',
      fieldName: 'name',
      label: $t('page.permission.name'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Input',
      fieldName: 'code',
      label: $t('page.permission.code'),
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

const gridOptions: VxeGridProps<Permission> = {
  toolbarConfig: {
    custom: true,
    export: true,
    // import: true,
    refresh: true,
    zoom: true,
  },

  exportConfig: {},
  pagerConfig: {
    enabled: false,
  },
  rowConfig: {
    isHover: true,
  },

  stripe: true,
  height: 'auto',

  proxyConfig: {
    ajax: {
      query: async ({ page }, formValues) => {

        return await permissionViewStore.fetchPermissionList(
          permissionViewStore.currentGroupId,
          page.currentPage,
          page.pageSize,
          formValues,
        );
      },
    },
  },

  columns: [
    {
      title: $t('page.permission.name'),
      field: 'name',
      fixed: 'left',
      align: 'left',
    },
    {
      title: $t('page.permission.code'),
      field: 'code',
      fixed: 'left',
      align: 'left',
    },
    { title: $t('page.permission.groupName'), field: 'groupName' },
    {
      title: $t('ui.table.status'),
      field: 'status',
      slots: { default: 'status' },
      width: 90,
    },
    {
      title: $t('ui.table.action'),
      field: 'action',
      fixed: 'right',
      slots: { default: 'action' },
      width: 140,
    },
  ],
};

const exportFetcher = (page: number, pageSize: number) =>
  permissionViewStore.fetchPermissionList(permissionViewStore.currentGroupId, page, pageSize, {});

const [Grid, gridApi] = useVbenVxeGrid({ gridOptions, formOptions });

const [Drawer, drawerApi] = useVbenDrawer({
  // 连接抽离的组件
  connectedComponent: PermissionDrawer,

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
    await deletePermission({ id: row.id });

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

/* 同步权限 */
async function handleSyncPermissions() {

  try {
    await syncPermissions();
    permissionViewStore.reloadGroupList();

    notification.success({
      message: $t('ui.notification.sync_success'),
    });
  } catch {
    notification.error({
      message: $t('ui.notification.sync_failed'),
    });
  }
}

watch(
  () => permissionViewStore.needReloadPermissionList,
  (newValues, oldValue) => {
    if (isEqual(newValues, oldValue) || !newValues) {
      return;
    }
    permissionViewStore.needReloadPermissionList = false;
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
          moduleName: $t('page.permission.moduleName'),
        }),
      },
      onClick: handleDelete,
    },
  ];
}
</script>

<template>
  <Grid :table-title="$t('menu.permission.permission')">
    <template #toolbar-tools>
      <a-popconfirm
        :cancel-text="$t('ui.button.cancel')"
        :ok-text="$t('ui.button.ok')"
        :title="
          $t('ui.text.do_you_want_sync_permissions', {
            moduleName: $t('page.permission.moduleName'),
          })
        "
        @confirm="() => handleSyncPermissions()"
      >
        <a-button class="gowind-btn-warning-outline mr-2" :loading="isSyncing">
          {{ $t('page.permission.button.syncPermissions') }}
        </a-button>
      </a-popconfirm>
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
