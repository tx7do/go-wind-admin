<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { h, ref } from 'vue';

import { Page, useVbenDrawer, type VbenFormProps } from '@vben/common-ui';
import { LucideFilePenLine, LucideTrash2 } from '@vben/icons';
// 包内 iconify 集未导出 ArchiveRestore —— 宿主自建（同 createIconifyIcon 工厂）
import { createIconifyIcon } from '@vben/icons';

const LucideArchiveRestore = createIconifyIcon('lucide:archive-restore');

import { notification } from 'ant-design-vue';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import { type identityservicev1_Tenant as Tenant } from '#/api';
import {
  fetchListTenants,
  fetchTrashTenants,
  PaginationQuery,
  tenantAuditStatusList,
  tenantAuditStatusToColor,
  tenantAuditStatusToName,
  tenantStatusList,
  tenantStatusToColor,
  tenantStatusToName,
  tenantTypeList,
  tenantTypeToColor,
  tenantTypeToName,
  useDeleteTenant,
  usePurgeTenant,
  useRestoreTenant,
} from '#/api';
import { $t } from '#/locales';
import TableActions from '#/components/TableActions.vue';
import type { TableAction } from '#/components/TableActions.vue';
import TableAddButton from '#/components/TableAddButton.vue';
import TableExportButton from '#/components/TableExportButton.vue';

import TenantDrawer from './tenant-drawer.vue';

const { mutateAsync: deleteTenant } = useDeleteTenant();
const { mutateAsync: restoreTenant } = useRestoreTenant();
const { mutateAsync: purgeTenant } = usePurgeTenant();

// 回收站二态（issue #3）：active=在营名录 / trash=已删除（停车）租户
const viewMode = ref<'active' | 'trash'>('active');

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
        const fetcher =
          viewMode.value === 'trash' ? fetchTrashTenants : fetchListTenants;
        return await fetcher(
          new PaginationQuery({
            paging: {
              page: page.currentPage,
              pageSize: page.pageSize,
            },
            formValues,
          }),
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
    {
      title: $t('page.tenant.memberCount'),
      field: 'memberCount',
    },
    { title: $t('ui.table.remark'), field: 'remark' },
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
  fetchListTenants(new PaginationQuery({ paging: { page, pageSize } }));

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

  openModal(true);
}

/* 编辑 */
function handleEdit(row: any) {
  openModal(false, row);
}

/* 删除 */
async function handleDelete(row: any) {

  try {
    await deleteTenant({ id: row.id });

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

/* 回收站：恢复停车租户（code/name 已被在营租户占用时后端报错） */
async function handleRestore(row: any) {
  try {
    await restoreTenant({ id: row.id });
    notification.success({ message: $t('page.tenant.restoreSuccess') });
    await gridApi.reload();
  } catch (error: any) {
    notification.error({
      message: error?.message || $t('ui.notification.delete_failed'),
    });
  }
}

/* 回收站：彻底删除（不可逆） */
async function handlePurge(row: any) {
  try {
    await purgeTenant({ id: row.id });
    notification.success({ message: $t('page.tenant.purgeSuccess') });
    await gridApi.reload();
  } catch (error: any) {
    notification.error({
      message: error?.message || $t('ui.notification.delete_failed'),
    });
  }
}

/* 回收站模式切换：重查（视图切换由 query 分叉承担） */
function handleViewModeChange(mode: 'active' | 'trash') {
  viewMode.value = mode;
  gridApi.reload();
}

/* 操作列描述符：在营=编辑/删除，回收站=恢复/彻底删除（≤3 个全行内，定式由 TableActions 统一执行） */
function buildRowActions(): TableAction[] {
  if (viewMode.value === 'trash') {
    return [
      {
        key: 'restore',
        label: $t('page.tenant.restore'),
        icon: h(LucideArchiveRestore),
        success: true,
        confirm: {
          title: $t('page.tenant.restoreConfirm', {
            moduleName: $t('page.tenant.moduleName'),
          }),
        },
        onClick: handleRestore,
      },
      {
        key: 'purge',
        label: $t('page.tenant.purge'),
        icon: h(LucideTrash2),
        danger: true,
        confirm: {
          title: $t('page.tenant.purgeConfirmTitle'),
          description: $t('page.tenant.purgeConfirmDesc'),
        },
        onClick: handlePurge,
      },
    ];
  }
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
          moduleName: $t('page.tenant.moduleName'),
        }),
      },
      onClick: handleDelete,
    },
  ];
}
</script>

<template>
  <Page auto-content-height>
    <Grid
      :table-title="
        viewMode === 'trash'
          ? $t('page.tenant.trash')
          : $t('menu.tenant.member')
      "
    >
      <template #toolbar-tools>
        <a-radio-group
          size="small"
          :value="viewMode"
          class="mr-2"
          @change="(e: any) => handleViewModeChange(e.target.value)"
        >
          <a-radio-button value="active">
            {{ $t('page.tenant.activeTenants') }}
          </a-radio-button>
          <a-radio-button value="trash">
            {{ $t('page.tenant.trash') }}
          </a-radio-button>
        </a-radio-group>
        <TableExportButton
          v-if="viewMode === 'active'"
          :fetcher="exportFetcher"
          :columns="gridOptions.columns"
          filename="tenants"
        />
        <TableAddButton v-if="viewMode === 'active'" @click="handleCreate" />
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
        <TableActions :row="row" :actions="buildRowActions()" />
      </template>
    </Grid>
    <Drawer />
  </Page>
</template>
