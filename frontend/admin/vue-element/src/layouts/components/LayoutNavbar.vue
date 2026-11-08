<template>
  <div class="navbar">
    <!-- ==================== 左侧区域 ==================== -->
    <div class="navbar__left">
      <!-- 侧边栏显示/隐藏按钮 -->
      <ElTooltip :content="$t('common.header.sidebar')" placement="bottom">
              <div v-if="widget.sidebarToggle" class="navbar-action" @click="toggleSidebarVisibility">
        <SvgIcon icon="collapse" :class="{ 'is-active': isSidebarEnabled }" />
      </div>
      </ElTooltip>

      <!-- 刷新内容区按钮 -->
      <ElTooltip :content="$t('common.header.refresh')" placement="bottom">
              <div v-if="widget.refresh" class="navbar-action" @click="handleRefresh">
        <SvgIcon icon="refresh" :class="{ 'is-spin': contentRefreshing }" />
      </div>
      </ElTooltip>

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
      <ElTooltip :content="$t('common.header.preferences')" placement="bottom">
              <div
        v-if="preferences.app.enablePreferences"
        class="navbar-action"
        @click="handleSettingsClick"
      >
        <SvgIcon icon="setting" />
      </div>
      </ElTooltip>

      <!-- 主题切换 -->
      <ElTooltip :content="$t('common.header.theme')" placement="bottom">
              <div v-if="widget.themeToggle" class="navbar-action navbar-action--icon">
        <ThemeSwitch />
      </div>
      </ElTooltip>

      <!-- 语言切换 -->
      <ElTooltip :content="$t('common.header.language')" placement="bottom">
              <div v-if="widget.languageToggle" class="navbar-action">
        <LangSelect />
      </div>
      </ElTooltip>

      <!-- 全屏 -->
      <ElTooltip :content="$t('common.header.fullscreen')" placement="bottom">
              <div v-if="widget.fullscreen" class="navbar-action">
        <LayoutFullscreen />
      </div>
      </ElTooltip>

      <!-- 通知 -->
      <ElTooltip :content="$t('common.header.notification')" placement="bottom">
              <div v-if="widget.notification" class="navbar-action">
        <NoticeDropdown />
      </div>
      </ElTooltip>

      <!-- 用户头像下拉 -->
      <div class="navbar-action navbar-action--profile">
        <UserDropdown />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { preferences, preferencesManager } from "@/core/preferences";
import { ElTooltip } from "element-plus";
import SvgIcon from "@/components/SvgIcon/index.vue";

import LayoutBreadcrumb from "./LayoutBreadcrumb.vue";
import CommandPalette from "@/components/CommandPalette/index.vue";
import ThemeSwitch from "@/components/ThemeSwitch/index.vue";
import LangSelect from "@/components/LangSelect/index.vue";
import LayoutFullscreen from "./LayoutFullscreen.vue";
import NoticeDropdown from "@/components/NoticeDropdown/index.vue";
import UserDropdown from "./UserDropdown.vue";

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
    gap: 4px;
  }

  &__right {
    display: flex;
    align-items: center;
    flex-shrink: 0;
    gap: 4px; // 对齐 react 间距基准：24px 按钮 + 4px 间隔 = pitch 28
  }
}

// ==================== 通用操作按钮 ====================
// 尺寸/间距基准对齐 react 顶栏（2026-11-08 实测：24×24 按钮、图标 14、pitch 28 = 24+gap4）
.navbar-action {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 24px;
  height: 24px;
  padding: 0;
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
    height: 24px;
  }

  :deep(.svg-local-icon) {
    font-size: 14px;
    line-height: 1;
    color: var(--el-text-color-regular);
    transition: all 0.2s ease;
  }

  // hover 态：柔和背景 + 图标主色高亮（react 同款，无缩放跳动）
  &:hover {
    background: rgba(0, 0, 0, 0.04);

    :deep(.svg-local-icon) {
      color: var(--gowind-primary-text);
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
  // 文字色在此设定，UserDropdown 内部用 currentColor 继承（子组件拿不到本文件的 scoped 类）
  &--profile {
    min-width: auto;
    padding: 0 4px;
    color: var(--el-text-color-regular);
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

  // 深色顶栏两种语境下，头像旁的用户名跟着提亮
  html.dark &--profile,
  html.semi-dark-header &--profile {
    color: rgba(255, 255, 255, 0.85);
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
