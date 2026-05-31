import type {
  AxiosInstance,
  AxiosRequestConfig,
  AxiosResponse,
  CreateAxiosDefaults,
} from 'axios';

import type {
  RequestClientCallbacks,
  RequestClientOptions,
  RequestContentType,
} from './types';

import axios from 'axios';
import { defu as merge } from 'defu';

import { FileDownloader } from './modules/downloader';
import { InterceptorManager } from './modules/interceptor';
import { FileUploader } from './modules/uploader';
import {
  authenticateResponseInterceptor,
  errorMessageResponseInterceptor,
} from './preset-interceptors';
import { bindMethods, defaultIdGenerator, getDefaultErrorMsg } from './utils';

class RequestClient {
  // ==========================
  // 静态单例管理
  // ==========================
  private static _instance: null | RequestClient = null; // eslint-disable-line no-use-before-define

  private readonly instance: AxiosInstance;
  public addRequestInterceptor: InterceptorManager['addRequestInterceptor'];

  public addResponseInterceptor: InterceptorManager['addResponseInterceptor'];
  public download: FileDownloader['download'];
  // 是否正在刷新token
  public isRefreshing = false;
  // 刷新token队列
  public refreshTokenQueue: ((token: string) => void)[] = [];

  public upload: FileUploader['upload'];

  /**
   * 构造函数，创建 Axios 实例并注册拦截器
   * @param options - Axios 请求配置
   * @param callbacks - 业务回调接口（token、认证、错误处理），由应用层注入
   */
  constructor(
    options: RequestClientOptions = {},
    callbacks?: RequestClientCallbacks,
  ) {
    // 合并默认配置和传入的配置
    const defaultConfig: CreateAxiosDefaults<RequestContentType> = {
      headers: {
        'Content-Type': 'application/json;charset=utf-8' as RequestContentType,
      },
      timeout: 10_000,
    };
    const { ...axiosConfig } = options;
    const requestConfig = merge(axiosConfig, defaultConfig);
    this.instance = axios.create(requestConfig);

    bindMethods(this);

    // 实例化拦截器管理器
    const interceptorManager = new InterceptorManager(this.instance);
    this.addRequestInterceptor =
      interceptorManager.addRequestInterceptor.bind(interceptorManager);
    this.addResponseInterceptor =
      interceptorManager.addResponseInterceptor.bind(interceptorManager);

    // 实例化文件上传器
    const fileUploader = new FileUploader(this);
    this.upload = fileUploader.upload.bind(fileUploader);
    // 实例化文件下载器
    const fileDownloader = new FileDownloader(this);
    this.download = fileDownloader.download.bind(fileDownloader);

    // ==========================
    // 注册内置拦截器
    // ==========================
    if (callbacks) {
      this.setupInterceptors(callbacks);
    }
  }

  /**
   * 获取全局单例
   */
  static getInstance(): RequestClient {
    if (!RequestClient._instance) {
      throw new Error(
        'RequestClient not initialized. Call RequestClient.init() during bootstrap.',
      );
    }
    return RequestClient._instance;
  }

  /**
   * 初始化全局单例（在 bootstrap 时调用一次）
   */
  static init(baseURL: string, callbacks: RequestClientCallbacks) {
    RequestClient._instance = new RequestClient({ baseURL }, callbacks);
  }

  /**
   * 格式化令牌
   */
  private formatToken(token: null | string) {
    return token ? `Bearer ${token}` : null;
  }

  /**
   * 注册所有内置拦截器
   */
  private setupInterceptors(callbacks: RequestClientCallbacks) {
    this.useTokenInterceptor(callbacks);
    this.useRequestIdInterceptor();
    this.useLocaleInterceptor(callbacks);
    this.useResponseDataInterceptor();
    this.useAuthInterceptor(callbacks);
    this.useErrorMessageInterceptor(callbacks);
  }

  /**
   * 401 认证拦截器：自动刷新 Token 或跳转登录页
   */
  private useAuthInterceptor(callbacks: RequestClientCallbacks) {
    this.addResponseInterceptor(
      authenticateResponseInterceptor({
        client: this,
        doReAuthenticate: async () => {
          console.warn('Token expired, redirecting to login...');
          if (callbacks.onReAuthenticate) {
            await callbacks.onReAuthenticate(true);
          } else {
            console.error(
              'onReAuthenticate callback not set. Call RequestClient.init() during bootstrap.',
            );
          }
        },
        doRefreshToken: async () => {
          if (callbacks.refreshToken) {
            return await callbacks.refreshToken();
          }
          return '';
        },
        enableRefreshToken: true,
        formatToken: this.formatToken.bind(this),
      }),
    );
  }

  /**
   * 统一错误消息拦截器：提取错误文本并回调
   */
  private useErrorMessageInterceptor(callbacks: RequestClientCallbacks) {
    this.addResponseInterceptor(
      errorMessageResponseInterceptor((msg: string) => {
        callbacks.onError?.(msg);
      }, callbacks.getErrorMsg ?? getDefaultErrorMsg),
    );
  }

  /**
   * 请求拦截器：注入 Accept-Language
   */
  private useLocaleInterceptor(callbacks: RequestClientCallbacks) {
    this.addRequestInterceptor({
      fulfilled: (config) => {
        if (callbacks.getLocale) {
          config.headers['Accept-Language'] = callbacks.getLocale();
        }
        return config as never;
      },
    });
  }

  /**
   * 请求拦截器：注入 X-Request-ID 和 XMLHttpRequest 标识
   */
  private useRequestIdInterceptor() {
    this.addRequestInterceptor({
      fulfilled: (config) => {
        const requestId =
          config.headers['X-Request-ID'] || defaultIdGenerator();
        (config as any)._requestId = requestId;
        config.headers['X-Request-ID'] = requestId;
        config.headers['X-Requested-With'] = 'XMLHttpRequest';
        return config as never;
      },
    });
  }

  /**
   * 响应拦截器：解构响应数据（只返回 data 部分）
   */
  private useResponseDataInterceptor() {
    this.addResponseInterceptor({
      fulfilled: (response) => {
        const { data: responseData, status } = response;

        if (status >= 200 && status < 400) {
          return responseData;
        }

        throw Object.assign({}, responseData, { response });
      },
    });
  }

  /**
   * 请求拦截器：注入 Authorization Token
   */
  private useTokenInterceptor(callbacks: RequestClientCallbacks) {
    this.addRequestInterceptor({
      fulfilled: (config) => {
        if (callbacks.getToken) {
          const token = callbacks.getToken();
          config.headers.Authorization = this.formatToken(token);
        }
        return config as never;
      },
    });
  }

  /**
   * DELETE请求方法
   */
  public delete<T = never>(
    url: string,
    config?: AxiosRequestConfig,
  ): Promise<T> {
    return this.request<T>(url, { ...config, method: 'DELETE' });
  }

  /**
   * GET请求方法
   */
  public get<T = never>(url: string, config?: AxiosRequestConfig): Promise<T> {
    return this.request<T>(url, { ...config, method: 'GET' });
  }

  /**
   * POST请求方法
   */
  public post<T = never>(
    url: string,
    data?: never,
    config?: AxiosRequestConfig,
  ): Promise<T> {
    return this.request<T>(url, { ...config, data, method: 'POST' });
  }

  /**
   * PUT请求方法
   */
  public put<T = never>(
    url: string,
    data?: never,
    config?: AxiosRequestConfig,
  ): Promise<T> {
    return this.request<T>(url, { ...config, data, method: 'PUT' });
  }

  /**
   * 通用的请求方法
   */
  public async request<T>(url: string, config: AxiosRequestConfig): Promise<T> {
    try {
      const response: AxiosResponse<T> = await this.instance({
        url,
        ...config,
      });
      return response as T;
    } catch (error: unknown) {
      // @ts-expect-error 忽略类型检查
      throw error.response ? error.response.data : error;
    }
  }
}

export { RequestClient };
