import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from "@tanstack/vue-query";
import type {
  authenticationservicev1_DeleteLoginPolicyRequest,
  authenticationservicev1_GetLoginPolicyRequest,
  authenticationservicev1_ListLoginPolicyResponse,
  authenticationservicev1_LoginPolicy,
} from "@/api/generated/admin/service/v1";
import { makeUpdateMask, type PaginationQuery } from "@/core/transport/rest";
import {
  listLoginPolicies,
  getLoginPolicy,
  createLoginPolicy,
  updateLoginPolicy,
  deleteLoginPolicy,
} from "@/api/service/login-policy";
import { queryClient } from "@/plugins/vue-query";

// ==============================
// 登录策略管理
// ==============================

export function useListLoginPolicies(
  query: PaginationQuery,
  options?: UseQueryOptions<authenticationservicev1_ListLoginPolicyResponse, Error>
) {
  return useQuery({
    queryKey: ["listLoginPolicies", query],
    queryFn: () => listLoginPolicies(query),
    ...options,
  });
}

export async function fetchListLoginPolicies(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ["listLoginPolicies", params],
    queryFn: () => listLoginPolicies(params),
    retry: 0,
  });
}

export function useGetLoginPolicy(
  req: authenticationservicev1_GetLoginPolicyRequest,
  options?: UseQueryOptions<authenticationservicev1_LoginPolicy, Error>
) {
  return useQuery({
    queryKey: ["getLoginPolicy", req],
    queryFn: () => getLoginPolicy(req),
    ...options,
  });
}

export function useCreateLoginPolicy(
  options?: UseMutationOptions<
    authenticationservicev1_LoginPolicy,
    Error,
    Record<string, any>
  >,
) {
  return useMutation({
    mutationFn: (values) =>
      createLoginPolicy({ ...values } as authenticationservicev1_LoginPolicy),
    ...options,
  });
}

export function useUpdateLoginPolicy(
  options?: UseMutationOptions<{}, Error, { id: number; values: Record<string, any> }>
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      updateLoginPolicy({
        id,
        data: { ...values },
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

export function useDeleteLoginPolicy(
  options?: UseMutationOptions<{}, Error, authenticationservicev1_DeleteLoginPolicyRequest>
) {
  return useMutation({
    mutationFn: (req) => deleteLoginPolicy(req),
    ...options,
  });
}
