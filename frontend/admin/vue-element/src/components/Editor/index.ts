/**
 * Editor 薄包装：re-export @go-wind/editor-vue 并注入应用 env。
 * 页面 import 路径不变（@/components/Editor），内部已切库消费。
 */
export { Editor, EditorType } from "@go-wind/editor-vue";
export {
  provideEditorEnv,
  useEditorDark,
  setEditorLocale,
  isDarkNow as isDarkMode,
} from "@go-wind/editor-vue";
export type { EditorEnv } from "@go-wind/editor-vue";
import "@go-wind/editor-vue/style.css";
