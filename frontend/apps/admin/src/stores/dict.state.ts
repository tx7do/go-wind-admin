import { useUserStore } from '@vben/stores';

import { defineStore } from 'pinia';

import { createDictServiceClient } from '#/generated/api/admin/service/v1';
import { makeOrderBy, makeQueryString, makeUpdateMask } from '#/utils/query';
import { type Paging, requestClientRequestHandler } from '#/utils/request';

export const useDictStore = defineStore('dict', () => {
  const service = createDictServiceClient(requestClientRequestHandler);
  const userStore = useUserStore();

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
    return await service.ListDictType({
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
   * 查询字典条目列表
   */
  async function listDictEntry(
    paging?: Paging,
    formValues?: null | object,
    fieldMask?: null | string,
    orderBy?: null | string[],
  ) {
    const noPaging =
      paging?.page === undefined && paging?.pageSize === undefined;
    return await service.ListDictEntry({
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
    return await service.GetDictType({
      id,
    });
  }

  /**
   * 获取字典类型
   */
  async function getDictTypeByCode(code: string) {
    return await service.GetDictType({
      code,
    });
  }

  /**
   * 创建字典类型
   */
  async function createDictType(values: object) {
    return await service.CreateDictType({
      data: {
        ...values,
      },
    });
  }

  /**
   * 创建字典条目
   */
  async function createDictEntry(values: object) {
    return await service.CreateDictEntry({
      data: {
        ...values,
      },
    });
  }

  /**
   * 更新字典类型
   */
  async function updateDictType(id: number, values: object) {
    return await service.UpdateDictType({
      id,
      data: {
        ...values,
      },
      // @ts-ignore proto generated code is error.
      updateMask: makeUpdateMask(Object.keys(values ?? [])),
    });
  }

  /**
   * 更新字典条目
   */
  async function updateDictEntry(id: number, values: object) {
    return await service.UpdateDictEntry({
      id,
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
    return await service.DeleteDictType({ ids });
  }

  /**
   * 删除字典条目
   */
  async function deleteDictEntry(ids: number[]) {
    return await service.DeleteDictEntry({ ids });
  }

  function $reset() {}

  return {
    $reset,
    listDictType,
    listDictEntry,
    getDictType,
    getDictTypeByCode,
    createDictType,
    createDictEntry,
    updateDictType,
    updateDictEntry,
    deleteDictType,
    deleteDictEntry,
  };
});
