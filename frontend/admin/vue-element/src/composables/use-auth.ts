/**
 * 认证相关 Composable
 * 替代 authentication.store.ts，提供登录/登出/注册/验证码/获取用户信息/权限码等功能
 */
import { ref } from "vue";
import { apiClient } from "@/api/client";
import { encryptPasswordEnvelope } from "@/utils";
import { router } from "@/router";

import { DEFAULT_HOME_PATH } from "@/constants";
import { resetAllStores, useAccessStore, useAppUserStore } from "@/stores";

import { ElNotification } from "element-plus";

import {
  login as authLogin,
  logout as authLogout,
  generateCaptcha as authGenerateCaptcha,
  getMyPermissionCode,
  getMe,
  verifyMfaMutation,
} from "@/api/composables";
import { i18n } from "@/core/i18n";
import { setCaptchaHeaders } from "@/core/transport/rest";
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
// 网络异常标记类
// ==============================

class NetworkError extends Error {
  constructor() {
    super("Network Error");
    this.name = "NetworkError";
  }
}

export { NetworkError };

// ==============================
// 登录加载状态（模块级单例）
// ==============================

const loginLoading = ref(false);

// ==============================
// 常量
// ==============================

const ACCESS_TOKEN_REFRESH_INTERVAL = 90 * 60 * 1000;

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
    // 网络异常时重新抛出，让 getUserPermissionCodes 的 catch 统一处理
    if (isNetworkError(error)) {
      throw new NetworkError();
    }
    // 其他错误（如 401）返回 null，让上层决定
    return null;
  }
}

async function fetchAccessCodes() {
  return await getMyPermissionCode();
}

async function login(
  params: Record<string, any>,
  onSuccess?: () => Promise<void> | void
): Promise<{ userInfo: null | UserInfo } | null> {
  // catch 分支必然 return，无需初始化为 null（no-useless-assignment）
  let userInfo: null | UserInfo;
  // MFA 分支要在请求后透传 redirect 到挑战页；请求返回后路由可能被副作用
  // 先行更新（响应式时序竞态）丢掉 query，故必须在发起请求前捕获。
  const redirectAtEntry = (router.currentRoute.value.query.redirect as string) || "";
  try {
    loginLoading.value = true;

    // 若表单携带验证码，先设置一次性 Header（由 transport.unary 消费）
    if (params.captchaId && params.captchaCode) {
      setCaptchaHeaders(params.captchaId, params.captchaCode);
    }

    const resp = await authLogin({
      username: params.username,
      password: await encryptPasswordEnvelope(params.password),
      tenant_code: params.tenant_code,
      grant_type: "password",
    });

    const accessStore = useAccessStore();

    // ===== MFA 闸门：后端在密码校验通过、需二次验证时返回 mfa_operation_id，access_token 为空。
    // 不写任何 token，记录 operation_id 并跳转 MFA 挑战页（路由守卫亦据此强制跳转）。
    if ((resp as any).mfa_operation_id) {
      accessStore.mfaOperationId = (resp as any).mfa_operation_id as string;
      // 携带 redirect 到挑战页（用请求前捕获值），验证通过后回到原目标页
      await router.push({
        name: "MfaChallenge",
        query: redirectAtEntry ? { redirect: redirectAtEntry } : {},
      });
      return { userInfo: null };
    }

    userInfo = await applySuccessfulLogin(resp as any, onSuccess);
  } catch (error) {
    await _doLogout();

    // 使用 i18n 翻译错误消息（与 RequestClient 的 getErrorMsg 一致）
    const errorMsg = getErrorMsg(error);
    // 通知给用户的只是翻译后的文案，原始错误对象必须同步落控制台供排查
    console.error("login 失败:", error);
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

// applySuccessfulLogin 处理"已拿到含真 token 的 LoginResponse"后的统一流程：
// 存 token → 拉用户信息/权限码 → 跳转。
// 登录成功与 MFA 验证成功都复用此函数。返回 userInfo（失败抛错）。
async function applySuccessfulLogin(
  resp: any,
  onSuccess?: () => Promise<void> | void
): Promise<UserInfo | null> {
  const accessToken = resp.access_token;
  let expiresAt = resp.expires_in;

  const accessStore = useAccessStore();

  const expiresAtSec = Number(expiresAt);
  expiresAt =
    !Number.isFinite(expiresAtSec) || expiresAtSec <= 0
      ? Date.now() + ACCESS_TOKEN_REFRESH_INTERVAL
      : Date.now() + Math.floor(expiresAtSec * 1000);

  if (!accessToken) {
    return null;
  }

  accessStore.setAccessToken(accessToken);
  accessStore.setAccessTokenExpireTime(expiresAt);

  // refresh token 通过 HttpOnly Cookie 传输，前端不可读也不存内存。
  // 页面刷新后由 bootstrap 静默恢复（凭 cookie 调 /refresh-token 换新 access token）。
  startRefreshTimer();

  const [fetchUserInfoResult, fetchAccessCodeResult] = await Promise.all([
    fetchUserInfo(),
    fetchAccessCodes(),
  ]);

  const userInfo = fetchUserInfoResult;
  if (!userInfo) {
    throw new Error(t("core.authentication.loginFailedDesc"));
  }

  const userStore = useAppUserStore();
  userStore.setUserInfo(userInfo);
  accessStore.setAccessCodes(fetchAccessCodeResult.codes ?? []);
  accessStore.setHiddenFields(fetchAccessCodeResult.hiddenFields ?? []);

  if (accessStore.loginExpired) {
    accessStore.setLoginExpired(false);
  } else {
    if (onSuccess) {
      await onSuccess();
    } else {
      await router.push(userInfo.homePath || DEFAULT_HOME_PATH);
    }
  }

  if (userInfo?.realname) {
    ElNotification({
      title: t("core.authentication.loginSuccess"),
      message: `${t("core.authentication.loginSuccessDesc")}:${userInfo?.realname}`,
      type: "success",
      duration: 3000,
    });
  }

  return userInfo;
}

// completeMfaChallenge 用 operation_id + TOTP 码调后端验证，通过则复用 applySuccessfulLogin。
async function completeMfaChallenge(
  totpCode: string,
  onSuccess?: () => Promise<void> | void
): Promise<{ userInfo: null | UserInfo } | null> {
  // catch 分支必然 return，无需初始化为 null（no-useless-assignment）
  let userInfo: null | UserInfo;
  const accessStore = useAccessStore();
  const opId = accessStore.mfaOperationId;
  if (!opId) {
    return null;
  }
  try {
    loginLoading.value = true;
    const resp = await verifyMfaMutation.execute({
      operationId: opId,
      totpCode,
    } as any);
    accessStore.mfaOperationId = null;
    userInfo = await applySuccessfulLogin(resp as any, onSuccess);
  } catch (error) {
    await _doLogout();
    const errorMsg = getErrorMsg(error);
    console.error("MFA 验证失败:", error);
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

// completeSsoLogin OIDC 回调换令牌：code+state 调 SsoLogin（后端已验证 state
// 并签发标准 JWT 对），复用 applySuccessfulLogin（存 token/拉用户/跳转）。
// refresh token 由后端经 HttpOnly Cookie 下发，与密码登录同形。
async function completeSsoLogin(
  code: string,
  state: string,
  onSuccess?: () => Promise<void> | void
): Promise<UserInfo | null> {
  const resp = await apiClient.authenticationService.SsoLogin({ code, state });
  return applySuccessfulLogin(resp, onSuccess);
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
  } catch (error) {
    // 本地清理照常执行，但服务端登出失败必须留痕（可能是 token 失效/网络/权限问题）
    console.error("服务端登出请求失败（已继续本地清理）:", error);
  }
  await _doLogout(redirect);
}

async function getCaptcha() {
  return await authGenerateCaptcha();
}

/**
 * 判断是否为网络异常（非业务错误）
 * 网络异常的特征：AxiosError 且 code 为 ERR_NETWORK，或 message 包含 Network Error
 */
function isNetworkError(error: unknown): boolean {
  if (!error || typeof error !== "object") return false;
  const err = error as Record<string, unknown>;
  // AxiosError: code === 'ERR_NETWORK'
  if (err.code === "ERR_NETWORK") return true;
  // 兜底：message 检测
  if (typeof err.message === "string" && err.message.includes("Network Error")) return true;
  return false;
}

async function getUserPermissionCodes() {
  const accessStore = useAccessStore();
  const userStore = useAppUserStore();

  if (userStore.userInfo === null || accessStore.accessCodes === null) {
    try {
      const [fetchUserInfoResult, fetchAccessCodeResult] = await Promise.all([
        fetchUserInfo(),
        fetchAccessCodes(),
      ]);
      if (fetchUserInfoResult === null || fetchAccessCodeResult === null) {
        console.warn("getUserPermissionCodes: 获取用户信息/权限码返回空", fetchUserInfoResult);
        return false;
      }
      userStore.setUserInfo(fetchUserInfoResult);

      // 只存权限码，角色码由 userStore.userRoles 管理
      const codes = fetchAccessCodeResult ? (fetchAccessCodeResult.codes ?? []) : [];
      accessStore.setAccessCodes(codes);
      accessStore.setHiddenFields(fetchAccessCodeResult?.hiddenFields ?? []);
    } catch (error: unknown) {
      // 网络异常：抛出特定标记，让路由守卫跳转错误页而非白屏
      if (isNetworkError(error)) {
        throw new NetworkError();
      }
      // 其他错误（如 401/403）：标记为认证失败。错误本体必须留痕，
      // 否则守卫只看到"认证失败"跳登录页，真因（500/权限配置错等）无从排查
      console.error("getUserPermissionCodes: 获取用户信息/权限码失败:", error);
      return false;
    }
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
    completeMfaChallenge,
    completeSsoLogin,
    logout,
    // 强制登出：纯前端清理+跳转，不调后端 logout API。
    // 用于改密成功等 token 已被后端吊销的场景，避免登出请求再吃 401。
    forceLogout: () => _doLogout(true),
    getCaptcha,
    fetchUserInfo,
    fetchAccessCodes,
    getUserPermissionCodes,
  };
}
