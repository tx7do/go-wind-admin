import type { server_monitorservicev1_ServerMonitorInfo } from '#/api/generated/admin/service/v1';

import { useQuery, type UseQueryOptions } from '@tanstack/vue-query';

import { apiClient } from '#/api/client';
import { queryClient } from '#/plugins/vue-query';

// ==============================
// 服务监控（只读）：Go 运行时 / 数据库 / 主机
// ==============================

const MONITOR_QUERY_KEY = 'getServerMonitorInfo';

export function useServerMonitorInfo(
  options?: UseQueryOptions<server_monitorservicev1_ServerMonitorInfo, Error>,
) {
  return useQuery({
    queryKey: [MONITOR_QUERY_KEY],
    queryFn: () => apiClient.serverMonitorService.Get({}),
    refetchInterval: 10_000,
    ...options,
  });
}

export async function fetchServerMonitorInfo() {
  return queryClient.fetchQuery({
    queryKey: [MONITOR_QUERY_KEY],
    queryFn: () => apiClient.serverMonitorService.Get({}),
    staleTime: 0,
  });
}
