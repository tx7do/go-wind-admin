import { PaginationQuery } from '@/core';
import { message } from 'antd';
import ExcelJS from 'exceljs';

/**
 * 审计日志导出（客户端聚合实现）：
 * 按当前搜索条件分页拉取（上限 maxRows），生成 CSV 或 XLSX（exceljs）触发下载。
 * 上限保护：1 万行足够日常审计查阅；全量归档走后端 JSONL 审计归档任务。
 */

export interface ExportColumn {
  key: string;
  title: string;
}

export interface ExportAuditOptions {
  fetcher: (query: PaginationQuery) => Promise<{ items?: any[]; total?: number | string }>;
  filename: string;
  columns: ExportColumn[];
  /** 当前 ProTable 搜索参数（不含分页字段） */
  params?: Record<string, any>;
  maxRows?: number;
}

export type AuditExportFormat = 'csv' | 'xlsx';

const PAGE_SIZE = 1000;
const DEFAULT_MAX_ROWS = 10_000;

function escapeCsvCell(value: unknown): string {
  if (value === null || value === undefined) return '';
  let str: string;
  if (value instanceof Date) str = value.toISOString();
  else if (typeof value === 'object') str = JSON.stringify(value);
  else str = String(value);

  // 值含逗号/引号/换行时包双引号，内部引号翻倍（RFC 4180）
  if (/[",\n\r]/.test(str)) {
    return `"${str.replace(/"/g, '""')}"`;
  }
  return str;
}

function buildCsv(columns: ExportColumn[], rows: Record<string, unknown>[]): string {
  const header = columns.map((c) => escapeCsvCell(c.title)).join(',');
  const lines = rows.map((row) =>
    columns.map((c) => escapeCsvCell(row?.[c.key])).join(','),
  );
  // BOM：保证 Excel 打开中文不乱码
  return `\uFEFF${header}\n${lines.join('\n')}`;
}

function downloadTextFile(filename: string, content: string, mime = 'text/csv;charset=utf-8') {
  const blob = new Blob([content], { type: mime });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
}

function downloadBinaryFile(filename: string, content: ArrayBuffer, mime: string) {
  const blob = new Blob([content], { type: mime });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
}

/** 剔除传入文件名的既有扩展名后替换为当前格式扩展名。 */
function withExtension(filename: string, format: AuditExportFormat): string {
  const base = filename.replace(/\.(csv|xlsx)$/i, '');
  return `${base}.${format}`;
}

/** 分页聚合拉取（受 maxRows 上限约束），CSV 与 XLSX 共用。 */
async function fetchAllRows(options: ExportAuditOptions): Promise<Record<string, unknown>[]> {
  const { fetcher, columns, params = {}, maxRows = DEFAULT_MAX_ROWS } = options;
  if (columns.length === 0) return [];

  // 剔除 ProTable 分页字段，保留搜索/排序条件
  const { current: _c, pageSize: _p, ...filters } = params as Record<string, any>;

  const allRows: Record<string, unknown>[] = [];
  let page = 1;
  let total = Infinity;

  while (allRows.length < maxRows && (page - 1) * PAGE_SIZE < total) {
    const query = new PaginationQuery({
      paging: { page, pageSize: PAGE_SIZE },
      formValues: Object.keys(filters).length > 0 ? filters : undefined,
    });
    const resp = await fetcher(query);
    const items = resp.items ?? [];
    // total 为 uint64（proto3 JSON 下是字符串），参与分页运算前转回 number
    total = Number(resp.total ?? allRows.length + items.length);
    allRows.push(...items);
    if (items.length === 0) break; // 防御：空页避免死循环
    page += 1;
  }

  return allRows.slice(0, maxRows);
}

async function exportRowsAsXlsx(options: ExportAuditOptions, rows: Record<string, unknown>[]) {
  const workbook = new ExcelJS.Workbook();
  const sheet = workbook.addWorksheet('Sheet1');
  sheet.addRow(options.columns.map((c) => c.title));
  rows.forEach((row) => {
    sheet.addRow(options.columns.map((c) => row?.[c.key] ?? ''));
  });
  const buffer = await workbook.xlsx.writeBuffer();
  downloadBinaryFile(
    withExtension(options.filename, 'xlsx'),
    buffer as ArrayBuffer,
    'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
  );
}

/** 执行导出：分页拉全量（受限）→ 按格式生成 → 下载。返回导出行数供提示。 */
export async function exportAuditLogs(
  options: ExportAuditOptions,
  format: AuditExportFormat,
): Promise<number> {
  const rows = await fetchAllRows(options);
  if (format === 'xlsx') {
    await exportRowsAsXlsx(options, rows);
  } else {
    downloadTextFile(withExtension(options.filename, 'csv'), buildCsv(options.columns, rows));
  }
  return rows.length;
}

/** @deprecated 使用 exportAuditLogs(options, format) */
export async function exportAuditLogsToCsv(options: ExportAuditOptions): Promise<number> {
  return exportAuditLogs(options, 'csv');
}

/** 页面级导出处理器：包装错误提示与行数反馈 */
export async function handleAuditExport(
  options: ExportAuditOptions,
  successHint?: (count: number) => void,
): Promise<void> {
  try {
    const count = await exportAuditLogsToCsv(options);
    successHint?.(count);
  } catch (error: any) {
    message.error(error?.message || `导出失败：${error}`);
  }
}
