<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig">
      <!-- 状态 -->
      <template #status="scope: any">
        <ElTag
          size="small"
          effect="dark"
          round
          :color="internalMessageRecipientStatusColor(scope.row.status)"
        >
          {{ internalMessageRecipientStatusLabel(scope.row.status) }}
        </ElTag>
      </template>

      <!-- 操作 -->
      <template #operation="scope: any">
        <el-button link type="primary" size="small" @click="handleView(scope.row)">
          {{ $t("common.button.view") }}
        </el-button>
      </template>
    </ProPage>

    <!-- 消息详情对话框 -->
    <el-dialog
      v-model="detailDialogVisible"
      :title="detail?.title ?? $t('core.notice.detail')"
      width="800px"
      custom-class="message-detail-dialog"
    >
      <div v-if="detail" class="message-detail">
        <div class="message-detail__header">
          <div class="message-detail__meta">
            <ElAvatar :size="40" :src="defaultAvatar">
              <el-icon :size="20"><UserFilled /></el-icon>
            </ElAvatar>
            <div class="message-detail__info">
              <div class="message-detail__sender">{{ detail.publisherName }}</div>
              <div class="message-detail__time">{{ detail.publishTime }}</div>
            </div>
          </div>
          <ElTag
            size="small"
            effect="dark"
            round
            :color="internalMessageRecipientStatusColor(detail.status)"
          >
            {{ internalMessageRecipientStatusLabel(detail.status) }}
          </ElTag>
        </div>

        <ElDivider />

        <div class="message-detail__content" v-html="detail.content"></div>
      </div>
    </el-dialog>
  </div>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import { ElTag, ElDivider, ElAvatar, ElMessage } from "element-plus";
import { UserFilled } from "@element-plus/icons-vue";
import dayjs from "dayjs";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";

import { $t } from "@/i18n";
import {
  internalMessageRecipientStatusColor,
  internalMessageRecipientStatusLabel,
  fetchListUserInbox,
  fetchGetInternalMessage,
  useMarkNotificationAsRead,
} from "@/api/composables";
import { useAppUserStore } from "@/stores";
import { PaginationQuery } from "@/core/transport/rest";
import defaultAvatar from "@/assets/images/default-avatar.png";

const { mutateAsync: markNotificationAsRead } = useMarkNotificationAsRead();
const userStore = useAppUserStore();

const pageRef = ref();
const detailDialogVisible = ref(false);
const detail = ref<any | null>(null);

const pageConfig = computed<ProPageConfig>(() => ({

  search: {
    grid: true,
    fields: [
      {
        type: "input",
        label: $t("pages.internal_message.title"),
        field: "title",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "select",
        label: $t("pages.internal_message.status"),
        field: "status",
        attrs: { placeholder: $t("common.placeholder.select"), clearable: true },
        options: [
          { label: $t("enum.internalMessageRecipient.status.SENT"), value: "SENT" },
          { label: $t("enum.internalMessageRecipient.status.RECEIVED"), value: "RECEIVED" },
          { label: $t("enum.internalMessageRecipient.status.READ"), value: "READ" },
        ],
      },
      {
        type: "date-picker",
        label: $t("common.table.createdAt"),
        field: "createdAt",
        attrs: {
          type: "datetimerange",
          startPlaceholder: $t("common.placeholder.date"),
          endPlaceholder: $t("common.placeholder.date"),
          clearable: true,
          shortcuts: [
            {
              text: $t("common.dateRange.today"),
              value: () => [dayjs().startOf("day").toDate(), dayjs().endOf("day").toDate()],
            },
            {
              text: $t("common.dateRange.yesterday"),
              value: () => [
                dayjs().subtract(1, "day").startOf("day").toDate(),
                dayjs().subtract(1, "day").endOf("day").toDate(),
              ],
            },
            {
              text: $t("common.dateRange.thisWeek"),
              value: () => [dayjs().startOf("week").toDate(), dayjs().endOf("week").toDate()],
            },
            {
              text: $t("common.dateRange.lastWeek"),
              value: () => [
                dayjs().subtract(1, "week").startOf("week").toDate(),
                dayjs().subtract(1, "week").endOf("week").toDate(),
              ],
            },
            {
              text: $t("common.dateRange.thisMonth"),
              value: () => [dayjs().startOf("month").toDate(), dayjs().endOf("month").toDate()],
            },
            {
              text: $t("common.dateRange.lastMonth"),
              value: () => [
                dayjs().subtract(1, "month").startOf("month").toDate(),
                dayjs().subtract(1, "month").endOf("month").toDate(),
              ],
            },
          ],
        },
      },
    ],
  },

  table: {
    listAction: async (query: any) => {
      const userId = userStore.userInfo?.id;
      if (!userId) return { items: [], total: 0 };

      const { page, pageSize, ...queryParams } = query;
      let startTime: string | undefined;
      let endTime: string | undefined;

      if (
        queryParams.createdAt !== undefined &&
        Array.isArray(queryParams.createdAt) &&
        queryParams.createdAt.length === 2
      ) {
        startTime = dayjs(queryParams.createdAt[0]).format("YYYY-MM-DD HH:mm:ss");
        endTime = dayjs(queryParams.createdAt[1]).format("YYYY-MM-DD HH:mm:ss");
      }

      const result = await fetchListUserInbox(
        new PaginationQuery({
          paging: { page: page || 1, pageSize: pageSize || 10 },
          formValues: {
            recipient_user_id: userId.toString(),
            title: queryParams.title || undefined,
            status: queryParams.status || undefined,
            created_at__gte: startTime,
            created_at__lte: endTime,
          },
        })
      );
      return { items: result.items || [], total: result.total || 0 };
    },
    toolbar: [],
    toolbarRight: [],
    defaultToolbar: ["refresh", "filter"],
    tableAttrs: { border: true, stripe: true, height: "auto" },
    columns: [
      {
        prop: "title",
        label: $t("pages.internal_message.title"),
        minWidth: 200,
        align: "left",
      },
      {
        prop: "status",
        label: $t("pages.internal_message.status"),
        width: 120,
        slotName: "status",
      },
      {
        prop: "readAt",
        label: $t("pages.internal_message.readAt"),
        width: 160,
        cellType: "date",
        dateFormat: "YYYY-MM-DD HH:mm:ss",
      },
      {
        prop: "createdAt",
        label: $t("common.table.createdAt"),
        width: 160,
        cellType: "date",
        dateFormat: "YYYY-MM-DD HH:mm:ss",
      },
      {
        prop: "operation",
        label: $t("common.table.operation"),
        width: 100,
        slotName: "operation",
        fixed: "right",
      },
    ],
  },
}));

async function handleView(row: any) {
  try {
    const messageId = Number(row.messageId);
    detail.value = await fetchGetInternalMessage({ id: messageId });
    detailDialogVisible.value = true;

    const userId = userStore.userInfo?.id;
    if (userId && row.status !== "READ") {
      try {
        await markNotificationAsRead({ userId, recipientIds: [Number(row.id)] });
        pageRef.value?.refresh();
      } catch {
        ElMessage.error($t("common.message.operationFailed"));
      }
    }
  } catch {
    ElMessage.error($t("common.message.getDetailFailed"));
  }
}
</script>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
  width: 100%;
  min-width: 0;
  flex-shrink: 0;
}

.message-detail {
  &__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 16px;
  }

  &__meta {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  &__info {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  &__sender {
    font-size: 14px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  &__time {
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  &__content {
    padding: 16px 0;
    line-height: 1.8;
    color: var(--el-text-color-primary);
    min-height: 200px;

    :deep(img) {
      max-width: 100%;
      height: auto;
    }
  }
}

:deep(.message-detail-dialog) {
  .el-dialog__header {
    padding-bottom: 16px;
  }

  .el-dialog__body {
    padding: 20px;
  }
}
</style>
