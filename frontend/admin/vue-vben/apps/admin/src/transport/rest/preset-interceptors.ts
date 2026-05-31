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

      // / 不是 401 → 直接抛错，交给错误拦截器处理
      if (response?.status !== 401) {
        throw error;
      }

      // 刷新 token 请求本身返回 401 → refresh_token 已失效，直接重新认证
      // 避免将 refresh 请求加入队列导致死锁（队列等待 refresh 完成，但 refresh 本身在队列中）
      const isRefreshTokenRequest = config.url?.includes('/refresh-token');

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
        return Promise.reject(handledError);
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
