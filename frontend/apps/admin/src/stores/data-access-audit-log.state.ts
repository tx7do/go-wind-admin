import { defineStore } from 'pinia';

import { createDataAccessAuditLogServiceClient } from '#/generated/api/admin/service/v1';
import { makeOrderBy, makeQueryString } from '#/utils/query';
import { type Paging, requestClientRequestHandler } from '#/utils/request';
import {useUserStore} from "@vben/stores";

export const useDataAccessAuditLogStore = defineStore(
  'data-access-audit-log',
  () => {
    const service = createDataAccessAuditLogServiceClient(
      requestClientRequestHandler,
    );

    const userStore = useUserStore();

    /**
     * 查询数据访问审计日志列表
     */
    async function listDataAccessAuditLog(
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
     * 查询数据访问审计日志日志
     */
    async function getDataAccessAuditLog(id: number) {
      return await service.Get({ id });
    }

    function $reset() {}

    return {
      $reset,
      listDataAccessAuditLog,
      getDataAccessAuditLog,
    };
  },
);
