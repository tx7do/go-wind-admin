<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { Page, type VbenFormProps } from '@vben/common-ui';

import dayjs from 'dayjs';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import { type ApiAuditLog } from '#/generated/api/admin/service/v1';
import { $t } from '#/locales';
import {
  methodList,
  successToColor,
  successToNameWithStatusCode,
  useApiAuditLogStore,
} from '#/stores';

const props = defineProps({
  userId: { type: Number, default: undefined },
});

const apiAuditLogStore = useApiAuditLogStore();

const formOptions: VbenFormProps = {
  // 默认展开
  collapsed: false,
  // 控制表单是否显示折叠按钮
  showCollapseButton: false,
  // 按下回车时是否提交表单
  submitOnEnter: true,
  schema: [
    {
      component: 'RangePicker',
      fieldName: 'createdAt',
      label: $t('page.apiAuditLog.createdAt'),
      componentProps: {
        showTime: true,
        allowClear: true,
      },
    },
    {
      component: 'Select',
      fieldName: 'method',
      label: $t('page.apiAuditLog.method'),
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
      fieldName: 'path',
      label: $t('page.apiAuditLog.path'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
  ],
};

const gridOptions: VxeGridProps<ApiAuditLog> = {
  stripe: true,
  height: 'auto',
  exportConfig: {},
  pagerConfig: {},
  rowConfig: {
    isHover: true,
  },

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

        return await apiAuditLogStore.listApiAuditLog(
          {
            page: page.currentPage,
            pageSize: page.pageSize,
          },
          {
            user_id: props.userId?.toString(),
            username: formValues.username,
            method: formValues.method,
            path: formValues.path,
            created_at__gte: startTime,
            created_at__lte: endTime,
          },
        );
      },
    },
  },

  columns: [
    { title: $t('ui.table.seq'), type: 'seq', width: 50 },
    { title: $t('page.apiAuditLog.username'), field: 'username' },
    {
      title: $t('page.apiAuditLog.success'),
      field: 'success',
      slots: { default: 'success' },
      width: 80,
    },
    {
      title: $t('page.apiAuditLog.createdAt'),
      field: 'createdAt',
      formatter: 'formatDateTime',
      width: 140,
    },
    {
      title: $t('page.apiAuditLog.method'),
      field: 'method',
      width: 80,
    },
    { title: $t('page.apiAuditLog.path'), field: 'path' },
    { title: $t('page.apiAuditLog.location'), field: 'location' },
    {
      title: $t('page.apiAuditLog.clientName'),
      field: 'clientName',
      slots: { default: 'platform' },
    },
    {
      title: $t('page.apiAuditLog.clientIp'),
      field: 'clientIp',
      width: 140,
    },
  ],
};

const [Grid] = useVbenVxeGrid({ gridOptions, formOptions });
</script>

<template>
  <Page auto-content-height>
    <Grid>
      <template #success="{ row }">
        <a-tag :color="successToColor(row.success)">
          {{ successToNameWithStatusCode(row.success, row.statusCode) }}
        </a-tag>
      </template>
      <template #platform="{ row }">
        <span> {{ row.osName }} {{ row.browserName }}</span>
      </template>
    </Grid>
  </Page>
</template>
