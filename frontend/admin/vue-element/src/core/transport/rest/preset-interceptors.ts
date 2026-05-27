import axios from "axios";

import type { RequestClient } from "./request-client";
import type { MakeErrorMessageFn, ResponseInterceptorConfig } from "./types";
import { getErrorMsg } from "./utils";

export const authenticateResponseInterceptor = ({
  client,
  doReAuthenticate,
  doRefreshToken,
  enableRefreshToken,
  formatToken,
}: {
  client: RequestClient;
  doReAuthenticate: () => Promise<void>;
  doRefreshToken: () => Promise<string>;
  enableRefreshToken: boolean;
  formatToken: (token: string) => null | string;
}): ResponseInterceptorConfig => {
  return {
    rejected: async (error) => {
      const { config, response } = error;

      /// 不是 401 → 直接抛错，交给错误拦截器处理
      if (response?.status !== 401) {
        throw error;
      }

      // 判断是否启用了 refreshToken 功能
      // 如果没有启用或者已经是重试请求了，直接跳转到重新登录
      if (!enableRefreshToken || config.__isRetryRequest) {
        await doReAuthenticate();
        // 标记错误已由认证拦截器处理
        const handledError = Object.assign(error, {
          __handledByAuthInterceptor: true,
        });
        throw handledError;
      }

      // 如果正在刷新 token，则将请求加入队列，等待刷新完成
      if (client.isRefreshing) {
        return new Promise((resolve) => {
          client.refreshTokenQueue.push((newToken: string) => {
            config.headers.Authorization = formatToken(newToken);
            resolve(client.request(config.url, { ...config }));
          });
        });
      }

      // 标记开始刷新 token
      client.isRefreshing = true;
      // 标记当前请求为重试请求，避免无限循环
      config.__isRetryRequest = true;

      try {
        const newToken = await doRefreshToken();

        // 处理队列中的请求
        client.refreshTokenQueue.forEach((callback) => callback(newToken));
        // 清空队列
        client.refreshTokenQueue = [];

        return client.request(error.config.url, { ...error.config });
      } catch (refreshError) {
        // 如果刷新 token 失败，处理错误（如强制登出或跳转登录页面）
        client.refreshTokenQueue.forEach((callback) => callback(""));
        client.refreshTokenQueue = [];

        console.error("Refresh token failed:", refreshError);

        await doReAuthenticate();

        // 标记错误已由认证拦截器处理，不继续抛出错误，避免触发错误消息拦截器
        const handledError = Object.assign(new Error("Authentication required"), {
          __handledByAuthInterceptor: true,
        });
        return Promise.reject(handledError);
      } finally {
        client.isRefreshing = false;
      }
    },
  };
};

export const errorMessageResponseInterceptor = (
  makeErrorMessage?: MakeErrorMessageFn
): ResponseInterceptorConfig => {
  return {
    rejected: (error: unknown) => {
      // 取消请求不处理
      if (axios.isCancel(error)) {
        return Promise.reject(error);
      }

      // 已由认证拦截器处理的错误，不弹窗
      if (error && typeof error === "object" && "__handledByAuthInterceptor" in error) {
        return Promise.reject(error);
      }

      // 统一获取错误信息并弹窗
      const msg = getErrorMsg(error);
      makeErrorMessage?.(msg, error);

      return Promise.reject(error);
    },
  };
};
