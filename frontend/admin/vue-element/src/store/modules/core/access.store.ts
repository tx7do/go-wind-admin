import type { RouteRecordRaw } from "vue-router";

import { acceptHMRUpdate, defineStore } from "pinia";

import type { MenuRecordRaw } from "@/types";

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
   * 登录 accessToken
   */
  refreshToken: AccessToken;

  /**
   * refreshToken 过期时间戳
   */
  refreshTokenExpireTime?: number;
}

/**
 * @zh_CN 访问权限相关状态管理
 */
export const useAccessStore = defineStore("core-access", {
  actions: {
    $reset() {
      this.accessToken = null;
      this.refreshToken = null;
      this.accessCodes = [];
      this.accessMenus = [];
      this.accessRoutes = [];
      this.isAccessChecked = false;
      this.loginExpired = false;
      this.accessTokenExpireTime = undefined;
      this.refreshTokenExpireTime = undefined;
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
    /**
     * @zh_CN 检查 refreshToken 是否过期
     */
    checkRefreshTokenExpired(): boolean {
      if (!this.refreshTokenExpireTime) {
        return true;
      }
      const now = Date.now();
      return now >= this.refreshTokenExpireTime;
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

    setRefreshToken(token: AccessToken) {
      this.refreshToken = token;
    },

    setRefreshTokenExpireTime(refreshTokenExpireTime: number) {
      this.refreshTokenExpireTime = refreshTokenExpireTime;
    },
  },
  persist: {
    // 持久化
    pick: [
      "accessToken",
      "refreshToken",
      "accessCodes",
      "refreshTokenExpireTime",
      "accessTokenExpireTime",
    ],
  },
  state: (): AccessState => ({
    accessCodes: [],
    accessMenus: [],
    accessRoutes: [],
    accessToken: null,
    accessTokenExpireTime: undefined,
    isAccessChecked: false,
    loginExpired: false,
    refreshToken: null,
    refreshTokenExpireTime: undefined,
  }),
});

// 解决热更新问题
const hot = import.meta.hot;
if (hot) {
  hot.accept(acceptHMRUpdate(useAccessStore, hot));
}
