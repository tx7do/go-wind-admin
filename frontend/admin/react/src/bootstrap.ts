import {
  configureAdminRuntime,
  configureQueryClient,
  connectSSEServer,
  coreResources,
  initI18n,
  setAuthGateway,
  startRefreshTimer,
  RequestClient,
  type HttpResponse,
  useAuthStore,
  usePreferencesStore,
} from '@go-wind/admin-react';
import i18n from 'i18next';
import { resources, allNamespaces } from '@/locales';
import {
  loginMutation,
  logoutMutation,
  refreshTokenMutation,
  verifyMfaMutation,
} from '@/api';
import { apiClient } from '@/api/client';
import { fetchUserProfile } from '@/api/hooks/user-profile';
import { encryptPasswordEnvelope } from '@/utils';

/**
 * 应用启动初始化
 */
export async function bootstrap() {
  // 运行时开关注入（包内不读 import.meta.env——lib 构建会把它静态替换成包构建时的值）
  // apiBaseUrl：serverExportFile 拼服务端全量导出地址用（与 RequestClient.init 同源）
  configureAdminRuntime({ isDev: import.meta.env.DEV, apiBaseUrl: import.meta.env.VITE_API_URL });
  configureQueryClient({ isDev: import.meta.env.DEV });

  // 认证数据面注入：机制（token 内存态/刷新调度/MFA 挑战/登出清理）在包内，
  // 后端调用与口令加密（密钥属宿主 env）全部由宿主实现 AuthGateway
  setAuthGateway({
    login: (params) => loginMutation.execute(params as Parameters<typeof loginMutation.execute>[0]),
    verifyMfa: (params) => verifyMfaMutation.execute(params),
    ssoLogin: (params) => apiClient.authenticationService.SsoLogin(params),
    logout: () => logoutMutation.execute({}),
    refresh: () => refreshTokenMutation.execute(undefined),
    fetchProfile: async () => (await fetchUserProfile()) as unknown as Parameters<
      ReturnType<typeof useAuthStore.getState>['setUserInfo']
    >[0],
    encryptPassword: (plain) => encryptPasswordEnvelope(plain),
  });

  await _initI18n();

  // 全局错误观测：未处理的 promise 拒绝/运行时异常若无人接管会静默丢失，
  // 统一加 [GlobalError] 前缀落 console，保证浏览器端排查有迹可循。
  window.addEventListener('unhandledrejection', (event) => {
    console.error('[GlobalError] unhandled rejection:', event.reason);
  });
  window.addEventListener('error', (event) => {
    console.error('[GlobalError] uncaught error:', event.error ?? event.message);
  });

  // 可放全局初始化逻辑
  console.log('✅ 应用启动初始化完成');
}

/**
 * 从 refresh_exp cookie 读取 refresh token 的过期时间戳（Unix 秒）。
 * 返回毫秒级时间戳或 null（cookie 不存在/已过期）。
 */
function getRefreshExpireAt(): number | null {
  const match = document.cookie
    .split('; ')
    .find((row) => row.startsWith('refresh_exp='));
  if (!match) return null;
  const parts = match.split('=');
  if (parts.length < 2) return null;
  const val = parseInt(parts[1], 10);
  if (!Number.isFinite(val) || val <= 0) return null;
  return val * 1000;
}

async function _initI18n() {
  // 从 preferences 获取初始语言
  const initialLocale = usePreferencesStore.getState().preferences.app.locale;

  // i18n 资源 = 包内置框架命名空间（common/auth/editor/preferences）+ 宿主命名空间，
  // 同名时宿主覆盖包内置（routes 是宿主业务路由标题，必须宿主侧提供）
  const mergedResources = {
    'zh-CN': { ...coreResources['zh-CN'], ...resources['zh-CN'] },
    'en-US': { ...coreResources['en-US'], ...resources['en-US'] },
  } as Record<string, Record<string, Record<string, unknown>>>;

  // 初始化 i18n（传入初始语言与合并资源）
  await initI18n(initialLocale, {
    resources: mergedResources,
    namespaces: Object.keys(mergedResources['zh-CN'] ?? allNamespaces),
    missingKeyWarn: import.meta.env.DEV,
  });

  // 注入 RequestClient 回调（业务层 → 基础设施层）
  // 必须在 initStores 之后，因为 getToken 依赖 accessStore
  RequestClient.init(import.meta.env.VITE_API_URL, {
    getToken: () => useAuthStore.getState().accessToken,
    getLocale: () => i18n.language,
    refreshToken: async () => useAuthStore.getState().refreshToken(),
    onReAuthenticate: async () => {
      useAuthStore.getState().forceLogout();
    },
    onError: (msg) => console.error('[RequestClient]', msg),
    getErrorMsg: getErrorMsg,
  });

  // SSE 基础地址由宿主从自身 env 解析后传入（包内零 env 直读），
  // 后续 token 刷新的重连会复用该地址
  const sseBaseUrl = import.meta.env.VITE_SSE_URL ?? '/api/sse';

  // 页面刷新后的会话恢复：
  // access token 仅存内存，刷新页面后丢失。refresh token 现以 HttpOnly Cookie 传输，
  // 刷新后仍在。若检测到有效的 refresh_exp cookie，静默调 /refresh-token 换回新 access token，
  // 使用户无感知地恢复会话，无需重新登录。
  const { accessToken } = useAuthStore.getState();
  if (accessToken) {
    // 内存中仍有 access token（如 SPA 内导航），仅需恢复定时器
    startRefreshTimer();
  } else {
    // 内存无 access token（页面刷新），尝试静默恢复
    const refreshExpireAt = getRefreshExpireAt();
    const now = Date.now();
    if (refreshExpireAt && refreshExpireAt > now) {
      try {
        // refresh token cookie 由浏览器自动携带，无需前端传参
        const newToken = await useAuthStore.getState().refreshToken();
        if (newToken) {
          // 恢复用户信息（刷新页面后内存丢失）。profile 拉取失败不能记录后继续——
          // AuthGuard 只查 token，放行半残会话（有 token 无 userInfo）会让所有依赖
          // userInfo.id 的页面把字面量 "undefined" 当用户 ID 发出去（后端解析 500），
          // 而补拉要等下一个刷新定时器 tick（最长约 90 分钟）。短暂重试消化后端
          // 重启窗口期，仍失败则强制登出收敛到干净的重登录。
          type Profile = Parameters<
            ReturnType<typeof useAuthStore.getState>['setUserInfo']
          >[0];
          let profile: Profile | null = null;
          let lastError: unknown = null;
          for (let attempt = 1; attempt <= 3 && !profile; attempt += 1) {
            try {
              profile = (await fetchUserProfile()) as unknown as Profile;
            } catch (e) {
              lastError = e;
              console.error(
                `[Bootstrap] silent restore: fetch user profile failed (attempt ${attempt}/3):`,
                e,
              );
            }
            if (!profile && attempt < 3) {
              await new Promise((resolve) => setTimeout(resolve, 400));
            }
          }
          if (profile) {
            useAuthStore.getState().setUserInfo(profile);
            // 恢复 SSE 连接与刷新定时器
            connectSSEServer(sseBaseUrl);
            startRefreshTimer();
            console.log('[Bootstrap] session silently restored via refresh cookie');
          } else {
            console.error(
              '[Bootstrap] silent restore: user profile unavailable after retries, forcing logout:',
              lastError,
            );
            useAuthStore.getState().forceLogout();
          }
        }
      } catch (e) {
        // 必须带出真实错误：固定话术会把 refresh 端点 500/网络错等真因
        // 掩盖成"cookie 无效"（认证链路排查教训，三端同款问题一并清理）
        console.error('[Bootstrap] silent restore failed (refresh cookie 可能无效或过期):', e);
      }
    }
  }
}

/**
 * 获取错误提示文本。
 *
 * 错误提示统一基于后端返回的 reason 字段经 i18n 翻译得到，
 * 不再依赖后端 message 字段（该字段后续将被移除）。
 * - 网络错误 / 超时：走对应的 i18n 文案
 * - reason 命中 i18n 映射：返回翻译文案
 * - 其余（无 reason 或无对应翻译）：返回通用兜底文案
 */
export function getErrorMsg(error: unknown) {
  const i18nPrefix = 'request.';

  // 网络错误
  const errStr = String(error ?? '');
  if (errStr.includes('Network Error')) {
    return i18n.t(i18nPrefix + 'error.networkError', { ns: 'common' });
  }

  // 超时
  if (
    error &&
    typeof error === 'object' &&
    'message' in error &&
    String(error.message).includes('timeout')
  ) {
    return i18n.t(i18nPrefix + 'error.timeout', { ns: 'common' });
  }

  // 获取后端返回数据
  const resData =
    error &&
    typeof error === 'object' &&
    'response' in error &&
    error.response &&
    typeof error.response === 'object' &&
    'data' in error.response
      ? (error.response.data as HttpResponse)
      : undefined;

  // 仅基于 reason 查询 i18n；不再回退到后端 message
  if (resData) {
    const { reason } = resData;
    if (reason) {
      const key = i18nPrefix + `reason.${reason}`;
      // 使用 i18n.exists() 时需要指定命名空间
      if (i18n.exists(key, { ns: 'common' })) {
        return i18n.t(key, { ns: 'common' });
      }
    }
  }

  // 兜底
  return i18n.t(i18nPrefix + 'error.unknownError', { ns: 'common' });
}
