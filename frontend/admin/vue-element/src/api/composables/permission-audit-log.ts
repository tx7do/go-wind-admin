import { useQuery, type UseQueryOptions } from "@tanstack/vue-query";
import type {
  auditservicev1_GetPermissionAuditLogRequest,
  auditservicev1_ListPermissionAuditLogResponse,
  auditservicev1_PermissionAuditLog,
} from "@/api/generated/admin/service/v1";
import type { PaginationQuery } from "@/core/transport/rest";
import { listPermissionAuditLogs, getPermissionAuditLog } from "@/api/service/permission-audit-log";
import { queryClient } from "@/plugins/vue-query";

// ==============================
// 权限审计日志
// ==============================

export function useListPermissionAuditLogs(
  query: PaginationQuery,
  options?: UseQueryOptions<auditservicev1_ListPermissionAuditLogResponse, Error>
) {
  return useQuery({
    queryKey: ["listPermissionAuditLogs", query],
    queryFn: () => listPermissionAuditLogs(query),
    ...options,
  });
}

export async function fetchListPermissionAuditLogs(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ["listPermissionAuditLogs", params],
    queryFn: () => listPermissionAuditLogs(params),
    retry: 0,
  });
}

export function useGetPermissionAuditLog(
  req: auditservicev1_GetPermissionAuditLogRequest,
  options?: UseQueryOptions<auditservicev1_PermissionAuditLog, Error>
) {
  return useQuery({
    queryKey: ["getPermissionAuditLog", req],
    queryFn: () => getPermissionAuditLog(req),
    ...options,
  });
}
