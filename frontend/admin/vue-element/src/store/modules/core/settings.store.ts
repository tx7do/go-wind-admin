import { SidebarColor, ThemeMode } from "@/enums";
import type { LayoutMode } from "@/enums";
import { applyTheme, generateThemeColors, toggleDarkMode, toggleSidebarColor } from "@/utils/theme";
import { STORAGE_KEYS } from "@/constants";
import { defaultPreferences } from "@/settings";

export const useSettingsStore = defineStore("setting", () => {
  // 界面显示
  const settingsVisible = ref(false);
  const showTagsView = useStorage(STORAGE_KEYS.SHOW_TAGS_VIEW, defaultPreferences.showTagsView);
  const showAppLogo = useStorage(STORAGE_KEYS.SHOW_APP_LOGO, defaultPreferences.showAppLogo);
  const showWatermark = useStorage(STORAGE_KEYS.SHOW_WATERMARK, defaultPreferences.showWatermark);
  const pageSwitchingAnimation = useStorage(
    STORAGE_KEYS.PAGE_SWITCHING_ANIMATION,
    defaultPreferences.pageSwitchingAnimation
  );

  // 布局
  const layout = useStorage<LayoutMode>(STORAGE_KEYS.LAYOUT, defaultPreferences.layout as LayoutMode);
  const sidebarColorScheme = useStorage(
    STORAGE_KEYS.SIDEBAR_COLOR_SCHEME,
    defaultPreferences.sidebarColorScheme
  );

  // 主题
  const theme = useStorage<ThemeMode>(STORAGE_KEYS.THEME, defaultPreferences.theme);
  const themeColor = useStorage(STORAGE_KEYS.THEME_COLOR, defaultPreferences.themeColor);

  // 特殊模式
  const grayMode = useStorage(STORAGE_KEYS.GRAY_MODE, false);
  const colorWeak = useStorage(STORAGE_KEYS.COLOR_WEAK, false);

  // 主题变化监听
  watch(
    [theme, themeColor],
    ([t, c]: [ThemeMode, string]) => {
      toggleDarkMode(t === ThemeMode.DARK);
      applyTheme(generateThemeColors(c, t));
    },
    { immediate: true }
  );

  watch(sidebarColorScheme, (v) => toggleSidebarColor(v === SidebarColor.CLASSIC_BLUE), {
    immediate: true,
  });

  // 灰色模式监听
  watch(
    grayMode,
    (v) => {
      document.documentElement.style.filter = v ? "grayscale(100%)" : "";
    },
    { immediate: true }
  );

  // 色弱模式监听
  watch(
    colorWeak,
    (v) => {
      document.documentElement.classList.toggle("color-weak", v);
    },
    { immediate: true }
  );

  function resetSettings() {
    showTagsView.value = defaultPreferences.showTagsView;
    showAppLogo.value = defaultPreferences.showAppLogo;
    showWatermark.value = defaultPreferences.showWatermark;
    pageSwitchingAnimation.value = defaultPreferences.pageSwitchingAnimation;
    grayMode.value = false;
    colorWeak.value = false;
    sidebarColorScheme.value = defaultPreferences.sidebarColorScheme;
    layout.value = defaultPreferences.layout as LayoutMode;
    themeColor.value = defaultPreferences.themeColor;
    theme.value = defaultPreferences.theme;
  }

  return {
    settingsVisible,
    showTagsView,
    showAppLogo,
    showWatermark,
    pageSwitchingAnimation,
    grayMode,
    colorWeak,
    sidebarColorScheme,
    layout,
    themeColor,
    theme,
    resetSettings,
  };
});
