<template>
  <div :class="['navbar-actions', navbarActionsClass]">
    <!-- 桌面端工具项 -->
    <template v-if="isDesktop">
      <!-- 代租户操作切换器（仅平台管理员渲染；issue #13） -->
      <div class="navbar-actions__item">
        <TenantSwitcher />
      </div>

      <!-- 搜索 -->
      <div class="navbar-actions__item navbar-actions__item--search">
        <CommandPalette />
      </div>

      <ElTooltip :content="$t('common.header.settings')" placement="bottom">
              <!-- 系统设置 -->
      <div
        v-if="preferences.app.enablePreferences"
        class="navbar-actions__item"
        @click="handleSettingsClick"
      >
        <SvgIcon icon="setting" />
      </div>
      </ElTooltip>

      <ElTooltip :content="$t('common.header.fullscreen')" placement="bottom">
              <!-- 全屏 -->
      <div class="navbar-actions__item">
        <LayoutFullscreen />
      </div>
      </ElTooltip>

      <ElTooltip :content="$t('common.header.layout')" placement="bottom">
              <!-- 布局大小 -->
      <div class="navbar-actions__item">
        <LayoutSizeSelect />
      </div>
      </ElTooltip>

      <ElTooltip :content="$t('common.header.language')" placement="bottom">
              <!-- 语言选择 -->
      <div class="navbar-actions__item">
        <LangSelect />
      </div>
      </ElTooltip>

      <ElTooltip :content="$t('common.header.notification')" placement="bottom">
              <!-- 通知 -->
      <div class="navbar-actions__item">
        <NoticeDropdown />
      </div>
      </ElTooltip>
    </template>

    <!-- 用户菜单 -->
    <div class="navbar-actions__item navbar-actions__item--profile">
      <UserDropdown />
    </div>
  </div>
</template>

<script setup lang="ts">
import { preferences, usePreferences } from "@go-wind/admin-element";
import { ElTooltip } from "element-plus";
import SvgIcon from "@/components/SvgIcon/index.vue";

// 导入子组件
import CommandPalette from "@/components/CommandPalette/index.vue";
import LayoutFullscreen from "./LayoutFullscreen.vue";
import LayoutSizeSelect from "./LayoutSizeSelect.vue";
import LangSelect from "@/components/LangSelect/index.vue";
import NoticeDropdown from "@/components/NoticeDropdown/index.vue";
import UserDropdown from "./UserDropdown.vue";
import TenantSwitcher from "./TenantSwitcher.vue";

const { isMobile, appPreferences } = usePreferences();

// 是否为桌面设备
const isDesktop = computed(() => !isMobile.value);

// 注入设置面板可见性状态
const settingsVisible = inject<Ref<boolean>>("settingsVisible", ref(false));

// 根据主题和布局选择样式类
const navbarActionsClass = computed(() => {
  const theme = preferences.theme.mode;
  const layout = appPreferences.value.layout;

  // 暗黑主题下，所有布局都使用白色文字
  if (theme === "dark") {
    return "navbar-actions--white-text";
  }

  // 明亮主题下
  if (theme === "light") {
    // 顶部布局和混合布局的顶部区域使用深色文字
    if (layout === "header-nav" || layout === "mixed-nav") {
      return "navbar-actions--dark-text";
    }
  }

  return "navbar-actions--dark-text";
});

/**
 * 打开系统设置页面
 */
function handleSettingsClick() {
  settingsVisible.value = true;
}
</script>

<style lang="scss" scoped>
.navbar-actions {
  display: flex;
  align-items: center;
  min-height: 44px;

  &__item {
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
    min-width: 44px; /* 增加最小点击区域到44px，符合人机交互标准 */
    height: 44px;
    padding: 0 8px;
    text-align: center;
    cursor: pointer;
    border-radius: 8px;
    transition: all 0.2s ease;
    background: transparent;

    // 确保子元素居中
    > * {
      display: flex;
      align-items: center;
      justify-content: center;
    }

    // 确保 Element Plus 组件可以正常工作
    :deep(.el-dropdown),
    :deep(.el-tooltip) {
      display: flex;
      align-items: center;
      justify-content: center;
      width: 100%;
      height: 44px;
    }

    // 图标样式
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
  }
}

// 白色文字样式（用于深色背景：暗黑主题、顶部布局、混合布局等）
.navbar-actions--white-text {
  .navbar-actions__item {
    :deep(.svg-local-icon) {
      color: rgba(255, 255, 255, 0.75);
    }

    &:hover {
      background: rgba(255, 255, 255, 0.08);
      transform: scale(1.05);

      :deep(.svg-local-icon) {
        color: #ffffff;
      }
    }
  }

  // profile 按钮不放大；用户名在此提亮
  .navbar-actions__item--profile {
    color: rgba(255, 255, 255, 0.85);
    transform: none !important;

    &:hover {
      background: rgba(255, 255, 255, 0.08);
      transform: none;
    }
  }

  // 租户选择器在白色文字模式下的样式
  ::v-deep(.tenant-switcher__trigger) {
    color: rgba(255, 255, 255, 0.85);
  }
  ::v-deep(.tenant-switcher__trigger .tenant-switcher__icon) {
    color: rgba(255, 255, 255, 0.85);
  }
  ::v-deep(.tenant-switcher__trigger:hover) {
    color: #ffffff;
    background: rgba(255, 255, 255, 0.08);
  }
  ::v-deep(.tenant-switcher__trigger:hover .tenant-switcher__icon) {
    color: #ffffff;
  }
}

// 深色文字样式（用于浅色背景：明亮主题下的左侧布局等）
.navbar-actions--dark-text {
  .navbar-actions__item {
    :deep(.svg-local-icon) {
      color: var(--el-text-color-regular) !important;
    }

    &:hover {
      background: rgba(0, 0, 0, 0.04);
      transform: scale(1.05);

      :deep(.svg-local-icon) {
        color: var(--gowind-primary-text) !important;
      }
    }
  }

  // profile 按钮不放大；用户名在此设定
  .navbar-actions__item--profile {
    color: var(--el-text-color-regular) !important;
    transform: none !important;

    &:hover {
      background: rgba(0, 0, 0, 0.04);
      transform: none;
    }
  }

  // 租户选择器在深色文字模式下的样式
  ::v-deep(.tenant-switcher__trigger) {
    color: var(--el-text-color-regular) !important;
  }
  ::v-deep(.tenant-switcher__trigger .tenant-switcher__icon) {
    color: var(--el-text-color-regular) !important;
  }
  ::v-deep(.tenant-switcher__trigger:hover) {
    color: var(--gowind-primary-text) !important;
    background: var(--el-fill-color-light);
  }
  ::v-deep(.tenant-switcher__trigger:hover .tenant-switcher__icon) {
    color: var(--gowind-primary-text) !important;
  }
}

// 确保下拉菜单中的图标不受影响
::v-deep(.el-dropdown-menu) {
  [class^="svg-local-icon"] {
    color: var(--el-text-color-regular) !important;

    &:hover {
      color: var(--gowind-primary-text) !important;
    }
  }
}
</style>
