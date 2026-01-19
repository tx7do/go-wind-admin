import { useUserStore } from '@vben/stores';

import { defineStore } from 'pinia';

import { createApiAuditLogServiceClient } from '#/generated/api/admin/service/v1';
import { makeOrderBy, makeQueryString } from '#/utils/query';
import { type Paging, requestClientRequestHandler } from '#/utils/request';

export const useApiAuditLogStore = defineStore('api-audit-log', () => {
  const service = createApiAuditLogServiceClient(requestClientRequestHandler);
  const userStore = useUserStore();

  /**
   * 查询API日志列表
   */
  async function listApiAuditLog(
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
   * 查询API日志
   */
  async function getApiAuditLog(id: number) {
    return await service.Get({ id });
  }

  function $reset() {}

  return {
    $reset,
    listApiAuditLog,
    getApiAuditLog,
  };
});
