import type { Router } from "vue-router";

import { setupCommonGuard } from "./common.guard";
import { setupAccessGuard } from "./auth.guard";

/**
 * 项目守卫配置
 * @param router
 */
function createRouterGuard(router: Router) {
  /** 通用 */
  setupCommonGuard(router);
  /** 权限访问 */
  setupAccessGuard(router);
}

export { createRouterGuard };
