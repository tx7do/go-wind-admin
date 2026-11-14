import { useAuthStore } from '@go-wind/admin-react';
import type { PaginationQuery } from '@go-wind/admin-react';

/**
 * 服务端全量导出（XLSX，后端统一行数上限）：query 走 PaginationQuery 的
 * 同源序列化（contains 转换/租户字段清理与列表页一致），导出的即当前搜索看到的。
 * 仅接入与各列表读接口同权限语义的 :export 锚点。
 */
export async function serverExportFile(url: string, query: PaginationQuery): Promise<void> {
  const token = useAuthStore.getState().accessToken;
  const qs = new URLSearchParams({ format: 'xlsx' });
  const queryJson = query.queryString;
  if (queryJson) {
    qs.set('query', queryJson);
  }
  // 基址与 transport（RequestClient.init）同源：开发态为 "/"（相对路径，走 vite
  // 代理），生产态为独立 API 域。尾斜杠剥除保证两形态拼出的 URL 都正确。
  const apiBase = (import.meta.env.VITE_API_URL ?? "").replace(/\/$/, "");
  const res = await fetch(`${apiBase}/${url}?${qs.toString()}`, {
    headers: token ? { authorization: `Bearer ${token}` } : undefined,
  });
  if (!res.ok) {
    throw new Error(`export failed (${res.status})`);
  }
  const blob = await res.blob();
  const disposition = res.headers.get('content-disposition') || '';
  const filename = disposition.match(/filename="(.+)"/)?.[1] ?? `export-${Date.now()}.xlsx`;
  const a = document.createElement('a');
  a.href = URL.createObjectURL(blob);
  a.download = filename;
  a.click();
  URL.revokeObjectURL(a.href);
}
