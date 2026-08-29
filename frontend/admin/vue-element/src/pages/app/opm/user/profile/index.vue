<template>
  <div class="app-container h-full flex flex-1">
    <ElCard :bordered="false" class="profile-card">
      <div class="profile-layout">
        <!-- 左侧导航菜单 -->
        <div class="profile-sidebar">
          <div
            v-for="item in settingList"
            :key="item.key"
            class="sidebar-item"
            :class="{ active: activeTab === item.key }"
            @click="activeTab = item.key"
          >
            {{ item.name }}
          </div>
        </div>

        <!-- 右侧内容区域 -->
        <div class="profile-content">
          <div class="content-header">
            <h2 class="content-title">
              {{ settingList.find((item) => item.key === activeTab)?.name }}
            </h2>
          </div>
          <div class="content-body">
            <component
              :is="settingList.find((item) => item.key === activeTab)?.component"
              @switch-tab="handleSwitchTab"
            />
          </div>
        </div>
      </div>
    </ElCard>
  </div>
</template>

<script lang="ts" setup>
import { ref } from "vue";

import BaseSettingPage from "./base-setting-page.vue";
import EditPasswordPage from "./edit-password-page.vue";
import AccountBindPage from "./account-bind-page.vue";
import SecureSettingPage from "./secure-setting-page.vue";

import { $t } from "@/core/i18n";

const activeTab = ref("1");

// 消息通知 tab 已移除：后端暂无用户通知偏好能力，原页为模板演示数据。

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
}

.profile-layout {
  display: flex;
  height: 100%;
}

// 左侧导航菜单
.profile-sidebar {
  width: 180px;
  flex-shrink: 0;
  border-right: 1px solid var(--el-border-color-lighter);
  padding: 16px 0;

  .sidebar-item {
    padding: 12px 20px;
    cursor: pointer;
    font-size: 14px;
    color: var(--el-text-color-regular);
    transition: all 0.3s;
    position: relative;

    &:hover {
      background-color: var(--el-fill-color-light);
      color: var(--el-color-primary);
    }

    &.active {
      color: var(--el-color-primary);
      font-weight: 500;

      &::after {
        content: "";
        position: absolute;
        right: 0;
        top: 50%;
        transform: translateY(-50%);
        width: 3px;
        height: 24px;
        background-color: var(--el-color-primary);
        border-radius: 2px;
      }
    }
  }
}

// 右侧内容区域
.profile-content {
  flex: 1;
  overflow-y: auto;
  padding: 20px 24px;
}

.content-header {
  margin-bottom: 24px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--el-border-color-lighter);

  .content-title {
    margin: 0;
    font-size: 18px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }
}

.content-body {
  min-height: 400px;
}
</style>
