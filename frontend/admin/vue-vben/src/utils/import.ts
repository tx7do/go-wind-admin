/**
 * Excel 导入工具（客户端解析）：模板生成、行解析与逐行落库。
 * 逐行走页面既有 create 变体——{ data } 包裹、proto 校验、租户隔离、
 * 审计埋点全部复用既有链路，与手工创建无差别。
 */

import ExcelJS from 'exceljs';

export interface ImportField {
  /** 模板表头/文件表头（面向用户的字段标签） */
  label: string;
  /** 落库字段名（create 请求体字段） */
  prop: string;
}

/** 生成导入模板（表头行 = 可导入字段标签），返回 { data } 供下载。 */
export async function generateImportTemplate(
  fields: ImportField[],
): Promise<{ data: ArrayBuffer }> {
  const workbook = new ExcelJS.Workbook();
  const sheet = workbook.addWorksheet('Sheet1');
  sheet.addRow(fields.map((f) => f.label));
  const buffer = await workbook.xlsx.writeBuffer();
  return { data: buffer as ArrayBuffer };
}

/**
 * 解析 xlsx 并逐行落库：表头按字段标签映射回 prop，空行/无可识别字段的行跳过；
 * 任一行失败即抛错（带行号），由调用方提示。返回实际导入行数。
 */
export async function runImportFile(
  file: File,
  fields: ImportField[],
  createRow: (values: Record<string, any>) => Promise<any>,
): Promise<number> {
  const buffer = await file.arrayBuffer();
  const workbook = new ExcelJS.Workbook();
  const wb = await workbook.xlsx.load(buffer);
  const ws = wb.getWorksheet(1) ?? wb.worksheets[0];
  if (!ws || ws.rowCount < 2) {
    throw new Error('no data parsed');
  }
  const labels: string[] = [];
  ws.getRow(1).eachCell((cell) => labels.push(String(cell.value ?? '')));
  const labelToProp = new Map(fields.map((f) => [f.label, f.prop]));

  let imported = 0;
  for (let i = 2; i <= ws.rowCount; i++) {
    const row: Record<string, any> = {};
    ws.getRow(i).eachCell((cell, col) => {
      const label = labels[col - 1];
      if (label) row[label] = cell.value;
    });
    const mapped: Record<string, any> = {};
    Object.keys(row).forEach((label) => {
      const prop = labelToProp.get(label);
      if (prop) mapped[prop] = row[label];
    });
    if (Object.keys(mapped).length === 0) continue;
    try {
      await createRow(mapped);
      imported += 1;
    } catch (err: any) {
      throw new Error(`row ${i}: ${err?.message ?? String(err)}`);
    }
  }
  return imported;
}
