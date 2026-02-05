<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { Page, type VbenFormProps } from '@vben/common-ui';

import dayjs from 'dayjs';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import { type auditservicev1_ApiAuditLog as ApiAuditLog } from '#/generated/api/admin/service/v1';
import { $t } from '#/locales';
import {
  permissionAuditLogActionList,
  permissionAuditLogActionToColor,
  permissionAuditLogActionToName,
  successToColor,
  successToNameWithStatusCode,
  usePermissionAuditLogStore,
} from '#/stores';

const permissionAuditLogStore = usePermissionAuditLogStore();

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
      },
    },
  ],
};

const gridOptions: VxeGridProps<ApiAuditLog> = {
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

        return await permissionAuditLogStore.listPermissionAuditLog(
          {
            page: page.currentPage,
            pageSize: page.pageSize,
          },
          {
            username: formValues.username,
            action: formValues.action,
            path: formValues.path,
            ipAddress: formValues.ipAddress,
            created_at__gte: startTime,
            created_at__lte: endTime,
          },
          null,
          ['-created_at'],
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
      slots: { default: 'action' },
      width: 80,
    },
    { title: $t('page.permissionAuditLog.targetType'), field: 'targetType' },
    { title: $t('page.permissionAuditLog.targetName'), field: 'targetName' },
    { title: $t('page.permissionAuditLog.reason'), field: 'reason' },
    {
      title: $t('page.permissionAuditLog.operatorName'),
      field: 'deviceInfo.operatorName',
    },
    {
      title: $t('page.permissionAuditLog.geoLocation'),
      field: 'geoLocation',
      slots: { default: 'geoLocation' },
    },
    {
      title: $t('page.permissionAuditLog.ipAddress'),
      field: 'ipAddress',
      width: 140,
    },
  ],
};

const [Grid] = useVbenVxeGrid({ gridOptions, formOptions });
</script>

<template>
  <Page auto-content-height>
    <Grid :table-title="$t('menu.log.permissionAuditLog')">
      <template #success="{ row }">
        <a-tag :color="successToColor(row.success)">
          {{ successToNameWithStatusCode(row.success, row.statusCode) }}
        </a-tag>
      </template>
      <template #geoLocation="{ row }">
        {{ row.geoLocation.province }} {{ row.geoLocation.city }}
      </template>
      <template #action="{ row }">
        <a-tag :color="permissionAuditLogActionToColor(row.action)">
          {{ permissionAuditLogActionToName(row.action) }}
        </a-tag>
      </template>
    </Grid>
  </Page>
</template>
