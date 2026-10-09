import { useAccessStore } from '@vben/stores';

import type { PaginationQuery } from '#/transport/rest/pagination';

/**
 * 服务端全量导出（XLSX，后端统一行数上限）：URL 指向各资源的 :export 手动锚点，
 * query 经 PaginationQuery 的同源序列化（contains 转换/租户字段清理与列表页一致）
 * 透传——导出的即当前搜索看到的。
 *
 * 手动 fetch 而非走 RequestClient：需要 blob 响应 + 触发浏览器下载，
 * axios 拦截器的 JSON 错误处理对二进制响应不适用。
 */
export async function serverExportFile(
  url: string,
  query: PaginationQuery,
): Promise<void> {
  const accessStore = useAccessStore();
  const token = accessStore.accessToken;
  const params = new URLSearchParams({ format: 'xlsx' });
  const queryJson = query.queryString;
  if (queryJson) {
    params.set('query', queryJson);
  }
  // 基址与 RequestClient.init 同源（直连后端）：裸相对路径会落本端 vite 代理（nitro
  // mock，未启用），404。
  const apiBase = (import.meta.env.VITE_GLOB_API_URL ?? "").replace(/\/$/, "");
  const res = await fetch(
    `${apiBase}/${url}?${params.toString()}`,
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
    `export-${Date.now()}.xlsx`;
  const a = document.createElement('a');
  a.href = URL.createObjectURL(blob);
  a.download = filename;
  a.click();
  URL.revokeObjectURL(a.href);
}
