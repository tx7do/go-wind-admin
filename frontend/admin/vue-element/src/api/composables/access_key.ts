import { apiClient } from "@/api/client";
import { makeUpdateMask, type PaginationQuery } from "@/core/transport/rest";
import type { access_keyservicev1_AccessKey as AccessKey } from "@/api/generated/admin/service/v1";

/** 分页查询访问凭证 */
export async function fetchListAccessKeys(query: PaginationQuery) {
  return apiClient.accessKeyService.List(query.toRawParams());
}

/** 创建凭证：响应含一次性明文 secret */
export async function createAccessKey(data: Partial<AccessKey>) {
  return apiClient.accessKeyService.Create({ data: data as any });
}

/** 更新凭证（名称/状态/过期时间）；服务端黑名单保护 access_key/secret_hash/tenant_id */
export async function updateAccessKey(id: string, values: Record<string, any>) {
  return apiClient.accessKeyService.Update({
    id,
    data: { ...values } as any,
    updateMask: makeUpdateMask(Object.keys(values ?? {})),
  });
}

/** 重置密钥：生成新 SK 明文返回一次（旧 SK 立即失效于交换） */
export async function resetAccessKeySecret(id: string) {
  return apiClient.accessKeyService.ResetSecret({ id });
}

/** 删除凭证 */
export async function deleteAccessKey(id: string) {
  return apiClient.accessKeyService.Delete({ id });
}
