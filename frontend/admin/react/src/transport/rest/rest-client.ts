import {RequestClient} from "@/transport/rest/request-client";
import {HttpResponse} from "@/transport/rest/types";
import {
  authenticateResponseInterceptor,
  errorMessageResponseInterceptor
} from "@/transport/rest/preset-interceptors";

// 用于存储获取 token 的函数
let getTokenCallback: (() => string | null) | null = null;

/**
 * 设置获取 Token 的回调函数
 * 需要在应用初始化时调用，传入从 AccessModel 获取 token 的方法
 */
export function setGetTokenCallback(callback: () => string | null) {
  getTokenCallback = callback;
}

export function createRequestClient(baseURL: string) {
  const client = new RequestClient({
    baseURL,
  });

  // 格式化令牌
  function formatToken(token: null | string) {
    return token ? `Bearer ${token}` : null;
  }

  // 请求头处理
  client.addRequestInterceptor({
    fulfilled: (config) => {

      if (getTokenCallback) {
        const token = getTokenCallback();
        console.log('token', token);
        config.headers.Authorization = formatToken(token);
      }
      // if (getLocale) {
      //   config.headers['Accept-Language'] = getLocale();
      // }
      return config as never;
    },
  });

  // response数据解构
  client.addResponseInterceptor({
    fulfilled: (response) => {
      const {data: responseData, status} = response;

      // TODO 根据Kratos进行定制

      if (status >= 200 && status < 400) {
        return responseData;
      }

      const {code} = responseData as HttpResponse;
      if (code !== null) {
        throw Object.assign({}, responseData, {responseData});
      }

      console.error('parse HttpResponse failed!', response);
      throw Object.assign({}, response, {response});
    },
  });

  // token 过期的处理
  client.addResponseInterceptor(
    authenticateResponseInterceptor({
      client,
      doReAuthenticate: async () => {
        console.warn('Token expired, need to re-authenticate');
      },
      doRefreshToken: async () => {
        // 这里需要从 AccessModel 获取 refresh_token
        // 实际使用时需要在应用初始化时设置
        return '';
      },
      enableRefreshToken: true,
      formatToken,
    }),
  );

  // 通用的错误处理，如果没有进入上面的错误处理逻辑，就会进入这里
  client.addResponseInterceptor(
    errorMessageResponseInterceptor(async (msg: string, error) => {
    }),
  );

  return client;
}


export const requestClient = createRequestClient(REACT_APP_API_URL);
