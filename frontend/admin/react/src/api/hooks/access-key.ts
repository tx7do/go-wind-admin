import { useMutation, type UseMutationOptions } from '@tanstack/react-query';
import { apiClient } from '@/api/client';
import { type PaginationQuery, queryClient } from '@/core';
import { makeUpdateMask } from '@/core/transport/rest/utils';
import type {
  access_keyservicev1_AccessKey as AccessKey,
  access_keyservicev1_CreateAccessKeyRequest,
  access_keyservicev1_CreateAccessKeyResponse,
  access_keyservicev1_DeleteAccessKeyRequest,
  access_keyservicev1_ResetAccessKeySecretRequest,
} from '@/api/generated/admin/service/v1';

/** 非组件上下文取列表（ProTable request / 菜单同步构建器） */
export async function fetchListAccessKeys(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listAccessKeys', params],
    queryFn: () => apiClient.accessKeyService.List(params.toRawParams()),
    retry: 0,
  });
}

/** 创建凭证：响应含一次性明文 secret */
export function useCreateAccessKey(
  options?: UseMutationOptions<
    access_keyservicev1_CreateAccessKeyResponse,
    Error,
    access_keyservicev1_CreateAccessKeyRequest
  >,
) {
  return useMutation({
    mutationFn: (data) => apiClient.accessKeyService.Create(data),
    ...options,
  });
}

/** 更新凭证（名称/状态/过期时间）；服务端黑名单保护 access_key/secret_hash/tenant_id */
export function useUpdateAccessKey(
  options?: UseMutationOptions<{}, Error, { id: string; values: Record<string, any> }>,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: string; values: Record<string, any> }) =>
      apiClient.accessKeyService.Update({
        id,
        data: { ...values } as any,
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

export function useDeleteAccessKey(
  options?: UseMutationOptions<{}, Error, access_keyservicev1_DeleteAccessKeyRequest>,
) {
  return useMutation({
    mutationFn: (req) => apiClient.accessKeyService.Delete(req),
    ...options,
  });
}

/** 重置密钥：生成新 SK 明文返回一次（旧 SK 立即失效于交换） */
export function useResetAccessKeySecret(
  options?: UseMutationOptions<
    access_keyservicev1_CreateAccessKeyResponse,
    Error,
    access_keyservicev1_ResetAccessKeySecretRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.accessKeyService.ResetSecret(req),
    ...options,
  });
}

export type { AccessKey };
