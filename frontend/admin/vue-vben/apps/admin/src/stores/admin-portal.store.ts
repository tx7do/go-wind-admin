import { defineStore } from 'pinia';

import { createAdminPortalServiceClient } from '#/generated/api/admin/service/v1';
import { requestClientRequestHandler } from '#/utils/request';

export const useAdminPortalStore = defineStore('admin-portal', () => {
  const service = createAdminPortalServiceClient(requestClientRequestHandler);

  /**
   * 查询路由列表
   */
  async function listRouter() {
    return await service.GetNavigation({});
  }

  function $reset() {}

  return {
    $reset,
    listRouter,
  };
});
