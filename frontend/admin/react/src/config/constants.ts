/**
 * 全局常量配置
 * 统一管理路由路径、表格配置、业务阈值等魔法值
 */

// ========== 路由路径 ==========
export const ROUTES = {
  /** 登录页 */
  LOGIN: '/auth/login',
  /** 注册页 */

  /** 默认首页（登录后跳转，与 ele/vben 的 DEFAULT_HOME_PATH 对齐） */
  DEFAULT_HOME: '/dashboard/analytics',

  /** 403 无权限页 */
  FORBIDDEN: '/error/403',
  /** 404 页 */
  NOT_FOUND: '/error/404',
} as const;

export type RouteKey = keyof typeof ROUTES;
export type RoutePath = (typeof ROUTES)[RouteKey];


// ========== 布局高度配置 ==========
export const LAYOUT_HEIGHTS = {
  /** 顶栏高度 */
  HEADER: 56,
  /** 标签栏高度 */
  TABS: 40,
  /** 面包屑高度 */
  BREADCRUMB: 40,
  /** 内容区上边距 */
  CONTENT_PADDING: 16,
} as const;

// ========== HTTP 方法列表 ==========
export const METHOD_LIST = [
  { label: 'GET', value: 'GET' },
  { label: 'POST', value: 'POST' },
  { label: 'PUT', value: 'PUT' },
  { label: 'DELETE', value: 'DELETE' },
  { label: 'PATCH', value: 'PATCH' },
  { label: 'HEAD', value: 'HEAD' },
  { label: 'OPTIONS', value: 'OPTIONS' },
];

// ========== 业务阈值（按需扩展） ==========
export const BUSINESS = {
  /** 请求超时时间 (ms) */
  REQUEST_TIMEOUT: 10000,
  /** 验证码有效期 (秒) */
  CAPTCHA_EXPIRE: 60,
  /** 密码最小长度 */
  PASSWORD_MIN_LENGTH: 8,
} as const;
