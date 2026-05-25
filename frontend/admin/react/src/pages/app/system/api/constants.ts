/**
 * API 模块枚举映射常量
 * 页面和 Drawer 共用
 */

/** HTTP 方法颜色映射 */
export const METHOD_COLOR_MAP: Record<string, string> = {
  GET: 'success',
  POST: 'processing',
  PUT: 'warning',
  DELETE: 'error',
  PATCH: 'default',
  HEAD: 'default',
  OPTIONS: 'default',
};

/** 获取 HTTP 方法颜色，未匹配时返回默认颜色 */
export function getMethodColor(method: string | undefined): string {
  return METHOD_COLOR_MAP[method || ''] || '#666';
}
