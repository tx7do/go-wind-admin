import { useUserStore } from '@vben/stores';

import { defineStore } from 'pinia';

import { createLanguageServiceClient } from '#/generated/api/admin/service/v1';
import { makeOrderBy, makeQueryString, makeUpdateMask } from '#/utils/query';
import { type Paging, requestClientRequestHandler } from '#/utils/request';

export const useLanguageStore = defineStore('language', () => {
  const service = createLanguageServiceClient(requestClientRequestHandler);

  const userStore = useUserStore();

  /**
   * 查询语言列表
   */
  async function listLanguage(
    paging?: Paging,
    formValues?: null | object,
    fieldMask?: null | string,
    orderBy?: null | string[],
  ) {
    const noPaging =
      paging?.page === undefined && paging?.pageSize === undefined;
    return await service.List({
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
   * 获取语言
   */
  async function getLanguage(id: number) {
    return await service.Get({ id });
  }

  /**
   * 创建语言
   */
  async function createLanguage(values: object) {
    return await service.Create({
      data: {
        ...values,
      },
    });
  }

  /**
   * 更新语言
   */
  async function updateLanguage(id: number, values: object) {
    return await service.Update({
      id,
      data: {
        ...values,
      },
      // @ts-ignore proto generated code is error.
      updateMask: makeUpdateMask(Object.keys(values ?? [])),
    });
  }

  /**
   * 删除语言
   */
  async function deleteLanguage(id: number) {
    return await service.Delete({ id });
  }

  function $reset() {}

  return {
    $reset,
    listLanguage,
    getLanguage,
    createLanguage,
    updateLanguage,
    deleteLanguage,
  };
});
