/**
 * 字典缓存 Composable
 * 提供字典项的全局缓存、按 typeCode 检索、选项转换等功能
 */
import { ref } from "vue";
import type { dictservicev1_DictEntry } from "@/api/generated/admin/service/v1";
import { i18n } from "@/i18n/setup";
import { fetchListDictTypes, fetchListDictEntries } from "./dict";
import { PaginationQuery } from "@/core/transport/rest";

// ==============================
// 字典项缓存（模块级单例）
// ==============================

const dictEntryCache = ref<Record<string, dictservicev1_DictEntry[]>>({});

// ==============================
// 缓存管理
// ==============================

/**
 * 获取所有字典项并缓存
 * 如果缓存已存在则跳过
 */
export async function fetchAllDictEntries() {
  if (dictEntryCache.value && Object.keys(dictEntryCache.value).length > 0) {
    return;
  }

  const types = await fetchListDictTypes(new PaginationQuery({ paging: { page: 1, pageSize: 9999 } }));
  const result = await fetchListDictEntries(new PaginationQuery({ paging: { page: 1, pageSize: 99999 } }));
  const items = result?.items || [];

  for (const item of items) {
    const typeCode = types?.items?.find((type: any) => type.id === item.typeId)?.typeCode;

    if (!typeCode) {
      continue;
    }
    if (dictEntryCache.value[typeCode]) {
      dictEntryCache.value[typeCode].push(item);
      continue;
    }
    dictEntryCache.value[typeCode] = [item];
  }
}

/**
 * 获取指定 typeCode 的字典项列表
 */
export function getDictEntriesByTypeCode(typeCode: string) {
  if (dictEntryCache.value[typeCode]) {
    return dictEntryCache.value[typeCode];
  }
  return [];
}

/**
 * 获取指定 typeCode 的字典项选项（label/value 格式）
 */
export function getDictEntriesOptionsByTypeCode(typeCode: string) {
  const options = getDictEntriesByTypeCode(typeCode);
  return options.map((option) => ({
    label: getDictEntryLabel(option),
    value: option.entryValue,
  }));
}

/**
 * 重置缓存
 */
export function resetCache() {
  dictEntryCache.value = {};
}

// ==============================
// 工具函数
// ==============================

/**
 * 获取字典项标签
 */
export function getDictEntryLabel(row: dictservicev1_DictEntry) {
  const currentLocale = i18n.global.locale.value;
  const currentI18n = row.i18n?.[currentLocale];
  if (currentI18n === undefined) {
    return "";
  }
  return currentI18n.entryLabel;
}

/**
 * 通过字典项值获取字典项标签
 */
export function getDictEntryLabelByValue(
  value?: string,
  dictEntries?: dictservicev1_DictEntry[]
): string {
  if (value === undefined) {
    return "";
  }
  if (dictEntries === undefined) {
    return value;
  }
  const dictEnt = dictEntries.find((entry) => entry.entryValue === value);
  if (!dictEnt) {
    return value;
  }
  return getDictEntryLabel(dictEnt) || value;
}
