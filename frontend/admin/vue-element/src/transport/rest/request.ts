import { useAuthStore, useAccessStore, useAppStore } from "@/store";
import { defaultPreferences } from "@/settings";

import type { HttpResponse, Request } from "./types";
import {
  authenticateResponseInterceptor,
  errorMessageResponseInterceptor,
} from "./preset-interceptors";
import { RequestClient } from "./request-client";
import { ElMessage } from "element-plus";

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
    return (await authStore.refreshToken()) || "";
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
      const appStore = useAppStore();

      // 修正accessToken类型，保证Authorization头格式正确
      const accessToken =
        typeof accessStore.accessToken === "string"
          ? accessStore.accessToken
          : (accessStore.accessToken ?? null);
      config.headers.Authorization = formatToken(accessToken);
      config.headers["Accept-Language"] = appStore.locale;
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

      console.error("parse HttpResponse failed!", response);
      throw Object.assign({}, response, { response });
    },
  });

  // token过期的处理
  client.addResponseInterceptor(
    authenticateResponseInterceptor({
      client,
      doReAuthenticate,
      doRefreshToken,
      enableRefreshToken: defaultPreferences.enableRefreshToken,
      formatToken,
    })
  );

  // 通用的错误处理,如果没有进入上面的错误处理逻辑，就会进入这里
  client.addResponseInterceptor(
    errorMessageResponseInterceptor(async (msg: string, error) => {
      // 这里可以根据业务进行定制,你可以拿到 error 内的信息进行定制化处理，根据不同的 code 做不同的提示，而不是直接使用 message.error 提示 msg
      // 当前mock接口返回的错误字段是 error 或者 message
      const responseData = error?.response?.data ?? {};
      const errorMessage = responseData?.error ?? responseData?.message ?? "";
      // 如果没有错误信息，则会根据状态码进行提示
      ElMessage.error(errorMessage || msg);
    })
  );

  return client;
}

export const requestClient = createRequestClient("");

export const baseRequestClient = new RequestClient({ baseURL: "" });

/**
 * 通用请求处理器
 */
export function requestClientRequestHandler({ path, method, body }: Request) {
  return requestClient.request(path, {
    method,
    data: body,
  } as any);
}
