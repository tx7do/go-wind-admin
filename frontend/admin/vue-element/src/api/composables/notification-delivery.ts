import { computed } from "vue";
import type { TagType } from "./shared";
import type {
  notificationservicev1_Channel,
  notificationservicev1_DeliveryStatus,
  notificationservicev1_EventType,
  notificationservicev1_ListNotificationDeliveryResponse,
} from "@/api/generated/admin/service/v1";
import type { PaginationQuery } from "@/core/transport/rest";
import { apiClient } from "@/api/client";
import { queryClient } from "@/plugins/vue-query";
import { i18n } from "@/core/i18n";

const t = i18n.global.t;

// ==============================
// 通知投递台账（平台级只读）
// ==============================
//
// 一行 = 一次投递事实，写侧只在进程内的 NotificationService.SendDirect（找回密码验证码、
// 联系人绑定码、渠道测试邮件、站内信定向投递），本域不含创建/更新/删除路由，页面因此只有查询能力。

const LIST_KEY = "listNotificationDeliveries";

/** 分页查询投递台账 */
export async function fetchListNotificationDeliveries(
  params: PaginationQuery
): Promise<notificationservicev1_ListNotificationDeliveryResponse> {
  return queryClient.fetchQuery({
    queryKey: [LIST_KEY, params],
    queryFn: () => apiClient.notificationService.ListNotificationDelivery(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

// ==============================
// 台账枚举与工具函数
// ==============================

// 枚举值即 i18n key 末段，故展示文案按 `enum.notificationDelivery.<field>.<VALUE>` 取。
const EVENT_TYPE_KEYS: notificationservicev1_EventType[] = [
  "PASSWORD_RESET_CODE",
  "CONTACT_BIND_CODE",
  "CHANNEL_TEST_EMAIL",
  "INTERNAL_MESSAGE",
];
const CHANNEL_KEYS: notificationservicev1_Channel[] = ["EMAIL", "SMS", "WEBHOOK", "INTERNAL"];
const STATUS_KEYS: notificationservicev1_DeliveryStatus[] = [
  "SENDING",
  "SENT",
  "FAILED",
  "SKIPPED",
];

/** 业务事件筛选项（台账全事件集） */
export const notificationDeliveryEventTypeList = computed(() =>
  EVENT_TYPE_KEYS.map((value) => ({
    value,
    label: t(`enum.notificationDelivery.eventType.${value}`),
  }))
);

/** 渠道展示全集：历史行可能出现任意渠道，展示层不留空 */
export const notificationDeliveryChannelList = computed(() =>
  CHANNEL_KEYS.map((value) => ({
    value,
    label: t(`enum.notificationDelivery.channel.${value}`),
  }))
);

/**
 * 渠道筛选项 = 已注册实现的渠道：目前只有 EMAIL 与 INTERNAL 真的在发信（路由表
 * 见后端 notification_service.go 的 eventChannels），SMS/WEBHOOK 暂无数据行，
 * 放进下拉只会误导「筛出来空表是我的条件错了」，故只在展示全集里保留文案。
 */
const FILTERED_CHANNEL_KEYS: notificationservicev1_Channel[] = ["EMAIL", "INTERNAL"];

export const notificationDeliveryChannelFilterList = computed(
  () =>
    FILTERED_CHANNEL_KEYS.map((value) => ({
      value,
      label: t(`enum.notificationDelivery.channel.${value}`),
    }))
);

/** 投递状态筛选项 */
export const notificationDeliveryStatusList = computed(() =>
  STATUS_KEYS.map((value) => ({
    value,
    label: t(`enum.notificationDelivery.status.${value}`),
  }))
);

/**
 * 状态色：SENDING 进行中（蓝）/ SENT 成功（绿）/ FAILED 失败（红）/ SKIPPED 没发出去（橙）。
 * SKIPPED 是「压根没发出去过」（渠道没配/没启用），FAILED 才是「发了但被退回」，
 * 两者的语义差别靠颜色一眼分开。
 */
const NOTIFICATION_DELIVERY_STATUS_TAG_TYPE_MAP: Record<string, TagType> = {
  SENDING: "primary",
  SENT: "success",
  FAILED: "danger",
  SKIPPED: "warning",
  DEFAULT: "info",
};

export function notificationDeliveryStatusToType(status: notificationservicev1_DeliveryStatus) {
  return (
    NOTIFICATION_DELIVERY_STATUS_TAG_TYPE_MAP[status as string] ||
    NOTIFICATION_DELIVERY_STATUS_TAG_TYPE_MAP.DEFAULT
  );
}

export function notificationDeliveryStatusToName(status: notificationservicev1_DeliveryStatus) {
  const matched = notificationDeliveryStatusList.value.find((item) => item.value === status);
  return matched ? matched.label : "";
}

/** 渠道色：Element Plus 无 purple/cyan 语义色，Webhook/站内信按 info/warning 归入现有色板 */
const NOTIFICATION_DELIVERY_CHANNEL_TAG_TYPE_MAP: Record<string, TagType> = {
  EMAIL: "primary",
  SMS: "success",
  WEBHOOK: "info",
  INTERNAL: "warning",
  DEFAULT: "info",
};

export function notificationDeliveryChannelToType(channel: notificationservicev1_Channel) {
  return (
    NOTIFICATION_DELIVERY_CHANNEL_TAG_TYPE_MAP[channel as string] ||
    NOTIFICATION_DELIVERY_CHANNEL_TAG_TYPE_MAP.DEFAULT
  );
}

export function notificationDeliveryChannelToName(channel: notificationservicev1_Channel) {
  const matched = notificationDeliveryChannelList.value.find((item) => item.value === channel);
  return matched ? matched.label : "";
}

/** 事件类型是纯文本列（不是标签），只取中文名 */
export function notificationDeliveryEventTypeToName(eventType?: notificationservicev1_EventType) {
  if (!eventType) return "";
  const matched = notificationDeliveryEventTypeList.value.find((item) => item.value === eventType);
  return matched ? matched.label : eventType;
}
