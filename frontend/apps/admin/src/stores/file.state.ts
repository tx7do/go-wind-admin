import { useUserStore } from '@vben/stores';

import { defineStore } from 'pinia';

import { createFileServiceClient } from '#/generated/api/admin/service/v1';
import { makeOrderBy, makeQueryString, makeUpdateMask } from '#/utils/query';
import { type Paging, requestClientRequestHandler } from '#/utils/request';

export const useFileStore = defineStore('file', () => {
  const service = createFileServiceClient(requestClientRequestHandler);
  const userStore = useUserStore();

  /**
   * 查询文件列表
   */
  async function listFile(
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
   * 获取文件
   */
  async function getFile(id: number) {
    return await service.Get({ id });
  }

  /**
   * 创建文件
   */
  async function createFile(values: object) {
    return await service.Create({
      data: {
        ...values,
      },
    });
  }

  /**
   * 更新文件
   */
  async function updateFile(id: number, values: object) {
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
   * 删除文件
   */
  async function deleteFile(id: number) {
    return await service.Delete({ id });
  }

  function $reset() {}

  return {
    $reset,
    listFile,
    getFile,
    createFile,
    updateFile,
    deleteFile,
  };
});
