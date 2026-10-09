import type {
  AxiosResponse,
  CreateAxiosDefaults,
  InternalAxiosRequestConfig,
} from 'axios';

type RequestResponse<T = any> = AxiosResponse<T>;

type RequestContentType =
  | 'application/json;charset=utf-8'
  | 'application/octet-stream;charset=utf-8'
  | 'application/x-www-form-urlencoded;charset=utf-8'
  | 'multipart/form-data;charset=utf-8';

type RequestClientOptions = CreateAxiosDefaults;

interface RequestInterceptorConfig {
  fulfilled?: (
    config: InternalAxiosRequestConfig,
  ) =>
    | InternalAxiosRequestConfig<any>
    | Promise<InternalAxiosRequestConfig<any>>;
  rejected?: (error: any) => any;
}

interface ResponseInterceptorConfig<T = any> {
  fulfilled?: (
    response: AxiosResponse<T>,
  ) => AxiosResponse | Promise<AxiosResponse>;
  rejected?: (error: any) => any;
}

type MakeErrorMessageFn = (message: string, error: any) => void;

interface HttpResponse {
  code: number;
  reason: string;
  message: string;
  metadata: object;
}

/**
 * 请求客户端回调接口
 * 由应用层（bootstrap）注入具体实现，基础设施层不感知业务逻辑
 */
interface RequestClientCallbacks {
  /** 获取当前 access token */
  getToken: () => null | string;
  /** 获取当前语言标识（如 zh-CN、en-US） */
  getLocale?: () => string;
  /** 刷新 token，返回新的 access token */
  refreshToken: () => Promise<string>;
  /** Token 彻底失效时的处理（登出 + 跳转登录页） */
  onReAuthenticate: (redirect?: boolean) => Promise<void>;
  /** 错误消息处理（弹窗、toast 等） */
  onError?: (message: string) => void;
  /** 从错误对象中提取用户友好的错误消息（可接入 i18n） */
  getErrorMsg?: (error: unknown) => string;
}

export type {
  HttpResponse,
  MakeErrorMessageFn,
  RequestClientCallbacks,
  RequestClientOptions,
  RequestContentType,
  RequestInterceptorConfig,
  RequestResponse,
  ResponseInterceptorConfig,
};
