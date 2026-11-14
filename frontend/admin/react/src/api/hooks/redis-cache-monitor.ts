import { useQuery, type UseQueryOptions } from '@tanstack/react-query';
import { type redis_cacheservicev1_RedisCacheMonitorInfo } from '@/api/generated/admin/service/v1';
import { queryClient } from '@go-wind/admin-react';
import { apiClient } from '@/api/client';

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
    retry: 0,
  });
}
