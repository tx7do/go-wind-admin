import type {RequestClient} from './request-client';
import type {MakeErrorMessageFn, ResponseInterceptorConfig} from './types';

import axios from 'axios';

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
            const {config, response} = error;
            // 如果不是 401 错误，直接抛出异常
            if (response?.status !== 401) {
                throw error;
            }
            // 判断是否启用了 refreshToken 功能
            // 如果没有启用或者已经是重试请求了，直接跳转到重新登录
            if (!enableRefreshToken || config.__isRetryRequest) {
                await doReAuthenticate();
                throw error;
            }
            // 如果正在刷新 token，则将请求加入队列，等待刷新完成
            if (client.isRefreshing) {
                return new Promise((resolve) => {
                    client.refreshTokenQueue.push((newToken: string) => {
                        config.headers.Authorization = formatToken(newToken);
                        resolve(client.request(config.url, {...config}));
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

                return client.request(error.config.url, {...error.config});
            } catch (refreshError) {
                // 如果刷新 token 失败，处理错误（如强制登出或跳转登录页面）
                client.refreshTokenQueue.forEach((callback) => callback(''));
                client.refreshTokenQueue = [];
                console.error('Refresh token failed, please login again.');
                await doReAuthenticate();

                throw refreshError;
            } finally {
                client.isRefreshing = false;
            }
        },
    };
};

export const errorMessageResponseInterceptor = (
    makeErrorMessage?: MakeErrorMessageFn,
): ResponseInterceptorConfig => {
    return {
        rejected: (error: unknown) => {
            if (axios.isCancel(error)) {
                return Promise.reject(error);
            }

            const err = String(error ?? '');
            let errMsg = '';
            // TODO: 需要集成 i18n 翻译
            if (err?.includes('Network Error')) {
                errMsg = 'network.error';
            } else if (error && typeof error === 'object' && 'message' in error && String(error.message).includes('timeout')) {
                errMsg = 'error.timeout';
            }
            if (errMsg) {
                makeErrorMessage?.(errMsg, error);
                return Promise.reject(error);
            }

            let errorMessage = '';
            const status = error && typeof error === 'object' && 'response' in error
                ? (error as { response?: { status?: number } }).response?.status
                : undefined;

            switch (status) {
                case 400: {
                    errorMessage = 'error.badRequest';
                    break;
                }
                case 401: {
                    errorMessage = 'error.unauthorized';
                    break;
                }
                case 403: {
                    errorMessage = 'error.forbidden';
                    break;
                }
                case 404: {
                    errorMessage = 'error.notFound';
                    break;
                }
                case 408: {
                    errorMessage = 'error.requestTimeout';
                    break;
                }
                default: {
                    errorMessage = 'error.internalServerError';
                }
            }
            makeErrorMessage?.(errorMessage, error);
            return Promise.reject(error);
        },
    };
};
