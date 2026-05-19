<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ElCard :bordered="false" class="profile-card">
      <template #header>
        <div class="card-header">
          {{ $t("pages.user.profile.tab.accountBind") }}
        </div>
      </template>

      <ElList>
        <ElListItem v-for="item in accountBindList" :key="item.key">
          <template #default>
            <div class="list-item-content">
              <div class="item-left">
                <ElIcon :size="40" :color="item.color" class="item-avatar">
                  <component :is="getIconComponent(item.avatar)" />
                </ElIcon>
                <div class="item-info">
                  <span class="item-title">{{ item.title }}</span>
                  <ElButton
                    v-if="item.extra"
                    type="primary"
                    link
                    size="small"
                    :disabled="item.disabled"
                    class="item-btn"
                  >
                    {{ item.extra }}
                  </ElButton>
                </div>
              </div>
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
import { ref } from "vue";
import { ElIcon } from "element-plus";
import { $t } from "@/i18n";

// Iconify 图标映射（实际使用时需要安装对应的图标库）
const iconMap: Record<string, any> = {
  "ri:mail-fill": "Mail",
  "ri:smartphone-fill": "Phone",
  "fa-brands:github": "GitHub",
  "ri:wechat-fill": "ChatDotRound",
  "ri:weibo-fill": "Service",
  "ri:dingding-fill": "Service",
  "ri:qq-fill": "Service",
  "ri:alipay-fill": "Service",
  "ri:google-fill": "Service",
  "ri:apple-fill": "Service",
  "ri:twitter-fill": "Service",
};

interface AccountBindItem {
  key: string;
  title: string;
  description: string;
  extra: string;
  avatar: string;
  color: string;
  status: "bound" | "pending" | "unbound";
  boundTime?: string;
  isPrimary?: boolean;
  disabled?: boolean;
  platform?: string;
  required?: boolean;
}

// 动态标题和描述，使用国际化
const accountBindList = ref<AccountBindItem[]>([
  {
    key: "email",
    title: $t("pages.user.accountBind.email"),
    description: $t("pages.user.accountBind.emailDesc"),
    extra: $t("common.button.edit"),
    avatar: "ri:mail-fill",
    color: "#5470c6",
    status: "bound",
    boundTime: "2023-09-01T10:00:00Z",
    isPrimary: true,
    required: true,
    platform: "email",
  },
  {
    key: "phone",
    title: $t("pages.user.accountBind.phone"),
    description: $t("pages.user.accountBind.phoneDesc"),
    extra: $t("common.button.edit"),
    avatar: "ri:smartphone-fill",
    color: "#722ed1",
    status: "bound",
    boundTime: "2023-09-01T10:05:00Z",
    isPrimary: true,
    required: true,
    platform: "phone",
  },
  {
    key: "github",
    title: $t("pages.user.accountBind.github"),
    description: $t("pages.user.accountBind.githubDesc"),
    extra: $t("pages.user.accountBind.manage"),
    avatar: "fa-brands:github",
    color: "#333",
    status: "bound",
    boundTime: "2023-10-15T09:20:00Z",
    isPrimary: true,
    disabled: true,
    platform: "github",
  },
  {
    key: "wechat",
    title: $t("pages.user.accountBind.wechat"),
    description: $t("pages.user.accountBind.wechatDesc"),
    extra: $t("pages.user.accountBind.unbind"),
    avatar: "ri:wechat-fill",
    color: "#2dc26b",
    status: "bound",
    boundTime: "2024-01-20T14:35:00Z",
    isPrimary: false,
    platform: "wechat",
  },
  {
    key: "weibo",
    title: $t("pages.user.accountBind.weibo"),
    description: $t("pages.user.accountBind.weiboDesc"),
    extra: $t("pages.user.accountBind.bindNow"),
    avatar: "ri:weibo-fill",
    color: "#e6162d",
    status: "unbound",
    platform: "weibo",
  },
  {
    key: "dingtalk",
    title: $t("pages.user.accountBind.dingtalk"),
    description: $t("pages.user.accountBind.dingtalkDesc"),
    extra: $t("pages.user.accountBind.bind"),
    avatar: "ri:dingding-fill",
    color: "#2eabff",
    status: "unbound",
    platform: "dingtalk",
  },
]);

// 获取图标组件
function getIconComponent(iconName: string) {
  return iconMap[iconName] || "Service";
}
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
  width: 100%;
}

.item-left {
  display: flex;
  align-items: center;
  width: 100%;
}

.item-avatar {
  margin-right: 16px;
  flex-shrink: 0;
}

.item-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  justify-content: center;
  min-width: 0;
}

.item-title {
  display: block;
  margin-bottom: 4px;
}

.item-btn {
  padding: 0;
  height: auto;
  line-height: 1;
}

.item-description {
  color: var(--el-text-color-secondary);
  font-size: 14px;
  padding-top: 4px;
}
</style>
