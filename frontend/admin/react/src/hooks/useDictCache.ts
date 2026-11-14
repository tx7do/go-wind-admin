/**
 * 字典缓存 Hook
 * 提供字典项的全局缓存、按 typeCode 检索、选项转换等功能
 * 对齐 Vue 版 use-dict-cache.ts
 */
import i18n from 'i18next';
import type {
  dictservicev1_DictEntry,
  dictservicev1_DictType,
} from '@/api/generated/admin/service/v1';
import { fetchListDictTypes, fetchListDictEntries } from '@/api/hooks/dict';
import { PaginationQuery } from '@go-wind/admin-react';

// ==============================
// 字典项缓存（模块级单例）
// ==============================

let dictEntryCache: Record<string, dictservicev1_DictEntry[]> = {};
let dictTypeCache: dictservicev1_DictType[] = [];
let cacheLoaded = false;

// ==============================
// 缓存管理
// ==============================

/**
 * 获取所有字典项并缓存
 * 如果缓存已存在则跳过
 */
export async function fetchAllDictEntries(): Promise<void> {
  if (cacheLoaded && Object.keys(dictEntryCache).length > 0) {
    return;
  }

  const [typesResult, entriesResult] = await Promise.all([
    fetchListDictTypes(new PaginationQuery({ paging: { page: 1, pageSize: 9999 } })),
    fetchListDictEntries(new PaginationQuery({ paging: { page: 1, pageSize: 99999 } })),
  ]);

  const types = typesResult?.items || [];
  const entries = entriesResult?.items || [];

  // 缓存类型列表
  dictTypeCache = types;

  // 按 typeCode 分组缓存条目
  const newCache: Record<string, dictservicev1_DictEntry[]> = {};
  for (const entry of entries) {
    const typeCode = types.find((type: any) => type.id === entry.typeId)?.typeCode;
    if (!typeCode) continue;

    if (!newCache[typeCode]) {
      newCache[typeCode] = [];
    }
    newCache[typeCode].push(entry);
  }

  dictEntryCache = newCache;
  cacheLoaded = true;
}

/**
 * 获取指定 typeCode 的字典项列表
 */
export function getDictEntriesByTypeCode(typeCode: string): dictservicev1_DictEntry[] {
  return dictEntryCache[typeCode] ?? [];
}

/**
 * 获取指定 typeCode 的字典项选项（label/value 格式）
 * label 会根据当前语言从 i18n 字段中提取
 */
export function getDictEntriesOptionsByTypeCode(typeCode: string): { label: string; value: string }[] {
  const entries = getDictEntriesByTypeCode(typeCode);
  return entries.map((entry) => ({
    label: getDictEntryLabel(entry),
    value: entry.entryValue ?? '',
  }));
}

/**
 * 重置缓存
 */
export function resetDictCache(): void {
  dictEntryCache = {};
  dictTypeCache = [];
  cacheLoaded = false;
}

// ==============================
// 工具函数
// ==============================

/**
 * 获取字典项标签（根据当前语言从 i18n 字段提取）
 */
export function getDictEntryLabel(row: dictservicev1_DictEntry): string {
  const currentLocale = i18n.language;
  const currentI18n = row.i18n?.[currentLocale];
  if (!currentI18n) return '';
  return currentI18n.entryLabel ?? '';
}

/**
 * 通过字典项值获取字典项标签
 */
export function getDictEntryLabelByValue(
  value?: string,
  dictEntries?: dictservicev1_DictEntry[],
): string {
  if (value === undefined) return '';
  if (!dictEntries) return value;

  const entry = dictEntries.find((e) => e.entryValue === value);
  if (!entry) return value;

  return getDictEntryLabel(entry) || value;
}

/**
 * 获取缓存的字典类型列表
 */
export function getDictTypes(): dictservicev1_DictType[] {
  return dictTypeCache;
}
