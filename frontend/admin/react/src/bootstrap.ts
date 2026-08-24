import { initI18n } from '@/core/i18n';
import { usePreferencesStore } from '@/core/preferences';
import { type HttpResponse, RequestClient } from '@/core/transport/rest';
import i18n from 'i18next';
import { useAuthStore } from '@/stores';
import {
  connectSSEServer,
  startRefreshTimer,
} from '@/hooks/useTokenRefresh';
import { fetchUserProfile } from '@/api/hooks/user-profile';

/**
 * 应用启动初始化
 */
export async function bootstrap() {
  await _initI18n();

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

  // 初始化 i18n（传入初始语言）
  await initI18n(initialLocale);

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
          // 恢复用户信息（刷新页面后内存丢失）
          try {
            const profile = (await fetchUserProfile()) as unknown as Parameters<
              ReturnType<typeof useAuthStore.getState>['setUserInfo']
            >[0];
            if (profile) {
              useAuthStore.getState().setUserInfo(profile);
            }
          } catch (e) {
            console.error('[Bootstrap] silent restore: fetch user profile failed:', e);
          }
          // 恢复 SSE 连接与刷新定时器
          connectSSEServer();
          startRefreshTimer();
          console.log('[Bootstrap] session silently restored via refresh cookie');
        }
      } catch (e) {
        // refresh cookie 无效/过期/后端拒验 → 保持登出态，静默忽略
        console.log('[Bootstrap] silent restore skipped: refresh cookie invalid or expired');
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
