<script lang="ts" setup>
import type { NotificationItem } from '@vben/layouts';

import { computed, ref, watch } from 'vue';

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

import { type internal_messageservicev1_InternalMessageRecipient as InternalMessageRecipient } from '#/generated/api/admin/service/v1';
import { $t } from '#/locales';
import { router } from '#/router';
import { useAuthStore, useInternalMessageStore } from '#/stores';
import { SSEClient } from '#/transport/sse';
import LoginForm from '#/views/_core/authentication/login.vue';

const userStore = useUserStore();
const authStore = useAuthStore();
const accessStore = useAccessStore();
const internalMessageStore = useInternalMessageStore();

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
  const resp = await internalMessageStore.listUserInbox(
    {
      page: 1,
      pageSize: 5,
    },
    {
      recipient_user_id: userStore.userInfo?.id.toString(),
    },
    null,
    ['-created_at'],
  );

  for (const item of resp.items ?? []) {
    notifications.value.push(convertInternalMessageRecipient(item));
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
    message: item.content || '',
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
function handleMarkAsRead(item: NotificationItem) {
  if (item.isRead) {
    return;
  }

  try {
    internalMessageStore.markNotificationAsRead(userStore.userInfo?.id ?? 0, [
      item.id,
    ]);

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
function handleMakeAll() {
  const ids: number[] = [];
  for (const item of notifications.value) {
    if (!item.isRead) {
      ids.push(item.id);
    }
  }

  if (ids.length === 0) {
    return;
  }

  try {
    internalMessageStore.markNotificationAsRead(
      userStore.userInfo?.id ?? 0,
      ids,
    );

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
  const targetSseUrl = `${import.meta.env.VITE_GLOB_SSE_URL}?stream=${encodeURIComponent(accessStore.accessToken ?? '')}`;
  const sseClient = new SSEClient({
    url: targetSseUrl,
    withCredentials: false,
  });

  sseClient.connect();
  sseClient.on<InternalMessageRecipient>('notification', handleSseNotification);
}

initSseClient();
reloadMessages();

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
