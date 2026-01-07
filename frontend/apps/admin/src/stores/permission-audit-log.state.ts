import { defineStore } from 'pinia';

import { createAdminLoginLogServiceClient } from '#/generated/api/admin/service/v1';
import { makeQueryString } from '#/utils/query';
import { type Paging, requestClientRequestHandler } from '#/utils/request';

export const usePermissionAuditLogStore = defineStore(
  'permission-audit-log',
  () => {
    const service = createAdminLoginLogServiceClient(
      requestClientRequestHandler,
    );

    /**
     * 查询权限变更审计日志列表
     */
    async function listPermissionAuditLog(
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
     * 查询权限变更审计日志日志
     */
    async function getPermissionAuditLog(id: number) {
      return await service.Get({ id });
    }

    function $reset() {}

    return {
      $reset,
      listPermissionAuditLog,
      getPermissionAuditLog,
    };
  },
);
