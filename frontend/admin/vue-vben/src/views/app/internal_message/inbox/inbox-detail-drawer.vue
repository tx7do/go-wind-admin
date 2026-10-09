<script lang="ts" setup>
import type {
  internal_messageservicev1_InternalMessage as InternalMessage,
  internal_messageservicev1_InternalMessageRecipient as InternalMessageRecipient,
} from '#/api';

import { computed, ref } from 'vue';

import { useVbenDrawer } from '@vben/common-ui';
import { dateUtil } from '@vben/utils';

import DOMPurify from 'dompurify';
import { Spin } from 'ant-design-vue';

import {
  fetchGetInternalMessage,
  internalMessageRecipientStatusColor,
  internalMessageRecipientStatusLabel,
  useMarkNotificationAsRead,
} from '#/api';
import { $t } from '#/locales';
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

const publishTime = computed(() =>
  detail.value?.createdAt
    ? dateUtil(detail.value.createdAt as string).format('YYYY-MM-DD HH:mm:ss')
    : '',
);

const readTime = computed(() =>
  row.value?.readAt
    ? dateUtil(row.value.readAt as string).format('YYYY-MM-DD HH:mm:ss')
    : '',
);

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
    const userId = userStore.userInfo?.id ?? 0;
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
        <!-- 头部信息 -->
        <div class="message-detail__header mb-4">
          <h2 class="mb-2 text-lg font-semibold">
            {{ detail?.title ?? row?.title }}
          </h2>
          <div class="flex flex-wrap items-center gap-x-6 gap-y-2 text-sm">
            <span>
              <span class="text-muted-foreground">
                {{ $t('page.internalMessage.senderName') }}:
              </span>
              {{ detail?.senderName || '-' }}
            </span>
            <span>
              <span class="text-muted-foreground">
                {{ $t('ui.table.createdAt') }}:
              </span>
              {{ publishTime || '-' }}
            </span>
            <span>
              <span class="text-muted-foreground">
                {{ $t('page.internalMessage.readAt') }}:
              </span>
              {{ readTime || '-' }}
            </span>
            <a-tag
              v-if="row?.status"
              :color="internalMessageRecipientStatusColor(row.status)"
            >
              {{ internalMessageRecipientStatusLabel(row.status) }}
            </a-tag>
          </div>
        </div>

        <div class="border-t border-border my-4" />

        <!-- 正文（富文本，已净化） -->
        <!-- eslint-disable-next-line vue/no-v-html -->
        <div class="message-detail__content" v-html="sanitizedContent" />
      </div>
    </Spin>
  </Drawer>
</template>

<style scoped>
.message-detail__content {
  line-height: 1.8;
  word-break: break-word;
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
