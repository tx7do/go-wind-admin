import {
  authenticateResponseInterceptor,
  errorMessageResponseInterceptor,
} from "./preset-interceptors";
import { RequestClient } from "./request-client";
import { defaultIdGenerator } from "./utils";

// 用于存储获取 token 的函数
let getTokenCallback: (() => string | null) | null = null;

// 用于存储错误消息处理函数
let errorHandlerCallback: ((msg: string) => void) | null = null;

/**
 * 设置获取 Token 的回调函数
 * 需要在应用初始化时调用，传入从 AccessModel 获取 token 的方法
 */
export function setGetTokenCallback(callback: () => string | null) {
  getTokenCallback = callback;
}

/**
 * 设置错误消息处理的回调函数
 * 需要在应用初始化时调用，传入使用 App.useApp().message 的处理函数
 */
export function setErrorHandlerCallback(callback: (msg: string) => void) {
  errorHandlerCallback = callback;
}

export function createRequestClient(baseURL: string) {
  const client = new RequestClient({
    baseURL,
  });

  // 格式化令牌
  function formatToken(token: null | string) {
    return token ? `Bearer ${token}` : null;
  }

  // ==========================
  // 请求拦截器
  // ==========================
  client.addRequestInterceptor({
    fulfilled: (config) => {
      if (getTokenCallback) {
        const token = getTokenCallback();
        // console.log('token', token);
        config.headers.Authorization = formatToken(token);
      }
      // if (getLocale) {
      //   config.headers['Accept-Language'] = getLocale();
      // }

      const requestId = config.headers["X-Request-ID"] || defaultIdGenerator();
      (config as any)._requestId = requestId;
      config.headers["X-Request-ID"] = requestId;
      config.headers["X-Requested-With"] = "XMLHttpRequest";

      return config as never;
    },
  });

  // ==========================
  // 响应解构拦截器
  // ==========================
  client.addResponseInterceptor({
    fulfilled: (response) => {
      const { data: responseData, status } = response;

      // HTTP 200-399 正常返回
      if (status >= 200 && status < 400) {
        return responseData;
      }

      // 业务错误 → 直接抛出
      throw Object.assign({}, responseData, { response });
    },
  });

  // ==========================
  // 401 认证拦截器
  // ==========================
  client.addResponseInterceptor(
    authenticateResponseInterceptor({
      client,
      doReAuthenticate: async () => {
        console.warn("Token expired, redirecting to login...");
        // 跳转登录页（重置 store + 路由跳转）
        const { logoutToLoginPage } = await import("@/composables/use-token-refresh");
        await logoutToLoginPage(true);
      },
      doRefreshToken: async () => {
        // 这里需要从 AccessModel 获取 refresh_token
        // 实际使用时需要在应用初始化时设置
        return "";
      },
      enableRefreshToken: true,
      formatToken,
    })
  );

  // ==========================
  // 统一错误消息拦截器
  // ==========================
  client.addResponseInterceptor(
    errorMessageResponseInterceptor((msg: string) => {
      // 如果设置了错误处理回调，则使用它
      if (errorHandlerCallback) {
        errorHandlerCallback(msg);
      }
      // 否则不显示任何消息（由调用方自行处理）
    })
  );

  return client;
}

export const requestClient = createRequestClient(import.meta.env.VITE_API_URL);
