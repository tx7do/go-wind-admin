<template>
  <el-dropdown trigger="click" placement="bottom-end">
    <div class="user-profile">
      <img :src="avatar" class="user-profile__avatar" />
      <span class="user-profile__name">{{ userStore.userInfo?.username || "" }}</span>
    </div>

    <template #dropdown>
      <div class="user-dropdown">
        <!-- 头部卡：昵称 + 邮箱 + 在线点 -->
        <div class="user-dropdown__header">
          <span class="user-dropdown__avatar-wrap">
            <img :src="avatar" class="user-dropdown__avatar" />
            <span class="user-dropdown__dot" />
          </span>
          <span class="user-dropdown__identity">
            <span class="user-dropdown__name">{{ displayName }}</span>
            <span class="user-dropdown__email">{{ userStore.userInfo?.email }}</span>
          </span>
        </div>

        <button type="button" class="user-dropdown__item" @click="handleProfileClick">
          <SvgIcon icon="lucide:user-round-pen" :size="16" />
          <span class="user-dropdown__label">{{ t("common.navbar.profile") }}</span>
        </button>

        <button type="button" class="user-dropdown__item" @click="handleInboxClick">
          <SvgIcon icon="lucide:inbox" :size="16" />
          <span class="user-dropdown__label">{{ t("common.navbar.inbox") }}</span>
        </button>

        <template v-if="lockScreenEnabled">
          <div class="user-dropdown__separator" />
          <button type="button" class="user-dropdown__item" @click="lockDialogVisible = true">
            <SvgIcon icon="lucide:lock-keyhole" :size="16" />
            <span class="user-dropdown__label">{{ t("common.navbar.lockScreen") }}</span>
            <span v-if="showLockShortcut" class="user-dropdown__shortcut">{{ altView }} L</span>
          </button>
        </template>

        <div class="user-dropdown__separator" />
        <button type="button" class="user-dropdown__item user-dropdown__item--danger" @click="logout">
          <SvgIcon icon="lucide:log-out" :size="16" />
          <span class="user-dropdown__label">{{ t("common.navbar.logout") }}</span>
          <span v-if="showLogoutShortcut" class="user-dropdown__shortcut">{{ altView }} Q</span>
        </button>
      </div>
    </template>
  </el-dropdown>

  <!-- 锁定屏幕：先设一次性解锁密码 -->
  <el-dialog
    v-model="lockDialogVisible"
    :title="t('common.lockScreen.title')"
    width="360px"
    append-to-body
    destroy-on-close
    align-center
  >
    <div class="lock-dialog">
      <img :src="avatar" class="lock-dialog__avatar" />
      <div class="lock-dialog__name">{{ displayName }}</div>
      <el-input
        v-model="lockPassword"
        type="password"
        show-password
        autocomplete="new-password"
        size="large"
        :placeholder="t('common.lockScreen.placeholder')"
      />
      <el-button
        class="lock-dialog__submit"
        type="primary"
        size="large"
        :disabled="!lockPassword"
        @click="handleLock"
      >
        {{ t("common.lockScreen.lock") }}
      </el-button>
    </div>
  </el-dialog>
</template>

<script setup lang="ts">
import { useAppUserStore } from '@go-wind/admin-element';
import { useLockStore } from '@/stores';
/**
 * 右上角用户下拉菜单（三端统一口径：头部卡 / 个人中心 / 收件箱 / 锁定屏幕 / 退出登录）
 *
 * 基准实现见 vue-vben framework/effects/layouts/widgets/user-dropdown。
 * LayoutNavbar（左侧布局）与 LayoutToolbar（Mix/Top 布局）共用本组件，
 * 避免同一份菜单在两处各写一遍后漂移。
 */
import { computed, onUnmounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import { ElMessageBox } from "element-plus";

import SvgIcon from "@/components/SvgIcon/index.vue";
import { preferences, usePreferences } from "@go-wind/admin-element";
import { useAuth } from "@go-wind/admin-element";

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const userStore = useAppUserStore();
const authStore = useAuth();
const lockStore = useLockStore();
const { globalLockScreenShortcutKey, globalLogoutShortcutKey } = usePreferences();

const lockScreenEnabled = computed(() => preferences.widget.lockScreen);
const showLockShortcut = globalLockScreenShortcutKey;
const showLogoutShortcut = globalLogoutShortcutKey;
const altView = computed(() => (/Mac|iPod|iPhone|iPad/.test(navigator.platform) ? "⌥" : "Alt"));

const avatar = computed(
  () => userStore.userInfo?.avatar || preferences.app.defaultAvatar || "/default-avatar.png"
);
const displayName = computed(
  () => userStore.userInfo?.realname || userStore.userInfo?.username || ""
);

const lockDialogVisible = ref(false);
const lockPassword = ref("");

function handleProfileClick() {
  router.push({ name: "UserProfile" });
}

function handleInboxClick() {
  router.push({ name: "InboxPage" });
}

function handleLock() {
  if (!lockPassword.value) return;
  lockStore.lockScreen(lockPassword.value);
  lockDialogVisible.value = false;
  lockPassword.value = "";
}

function logout() {
  ElMessageBox.confirm(t("common.navbar.logoutConfirmMessage"), t("common.navbar.logoutConfirmTitle"), {
    confirmButtonText: t("common.button.confirm"),
    cancelButtonText: t("common.button.cancel"),
    type: "warning",
    lockScroll: false,
  })
    .then(() => {
      authStore.logout().then(() => {
        router.push(`/login?redirect=${route.fullPath}`);
      });
    })
    .catch(() => {
      // 用户取消：ElMessageBox 以 reject(false) 表示，非异常，无需留痕
    });
}

// Alt+L 锁屏 / Alt+Q 退出：与菜单项右侧的键位提示读同一份开关
function onKeydown(event: KeyboardEvent) {
  if (!event.altKey) return;
  if (event.code === "KeyL" && lockScreenEnabled.value && globalLockScreenShortcutKey.value) {
    event.preventDefault();
    lockDialogVisible.value = true;
  } else if (event.code === "KeyQ" && globalLogoutShortcutKey.value) {
    event.preventDefault();
    logout();
  }
}
window.addEventListener("keydown", onKeydown);
onUnmounted(() => window.removeEventListener("keydown", onKeydown));
</script>

<style lang="scss" scoped>
.user-profile {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  padding: 0 4px;
  cursor: pointer;
  border-radius: 8px;
  transition: background-color 0.2s ease;

  &__avatar {
    flex-shrink: 0;
    width: 28px;
    height: 28px;
    overflow: hidden;
    border-radius: 50%;
    object-fit: cover;
    object-position: center;
  }

  // 颜色由父级（navbar-actions--white-text / --dark-text）决定，此处只继承
  &__name {
    margin-left: 8px;
    font-size: 14px;
    color: inherit;
    white-space: nowrap;
  }
}

.user-dropdown {
  min-width: 240px;
  padding: 4px 0 6px;
  background-color: var(--el-bg-color-overlay);
  border-radius: 8px;
  box-shadow: var(--el-box-shadow-light);

  &__header {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 14px 16px;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  &__avatar-wrap {
    position: relative;
    display: inline-flex;
    flex-shrink: 0;
  }

  &__avatar {
    width: 48px;
    height: 48px;
    border-radius: 50%;
    object-fit: cover;
    object-position: center;
  }

  &__dot {
    position: absolute;
    right: 1px;
    bottom: 2px;
    width: 10px;
    height: 10px;
    background-color: var(--el-color-success);
    border: 2px solid var(--el-bg-color-overlay);
    border-radius: 50%;
  }

  &__identity {
    display: flex;
    flex: 1;
    flex-direction: column;
    min-width: 0;
  }

  &__name {
    overflow: hidden;
    font-size: 14px;
    font-weight: 500;
    color: var(--el-text-color-primary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__email {
    overflow: hidden;
    font-size: 12px;
    color: var(--el-text-color-secondary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__separator {
    height: 1px;
    margin: 4px 0;
    background-color: var(--el-border-color-lighter);
  }

  &__item {
    display: flex;
    align-items: center;
    width: calc(100% - 8px);
    padding: 8px 12px;
    margin: 0 4px;
    font-size: 14px;
    color: var(--el-text-color-regular);
    text-align: left;
    cursor: pointer;
    background: none;
    border: none;
    border-radius: 4px;
    transition: all 0.2s ease;

    &:hover {
      color: var(--el-text-color-primary);
      background-color: var(--el-fill-color-light);
    }

    &--danger {
      color: var(--el-color-danger);
    }
  }

  &__label {
    flex: 1;
    margin-left: 8px;
  }

  &__shortcut {
    min-width: 18px;
    padding: 1px 5px;
    font-family: monospace;
    font-size: 11px;
    line-height: 1.4;
    color: var(--el-text-color-secondary);
    background-color: var(--el-fill-color);
    border: 1px solid var(--el-border-color);
    border-radius: 4px;
  }
}

.lock-dialog {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 8px 0 4px;

  &__avatar {
    width: 64px;
    height: 64px;
    border-radius: 50%;
    object-fit: cover;
    object-position: center;
  }

  &__name {
    margin: 16px 0 20px;
    font-size: 14px;
    font-weight: 500;
    color: var(--el-text-color-primary);
  }

  &__submit {
    width: 100%;
    margin-top: 12px;
    margin-left: 0;
  }
}
</style>
