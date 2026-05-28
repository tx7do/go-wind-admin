<template>
  <BaseLayout>
    <!-- 左侧菜单 -->
    <div
      v-show="sidebarVisible && !sidebarHidden"
      class="layout__sidebar"
      :class="sidebarClass"
      :style="sidebarStyle"
      @mouseenter="onSidebarMouseEnter"
      @mouseleave="onSidebarMouseLeave"
    >
      <div :class="{ 'has-logo': showLogo }" class="layout-sidebar">
        <LayoutLogo :collapse="!isSidebarActuallyOpen" />
        <el-scrollbar>
          <LayoutSidebar :data="routes" base-path="" :collapse="!isSidebarActuallyOpen" />
        </el-scrollbar>
        <SidebarControlPanel
          :collapsed="!isSidebarOpen"
          :expand-on-hover="expandOnHover"
          @toggle-collapse="toggleCollapse"
          @toggle-expand-on-hover="toggleExpandOnHover"
        />
      </div>
    </div>

    <!-- 主内容区 -->
    <div
      class="layout__main"
      :class="{
        hasTagsView: showTagsView,
        'layout__main--no-sidebar': !sidebarVisible || sidebarHidden,
      }"
      :style="mainStyle"
    >
      <LayoutNavbar />
      <LayoutTagsView v-if="showTagsView" />
      <LayoutMain />
    </div>
  </BaseLayout>
</template>

<script setup lang="ts">
import { useLayout } from "./useLayout";
import { preferences } from "@/core/preferences";
import BaseLayout from "./BaseLayout.vue";
import LayoutLogo from "./components/LayoutLogo.vue";
import LayoutNavbar from "./components/LayoutNavbar.vue";
import LayoutTagsView from "./components/LayoutTagsView.vue";
import LayoutMain from "./components/LayoutMain.vue";
import LayoutSidebar from "./components/LayoutSidebar.vue";
import SidebarControlPanel from "./components/SidebarControlPanel.vue";
import { preferencesManager } from "@/core/preferences";

const { showTagsView, showLogo, isSidebarOpen, toggleSidebar, routes } = useLayout();

const SIDEBAR_COLLAPSED_WIDTH = 54;

// =====================
// 侧边栏基础状态
// =====================

// 侧边栏可见性 (enable)
const sidebarVisible = computed(() => preferences.sidebar.enable);

// 侧边栏 CSS 隐藏 (hidden)
const sidebarHidden = computed(() => preferences.sidebar.hidden);

// collapsedShowTitle: 折叠时是否显示标题
const collapsedShowTitle = computed(() => preferences.sidebar.collapsedShowTitle);

// expandOnHover: 折叠时鼠标悬停是否自动展开
const expandOnHover = computed(() => preferences.sidebar.expandOnHover);

// 鼠标悬停展开状态
const isHoverExpanded = ref(false);

// =====================
// 侧边栏展开/折叠计算
// =====================

// 实际是否展开
// 自动模式 (expandOnHover): 视觉状态仅由 hover 控制，忽略 collapsed 偏好
// 手动模式 (固定): 视觉状态由 collapsed 偏好控制
const isSidebarActuallyOpen = computed(() => {
  if (expandOnHover.value) {
    return isHoverExpanded.value;
  }
  return isSidebarOpen.value;
});

// 侧边栏展开宽度（响应 preferences）
const sidebarExpandedWidth = computed(() => preferences.sidebar.width);

// 侧边栏实际宽度
const sidebarActualWidth = computed(() =>
  isSidebarActuallyOpen.value ? sidebarExpandedWidth.value : SIDEBAR_COLLAPSED_WIDTH
);

// =====================
// 侧边栏 CSS class
// =====================

const sidebarClass = computed(() => ({
  "layout__sidebar--collapsed": !isSidebarActuallyOpen.value,
  "layout__sidebar--collapsed-show-title": collapsedShowTitle.value && !isSidebarActuallyOpen.value,
}));

// =====================
// 侧边栏内联样式
// =====================

const sidebarStyle = computed(() => ({
  width: `${sidebarActualWidth.value}px`,
}));

// 主内容区左边距
const mainStyle = computed(() => {
  if (!sidebarVisible.value || sidebarHidden.value) return { left: "0px" };
  return { left: `${sidebarActualWidth.value}px` };
});

// =====================
// 鼠标悬停展开/收起
// =====================

function onSidebarMouseEnter() {
  if (expandOnHover.value) {
    isHoverExpanded.value = true;
  }
}

function onSidebarMouseLeave() {
  if (isHoverExpanded.value) {
    isHoverExpanded.value = false;
  }
}

// =====================
// 控制面板操作
// =====================

/** 折叠/展开侧边栏（修改偏好） */
function toggleCollapse() {
  toggleSidebar();
}

/** 切换鼠标悬停自动展开模式 */
function toggleExpandOnHover() {
  const newExpandOnHover = !expandOnHover.value;

  if (newExpandOnHover) {
    // 切换到自动模式：保持当前视觉状态作为 hover 状态
    isHoverExpanded.value = isSidebarActuallyOpen.value;
  } else {
    // 切换到手动模式：清除 hover 状态
    isHoverExpanded.value = false;
  }

  preferencesManager.updatePreferences({
    sidebar: { expandOnHover: newExpandOnHover },
  });
}
</script>

<style lang="scss" scoped>
.layout {
  &__sidebar {
    position: fixed;
    top: 0;
    bottom: 0;
    left: 0;
    z-index: 999;
    // 宽度由内联 style 控制（响应 preferences.sidebar.width）
    background-color: $menu-background;
    transition: width 0.28s;

    .layout-sidebar {
      position: relative;
      display: flex;
      flex-direction: column;
      height: 100%;
      background-color: var(--menu-background);
      transition: width 0.28s;

      .el-scrollbar {
        flex: 1;
        min-height: 0;
      }

      &.has-logo {
        .el-scrollbar {
          // 由 flex 布局自动计算高度，不再需要 calc
        }
      }

      :deep(.el-menu) {
        border: none;
      }
    }

    // =====================
    // collapsedShowTitle: 折叠时显示菜单标题
    // =====================
    &.layout__sidebar--collapsed-show-title {
      :deep(.el-menu--collapse) {
        .el-menu-item,
        .el-sub-menu > .el-sub-menu__title {
          height: auto !important;
          line-height: 1.2 !important;
          padding: 8px 0 !important;
          text-align: center;

          .menu-title {
            display: block !important;
            visibility: visible !important;
            width: auto !important;
            height: auto !important;
            overflow: visible !important;
            margin-left: 0 !important;
            font-size: 12px;
            text-align: center;
          }
        }

        // 让 el-tooltip 不触发（已有文字，不需要 tooltip）
        .el-tooltip__trigger {
          display: flex;
          flex-direction: column;
          align-items: center;
          justify-content: center;
        }
      }
    }
  }

  &__main {
    position: fixed;
    top: 0;
    right: 0;
    bottom: 0;
    // left 由内联 style 控制
    overflow-y: auto;
    transition: left 0.28s;

    &--no-sidebar {
      left: 0 !important;
    }

    .fixed-header {
      position: sticky;
      top: 0;
      z-index: 9;
      transition: width 0.28s;
    }
  }
}

/* 移动端样式*/
.mobile {
  .layout__sidebar {
    transition:
      transform 0.28s,
      width 0s;
  }

  &.hideSidebar {
    .layout__sidebar {
      transform: translateX(-100%);
    }
  }

  &.openSidebar {
    .layout__sidebar {
      transform: translateX(0);
    }
  }

  .layout__main {
    left: 0 !important;
  }
}

.hasTagsView {
  :deep(.app-main) {
    height: calc(100vh - $navbar-height - $tags-view-height) !important;
  }
}
</style>
