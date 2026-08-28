<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { Page, useVbenDrawer, type VbenFormProps } from '@vben/common-ui';
import { LucideEye } from '@vben/icons';

import dayjs from 'dayjs';

import { h } from 'vue';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import {
  fetchListPolicyEvaluationLogs,
  methodList,
  PaginationQuery,
  successStatusList,
  successToColor,
  successToName,
} from '#/api';
import { type permissionservicev1_PolicyEvaluationLog as PolicyEvaluationLog } from '#/api';
import { $t } from '#/locales';

import PolicyEvaluationLogDetailDrawer from './policy-evaluation-log-detail-drawer.vue';

const formOptions: VbenFormProps = {
  // 默认展开
  collapsed: false,
  // 控制表单是否显示折叠按钮
  showCollapseButton: false,
  // 按下回车时是否提交表单
  submitOnEnter: true,
  schema: [
    {
      component: 'Select',
      fieldName: 'requestMethod',
      label: $t('page.policyEvaluationLog.requestMethod'),
      componentProps: {
        options: methodList,
        placeholder: $t('ui.placeholder.select'),
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        allowClear: true,
        showSearch: true,
      },
    },
    {
      component: 'Input',
      fieldName: 'requestPath',
      label: $t('page.policyEvaluationLog.requestPath'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Select',
      fieldName: 'result',
      label: $t('page.policyEvaluationLog.result'),
      componentProps: {
        options: successStatusList,
        placeholder: $t('ui.placeholder.select'),
        allowClear: true,
      },
    },
    {
      component: 'Input',
      fieldName: 'userId',
      label: $t('page.policyEvaluationLog.userId'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Input',
      fieldName: 'ipAddress',
      label: $t('page.policyEvaluationLog.ipAddress'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'RangePicker',
      fieldName: 'createdAt',
      label: $t('page.policyEvaluationLog.createdAt'),
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

const gridOptions: VxeGridProps<PolicyEvaluationLog> = {
  toolbarConfig: {
    custom: true,
    export: true,
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
        }

        return await fetchListPolicyEvaluationLogs(
          new PaginationQuery({
            paging: { page: page.currentPage, pageSize: page.pageSize },
            formValues: {
              requestMethod: formValues.requestMethod,
              requestPath: formValues.requestPath,
              result: formValues.result,
              userId: formValues.userId,
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
      title: $t('page.policyEvaluationLog.createdAt'),
      field: 'createdAt',
      formatter: 'formatDateTime',
      width: 140,
    },
    {
      title: $t('page.policyEvaluationLog.result'),
      field: 'result',
      slots: { default: 'result' },
      width: 90,
    },
    {
      title: $t('page.policyEvaluationLog.policyId'),
      field: 'policyId',
      width: 100,
    },
    {
      title: $t('page.policyEvaluationLog.permissionId'),
      field: 'permissionId',
      width: 120,
    },
    {
      title: $t('page.policyEvaluationLog.userId'),
      field: 'userId',
      width: 100,
    },
    {
      title: $t('page.policyEvaluationLog.membershipId'),
      field: 'membershipId',
      width: 120,
    },
    {
      title: $t('page.policyEvaluationLog.requestMethod'),
      field: 'requestMethod',
      width: 110,
    },
    {
      title: $t('page.policyEvaluationLog.requestPath'),
      field: 'requestPath',
      minWidth: 200,
    },
    {
      title: $t('page.policyEvaluationLog.tenantId'),
      field: 'tenantId',
      width: 100,
    },
    {
      title: $t('page.policyEvaluationLog.ipAddress'),
      field: 'ipAddress',
      width: 140,
    },
    {
      title: $t('page.policyEvaluationLog.traceId'),
      field: 'traceId',
      width: 200,
    },
    {
      title: $t('page.policyEvaluationLog.effectDetails'),
      field: 'effectDetails',
      minWidth: 200,
    },
    {
      title: $t('page.policyEvaluationLog.evaluationContext'),
      field: 'evaluationContext',
      minWidth: 200,
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
  connectedComponent: PolicyEvaluationLogDetailDrawer,
});

function handleView(row: PolicyEvaluationLog) {
  drawerApi.setData({ row });
  drawerApi.open();
}
</script>

<template>
  <Page auto-content-height>
    <Grid :table-title="$t('menu.log.policyEvaluationLog')">
      <template #result="{ row }">
        <a-tag :color="successToColor(row.result)">
          {{ successToName(row.result) }}
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
