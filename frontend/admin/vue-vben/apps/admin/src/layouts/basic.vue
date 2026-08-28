<script lang="ts" setup>
import type { NotificationItem } from '@vben/layouts';

import { computed, ref, watch, onUnmounted } from 'vue';

import { AuthenticationLoginExpiredModal } from '@vben/common-ui';
import { useWatermark } from '@vben/hooks';
import { LucideInbox, LucideUserRoundPen } from '@vben/icons';
import {
  BasicLayout,
  LockScreen,
  Notification,
  UserDropdown,
} from '@vben/layouts';
import { preferences } from '@vben/preferences';
import { useAccessStore, useUserStore } from '@vben/stores';
import { dateUtil } from '@vben/utils';

import { notification } from 'ant-design-vue';

import {
  fetchListUserInbox,
  PaginationQuery,
  useMarkNotificationAsRead,
} from '#/api';
import { type internal_messageservicev1_InternalMessageRecipient as InternalMessageRecipient } from '#/api';
import { $t } from '#/locales';
import { router } from '#/router';
import { useAuthStore } from '#/stores';
import { globalSSEClient } from '#/transport/sse';
import LoginForm from '#/views/_core/authentication/login.vue';

const userStore = useUserStore();
const authStore = useAuthStore();
const accessStore = useAccessStore();
const { mutateAsync: markNotificationAsRead } = useMarkNotificationAsRead();

const notifications = ref<NotificationItem[]>([]);

const showDot = computed(() =>
  notifications.value.some((item) => !item.isRead),
);

const { destroyWatermark, updateWatermark } = useWatermark();

const menus = computed(() => [
  {
    handler: () => router.push('/profile'),
    icon: LucideUserRoundPen,
    text: $t('menu.profile.settings'),
  },
  {
    handler: () => router.push('/inbox'),
    icon: LucideInbox,
    text: $t('menu.profile.internalMessage'),
  },
]);

const avatar = computed(() => {
  return userStore.userInfo?.avatar ?? preferences.app.defaultAvatar;
});

/**
 * 重载用户收件箱列表
 */
async function reloadMessages() {
  const resp = await fetchListUserInbox(
    new PaginationQuery({
      paging: {
        page: 1,
        pageSize: 5,
      },
      formValues: {
        recipient_user_id: userStore.userInfo?.id.toString(),
      },
      orderBy: ['-created_at'],
    }),
  );

  // 整体替换而非 push，避免重复挂载/多次调用时同一批消息被累积进列表。
  notifications.value = (resp.items ?? []).map(
    convertInternalMessageRecipient,
  );
}

/**
 * 把富文本 HTML 去标签转为纯文本摘要。
 * 用 DOMParser 而非 innerHTML，避免设置 innerHTML 时执行 <img onerror> 等脚本。
 * 用于通知面板的预览摘要（无需保留富文本格式，纯文本即可）。
 */
function htmlToText(html: string): string {
  try {
    const doc = new DOMParser().parseFromString(html, 'text/html');
    return doc.body.textContent || '';
  } catch {
    return '';
  }
}

/**
 * 把收件箱数据转换为UI数据
 * @param item
 */
function convertInternalMessageRecipient(item: InternalMessageRecipient) {
  const date = dateUtil(item.createdAt as string).fromNow();
  return {
    id: item.id ?? 0,
    messageId: item.messageId ?? 0,
    avatar: preferences.app.defaultAvatar,
    date,
    isRead: item.status === 'READ',
    // item.content 是不可信富文本 HTML，通知面板只需纯文本摘要。
    // 用 DOMParser 去标签，避免 XSS（不执行脚本）且避免显示原始标签文本。
    message: htmlToText(item.content || ''),
    title: item.title || '',
  };
}

/**
 * 登出账号
 */
async function handleLogout() {
  await authStore.logout(false);
}

/**
 * 清空通知
 */
function handleNoticeClear() {
  notifications.value = [];
}

/**
 * 标记为已读
 * @param item
 */
async function handleMarkAsRead(item: NotificationItem) {
  if (item.isRead) {
    return;
  }

  try {
    await markNotificationAsRead({
      userId: userStore.userInfo?.id ?? 0,
      recipientIds: [item.id],
    });

    notification.success({
      message: $t('ui.notification.update_success'),
    });
  } catch {
    notification.error({
      message: $t('ui.notification.update_failed'),
    });
  } finally {
    for (const n of notifications.value) {
      if (n.id === item.id) {
        n.isRead = true;
      }
    }
  }
}

/**
 * 全部通知标识为已读
 */
async function handleMakeAll() {
  if (!notifications.value.some((item) => !item.isRead)) {
    return;
  }

  try {
    // 空 recipientIds 表示标记该用户全部未读（服务端按用户维度兜底）：
    // 面板只加载了当前页，无法枚举全部未读ID。
    await markNotificationAsRead({
      userId: userStore.userInfo?.id ?? 0,
      recipientIds: [],
    });

    notification.success({
      message: $t('ui.notification.update_success'),
    });
  } catch {
    notification.error({
      message: $t('ui.notification.update_failed'),
    });
  } finally {
    notifications.value.forEach((item) => (item.isRead = true));
  }
}

/**
 * 查看所有消息，跳转至收件箱
 */
function handleViewAllNotifications() {
  router.push('/inbox');
}

function hasMessage(data: InternalMessageRecipient): boolean {
  for (const item of notifications.value) {
    if (item.messageId === data.messageId) {
      return true;
    }
  }
  return false;
}

function handleSseNotification(
  data: InternalMessageRecipient,
  event: MessageEvent,
) {
  console.log('SSE', event, data);

  if (!hasMessage(data)) {
    notifications.value.unshift(convertInternalMessageRecipient(data));
  }
}

function initSseClient() {
  globalSSEClient.on<InternalMessageRecipient>(
    'notification',
    handleSseNotification,
  );
}

initSseClient();
reloadMessages();

// 必须在卸载时注销 SSE 监听器，否则 globalSSEClient.handlers 会持续累加：
// basicLayout 在登录态切换（登出再登入、登录过期 modal 流程）时会被反复挂载，
// 若不 off，同一 SSE 推送会触发 N 次回调（N=累计挂载次数），通知项重复出现，
// 且旧组件闭包引用旧 ref 造成内存泄漏。
onUnmounted(() => {
  globalSSEClient.off('notification', handleSseNotification);
});

watch(
  () => preferences.app.watermark,
  async (enable) => {
    if (enable) {
      await updateWatermark({
        content: `${userStore.userInfo?.username}`,
      });
    } else {
      destroyWatermark();
    }
  },
  {
    immediate: true,
  },
);
</script>

<template>
  <BasicLayout @clear-preferences-and-logout="handleLogout">
    <template #user-dropdown>
      <UserDropdown
        :avatar
        :menus
        :text="userStore.userInfo?.realname"
        :description="userStore.userInfo?.email"
        @logout="handleLogout"
      />
    </template>
    <template #notification>
      <Notification
        :dot="showDot"
        :notifications="notifications"
        @clear="handleNoticeClear"
        @make-all="handleMakeAll"
        @read="handleMarkAsRead"
        @view-all="handleViewAllNotifications"
      />
    </template>
    <template #extra>
      <AuthenticationLoginExpiredModal
        v-model:open="accessStore.loginExpired"
        :avatar
      >
        <LoginForm />
      </AuthenticationLoginExpiredModal>
    </template>
    <template #lock-screen>
      <LockScreen :avatar @to-login="handleLogout" />
    </template>
  </BasicLayout>
</template>
