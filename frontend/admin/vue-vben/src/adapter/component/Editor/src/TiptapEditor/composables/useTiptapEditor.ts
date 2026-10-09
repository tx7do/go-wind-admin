import type { Ref } from 'vue';
import { computed, ref, watch } from 'vue';

import { preferences } from '@vben/preferences';

import Color from '@tiptap/extension-color';
import Highlight from '@tiptap/extension-highlight';
import HorizontalRule from '@tiptap/extension-horizontal-rule';
import Image from '@tiptap/extension-image';
import Link from '@tiptap/extension-link';
import Placeholder from '@tiptap/extension-placeholder';
import Subscript from '@tiptap/extension-subscript';
import Superscript from '@tiptap/extension-superscript';
import { Table } from '@tiptap/extension-table';
import TableCell from '@tiptap/extension-table-cell';
import TableHeader from '@tiptap/extension-table-header';
import TableRow from '@tiptap/extension-table-row';
import TaskItem from '@tiptap/extension-task-item';
import TaskList from '@tiptap/extension-task-list';
import TextAlign from '@tiptap/extension-text-align';
import { TextStyle } from '@tiptap/extension-text-style';
import Underline from '@tiptap/extension-underline';
import StarterKit from '@tiptap/starter-kit';
import { useEditor } from '@tiptap/vue-3';
import { all, createLowlight } from 'lowlight';

import { createCustomCodeBlock } from '../extensions/CustomCodeBlock';
import { CustomIframe } from '../extensions/CustomIframe';
import { CustomVideo } from '../extensions/CustomVideo';

interface UseTiptapEditorOptions {
  modelValue: Ref<string>;
  disabled: Ref<boolean>;
  placeholder: Ref<string>;
  showToolbar: Ref<boolean>;
  showStatusBar: Ref<boolean>;
  uploadImage: Ref<((file: File) => Promise<string>) | undefined>;
  fullHeight: Ref<boolean>;
  height: Ref<number | string>;
  config: Ref<Record<string, any>>;
  emit: {
    (e: 'change', value: string): void;
    (e: 'imageUpload', file: File): void;
    (e: 'ready', editor: any): void;
    // eslint-disable-next-line @typescript-eslint/unified-signatures
    (e: 'update:modelValue', value: string): void;
  };
}

export function useTiptapEditor(options: UseTiptapEditorOptions) {
  const { modelValue, disabled, placeholder, emit } = options;

  // 内部内容追踪，防止循环更新
  const contentRef = ref(modelValue.value);
  let isInternalUpdate = false;

  // 创建 lowlight 实例
  const lowlight = createLowlight(all);

  // 初始化编辑器
  const editor = useEditor({
    content: modelValue.value,
    extensions: [
      StarterKit.configure({
        link: false,
        underline: false,
        horizontalRule: false,
        codeBlock: false,
      }),
      Underline,
      Subscript,
      Superscript,
      TaskList,
      TaskItem.configure({ nested: true }),
      HorizontalRule,
      Highlight.configure({ multicolor: true }),
      Color,
      TextStyle,
      createCustomCodeBlock(lowlight),
      Table.configure({ resizable: true }),
      TableRow,
      TableHeader,
      TableCell,
      Placeholder.configure({ placeholder: placeholder.value }),
      Link.configure({ openOnClick: false, autolink: true }),
      Image.configure({ inline: true }),
      CustomVideo,
      CustomIframe,
      TextAlign.configure({ types: ['heading', 'paragraph'] }),
    ],
    editable: !disabled.value,
    autofocus: 'end',
    editorProps: {
      attributes: {
        class: 'prose dark:prose-invert focus:outline-none min-h-full',
      },
    },
    onCreate: ({ editor }) => {
      emit('ready', editor);
    },
    onUpdate: ({ editor }) => {
      if (isInternalUpdate) {
        isInternalUpdate = false;
        return;
      }
      const html = editor.getHTML();
      contentRef.value = html;
      emit('update:modelValue', html);
      emit('change', html);
    },
  });

  // 监听外部值变化
  watch(modelValue, (newVal) => {
    if (editor.value && newVal !== contentRef.value) {
      isInternalUpdate = true;
      editor.value.commands.setContent(newVal);
      contentRef.value = newVal;
    }
  });

  // 监听禁用状态
  watch(disabled, (newVal) => {
    editor.value?.setEditable(!newVal);
  });

  // 监听 placeholder 变化
  watch(placeholder, (newVal) => {
    const placeholderExt = editor.value?.extensionManager?.extensions.find(
      (e: any) => e.name === 'placeholder',
    );
    if (placeholderExt?.options) {
      placeholderExt.options.placeholder = newVal;
    }
  });

  // 暗色模式
  const isDark = ref(preferences.theme.mode === 'dark');

  watch(
    () => preferences.theme.mode,
    (newMode) => {
      isDark.value = newMode === 'dark';
      if (editor.value?.view?.dom) {
        editor.value.view.dom.classList.toggle('dark', isDark.value);
        void editor.value.view.dom.offsetWidth;
      }
    },
    { immediate: true },
  );

  // 文本颜色：根据主题模式动态推导
  const textColor = computed(() => (isDark.value ? '#f1f5f9' : '#1e293b'));
  const highlightColor = ref('#FFFF00');

  return {
    editor,
    isDark,
    textColor,
    highlightColor,
    contentRef,
  };
}
