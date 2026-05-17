<script lang="ts" setup>
import { Page } from '@vben/common-ui';
import { $t } from '@vben/locales';

import AccountBindPage from './account-bind-page.vue';
import BaseSettingPage from './base-setting-page.vue';
import EditPasswordPage from './edit-password-page.vue';
import MsgNotifyPage from './msg-notify-page.vue';
import SecureSettingPage from './secure-setting-page.vue';

const settingList = [
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
    name: $t('page.user.profile.tab.notification'),
    component: MsgNotifyPage,
  },
];
</script>

<template>
  <Page auto-content-height>
    <a-card>
      <a-tabs
        tab-position="left"
        :tab-bar-style="{ width: '220px' }"
        class="edge-tabs"
      >
        <template v-for="item in settingList" :key="item.key">
          <a-tab-pane :tab="item.name">
            <component :is="item.component" />
          </a-tab-pane>
        </template>
      </a-tabs>
    </a-card>
  </Page>
</template>

<style lang="less">
.edge-tabs {
  margin: 0;
}

/* 使用 Vue scoped 的深度选择器，覆盖 Antd 的默认内边距/外边距 */
.edge-tabs,
::v-deep(.ant-tabs-content, .ant-tabs-content-holder, .ant-tabs-tabpane) {
  padding: 0 !important;
  margin: 0 !important;
  box-sizing: border-box;
}
</style>
