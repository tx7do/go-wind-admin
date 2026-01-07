<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { h } from 'vue';

import { Page, useVbenDrawer, type VbenFormProps } from '@vben/common-ui';
import { LucideFilePenLine, LucideTrash2 } from '@vben/icons';

import { notification } from 'ant-design-vue';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import { type userservicev1_Tenant as Tenant } from '#/generated/api/admin/service/v1';
import { $t } from '#/locales';
import {
  tenantAuditStatusList,
  tenantAuditStatusToColor,
  tenantAuditStatusToName,
  tenantStatusList,
  tenantStatusToColor,
  tenantStatusToName,
  tenantTypeList,
  tenantTypeToColor,
  tenantTypeToName,
  useTenantStore,
} from '#/stores';

import TenantDrawer from './tenant-drawer.vue';

const tenantStore = useTenantStore();

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
      label: $t('page.tenant.name'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Input',
      fieldName: 'code',
      label: $t('page.tenant.code'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Select',
      fieldName: 'type',
      label: $t('page.tenant.type'),
      componentProps: {
        placeholder: $t('ui.placeholder.select'),
        options: tenantTypeList,
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        allowClear: true,
        showSearch: true,
      },
    },
    {
      component: 'Select',
      fieldName: 'auditStatus',
      label: $t('page.tenant.auditStatus'),
      componentProps: {
        placeholder: $t('ui.placeholder.select'),
        options: tenantAuditStatusList,
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        allowClear: true,
        showSearch: true,
      },
    },
    {
      component: 'Select',
      fieldName: 'status',
      label: $t('ui.table.status'),
      componentProps: {
        options: tenantStatusList,
        placeholder: $t('ui.placeholder.select'),
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        allowClear: true,
        showSearch: true,
      },
    },
  ],
};

const gridOptions: VxeGridProps<Tenant> = {
  height: 'auto',
  stripe: false,
  toolbarConfig: {
    custom: true,
    export: true,
    import: false,
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

  proxyConfig: {
    ajax: {
      query: async ({ page }, formValues) => {
        console.log('query:', formValues);
        return await tenantStore.listTenant(
          {
            page: page.currentPage,
            pageSize: page.pageSize,
          },
          formValues,
        );
      },
    },
  },

  columns: [
    { title: $t('ui.table.seq'), type: 'seq', width: 50 },
    { title: $t('page.tenant.name'), field: 'name' },
    { title: $t('page.tenant.code'), field: 'code' },
    { title: $t('page.tenant.adminUserName'), field: 'adminUserName' },
    {
      title: $t('page.tenant.type'),
      field: 'type',
      slots: { default: 'type' },
      width: 95,
    },
    {
      title: $t('page.tenant.auditStatus'),
      field: 'auditStatus',
      slots: { default: 'audit-status' },
      width: 95,
    },
    {
      title: $t('ui.table.status'),
      field: 'status',
      slots: { default: 'status' },
      width: 95,
    },
    {
      title: $t('ui.table.createdAt'),
      field: 'createdAt',
      formatter: 'formatDateTime',
      width: 140,
    },
    { title: $t('ui.table.remark'), field: 'remark' },
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
  connectedComponent: TenantDrawer,

  onOpenChange(isOpen: boolean) {
    if (!isOpen) {
      // 关闭时，重载表格数据
      gridApi.reload();
    }
  },
});

/* 打开模态窗口 */
function openModal(create: boolean, row?: any) {
  drawerApi.setData({
    create,
    row,
  });

  drawerApi.open();
}

/* 创建 */
function handleCreate() {
  console.log('创建');

  openModal(true);
}

/* 编辑 */
function handleEdit(row: any) {
  console.log('编辑', row);
  openModal(false, row);
}

/* 删除 */
async function handleDelete(row: any) {
  console.log('删除', row);

  try {
    await tenantStore.deleteTenant(row.id);

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
  <Page auto-content-height>
    <Grid :table-title="$t('menu.tenant.member')">
      <template #toolbar-tools>
        <a-button class="mr-2" type="primary" @click="handleCreate">
          {{ $t('page.tenant.button.create') }}
        </a-button>
      </template>

      <template #status="{ row }">
        <a-tag :color="tenantStatusToColor(row.status)">
          {{ tenantStatusToName(row.status) }}
        </a-tag>
      </template>
      <template #type="{ row }">
        <a-tag :color="tenantTypeToColor(row.type)">
          {{ tenantTypeToName(row.type) }}
        </a-tag>
      </template>
      <template #audit-status="{ row }">
        <a-tag :color="tenantAuditStatusToColor(row.auditStatus)">
          {{ tenantAuditStatusToName(row.auditStatus) }}
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
              moduleName: $t('page.tenant.moduleName'),
            })
          "
          @confirm="handleDelete(row)"
        >
          <a-button danger type="link" :icon="h(LucideTrash2)" />
        </a-popconfirm>
      </template>
    </Grid>
    <Drawer />
  </Page>
</template>
