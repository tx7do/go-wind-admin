<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ElCard :bordered="false" class="profile-card">
      <template #header>
        <div class="card-header">
          {{ $t("pages.user.profile.tab.notification") }}
        </div>
      </template>

      <ElList>
        <ElListItem v-for="item in msgNotifyList" :key="item.key">
          <template #default>
            <div class="list-item-content">
              <span class="item-title">{{ item.title }}</span>
              <ElSwitch
                v-model="item.checked"
                :active-text="$t('common.switch.active')"
                :inactive-text="$t('common.switch.inactive')"
                class="item-switch"
              />
            </div>
          </template>
          <template #description>
            <div class="item-description">{{ item.description }}</div>
          </template>
        </ElListItem>
      </ElList>
    </ElCard>
  </div>
</template>

<script lang="ts" setup>
import { reactive } from "vue";

import { $t } from "@/i18n";

interface NotifyItem {
  key: string;
  title: string;
  description: string;
  checked: boolean;
}

const msgNotifyList = reactive<NotifyItem[]>([
  {
    key: "1",
    title: $t("pages.user.msgNotify.accountPassword"),
    description: $t("pages.user.msgNotify.accountPasswordDesc"),
    checked: true,
  },
  {
    key: "2",
    title: $t("pages.user.msgNotify.systemMessage"),
    description: $t("pages.user.msgNotify.systemMessageDesc"),
    checked: true,
  },
  {
    key: "3",
    title: $t("pages.user.msgNotify.todoTask"),
    description: $t("pages.user.msgNotify.todoTaskDesc"),
    checked: true,
  },
]);
</script>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
  width: 100%;
  min-width: 0;
  flex-shrink: 0;
}

.profile-card {
  max-width: 800px;
}

.card-header {
  font-size: 16px;
  font-weight: 500;
}

.list-item-content {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
}

.item-title {
  flex: 1;
  margin-right: 16px;
}

.item-switch {
  flex-shrink: 0;
}

.item-description {
  color: var(--el-text-color-secondary);
  font-size: 14px;
  padding-top: 4px;
}
</style>
