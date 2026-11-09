import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from '@tanstack/react-query';
import {
  type authenticationservicev1_DeleteLoginPolicyRequest,
  type authenticationservicev1_GetLoginPolicyRequest,
  type authenticationservicev1_ListLoginPolicyResponse,
  type authenticationservicev1_LoginPolicy,
} from '@/api/generated/admin/service/v1';
import { makeUpdateMask, type PaginationQuery, queryClient } from '@/core';
import { apiClient } from '@/api/client';

// ==============================
// 登录策略管理
// ==============================

export function useListLoginPolicies(
  query: PaginationQuery,
  options?: UseQueryOptions<authenticationservicev1_ListLoginPolicyResponse, Error>,
) {
  return useQuery({
    queryKey: ['listLoginPolicies', query],
    queryFn: () => apiClient.loginPolicyService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListLoginPolicies(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listLoginPolicies', params],
    queryFn: () => apiClient.loginPolicyService.List(params.toRawParams()),
    retry: 0,
  });
}

export function useGetLoginPolicy(
  req: authenticationservicev1_GetLoginPolicyRequest,
  options?: UseQueryOptions<authenticationservicev1_LoginPolicy, Error>,
) {
  return useQuery({
    queryKey: ['getLoginPolicy', req],
    queryFn: () => apiClient.loginPolicyService.Get(req),
    ...options,
  });
}

export function useCreateLoginPolicy(
  options?: UseMutationOptions<
    authenticationservicev1_LoginPolicy,
    Error,
    authenticationservicev1_LoginPolicy
  >,
) {
  return useMutation({
    mutationFn: (data) => apiClient.loginPolicyService.Create({ data }),
    ...options,
  });
}

export function useUpdateLoginPolicy(
  options?: UseMutationOptions<{}, Error, { id: string; values: Record<string, any> }>,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: string; values: Record<string, any> }) =>
      apiClient.loginPolicyService.Update({
        id,
        data: { ...values },
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

export function useDeleteLoginPolicy(
  options?: UseMutationOptions<{}, Error, authenticationservicev1_DeleteLoginPolicyRequest>,
) {
  return useMutation({
    mutationFn: (req) => apiClient.loginPolicyService.Delete(req),
    ...options,
  });
}
