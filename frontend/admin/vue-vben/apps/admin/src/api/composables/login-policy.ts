import type {
  authenticationservicev1_DeleteLoginPolicyRequest,
  authenticationservicev1_GetLoginPolicyRequest,
  authenticationservicev1_ListLoginPolicyResponse,
  authenticationservicev1_LoginPolicy,
  authenticationservicev1_LoginPolicy_Method as LoginPolicy_Method,
  authenticationservicev1_LoginPolicy_Type as LoginPolicy_Type,
} from '#/api/generated/admin/service/v1';

import { computed } from 'vue';

import { i18n } from '@vben/locales';

import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from '@tanstack/vue-query';

import { apiClient } from '#/api/client';
import { queryClient } from '#/plugins/vue-query';
import { makeUpdateMask, type PaginationQuery } from '#/transport/rest';

const t = i18n.global.t;

// ==============================
// 登录策略管理
// ==============================

export function useListLoginPolicies(
  query: PaginationQuery,
  options?: UseQueryOptions<
    authenticationservicev1_ListLoginPolicyResponse,
    Error
  >,
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
    staleTime: 0,
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
    Record<string, any>
  >,
) {
  return useMutation({
    mutationFn: (values) =>
      apiClient.loginPolicyService.Create({ data: { ...values } as authenticationservicev1_LoginPolicy }),
    ...options,
  });
}

export function useUpdateLoginPolicy(
  options?: UseMutationOptions<
    object,
    Error,
    { id: number; values: Record<string, any> }
  >,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      apiClient.loginPolicyService.Update({
        id,
        data: { ...values },
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

export function useDeleteLoginPolicy(
  options?: UseMutationOptions<
    object,
    Error,
    authenticationservicev1_DeleteLoginPolicyRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.loginPolicyService.Delete(req),
    ...options,
  });
}

// ==============================
// 登录策略枚举与工具函数
// ==============================

export const loginPolicyTypeList = computed(() => [
  { value: 'BLACKLIST', label: t('enum.loginPolicy.type.BLACKLIST') },
  { value: 'WHITELIST', label: t('enum.loginPolicy.type.WHITELIST') },
]);

export const loginPolicyMethodList = computed(() => [
  { value: 'IP', label: t('enum.loginPolicy.method.IP') },
  { value: 'MAC', label: t('enum.loginPolicy.method.MAC') },
  { value: 'REGION', label: t('enum.loginPolicy.method.REGION') },
  { value: 'TIME', label: t('enum.loginPolicy.method.TIME') },
  { value: 'DEVICE', label: t('enum.loginPolicy.method.DEVICE') },
]);

const LOGIN_POLICY_METHOD_COLOR_MAP: Record<string, string> = {
  IP: '#4096FF',
  MAC: '#909399',
  REGION: '#FF9A2E',
  TIME: '#F56C6C',
  DEVICE: '#86909C',
  DEFAULT: '#86909C',
};

export function loginPolicyMethodToColor(methodName: LoginPolicy_Method) {
  return (
    LOGIN_POLICY_METHOD_COLOR_MAP[methodName as string] ||
    LOGIN_POLICY_METHOD_COLOR_MAP.DEFAULT
  );
}

export function loginPolicyTypeToName(typeName: LoginPolicy_Type) {
  const values = loginPolicyTypeList.value;
  const matchedItem = values.find((item) => item.value === typeName);
  return matchedItem ? matchedItem.label : '';
}

export function loginPolicyTypeToColor(typeName: LoginPolicy_Type) {
  switch (typeName) {
    case 'BLACKLIST': {
      return 'red';
    }
    case 'WHITELIST': {
      return 'green';
    }
    default: {
      return 'gray';
    }
  }
}

export function loginPolicyMethodToName(methodName: LoginPolicy_Method) {
  const values = loginPolicyMethodList.value;
  const matchedItem = values.find((item) => item.value === methodName);
  return matchedItem ? matchedItem.label : '';
}
