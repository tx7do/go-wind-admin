<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { Page, type VbenFormProps } from '@vben/common-ui';

import { notification } from 'ant-design-vue';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import {
  fetchListNotificationDeliveries,
  notificationDeliveryChannelFilterList,
  notificationDeliveryChannelToColor,
  notificationDeliveryChannelToName,
  notificationDeliveryEventTypeList,
  notificationDeliveryEventTypeToName,
  notificationDeliveryStatusList,
  notificationDeliveryStatusToColor,
  notificationDeliveryStatusToName,
  PaginationQuery,
} from '#/api';
import type {
  notificationservicev1_NotificationDelivery as NotificationDelivery,
} from '#/api';
import { $t } from '#/locales';

/**
 * 通知投递台账（只读）
 *
 * 一行 = 一次投递事实，写侧在 NotificationService.SendDirect（找回密码验证码、联系人绑定码、
 * 渠道测试邮件、站内信定向投递），本域没有增删改路由，所以页面只有查询与排序。
 * status 的语义差别值得看清：SKIPPED 是"压根没发出去过"（渠道没配/没启用），
 * FAILED 才是"发了但被退回"。
 */

// vxe 列名 → 后端 orderBy 的蛇形字段（只有时间列参与排序）
const SORT_FIELD_MAP: Record<string, string> = {
  createdAt: 'created_at',
};

const formOptions: VbenFormProps = {
  collapsed: false,
  showCollapseButton: false,
  submitOnEnter: true,
  schema: [
    {
      component: 'Select',
      fieldName: 'eventType',
      label: $t('page.notificationDelivery.eventType'),
      componentProps: {
        options: notificationDeliveryEventTypeList,
        placeholder: $t('ui.placeholder.select'),
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        allowClear: true,
        showSearch: true,
      },
    },
    {
      component: 'Select',
      fieldName: 'channel',
      label: $t('page.notificationDelivery.channel'),
      componentProps: {
        options: notificationDeliveryChannelFilterList,
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
      label: $t('page.notificationDelivery.status'),
      componentProps: {
        options: notificationDeliveryStatusList,
        placeholder: $t('ui.placeholder.select'),
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        allowClear: true,
        showSearch: true,
      },
    },
    {
      component: 'Input',
      fieldName: 'target',
      label: $t('page.notificationDelivery.target'),
      // 台账存的就是脱敏串：搜索框输入完整地址同样只会命中这一形态
      help: $t('page.notificationDelivery.targetMaskedHint'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
  ],
};

const gridOptions: VxeGridProps<NotificationDelivery> = {
  toolbarConfig: {
    custom: true,
    refresh: true,
    zoom: true,
  },
  height: 'auto',
  pagerConfig: {},
  rowConfig: {
    isHover: true,
    keyField: 'id',
  },
  stripe: true,
  // 服务端排序：点表头即重查，默认按投递时间倒序（与 react 端 -created_at 一致）
  sortConfig: {
    defaultSort: { field: 'createdAt', order: 'desc' },
    remote: true,
  },

  proxyConfig: {
    ajax: {
      query: async ({ page, sorts }, formValues) => {
        const values = (formValues ?? {}) as Record<string, any>;
        // 排序：只传裸字段名给后端，前缀 '-' 表示倒序
        const sort = sorts?.[0];
        const sortField = sort?.field
          ? (SORT_FIELD_MAP[sort.field] ?? sort.field)
          : undefined;
        let orderBy = ['-created_at'];
        if (sortField && (sort?.order === 'asc' || sort?.order === 'desc')) {
          orderBy = [sort.order === 'asc' ? sortField : `-${sortField}`];
        }

        try {
          return await fetchListNotificationDeliveries(
            new PaginationQuery({
              paging: { page: page.currentPage, pageSize: page.pageSize },
              // 一律传裸字段名，模糊算子由 PaginationQuery 统一追加；
              // ID 列（recipientUserId / channelId）不进搜索表单
              formValues: {
                channel: values.channel,
                eventType: values.eventType,
                status: values.status,
                target: values.target,
              },
              orderBy,
            }),
          );
        } catch (error: any) {
          // 不吞错：原始错误对象进控制台，同时给用户可读文案
          console.error('[notification-delivery] list failed:', error);
          notification.error({
            message: error?.message || $t('page.notificationDelivery.fetchFailed'),
          });
          return { items: [], total: 0 };
        }
      },
    },
  },

  columns: [
    {
      title: $t('page.notificationDelivery.createdAt'),
      field: 'createdAt',
      formatter: 'formatDateTime',
      sortable: true,
      width: 170,
    },
    {
      title: $t('page.notificationDelivery.eventType'),
      field: 'eventType',
      slots: { default: 'eventType' },
      width: 130,
    },
    {
      title: $t('page.notificationDelivery.channel'),
      field: 'channel',
      slots: { default: 'channel' },
      width: 100,
    },
    {
      title: $t('page.notificationDelivery.target'),
      field: 'target',
      minWidth: 180,
      showOverflow: 'tooltip',
    },
    {
      title: $t('page.notificationDelivery.status'),
      field: 'status',
      slots: { default: 'status' },
      width: 110,
    },
    {
      title: $t('page.notificationDelivery.recipientUserId'),
      field: 'recipientUserId',
      width: 110,
      formatter: ({ cellValue }) => cellValue ?? '-',
    },
    {
      title: $t('page.notificationDelivery.channelId'),
      field: 'channelId',
      width: 100,
      formatter: ({ cellValue }) => cellValue ?? '-',
    },
    {
      // 台账不存正文快照，related_id 是"这条投递发的是什么"的唯一回跳入口（站内信 = 消息 ID）
      title: $t('page.notificationDelivery.relatedId'),
      field: 'relatedId',
      width: 110,
      formatter: ({ cellValue }) => cellValue ?? '-',
    },
    {
      title: $t('page.notificationDelivery.sentAt'),
      field: 'sentAt',
      formatter: 'formatDateTime',
      width: 170,
    },
    {
      title: $t('page.notificationDelivery.lastError'),
      field: 'lastError',
      minWidth: 260,
      showOverflow: 'tooltip',
    },
  ],
};

const [Grid] = useVbenVxeGrid({ gridOptions, formOptions });
</script>

<template>
  <Page auto-content-height>
    <Grid :table-title="$t('page.notificationDelivery.moduleName')">
      <template #eventType="{ row }">
        {{ notificationDeliveryEventTypeToName(row.eventType) || '-' }}
      </template>
      <template #channel="{ row }">
        <a-tag
          v-if="row.channel"
          :color="notificationDeliveryChannelToColor(row.channel)"
        >
          {{ notificationDeliveryChannelToName(row.channel) }}
        </a-tag>
        <template v-else>-</template>
      </template>
      <template #status="{ row }">
        <a-tag
          v-if="row.status"
          :color="notificationDeliveryStatusToColor(row.status)"
        >
          {{ notificationDeliveryStatusToName(row.status) }}
        </a-tag>
        <template v-else>-</template>
      </template>
    </Grid>
  </Page>
</template>
