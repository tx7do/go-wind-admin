<script lang="ts" setup>
import type { Component } from 'vue';

import { ref } from 'vue';

import { Page } from '@vben/common-ui';
import { $t } from '@vben/locales';

import AccountBindPage from './account-bind-page.vue';
import BaseSettingPage from './base-setting-page.vue';
import EditPasswordPage from './edit-password-page.vue';
import NotificationPreferencePage from './notification-preference-page.vue';
import SecureSettingPage from './secure-setting-page.vue';
import MySessionsPage from './my-sessions-page.vue';

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

// 「消息通知」tab（P3）：后端已有用户通知偏好能力，见 notification-preference-page.vue

const activeKey = ref('1');

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
        <template v-for="item in settingList" :key="item.key">
          <a-tab-pane :tab="item.name">
            <component :is="item.component" @switch-tab="handleSwitchTab" />
          </a-tab-pane>
        </template>
      </a-tabs>
    </a-card>
  </Page>
</template>
