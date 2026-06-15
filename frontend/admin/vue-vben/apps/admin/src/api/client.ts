/**
 * ApiClient 单例
 *
 * 生成的代码提供了统一的 ApiClient，它通过 ClientTransport 接口发送请求。
 * 这里将已有的 requestApi（基于 axios 的 RequestClient）适配为 ClientTransport，
 * 保留 token 注入、错误拦截、自动刷新等全部已有逻辑。
 */
import {
  type ClientTransport,
  createApiClient,
} from '#/api/generated/admin/service/v1';
import { requestApi } from '#/transport/rest';

const transport: ClientTransport = {
  unary(path, method, body, _meta) {
    return requestApi({ body, method, path });
  },
  serverStream(path, _meta) {
    // SSE 流由 transport/sse 模块独立管理，此处不应被调用
    throw new Error(`serverStream not supported via ApiClient: ${path}`);
  },
  duplexStream(path, _meta) {
    throw new Error(`duplexStream not supported via ApiClient: ${path}`);
  },
};

export const apiClient = createApiClient(transport);
