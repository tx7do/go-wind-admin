import { ref } from 'vue';

import { i18n } from '@vben/locales';

import { defineStore } from 'pinia';

import {
  fetchListDictEntries,
  fetchListDictTypes,
  PaginationQuery,
} from '#/api';
import { type dictservicev1_DictEntry } from '#/api';

export const useDictStore = defineStore('dict', () => {
  const dictEntryCache = ref<Record<string, dictservicev1_DictEntry[]>>({});

  /**
   * 获取所有字典项
   */
  async function fetchAllDictEntries() {
    if (
      dictEntryCache &&
      dictEntryCache.value &&
      Object.keys(dictEntryCache.value).length > 0
    ) {
      return;
    }

    const types = await fetchListDictTypes(new PaginationQuery({}));

    const result = await fetchListDictEntries(new PaginationQuery({}));
    const items = result?.items || [];
    for (const item of items) {
      const typeCode = types?.items?.find(
        (type) => type.id === item.typeId,
      )?.typeCode;

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
   * 获取并缓存指定 typeCode 的字典项列表
   */
  function getDictEntriesByTypeCode(typeCode: string) {
    if (dictEntryCache.value[typeCode]) {
      return dictEntryCache.value[typeCode];
    }
    return [];
  }

  function getDictEntriesOptionsByTypeCode(typeCode: string) {
    const options = getDictEntriesByTypeCode(typeCode);
    return options.map((option) => ({
      label: getDictEntryLabel(option),
      value: option.entryValue,
    }));
  }

  function $reset() {
    // 登出/重认证时 resetAllStores 会遍历调各 store 的 $reset()。
    // 此前为空实现导致 dictEntryCache 残留，配合 fetchAllDictEntries 的
    // 「缓存非空即早返回」逻辑，会使用户 A 登出后用户 B（SPA 不刷新）
    // 直接命中 A 的字典缓存，造成跨用户/跨租户字典数据泄漏。
    dictEntryCache.value = {};
  }

  return {
    $reset,

    dictEntryCache,
    fetchAllDictEntries,
    getDictEntriesByTypeCode,
    getDictEntriesOptionsByTypeCode,
  };
});

/**
 * 获取字典项标签
 */
export function getDictEntryLabel(row: dictservicev1_DictEntry) {
  const currentI18n = row.i18n?.[i18n.global.locale.value];
  if (currentI18n === undefined) {
    return '';
  }
  return currentI18n.entryLabel;
}

/**
 * 通过字典项值获取字典项标签
 * @param value
 * @param dictEntries
 */
export function getDictEntryLabelByValue(
  value?: string,
  dictEntries?: dictservicev1_DictEntry[],
): string {
  if (value === undefined) {
    return '';
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
