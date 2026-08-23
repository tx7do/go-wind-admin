import type { MenuRecordRaw } from '@vben-core/typings';
import type { RouteRecordRaw } from 'vue-router';

import { acceptHMRUpdate, defineStore } from 'pinia';

/**
 * @zh_CN 访问令牌类型
 */
type AccessToken = null | string;

/**
 * @zh_CN 访问权限相关状态定义
 */
interface AccessState {
  /**
   * 权限码
   */
  accessCodes: string[];
  /**
   * 可访问的菜单列表
   */
  accessMenus: MenuRecordRaw[];
  /**
   * 可访问的路由列表
   */
  accessRoutes: RouteRecordRaw[];
  /**
   * 登录 accessToken
   */
  accessToken: AccessToken;
  /**
   * accessToken 过期时间戳
   */
  accessTokenExpireTime?: number;
  /**
   * 是否已经检查过权限
   */
  isAccessChecked: boolean;
  /**
   * 登录是否过期
   */
  loginExpired: boolean;

  /**
   * MFA 挑战操作标识：登录密码校验通过但需二次验证时，后端返回。
   * 非空表示当前处于 MFA 挑战待验证阶段，前端据此跳挑战页。
   */
  mfaOperationId: string | null;
}

/**
 * @zh_CN 访问权限相关状态管理
 */
export const useAccessStore = defineStore('core-access', {
  actions: {
    $reset() {
      this.accessToken = null;
      this.accessCodes = [];
      this.accessMenus = [];
      this.accessRoutes = [];
      this.isAccessChecked = false;
      this.loginExpired = false;
      this.mfaOperationId = null;
      this.accessTokenExpireTime = undefined;
    },
    /**
     * @zh_CN 检查 accessToken 是否过期
     */
    checkAccessTokenExpired(): boolean {
      if (!this.accessTokenExpireTime) {
        return true;
      }
      const now = Date.now();
      return now >= this.accessTokenExpireTime;
    },
    setAccessCodes(codes: string[]) {
      this.accessCodes = codes;
    },
    setAccessMenus(menus: MenuRecordRaw[]) {
      this.accessMenus = menus;
    },
    setAccessRoutes(routes: RouteRecordRaw[]) {
      this.accessRoutes = routes;
    },
    setAccessToken(token: AccessToken) {
      this.accessToken = token;
    },
    setAccessTokenExpireTime(accessTokenExpireTime: number) {
      this.accessTokenExpireTime = accessTokenExpireTime;
    },
    setIsAccessChecked(isAccessChecked: boolean) {
      this.isAccessChecked = isAccessChecked;
    },

    setLoginExpired(loginExpired: boolean) {
      this.loginExpired = loginExpired;
    },
  },
  persist: {
    // Token（access/refresh 及其过期时间）全部仅存内存，不落 localStorage。
    // 刷新页面后 token 丢失 → 守卫看到 accessToken 为 null → 跳登录页。
    // accessCodes 是权限码（非 secret），仍持久化以避免刷新后重新拉取。
    pick: ['accessCodes'],
  },
  state: (): AccessState => ({
    accessCodes: [],
    accessMenus: [],
    accessRoutes: [],
    accessToken: null,
    accessTokenExpireTime: undefined,
    isAccessChecked: false,
    loginExpired: false,
    mfaOperationId: null,
  }),
});

// 解决热更新问题
const hot = import.meta.hot;
if (hot) {
  hot.accept(acceptHMRUpdate(useAccessStore, hot));
}
