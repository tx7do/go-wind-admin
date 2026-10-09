import type {
  notificationservicev1_Channel,
  notificationservicev1_CreateNotificationRuleRequest,
  notificationservicev1_DeleteNotificationRuleRequest,
  notificationservicev1_EventType,
  notificationservicev1_ListNotificationRuleResponse,
  notificationservicev1_NotificationRule,
  notificationservicev1_TestDispatchNotificationRequest,
  notificationservicev1_TestDispatchNotificationResponse,
  notificationservicev1_UpdateNotificationRuleRequest,
} from '#/api/generated/admin/service/v1';
import type { PaginationQuery } from '#/transport/rest';

import { computed } from 'vue';

import { i18n } from '@vben/locales';

import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from '@tanstack/vue-query';

import { apiClient } from '#/api/client';
import { queryClient } from '#/plugins/vue-query';

const t = i18n.global.t;

// ==============================
// 通知路由规则（业务事件 → 投递渠道 + 派发方式）
//
// 这张表是投递时的唯一真相：改完这一行，下一次投递立刻按新路由走（服务端不缓存，
// 见后端 notification_rule_repo.go 的 GetByEventType）。路由曾是一张 Go 静态常量表，
// 现在它落在 sys_notification_rules，启动时仅按 DefaultNotificationRules 播一次空表。
// ==============================

const LIST_KEY = 'listNotificationRules';

/**
 * 更新掩码固定这四列：event_type 刻意不进掩码——改事件类型等于换一条路由，
 * 删旧建新才说得清（所以编辑态里那一栏也是禁用的）。
 */
export const NOTIFICATION_RULE_UPDATE_MASK = 'channel,isAsync,isEnabled,remark';

export function useListNotificationRules(
  query: PaginationQuery,
  options?: UseQueryOptions<
    notificationservicev1_ListNotificationRuleResponse,
    Error
  >,
) {
  return useQuery({
    queryKey: [LIST_KEY, query],
    queryFn: () =>
      apiClient.notificationRuleService.ListNotificationRule(
        query.toRawParams(),
      ),
    ...options,
  });
}

export async function fetchListNotificationRules(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: [LIST_KEY, params],
    queryFn: () =>
      apiClient.notificationRuleService.ListNotificationRule(
        params.toRawParams(),
      ),
    staleTime: 0,
    retry: 0,
  });
}

export function useCreateNotificationRule(
  options?: UseMutationOptions<
    notificationservicev1_NotificationRule,
    Error,
    notificationservicev1_CreateNotificationRuleRequest
  >,
) {
  return useMutation({
    mutationFn: (req: notificationservicev1_CreateNotificationRuleRequest) =>
      apiClient.notificationRuleService.CreateNotificationRule(req),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [LIST_KEY] });
    },
    ...options,
  });
}

/**
 * 更新只带固定掩码（NOTIFICATION_RULE_UPDATE_MASK），不接受"按变化字段生成"：
 * 掩码里少了哪一列，那一列就悄悄改不动，而规则页的表单本来就一次提交整行。
 */
export function useUpdateNotificationRule(
  options?: UseMutationOptions<
    {},
    Error,
    notificationservicev1_UpdateNotificationRuleRequest
  >,
) {
  return useMutation({
    mutationFn: (req: notificationservicev1_UpdateNotificationRuleRequest) =>
      apiClient.notificationRuleService.UpdateNotificationRule(req),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [LIST_KEY] });
    },
    ...options,
  });
}

export function useDeleteNotificationRule(
  options?: UseMutationOptions<
    {},
    Error,
    notificationservicev1_DeleteNotificationRuleRequest
  >,
) {
  return useMutation({
    mutationFn: (req: notificationservicev1_DeleteNotificationRuleRequest) =>
      apiClient.notificationRuleService.DeleteNotificationRule(req),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [LIST_KEY] });
    },
    ...options,
  });
}

/**
 * 测试投递：按这条规则当场走一遍完整投递链（路由、渠道、台账全照旧）。
 *
 * 请求体是**扁平**的 { id, target }：这个 RPC 在 proto 里是 body: "*" 的自定义路由，
 * 不是 CRUD 的 { data: {...} } 形态。多包一层 data 会被 protojson 当未知字段丢掉，
 * 接口照样 200、字段全为空——于是"id 必填"在服务端报出来，看着像页面没传 ID。
 *
 * 异步规则回的是 SENDING 而不是 SENT：这次只是「已交给队列」，结论落在队列那一趟，
 * 页面上要按台账状态如实显示（见调用处的 testDispatchQueued / testDispatchSent 分支）。
 */
export function useTestDispatchNotification(
  options?: UseMutationOptions<
    notificationservicev1_TestDispatchNotificationResponse,
    Error,
    notificationservicev1_TestDispatchNotificationRequest
  >,
) {
  return useMutation({
    mutationFn: (req: notificationservicev1_TestDispatchNotificationRequest) =>
      apiClient.notificationRuleService.TestDispatchNotification(req),
    ...options,
  });
}

// ==============================
// 通知路由规则枚举与工具函数
// UNSPECIFIED 不入下拉（与投递台账/react 端一致），未命中值一律回落空串/default 色。
// ==============================

export const notificationRuleEventTypeList = computed(() => [
  {
    value: 'PASSWORD_RESET_CODE',
    label: t('enum.notificationRule.eventType.PASSWORD_RESET_CODE'),
  },
  {
    value: 'CONTACT_BIND_CODE',
    label: t('enum.notificationRule.eventType.CONTACT_BIND_CODE'),
  },
  {
    value: 'CHANNEL_TEST_EMAIL',
    label: t('enum.notificationRule.eventType.CHANNEL_TEST_EMAIL'),
  },
  {
    value: 'INTERNAL_MESSAGE',
    label: t('enum.notificationRule.eventType.INTERNAL_MESSAGE'),
  },
]);

/** 全量渠道：列渲染按 record.channel 查这张表，未配实现的 SMS 也要答得出名字。 */
export const notificationRuleChannelList = computed(() => [
  { value: 'EMAIL', label: t('enum.notificationRule.channel.EMAIL') },
  { value: 'SMS', label: t('enum.notificationRule.channel.SMS') },
  { value: 'WEBHOOK', label: t('enum.notificationRule.channel.WEBHOOK') },
  { value: 'INTERNAL', label: t('enum.notificationRule.channel.INTERNAL') },
]);

// 可新建的渠道：SMS 还没有投递实现，给个选了也发不出去的选项只会把人引向
// 「配好了怎么没收到」，所以表单选项与搜索框都只出这三个（与 react 端 channelOptions 一致）。
const RULE_OPTION_CHANNELS = ['EMAIL', 'INTERNAL', 'WEBHOOK'];

export const notificationRuleChannelOptionList = computed(() =>
  notificationRuleChannelList.value.filter((item) =>
    RULE_OPTION_CHANNELS.includes(item.value as string),
  ),
);

export function notificationRuleEventTypeToName(
  eventType?: notificationservicev1_EventType,
) {
  const matchedItem = notificationRuleEventTypeList.value.find(
    (item) => item.value === eventType,
  );
  return matchedItem ? matchedItem.label : '';
}

export function notificationRuleChannelToName(
  channel?: notificationservicev1_Channel,
) {
  const matchedItem = notificationRuleChannelList.value.find(
    (item) => item.value === channel,
  );
  return matchedItem ? matchedItem.label : '';
}

// 颜色与文案对应 react 端语义：异步=geekblue（"还没定案"）、同步=default；
// 启用=success、停用=default（停用不是错误，是让那个事件报"没有启用路由规则"）。
type RuleTagColor =
  | 'blue'
  | 'cyan'
  | 'default'
  | 'geekblue'
  | 'green'
  | 'purple'
  | 'success';

const RULE_CHANNEL_COLOR_MAP: Record<string, RuleTagColor> = {
  EMAIL: 'blue',
  SMS: 'green',
  WEBHOOK: 'purple',
  INTERNAL: 'cyan',
  DEFAULT: 'default',
};

export function notificationRuleChannelToColor(
  channel?: notificationservicev1_Channel,
): RuleTagColor {
  return (channel && RULE_CHANNEL_COLOR_MAP[channel]) || 'default';
}

export function notificationRuleAsyncToName(isAsync?: boolean) {
  return isAsync
    ? t('enum.notificationRule.asyncMode.ASYNC')
    : t('enum.notificationRule.asyncMode.SYNC');
}

export function notificationRuleAsyncToColor(isAsync?: boolean): RuleTagColor {
  return isAsync ? 'geekblue' : 'default';
}

export function notificationRuleEnabledToName(isEnabled?: boolean) {
  return isEnabled
    ? t('enum.notificationRule.enabled.ON')
    : t('enum.notificationRule.enabled.OFF');
}

export function notificationRuleEnabledToColor(
  isEnabled?: boolean,
): RuleTagColor {
  return isEnabled ? 'success' : 'default';
}
