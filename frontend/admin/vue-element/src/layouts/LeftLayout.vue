<template>
  <BaseLayout>
    <!-- 左侧菜单 -->
    <div
      v-show="sidebarVisible"
      class="layout__sidebar"
      :class="{ 'layout__sidebar--collapsed': !isSidebarOpen }"
      :style="sidebarStyle"
    >
      <div :class="{ 'has-logo': showLogo }" class="layout-sidebar">
        <LayoutLogo v-if="showLogo" :collapse="!isSidebarOpen" />
        <el-scrollbar>
          <LayoutSidebar :data="routes" base-path="" />
        </el-scrollbar>
      </div>
    </div>

    <!-- 主内容区 -->
    <div
      class="layout__main"
      :class="{
        hasTagsView: showTagsView,
        'layout__main--collapsed': !isSidebarOpen,
        'layout__main--no-sidebar': !sidebarVisible,
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

const { showTagsView, showLogo, isSidebarOpen, routes } = useLayout();

const SIDEBAR_COLLAPSED_WIDTH = 54;

// 侧边栏可见性
const sidebarVisible = computed(() => preferences.sidebar.enable);

// 侧边栏展开宽度（响应 preferences）
const sidebarExpandedWidth = computed(() => preferences.sidebar.width);

// 侧边栏实际宽度
const sidebarActualWidth = computed(() =>
  isSidebarOpen.value ? sidebarExpandedWidth.value : SIDEBAR_COLLAPSED_WIDTH,
);

// 侧边栏内联样式
const sidebarStyle = computed(() => ({
  width: `${sidebarActualWidth.value}px`,
}));

// 主内容区左边距
const mainStyle = computed(() => {
  if (!sidebarVisible.value) return { left: "0px" };
  return { left: `${sidebarActualWidth.value}px` };
});
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
      height: 100%;
      background-color: var(--menu-background);
      transition: width 0.28s;

      &.has-logo {
        .el-scrollbar {
          height: calc(100vh - $navbar-height);
        }
      }

      :deep(.el-menu) {
        border: none;
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
