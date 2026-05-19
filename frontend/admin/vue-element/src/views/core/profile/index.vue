<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ElCard :bordered="false" class="profile-card">
      <ElTabs v-model="activeTab" type="border-card" class="profile-tabs">
        <ElTabPane v-for="item in settingList" :key="item.key" :label="item.name" :name="item.key">
          <component :is="item.component" />
        </ElTabPane>
      </ElTabs>
    </ElCard>
  </div>
</template>

<script lang="ts" setup>
import { ref } from "vue";

import BaseSettingPage from "./base-setting-page.vue";
import EditPasswordPage from "./edit-password-page.vue";
import AccountBindPage from "./account-bind-page.vue";
import MsgNotifyPage from "./msg-notify-page.vue";
import SecureSettingPage from "./secure-setting-page.vue";

import { $t } from "@/i18n";

const activeTab = ref("1");

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
    name: $t("pages.user.profile.tab.notification"),
    component: MsgNotifyPage,
  },
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
}

.profile-tabs {
  min-height: 600px;
}
</style>
