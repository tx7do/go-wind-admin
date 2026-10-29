import { App } from "vue";

import { ElDialog, ElDrawer } from "element-plus";
import * as ElementPlusIcons from "@element-plus/icons-vue";

import { configureVxeTable } from "@/plugins/vxe-table";
import VXETable from "vxe-table";

import { InstallCodeMirror } from "codemirror-editor-vue3";

// ============================================
// 修复 Element Plus lockscreen 导致布局跳动
// ============================================
// 实际修复在 LeftLayout.vue：使用 100vw 替代 100% 计算 .layout__main 宽度，
// 使其不受 body.style.width 内联样式变化的影响。
// 以下 Dialog/Drawer 全局禁用 lockScroll 作为双重保障。

ElDialog.props.lockScroll!.default = false;
ElDrawer.props.lockScroll!.default = false;

export function registerGlobComp(app: App) {
  configureVxeTable();
  app.use(VXETable);
  app.use(InstallCodeMirror);

  // 全局组件（Element Plus 图标）
  Object.entries(ElementPlusIcons).forEach(([name, comp]) => app.component(name, comp));

}
