import { useAuthStore } from '@go-wind/admin-react';
import type { PaginationQuery } from '@go-wind/admin-react';

/**
 * 审计日志服务端导出：走 /admin/v1/audit-logs:export（XLSX，突破客户端
 * 聚合导出的 1 万行上限，服务端上限 50 万）。query 经 PaginationQuery 的
 * 同源序列化（contains 转换/空值清理与列表页一致）透传——导出的即当前
 * 搜索看到的。
 */
export async function exportAuditLogsServer(
  type: 'api' | 'data_access' | 'login' | 'operation' | 'permission' | 'policy_evaluation',
  query: PaginationQuery,
): Promise<void> {
  const token = useAuthStore.getState().accessToken;
  const params = new URLSearchParams({ format: 'xlsx', type });
  const queryJson = query.queryString;
  if (queryJson) {
    params.set('query', queryJson);
  }
  // 基址与 transport（RequestClient.init）同源：开发态为 "/"（相对路径，走 vite
  // 代理），生产态为独立 API 域。尾斜杠剥除保证两形态拼出的 URL 都正确。
  const apiBase = (import.meta.env.VITE_API_URL ?? "").replace(/\/$/, "");
  const res = await fetch(
    `${apiBase}/admin/v1/audit-logs:export?${params.toString()}`,
    {
    headers: token ? { authorization: `Bearer ${token}` } : undefined,
  });
  if (!res.ok) {
    throw new Error(`export failed (${res.status})`);
  }
  const blob = await res.blob();
  const disposition = res.headers.get('content-disposition') || '';
  const filename = disposition.match(/filename="(.+)"/)?.[1] ?? `audit-logs-${Date.now()}.xlsx`;
  const a = document.createElement('a');
  a.href = URL.createObjectURL(blob);
  a.download = filename;
  a.click();
  URL.revokeObjectURL(a.href);
}
