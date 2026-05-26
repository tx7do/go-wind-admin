import { useQuery, type UseQueryOptions } from "@tanstack/vue-query";
import type {
  auditservicev1_GetOperationAuditLogRequest,
  auditservicev1_ListOperationAuditLogResponse,
  auditservicev1_OperationAuditLog,
} from "@/api/generated/admin/service/v1";
import type { PaginationQuery } from "@/core/transport/rest";
import { listOperationAuditLogs, getOperationAuditLog } from "@/api/service/operation-audit-log";
import { queryClient } from "@/plugins/vue-query";

// ==============================
// 操作审计日志
// ==============================

export function useListOperationAuditLogs(
  query: PaginationQuery,
  options?: UseQueryOptions<auditservicev1_ListOperationAuditLogResponse, Error>
) {
  return useQuery({
    queryKey: ["listOperationAuditLogs", query],
    queryFn: () => listOperationAuditLogs(query),
    ...options,
  });
}

export async function fetchListOperationAuditLogs(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ["listOperationAuditLogs", params],
    queryFn: () => listOperationAuditLogs(params),
    retry: 0,
  });
}

export function useGetOperationAuditLog(
  req: auditservicev1_GetOperationAuditLogRequest,
  options?: UseQueryOptions<auditservicev1_OperationAuditLog, Error>
) {
  return useQuery({
    queryKey: ["getOperationAuditLog", req],
    queryFn: () => getOperationAuditLog(req),
    ...options,
  });
}
