import { initI18n } from '@/core/i18n';
import { usePreferencesStore } from '@/core/preferences';
import { type HttpResponse, RequestClient } from '@/core/transport/rest';
import i18n from 'i18next';
import { useAuthStore } from '@/stores';

/**
 * 应用启动初始化
 */
export async function bootstrap() {
  await _initI18n();

  // 可放全局初始化逻辑
  console.log('✅ 应用启动初始化完成');
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
}

/**
 * 按优先级获取错误提示文本
 * 1. reason → i18n error.xxx
 * 2. reason 无翻译 → 使用 message
 * 3. 都无 → 使用 status → i18n status.xxx
 * 4. 都无 → fallback
 */
export function getErrorMsg(error: unknown) {
  const i18nPrefix = 'request.';

  // 网络错误
  const errStr = String(error ?? '');
  if (errStr.includes('Network Error')) {
    return i18n.t(i18nPrefix + 'error.networkError');
  }

  // 超时
  if (
    error &&
    typeof error === 'object' &&
    'message' in error &&
    String(error.message).includes('timeout')
  ) {
    return i18n.t(i18nPrefix + 'error.timeout');
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

  if (!resData) {
    return i18n.t(i18nPrefix + 'error.unknownError');
  }

  const { reason, message, code } = resData;

  console.log('resData:', resData);

  // =========================================
  // 1. 优先：reason → request.reason.xxx
  // =========================================
  if (reason) {
    const key = i18nPrefix + `reason.${reason}`;
    // 使用 i18n.exists() 时需要指定命名空间
    if (i18n.exists(key, { ns: 'common' })) {
      return i18n.t(key, { ns: 'common' });
    }
  }

  // =========================================
  // 2. reason 无翻译 → 使用后端 message
  // =========================================
  if (message?.trim()) {
    return message.trim();
  }

  // =========================================
  // 3. 都没有 → 使用 code 查 status
  // =========================================
  if (code) {
    const statusKey = i18nPrefix + `status.${code}`;
    if (i18n.exists(statusKey, { ns: 'common' })) {
      return i18n.t(statusKey, { ns: 'common' });
    }
  }

  // =========================================
  // 4. 全部失败 → 兜底
  // =========================================
  return i18n.t(i18nPrefix + 'error.unknownError', { ns: 'common' });
}
