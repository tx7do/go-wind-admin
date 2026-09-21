import type {
  notificationservicev1_Channel,
  notificationservicev1_DeliveryStatus,
  notificationservicev1_EventType,
  notificationservicev1_ListNotificationDeliveryResponse,
} from '#/api/generated/admin/service/v1';
import type { PaginationQuery } from '#/transport/rest';

import { computed } from 'vue';

import { i18n } from '@vben/locales';

import { useQuery, type UseQueryOptions } from '@tanstack/vue-query';

import { apiClient } from '#/api/client';
import { queryClient } from '#/plugins/vue-query';

const t = i18n.global.t;

// ==============================
// 通知投递台账（平台级只读）
// 一行 = 一次投递事实；写侧只在进程内 SendDirect，本域没有增删改路由。
// ==============================

const LIST_KEY = 'listNotificationDeliveries';

export function useListNotificationDeliveries(
  query: PaginationQuery,
  options?: UseQueryOptions<
    notificationservicev1_ListNotificationDeliveryResponse,
    Error
  >,
) {
  return useQuery({
    queryKey: [LIST_KEY, query],
    queryFn: () =>
      apiClient.notificationService.ListNotificationDelivery(
        query.toRawParams(),
      ),
    ...options,
  });
}

export async function fetchListNotificationDeliveries(
  params: PaginationQuery,
) {
  return queryClient.fetchQuery({
    queryKey: [LIST_KEY, params],
    queryFn: () =>
      apiClient.notificationService.ListNotificationDelivery(
        params.toRawParams(),
      ),
    staleTime: 0,
    retry: 0,
  });
}

// ==============================
// 通知投递台账枚举与工具函数
// 台账里的 target 已由服务端脱敏，这里不做二次遮蔽；
// UNSPECIFIED 不入下拉（与 react 端一致），未命中值一律回落空串/default 色。
// ==============================

export const notificationDeliveryEventTypeList = computed(() => [
  {
    value: 'PASSWORD_RESET_CODE',
    label: t('enum.notificationDelivery.eventType.PASSWORD_RESET_CODE'),
  },
  {
    value: 'CONTACT_BIND_CODE',
    label: t('enum.notificationDelivery.eventType.CONTACT_BIND_CODE'),
  },
  {
    value: 'CHANNEL_TEST_EMAIL',
    label: t('enum.notificationDelivery.eventType.CHANNEL_TEST_EMAIL'),
  },
  {
    value: 'INTERNAL_MESSAGE',
    label: t('enum.notificationDelivery.eventType.INTERNAL_MESSAGE'),
  },
]);

export const notificationDeliveryChannelList = computed(() => [
  { value: 'EMAIL', label: t('enum.notificationDelivery.channel.EMAIL') },
  { value: 'SMS', label: t('enum.notificationDelivery.channel.SMS') },
  {
    value: 'WEBHOOK',
    label: t('enum.notificationDelivery.channel.WEBHOOK'),
  },
  {
    value: 'INTERNAL',
    label: t('enum.notificationDelivery.channel.INTERNAL'),
  },
]);

/**
 * 渠道筛选项 = 有投递实现、因此真能写出台账行的渠道。
 *
 * 路由不写在 Go 里：后端那份静态 eventChannels 表已经删了，「事件 → 渠道 + 是否异步」现在
 * 是数据库表 sys_notification_rules——启动时只按 pkg/constants.DefaultNotificationRules
 * 播一次空表，之后在「通知路由规则」页（views/app/notification/rule）维护，改完立刻生效。
 * 所以默认规则里没指向 WEBHOOK 不代表筛不到 WEBHOOK 行：管理员建一行规则它就来了。
 *
 * SMS 至今没有 Sender 实现，任何事件都路由不过去，永远不会有台账行，故不进下拉。
 */
export const notificationDeliveryChannelFilterList = computed(() =>
  notificationDeliveryChannelList.value.filter(
    (item) =>
      item.value === 'EMAIL' ||
      item.value === 'INTERNAL' ||
      item.value === 'WEBHOOK',
  ),
);

export const notificationDeliveryStatusList = computed(() => [
  { value: 'SENDING', label: t('enum.notificationDelivery.status.SENDING') },
  { value: 'SENT', label: t('enum.notificationDelivery.status.SENT') },
  { value: 'FAILED', label: t('enum.notificationDelivery.status.FAILED') },
  { value: 'SKIPPED', label: t('enum.notificationDelivery.status.SKIPPED') },
]);

// 状态色对应 react 端语义：SENDING=processing/blue、SENT=success/green、
// FAILED=error/red、SKIPPED=warning/orange（SKIPPED 是"没发出去过"，FAILED 才是"被退回"）。
type DeliveryTagColor =
  | 'blue'
  | 'cyan'
  | 'default'
  | 'green'
  | 'orange'
  | 'purple'
  | 'red';

const DELIVERY_STATUS_COLOR_MAP: Record<string, DeliveryTagColor> = {
  SENDING: 'blue',
  SENT: 'green',
  FAILED: 'red',
  SKIPPED: 'orange',
  DEFAULT: 'default',
};

const CHANNEL_COLOR_MAP: Record<string, DeliveryTagColor> = {
  EMAIL: 'blue',
  SMS: 'green',
  WEBHOOK: 'purple',
  INTERNAL: 'cyan',
  DEFAULT: 'default',
};

export function notificationDeliveryStatusToColor(
  status?: notificationservicev1_DeliveryStatus,
): DeliveryTagColor {
  return (status && DELIVERY_STATUS_COLOR_MAP[status]) || 'default';
}

export function notificationDeliveryChannelToColor(
  channel?: notificationservicev1_Channel,
): DeliveryTagColor {
  return (channel && CHANNEL_COLOR_MAP[channel]) || 'default';
}

export function notificationDeliveryStatusToName(
  status?: notificationservicev1_DeliveryStatus,
) {
  const matchedItem = notificationDeliveryStatusList.value.find(
    (item) => item.value === status,
  );
  return matchedItem ? matchedItem.label : '';
}

export function notificationDeliveryChannelToName(
  channel?: notificationservicev1_Channel,
) {
  const matchedItem = notificationDeliveryChannelList.value.find(
    (item) => item.value === channel,
  );
  return matchedItem ? matchedItem.label : '';
}

export function notificationDeliveryEventTypeToName(
  eventType?: notificationservicev1_EventType,
) {
  const matchedItem = notificationDeliveryEventTypeList.value.find(
    (item) => item.value === eventType,
  );
  return matchedItem ? matchedItem.label : '';
}
