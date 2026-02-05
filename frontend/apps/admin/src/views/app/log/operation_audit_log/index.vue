<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { Page, type VbenFormProps } from '@vben/common-ui';

import dayjs from 'dayjs';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import { type auditservicev1_ApiAuditLog as ApiAuditLog } from '#/generated/api/admin/service/v1';
import { $t } from '#/locales';
import {
  operationAuditLogActionList,
  operationAuditLogActionToColor,
  operationAuditLogActionToName,
  successStatusList,
  successToColor,
  successToNameWithStatusCode,
  useOperationAuditLogStore,
} from '#/stores';

const operationAuditLogStore = useOperationAuditLogStore();

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
      fieldName: 'username',
      label: $t('page.operationAuditLog.username'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Input',
      fieldName: 'resourceType',
      label: $t('page.operationAuditLog.resourceType'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Select',
      fieldName: 'action',
      label: $t('page.operationAuditLog.action'),
      componentProps: {
        options: operationAuditLogActionList,
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
      label: $t('page.operationAuditLog.ipAddress'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Select',
      fieldName: 'success',
      label: $t('page.operationAuditLog.success'),
      componentProps: {
        options: successStatusList,
        placeholder: $t('ui.placeholder.select'),
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        allowClear: true,
        showSearch: true,
      },
    },
    {
      component: 'RangePicker',
      fieldName: 'createdAt',
      label: $t('page.operationAuditLog.createdAt'),
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

        return await operationAuditLogStore.listOperationAuditLog(
          {
            page: page.currentPage,
            pageSize: page.pageSize,
          },
          {
            username: formValues.username,
            resourceType: formValues.resourceType,
            action: formValues.action,
            ipAddress: formValues.ipAddress,
            success: formValues.success,
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
      title: $t('page.operationAuditLog.createdAt'),
      field: 'createdAt',
      formatter: 'formatDateTime',
      width: 140,
    },
    {
      title: $t('page.operationAuditLog.success'),
      field: 'success',
      slots: { default: 'success' },
      width: 80,
    },
    {
      title: $t('page.operationAuditLog.action'),
      field: 'action',
      slots: { default: 'action' },
      width: 80,
    },
    { title: $t('page.operationAuditLog.resourceType'), field: 'resourceType' },
    { title: $t('page.operationAuditLog.resourceId'), field: 'resourceId' },
    {
      title: $t('page.operationAuditLog.requestId'),
      field: 'requestId',
    },
    { title: $t('page.operationAuditLog.username'), field: 'username' },
    {
      title: $t('page.operationAuditLog.geoLocation'),
      field: 'geoLocation',
      slots: { default: 'geoLocation' },
    },
    {
      title: $t('page.operationAuditLog.ipAddress'),
      field: 'ipAddress',
      width: 140,
    },
  ],
};

const [Grid] = useVbenVxeGrid({ gridOptions, formOptions });
</script>

<template>
  <Page auto-content-height>
    <Grid :table-title="$t('menu.log.operationAuditLog')">
      <template #success="{ row }">
        <a-tag :color="successToColor(row.success)">
          {{ successToNameWithStatusCode(row.success, row.statusCode) }}
        </a-tag>
      </template>
      <template #geoLocation="{ row }">
        {{ row.geoLocation.province }} {{ row.geoLocation.city }}
      </template>
      <template #action="{ row }">
        <a-tag :color="operationAuditLogActionToColor(row.action)">
          {{ operationAuditLogActionToName(row.action) }}
        </a-tag>
      </template>
    </Grid>
  </Page>
</template>
