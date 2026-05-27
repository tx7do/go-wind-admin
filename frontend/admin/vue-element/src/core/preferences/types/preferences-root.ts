import type {
  AppPreferences,
  BreadcrumbPreferences,
  CopyrightPreferences,
  FooterPreferences,
  HeaderPreferences,
  LogoPreferences,
  NavigationPreferences,
  ShortcutKeyPreferences,
  SidebarPreferences,
  TabbarPreferences,
  ThemePreferences,
  TransitionPreferences,
  WidgetPreferences,
} from "./app.ts";

/**
 * 偏好设置总接口
 */
export interface Preferences {
  /** 全局配置 */
  app: AppPreferences;
  /** 面包屑偏好设置 */
  breadcrumb: BreadcrumbPreferences;
  /** 版权配置 */
  copyright: CopyrightPreferences;
  /** 底栏配置 */
  footer: FooterPreferences;
  /** 顶栏偏好设置 */
  header: HeaderPreferences;
  /** logo配置 */
  logo: LogoPreferences;
  /** 导航配置 */
  navigation: NavigationPreferences;
  /** 快捷键配置 */
  shortcutKeys: ShortcutKeyPreferences;
  /** 侧边栏配置 */
  sidebar: SidebarPreferences;
  /** 标签页配置 */
  tabbar: TabbarPreferences;
  /** 主题配置 */
  theme: ThemePreferences;
  /** 动画配置 */
  transition: TransitionPreferences;
  /** 功能配置 */
  widget: WidgetPreferences;
}

/**
 * 偏好设置接口
 */
export type PreferencesKeys = keyof Preferences;

export type DeepPartial<T> = {
  [P in keyof T]?: T[P] extends object ? DeepPartial<T[P]> : T[P];
};

/**
 * 偏好设置初始化参数
 */
export interface InitialOptions {
  /** 命名空间 */
  namespace: string;
  /** 覆盖默认偏好设置 */
  overrides?: DeepPartial<Preferences>;
}
