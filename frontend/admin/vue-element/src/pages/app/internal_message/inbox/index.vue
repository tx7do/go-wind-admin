<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig">
      <!-- 标题：点击查看详情（操作列不再放查看按钮） -->
      <template #title="scope: any">
        <ElLink type="primary" underline="never" @click="handleView(scope.row)">
          {{ scope.row.title || "-" }}
        </ElLink>
      </template>

      <!-- 状态 -->
      <template #status="scope: any">
        <ElTag
          size="small"
          round
          :type="internalMessageRecipientStatusToType(scope.row.status)"
        >
          {{ internalMessageRecipientStatusLabel(scope.row.status) }}
        </ElTag>
      </template>

      <!-- 操作：标记已读（未读时）+ 删除；查看走标题链接。
           操作列统一为图标+文本 link 按钮（同 cellType:"tool" 的 ToolCell 形态） -->
      <template #operation="scope: any">
        <ElButton
          v-if="scope.row.status !== 'READ'"
          link
          type="primary"
          size="small"
          @click="handleMarkRead(scope.row)"
        >
          <SvgIcon icon="lucide:check" :size="14" />
          {{ $t("pages.internal_message.button.markAsRead") }}
        </ElButton>
        <ElButton link type="danger" size="small" @click="handleDelete(scope.row)">
          <SvgIcon icon="lucide:trash-2" :size="14" />
          {{ $t("common.button.delete") }}
        </ElButton>
      </template>

      <!-- 全部已读（工具栏）：recipientIds 传空 = 标记该用户全部未读 -->
      <template #toolbar-right>
        <ElButton
          type="primary"
          plain
          :loading="markAllReadLoading"
          :disabled="!userStore.userInfo?.id"
          @click="handleMarkAllRead"
        >
          {{ $t("pages.internal_message.button.markAllRead") }}
        </ElButton>
      </template>
    </ProPage>

    <!-- 消息详情抽屉 -->
    <ProModal
      v-model:visible="detailDialogVisible"
      :title="detail?.title ?? $t('core.notice.detail')"
      :config="{
        component: 'drawer',
        drawer: { size: '720px', closeOnClickModal: true },
      }"
    >
      <div v-if="detail" class="message-detail">
        <div class="message-detail__header">
          <div class="message-detail__meta">
            <ElAvatar :size="40" :src="defaultAvatar">
              <el-icon :size="20"><UserFilled /></el-icon>
            </ElAvatar>
            <div class="message-detail__info">
              <div class="message-detail__sender">{{ senderShown }}</div>
              <div class="message-detail__time">
                {{ $t("pages.internal_message.createdAt") }} {{ receivedTimeText }}
              </div>
              <div v-if="readTimeText" class="message-detail__time">
                {{ $t("pages.internal_message.readAt") }} {{ readTimeText }}
              </div>
            </div>
          </div>
          <ElTag
            size="small"
            round
            :type="internalMessageRecipientStatusToType(detailStatus)"
          >
            {{ internalMessageRecipientStatusLabel(detailStatus) }}
          </ElTag>
        </div>

        <ElDivider />

        <!-- 正文格式自适应：富文本 HTML 净化渲染；纯文本/Markdown 走 MdPreview -->
        <div class="message-detail__content">
          <MdPreview
            v-if="!isHtmlContent"
            :model-value="rawContent"
            :theme="isDark ? 'dark' : 'light'"
            preview-theme="default"
            :sanitize="sanitizeHtml"
          />
          <!-- eslint-disable-next-line vue/no-v-html -- 已由 DOMPurify 净化（见 sanitizedContent） -->
          <div v-else class="message-detail__html" v-html="sanitizedContent"></div>
        </div>
      </div>
    </ProModal>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from "vue";
import {
  ElTag,
  ElDivider,
  ElAvatar,
  ElMessage,
  ElMessageBox,
  ElLink,
  ElButton,
} from "element-plus";
import { UserFilled } from "@element-plus/icons-vue";
import dayjs from "dayjs";
import DOMPurify from "dompurify";
import { MdPreview } from "md-editor-v3";
import "md-editor-v3/lib/preview.css";

import ProPage from "@/components/Pro/ProPage/index.vue";
import ProModal from "@/components/Pro/ProModal/index.vue";
import SvgIcon from "@/components/SvgIcon/index.vue";
import { isDarkMode } from "@/components/Editor";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";

import { $t } from "@/core/i18n";
import {
  internalMessageRecipientStatusToType,
  internalMessageRecipientStatusLabel,
  fetchListUserInbox,
  fetchGetInternalMessage,
  useMarkNotificationAsRead,
  useDeleteNotificationFromInbox,
} from "@/api/composables";
import { useAppUserStore } from "@/stores";
import { PaginationQuery } from "@/core/transport/rest";
import defaultAvatar from "@/assets/images/default-avatar.png";

const { mutateAsync: markNotificationAsRead } = useMarkNotificationAsRead();
const { mutateAsync: deleteNotificationFromInbox } =
  useDeleteNotificationFromInbox();
const userStore = useAppUserStore();

const pageRef = ref();
const detailDialogVisible = ref(false);
const detail = ref<any | null>(null);
// 收件行的已读时间与状态（readAt/status 在 recipient 行上，消息详情里没有）
const detailReadAt = ref<string>("");
// eslint-disable-next-line @typescript-eslint/no-explicit-any -- 行数据来自表格 dynamic row
const detailStatus = ref<any>("");
const detailReceivedAt = ref<string>("");

// 对消息正文进行 XSS 净化，防止富文本内容中注入脚本（存储型 XSS）
const sanitizedContent = computed(() => {
  const raw = detail.value?.content;
  return raw ? DOMPurify.sanitize(String(raw)) : "";
});

// 正文格式自适应（对齐 react 抽屉）：富文本 HTML 净化渲染，
// 纯文本/Markdown（AI 日报等）走 MdPreview
const rawContent = computed(() => (detail.value?.content ? String(detail.value.content) : ""));
const isHtmlContent = computed(() => /<\/?[a-z][\s\S]*>/i.test(rawContent.value));
const isDark = computed(() => isDarkMode());
const sanitizeHtml = (html: string) => DOMPurify.sanitize(html);

// 发送人：sender_id=0 的系统消息（AI 日报等）senderName 为空，显示"系统"而非占位符
const senderShown = computed(
  () => detail.value?.senderName || $t("pages.internal_message.senderSystem"),
);
const receivedTimeText = computed(() =>
  detailReceivedAt.value
    ? dayjs(detailReceivedAt.value).format("YYYY-MM-DD HH:mm:ss")
    : "-",
);
const readTimeText = computed(() =>
  detailReadAt.value ? dayjs(detailReadAt.value).format("YYYY-MM-DD HH:mm:ss") : "",
);

const pageConfig = computed<ProPageConfig>(() => ({
  skeleton: true,
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
        label: $t("pages.internal_message.createdAt"),
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
    defaultToolbar: ["exports", "refresh", "zoom", "filter"],
    // 导出与列表同口径：区间归一为 gte/lte、recipient_user_id 强制注入（隐私闸门）
    exportFilename: "im-inbox",
    tableAttrs: { border: true, stripe: true, height: "auto" },
    columns: [
      {
        prop: "title",
        label: $t("pages.internal_message.title"),
        minWidth: 200,
        align: "left",
        slotName: "title",
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
        label: $t("pages.internal_message.createdAt"),
        width: 160,
        cellType: "date",
        dateFormat: "YYYY-MM-DD HH:mm:ss",
      },
      {
        prop: "operation",
        label: $t("common.table.operation"),
        width: 170,
        slotName: "operation",
        fixed: "right",
      },
    ],
  },
}));

async function handleView(row: any) {
  try {
    const messageId = Number(row.messageId);
    detailReadAt.value = row.readAt ? String(row.readAt) : "";
    detailReceivedAt.value = row.receivedAt || row.createdAt || "";
    detailStatus.value = row.status || "";
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

/* 标记为已读（操作列） */
async function handleMarkRead(row: any) {
  const userId = userStore.userInfo?.id;
  if (!userId) return;
  try {
    await markNotificationAsRead({ userId, recipientIds: [Number(row.id)] });
    ElMessage.success($t("common.notice.markedRead"));
    pageRef.value?.refresh();
  } catch {
    ElMessage.error($t("common.message.operationFailed"));
  }
}

/* 删除（操作列） */
async function handleDelete(row: any) {  try {
    await ElMessageBox.confirm(
      $t("common.text.do_you_want_delete", {
        moduleName: $t("pages.internal_message.moduleName"),
      }),
      $t("common.title.confirm"),
      {
        confirmButtonText: $t("common.button.confirm"),
        cancelButtonText: $t("common.button.cancel"),
        type: "warning",
        lockScroll: false,
      }
    );
  } catch (error) {
    // 取消是 ElMessageBox 的 reject("cancel")，属正常路径；其余形态留痕
    if (error !== "cancel" && error !== "close") {
      console.error("delete confirm dialog rejected unexpectedly", error);
    }
    return;
  }

  const userId = userStore.userInfo?.id;
  if (!userId) return;
  try {
    await deleteNotificationFromInbox({ userId, recipientIds: [Number(row.id)] });
    ElMessage.success($t("common.notification.deleteSuccess"));
    pageRef.value?.refresh();
  } catch {
    ElMessage.error($t("common.notification.deleteFailed"));
  }
}

/* 全部已读（工具栏）：recipientIds 传空 = 标记该用户全部未读 */
const markAllReadLoading = ref(false);
async function handleMarkAllRead() {
  const userId = userStore.userInfo?.id;
  if (!userId) return;
  markAllReadLoading.value = true;
  try {
    await markNotificationAsRead({ userId, recipientIds: [] });
    ElMessage.success($t("common.notice.markedRead"));
    pageRef.value?.refresh();
  } catch {
    ElMessage.error($t("common.message.operationFailed"));
  } finally {
    markAllReadLoading.value = false;
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
  /* md-editor 暗色预览自带近黑底色（--md-bk-color），与抽屉表面冲突——
     置透明融入抽屉，文字颜色继承主题 */
  :deep(.md-editor-previewOnly),
  :deep(.md-editor-previewWrapper),
  :deep(.md-editor-preview) {
    background-color: transparent;
  }

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
</style>
