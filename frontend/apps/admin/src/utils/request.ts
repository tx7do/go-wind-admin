/**
 * 该文件可自行根据业务逻辑进行调整
 */
import type { HttpResponse } from '@vben/request';

import { useAppConfig } from '@vben/hooks';
import { preferences } from '@vben/preferences';
import {
  authenticateResponseInterceptor,
  errorMessageResponseInterceptor,
  RequestClient,
} from '@vben/request';
import { useAccessStore } from '@vben/stores';

import { message } from 'ant-design-vue';

import { useAuthStore } from '#/stores';

const { apiURL } = useAppConfig(import.meta.env, import.meta.env.PROD);

/**
 * 默认的请求 ID 生成器
 */
function defaultIdGenerator(): string {
  try {
    // 优先使用标准 API
    const rnd = (globalThis as any)?.crypto?.randomUUID?.();
    if (typeof rnd === 'string' && rnd.length > 0) return rnd;
  } catch {
    // ignore
  }
  // 降级方案
  return Math.random().toString(36).slice(2) + Date.now().toString(36);
}

function createRequestClient(baseURL: string) {
  const client = new RequestClient({
    baseURL,
  });

  /**
   * 重新认证逻辑
   */
  async function doReAuthenticate() {
    const authStore = useAuthStore();
    await authStore.reauthenticate();
  }

  /**
   * 刷新token逻辑
   */
  async function doRefreshToken() {
    const authStore = useAuthStore();
    return await authStore.refreshToken();
  }

  /**
   * 格式化令牌
   * @param token
   */
  function formatToken(token: null | string) {
    return token ? `Bearer ${token}` : null;
  }

  // 请求头处理
  client.addRequestInterceptor({
    fulfilled: async (config) => {
      const accessStore = useAccessStore();

      const requestId = config.headers['X-Request-ID'] || defaultIdGenerator();
      (config as any)._requestId = requestId;

      config.headers.Authorization = formatToken(accessStore.accessToken);
      config.headers['Accept-Language'] = preferences.app.locale;
      config.headers['X-Request-ID'] = requestId;
      config.headers['X-Requested-With'] = 'XMLHttpRequest';
      return config;
    },
  });

  // response数据解构
  client.addResponseInterceptor({
    fulfilled: (response) => {
      const { data: responseData, status } = response;

      // TODO 根据Kratos进行定制

      if (status >= 200 && status < 400) {
        return responseData;
      }

      const { code } = responseData as HttpResponse;
      if (code !== null) {
        throw Object.assign({}, responseData, { responseData });
      }

      console.error('parse HttpResponse failed!', response);
      throw Object.assign({}, response, { response });
    },
  });

  // token过期的处理
  client.addResponseInterceptor(
    authenticateResponseInterceptor({
      client,
      doReAuthenticate,
      doRefreshToken,
      enableRefreshToken: preferences.app.enableRefreshToken,
      formatToken,
    }),
  );

  // 通用的错误处理,如果没有进入上面的错误处理逻辑，就会进入这里
  client.addResponseInterceptor(
    errorMessageResponseInterceptor(async (msg: string, error) => {
      // 这里可以根据业务进行定制,你可以拿到 error 内的信息进行定制化处理，根据不同的 code 做不同的提示，而不是直接使用 message.error 提示 msg
      // 当前mock接口返回的错误字段是 error 或者 message
      const responseData = error?.response?.data ?? {};
      const errorMessage = responseData?.error ?? responseData?.message ?? '';
      // 如果没有错误信息，则会根据状态码进行提示
      await message.error(errorMessage || msg);
    }),
  );

  return client;
}

export const requestClient = createRequestClient(apiURL);

export const baseRequestClient = new RequestClient({ baseURL: apiURL });

type Request = {
  body: null | string;
  method: string;
  path: string;
};

export type Paging = { page?: number; pageSize?: number } | undefined;

/**
 * 通用请求处理器
 */
export function requestClientRequestHandler({ path, method, body }: Request) {
  return requestClient.request(path, {
    method,
    data: body,
  } as any);
}
