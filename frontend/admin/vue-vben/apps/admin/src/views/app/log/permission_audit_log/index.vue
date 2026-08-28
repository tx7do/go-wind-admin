<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { Page, useVbenDrawer, type VbenFormProps } from '@vben/common-ui';
import { LucideEye } from '@vben/icons';

import dayjs from 'dayjs';

import { h } from 'vue';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import {
  fetchListPermissionAuditLogs,
  PaginationQuery,
  permissionAuditLogActionList,
  permissionAuditLogActionToColor,
  permissionAuditLogActionToName,
} from '#/api';
import { type auditservicev1_PermissionAuditLog as PermissionAuditLog } from '#/api';
import { $t } from '#/locales';

import PermissionAuditLogDetailDrawer from './permission-audit-log-detail-drawer.vue';

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
      fieldName: 'targetType',
      label: $t('page.permissionAuditLog.targetType'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Input',
      fieldName: 'operatorName',
      label: $t('page.permissionAuditLog.operatorName'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Select',
      fieldName: 'action',
      label: $t('page.permissionAuditLog.action'),
      componentProps: {
        options: permissionAuditLogActionList,
        placeholder: $t('ui.placeholder.select'),
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        allowClear: true,
        showSearch: true,
      },
    },
    {
      component: 'Input',
      fieldName: 'ipAddress',
      label: $t('page.permissionAuditLog.ipAddress'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'RangePicker',
      fieldName: 'createdAt',
      label: $t('page.permissionAuditLog.createdAt'),
      componentProps: {
        showTime: true,
        allowClear: true,
        presets: [
          {
            label: $t('ui.dateRange.today'),
            value: [dayjs().startOf('day'), dayjs().endOf('day')],
          },
          {
            label: $t('ui.dateRange.yesterday'),
            value: [
              dayjs().subtract(1, 'day').startOf('day'),
              dayjs().subtract(1, 'day').endOf('day'),
            ],
          },
          {
            label: $t('ui.dateRange.thisWeek'),
            value: [dayjs().startOf('week'), dayjs().endOf('week')],
          },
          {
            label: $t('ui.dateRange.lastWeek'),
            value: [
              dayjs().subtract(1, 'week').startOf('week'),
              dayjs().subtract(1, 'week').endOf('week'),
            ],
          },
          {
            label: $t('ui.dateRange.thisMonth'),
            value: [dayjs().startOf('month'), dayjs().endOf('month')],
          },
          {
            label: $t('ui.dateRange.lastMonth'),
            value: [
              dayjs().subtract(1, 'month').startOf('month'),
              dayjs().subtract(1, 'month').endOf('month'),
            ],
          },
        ],
      },
    },
  ],
};

const gridOptions: VxeGridProps<PermissionAuditLog> = {
  toolbarConfig: {
    custom: true,
    export: true,
    // import: true,
    refresh: true,
    zoom: true,
  },
  height: 'auto',
  exportConfig: {},
  pagerConfig: {},
  rowConfig: {
    isHover: true,
  },
  stripe: true,

  proxyConfig: {
    ajax: {
      query: async ({ page }, formValues) => {
        console.log('query:', formValues);

        let startTime: any;
        let endTime: any;
        if (
          formValues.createdAt !== undefined &&
          formValues.createdAt.length === 2
        ) {
          startTime = dayjs(formValues.createdAt[0]).format(
            'YYYY-MM-DD HH:mm:ss',
          );
          endTime = dayjs(formValues.createdAt[1]).format(
            'YYYY-MM-DD HH:mm:ss',
          );
          console.log(startTime, endTime);
        }

        return await fetchListPermissionAuditLogs(
          new PaginationQuery({
            paging: { page: page.currentPage, pageSize: page.pageSize },
            formValues: {
              username: formValues.username,
              action: formValues.action,
              path: formValues.path,
              ipAddress: formValues.ipAddress,
              created_at__gte: startTime,
              created_at__lte: endTime,
            },
            orderBy: ['-created_at'],
          }),
        );
      },
    },
  },

  columns: [
    {
      title: $t('page.permissionAuditLog.createdAt'),
      field: 'createdAt',
      formatter: 'formatDateTime',
      width: 140,
    },
    {
      title: $t('page.permissionAuditLog.action'),
      field: 'action',
      slots: { default: 'actionTag' },
    },
    { title: $t('page.permissionAuditLog.targetType'), field: 'targetType' },
    { title: $t('page.permissionAuditLog.targetName'), field: 'targetName' },
    { title: $t('page.permissionAuditLog.reason'), field: 'reason' },
    {
      title: $t('page.permissionAuditLog.operatorName'),
      field: 'operatorName',
    },
    {
      title: $t('page.permissionAuditLog.ipAddress'),
      field: 'ipAddress',
      width: 140,
    },
    {
      title: $t('ui.table.action'),
      field: 'action',
      fixed: 'right',
      slots: { default: 'action' },
      width: 80,
    },
  ],
};

const [Grid] = useVbenVxeGrid({ gridOptions, formOptions });

const [Drawer, drawerApi] = useVbenDrawer({
  connectedComponent: PermissionAuditLogDetailDrawer,
});

function handleView(row: PermissionAuditLog) {
  drawerApi.setData({ row });
  drawerApi.open();
}
</script>

<template>
  <Page auto-content-height>
    <Grid :table-title="$t('menu.log.permissionAuditLog')">
      <template #actionTag="{ row }">
        <a-tag :color="permissionAuditLogActionToColor(row.action)">
          {{ permissionAuditLogActionToName(row.action) }}
        </a-tag>
      </template>
      <template #action="{ row }">
        <a-button
          type="link"
          :icon="h(LucideEye)"
          @click="handleView(row)"
        />
      </template>
    </Grid>
    <Drawer />
  </Page>
</template>
