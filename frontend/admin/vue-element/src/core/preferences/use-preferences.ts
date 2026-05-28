import { computed } from "vue";

import zhCn from "element-plus/es/locale/lang/zh-cn";
import en from "element-plus/es/locale/lang/en";

import { preferencesManager } from "./preferences";
import { isDarkTheme } from "./update-css-variables";
import { toggleDarkMode } from "@/utils/theme";
import { diff } from "@/utils/diff";
import { ThemeModeType } from "@/core/preferences/types";

function usePreferences() {
  const preferences = preferencesManager.getPreferences();
  const initialPreferences = preferencesManager.getInitialPreferences();
  /**
   * @zh_CN 计算偏好设置的变化
   */
  const diffPreference = computed(() => {
    return diff(initialPreferences, preferences);
  });

  const appPreferences = computed(() => preferences.app);

  const tabbarPreferences = computed(() => preferences.tabbar);

  const logoPreferences = computed(() => preferences.logo);

  const navigationPreferences = computed(() => preferences.navigation);

  const shortcutKeysPreferences = computed(() => preferences.shortcutKeys);

  /**
   * @zh_CN 判断是否为暗黑模式
   * @param  preferences - 当前偏好设置对象，它的主题值将被用来判断是否为暗黑模式。
   * @returns 如果主题为暗黑模式，返回 true，否则返回 false。
   */
  const isDark = computed(() => {
    return isDarkTheme(preferences.theme.mode);
  });

  const locale = computed(() => {
    return preferences.app.locale;
  });

  // Element Plus 语言包映射
  const elementPlusLocales: Record<string, any> = {
    "zh-cn": zhCn,
    "en-us": en,
  };

  /**
   * 获取 Element Plus 语言包
   */
  const getElementPlusLocale = computed(() => {
    // 将 locale 转换为小写以匹配映射
    const normalizedLocale = preferences.app.locale.toLowerCase();
    return elementPlusLocales[normalizedLocale] || zhCn;
  });

  /**
   * 设置主题模式
   * @param mode 主题模式（light/dark/auto）
   */
  const setTheme = (mode: ThemeModeType) => {
    // 处理自动模式
    const actualMode =
      mode === "auto"
        ? window.matchMedia("(prefers-color-scheme: dark)").matches
          ? "dark"
          : "light"
        : mode;

    // 更新 DOM 类名
    toggleDarkMode(actualMode === "dark");

    // 更新 preferences
    preferencesManager.updatePreferences({
      theme: { mode },
    });
  };

  /**
   * 切换主题模式（在浅色和深色之间切换）
   */
  const toggleTheme = () => {
    const newMode = preferences.theme.mode === "dark" ? "light" : "dark";
    setTheme(newMode);
  };

  const isMobile = computed(() => {
    return appPreferences.value.isMobile;
  });

  const theme = computed(() => {
    return isDark.value ? "dark" : "light";
  });

  /**
   * @zh_CN 布局方式
   */
  const layout = computed(() => (isMobile.value ? "sidebar-nav" : appPreferences.value.layout));

  /**
   * @zh_CN 是否显示顶栏
   */
  const isShowHeaderNav = computed(() => {
    return preferences.header.enable;
  });

  /**
   * @zh_CN 是否全屏显示content，不需要侧边、底部、顶部、tab区域
   */
  const isFullContent = computed(() => appPreferences.value.layout === "full-content");

  /**
   * @zh_CN 是否侧边导航模式
   */
  const isSideNav = computed(() => appPreferences.value.layout === "sidebar-nav");

  /**
   * @zh_CN 是否侧边混合模式
   */
  const isSideMixedNav = computed(() => appPreferences.value.layout === "sidebar-mixed-nav");

  /**
   * @zh_CN 是否为头部导航模式
   */
  const isHeaderNav = computed(() => appPreferences.value.layout === "header-nav");

  /**
   * @zh_CN 是否为混合导航模式
   */
  const isMixedNav = computed(() => appPreferences.value.layout === "mixed-nav");

  /**
   * @zh_CN 是否包含侧边导航模式
   */
  const isSideMode = computed(() => {
    return isMixedNav.value || isSideMixedNav.value || isSideNav.value;
  });

  const sidebarCollapsed = computed(() => {
    return preferences.sidebar.collapsed;
  });

  /**
   * @zh_CN 是否开启keep-alive
   * 在tabs可见以及开启keep-alive的情况下才开启
   */
  const keepAlive = computed(() => preferences.tabbar.enable && preferences.tabbar.keepAlive);

  /**
   * @zh_CN 登录注册页面布局是否为左侧
   */
  const authPanelLeft = computed(() => {
    return appPreferences.value.authPageLayout === "panel-left";
  });

  /**
   * @zh_CN 登录注册页面布局是否为左侧
   */
  const authPanelRight = computed(() => {
    return appPreferences.value.authPageLayout === "panel-right";
  });

  /**
   * @zh_CN 登录注册页面布局是否为中间
   */
  const authPanelCenter = computed(() => {
    return appPreferences.value.authPageLayout === "panel-center";
  });

  /**
   * @zh_CN 内容是否已经最大化
   * 排除 full-content模式
   */
  const contentIsMaximize = computed(() => {
    const headerIsHidden = preferences.header.hidden;
    const sidebarIsHidden = preferences.sidebar.hidden;
    return headerIsHidden && sidebarIsHidden && !isFullContent.value;
  });

  /**
   * @zh_CN 是否启用全局搜索快捷键
   */
  const globalSearchShortcutKey = computed(() => {
    const { enable, globalSearch } = shortcutKeysPreferences.value;
    return enable && globalSearch;
  });

  /**
   * @zh_CN 是否启用全局注销快捷键
   */
  const globalLogoutShortcutKey = computed(() => {
    const { enable, globalLogout } = shortcutKeysPreferences.value;
    return enable && globalLogout;
  });

  const globalLockScreenShortcutKey = computed(() => {
    const { enable, globalLockScreen } = shortcutKeysPreferences.value;
    return enable && globalLockScreen;
  });

  /**
   * @zh_CN 偏好设置按钮位置
   */
  const preferencesButtonPosition = computed(() => {
    const { enablePreferences, preferencesButtonPosition } = preferences.app;

    // 如果没有启用偏好设置按钮
    if (!enablePreferences) {
      return {
        fixed: false,
        header: false,
      };
    }

    const { header, sidebar } = preferences;
    const headerHidden = header.hidden;
    const sidebarHidden = sidebar.hidden;

    const contentIsMaximize = headerHidden && sidebarHidden;

    const isHeaderPosition = preferencesButtonPosition === "header";

    // 如果设置了固定位置
    if (preferencesButtonPosition !== "auto") {
      return {
        fixed: preferencesButtonPosition === "fixed",
        header: isHeaderPosition,
      };
    }

    // 如果是全屏模式或者没有固定在顶部，
    const fixed =
      contentIsMaximize || isFullContent.value || isMobile.value || !isShowHeaderNav.value;

    return {
      fixed,
      header: !fixed,
    };
  });

  return {
    appPreferences,
    authPanelCenter,
    authPanelLeft,
    authPanelRight,
    contentIsMaximize,
    diffPreference,
    getElementPlusLocale,
    globalLockScreenShortcutKey,
    globalLogoutShortcutKey,
    globalSearchShortcutKey,
    isDark,
    isFullContent,
    isHeaderNav,
    isMixedNav,
    isMobile,
    isSideMixedNav,
    isSideMode,
    isSideNav,
    keepAlive,
    layout,
    locale,
    logoPreferences,
    navigationPreferences,
    preferencesButtonPosition,
    setTheme,
    sidebarCollapsed,
    tabbarPreferences,
    theme,
    toggleTheme,
  };
}

export { usePreferences };
