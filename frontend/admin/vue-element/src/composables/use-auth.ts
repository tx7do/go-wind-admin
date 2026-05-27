/**
 * 认证相关 Composable
 * 替代 authentication.store.ts，提供登录/登出/注册/验证码/获取用户信息/权限码等功能
 */
import { ref } from "vue";
import { router } from "@/router";

import { DEFAULT_HOME_PATH } from "@/constants";
import { resetAllStores, useAccessStore, useAppUserStore } from "@/stores";

import { ElNotification } from "element-plus";
import CryptoJS from "crypto-js";

import {
  login as authLogin,
  logout as authLogout,
  registerUser as authRegisterUser,
  generateCaptcha as authGenerateCaptcha,
  getMyPermissionCode,
  getMe,
} from "@/api/composables";
import { i18n } from "@/i18n/setup";
import {
  startRefreshTimer,
  stopRefreshTimer,
  connectSSEServer,
  logoutToLoginPage,
} from "@/composables/use-token-refresh";

import { createI18nGetErrorMsg } from "@/composables/use-request-error-msg";

const t = i18n.global.t;
const getErrorMsg = createI18nGetErrorMsg();

// ==============================
// 登录加载状态（模块级单例）
// ==============================

const loginLoading = ref(false);

// ==============================
// 加密工具
// ==============================

function encryptData(data: string, key: string, iv: string): string {
  const keyHex = CryptoJS.enc.Utf8.parse(key);
  const ivHex = CryptoJS.enc.Utf8.parse(iv);
  const encrypted = CryptoJS.AES.encrypt(data, keyHex, {
    iv: ivHex,
    mode: CryptoJS.mode.CBC,
    padding: CryptoJS.pad.Pkcs7,
  });
  return encrypted.toString();
}

function encryptPassword(password: string): string {
  const key = import.meta.env.VITE_AES_KEY;
  if (!key) {
    throw new Error("VITE_AES_KEY is not set in environment");
  }
  return encryptData(password, key, key);
}

// ==============================
// 常量
// ==============================

const ACCESS_TOKEN_REFRESH_INTERVAL = 90 * 60 * 1000;
const REFRESH_TOKEN_REFRESH_INTERVAL = 12 * 60 * 60 * 1000;

// ==============================
// 核心业务逻辑
// ==============================

async function fetchUserInfo() {
  try {
    const user = await getMe();
    if (!user) return null;
    // identityservicev1_User → UserInfo 适配
    // identityservicev1_User 字段与 BasicUserInfo/UserInfo 兼容，直接作为 UserInfo 使用
    return user as unknown as UserInfo;
  } catch (error) {
    console.error("fetchUserInfo failed:", error);
    await _doLogout();
    return null;
  }
}

async function fetchAccessCodes() {
  return await getMyPermissionCode();
}

async function login(
  params: Recordable<any>,
  onSuccess?: () => Promise<void> | void
): Promise<{ userInfo: null | UserInfo } | null> {
  let userInfo: null | UserInfo = null;
  try {
    loginLoading.value = true;

    const resp = await authLogin({
      username: params.username,
      password: encryptPassword(params.password),
      grant_type: "password",
    });

    const accessToken = (resp as any).access_token;
    const refresh_token = (resp as any).refresh_token;
    let expiresAt = (resp as any).expires_in;
    let refreshExpiresAt = (resp as any).refresh_expires_in;

    const accessStore = useAccessStore();

    const expiresAtSec = Number(expiresAt);
    expiresAt =
      !Number.isFinite(expiresAtSec) || expiresAtSec <= 0
        ? Date.now() + ACCESS_TOKEN_REFRESH_INTERVAL
        : Date.now() + Math.floor(expiresAtSec * 1000);

    const refreshExpiresAtSec = Number(refreshExpiresAt);
    refreshExpiresAt =
      !Number.isFinite(refreshExpiresAtSec) || refreshExpiresAtSec <= 0
        ? Date.now() + REFRESH_TOKEN_REFRESH_INTERVAL
        : Date.now() + Math.floor(refreshExpiresAtSec * 1000);

    if (accessToken) {
      accessStore.setAccessToken(accessToken);
      accessStore.setAccessTokenExpireTime(expiresAt);

      if (refresh_token) {
        accessStore.setRefreshToken(refresh_token);
        accessStore.setRefreshTokenExpireTime(refreshExpiresAt);
        startRefreshTimer();
      }

      const [fetchUserInfoResult, fetchAccessCodeResult] = await Promise.all([
        fetchUserInfo(),
        fetchAccessCodes(),
      ]);

      userInfo = fetchUserInfoResult;
      if (!userInfo) {
        throw new Error(t("core.authentication.loginFailedDesc"));
      }

      const userStore = useAppUserStore();
      userStore.setUserInfo(userInfo);
      accessStore.setAccessCodes(fetchAccessCodeResult.codes ?? []);

      if (accessStore.loginExpired) {
        accessStore.setLoginExpired(false);
      } else {
        onSuccess ? await onSuccess?.() : await router.push(userInfo.homePath || DEFAULT_HOME_PATH);
      }

      if (userInfo?.realname) {
        ElNotification({
          title: t("core.authentication.loginSuccess"),
          message: `${t("core.authentication.loginSuccessDesc")}:${userInfo?.realname}`,
          type: "success",
          duration: 3000,
        });
      }
    }
  } catch (error) {
    await _doLogout();

    // 使用 i18n 翻译错误消息（与 RequestClient 的 getErrorMsg 一致）
    const errorMsg = getErrorMsg(error);
    ElNotification({
      title: t("core.authentication.loginFailed"),
      message: errorMsg,
      type: "error",
    });
    return null;
  } finally {
    loginLoading.value = false;
  }

  return { userInfo };
}

async function _doLogout(redirect: boolean = true) {
  console.log("_doLogout");
  stopRefreshTimer();
  resetAllStores();
  const accessStore = useAccessStore();
  accessStore.setLoginExpired(false);
  loginLoading.value = false;
  await logoutToLoginPage(redirect);
}

async function logout(redirect: boolean = true) {
  const accessStore = useAccessStore();
  try {
    if (accessStore.accessToken !== null && accessStore.accessToken !== "") {
      await authLogout();
    }
  } catch {
    // 忽略错误
  }
  await _doLogout(redirect);
}

async function register(username: string, password: string) {
  return await authRegisterUser({
    username,
    password: encryptPassword(password),
    tenantCode: "master",
  });
}

async function getCaptcha() {
  return await authGenerateCaptcha();
}

async function getUserPermissionCodes() {
  const accessStore = useAccessStore();
  const userStore = useAppUserStore();

  if (userStore.userInfo === null || accessStore.accessCodes === null) {
    const [fetchUserInfoResult, fetchAccessCodeResult] = await Promise.all([
      fetchUserInfo(),
      fetchAccessCodes(),
    ]);
    if (fetchUserInfoResult === null || fetchAccessCodeResult === null) {
      console.warn("setupAccessGuard failed fetch user info:", fetchUserInfoResult);
      return false;
    }
    userStore.setUserInfo(fetchUserInfoResult);

    // 只存权限码，角色码由 userStore.userRoles 管理
    const codes = fetchAccessCodeResult ? (fetchAccessCodeResult.codes ?? []) : [];
    accessStore.setAccessCodes(codes);
  }

  startRefreshTimer();
  connectSSEServer();

  // 返回 true 表示成功获取权限数据
  return true;
}

// ==============================
// 导出
// ==============================

export function useAuth() {
  return {
    loginLoading,
    login,
    logout,
    register,
    getCaptcha,
    fetchUserInfo,
    fetchAccessCodes,
    getUserPermissionCodes,
  };
}
