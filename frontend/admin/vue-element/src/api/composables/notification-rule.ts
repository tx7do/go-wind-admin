import { computed } from "vue";
import type { TagType } from "./shared";
import { useMutation, useQuery, type UseMutationOptions, type UseQueryOptions } from "@tanstack/vue-query";
import type {
  notificationservicev1_Channel,
  notificationservicev1_CreateNotificationRuleRequest,
  notificationservicev1_DeleteNotificationRuleRequest,
  notificationservicev1_EventType,
  notificationservicev1_ListNotificationRuleResponse,
  notificationservicev1_NotificationRule,
  notificationservicev1_TestDispatchNotificationRequest,
  notificationservicev1_TestDispatchNotificationResponse,
} from "@/api/generated/admin/service/v1";
import type { PaginationQuery } from "@/core/transport/rest";
import { apiClient } from "@/api/client";
import { queryClient } from "@/plugins/vue-query";
import { i18n } from "@/core/i18n";

const t = i18n.global.t;

// ==============================
// 通知路由规则（事件类型 → 渠道 + 派发方式）
// ==============================
//
// 这张表是投递时的唯一真相：改完这一行，下一次投递立刻按新路由走（服务端不缓存）。
// 两个决定来自同一行，所以「渠道」与「异步」必须一起看：异步 = 请求入队即回，
// 结论由队列回写台账；同步 = 当场拨号，接口的回话就是投递结论。
//
// 删掉一行 = 「这个事件不再通知」：服务端不会在下次启动补回来，但那条事件的调用
// 会拿到「没有启用路由规则」的报错（不静默丢通知）。

const LIST_KEY = "listNotificationRules";

/**
 * 更新掩码固定列举业务字段：event_type 刻意不进掩码——改事件类型等于换一条路由，
 * 删旧建新才说得清（与 react 基准一致）。
 */
const UPDATE_MASK = "channel,isAsync,isEnabled,remark";

/** 分页查询路由规则（组件 setup 中使用） */
export function useListNotificationRules(
  query: PaginationQuery,
  options?: UseQueryOptions<notificationservicev1_ListNotificationRuleResponse, Error>
) {
  return useQuery({
    queryKey: [LIST_KEY, query],
    queryFn: () => apiClient.notificationRuleService.ListNotificationRule(query.toRawParams()),
    ...options,
  });
}

/** 分页查询路由规则（非 Hook，列表页 listAction / 导出等手动调用） */
export async function fetchListNotificationRules(
  params: PaginationQuery
): Promise<notificationservicev1_ListNotificationRuleResponse> {
  return queryClient.fetchQuery({
    queryKey: [LIST_KEY, params],
    queryFn: () => apiClient.notificationRuleService.ListNotificationRule(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

/** 创建路由规则：CRUD 请求体必须包 { data: {...} } */
export function useCreateNotificationRule(
  options?: UseMutationOptions<
    notificationservicev1_NotificationRule,
    Error,
    notificationservicev1_CreateNotificationRuleRequest
  >
) {
  return useMutation({
    mutationFn: (req) => apiClient.notificationRuleService.CreateNotificationRule(req),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [LIST_KEY] }),
    ...options,
  });
}

/** 更新路由规则：掩码在 composable 内固定（不含 eventType），调用方无从改坏它 */
export function useUpdateNotificationRule(
  options?: UseMutationOptions<
    {},
    Error,
    { id: number; data: notificationservicev1_NotificationRule }
  >
) {
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: notificationservicev1_NotificationRule }) =>
      apiClient.notificationRuleService.UpdateNotificationRule({
        id,
        data,
        updateMask: UPDATE_MASK,
      }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [LIST_KEY] }),
    ...options,
  });
}

/** 删除路由规则 */
export function useDeleteNotificationRule(
  options?: UseMutationOptions<{}, Error, notificationservicev1_DeleteNotificationRuleRequest>
) {
  return useMutation({
    mutationFn: (req) => apiClient.notificationRuleService.DeleteNotificationRule(req),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [LIST_KEY] }),
    ...options,
  });
}

/**
 * 测试投递：按这条规则当场走一遍完整投递链。
 *
 * 这是 body:"*" 的自定义 RPC，收**扁平**请求体 { id, target }：多包一层 { data } 会被
 * protojson 当未知字段丢掉，接口照样 200、字段全为空。
 *
 * 异步规则回的是 SENDING 而不是 SENT —— 结论落在队列那一趟，页面上要按台账状态如实显示，
 * 不能把它翻译成一句「发送成功」。
 */
export function useTestDispatchNotification(
  options?: UseMutationOptions<
    notificationservicev1_TestDispatchNotificationResponse,
    Error,
    notificationservicev1_TestDispatchNotificationRequest
  >
) {
  return useMutation({
    mutationFn: (req) => apiClient.notificationRuleService.TestDispatchNotification(req),
    ...options,
  });
}

// ==============================
// 规则枚举与工具函数
// ==============================

// 枚举值即 i18n key 末段，故展示文案按 `enum.notificationRule.<field>.<VALUE>` 取
// （与 notification-delivery.ts 同构，三端可逐条对译）。
const EVENT_TYPE_KEYS: notificationservicev1_EventType[] = [
  "PASSWORD_RESET_CODE",
  "CONTACT_BIND_CODE",
  "CHANNEL_TEST_EMAIL",
  "INTERNAL_MESSAGE",
];

/** 可路由的渠道：只有这三个真有投递实现，SMS 放进下拉只会误导「筛出来空表是我条件错了」 */
const ASSIGNABLE_CHANNEL_KEYS: notificationservicev1_Channel[] = ["EMAIL", "WEBHOOK", "INTERNAL"];

/** 展示全集：历史行可能出现任意渠道，展示层不留空 */
const CHANNEL_KEYS: notificationservicev1_Channel[] = ["EMAIL", "SMS", "WEBHOOK", "INTERNAL"];

/** 业务事件选项（搜索与表单共用，同一套全集） */
export const notificationRuleEventTypeList = computed(() =>
  EVENT_TYPE_KEYS.map((value) => ({
    value,
    label: t(`enum.notificationRule.eventType.${value}`),
  }))
);

/** 渠道选项（可赋值集：搜索条件与表单下拉都用它） */
export const notificationRuleChannelOptionList = computed(() =>
  ASSIGNABLE_CHANNEL_KEYS.map((value) => ({
    value,
    label: t(`enum.notificationRule.channel.${value}`),
  }))
);

/** 渠道展示全集 */
export const notificationRuleChannelList = computed(() =>
  CHANNEL_KEYS.map((value) => ({
    value,
    label: t(`enum.notificationRule.channel.${value}`),
  }))
);

/** 事件类型是纯文本列（不是标签），只取中文名；未知值原样回显，别读成空 */
export function notificationRuleEventTypeToName(eventType?: notificationservicev1_EventType) {
  if (!eventType) return "";
  const matched = notificationRuleEventTypeList.value.find((item) => item.value === eventType);
  return matched ? matched.label : eventType;
}

export function notificationRuleChannelToName(channel?: notificationservicev1_Channel) {
  if (!channel) return "";
  const matched = notificationRuleChannelList.value.find((item) => item.value === channel);
  return matched ? matched.label : channel;
}

/** 渠道色：Element Plus 无 purple/cyan 语义色，Webhook/站内信按 info/warning 归入现有色板 */
const NOTIFICATION_RULE_CHANNEL_TAG_TYPE_MAP: Record<string, TagType> = {
  EMAIL: "primary",
  SMS: "success",
  WEBHOOK: "info",
  INTERNAL: "warning",
  DEFAULT: "info",
};

export function notificationRuleChannelToType(channel: notificationservicev1_Channel): TagType {
  return (
    NOTIFICATION_RULE_CHANNEL_TAG_TYPE_MAP[channel as string] ||
    NOTIFICATION_RULE_CHANNEL_TAG_TYPE_MAP.DEFAULT
  );
}
