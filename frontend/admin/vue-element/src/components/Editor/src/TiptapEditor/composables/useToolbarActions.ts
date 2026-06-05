import type { Level } from "@tiptap/extension-heading";
import type { Editor } from "@tiptap/vue-3";
import type { Ref } from "vue";

import { WarningFilled } from "@element-plus/icons-vue";
import { $t } from "@/core/i18n";
import { ElMessageBox } from "element-plus";

interface ModalActions {
  openLinkModal: () => void;
  openCodeBlockModal: () => void;
  openVideoModal: () => void;
  openIframeModal: () => void;
}

/**
 * 工具栏所有操作函数
 * - 接收 editor ref + modal 操作（避免循环依赖）
 * - 返回可直接在模板中使用的方法集合
 */
export function useToolbarActions(
  editor: Ref<Editor | undefined>,
  modals: ModalActions,
  fileInputRef: Ref<HTMLInputElement | undefined>,
  markdownInputRef: Ref<HTMLInputElement | undefined>
) {
  const toggleBold = () => editor.value?.chain().focus().toggleBold().run();
  const toggleItalic = () => editor.value?.chain().focus().toggleItalic().run();
  const toggleStrike = () => editor.value?.chain().focus().toggleStrike().run();
  const toggleUnderline = () => editor.value?.chain().focus().toggleUnderline().run();
  const toggleCode = () => editor.value?.chain().focus().toggleCode().run();
  const toggleHeading = (level: Level) =>
    editor.value?.chain().focus().toggleHeading({ level }).run();
  const toggleBulletList = () => editor.value?.chain().focus().toggleBulletList().run();
  const toggleOrderedList = () => editor.value?.chain().focus().toggleOrderedList().run();
  const toggleTaskList = () => editor.value?.chain().focus().toggleTaskList().run();
  const insertCodeBlock = () => modals.openCodeBlockModal();
  const toggleBlockquote = () => editor.value?.chain().focus().toggleBlockquote().run();
  const toggleSubscript = () => editor.value?.chain().focus().toggleSubscript().run();
  const toggleSuperscript = () => editor.value?.chain().focus().toggleSuperscript().run();
  const setParagraph = () => editor.value?.chain().focus().setParagraph().run();
  const clearFormatting = () => editor.value?.chain().focus().unsetAllMarks().clearNodes().run();
  const insertHorizontalRule = () => editor.value?.chain().focus().setHorizontalRule().run();
  const insertTable = () =>
    editor.value?.chain().focus().insertTable({ rows: 3, cols: 3, withHeaderRow: true }).run();
  const deleteTable = () => editor.value?.chain().focus().deleteTable().run();
  const addRowBefore = () => editor.value?.chain().focus().addRowBefore().run();
  const addRowAfter = () => editor.value?.chain().focus().addRowAfter().run();
  const deleteRow = () => editor.value?.chain().focus().deleteRow().run();
  const addColumnBefore = () => editor.value?.chain().focus().addColumnBefore().run();
  const addColumnAfter = () => editor.value?.chain().focus().addColumnAfter().run();
  const deleteColumn = () => editor.value?.chain().focus().deleteColumn().run();
  const mergeCells = () => editor.value?.chain().focus().mergeCells().run();
  const splitCell = () => editor.value?.chain().focus().splitCell().run();
  const toggleHeaderRow = () => editor.value?.chain().focus().toggleHeaderRow().run();
  const toggleHeaderColumn = () => editor.value?.chain().focus().toggleHeaderColumn().run();
  const toggleHeaderCell = () => editor.value?.chain().focus().toggleHeaderCell().run();
  const setAlign = (align: "center" | "justify" | "left" | "right") =>
    editor.value?.chain().focus().setTextAlign(align).run();
  const setTextColor = (color: string) => editor.value?.chain().focus().setColor(color).run();
  const setHighlight = (color: string) =>
    editor.value?.chain().focus().toggleHighlight({ color }).run();
  const uploadImage = () => fileInputRef.value?.click();
  const insertVideo = () => modals.openVideoModal();
  const insertIframe = () => modals.openIframeModal();
  const importMarkdown = () => markdownInputRef.value?.click();
  const undo = () => editor.value?.chain().focus().undo().run();
  const redo = () => editor.value?.chain().focus().redo().run();

  const clearContent = async () => {
    try {
      await ElMessageBox.confirm(
        $t("common.editor.clear_content_confirm"),
        $t("common.title.confirm"),
        {
          confirmButtonText: $t("common.confirm"),
          cancelButtonText: $t("common.cancel"),
          type: "warning",
          icon: WarningFilled,
        }
      );
      editor.value?.commands.setContent("");
    } catch {
      // 用户取消操作
    }
  };

  return {
    toggleBold,
    toggleItalic,
    toggleStrike,
    toggleUnderline,
    toggleCode,
    toggleHeading,
    toggleBulletList,
    toggleOrderedList,
    toggleTaskList,
    insertCodeBlock,
    toggleBlockquote,
    toggleSubscript,
    toggleSuperscript,
    setParagraph,
    clearFormatting,
    insertHorizontalRule,
    insertTable,
    deleteTable,
    addRowBefore,
    addRowAfter,
    deleteRow,
    addColumnBefore,
    addColumnAfter,
    deleteColumn,
    mergeCells,
    splitCell,
    toggleHeaderRow,
    toggleHeaderColumn,
    toggleHeaderCell,
    setAlign,
    setTextColor,
    setHighlight,
    uploadImage,
    insertVideo,
    insertIframe,
    importMarkdown,
    undo,
    redo,
    clearContent,
  };
}
