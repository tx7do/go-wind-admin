import { defineStore } from 'pinia';

import { createPermissionGroupServiceClient } from '#/generated/api/admin/service/v1';
import { makeQueryString, makeUpdateMask } from '#/utils/query';
import { type Paging, requestClientRequestHandler } from '#/utils/request';

export const usePermissionGroupStore = defineStore('permission-group', () => {
  const service = createPermissionGroupServiceClient(
    requestClientRequestHandler,
  );

  /**
   * 查询权限点分组列表
   */
  async function listPermissionGroup(
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
      orderBy: orderBy ?? [],
      query: makeQueryString(formValues ?? null),
      page: paging?.page,
      pageSize: paging?.pageSize,
      noPaging,
    });
  }

  /**
   * 获取权限点分组
   */
  async function getPermissionGroup(id: number) {
    return await service.Get({ id });
  }

  /**
   * 创建权限点分组
   */
  async function createPermissionGroup(values: object) {
    return await service.Create({
      // @ts-ignore proto generated code is error.
      data: {
        ...values,
      },
    });
  }

  /**
   * 更新权限点分组
   */
  async function updatePermissionGroup(id: number, values: object) {
    return await service.Update({
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
   * 删除权限点分组
   */
  async function deletePermissionGroup(id: number) {
    return await service.Delete({ id });
  }

  function $reset() {}

  return {
    $reset,
    listPermissionGroup,
    getPermissionGroup,
    createPermissionGroup,
    updatePermissionGroup,
    deletePermissionGroup,
  };
});
