import { useUserStore } from '@vben/stores';

import { defineStore } from 'pinia';

import { createLoginAuditLogServiceClient } from '#/generated/api/admin/service/v1';
import { makeOrderBy, makeQueryString } from '#/utils/query';
import { type Paging, requestClientRequestHandler } from '#/utils/request';

export const useOperationAuditLogStore = defineStore(
  'operation-audit-log',
  () => {
    const service = createLoginAuditLogServiceClient(
      requestClientRequestHandler,
    );

    const userStore = useUserStore();

    /**
     * 查询操作审计日志列表
     */
    async function listOperationAuditLog(
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
     * 查询操作审计日志日志
     */
    async function getOperationAuditLog(id: number) {
      return await service.Get({ id });
    }

    function $reset() {}

    return {
      $reset,
      listOperationAuditLog,
      getOperationAuditLog,
    };
  },
);
