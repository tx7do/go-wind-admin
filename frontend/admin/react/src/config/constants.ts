/**
 * 全局常量配置
 * 统一管理路由路径、表格配置、业务阈值等魔法值
 */

// ========== 路由路径 ==========
export const ROUTES = {
  /** 登录页 */
  LOGIN: '/auth/login',
  /** 注册页 */
  REGISTER: '/auth/register',

  /** 默认首页（登录后跳转） */
  DEFAULT_HOME: '/dashboard',

  /** 403 无权限页 */
  FORBIDDEN: '/error/403',
  /** 404 页 */
  NOT_FOUND: '/error/404',
} as const;

export type RouteKey = keyof typeof ROUTES;
export type RoutePath = (typeof ROUTES)[RouteKey];

// ========== 表格配置 ==========
export const TABLE = {
  /** 默认每页条数 */
  DEFAULT_PAGE_SIZE: 20,
  /** 每页条数选项 */
  PAGE_SIZE_OPTIONS: ['10', '20', '50', '100'],
  /** 最大每页条数 */
  MAX_PAGE_SIZE: 100,
  /** 表格组件内部高度 */
  COMPONENT_HEIGHTS: {
    /** 工具栏高度 */
    TOOLBAR: 50,
    /** 分页器高度 */
    PAGINATION: 50,
    /** 表头高度 */
    HEADER: 50,
    /** 搜索表单高度（展开时）*/
    SEARCH_FORM: 80,
  },
  /** 表格滚动配置 */
  SCROLL_CONFIG: {
    /** 默认最小高度 */
    MIN_HEIGHT: 300,
    /** 额外边距 */
    OFFSET: 16,
  },
} as const;

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
