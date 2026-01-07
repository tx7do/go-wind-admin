import { $t } from '@vben/locales';

import { defineStore } from 'pinia';

import { createAdminLoginLogServiceClient } from '#/generated/api/admin/service/v1';
import { makeQueryString } from '#/utils/query';
import { type Paging, requestClientRequestHandler } from '#/utils/request';

export const useAdminLoginLogStore = defineStore('admin-login-log', () => {
  const service = createAdminLoginLogServiceClient(requestClientRequestHandler);

  /**
   * 查询登录日志列表
   */
  async function listAdminLoginLog(
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
   * 查询登录日志
   */
  async function getAdminLoginLog(id: number) {
    return await service.Get({ id });
  }

  function $reset() {}

  return {
    $reset,
    listAdminLoginLog,
    getAdminLoginLog,
  };
});

/**
 * 成功失败的颜色
 * @param success
 */
export function successToColor(success: boolean) {
  // 成功用柔和的绿色，失败用柔和的红色，兼顾视觉舒适度与直观性
  return success ? 'limegreen' : 'crimson';
}

export function successToName(success: boolean, statusCode: number) {
  return success
    ? $t('enum.successStatus.success')
    : ` ${$t('enum.successStatus.failed')} (${statusCode})`;
}
