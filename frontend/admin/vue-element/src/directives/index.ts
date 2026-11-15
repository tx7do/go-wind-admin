import type { App } from "vue";

import { registerAccessDirective } from "@go-wind/admin-element";

// 全局注册 directive
export function setupDirective(app: App<Element>) {
  // 注册权限指令（v-access）
  registerAccessDirective(app);
}
