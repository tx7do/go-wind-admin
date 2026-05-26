import { useQuery, type UseQueryOptions } from "@tanstack/vue-query";
import type {
  auditservicev1_GetLoginAuditLogRequest,
  auditservicev1_ListOperationAuditLogResponse,
  auditservicev1_LoginAuditLog,
} from "@/api/generated/admin/service/v1";
import type { PaginationQuery } from "@/core/transport/rest";
import { listLoginAuditLogs, getLoginAuditLog } from "@/api/service/login-audit-log";
import { queryClient } from "@/plugins/vue-query";

// ==============================
// 登录审计日志
// ==============================

export function useListLoginAuditLogs(
  query: PaginationQuery,
  options?: UseQueryOptions<auditservicev1_ListOperationAuditLogResponse, Error>
) {
  return useQuery({
    queryKey: ["listLoginAuditLogs", query],
    queryFn: () => listLoginAuditLogs(query),
    ...options,
  });
}

export async function fetchListLoginAuditLogs(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ["listLoginAuditLogs", params],
    queryFn: () => listLoginAuditLogs(params),
    retry: 0,
  });
}

export function useGetLoginAuditLog(
  req: auditservicev1_GetLoginAuditLogRequest,
  options?: UseQueryOptions<auditservicev1_LoginAuditLog, Error>
) {
  return useQuery({
    queryKey: ["getLoginAuditLog", req],
    queryFn: () => getLoginAuditLog(req),
    ...options,
  });
}
