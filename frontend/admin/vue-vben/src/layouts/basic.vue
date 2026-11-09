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
  registerSemanticSearchProvider,
  UserDropdown,
} from '@vben/layouts';
import { preferences } from '@vben/preferences';
import { useAccessStore, useUserStore } from '@vben/stores';
import { dateUtil } from '@vben/utils';

import { notification } from 'ant-design-vue';

import {
  fetchListUserInbox,
  fetchSemanticSearch,
  PaginationQuery,
  useMarkNotificationAsRead,
} from '#/api';
import { type internal_messageservicev1_InternalMessageRecipient as InternalMessageRecipient } from '#/api';
import { $t } from '#/locales';
import { router } from '#/router';
import { useAuthStore } from '#/stores';
import { globalSSEClient, SSE_EVENT } from '#/transport/sse';
import LoginForm from '#/views/_core/authentication/login.vue';

// 语义搜索（pgvector 菜单索引）接入全局搜索面板；框架层经注册表解耦，不直接依赖本应用 API
registerSemanticSearchProvider(async (query) => {
  const resp = await fetchSemanticSearch(query, 8);
  return (resp.items ?? []).map((item) => ({
    route: item.route ?? '',
    score: item.score,
    title: item.title ?? '',
  }));
});

const userStore = useUserStore();
const authStore = useAuthStore();
const accessStore = useAccessStore();
const { mutateAsync: markNotificationAsRead } = useMarkNotificationAsRead();

const notifications = ref<NotificationItem[]>([]);

// 铃铛角标未读数：服务端独立 COUNT（status=RECEIVED），不能用面板当前页条数——后者封顶在 pageSize
const unreadCount = ref(0);

// 「全部已读」请求进行中（按钮 loading + 防重入）
const makeAllLoading = ref(false);

// 兜底轮询：实时更新靠 SSE notification 事件触发重拉，这里只在 SSE 失联时补拉
const POLL_INTERVAL = 300_000;
let pollTimer: ReturnType<typeof setInterval> | undefined;

async function reloadUnreadCount() {
  const userId = userStore.userInfo?.id;
  if (!userId) return;
  const resp = await fetchListUserInbox(
    new PaginationQuery({
      paging: { page: 1, pageSize: 1 },
      formValues: {
        recipient_user_id: userId.toString(),
        status: 'RECEIVED',
      },
    }),
  );
  // proto int64 经 protojson 运行时是字符串（TS 类型却标 number），必须显式转数字，
  // 否则 NotificationPopup 的 count prop 类型校验告警。
  unreadCount.value = Number(resp.total ?? 0);
}

const { destroyWatermark, updateWatermark } = useWatermark();

const menus = computed(() => [
  {
    handler: () => router.push('/profile'),
    icon: LucideUserRoundPen,
    text: $t('menu.profile.settings'),
  },
  {
    handler: () => router.push('/internal-message/inbox'),
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
  // userId 未就绪必须早退：undefined 条目会被 PaginationQuery 剥掉，
  // 请求变成无 recipient 过滤的租户级查询，列出其他用户的收件记录。
  const userId = userStore.userInfo?.id;
  if (!userId) return;

  const resp = await fetchListUserInbox(
    new PaginationQuery({
      paging: {
        page: 1,
        pageSize: 5,
      },
      formValues: {
        recipient_user_id: userId.toString(),
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
 * 把收件箱数据转换为UI数据。
 * 面板形态对齐 react 端：标题单行、纯文本摘要 46 字符封顶、相对时间右侧。
 * @param item
 */
function convertInternalMessageRecipient(item: InternalMessageRecipient) {
  // 缺失/非法时间显示 "-"（dayjs(undefined) 会落到当前时刻，fromNow 变"几秒前"）
  const created = item.createdAt ? dateUtil(item.createdAt as string) : null;
  const date = created?.isValid() ? created.fromNow() : '-';
  return {
    id: item.id ?? '',
    messageId: item.messageId ?? '',
    date,
    isRead: item.status === 'READ',
    // item.content 是不可信富文本 HTML，通知面板只需纯文本摘要。
    // 用 DOMParser 去标签，避免 XSS（不执行脚本）且避免显示原始标签文本。
    message: htmlToText(item.content || '')
      .replace(/\s+/g, ' ')
      .trim()
      .slice(0, 46),
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
 * 全部通知标识为已读（对齐 react 端：空 recipientIds 标记该用户全部未读）
 */
async function handleMakeAll() {
  const userId = userStore.userInfo?.id;
  if (!userId || unreadCount.value <= 0 || makeAllLoading.value) {
    return;
  }

  makeAllLoading.value = true;
  try {
    await markNotificationAsRead({
      userId,
      recipientIds: [],
    });

    notification.success({
      message: $t('ui.widgets.markAllReadSuccess'),
    });
    reloadMessages();
    reloadUnreadCount();
  } catch (error) {
    console.error('markAllRead failed:', error);
    notification.error({
      message: $t('ui.widgets.markAllReadFailed'),
    });
  } finally {
    makeAllLoading.value = false;
  }
}

/**
 * 查看所有消息，跳转至收件箱
 */
function handleViewAllNotifications() {
  router.push('/internal-message/inbox');
}

/**
 * SSE 收到新站内信 / 兜底轮询：整体重拉预览与未读数（不做增量插入）
 */
function reloadInboxPreview() {
  reloadMessages().catch((error) =>
    console.error('reload inbox preview failed:', error),
  );
  reloadUnreadCount().catch((error) =>
    console.error('reload unread count failed:', error),
  );
}

function handleSseNotification() {
  reloadInboxPreview();
}

function initSseClient() {
  globalSSEClient.on(SSE_EVENT.Notification, handleSseNotification);
}

initSseClient();
reloadInboxPreview();
pollTimer = setInterval(reloadInboxPreview, POLL_INTERVAL);

// 必须在卸载时注销 SSE 监听器，否则 globalSSEClient.handlers 会持续累加：
// basicLayout 在登录态切换（登出再登入、登录过期 modal 流程）时会被反复挂载，
// 若不 off，同一 SSE 推送会触发 N 次回调（N=累计挂载次数），通知项重复出现，
// 且旧组件闭包引用旧 ref 造成内存泄漏。
onUnmounted(() => {
  globalSSEClient.off(SSE_EVENT.Notification, handleSseNotification);
  if (pollTimer) clearInterval(pollTimer);
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
        :count="unreadCount"
        :make-all-loading="makeAllLoading"
        :notifications="notifications"
        @make-all="handleMakeAll"
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
