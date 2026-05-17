import { App } from "vue";

import * as ElementPlusIcons from "@element-plus/icons-vue";

import { configureVxeTable } from "@/plugins/vxe-table";
import VXETable from "vxe-table";

import { InstallCodeMirror } from "codemirror-editor-vue3";

export function registerGlobComp(app: App) {
  // 第三方插件
  configureVxeTable();
  app.use(VXETable);
  app.use(InstallCodeMirror);

  // 全局组件（Element Plus 图标）
  Object.entries(ElementPlusIcons).forEach(([name, comp]) => app.component(name, comp));
}
