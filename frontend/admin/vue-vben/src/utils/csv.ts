/**
 * 审计日志导出工具（vben 端）：按传入行/表头生成带 BOM 的 CSV 或
 * XLSX（exceljs）并触发下载。全量聚合由各页面组合 fetchList* +
 * 分页循环实现（上限自控）。
 */

import ExcelJS from 'exceljs';

export interface CsvColumn {
  title: string;
  key: string;
}

export type TableExportFormat = 'csv' | 'xlsx';

function escapeCsvCell(value: unknown): string {
  if (value === null || value === undefined) return '';
  const str = String(value);
  if (/[",\n\r]/.test(str)) {
    return `"${str.replace(/"/g, '""')}"`;
  }
  return str;
}

function triggerDownload(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
}

export function downloadCsv(filename: string, columns: CsvColumn[], rows: any[]): void {
  const header = columns.map((c) => escapeCsvCell(c.title)).join(',');
  const lines = rows.map((row) =>
    columns.map((c) => escapeCsvCell((row as any)?.[c.key])).join(','),
  );
  // BOM：保证 Excel 打开中文不乱码
  const content = `\uFEFF${header}\n${lines.join('\n')}`;
  const blob = new Blob([content], { type: 'text/csv;charset=utf-8' });
  triggerDownload(blob, filename);
}

async function downloadXlsx(filename: string, columns: CsvColumn[], rows: any[]): Promise<void> {
  const workbook = new ExcelJS.Workbook();
  const sheet = workbook.addWorksheet('Sheet1');
  sheet.addRow(columns.map((c) => c.title));
  rows.forEach((row) => {
    sheet.addRow(columns.map((c) => (row as any)?.[c.key] ?? ''));
  });
  const buffer = await workbook.xlsx.writeBuffer();
  const blob = new Blob([buffer as ArrayBuffer], {
    type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
  });
  triggerDownload(blob, filename);
}

/**
 * 按格式分发导出下载。传入文件名不带扩展名，由本函数按格式追加。
 */
export async function downloadTableFile(
  format: TableExportFormat,
  filename: string,
  columns: CsvColumn[],
  rows: any[],
): Promise<void> {
  if (format === 'xlsx') {
    await downloadXlsx(`${filename}.xlsx`, columns, rows);
    return;
  }
  downloadCsv(`${filename}.csv`, columns, rows);
}
