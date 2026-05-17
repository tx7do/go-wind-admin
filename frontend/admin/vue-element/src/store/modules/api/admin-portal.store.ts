import { defineStore } from "pinia";

import { createAdminPortalServiceClient } from "@/api/generated/admin/service/v1";
import { requestClientRequestHandler } from "@/transport/rest";

export const useAdminPortalStore = defineStore("admin-portal", () => {
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
