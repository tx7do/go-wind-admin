<template>
  <div class="app-container h-full flex flex-1">
    <ElCard :bordered="false" class="profile-card">
      <ElTabs v-model="activeTab" tab-position="left" class="profile-tabs">
        <ElTabPane
          v-for="item in settingList"
          :key="item.key"
          :name="item.key"
          :label="item.name"
          lazy
        >
          <div class="content-header">
            <h2 class="content-title">{{ item.name }}</h2>
          </div>
          <div class="content-body">
            <component :is="item.component" @switch-tab="handleSwitchTab" />
          </div>
        </ElTabPane>
      </ElTabs>
    </ElCard>
  </div>
</template>

<script lang="ts" setup>
import { ref } from "vue";

import { ElCard, ElTabPane, ElTabs } from "element-plus";

import BaseSettingPage from "./base-setting-page.vue";
import EditPasswordPage from "./edit-password-page.vue";
import AccountBindPage from "./account-bind-page.vue";
import SecureSettingPage from "./secure-setting-page.vue";
import MySessionsPage from "./my-sessions-page.vue";
import NotificationPreferencePage from "./notification-preference-page.vue";
import TenantUsagePanel from "./tenant-usage-panel.vue";
import { useAppUserStore } from "@/stores";

import { $t } from "@/core/i18n";

const activeTab = ref("1");

// 「消息通知」tab（P3）：后端已有用户通知偏好能力，见 notification-preference-page.vue

// 子页（安全设置/账号绑定）通过 switch-tab 事件跳转到对应设置页
const handleSwitchTab = (key: string) => {
  activeTab.value = key;
};

const settingList = [
  {
    key: "1",
    name: $t("pages.user.profile.tab.basicSettings"),
    component: BaseSettingPage,
  },
  {
    key: "2",
    name: $t("pages.user.profile.tab.editPassword"),
    component: EditPasswordPage,
  },
  {
    key: "3",
    name: $t("pages.user.profile.tab.securitySettings"),
    component: SecureSettingPage,
  },
  {
    key: "4",
    name: $t("pages.user.profile.tab.accountBind"),
    component: AccountBindPage,
  },
  {
    key: "5",
    name: $t("pages.user.profile.tab.activeSessions"),
    component: MySessionsPage,
  },
  {
    key: "6",
    name: $t("pages.user.profile.tab.notification"),
    component: NotificationPreferencePage,
  },
  // 套餐用量：仅租户用户展示（平台用户 tenantId 为 "0"/空，无套餐语义）
  ...(!!useAppUserStore().userInfo?.tenantId && useAppUserStore().userInfo?.tenantId !== "0"
    ? [{
        key: "7",
        name: $t("pages.user.profile.tab.tenantUsage"),
        component: TenantUsagePanel,
      }]
    : []),
];
</script>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
  width: 100%;
  min-width: 0;
  flex-shrink: 0;
}

.profile-card {
  height: 100%;
  flex: 1;
  :deep(.el-card__body) {
    height: 100%;
    display: flex;
  }
}

.profile-tabs {
  height: 100%;
  flex: 1;
  display: flex;

  // 左页签栏宽度对齐另两端（vben 与 react 均为 220px）
  :deep(.el-tabs__header.is-left) {
    width: 220px;
    margin-right: 0;
  }

  :deep(.el-tabs__nav-wrap) {
    padding-right: 16px;
  }

  :deep(.el-tabs__content) {
    flex: 1;
    padding: 8px 24px 0 0;
    overflow: hidden;
  }

  :deep(.el-tab-pane) {
    height: 100%;
    display: flex;
    flex-direction: column;
  }
}

.content-header {
  margin-bottom: 24px;

  .content-title {
    margin: 0;
    font-size: 18px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }
}

.content-body {
  min-height: 400px;
  overflow-y: auto;
}
</style>
