import { useMutation, useQuery } from '@tanstack/vue-query';

import { apiClient } from '#/api/client';

/**
 * 租户自助用量：当前操作者所属租户的套餐用量与配额（服务端钉定租户，
 * 不接受传参）。配额硬限制（USER_LIMIT/STORAGE 403）发生时，租户管理员
 * 在此看到原因。平台用户（tenantId==0）返回空 Usage，调用方不展示。
 */
export function useMyTenantUsage() {
  return useQuery({
    queryKey: ['myTenantUsage'],
    queryFn: () => apiClient.myTenantUsageService.GetMyTenantUsage({}),
    staleTime: 60_000,
  });
}

// 预留：将来如需手动刷新入口
export function useRefreshMyTenantUsage() {
  return useMutation({
    mutationFn: async () => apiClient.myTenantUsageService.GetMyTenantUsage({}),
  });
}
