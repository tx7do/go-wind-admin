<template>
  <div class="page-container">
    <div class="setting-list">
      <!-- 登录密码：跳转“修改密码”页 -->
      <div class="setting-item">
        <div class="setting-item-content">
          <span class="item-title">{{ $t("pages.user.secureSetting.accountPassword") }}</span>
          <ElLink type="primary" underline="never" class="item-link" @click="emit('switchTab', '2')">
            {{ $t("pages.user.secureSetting.modify") }}
          </ElLink>
        </div>
        <div class="item-description">
          {{ $t("pages.user.secureSetting.accountPasswordDesc") }}
        </div>
      </div>

      <!-- MFA（TOTP）绑定/解绑：真实功能组件，绑定后登录需二次验证 -->
      <div class="setting-item">
        <MfaManagement />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import MfaManagement from "./MfaManagement.vue";

// 密保问题/备用邮箱等演示条目已移除：后端无对应能力。
// 手机/邮箱绑定状态展示在“账号绑定”页；修改密码入口跳转对应 tab。
const emit = defineEmits<{ switchTab: [key: string] }>();
</script>

<style lang="scss" scoped>
.page-container {
  width: 100%;
  max-width: 800px;
}

.setting-list {
  padding-top: 20px;
}

.setting-item {
  padding: 16px 0;
  border-bottom: 1px solid var(--el-border-color-lighter);

  &:last-child {
    border-bottom: none;
  }
}

.setting-item-content {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  margin-bottom: 8px;
}

.item-title {
  flex: 1;
  font-size: 14px;
  font-weight: 500;
}

.item-link {
  flex-shrink: 0;
}

.item-description {
  color: var(--el-text-color-secondary);
  font-size: 13px;
  line-height: 1.5;
}
</style>
