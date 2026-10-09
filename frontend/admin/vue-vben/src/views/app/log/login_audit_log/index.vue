<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { h, ref } from 'vue';

import { Page, useVbenDrawer, type VbenFormProps } from '@vben/common-ui';
import { LucideEye } from '@vben/icons';

import dayjs from 'dayjs';

import { message } from 'ant-design-vue';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import { exportAuditLogsServer } from '#/api';
import {
  fetchListLoginAuditLogs,
  getLoginAuditLogActionTypeColor,
  getLoginAuditLogRiskLevelColor,
  getLoginAuditLogStatusColor,
  loginAuditLogActionTypeList,
  loginAuditLogActionTypeToName,
  loginAuditLogRiskLevelList,
  loginAuditLogRiskLevelToName,
  loginAuditLogStatusList,
  loginAuditLogStatusToName,
  PaginationQuery,
} from '#/api';
import { type auditservicev1_LoginAuditLog as LoginAuditLog } from '#/api';
import { $t } from '#/locales';
import { downloadTableFile } from '#/utils/csv';

import LoginAuditLogDetailDrawer from './login-audit-log-detail-drawer.vue';

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
      component: 'Input',
      fieldName: 'ipAddress',
      label: $t('page.loginAuditLog.ipAddress'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Select',
      fieldName: 'actionType',
      label: $t('page.loginAuditLog.actionType'),
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
      label: $t('page.loginAuditLog.riskLevel'),
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
      label: $t('page.loginAuditLog.status'),
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
      label: $t('page.loginAuditLog.createdAt'),
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
        }

        // 列表与导出共用同一份搜索条件：服务端导出透传的就是这里看到的
        const conditions = {
          username: formValues.username,
          ipAddress: formValues.ipAddress,
          status: formValues.status,
          actionType: formValues.actionType,
          riskType: formValues.riskType,
          created_at__gte: startTime,
          created_at__lte: endTime,
        };
        latestFormValuesRef.value = conditions;

        return await fetchListLoginAuditLogs(
          new PaginationQuery({
            paging: { page: page.currentPage, pageSize: page.pageSize },
            formValues: conditions,
            orderBy: ['-created_at'],
          }),
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
    {
      title: $t('page.loginAuditLog.status'),
      field: 'status',
      width: 80,
      slots: { default: 'status' },
    },
    { title: $t('page.loginAuditLog.username'), field: 'username' },
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
      title: $t('page.loginAuditLog.platform'),
      field: 'deviceInfo.platform',
      slots: { default: 'platform' },
    },
    {
      title: $t('page.loginAuditLog.geoLocation'),
      field: 'geoLocation',
      slots: { default: 'geoLocation' },
    },
    {
      title: $t('page.loginAuditLog.ipAddress'),
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

// 当前搜索条件（服务端导出透传用，与 Grid proxy 查询同源——"导出的就是当前搜索看到的"）
const latestFormValuesRef = ref<Record<string, unknown>>({});

const [Grid] = useVbenVxeGrid({ gridOptions, formOptions });

const [Drawer, drawerApi] = useVbenDrawer({
  connectedComponent: LoginAuditLogDetailDrawer,
});

function handleView(row: any) {
  drawerApi.setData({ row });
  drawerApi.open();
}

// 导出全部：按当前条件聚合拉取（上限 1 万行）生成 CSV
async function handleServerExport() {
  try {
    await exportAuditLogsServer(
      'login',
      new PaginationQuery({ formValues: latestFormValuesRef.value }),
    );
    message.success($t('page.auditExport.serverSuccess'));
  } catch (error: any) {
    // 原始错误必须留在控制台：用户可见的那句翻译不包含服务端的原因
    console.error('[audit-export] server-side export failed', error);
    message.error(error?.message || $t('page.auditExport.serverFailed'));
  }
}

async function handleExportAll(info: { key: string | number }) {
  const format = String(info.key);
  if (format === 'server-xlsx') {
    await handleServerExport();
    return;
  }
  if (format !== 'csv' && format !== 'xlsx') return;
  const pageSize = 1000;
  const maxRows = 10000;
  const rows: any[] = [];
  for (let p = 1; rows.length < maxRows; p++) {
    const resp = await fetchListLoginAuditLogs(
      new PaginationQuery({ paging: { page: p, pageSize } }),
    );
    const items = (resp.items ?? []) as any[];
    rows.push(...items);
    if (items.length < pageSize) break;
  }
  const gridColumns = (gridOptions.columns ?? []) as any[];
  const cols = gridColumns
    .filter((c) => c.field)
    .map((c) => ({ title: c.title as string, key: c.field as string }));
  await downloadTableFile(format, `login-audit-logs-${Date.now()}`, cols, rows.slice(0, maxRows));
}

</script>

<template>
  <Page auto-content-height>
    <Grid :table-title="$t('menu.log.loginAuditLog')">
      <template #toolbar-tools>
        <a-dropdown class="mr-2">
          <a-button>{{ $t('ui.button.exportAll') }}</a-button>
          <template #overlay>
            <a-menu @click="handleExportAll">
              <a-menu-item key="csv">{{ $t('ui.button.exportFormatCsv') }}</a-menu-item>
              <a-menu-item key="xlsx">{{ $t('ui.button.exportFormatXlsx') }}</a-menu-item>
              <a-menu-divider />
              <a-menu-item key="server-xlsx">{{ $t('page.auditExport.serverFull') }}</a-menu-item>
            </a-menu>
          </template>
        </a-dropdown>
      </template>
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
        {{ row.geoLocation?.province }} {{ row.geoLocation?.city }}
      </template>
      <template #platform="{ row }">
        {{ row.deviceInfo?.osName }} {{ row.deviceInfo?.browserName }}
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
