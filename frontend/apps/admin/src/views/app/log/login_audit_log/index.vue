<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { Page, type VbenFormProps } from '@vben/common-ui';

import dayjs from 'dayjs';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import { type auditservicev1_LoginAuditLog as LoginAuditLog } from '#/generated/api/admin/service/v1';
import { $t } from '#/locales';
import {
  getLoginAuditLogActionTypeColor,
  getLoginAuditLogRiskLevelColor,
  getLoginAuditLogStatusColor,
  loginAuditLogActiveTypeToName,
  loginAuditLogRiskLevelToName,
  loginAuditLogStatusToName,
  useLoginAuditLogStore,
} from '#/stores';

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
      label: $t('page.loginAuditLog.username'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'RangePicker',
      fieldName: 'createdAt',
      label: $t('page.loginAuditLog.createdAt'),
      componentProps: {
        showTime: true,
        allowClear: true,
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
            created_at__gte: startTime,
            created_at__lte: endTime,
          },
        );
      },
    },
  },

  columns: [
    {
      title: $t('page.loginAuditLog.createdAt'),
      field: 'createdAt',
      formatter: 'formatDateTime',
      width: 140,
    },
    { title: $t('page.loginAuditLog.username'), field: 'username' },
    {
      title: $t('page.loginAuditLog.status'),
      field: 'status',
      slots: { default: 'status' },
    },
    {
      title: $t('page.loginAuditLog.actionType'),
      field: 'actionType',
      slots: { default: 'actionType' },
    },
    {
      title: $t('page.loginAuditLog.riskLevel'),
      field: 'riskLevel',
      slots: { default: 'riskLevel' },
    },
    {
      title: $t('page.loginAuditLog.geoLocation'),
      field: 'geoLocation',
      slots: { default: 'geoLocation' },
    },
    {
      title: $t('page.loginAuditLog.platform'),
      field: 'deviceInfo.platform',
      slots: { default: 'platform' },
    },
    {
      title: $t('page.loginAuditLog.ipAddress'),
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
          {{ loginAuditLogActiveTypeToName(row.actionType) }}
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
