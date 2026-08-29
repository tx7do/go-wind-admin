<script lang="ts" setup>
import type { Component } from 'vue';

import { ref } from 'vue';

import { Page } from '@vben/common-ui';
import { $t } from '@vben/locales';

import AccountBindPage from './account-bind-page.vue';
import BaseSettingPage from './base-setting-page.vue';
import EditPasswordPage from './edit-password-page.vue';
import SecureSettingPage from './secure-setting-page.vue';

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
];

// 消息通知 tab 已移除：后端暂无用户通知偏好能力，原页为模板演示数据。

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
        class="edge-tabs"
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
