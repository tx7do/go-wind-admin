<script lang="ts" setup>
import type {
  internal_messageservicev1_InternalMessage as InternalMessage,
  internal_messageservicev1_InternalMessageRecipient as InternalMessageRecipient,
} from '#/api';

import { computed, ref } from 'vue';

import { useVbenDrawer } from '@vben/common-ui';
import { dateUtil } from '@vben/utils';

import DOMPurify from 'dompurify';
import { MdPreview } from 'md-editor-v3';
import 'md-editor-v3/lib/preview.css';
import { Avatar, Spin } from 'ant-design-vue';

import {
  fetchGetInternalMessage,
  internalMessageRecipientStatusColor,
  internalMessageRecipientStatusLabel,
  useMarkNotificationAsRead,
} from '#/api';
import { isDarkMode } from '#/adapter/component/Editor/src/utils';
import { $t } from '#/locales';
import { preferences } from '@vben/preferences';
import { useUserStore } from '@vben/stores';
import { notification } from 'ant-design-vue';

const userStore = useUserStore();
const { mutateAsync: markNotificationAsRead } = useMarkNotificationAsRead();

const loading = ref(false);
// 收件箱行（含 recipientId、messageId、status 等）
const row = ref<InternalMessageRecipient>();
// 消息完整内容（含富文本正文）
const detail = ref<InternalMessage>();

const getTitle = computed(() => row.value?.title || $t('page.internalMessage.detail'));

/**
 * 富文本正文经 DOMPurify 净化后再渲染，避免存储型 XSS。
 * content 来自后端不可信输入，直接 v-html 会有执行脚本风险。
 */
const sanitizedContent = computed(() => {
  const raw = detail.value?.content ?? row.value?.content ?? '';
  return raw ? DOMPurify.sanitize(String(raw)) : '';
});

const readTime = computed(() =>
  row.value?.readAt
    ? dateUtil(row.value.readAt as string).format('YYYY-MM-DD HH:mm:ss')
    : '',
);

// 接收时间取收件行（消息详情接口的 createdAt 为空）
const receivedTime = computed(() => {
  const v = row.value?.receivedAt || row.value?.createdAt;
  return v ? dateUtil(v as string).format('YYYY-MM-DD HH:mm:ss') : '';
});

// 主题（MdPreview 跟随；isDarkMode 读响应式 preferences）
const isDark = computed(() => isDarkMode());

// 正文格式自适应（对齐 react 抽屉）：富文本 HTML 净化渲染，
// 纯文本/Markdown（AI 日报等）走 MdPreview
const rawContent = computed(() =>
  String(detail.value?.content ?? row.value?.content ?? ''),
);
const isHtmlContent = computed(() => /<\/?[a-z][\s\S]*>/i.test(rawContent.value));
const sanitizeHtml = (html: string) => DOMPurify.sanitize(html);

// sender_id=0 的系统消息（AI 日报等）senderName 为空，显示"系统"而非占位符
const senderShown = computed(() => {
  if (detail.value?.senderName) return detail.value.senderName;
  return loading.value ? '-' : $t('page.internalMessage.senderSystem');
});

const [Drawer, drawerApi] = useVbenDrawer({
  footer: false,
  onOpenChange(isOpen: boolean) {
    if (isOpen) {
      loadDetail();
    } else {
      // 关闭时清空，避免下次打开闪现旧数据
      detail.value = undefined;
      row.value = undefined;
    }
  },
});

async function loadDetail() {
  row.value = drawerApi.getData<{ row: InternalMessageRecipient }>()?.row;

  const messageId = row.value?.messageId;
  if (!messageId) return;

  loading.value = true;
  try {
    detail.value = await fetchGetInternalMessage({ id: messageId });

    // 打开详情即标记为已读（仅当当前未读）。
    // 注意：用收件记录 id（recipientId），而非 messageId。
    const userId = userStore.userInfo?.id ?? '';
    const recipientId = row.value?.id;
    if (userId && recipientId != null && row.value?.status !== 'READ') {
      try {
        await markNotificationAsRead({
          userId,
          recipientIds: [recipientId],
        });
        // 本地同步状态，避免关闭前显示仍为未读
        if (row.value) row.value.status = 'READ';
      } catch {
        // 标记已读失败不阻断查看正文
        notification.error({
          message: $t('ui.notification.operation_failed'),
        });
      }
    }
  } catch {
    notification.error({
      message: $t('ui.notification.operation_failed'),
    });
  } finally {
    loading.value = false;
  }
}
</script>

<template>
    <Drawer :title="getTitle" class="w-full max-w-[800px]">
      <Spin :spinning="loading" class="min-h-[200px] w-full">
        <div class="message-detail mx-4 py-2">
          <!-- 发送人元信息头：头像 + 发送人/时间，状态 Tag 右置（对齐 react 抽屉） -->
          <div class="mb-4 flex items-center justify-between gap-3">
            <div class="flex min-w-0 items-center gap-3">
              <Avatar :size="40" :src="preferences.app.defaultAvatar" />
              <div class="flex min-w-0 flex-col gap-1">
                <div class="truncate text-sm font-semibold">
                  {{ senderShown }}
                </div>
                <div class="text-xs text-muted-foreground">
                  {{ $t('page.internalMessage.createdAt') }} {{ receivedTime || '-' }}
                </div>
                <div v-if="readTime" class="text-xs text-muted-foreground">
                  {{ $t('page.internalMessage.readAt') }} {{ readTime }}
                </div>
              </div>
            </div>
            <a-tag
              v-if="row?.status"
              :color="internalMessageRecipientStatusColor(row.status)"
            >
              {{ internalMessageRecipientStatusLabel(row.status) }}
            </a-tag>
          </div>

          <div class="border-t border-border my-4" />

          <!-- 正文：富文本 HTML 净化渲染；纯文本/Markdown 走 MdPreview -->
          <div
            class="message-detail__content rounded-lg border border-border bg-background p-4"
          >
            <MdPreview
              v-if="!isHtmlContent"
              :model-value="rawContent"
              :theme="isDark ? 'dark' : 'light'"
              preview-theme="default"
              :sanitize="sanitizeHtml"
            />
            <!-- eslint-disable-next-line vue/no-v-html -->
            <div v-else v-html="sanitizedContent" />
          </div>
        </div>
      </Spin>
    </Drawer>
</template>

<style scoped>
.message-detail__content {
  line-height: 1.8;
  word-break: break-word;
}

/* md-editor 暗色预览自带近黑底色（--md-bk-color），与抽屉表面冲突——
   置透明融入抽屉，文字颜色继承主题 */
.message-detail__content :deep(.md-editor-previewOnly),
.message-detail__content :deep(.md-editor-previewWrapper),
.message-detail__content :deep(.md-editor-preview) {
  background-color: transparent;
}

.message-detail__content :deep(img) {
  max-width: 100%;
  height: auto;
}

.message-detail__content :deep(a) {
  word-break: break-all;
}

.message-detail__content :deep(table) {
  max-width: 100%;
}
</style>
