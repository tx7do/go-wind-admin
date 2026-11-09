<script lang="ts" setup>
import type { Component } from 'vue';

import { computed, ref } from 'vue';

import { Page } from '@vben/common-ui';
import { $t } from '@vben/locales';
import { useUserStore } from '@vben/stores';

import AccountBindPage from './account-bind-page.vue';
import BaseSettingPage from './base-setting-page.vue';
import EditPasswordPage from './edit-password-page.vue';
import MySessionsPage from './my-sessions-page.vue';
import NotificationPreferencePage from './notification-preference-page.vue';
import SecureSettingPage from './secure-setting-page.vue';
import TenantUsagePanel from './tenant-usage-panel.vue';

const settingList: { component: Component; key: string; name: string }[] = [
  {
    key: '1',
    name: $t('page.user.profile.tab.basicSettings'),
    component: BaseSettingPage,
  },
  {
    key: '2',
    name: $t('page.user.profile.tab.editPassword'),
    component: EditPasswordPage,
  },
  {
    key: '3',
    name: $t('page.user.profile.tab.securitySettings'),
    component: SecureSettingPage,
  },
  {
    key: '4',
    name: $t('page.user.profile.tab.accountBind'),
    component: AccountBindPage,
  },
  {
    key: '5',
    name: $t('page.user.profile.tab.activeSessions'),
    component: MySessionsPage,
  },
  {
    key: '6',
    name: $t('page.user.profile.tab.notification'),
    component: NotificationPreferencePage,
  },
];

const activeKey = ref('1');

// 套餐用量：仅租户用户可见（平台账号 tenantId 空 无套餐语义；ID 已是字符串）
const userStore = useUserStore();
const visibleSettingList = computed(() =>
  !!userStore.userInfo?.tenantId
    ? [
        ...settingList,
        {
          key: '7',
          name: $t('page.user.profile.tab.tenantUsage'),
          component: TenantUsagePanel,
        },
      ]
    : settingList,
);

// 子页（安全设置/账号绑定）通过 switch-tab 事件跳转到对应设置页
const handleSwitchTab = (key: string) => {
  activeKey.value = key;
};
</script>

<template>
  <Page auto-content-height>
    <a-card>
      <a-tabs
        v-model:active-key="activeKey"
        tab-position="left"
        :tab-bar-style="{ width: '220px' }"
      >
        <template v-for="item in visibleSettingList" :key="item.key">
          <a-tab-pane :tab="item.name">
            <div class="pane-title">{{ item.name }}</div>
            <component :is="item.component" @switch-tab="handleSwitchTab" />
          </a-tab-pane>
        </template>
      </a-tabs>
    </a-card>
  </Page>
</template>

<style lang="less" scoped>
.pane-title {
  margin-bottom: 24px;
  font-size: 18px;
  font-weight: 600;
}
</style>
