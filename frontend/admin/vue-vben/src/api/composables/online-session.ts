import type {
  online_sessionservicev1_ForceLogoutSessionRequest,
  online_sessionservicev1_ForceLogoutSessionResponse,
  online_sessionservicev1_ListOnlineSessionResponse,
  online_sessionservicev1_RevokeMyOnlineSessionRequest,
  online_sessionservicev1_RevokeMyOnlineSessionResponse,
} from '#/api/generated/admin/service/v1';

import {
  useMutation,
  useQuery,
  useQueryClient,
  type UseMutationOptions,
  type UseQueryOptions,
} from '@tanstack/vue-query';

import { apiClient } from '#/api/client';
import { queryClient } from '#/plugins/vue-query';

// ==============================
// 在线会话（在线用户 + 强制下线）
// ==============================

const LIST_KEY = 'listOnlineSessions';
const MY_KEY = 'listMyOnlineSessions';

export type OnlineSessionListParams = {
  keyword?: string;
  page?: number;
  pageSize?: number;
};

export async function fetchListOnlineSessions(params: OnlineSessionListParams) {
  return queryClient.fetchQuery({
    queryKey: [LIST_KEY, params],
    queryFn: () => apiClient.onlineSessionService.ListOnlineSession(params),
    staleTime: 0,
    retry: 0,
  });
}

// ==============================
// 个人中心：我的活跃会话（自助视图）
// ==============================

export function useListMyOnlineSessions(
  options?: UseQueryOptions<online_sessionservicev1_ListOnlineSessionResponse, Error>,
) {
  return useQuery({
    queryKey: [MY_KEY],
    queryFn: () => apiClient.onlineSessionService.ListMyOnlineSession({}),
    ...options,
  });
}

export function useRevokeMyOnlineSession(
  options?: UseMutationOptions<
    online_sessionservicev1_RevokeMyOnlineSessionResponse,
    Error,
    online_sessionservicev1_RevokeMyOnlineSessionRequest
  >,
) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: online_sessionservicev1_RevokeMyOnlineSessionRequest) =>
      apiClient.onlineSessionService.RevokeMyOnlineSession(req),
    // 被踢会话已从 Redis 删除，刷新"我的会话"列表
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [MY_KEY] });
    },
    ...options,
  });
}

export function useForceLogoutSession(
  options?: UseMutationOptions<
    online_sessionservicev1_ForceLogoutSessionResponse,
    Error,
    online_sessionservicev1_ForceLogoutSessionRequest
  >,
) {
  return useMutation({
    mutationFn: (req: online_sessionservicev1_ForceLogoutSessionRequest) =>
      apiClient.onlineSessionService.ForceLogoutSession(req),
    ...options,
  });
}
