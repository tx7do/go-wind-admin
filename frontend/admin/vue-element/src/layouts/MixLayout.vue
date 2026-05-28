<template>
  <BaseLayout>
    <!-- 顶部菜单栏 -->
    <div class="layout__header">
      <div class="layout__header-content">
        <div v-if="showLogo" class="layout__header-logo">
          <LayoutLogo :collapse="isLogoCollapsed" />
        </div>

        <!-- 顶部菜单（仅 split 模式下显示） -->
        <div v-if="isSplit" class="layout__header-menu">
          <el-menu
            mode="horizontal"
            :default-active="activeTopMenuPath"
            :background-color="useMenuColors ? variables['menu-background'] : undefined"
            :text-color="useMenuColors ? variables['menu-text'] : undefined"
            :active-text-color="useMenuColors ? variables['menu-active-text'] : undefined"
            @select="handleTopMenuSelect"
          >
            <el-menu-item v-for="item in topMenuItems" :key="item.path" :index="item.path">
              <template v-if="item.meta">
                <!-- eslint-disable-next-line vue/no-deprecated-filter -->
                <MenuIcon :icon="item.meta.icon as string | undefined" />
                <span v-if="item.meta.title" class="ml-1">
                  {{ translateRouteTitle(item.meta.title as string) }}
                </span>
              </template>
            </el-menu-item>
          </el-menu>
        </div>

        <div class="layout__header-actions">
          <LayoutToolbar />
        </div>
      </div>
    </div>

    <!-- 主内容区容器 -->
    <div class="layout__container">
      <!-- 左侧菜单栏 -->
      <div
        class="layout__sidebar--left"
        :class="{ 'layout__sidebar--collapsed': !isSidebarActuallyOpen }"
        :style="{ width: sidebarActualWidth + 'px' }"
        @mouseenter="onSidebarMouseEnter"
        @mouseleave="onSidebarMouseLeave"
      >
        <el-scrollbar>
          <el-menu
            :default-active="activeSideMenuPath"
            :collapse="!isSidebarActuallyOpen"
            :collapse-transition="false"
            :unique-opened="accordion"
            :background-color="variables['menu-background']"
            :text-color="variables['menu-text']"
            :active-text-color="variables['menu-active-text']"
          >
            <LayoutSidebarItem
              v-for="item in effectiveSideMenuRoutes"
              :key="item.path"
              :item="item"
              :base-path="resolvePath(item.path)"
            />
          </el-menu>
        </el-scrollbar>
        <SidebarControlPanel
          :collapsed="!isSidebarOpen"
          :expand-on-hover="expandOnHover"
          @toggle-collapse="toggleSidebar"
          @toggle-expand-on-hover="toggleExpandOnHover"
        />
      </div>

      <!-- 主内容区 -->
      <div :class="{ hasTagsView: showTagsView }" class="layout__main">
        <LayoutTagsView v-if="showTagsView" />
        <LayoutMain />
      </div>
    </div>
  </BaseLayout>
</template>

<script setup lang="ts">
import type { LocationQueryRaw, RouteRecordRaw } from "vue-router";
import { useWindowSize } from "@vueuse/core";
import { ElIcon } from "element-plus";

import { useLayout } from "./useLayout";
import { useAccessStore } from "@/stores";
import { isExternal } from "@/utils";
import { translateRouteTitle } from "@/i18n";
import { preferences, preferencesManager, usePreferences } from "@/core/preferences";

import BaseLayout from "./BaseLayout.vue";
import LayoutLogo from "./components/LayoutLogo.vue";
import LayoutToolbar from "./components/LayoutToolbar.vue";
import LayoutTagsView from "./components/LayoutTagsView.vue";
import LayoutMain from "./components/LayoutMain.vue";
import LayoutSidebarItem from "./components/LayoutSidebarItem.vue";
import SidebarControlPanel from "./components/SidebarControlPanel.vue";
import variables from "@/styles/variables.module.scss";

// 菜单图标渲染组件
const MenuIcon = defineComponent({
  props: { icon: String },
  setup(props) {
    const isElIcon = computed(() => props.icon?.startsWith("el-icon"));
    const iconName = computed(() => props.icon?.replace("el-icon-", ""));

    return () => {
      if (!props.icon) {
        return h("div", { class: "i-svg:menu" });
      }

      // Element Plus 图标
      if (isElIcon.value) {
        return h(ElIcon, null, () => h(resolveComponent(iconName.value!)));
      }

      // SVG 图标
      return h("div", { class: `i-svg:${props.icon}` });
    };
  },
});

const route = useRoute();
const router = useRouter();
const { width } = useWindowSize();

const accessStore = useAccessStore();

const {
  showTagsView,
  showLogo,
  isSidebarOpen,
  toggleSidebar,
  routes,
  sideMenuRoutes,
  activeTopMenuPath,
} = useLayout();

const { navigationPreferences } = usePreferences();

const SIDEBAR_COLLAPSED_WIDTH = 54;

// 侧边栏 hover 展开
const expandOnHover = computed(() => preferences.sidebar.expandOnHover);
const isHoverExpanded = ref(false);

// 自动模式: 视觉状态仅由 hover 控制
// 手动模式: 视觉状态由 collapsed 偏好控制
const isSidebarActuallyOpen = computed(() => {
  if (expandOnHover.value) {
    return isHoverExpanded.value;
  }
  return isSidebarOpen.value;
});

const sidebarActualWidth = computed(() =>
  isSidebarActuallyOpen.value ? preferences.sidebar.width : SIDEBAR_COLLAPSED_WIDTH
);
const accordion = computed(() => navigationPreferences.value.accordion);
const isSplit = computed(() => navigationPreferences.value.split);

// 侧边栏菜单数据：split 模式下显示二级菜单，否则显示完整菜单
const effectiveSideMenuRoutes = computed(() =>
  isSplit.value ? sideMenuRoutes.value : routes.value
);

const isLogoCollapsed = computed(() => width.value < 768);

// 是否使用深色菜单配色（暗色主题）
const useMenuColors = computed(() => {
  const { theme } = preferencesManager.getPreferences();
  return theme.mode === "dark";
});

// 顶部菜单项（处理单子菜单显示优化）
const topMenuItems = computed(() => {
  const routes = accessStore.accessRoutes.filter((item) => !item.meta?.hideInMenu);

  return routes.map((route) => {
    // 无子菜单，直接返回
    if (!route.children?.length) return route;

    // 过滤可见子菜单
    const visibleChildren = route.children.filter((child) => !child.meta?.hideInMenu);

    // 仅一个可见子菜单时，显示子菜单信息
    if (visibleChildren.length === 1) {
      const child = visibleChildren[0];
      return {
        ...route,
        meta: {
          ...route.meta,
          title: child.meta?.title || route.meta?.title,
          icon: child.meta?.icon || route.meta?.icon,
        },
      };
    }
    return route;
  });
});

// 左侧菜单激活路径
const activeSideMenuPath = computed(() => {
  const { meta, path } = route;
  return typeof meta?.activeMenu === "string" ? meta.activeMenu : path;
});

// 解析左侧菜单路径
function resolvePath(routePath: string) {
  if (isExternal(routePath)) return routePath;
  if (isSplit.value) {
    // split 模式：基于顶部激活菜单拼接路径
    if (routePath.startsWith("/")) return activeTopMenuPath.value + routePath;
    return `${activeTopMenuPath.value}/${routePath}`;
  }
  // 非 split 模式：直接使用路由路径
  if (routePath.startsWith("/")) return routePath;
  return routePath;
}

// 顶部菜单点击
function handleTopMenuSelect(menuPath: string) {
  if (menuPath === activeTopMenuPath.value) return;

  // 跳转到该顶级菜单下的第一个子菜单
  const topMenu = accessStore.accessRoutes.find((route) => route.path === menuPath);
  if (topMenu?.children?.length) {
    navigateToFirstMenu(topMenu.children as RouteRecordRaw[]);
  }
}

// 导航到第一个可访问菜单
function navigateToFirstMenu(menus: RouteRecordRaw[]) {
  if (!menus.length) return;

  const [first] = menus;
  if (first.children?.length) {
    navigateToFirstMenu(first.children as RouteRecordRaw[]);
  } else if (first.name) {
    router.push({
      name: first.name,
      query:
        typeof first.meta?.params === "object"
          ? (first.meta.params as LocationQueryRaw)
          : undefined,
    });
  }
}

// 监听路由变化，同步顶部菜单状态
// activeTopMenuPath 和 sideMenuRoutes 会自动根据路由计算，无需手动同步

// =====================
// 侧边栏 hover 展开/收起
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
  &__header {
    position: sticky;
    top: 0;
    z-index: 999;
    width: 100%;
    height: $navbar-height;
    background-color: var(--navbar-background);
    border-bottom: 1px solid var(--navbar-border-color);

    &-content {
      display: flex;
      align-items: center;
      height: 100%;
      padding: 0;
    }

    &-logo {
      display: flex;
      flex-shrink: 0;
      align-items: center;
      justify-content: center;
      height: 100%;
    }

    &-menu {
      display: flex;
      flex: 1;
      align-items: center;
      min-width: 0;
      height: 100%;
      overflow: hidden;

      :deep(.el-menu) {
        height: 100%;
        background-color: transparent;
        border: none;
      }

      :deep(.el-menu--horizontal) {
        display: flex;
        align-items: center;
        height: 100%;

        .el-menu-item {
          height: 100%;
          line-height: $navbar-height;
          border-bottom: none;

          &.is-active {
            background-color: rgba(255, 255, 255, 0.12);
            border-bottom: 2px solid var(--el-color-primary);
          }
        }
      }
    }

    &-actions {
      display: flex;
      flex-shrink: 0;
      align-items: center;
      height: 100%;
      padding: 0 16px;
    }
  }

  &__container {
    display: flex;
    height: calc(100vh - $navbar-height);
    padding-top: 0;

    .layout__sidebar--left {
      position: relative;
      display: flex;
      flex-direction: column;
      // 宽度由内联 style 控制
      height: 100%;
      background-color: var(--menu-background);
      transition: width 0.28s;

      &.layout__sidebar--collapsed {
        // 折叠宽度由内联 style 控制
      }

      .el-scrollbar {
        flex: 1;
        min-height: 0;
      }

      :deep(.el-menu) {
        height: 100%;
        border: none;
      }
    }

    .layout__main {
      flex: 1;
      min-width: 0;
      height: 100%;
      margin-left: 0;
      overflow-y: auto;
    }
  }
}

:deep(.mobile) {
  .layout__container {
    .layout__sidebar--left {
      position: fixed;
      top: $navbar-height;
      bottom: 0;
      left: 0;
      z-index: 1000;
      transition: transform 0.28s;
    }
  }

  &.hideSidebar {
    .layout__sidebar--left {
      transform: translateX(-100%);
    }
  }
}

:deep(.hasTagsView) {
  .app-main {
    height: calc(100vh - $navbar-height - $tags-view-height) !important;
  }
}
</style>
