import { ref } from 'vue';

import { i18n } from '@vben/locales';
import { useUserStore } from '@vben/stores';

import { defineStore } from 'pinia';

import {
  createDictEntryServiceClient,
  createDictTypeServiceClient,
  type dictservicev1_DictEntry,
} from '#/generated/api/admin/service/v1';
import { makeOrderBy, makeQueryString, makeUpdateMask } from '#/utils/query';
import { type Paging, requestClientRequestHandler } from '#/utils/request';

export const useDictStore = defineStore('dict', () => {
  const dictTypeService = createDictTypeServiceClient(
    requestClientRequestHandler,
  );
  const dictEntryService = createDictEntryServiceClient(
    requestClientRequestHandler,
  );
  const userStore = useUserStore();
  const dictEntryCache = ref<Record<string, dictservicev1_DictEntry[]>>({});

  /**
   * 查询字典类型列表
   */
  async function listDictType(
    paging?: Paging,
    formValues?: null | object,
    fieldMask?: null | string,
    orderBy?: null | string[],
  ) {
    const noPaging =
      paging?.page === undefined && paging?.pageSize === undefined;
    return await dictTypeService.List({
      // @ts-ignore proto generated code is error.
      fieldMask,
      orderBy: makeOrderBy(orderBy),
      query: makeQueryString(formValues, userStore.isTenantUser()),
      page: paging?.page,
      pageSize: paging?.pageSize,
      noPaging,
    });
  }

  /**
   * 查询字典项列表
   */
  async function listDictEntry(
    paging?: Paging,
    formValues?: null | object,
    fieldMask?: null | string,
    orderBy?: null | string[],
  ) {
    const noPaging =
      paging?.page === undefined && paging?.pageSize === undefined;
    return await dictEntryService.List({
      // @ts-ignore proto generated code is error.
      fieldMask,
      orderBy: makeOrderBy(orderBy),
      query: makeQueryString(formValues, userStore.isTenantUser()),
      page: paging?.page,
      pageSize: paging?.pageSize,
      noPaging,
    });
  }

  /**
   * 获取字典类型
   */
  async function getDictType(id: number) {
    return await dictTypeService.Get({
      id,
    });
  }

  /**
   * 获取字典类型
   */
  async function getDictTypeByCode(code: string) {
    return await dictTypeService.Get({
      code,
    });
  }

  /**
   * 创建字典类型
   */
  async function createDictType(values: Record<string, any> = {}) {
    return await dictTypeService.Create({
      // @ts-ignore proto generated code is error.
      data: {
        ...values,
      },
    });
  }

  /**
   * 创建字典项
   */
  async function createDictEntry(values: Record<string, any> = {}) {
    return await dictEntryService.Create({
      // @ts-ignore proto generated code is error.
      data: {
        ...values,
      },
    });
  }

  /**
   * 更新字典类型
   */
  async function updateDictType(id: number, values: Record<string, any> = {}) {
    return await dictTypeService.Update({
      id,
      // @ts-ignore proto generated code is error.
      data: {
        ...values,
      },
      // @ts-ignore proto generated code is error.
      updateMask: makeUpdateMask(Object.keys(values ?? [])),
    });
  }

  /**
   * 更新字典项
   */
  async function updateDictEntry(id: number, values: Record<string, any> = {}) {
    return await dictEntryService.Update({
      id,
      // @ts-ignore proto generated code is error.
      data: {
        ...values,
      },
      // @ts-ignore proto generated code is error.
      updateMask: makeUpdateMask(Object.keys(values ?? [])),
    });
  }

  /**
   * 删除字典类型
   */
  async function deleteDictType(ids: number[]) {
    return await dictTypeService.Delete({ ids });
  }

  /**
   * 删除字典项
   */
  async function deleteDictEntry(ids: number[]) {
    return await dictEntryService.Delete({ ids });
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

    const types = await listDictType();

    const result = await listDictEntry();
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

  function $reset() {}

  return {
    $reset,

    listDictType,
    getDictType,
    getDictTypeByCode,
    createDictType,
    updateDictType,
    deleteDictType,

    listDictEntry,
    createDictEntry,
    updateDictEntry,
    deleteDictEntry,
    dictEntryCache,
    getDictEntriesByTypeCode,
    fetchAllDictEntries,
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
