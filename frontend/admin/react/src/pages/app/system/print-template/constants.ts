import type { printservicev1_PrintTemplate as PrintTemplate } from '@/api/generated/admin/service/v1';

/** 纸张尺寸展示标签（冗余列直显，免解析 payload JSON） */
export const PAPER_SIZE_OPTIONS = ['A4', 'A3', 'A5', 'Letter', 'Legal', '80x60', '60x40', '57mm', '80mm'] as const;

export const ORIENTATION_OPTIONS = ['portrait', 'landscape'] as const;

export function orientationLabel(t: (k: string) => string, value?: string | null): string {
  if (value === 'landscape') return t('landscape');
  return t('portrait');
}

/** 由示例数据对象派生设计器字段面板（PrintBusinessField）：
 *  对象拍平成点路径，数组记为 list 类型并派生 items[].xxx 子路径。
 *  M2 接「列表行打印」时同一函数吃真实行数据，模板字段与数据天然对齐。 */
export function fieldsFromSample(data: Record<string, unknown>): Array<Record<string, unknown>> {
  const out: Array<Record<string, unknown>> = [];
  let order = 0;
  const push = (key: string, label: string, fieldType: string) => {
    out.push({ fieldKey: key, fieldLabel: label, fieldType, sortOrder: ++order });
  };
  const walk = (obj: Record<string, unknown>, prefix: string) => {
    Object.entries(obj || {}).forEach(([k, v]) => {
      const key = prefix ? `${prefix}.${k}` : k;
      if (Array.isArray(v)) {
        push(key, k, 'list');
        v.forEach((item) => {
          if (item && typeof item === 'object' && !Array.isArray(item)) {
            walk(item as Record<string, unknown>, `${key}[]`);
          }
        });
      } else if (v && typeof v === 'object') {
        walk(v as Record<string, unknown>, key);
      } else {
        push(key, k, typeof v === 'number' ? 'number' : 'text');
      }
    });
  };
  walk(data, '');
  return out;
}

/** 从模板 JSON 顶层提取冗余列值（paperSize/orientation），保存时与 payload 同步落库 */
export function extractRedundant(payload: string): { paperSize?: string; orientation?: string } {
  try {
    const tpl = JSON.parse(payload || '{}');
    return { paperSize: tpl.paperSize, orientation: tpl.orientation };
  } catch {
    return {};
  }
}

export type PrintTemplateMode = 'create' | 'edit' | 'copy';

export function templateTitle(t: (k: string) => string, mode: PrintTemplateMode): string {
  if (mode === 'create') return t('createTitle');
  if (mode === 'copy') return t('copyTitle');
  return t('editTitle');
}

export type { PrintTemplate };
