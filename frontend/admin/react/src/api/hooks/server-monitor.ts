import { useQuery, type UseQueryOptions } from '@tanstack/react-query';
import { type server_monitorservicev1_ServerMonitorInfo } from '@/api/generated/admin/service/v1';
import { queryClient } from '@go-wind/admin-react';
import { apiClient } from '@/api/client';

// ==============================
// 服务监控（只读）：Go 运行时 / 数据库 / 主机
// ==============================

const MONITOR_QUERY_KEY = 'getServerMonitorInfo';

export function useServerMonitorInfo(
  options?: Omit<UseQueryOptions<server_monitorservicev1_ServerMonitorInfo, Error, server_monitorservicev1_ServerMonitorInfo, readonly unknown[]>, 'queryKey' | 'queryFn'>,
) {
  return useQuery({
    queryKey: [MONITOR_QUERY_KEY],
    queryFn: () => apiClient.serverMonitorService.Get({}),
    ...options,
  });
}

export async function fetchServerMonitorInfo() {
  return queryClient.fetchQuery({
    queryKey: [MONITOR_QUERY_KEY],
    queryFn: () => apiClient.serverMonitorService.Get({}),
    retry: 0,
  });
}
