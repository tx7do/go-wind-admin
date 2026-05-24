import { useMutation, type UseMutationOptions } from '@tanstack/react-query';
import {
  type authenticationservicev1_DeleteLoginPolicyRequest,
  type authenticationservicev1_GetLoginPolicyRequest,
  type authenticationservicev1_ListLoginPolicyResponse,
  type authenticationservicev1_LoginPolicy,
  type authenticationservicev1_UpdateLoginPolicyRequest,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery } from '@/core';
import { listLoginPolicies, getLoginPolicy, createLoginPolicy, updateLoginPolicy, deleteLoginPolicy } from '@/api/service/login-policy';

// ==============================
// 登录策略管理
// ==============================

export function useListLoginPolicies(
  options?: UseMutationOptions<
    authenticationservicev1_ListLoginPolicyResponse,
    Error,
    PaginationQuery
  >,
) {
  return useMutation({
    mutationFn: (query) => listLoginPolicies(query),
    ...options,
  });
}

export function useGetLoginPolicy(
  options?: UseMutationOptions<
    authenticationservicev1_LoginPolicy,
    Error,
    authenticationservicev1_GetLoginPolicyRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => getLoginPolicy(req),
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
    mutationFn: (data) => createLoginPolicy(data),
    ...options,
  });
}

export function useUpdateLoginPolicy(
  options?: UseMutationOptions<{}, Error, authenticationservicev1_UpdateLoginPolicyRequest>,
) {
  return useMutation({
    mutationFn: (data) => updateLoginPolicy(data),
    ...options,
  });
}

export function useDeleteLoginPolicy(
  options?: UseMutationOptions<{}, Error, authenticationservicev1_DeleteLoginPolicyRequest>,
) {
  return useMutation({
    mutationFn: (req) => deleteLoginPolicy(req),
    ...options,
  });
}
