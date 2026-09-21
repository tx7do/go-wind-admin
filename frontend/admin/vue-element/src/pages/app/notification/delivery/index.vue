<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig">
      <!-- 渠道 -->
      <template #channel="scope: any">
        <ElTag
          v-if="scope.row.channel"
          size="small"
          round
          :type="notificationDeliveryChannelToType(scope.row.channel)"
        >
          {{ notificationDeliveryChannelToName(scope.row.channel) }}
        </ElTag>
        <span v-else>-</span>
      </template>

      <!-- 投递结果 -->
      <template #status="scope: any">
        <ElTag
          v-if="scope.row.status"
          size="small"
          round
          :type="notificationDeliveryStatusToType(scope.row.status)"
        >
          {{ notificationDeliveryStatusToName(scope.row.status) }}
        </ElTag>
        <span v-else>-</span>
      </template>
    </ProPage>
  </div>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";
import { ElMessage, ElTag } from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import type { notificationservicev1_NotificationDelivery as NotificationDelivery } from "@/api/generated/admin/service/v1";
import {
  createPagedExportAction,
  fetchListNotificationDeliveries,
  notificationDeliveryChannelFilterList,
  notificationDeliveryChannelToName,
  notificationDeliveryChannelToType,
  notificationDeliveryEventTypeList,
  notificationDeliveryEventTypeToName,
  notificationDeliveryStatusList,
  notificationDeliveryStatusToName,
  notificationDeliveryStatusToType,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/core/i18n";

/**
 * 通知投递台账（只读）
 *
 * 一行 = 一次投递事实：写侧只在进程内的 NotificationService.SendDirect（找回密码验证码、
 * 联系人绑定码、渠道测试邮件、站内信定向投递），本页只查不改。
 * target 服务端已脱敏（b***@example.com），台账不得成为明文集邮地址的第二个真相源，
 * 故此处原样展示、不再二次掩码。
 *
 * 找回密码/换绑验证码的投递是异步的（asynq 任务），因此「发送中」是正常中间态而不是卡住；
 * 分辨它靠 attempts，见该列注释。
 */
const pageRef = ref();

const pageConfig = computed<ProPageConfig<NotificationDelivery>>(() => ({
  skeleton: true,
  exportFilename: "notification-deliveries",
  search: {
    grid: true,
    fields: [
      {
        type: "select",
        label: $t("pages.notification_delivery.eventType"),
        field: "eventType",
        attrs: { placeholder: $t("common.placeholder.select"), clearable: true, filterable: true },
        options: notificationDeliveryEventTypeList.value,
      },
      {
        type: "select",
        label: $t("pages.notification_delivery.channel"),
        field: "channel",
        attrs: { placeholder: $t("common.placeholder.select"), clearable: true, filterable: true },
        options: notificationDeliveryChannelFilterList.value,
      },
      {
        type: "input",
        label: $t("pages.notification_delivery.target"),
        field: "target",
        // 台账里存的就是脱敏串，输入完整地址同样只会命中这一形态
        tips: $t("pages.notification_delivery.targetMaskedHint"),
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "select",
        label: $t("pages.notification_delivery.status"),
        field: "status",
        attrs: { placeholder: $t("common.placeholder.select"), clearable: true, filterable: true },
        options: notificationDeliveryStatusList.value,
      },
    ],
  },

  table: {
    exportsAction: createPagedExportAction(fetchListNotificationDeliveries),

    listAction: async (query: any) => {
      const { page, pageSize, ...queryParams } = query;
      try {
        const result = await fetchListNotificationDeliveries(
          new PaginationQuery({
            paging: { page: page || 1, pageSize: pageSize || 20 },
            formValues: {
              eventType: queryParams.eventType,
              channel: queryParams.channel,
              target: queryParams.target,
              status: queryParams.status,
            },
            orderBy: ["-created_at"],
          })
        );
        return { items: result.items || [], total: result.total || 0 };
      } catch (error: any) {
        // 不吞错：ElMessage 只是给用户看的文案，排查要靠控制台里的原始错误对象
        console.error("list notification deliveries failed:", error);
        ElMessage.error(error?.message || $t("pages.notification_delivery.fetchFailed"));
        return { items: [], total: 0 };
      }
    },
    toolbar: [],
    toolbarRight: [],
    defaultToolbar: ["refresh", "exports", "filter"],
    tableAttrs: { border: true, stripe: false },
    columns: [
      {
        prop: "createdAt",
        label: $t("pages.notification_delivery.createdAt"),
        width: 170,
        cellType: "date",
        dateFormat: "YYYY-MM-DD HH:mm:ss",
      },
      {
        prop: "eventType",
        label: $t("pages.notification_delivery.eventType"),
        width: 130,
        formatter: (row: NotificationDelivery) =>
          notificationDeliveryEventTypeToName(row.eventType) || "-",
      },
      {
        prop: "channel",
        label: $t("pages.notification_delivery.channel"),
        width: 110,
        slotName: "channel",
      },
      {
        prop: "target",
        label: $t("pages.notification_delivery.target"),
        minWidth: 180,
      },
      {
        prop: "status",
        label: $t("pages.notification_delivery.status"),
        width: 110,
        slotName: "status",
      },
      {
        // 含首次。同步投递恒为 1；异步投递每次尝试先加再一次拨号，
        // 所以「发送中 + 0」是还在队列里等，「发送中 + ≥1」是拨过号还没定案。
        prop: "attempts",
        label: $t("pages.notification_delivery.attempts"),
        width: 100,
        align: "right",
        formatter: (row: NotificationDelivery) => row.attempts ?? 0,
      },
      {
        prop: "recipientUserId",
        label: $t("pages.notification_delivery.recipientUserId"),
        width: 110,
        align: "right",
      },
      {
        prop: "channelId",
        label: $t("pages.notification_delivery.channelId"),
        width: 110,
        align: "right",
      },
      {
        // 台账不存正文快照，related_id 是"这条投递发的是什么"的唯一回跳入口（站内信 = 消息 ID）
        prop: "relatedId",
        label: $t("pages.notification_delivery.relatedId"),
        width: 110,
        align: "right",
        formatter: (row: NotificationDelivery) => row.relatedId ?? "-",
      },
      {
        // 同一次业务调用产生的多条投递共享此 ID（调用方不传时服务端生成），排障时按它把一行行投递串起来
        prop: "requestId",
        label: $t("pages.notification_delivery.requestId"),
        minWidth: 150,
        formatter: (row: NotificationDelivery) => row.requestId || "-",
      },
      {
        prop: "sentAt",
        label: $t("pages.notification_delivery.sentAt"),
        width: 170,
        cellType: "date",
        dateFormat: "YYYY-MM-DD HH:mm:ss",
      },
      {
        prop: "lastError",
        label: $t("pages.notification_delivery.lastError"),
        minWidth: 260,
        formatter: (row: NotificationDelivery) => row.lastError || "-",
      },
    ],
  },
}));
</script>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
  width: 100%;
  min-width: 0;
  flex-shrink: 0;
}
</style>
