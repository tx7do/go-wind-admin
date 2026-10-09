import type { redis_cacheservicev1_RedisCacheMonitorInfo } from '#/api/generated/admin/service/v1';

import { useQuery, type UseQueryOptions } from '@tanstack/vue-query';

import { apiClient } from '#/api/client';
import { queryClient } from '#/plugins/vue-query';

// ==============================
// Redis 缓存监控（只读）
// ==============================

const MONITOR_QUERY_KEY = 'getRedisCacheMonitorInfo';

export function useRedisCacheMonitorInfo(
  options?: UseQueryOptions<redis_cacheservicev1_RedisCacheMonitorInfo, Error>,
) {
  return useQuery({
    queryKey: [MONITOR_QUERY_KEY],
    queryFn: () => apiClient.redisCacheMonitorService.Get({}),
    ...options,
  });
}

export async function fetchRedisCacheMonitorInfo() {
  return queryClient.fetchQuery({
    queryKey: [MONITOR_QUERY_KEY],
    queryFn: () => apiClient.redisCacheMonitorService.Get({}),
    staleTime: 0,
    retry: 0,
  });
}
