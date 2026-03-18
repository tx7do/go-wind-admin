import {identityservicev1_User} from "@/api/admin/service/v1";

/**
 * 令牌载荷
 */
export interface TokenPayload {
  /**
   * 令牌值
   */
  value: string;
  /**
   * 令牌过期时间
   */
  expiresAt?: number;
}

export interface IAccessState {
  /**
   * 权限码
   */
  permissions: string[]

  /**
   * 登录 accessToken
   */
  accessToken: TokenPayload | null
  /**
   * 登录 refreshToken
   */
  refreshToken: TokenPayload | null
}

/**
 * 会话状态
 * 管理用户登录会话的生命周期
 */
export interface ISessionState {
  /**
   * 是否已经检查过权限
   */
  isAccessChecked: boolean

  /**
   * 登录是否过期
   */
  loginExpired: boolean

  /**
   * 最后登录时间（可选，时间戳）
   */
  lastLoginTime?: number

  /**
   * 最后活跃时间（可选，时间戳）
   */
  lastActiveTime?: number;
}

/**
 * 语言状态
 */
export interface ILanguageState {
  locale: string;
}

/**
 * 加载状态
 */
export interface ILoadingState {
  isLoading: boolean;
  error: boolean | null;
}

/**
 * 主题模式
 * - dark: 暗色模式
 * - light: 亮色模式
 * - system: 跟随系统
 */
export type ThemeMode = 'dark' | 'light' | 'system';

/**
 * 主题状态
 */
export interface IThemeState {
  mode: ThemeMode;
}

/**
 * 用户信息
 */
export interface IUser {
  /**
   * 用户 id
   */
  id: number;

  /**
   * 用户名
   */
  username: string;

  /**
   * 用户昵称
   */
  nickname: string;

  /**
   * 用户实名
   */
  realname: string;

  /**
   * 头像
   */
  avatar: string;

  /**
   * 用户角色
   */
  roles?: string[];

  homePage: string;
}

export interface IAuthenticationState {
  loginLoading: boolean;
  loading: boolean;
  error: string | null;
}

export interface IUserProfileState {
  detail: identityservicev1_User | null;
  loading: boolean;
}

export interface IUserState {
  user: IUser | null;
}
