import type { RequestClient } from './request-client';
import type { MakeErrorMessageFn, ResponseInterceptorConfig } from './types';

import axios from 'axios';

import { getDefaultErrorMsg } from './utils';

/**
 * 认证响应拦截器：处理 401 错误，支持自动刷新 token 和重新认证
 * @param client 请求客户端实例
 * @param doReAuthenticate 重新认证函数，返回 Promise<void>
 * @param doRefreshToken 刷新 token 函数，返回 Promise<string>，成功时返回新的 token
 * @param enableRefreshToken 是否启用刷新 token 功能
 * @param formatToken 格式化 token 的函数，接受原始 token 字符串，返回格式化后的 token 字符串（如添加 "Bearer " 前缀），如果返回 null 则不设置 Authorization 头
 * @returns 响应拦截器配置对象
 */
/** 排队等待刷新的超时兜底：刷新请求自身有 10s 客户端超时，此值仅防异常挂起 */
const REFRESH_QUEUE_TIMEOUT_MS = 30_000;

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

      // 判断是否为 401 认证错误：HTTP 401 或业务层 code=401
      const isHttp401 = response?.status === 401;
      const isBiz401 =
        response?.status >= 200 &&
        response?.status < 400 &&
        response?.data?.code === 401;

      // 不是 401 → 直接抛错，交给错误拦截器处理
      if (!isHttp401 && !isBiz401) {
        throw error;
      }

      // 刷新 token 请求本身返回 401 → refresh_token 已失效，直接重新认证
      // 避免将 refresh 请求加入队列导致死锁（队列等待 refresh 完成，但 refresh 本身在队列中）
      const isRefreshTokenRequest =
        config.url?.includes('/refresh-token') ||
        config.url?.includes('/refresh_token');

      // 登录请求返回 401 → 密码错误或凭证无效，属于业务错误
      // 直接抛出，不触发 token 刷新 / 重新认证逻辑，
      // 让错误正常传递给调用方（authLogin 的 catch）处理
      const isLoginRequest =
        config.url?.includes('/login') && !isRefreshTokenRequest;

      if (isLoginRequest) {
        throw error;
      }

      // 判断是否启用了 refreshToken 功能
      // 如果没有启用或者已经是重试请求了，直接跳转到重新登录
      if (
        !enableRefreshToken ||
        config.__isRetryRequest ||
        isRefreshTokenRequest
      ) {
        await doReAuthenticate();
        // 标记错误已由认证拦截器处理

        throw Object.assign(error, {
          __handledByAuthInterceptor: true,
        });
      }

      // 如果正在刷新 token，则将请求加入队列，等待刷新完成。
      // 防挂死兜底（历史故障：队列无超时无 reject，刷新链路一旦挂起，
      // 全部数据页渐进进入永久 loading，只能重登恢复）：
      // 排队重试带 __isRetryRequest——重试再 401 不再入刷，防循环；
      // 排队条目 30s 超时摘除并拒绝——刷新链路异常挂起时请求快速失败。
      if (client.isRefreshing) {
        config.__isRetryRequest = true;
        return new Promise((resolve, reject) => {
          const callback = (newToken: string) => {
            clearTimeout(timer);
            // 刷新失败时队列会以空串唤醒：此时重发注定再吃 401，
            // 直接以已处理的认证错误拒绝，交由调用方与 doReAuthenticate 收尾
            if (!newToken) {
              reject(
                Object.assign(new Error('Authentication required'), {
                  __handledByAuthInterceptor: true,
                }),
              );
              return;
            }
            config.headers.Authorization = formatToken(newToken);
            resolve(client.request(config.url, { ...config }));
          };
          const timer = setTimeout(() => {
            client.refreshTokenQueue = client.refreshTokenQueue.filter((cb) => cb !== callback);
            console.warn('[Auth] 刷新等待超时，排队请求快速失败:', config.url);
            reject(
              Object.assign(new Error('Authentication refresh wait timeout'), {
                __handledByAuthInterceptor: true,
              }),
            );
          }, REFRESH_QUEUE_TIMEOUT_MS);
          client.refreshTokenQueue.push(callback);
        });
      }

      // 标记开始刷新 token
      client.isRefreshing = true;
      // 标记当前请求为重试请求，避免无限循环
      config.__isRetryRequest = true;

      try {
        const newToken = await doRefreshToken();

        // doRefreshToken 返回空字符串说明刷新已失败（refreshToken 内部 catch 了错误），
        // 此时 reauthenticate 已被执行，不应继续用空 token 重试原始请求
        if (!newToken) {
          throw new Error('Refresh token returned empty');
        }

        // 处理队列中的请求
        client.refreshTokenQueue.forEach((callback) => callback(newToken));
        // 清空队列
        client.refreshTokenQueue = [];

        return client.request(error.config.url, { ...error.config });
      } catch (refreshError) {
        // 刷新失败：队列中的请求以空串唤醒（回调内会拒绝而非重发），
        // 由 doReAuthenticate 统一收尾跳登录
        client.refreshTokenQueue.forEach((callback) => callback(''));
        client.refreshTokenQueue = [];

        console.error('Refresh token failed:', refreshError);

        await doReAuthenticate();

        // 标记错误已由认证拦截器处理，不继续抛出错误，避免触发错误消息拦截器
        const handledError = Object.assign(
          new Error('Authentication required'),
          {
            __handledByAuthInterceptor: true,
          },
        );
        throw handledError;
      } finally {
        client.isRefreshing = false;
      }
    },
  };
};

/**
 * 错误消息拦截器：提取错误文本并回调
 * @param makeErrorMessage 错误消息回调函数
 * @param getErrorMsg 获取错误消息函数，默认为 getDefaultErrorMsg
 * @returns 响应拦截器配置对象
 */
export const errorMessageResponseInterceptor = (
  makeErrorMessage?: MakeErrorMessageFn,
  getErrorMsg: (error: unknown) => string = getDefaultErrorMsg,
): ResponseInterceptorConfig => {
  return {
    rejected: (error: unknown) => {
      // 取消请求不处理
      if (axios.isCancel(error)) {
        return Promise.reject(error);
      }

      // 已由认证拦截器处理的错误，不弹窗
      if (
        error &&
        typeof error === 'object' &&
        '__handledByAuthInterceptor' in error
      ) {
        return Promise.reject(error);
      }

      // 统一获取错误信息并弹窗
      const msg = getErrorMsg(error);
      makeErrorMessage?.(msg, error);

      return Promise.reject(error);
    },
  };
};
