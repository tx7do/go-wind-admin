/**
 * 应用配置
 */

import { LayoutMode, ComponentSize, SidebarColor, ThemeMode } from "@/enums";
import type { SupportedLanguagesType } from "@/i18n/types";

const env = import.meta.env;
const { pkg } = __APP_INFO__;
const prefersDark = window.matchMedia("(prefers-color-scheme: dark)").matches;

// ============================================
// 应用配置
// ============================================
export const appConfig = {
  name: pkg.name as string,
  version: pkg.version as string,
  title: (env.VITE_APP_TITLE as string) || pkg.name,

  // 功能开关
  tenantEnabled: env.VITE_APP_TENANT_ENABLED === "true",
} as const;

// ============================================
// 用户偏好默认值
// ============================================
export const defaultPreferences = {
  theme: prefersDark ? ThemeMode.DARK : ThemeMode.LIGHT,
  themeColor: "#4080FF",
  sidebarColorScheme: SidebarColor.CLASSIC_BLUE,
  layout: LayoutMode.LEFT,
  size: ComponentSize.DEFAULT,
  language: "zh-cn" as SupportedLanguagesType,
  showTagsView: true,
  showAppLogo: true,
  showWatermark: false,
  pageSwitchingAnimation: "fade-slide",
  showSettings: true,
  watermarkContent: pkg.name,
  enableRefreshToken: false,
  loginExpiredMode: "page" as "modal" | "page",
  accessMode: import.meta.env.VITE_ROUTER_ACCESS_MODE,
  enableProgress: true, // 是否开启进度条
};

// ============================================
// 主题色预设
// ============================================
export const themeColorPresets = [
  "#4080FF",
  "#1890FF",
  "#409EFF",
  "#FA8C16",
  "#722ED1",
  "#13C2C2",
  "#52C41A",
  "#F5222D",
  "#2F54EB",
  "#EB2F96",
] as const;
