import { defineStore } from 'pinia';

import { createAdminOperationLogServiceClient } from '#/generated/api/admin/service/v1';
import { makeQueryString } from '#/utils/query';
import { type Paging, requestClientRequestHandler } from '#/utils/request';

export const useAdminOperationLogStore = defineStore(
  'admin-operation-log',
  () => {
    const service = createAdminOperationLogServiceClient(
      requestClientRequestHandler,
    );

    /**
     * 查询操作日志列表
     */
    async function listAdminOperationLog(
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
     * 查询操作日志
     */
    async function getAdminOperationLog(id: number) {
      return await service.Get({ id });
    }

    function $reset() {}

    return {
      $reset,
      listAdminOperationLog,
      getAdminOperationLog,
    };
  },
);
