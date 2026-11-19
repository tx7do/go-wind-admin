/**
 * ApiClient 单例
 *
 * 生成的代码提供了统一的 ApiClient，它通过 ClientTransport 接口发送请求。
 * 这里将已有的 requestApi（基于 axios 的 RequestClient）适配为 ClientTransport，
 * 保留 token 注入、错误拦截、自动刷新等全部已有逻辑。
 */
import { type ClientTransport, createApiClient } from "@/api/generated/admin/service/v1";
import { consumeCaptchaHeaders, requestApi } from "@go-wind/admin-core";
import { getImpersonatedTenant } from "@/utils/tenant-impersonate";

const transport: ClientTransport = {
  unary(path, method, body, _meta) {
    // 消费登录前由 use-auth 设置的验证码 Header（一次性，取后即清）
    const headers = consumeCaptchaHeaders() ?? {};
    // 代租户操作（issue #13）：平台管理员选择了生效租户时携带
    // X-Tenant-Id，后端把数据面与租户闸门切到该租户；空选不携带。
    const tenant = getImpersonatedTenant();
    if (tenant) {
      headers["X-Tenant-Id"] = tenant.id;
    }
    return requestApi({
      body,
      headers: Object.keys(headers).length > 0 ? headers : undefined,
      method,
      path,
    });
  },
  serverStream(path, _meta) {
    // SSE 流由 transport/sse 模块独立管理，此处不应被调用
    throw new Error(`serverStream not supported via ApiClient: ${path}`);
  },
  byteStream(path, _meta) {
    // google.api.HttpBody 字节流（二进制流不走请求适配层）
    throw new Error(`byteStream not supported via ApiClient: ${path}`);
  },
  duplexStream(path, _meta) {
    throw new Error(`duplexStream not supported via ApiClient: ${path}`);
  },
};

export const apiClient = createApiClient(transport);
