<script lang="ts" setup>
import type { VxeGridProps } from '@/adapter/vxe-table';

import { Page, type VbenFormProps } from '@vben/common-ui';

import dayjs from 'dayjs';

import { useVbenVxeGrid } from '@/adapter/vxe-table';
import { type auditservicev1_LoginAuditLog as LoginAuditLog } from '@/api/generated/admin/service/v1';
import { $t } from '@/locales';
import {
  getLoginAuditLogActionTypeColor,
  getLoginAuditLogRiskLevelColor,
  getLoginAuditLogStatusColor,
  loginAuditLogActionTypeList,
  loginAuditLogActionTypeToName,
  loginAuditLogRiskLevelList,
  loginAuditLogRiskLevelToName,
  loginAuditLogStatusList,
  loginAuditLogStatusToName,
  useLoginAuditLogStore,
} from '@/stores';

const loginAuditLogStore = useLoginAuditLogStore();

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
      label: t('pages.loginAuditLog.username'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Input',
      fieldName: 'ipAddress',
      label: t('pages.loginAuditLog.ipAddress'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Select',
      fieldName: 'actionType',
      label: t('pages.loginAuditLog.actionType'),
      componentProps: {
        options: loginAuditLogActionTypeList,
        placeholder: $t('ui.placeholder.select'),
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        allowClear: true,
        showSearch: true,
      },
    },
    {
      component: 'Select',
      fieldName: 'riskLevel',
      label: t('pages.loginAuditLog.riskLevel'),
      componentProps: {
        options: loginAuditLogRiskLevelList,
        placeholder: $t('ui.placeholder.select'),
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        allowClear: true,
        showSearch: true,
      },
    },
    {
      component: 'Select',
      fieldName: 'status',
      label: t('pages.loginAuditLog.status'),
      componentProps: {
        options: loginAuditLogStatusList,
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
      label: t('pages.loginAuditLog.createdAt'),
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

const gridOptions: VxeGridProps<LoginAuditLog> = {
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
          formValues.loginTime !== undefined &&
          formValues.loginTime.length === 2
        ) {
          startTime = dayjs(formValues.loginTime[0]).format(
            'YYYY-MM-DD HH:mm:ss',
          );
          endTime = dayjs(formValues.loginTime[1]).format(
            'YYYY-MM-DD HH:mm:ss',
          );
          console.log(startTime, endTime);
        }

        return await loginAuditLogStore.listLoginAuditLog(
          {
            page: page.currentPage,
            pageSize: page.pageSize,
          },
          {
            username: formValues.username,
            ipAddress: formValues.ipAddress,
            status: formValues.status,
            actionType: formValues.actionType,
            riskType: formValues.riskType,
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
      title: t('pages.loginAuditLog.createdAt'),
      field: 'createdAt',
      formatter: 'formatDateTime',
      width: 140,
    },
    {
      title: t('pages.loginAuditLog.status'),
      field: 'status',
      width: 80,
      slots: { default: 'status' },
    },
    { title: t('pages.loginAuditLog.username'), field: 'username' },
    {
      title: t('pages.loginAuditLog.actionType'),
      field: 'actionType',
      slots: { default: 'actionType' },
    },
    {
      title: t('pages.loginAuditLog.riskLevel'),
      field: 'riskLevel',
      slots: { default: 'riskLevel' },
    },
    {
      title: t('pages.loginAuditLog.platform'),
      field: 'deviceInfo.platform',
      slots: { default: 'platform' },
    },
    {
      title: t('pages.loginAuditLog.geoLocation'),
      field: 'geoLocation',
      slots: { default: 'geoLocation' },
    },
    {
      title: t('pages.loginAuditLog.ipAddress'),
      field: 'ipAddress',
      width: 140,
    },
  ],
};

const [Grid] = useVbenVxeGrid({ gridOptions, formOptions });
</script>

<template>
  <Page auto-content-height>
    <Grid :table-title="$t('menu.log.loginAuditLog')">
      <template #status="{ row }">
        <a-tag :color="getLoginAuditLogStatusColor(row.status)">
          {{ loginAuditLogStatusToName(row.status) }}
        </a-tag>
      </template>
      <template #actionType="{ row }">
        <a-tag :color="getLoginAuditLogActionTypeColor(row.actionType)">
          {{ loginAuditLogActionTypeToName(row.actionType) }}
        </a-tag>
      </template>
      <template #riskLevel="{ row }">
        <a-tag :color="getLoginAuditLogRiskLevelColor(row.riskLevel)">
          {{ loginAuditLogRiskLevelToName(row.riskLevel) }}
        </a-tag>
      </template>
      <template #geoLocation="{ row }">
        {{ row.geoLocation.province }} {{ row.geoLocation.city }}
      </template>
      <template #platform="{ row }">
        {{ row.deviceInfo.osName }} {{ row.deviceInfo.browserName }}
      </template>
    </Grid>
  </Page>
</template>
