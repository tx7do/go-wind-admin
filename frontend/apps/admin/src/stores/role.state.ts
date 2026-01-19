import { useUserStore } from '@vben/stores';

import { defineStore } from 'pinia';

import { createRoleServiceClient } from '#/generated/api/admin/service/v1';
import { makeOrderBy, makeQueryString, makeUpdateMask } from '#/utils/query';
import { type Paging, requestClientRequestHandler } from '#/utils/request';

export const useRoleStore = defineStore('role', () => {
  const service = createRoleServiceClient(requestClientRequestHandler);
  const userStore = useUserStore();

  /**
   * 查询角色列表
   */
  async function listRole(
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
   * 获取角色
   */
  async function getRole(id: number) {
    return await service.Get({ id });
  }

  /**
   * 创建角色
   */
  async function createRole(values: object) {
    return await service.Create({
      // @ts-ignore proto generated code is error.
      data: {
        ...values,
      },
    });
  }

  /**
   * 更新角色
   */
  async function updateRole(id: number, values: object) {
    if ('id' in values) delete values.id;

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
   * 删除角色
   */
  async function deleteRole(id: number) {
    return await service.Delete({ id });
  }

  function $reset() {}

  return {
    $reset,
    listRole,
    getRole,
    createRole,
    updateRole,
    deleteRole,
  };
});
