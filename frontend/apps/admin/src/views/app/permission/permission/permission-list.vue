<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { h, watch } from 'vue';

import { useVbenDrawer, type VbenFormProps } from '@vben/common-ui';
import { LucideFilePenLine, LucideTrash2 } from '@vben/icons';

import { notification } from 'ant-design-vue';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import { type permissionservicev1_Permission as Permission } from '#/generated/api/admin/service/v1';
import { $t } from '#/locales';
import {
  statusList,
  statusToColor,
  statusToName,
  usePermissionStore,
} from '#/stores';
import { usePermissionViewStore } from '#/views/app/permission/permission/permision_view.state';

import PermissionDrawer from './permission-drawer.vue';

const permissionStore = usePermissionStore();
const permissionViewStore = usePermissionViewStore();

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
        console.log('query:', formValues);

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
      width: 90,
    },
  ],
};

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
  console.log('创建');

  openDrawer(true);
}

/* 编辑 */
function handleEdit(row: any) {
  console.log('编辑', row);
  openDrawer(false, row);
}

/* 删除 */
async function handleDelete(row: any) {
  console.log('删除', row);

  try {
    await permissionStore.deletePermission(row.id);

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

async function handleSyncApis() {
  console.log('同步');

  try {
    await permissionStore.syncApis();

    notification.success({
      message: $t('ui.notification.sync_success'),
    });

    await gridApi.reload();
  } catch {
    notification.error({
      message: $t('ui.notification.sync_failed'),
    });
  }
}

async function handleSyncMenus() {
  console.log('同步');

  try {
    await permissionStore.syncMenus();

    notification.success({
      message: $t('ui.notification.sync_success'),
    });

    permissionViewStore.reloadGroupList();
  } catch {
    notification.error({
      message: $t('ui.notification.sync_failed'),
    });
  }
}

watch(
  () => permissionViewStore.needReloadPermissionList,
  () => {
    if (!permissionViewStore.needReloadPermissionList) {
      return;
    }
    permissionViewStore.needReloadPermissionList = false;
    gridApi.reload();
  },
);
</script>

<template>
  <Grid :table-title="$t('menu.permission.permission')">
    <template #toolbar-tools>
      <a-button class="mr-2" type="primary" @click="handleCreate">
        {{ $t('page.permission.button.create') }}
      </a-button>
      <a-popconfirm
        :cancel-text="$t('ui.button.cancel')"
        :ok-text="$t('ui.button.ok')"
        :title="
          $t('ui.text.do_you_want_sync_api', {
            moduleName: $t('page.permission.moduleName'),
          })
        "
        @confirm="() => handleSyncApis()"
      >
        <a-button type="primary" danger class="mr-2">
          {{ $t('page.permission.button.syncApi') }}
        </a-button>
      </a-popconfirm>
      <a-popconfirm
        :cancel-text="$t('ui.button.cancel')"
        :ok-text="$t('ui.button.ok')"
        :title="
          $t('ui.text.do_you_want_sync_api', {
            moduleName: $t('page.permission.moduleName'),
          })
        "
        @confirm="() => handleSyncMenus()"
      >
        <a-button type="primary" danger class="mr-2">
          {{ $t('page.permission.button.syncMenu') }}
        </a-button>
      </a-popconfirm>
    </template>
    <template #status="{ row }">
      <a-tag :color="statusToColor(row.status)">
        {{ statusToName(row.status) }}
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
            moduleName: $t('page.permission.moduleName'),
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
