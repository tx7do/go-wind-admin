<script lang="ts" setup>
import type {
  internal_messageservicev1_InternalMessageRecipient as InboxItem,
} from '#/api/generated/admin/service/v1';
import type {
  online_sessionservicev1_OnlineSession as OnlineSession,
} from '#/api/generated/admin/service/v1';

import { computed } from 'vue';
import { useRouter } from 'vue-router';

import { Page } from '@vben/common-ui';
import { $t } from '@vben/locales';
import { preferences } from '@vben/preferences';
import { useUserStore } from '@vben/stores';
import { formatDateTime } from '@vben/utils';

import { Icon as IconifyIcon } from '@iconify/vue';
import { Avatar, Card, Col, Empty, Row, Tag } from 'ant-design-vue';

import {
  PaginationQuery,
  useListMyOnlineSessions,
  useListUserInbox,
} from '#/api';

/**
 * 工作台：操作入口聚合页（概览组的第二页，与分析页的"数据洞察"分工）。
 * 与 react / vue-element 端同构：快捷入口 + 我的站内信 + 我的活跃会话。
 * 快捷入口不做权限过滤——与「菜单可见但禁止访问」口径一致，无权限由路由闸渲染 403。
 * 注意 Avatar/Row/Col/List/Empty 未在 registerGlobComp 全局注册，必须显式导入，
 * 否则 Vue 当未知元素静默不渲染（见 registerGlobComp.ts 注释）。
 */
const QUICK_LINKS = [
  { path: '/opm/users', labelKey: 'menu.opm.user', icon: 'lucide:user' },
  { path: '/permission/roles', labelKey: 'menu.permission.role', icon: 'lucide:users' },
  { path: '/permission/menus', labelKey: 'menu.permission.menu', icon: 'lucide:menu' },
  { path: '/system/dict', labelKey: 'menu.system.dict', icon: 'lucide:book-open' },
  { path: '/system/tasks', labelKey: 'menu.system.task', icon: 'lucide:clock' },
  { path: '/system/files', labelKey: 'menu.system.file', icon: 'lucide:folder' },
  { path: '/system/online-sessions', labelKey: 'menu.system.onlineSessions', icon: 'lucide:globe' },
  { path: '/log/login-audit-logs', labelKey: 'menu.log.loginAuditLog', icon: 'lucide:file-search' },
];

const router = useRouter();
const userStore = useUserStore();

// 问候时段与日期：挂载/语言切换时计算一次，避免跨午夜渲染抖动。
// vue-i18n 命名插值是单花括号（{name}），文案键已按此书写。
const greetingKey = computed(() => {
  const hour = new Date().getHours();
  if (hour < 6) return 'night';
  if (hour < 12) return 'morning';
  if (hour < 18) return 'afternoon';
  return 'evening';
});

const displayName = computed(
  () => userStore.userInfo?.nickname || userStore.userInfo?.username || '',
);

const greetingText = computed(() => {
  const base = `page.workspace.greeting.${greetingKey.value}`;
  return displayName.value
    ? $t(`${base}With`, { name: displayName.value })
    : $t(base);
});

const todayText = computed(() => {
  const date = new Intl.DateTimeFormat(preferences.app.locale, {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    weekday: 'long',
  }).format(new Date());
  return $t('page.workspace.today', { date });
});

// 站内信未读数与最近消息：queryKey 与顶栏通知弹层口径一致，缓存天然共享
const unreadQuery = useListUserInbox(
  new PaginationQuery({
    paging: { page: 1, pageSize: 1 },
    formValues: {
      recipient_user_id: userStore.userInfo?.id.toString(),
      status: 'RECEIVED',
    },
    orderBy: ['-created_at'],
  }),
);
const unreadCount = computed(() => unreadQuery.data.value?.total ?? 0);

const inboxRecentQuery = useListUserInbox(
  new PaginationQuery({
    paging: { page: 1, pageSize: 5 },
    formValues: { recipient_user_id: userStore.userInfo?.id.toString() },
    orderBy: ['-created_at'],
  }),
);
const inboxItems = computed(
  () => (inboxRecentQuery.data.value?.items ?? []) as InboxItem[],
);

const sessionsQuery = useListMyOnlineSessions();
const sessionItems = computed(
  () => (sessionsQuery.data.value?.items ?? []) as OnlineSession[],
);
</script>

<template>
  <Page>
    <div class="workspace-page">
      <!-- 欢迎横幅 -->
      <Card class="mb-4">
        <div class="flex items-center gap-4">
          <Avatar :size="48" :src="userStore.userInfo?.avatar">
            {{
              (userStore.userInfo?.nickname || userStore.userInfo?.username || '?').slice(0, 1)
            }}
          </Avatar>
          <div class="min-w-0 flex-1">
            <div class="text-lg font-semibold">{{ greetingText }}</div>
            <div class="mt-1 text-xs opacity-65">{{ todayText }}</div>
          </div>
        </div>
      </Card>

      <!-- 快捷入口 -->
      <Card class="mb-4" :body-style="{ paddingTop: '12px' }">
        <template #title>{{ $t('page.workspace.quickAccess.title') }}</template>
        <Row :gutter="8">
          <Col
            v-for="link in QUICK_LINKS"
            :key="link.path"
            :xs="12"
            :sm="8"
            :lg="6"
          >
            <div
              class="quick-item"
              role="button"
              tabindex="0"
              @click="router.push(link.path)"
              @keydown.enter="router.push(link.path)"
            >
              <IconifyIcon
                :icon="link.icon"
                :width="20"
                :height="20"
                class="quick-item__icon"
              />
              <span class="text-sm">{{ $t(link.labelKey) }}</span>
            </div>
          </Col>
        </Row>
      </Card>

      <Row :gutter="16">
        <!-- 我的站内信 -->
        <Col :xs="24" :lg="12" class="mb-4 lg:mb-0">
          <Card>
            <template #title>{{ $t('page.workspace.myMessages.title') }}</template>
            <template #extra>
              <a-button
                type="link"
                size="small"
                @click="router.push('/internal-message/inbox')"
              >
                {{ $t('page.workspace.myMessages.viewAll') }}
                <template v-if="unreadCount > 0">({{ unreadCount }})</template>
              </a-button>
            </template>
            <Empty
              v-if="inboxRecentQuery.isError.value"
              :description="inboxRecentQuery.error.value?.message"
            />
            <Empty
              v-else-if="inboxItems.length === 0"
              :description="$t('page.workspace.myMessages.empty')"
            />
            <div v-else>
              <div
                v-for="item in inboxItems"
                :key="item.id"
                class="list-row cursor-pointer"
                @click="router.push('/internal-message/inbox')"
              >
                <span
                  class="list-row__dot"
                  :class="{ 'list-row__dot--unread': item.status === 'RECEIVED' }"
                ></span>
                <span
                  class="list-row__title"
                  :class="{ 'list-row__title--unread': item.status === 'RECEIVED' }"
                >
                  {{ item.title || '-' }}
                </span>
                <span class="list-row__time">
                  {{ formatDateTime(item.createdAt ?? '') }}
                </span>
              </div>
            </div>
          </Card>
        </Col>

        <!-- 我的活跃会话 -->
        <Col :xs="24" :lg="12">
          <Card>
            <template #title>{{ $t('page.workspace.mySessions.title') }}</template>
            <template #extra>
              <a-button
                type="link"
                size="small"
                @click="router.push('/system/online-sessions')"
              >
                {{ $t('page.workspace.mySessions.manage') }}
              </a-button>
            </template>
            <Empty
              v-if="sessionsQuery.isError.value"
              :description="sessionsQuery.error.value?.message"
            />
            <Empty
              v-else-if="sessionItems.length === 0"
              :description="$t('page.workspace.mySessions.empty')"
            />
            <div v-else>
              <div
                v-for="item in sessionItems"
                :key="item.jti"
                class="list-row"
              >
                <div class="flex min-w-0 flex-1 items-center gap-2">
                  <Tag :color="item.clientType === 'app' ? 'purple' : 'blue'">
                    {{
                      item.clientType === 'app'
                        ? $t('page.onlineSession.clientApp')
                        : $t('page.onlineSession.clientAdmin')
                    }}
                  </Tag>
                  <Tag v-if="item.current" color="success">
                    {{ $t('page.onlineSession.currentSession') }}
                  </Tag>
                  <span class="truncate text-sm">{{ item.ipAddress || '-' }}</span>
                </div>
                <span class="list-row__time">
                  {{ formatDateTime(item.loginAt ?? '') }}
                </span>
              </div>
            </div>
          </Card>
        </Col>
      </Row>
    </div>
  </Page>
</template>

<style scoped>
.workspace-page {
  width: 100%;
}

.quick-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 16px;
  margin-bottom: 8px;
  cursor: pointer;
  border-radius: 8px;
  transition: background 0.2s;
  background: hsl(var(--accent)) /* 主题 accent 淡底，明暗两态自适应 */;

  &:hover {
    background: hsl(var(--accent-dark));
  }

  &__icon {
    flex-shrink: 0;
    color: hsl(var(--primary));
  }
}

.list-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 9px 4px;
  border-bottom: 1px solid hsl(var(--border)) /* 分隔线走主题边框变量 */;

  &:last-child {
    border-bottom: none;
  }

  &__dot {
    flex-shrink: 0;
    width: 6px;
    height: 6px;
    background: transparent;
    border-radius: 50%;

    &--unread {
      background: hsl(var(--primary));
    }
  }

  &__title {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;

    &--unread {
      font-weight: 600;
    }
  }

  &__time {
    flex-shrink: 0;
    font-size: 12px;
    opacity: 0.65;
  }
}
</style>
