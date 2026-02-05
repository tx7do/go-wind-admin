<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { Page, type VbenFormProps } from '@vben/common-ui';

import dayjs from 'dayjs';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import { type auditservicev1_ApiAuditLog as ApiAuditLog } from '#/generated/api/admin/service/v1';
import { $t } from '#/locales';
import {
  dataAccessAuditLogAccessTypeList,
  dataAccessAuditLogAccessTypeToColor,
  dataAccessAuditLogAccessTypeToName,
  successStatusList,
  successToColor,
  successToNameWithStatusCode,
  useDataAccessAuditLogStore,
} from '#/stores';

const dataAccessAuditLogStore = useDataAccessAuditLogStore();

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
      label: $t('page.dataAccessAuditLog.username'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Input',
      fieldName: 'tableName',
      label: $t('page.dataAccessAuditLog.tableName'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Select',
      fieldName: 'accessType',
      label: $t('page.dataAccessAuditLog.accessType'),
      componentProps: {
        options: dataAccessAuditLogAccessTypeList,
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
      label: $t('page.dataAccessAuditLog.ipAddress'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Select',
      fieldName: 'success',
      label: $t('page.dataAccessAuditLog.success'),
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
      label: $t('page.dataAccessAuditLog.createdAt'),
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

        return await dataAccessAuditLogStore.listDataAccessAuditLog(
          {
            page: page.currentPage,
            pageSize: page.pageSize,
          },
          {
            username: formValues.username,
            accessType: formValues.accessType,
            tableName: formValues.tableName,
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
      title: $t('page.dataAccessAuditLog.createdAt'),
      field: 'createdAt',
      formatter: 'formatDateTime',
      width: 140,
    },
    {
      title: $t('page.dataAccessAuditLog.success'),
      field: 'success',
      slots: { default: 'success' },
      width: 80,
    },
    {
      title: $t('page.dataAccessAuditLog.accessType'),
      field: 'accessType',
      slots: { default: 'accessType' },
      width: 80,
    },
    { title: $t('page.dataAccessAuditLog.tableName'), field: 'tableName' },
    {
      title: $t('page.dataAccessAuditLog.dataCategory'),
      field: 'dataCategory',
    },
    { title: $t('page.dataAccessAuditLog.latencyMs'), field: 'latencyMs' },
    { title: $t('page.dataAccessAuditLog.username'), field: 'username' },
    {
      title: $t('page.dataAccessAuditLog.geoLocation'),
      field: 'geoLocation',
      slots: { default: 'geoLocation' },
    },
    {
      title: $t('page.dataAccessAuditLog.ipAddress'),
      field: 'ipAddress',
      width: 140,
    },
  ],
};

const [Grid] = useVbenVxeGrid({ gridOptions, formOptions });
</script>

<template>
  <Page auto-content-height>
    <Grid :table-title="$t('menu.log.dataAccessAuditLog')">
      <template #success="{ row }">
        <a-tag :color="successToColor(row.success)">
          {{ successToNameWithStatusCode(row.success, row.statusCode) }}
        </a-tag>
      </template>
      <template #geoLocation="{ row }">
        {{ row.geoLocation.province }} {{ row.geoLocation.city }}
      </template>
      <template #accessType="{ row }">
        <a-tag :color="dataAccessAuditLogAccessTypeToColor(row.accessType)">
          {{ dataAccessAuditLogAccessTypeToName(row.accessType) }}
        </a-tag>
      </template>
    </Grid>
  </Page>
</template>
