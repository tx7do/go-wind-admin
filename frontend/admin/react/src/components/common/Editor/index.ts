/**
 * Editor 薄包装：re-export @go-wind/editor-react 并注入应用 env。
 * 页面 import 路径不变（@/components/common/Editor），内部已切库消费。
 */
export { Editor, EditorType } from "@go-wind/editor-react";
export {
  EditorProvider,
  setEditorLocale,
  isDarkMode,
  defaultUploadImage,
} from "@go-wind/editor-react";
export type { EditorEnv } from "@go-wind/editor-react";
import "@go-wind/editor-react/style.css";
