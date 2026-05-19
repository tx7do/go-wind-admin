<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ElCard :bordered="false" class="profile-card">
      <template #header>
        <div class="card-header">
          {{ $t("pages.user.profile.tab.securitySettings") }}
        </div>
      </template>

      <ElList>
        <ElListItem v-for="item in secureSettingList" :key="item.key">
          <template #default>
            <div class="list-item-content">
              <span class="item-title">{{ item.title }}</span>
              <ElLink v-if="item.extra" type="primary" :underline="false" class="item-link">
                {{ item.extra }}
              </ElLink>
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
import { $t } from "@/i18n";

interface SettingItem {
  key: string;
  title: string;
  description: string;
  extra?: string;
}

const secureSettingList: SettingItem[] = [
  {
    key: "1",
    title: $t("pages.user.secureSetting.accountPassword"),
    description: $t("pages.user.secureSetting.accountPasswordDesc"),
    extra: $t("common.button.edit"),
  },
  {
    key: "2",
    title: $t("pages.user.secureSetting.securePhone"),
    description: $t("pages.user.secureSetting.securePhoneDesc"),
    extra: $t("common.button.edit"),
  },
  {
    key: "3",
    title: $t("pages.user.secureSetting.secureQuestion"),
    description: $t("pages.user.secureSetting.secureQuestionDesc"),
    extra: $t("common.button.edit"),
  },
  {
    key: "4",
    title: $t("pages.user.secureSetting.backupEmail"),
    description: $t("pages.user.secureSetting.backupEmailDesc"),
    extra: $t("common.button.edit"),
  },
  {
    key: "5",
    title: $t("pages.user.secureSetting.mfaDevice"),
    description: $t("pages.user.secureSetting.mfaDeviceDesc"),
    extra: $t("common.button.edit"),
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
}

.item-link {
  flex-shrink: 0;
}

.item-description {
  color: var(--el-text-color-secondary);
  font-size: 14px;
  padding-top: 4px;
}
</style>
