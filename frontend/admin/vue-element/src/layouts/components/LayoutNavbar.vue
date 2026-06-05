<template>
  <div class="navbar">
    <!-- ==================== 左侧区域 ==================== -->
    <div class="navbar__left">
      <!-- 侧边栏显示/隐藏按钮 -->
      <div v-if="widget.sidebarToggle" class="navbar-action" @click="toggleSidebarVisibility">
        <SvgIcon icon="collapse" :class="{ 'is-active': isSidebarEnabled }" />
      </div>

      <!-- 刷新内容区按钮 -->
      <div v-if="widget.refresh" class="navbar-action" @click="handleRefresh">
        <SvgIcon icon="refresh" :class="{ 'is-spin': contentRefreshing }" />
      </div>

      <!-- 面包屑 -->
      <LayoutBreadcrumb />
    </div>

    <!-- ==================== 右侧区域 ==================== -->
    <div class="navbar__right">
      <!-- 全局搜索 -->
      <div v-if="widget.globalSearch" class="navbar-action navbar-action--search">
        <CommandPalette />
      </div>

      <!-- 偏好设置 -->
      <div
        v-if="preferences.app.enablePreferences"
        class="navbar-action"
        @click="handleSettingsClick"
      >
        <SvgIcon icon="setting" />
      </div>

      <!-- 主题切换 -->
      <div v-if="widget.themeToggle" class="navbar-action navbar-action--icon">
        <ThemeSwitch />
      </div>

      <!-- 语言切换 -->
      <div v-if="widget.languageToggle" class="navbar-action">
        <LangSelect />
      </div>

      <!-- 全屏 -->
      <div v-if="widget.fullscreen" class="navbar-action">
        <LayoutFullscreen />
      </div>

      <!-- 通知 -->
      <div v-if="widget.notification" class="navbar-action">
        <NoticeDropdown />
      </div>

      <!-- 用户头像下拉 -->
      <div class="navbar-action navbar-action--profile">
        <el-dropdown trigger="click">
          <div class="user-profile">
            <img
              :src="userStore.userInfo?.avatar || '/default-avatar.png'"
              class="user-profile__avatar"
            />
            <span class="user-profile__name">{{ userStore.userInfo?.username || "" }}</span>
          </div>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item @click="handleProfileClick">
                {{ t("common.navbar.profile") }}
              </el-dropdown-item>
              <el-dropdown-item divided @click="logout">
                {{ t("common.navbar.logout") }}
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";

import { useAppUserStore } from "@/stores";
import { useAuth } from "@/composables/use-auth";
import { preferences, preferencesManager } from "@/core/preferences";
import SvgIcon from "@/components/SvgIcon/index.vue";

import LayoutBreadcrumb from "./LayoutBreadcrumb.vue";
import CommandPalette from "@/components/CommandPalette/index.vue";
import ThemeSwitch from "@/components/ThemeSwitch/index.vue";
import LangSelect from "@/components/LangSelect/index.vue";
import LayoutFullscreen from "./LayoutFullscreen.vue";
import NoticeDropdown from "@/components/NoticeDropdown/index.vue";

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const userStore = useAppUserStore();
const authStore = useAuth();

// 偏好设置引用
const widget = computed(() => preferences.widget);

// 注入设置面板可见性状态
const settingsVisible = inject<Ref<boolean>>("settingsVisible", ref(false));

// 注入内容区刷新状态
const contentRefreshing = inject<Ref<boolean>>("contentRefreshing", ref(false));

// 注入刷新 key
const contentRefreshKey = inject<Ref<number>>("contentRefreshKey", ref(0));

// 侧边栏是否启用
const isSidebarEnabled = computed(() => preferences.sidebar.enable);

// ==================== 方法 ====================

/** 切换侧边栏显示/隐藏 */
function toggleSidebarVisibility() {
  preferencesManager.updatePreferences({
    sidebar: { enable: !preferences.sidebar.enable },
  });
}

/** 刷新内容区 */
function handleRefresh() {
  if (contentRefreshing.value) return;
  contentRefreshing.value = true;
  // 递增 key 强制组件销毁重建（router-view 保持挂载，不会触发 ResizeObserver 错误）
  contentRefreshKey.value++;
  nextTick(() => {
    contentRefreshing.value = false;
  });
}

/** 打开个人中心 */
function handleProfileClick() {
  router.push({ name: "UserProfile" });
}

/** 退出登录 */
function logout() {
  ElMessageBox.confirm(
    t("common.navbar.logoutConfirmMessage"),
    t("common.navbar.logoutConfirmTitle"),
    {
      confirmButtonText: t("common.button.confirm"),
      cancelButtonText: t("common.button.cancel"),
      type: "warning",
      lockScroll: false,
    }
  ).then(() => {
    authStore.logout().then(() => {
      router.push(`/login?redirect=${route.fullPath}`);
    });
  });
}

/** 打开偏好设置面板 */
function handleSettingsClick() {
  settingsVisible.value = true;
}
</script>

<style lang="scss" scoped>
.navbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: $navbar-height;
  background-color: var(--navbar-background);
  border-bottom: 1px solid var(--navbar-border-color);

  &__left {
    display: flex;
    align-items: center;
    flex: 1;
    min-width: 0;
  }

  &__right {
    display: flex;
    align-items: center;
    flex-shrink: 0;
  }
}

// ==================== 通用操作按钮 ====================
.navbar-action {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 40px;
  height: 32px; // 与面包屑高度匹配
  padding: 0 8px;
  cursor: pointer;
  border-radius: 8px;
  transition: all 0.2s ease;
  background: transparent;

  > * {
    display: flex;
    align-items: center;
    justify-content: center;
  }

  :deep(.el-dropdown),
  :deep(.el-tooltip) {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 100%;
    height: 40px;
  }

  :deep(.svg-local-icon) {
    font-size: 18px;
    line-height: 1;
    color: var(--el-text-color-regular);
    transition: all 0.2s ease;
  }

  // hover 态：柔和背景 + 图标主色高亮 + 轻微放大
  &:hover {
    background: rgba(0, 0, 0, 0.04);
    transform: scale(1.05);

    :deep(.svg-local-icon) {
      color: var(--el-color-primary);
    }
  }

  // 搜索按钮特殊处理：不放大、不显示背景，让内部胶囊条自行处理 hover
  &--search {
    min-width: auto;
    padding: 0;
    margin-right: 4px;
    background: transparent !important;
    transform: none !important;
    box-shadow: none !important;

    &:hover {
      background: transparent;
      transform: none;
      box-shadow: none;

      :deep(.svg-local-icon) {
        color: var(--el-text-color-secondary);
      }
    }
  }

  // 用户头像区域：不放大，仅显示柔和背景
  &--profile {
    min-width: auto;
    padding: 0 4px;
    transform: none !important;

    &:hover {
      background: rgba(0, 0, 0, 0.04);
      transform: none;
    }
  }

  // 暗黑模式 hover
  html.dark & {
    :deep(.svg-local-icon) {
      color: rgba(255, 255, 255, 0.75);
    }

    &:hover {
      background: rgba(255, 255, 255, 0.08);

      :deep(.svg-local-icon) {
        color: #ffffff;
      }
    }
  }

  // 半深色顶栏 hover
  html.semi-dark-header & {
    :deep(.svg-local-icon) {
      color: rgba(255, 255, 255, 0.75);
    }

    &:hover {
      background: rgba(255, 255, 255, 0.08);

      :deep(.svg-local-icon) {
        color: #ffffff;
      }
    }
  }

  // 折叠按钮激活状态
  .svg-local-icon--collapse {
    transition: transform 0.2s ease;
    transform: scaleX(-1);

    &.is-active {
      transform: scaleX(1);
    }
  }

  // 刷新按钮旋转动画
  .svg-local-icon--refresh {
    transition: transform 0.2s ease;

    &.is-spin {
      animation: spin 0.6s linear;
    }
  }
}

// ==================== 用户头像 ====================
.user-profile {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 40px;
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

  &__name {
    margin-left: 8px;
    font-size: 14px;
    color: var(--el-text-color-regular);
    white-space: nowrap;
    transition: color 0.2s ease;
  }

  // 暗黑模式
  html.dark & {
    &__name {
      color: rgba(255, 255, 255, 0.85);
    }
  }

  // 半深色顶栏
  html.semi-dark-header & {
    &__name {
      color: rgba(255, 255, 255, 0.85);
    }
  }
}

// 旋转动画
@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}
</style>