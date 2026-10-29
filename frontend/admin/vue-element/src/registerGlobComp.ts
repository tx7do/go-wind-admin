import { App } from "vue";

import { ElButton, ElDialog, ElDrawer } from "element-plus";
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

  // ElButton 全局注册：@go-wind/editor-vue@0.2.3 的 Tiptap 工具栏模板用了 <el-button> 但未 import，
  // 产物里是 resolveComponent("el-button")，而本应用是 ElementPlusResolver 按需导入（不全局 use），
  // 因此在消息抽屉等用到富文本的地方会报 "Failed to resolve component: el-button"。
  // 库里已改为显式 import 并重新构建，等 0.2.4 发布后可以删掉这行。
  app.component("ElButton", ElButton);
}
