import { useAccessStore } from '@vben/stores';

import type { PaginationQuery } from '#/transport/rest/pagination';

/** 服务端审计导出支持的日志类型（与后端 ExportService 的 type 参数一致） */
export type AuditExportLogType =
  | 'api'
  | 'data_access'
  | 'login'
  | 'operation'
  | 'permission'
  | 'policy_evaluation';

/**
 * 审计日志服务端导出：走 /admin/v1/audit-logs:export（XLSX，突破前端
 * 分页聚合导出的 1 万行上限，服务端上限 50 万）。query 经 PaginationQuery
 * 的同源序列化（contains 转换/空值清理与列表页一致）透传——导出的即当前
 * 搜索看到的。
 *
 * 手动 fetch 而非走 RequestClient：需要 blob 响应 + 触发浏览器下载，
 * axios 拦截器的 JSON 错误处理对二进制响应不适用。
 */
export async function exportAuditLogsServer(
  type: AuditExportLogType,
  query: PaginationQuery,
): Promise<void> {
  const accessStore = useAccessStore();
  const token = accessStore.accessToken;
  const params = new URLSearchParams({ format: 'xlsx', type });
  const queryJson = query.queryString;
  if (queryJson) {
    params.set('query', queryJson);
  }
  // 基址与 RequestClient.init 同源（直连后端）：裸相对路径会落本端 vite 代理（nitro
  // mock，未启用），404。
  const apiBase = (import.meta.env.VITE_GLOB_API_URL ?? "").replace(/\/$/, "");
  const res = await fetch(
    `${apiBase}/admin/v1/audit-logs:export?${params.toString()}`,
    {
    headers: token ? { Authorization: `Bearer ${token}` } : undefined,
  });
  if (!res.ok) {
    throw new Error(`export failed (${res.status})`);
  }
  const blob = await res.blob();
  const disposition = res.headers.get('Content-Disposition') || '';
  const filename =
    disposition.match(/filename="(.+)"/)?.[1] ??
      `audit-logs-${Date.now()}.xlsx`;
  const a = document.createElement('a');
  a.href = URL.createObjectURL(blob);
  a.download = filename;
  a.click();
  URL.revokeObjectURL(a.href);
}
