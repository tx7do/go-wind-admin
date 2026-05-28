/**
 * 布局 Composable
 *
 * 整合布局状态、设备检测、菜单数据
 */
import { computed, watch } from "vue";
import { useRoute } from "vue-router";

import { useAccessStore } from "@/stores";
import { preferences, preferencesManager, usePreferences } from "@/core/preferences";

export function useLayout() {
  const route = useRoute();
  const permissionStore = useAccessStore();
  const { isMobile, sidebarCollapsed, appPreferences, tabbarPreferences, logoPreferences } =
    usePreferences();

  // ============================================
  // 设备检测
  // ============================================

  const isDesktop = computed(() => !isMobile.value);

  // 监听设备类型变化，自动调整侧边栏
  // 仅在桌面↔移动切换时触发，不干预用户手动折叠/展开
  watch(
    () => isMobile.value,
    (mobile) => {
      // 同步 isMobile 到 preferences
      if (mobile !== preferences.app.isMobile) {
        preferencesManager.updatePreferences({ app: { isMobile: mobile } });
      }
      // 切到移动端 → 收起侧边栏；切到桌面端 → 展开侧边栏
      preferencesManager.updatePreferences({
        sidebar: { collapsed: mobile },
      });
    },
    { immediate: true }
  );

  // ============================================
  // 布局状态
  // ============================================

  const currentLayout = computed(() => appPreferences.value.layout);
  const isSidebarOpen = computed(() => !sidebarCollapsed.value);
  const showTagsView = computed(() => tabbarPreferences.value.enable);
  const showSettings = computed(() => appPreferences.value.enablePreferences);
  const showLogo = computed(() => logoPreferences.value.enable);

  const layoutClass = computed(() => ({
    hideSidebar: sidebarCollapsed.value,
    openSidebar: !sidebarCollapsed.value,
    mobile: isMobile.value,
    [`layout-${appPreferences.value.layout}`]: true,
  }));

  // ============================================
  // 菜单数据
  // ============================================

  /** 路由列表（左侧/顶部菜单） */
  const routes = computed(() => permissionStore.accessRoutes);

  /** 混合布局侧边菜单（根据顶级菜单路径动态计算） */
  const sideMenuRoutes = computed(() => {
    const topMenuPath = activeTopMenuPath.value;
    // 从所有路由中找到匹配的顶级菜单
    const topMenu = permissionStore.accessRoutes.find((route) => route.path === topMenuPath);

    if (!topMenu?.children) {
      return [];
    }

    // 过滤掉隐藏的菜单
    return topMenu.children.filter((child) => !child.meta?.hideInMenu);
  });

  /** 顶部菜单激活路径（仅混合布局使用） */
  const activeTopMenuPath = computed(() => {
    const path = route.path;
    // 提取第一段路径作为顶级菜单
    // /system/user → /system
    // /dashboard → /dashboard
    const segments = path.split("/").filter(Boolean);
    return segments.length > 0 ? `/${segments[0]}` : "/";
  });

  /** 当前激活菜单 */
  const activeMenu = computed(() => {
    const { meta, path } = route;
    return meta?.activeMenu || path;
  });

  // ============================================
  // 操作方法
  // ============================================

  function toggleSidebar() {
    preferencesManager.updatePreferences({
      sidebar: { collapsed: !sidebarCollapsed.value },
    });
  }

  function closeSidebar() {
    preferencesManager.updatePreferences({
      sidebar: { collapsed: true },
    });
  }

  return {
    // 设备
    isDesktop,
    isMobile,
    // 布局
    currentLayout,
    layoutClass,
    isSidebarOpen,
    showTagsView,
    showSettings,
    showLogo,
    // 菜单
    routes,
    sideMenuRoutes,
    activeMenu,
    activeTopMenuPath,
    // 方法
    toggleSidebar,
    closeSidebar,
  };
}
