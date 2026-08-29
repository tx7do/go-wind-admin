<template>
  <div class="page-container">
    <div class="account-list">
      <div v-for="item in bindList" :key="item.key" class="account-item">
        <div class="account-item-content">
          <!-- 左侧：图标和标题 -->
          <div class="item-left">
            <IconifyIcon
              :icon="item.avatar"
              :width="28"
              :height="28"
              color="var(--el-text-color-regular)"
              class="item-avatar"
            />
            <div class="item-info">
              <span class="item-title">{{ item.title }}</span>
              <span class="item-description">{{ item.description }}</span>
            </div>
          </div>
          <!-- 右侧：跳转基本设置修改 -->
          <ElLink type="primary" underline="never" class="item-link" @click="emit('switchTab', '1')">
            {{ $t("pages.user.accountBind.modify") }}
          </ElLink>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { Icon as IconifyIcon } from "@iconify/vue";

import { useGetUserProfile } from "@/api/composables";
import { $t } from "@/core/i18n";

// 第三方账号（GitHub/微信/微博等）绑定条目已移除：
// 后端社交登录未实现，BindContact/VerifyContact 换绑接口亦为占位实现。
// 邮箱/手机展示当前账号的真实绑定状态，修改入口跳转基本设置
// （该页经 UpdateUser 直接更新联系方式，为现行产品行为）。
const emit = defineEmits<{ switchTab: [key: string] }>();

const { data: user } = useGetUserProfile();

const maskEmail = (email: string) => {
  const at = email.lastIndexOf("@");
  if (at <= 0) return email;
  return `${email.slice(0, Math.min(2, at))}***${email.slice(at)}`;
};

const maskPhone = (phone: string) => {
  if (phone.length < 7) return phone;
  return `${phone.slice(0, 3)}****${phone.slice(-4)}`;
};

interface BindItem {
  key: string;
  title: string;
  description: string;
  avatar: string;
}

const bindList = computed<BindItem[]>(() => {
  const email = user.value?.email ?? "";
  const mobile = user.value?.mobile ?? "";

  return [
    {
      key: "email",
      title: $t("pages.user.accountBind.email"),
      description: email
        ? $t("pages.user.accountBind.bound", { value: maskEmail(email) })
        : $t("pages.user.accountBind.unbound"),
      avatar: "ri:mail-fill",
    },
    {
      key: "phone",
      title: $t("pages.user.accountBind.phone"),
      description: mobile
        ? $t("pages.user.accountBind.bound", { value: maskPhone(mobile) })
        : $t("pages.user.accountBind.unbound"),
      avatar: "ri:smartphone-fill",
    },
  ];
});
</script>

<style lang="scss" scoped>
.page-container {
  width: 100%;
  max-width: 800px;
}

.account-list {
  padding-top: 20px;
}

.account-item {
  padding: 16px 0;
  border-bottom: 1px solid var(--el-border-color-lighter);

  &:last-child {
    border-bottom: none;
  }
}

.account-item-content {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
}

.item-left {
  display: flex;
  align-items: center;
  flex: 1;
  min-width: 0;
}

.item-avatar {
  margin-right: 16px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

.item-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.item-title {
  font-size: 14px;
  font-weight: 500;
  color: var(--el-text-color-primary);
  line-height: 1.4;
}

.item-description {
  font-size: 13px;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
}

.item-link {
  flex-shrink: 0;
  font-size: 14px;
}
</style>
