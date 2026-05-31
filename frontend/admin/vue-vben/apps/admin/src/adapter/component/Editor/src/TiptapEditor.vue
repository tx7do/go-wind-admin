<script setup lang="ts">
import type { Level } from '@tiptap/extension-heading';

import { computed, h, nextTick, onUnmounted, ref, watch } from 'vue';

import { LucideCircleAlert } from '@vben/icons';
import { $t } from '@vben/locales';
import { preferences } from '@vben/preferences';

import { mergeAttributes, Node } from '@tiptap/core';
import CodeBlockLowlight from '@tiptap/extension-code-block-lowlight';
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
import { EditorContent, useEditor } from '@tiptap/vue-3';
import { Input, Modal } from 'ant-design-vue';
import { all, createLowlight } from 'lowlight';
import { marked } from 'marked';

interface Props {
  modelValue: string;
  height?: number | string;
  disabled?: boolean;
  placeholder?: string;
  config?: Record<string, any>;
  showToolbar?: boolean;
  showStatusBar?: boolean;
  uploadImage?: (file: File) => Promise<string>;
  fullHeight?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  height: 500,
  disabled: false,
  placeholder: $t('ui.editor.please_input_content'),
  config: () => ({}),
  showToolbar: true,
  showStatusBar: true,
  uploadImage: undefined,
  fullHeight: true,
});

const emit = defineEmits<{
  (e: 'change', value: string): void;
  (e: 'imageUpload', file: File): void;
  (e: 'ready', editor: any): void;
  (e: 'update:modelValue', value: string): void;
}>();

// 创建 lowlight 实例（all 已包含 200+ 种语言，无需额外注册）
const lowlight = createLowlight(all);

// 扩展 CodeBlockLowlight，添加 data-language 属性以便 CSS 和 JS 识别
const CustomCodeBlockLowlight = CodeBlockLowlight.extend({
  addAttributes() {
    return {
      ...this.parent?.(),
      language: {
        default: 'javascript',
        parseHTML: (element) => element.dataset.language,
        renderHTML: (attributes) => {
          return {
            'data-language': attributes.language || 'javascript',
            class: `language-${attributes.language || 'javascript'}`,
          };
        },
      },
    };
  },
  renderHTML({ HTMLAttributes }) {
    return [
      'pre',
      mergeAttributes(this.options.HTMLAttributes, HTMLAttributes),
      ['code', {}, 0],
    ];
  },
  addNodeView() {
    return ({ node, getPos, editor: editorInstance }) => {
      const dom = document.createElement('pre');
      const contentDOM = document.createElement('code');

      // 设置属性
      dom.dataset.language = node.attrs.language || 'javascript';
      dom.className = `language-${node.attrs.language || 'javascript'}`;

      // 创建语言选择器容器
      const selectorWrapper = document.createElement('div');
      selectorWrapper.className = 'code-block-language-selector';
      selectorWrapper.contentEditable = 'false';

      // 创建 select 元素
      const select = document.createElement('select');
      select.contentEditable = 'false';

      // 添加语言选项
      const languageOptions = [
        { value: 'javascript', label: 'JavaScript' },
        { value: 'typescript', label: 'TypeScript' },
        { value: 'python', label: 'Python' },
        { value: 'java', label: 'Java' },
        { value: 'cpp', label: 'C++' },
        { value: 'c', label: 'C' },
        { value: 'csharp', label: 'C#' },
        { value: 'go', label: 'Go' },
        { value: 'rust', label: 'Rust' },
        { value: 'php', label: 'PHP' },
        { value: 'ruby', label: 'Ruby' },
        { value: 'swift', label: 'Swift' },
        { value: 'kotlin', label: 'Kotlin' },
        { value: 'html', label: 'HTML' },
        { value: 'css', label: 'CSS' },
        { value: 'scss', label: 'SCSS' },
        { value: 'json', label: 'JSON' },
        { value: 'yaml', label: 'YAML' },
        { value: 'sql', label: 'SQL' },
        { value: 'bash', label: 'Bash' },
        { value: 'shell', label: 'Shell' },
        { value: 'markdown', label: 'Markdown' },
        { value: 'plaintext', label: 'Plain Text' },
      ];

      languageOptions.forEach((lang) => {
        const option = document.createElement('option');
        option.value = lang.value;
        option.textContent = lang.label;
        if (lang.value === node.attrs.language) {
          option.selected = true;
        }
        select.append(option);
      });

      // 监听语言变化
      select.addEventListener('change', (e) => {
        const newLanguage = (e.target as HTMLSelectElement).value;
        if (typeof getPos === 'function') {
          const pos = getPos();
          if (typeof pos === 'number') {
            editorInstance.view.dispatch(
              editorInstance.view.state.tr.setNodeMarkup(pos, undefined, {
                ...node.attrs,
                language: newLanguage,
              }),
            );

            // 更新 DOM 属性
            dom.dataset.language = newLanguage;
            dom.className = `language-${newLanguage}`;
          }
        }
      });

      selectorWrapper.append(select);
      dom.append(selectorWrapper);
      dom.append(contentDOM);

      return {
        dom,
        contentDOM,
        update: (updatedNode) => {
          if (updatedNode.type !== node.type) {
            return false;
          }

          // 更新选择器的值
          if (updatedNode.attrs.language !== node.attrs.language) {
            select.value = updatedNode.attrs.language;
            dom.dataset.language = updatedNode.attrs.language;
            dom.className = `language-${updatedNode.attrs.language}`;
          }

          return true;
        },
      };
    };
  },
}).configure({
  lowlight,
  defaultLanguage: 'javascript',
});

// 自定义视频扩展
const CustomVideo = Node.create({
  name: 'video',
  group: 'block',
  atom: true,
  draggable: true,

  addAttributes() {
    return {
      src: {
        default: null,
        parseHTML: (element) => element.getAttribute('src'),
        renderHTML: (attributes) => {
          if (!attributes.src) {
            return {};
          }
          return { src: attributes.src };
        },
      },
      width: {
        default: '100%',
        parseHTML: (element) => element.style.width || '100%',
        renderHTML: (attributes) => {
          return { style: `width: ${attributes.width}` };
        },
      },
      height: {
        default: 'auto',
        parseHTML: (element) => element.style.height || 'auto',
        renderHTML: (attributes) => {
          return { style: `height: ${attributes.height}` };
        },
      },
    };
  },

  parseHTML() {
    return [
      {
        tag: 'video',
      },
    ];
  },

  renderHTML({ HTMLAttributes }) {
    return [
      'video',
      mergeAttributes(HTMLAttributes, {
        controls: 'controls',
        style: `max-width: 100%; height: auto;`,
      }),
    ];
  },

  addCommands() {
    return {
      setVideo:
        (options: { height?: string; src: string; width?: string }) =>
        ({ commands }: { commands: any }) => {
          return commands.insertContent({
            type: this.name,
            attrs: options,
          });
        },
    } as any;
  },
});

// 自定义 Iframe 扩展
const CustomIframe = Node.create({
  name: 'iframe',
  group: 'block',
  atom: true,
  draggable: true,

  addAttributes() {
    return {
      src: {
        default: null,
        parseHTML: (element) => element.getAttribute('src'),
        renderHTML: (attributes) => {
          if (!attributes.src) {
            return {};
          }
          return { src: attributes.src };
        },
      },
      width: {
        default: '100%',
        parseHTML: (element) => element.style.width || '100%',
        renderHTML: (attributes) => {
          return { style: `width: ${attributes.width}` };
        },
      },
      height: {
        default: '500px',
        parseHTML: (element) => element.style.height || '500px',
        renderHTML: (attributes) => {
          return { style: `height: ${attributes.height}` };
        },
      },
      title: {
        default: null,
        parseHTML: (element) => element.getAttribute('title'),
        renderHTML: (attributes) => {
          if (!attributes.title) {
            return {};
          }
          return { title: attributes.title };
        },
      },
      allowfullscreen: {
        default: true,
        parseHTML: (element) => element.hasAttribute('allowfullscreen'),
        renderHTML: (attributes) => {
          if (!attributes.allowfullscreen) {
            return {};
          }
          return { allowfullscreen: 'allowfullscreen' };
        },
      },
    };
  },

  parseHTML() {
    return [
      {
        tag: 'iframe',
      },
    ];
  },

  renderHTML({ HTMLAttributes }) {
    return [
      'iframe',
      mergeAttributes(HTMLAttributes, {
        frameborder: '0',
        style: `max-width: 100%; border: 1px solid #e5e7eb; border-radius: 6px; margin: 12px 0;`,
      }),
    ];
  },

  addCommands() {
    return {
      setIframe:
        (options: {
          allowfullscreen?: boolean;
          height?: string;
          src: string;
          title?: string;
          width?: string;
        }) =>
        ({ commands }: { commands: any }) => {
          return commands.insertContent({
            type: this.name,
            attrs: options,
          });
        },
    } as any;
  },
});

// 响应式数据
const isDark = ref(preferences.theme.mode === 'dark');
const contentRef = ref(props.modelValue);
let isInternalUpdate = false;
const fileInputRef = ref<HTMLInputElement>();
const markdownInputRef = ref<HTMLInputElement>();

// Modal 状态
const linkModalVisible = ref(false);
const linkUrl = ref('');

const codeBlockModalVisible = ref(false);
const codeBlockLanguage = ref('javascript');
const codeBlockContent = ref('');

const videoModalVisible = ref(false);
const videoUrl = ref('');
const videoWidth = ref('100%');

const iframeModalVisible = ref(false);
const iframeUrl = ref('');
const iframeWidth = ref('100%');
const iframeHeight = ref('500px');
const iframeTitle = ref('');
const iframeAllowFullscreen = ref(true);

// 常用编程语言列表
const languages = [
  { value: 'javascript', label: 'JavaScript' },
  { value: 'typescript', label: 'TypeScript' },
  { value: 'python', label: 'Python' },
  { value: 'java', label: 'Java' },
  { value: 'cpp', label: 'C++' },
  { value: 'c', label: 'C' },
  { value: 'csharp', label: 'C#' },
  { value: 'go', label: 'Go' },
  { value: 'rust', label: 'Rust' },
  { value: 'php', label: 'PHP' },
  { value: 'ruby', label: 'Ruby' },
  { value: 'swift', label: 'Swift' },
  { value: 'kotlin', label: 'Kotlin' },
  { value: 'html', label: 'HTML' },
  { value: 'css', label: 'CSS' },
  { value: 'scss', label: 'SCSS' },
  { value: 'json', label: 'JSON' },
  { value: 'yaml', label: 'YAML' },
  { value: 'sql', label: 'SQL' },
  { value: 'bash', label: 'Bash' },
  { value: 'shell', label: 'Shell' },
  { value: 'markdown', label: 'Markdown' },
  { value: 'plaintext', label: 'Plain Text' },
];

// 文本颜色状态
const textColor = ref('#000000');
const highlightColor = ref('#FFFF00');

// 初始化编辑器
const editor = useEditor({
  content: props.modelValue,
  extensions: [
    StarterKit.configure({
      link: false,
      underline: false,
      horizontalRule: false,
      codeBlock: false, // 禁用默认的 codeBlock，使用 CodeBlockLowlight
    }),
    Underline,
    Subscript,
    Superscript,
    TaskList,
    TaskItem.configure({
      nested: true,
    }),
    HorizontalRule,
    Highlight.configure({ multicolor: true }),
    Color,
    TextStyle,
    CustomCodeBlockLowlight,
    Table.configure({
      resizable: true,
    }),
    TableRow,
    TableHeader,
    TableCell,
    Placeholder.configure({ placeholder: props.placeholder }),
    Link.configure({ openOnClick: false, autolink: true }),
    Image.configure({ inline: true }),
    CustomVideo,
    CustomIframe,
    TextAlign.configure({ types: ['heading', 'paragraph'] }),
  ],
  editable: !props.disabled,
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
watch(
  () => props.modelValue,
  (newVal) => {
    if (editor.value && newVal !== contentRef.value) {
      isInternalUpdate = true;
      editor.value.commands.setContent(newVal);
      contentRef.value = newVal;
    }
  },
);

// 监听禁用状态
watch(
  () => props.disabled,
  (newVal) => {
    editor.value?.setEditable(!newVal);
  },
);

// 监听主题变化
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

// 监听 placeholder 变化
watch(
  () => props.placeholder,
  (newVal) => {
    const placeholderExt = editor.value?.extensionManager?.extensions.find(
      (e: any) => e.name === 'placeholder',
    );
    if (placeholderExt?.options) {
      placeholderExt.options.placeholder = newVal;
    }
  },
);

// 监听 Modal 可见性，自动聚焦输入框
watch(
  () => linkModalVisible.value,
  (visible) => {
    if (visible) {
      nextTick(() => {
        const input = document.querySelector(
          '.ant-modal:last-of-type .ant-input',
        ) as HTMLInputElement;
        input?.focus();
      });
    }
  },
);

// 图片上传处理
const handleImageUpload = async (event: Event) => {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  if (!file || !props.uploadImage) {
    emit('imageUpload', file!);
    return;
  }

  try {
    const url = await props.uploadImage(file);
    if (url && editor.value) {
      editor.value.chain().focus().setImage({ src: url }).run();
    }
  } catch (error) {
    console.error('Image upload failed:', error);
  } finally {
    input.value = '';
  }
};

const handleMarkdownImport = async (event: Event) => {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  if (!file) {
    return;
  }

  try {
    const markdown = await file.text();
    const html = marked.parse(markdown);
    if (editor.value) {
      editor.value.commands.setContent(String(html));
    }
  } catch (error) {
    console.error('Markdown import failed:', error);
  } finally {
    input.value = '';
  }
};

// 工具栏操作
const openLinkModal = () => {
  linkUrl.value = '';
  linkModalVisible.value = true;
};

const openCodeBlockModal = () => {
  codeBlockLanguage.value = 'javascript';
  codeBlockContent.value = '';
  codeBlockModalVisible.value = true;
};

const openVideoModal = () => {
  videoUrl.value = '';
  videoWidth.value = '100%';
  videoModalVisible.value = true;
};

const openIframeModal = () => {
  iframeUrl.value = '';
  iframeWidth.value = '100%';
  iframeHeight.value = '500px';
  iframeTitle.value = '';
  iframeAllowFullscreen.value = true;
  iframeModalVisible.value = true;
};

const toolbarActions = {
  toggleBold: () => editor.value?.chain().focus().toggleBold().run(),
  toggleItalic: () => editor.value?.chain().focus().toggleItalic().run(),
  toggleStrike: () => editor.value?.chain().focus().toggleStrike().run(),
  toggleUnderline: () => editor.value?.chain().focus().toggleUnderline().run(),
  toggleCode: () => editor.value?.chain().focus().toggleCode().run(),
  toggleHeading: (level: Level) =>
    editor.value?.chain().focus().toggleHeading({ level }).run(),
  toggleBulletList: () =>
    editor.value?.chain().focus().toggleBulletList().run(),
  toggleOrderedList: () =>
    editor.value?.chain().focus().toggleOrderedList().run(),
  toggleTaskList: () => editor.value?.chain().focus().toggleTaskList().run(),
  insertCodeBlock: () => openCodeBlockModal(),
  toggleBlockquote: () =>
    editor.value?.chain().focus().toggleBlockquote().run(),
  toggleSubscript: () => editor.value?.chain().focus().toggleSubscript().run(),
  toggleSuperscript: () =>
    editor.value?.chain().focus().toggleSuperscript().run(),
  setParagraph: () => editor.value?.chain().focus().setParagraph().run(),
  clearFormatting: () =>
    editor.value?.chain().focus().unsetAllMarks().clearNodes().run(),
  insertHorizontalRule: () =>
    editor.value?.chain().focus().setHorizontalRule().run(),
  insertTable: () =>
    editor.value
      ?.chain()
      .focus()
      .insertTable({ rows: 3, cols: 3, withHeaderRow: true })
      .run(),
  deleteTable: () => editor.value?.chain().focus().deleteTable().run(),
  addRowBefore: () => editor.value?.chain().focus().addRowBefore().run(),
  addRowAfter: () => editor.value?.chain().focus().addRowAfter().run(),
  deleteRow: () => editor.value?.chain().focus().deleteRow().run(),
  addColumnBefore: () => editor.value?.chain().focus().addColumnBefore().run(),
  addColumnAfter: () => editor.value?.chain().focus().addColumnAfter().run(),
  deleteColumn: () => editor.value?.chain().focus().deleteColumn().run(),
  mergeCells: () => editor.value?.chain().focus().mergeCells().run(),
  splitCell: () => editor.value?.chain().focus().splitCell().run(),
  toggleHeaderRow: () => editor.value?.chain().focus().toggleHeaderRow().run(),
  toggleHeaderColumn: () =>
    editor.value?.chain().focus().toggleHeaderColumn().run(),
  toggleHeaderCell: () =>
    editor.value?.chain().focus().toggleHeaderCell().run(),
  setAlign: (align: 'center' | 'justify' | 'left' | 'right') =>
    editor.value?.chain().focus().setTextAlign(align).run(),
  setTextColor: (color: string) =>
    editor.value?.chain().focus().setColor(color).run(),
  setHighlight: (color: string) =>
    editor.value?.chain().focus().toggleHighlight({ color }).run(),
  uploadImage: () => fileInputRef.value?.click(),
  insertVideo: () => openVideoModal(),
  insertIframe: () => openIframeModal(),
  importMarkdown: () => markdownInputRef.value?.click(),
  undo: () => editor.value?.chain().focus().undo().run(),
  redo: () => editor.value?.chain().focus().redo().run(),
  clearContent: () => {
    Modal.confirm({
      title: $t('ui.title.confirm'),
      icon: () => h(LucideCircleAlert),
      content: $t('ui.editor.clear_content_confirm'),
      okText: $t('common.confirm'),
      cancelText: $t('common.cancel'),
      onOk() {
        editor.value?.commands.setContent('');
      },
    });
  },
};

// 状态栏计算
const statusInfo = computed(() => {
  if (!editor.value) return { chars: 0, words: 0, cursor: '0:0' };

  const text = editor.value.getText();
  const chars = text.length;
  const words = text.trim() ? text.trim().split(/\s+/).length : 0;

  // 获取光标位置（行:列）
  const { from } = editor.value.state.selection;
  const doc = editor.value.state.doc;
  let col = 1;
  let line = 1;
  let pos = 1;
  doc.descendants((node) => {
    if (node.isText) {
      const text = node.text || '';
      for (const element of text) {
        if (pos === from) {
          return false;
        }
        if (element === '\n') {
          line++;
          col = 1;
        } else {
          col++;
        }
        pos++;
      }
    }
    return true;
  });

  return { chars, words, cursor: `${line}:${col}` };
});

// 检查当前激活状态
const isActive = (name: string, options?: any) => {
  if (!editor.value) return false;
  return editor.value.isActive(name, options);
};

// 处理链接 Modal
const handleLinkOk = () => {
  const url = linkUrl.value.trim();
  if (url && editor.value) {
    editor.value
      .chain()
      .focus()
      .extendMarkRange('link')
      .setLink({ href: url })
      .run();
  }
  linkModalVisible.value = false;
  linkUrl.value = '';
};

const handleLinkCancel = () => {
  linkModalVisible.value = false;
  linkUrl.value = '';
  editor.value?.chain().focus().run();
};

// 处理代码块 Modal
const handleCodeBlockOk = () => {
  const code = codeBlockContent.value.trim();
  if (code && editor.value) {
    editor.value
      .chain()
      .focus()
      .insertContent({
        type: 'codeBlock',
        attrs: { language: codeBlockLanguage.value },
        content: [{ type: 'text', text: code }],
      })
      .run();
  }
  codeBlockModalVisible.value = false;
  codeBlockContent.value = '';
  codeBlockLanguage.value = 'javascript';
};

const handleCodeBlockCancel = () => {
  codeBlockModalVisible.value = false;
  codeBlockContent.value = '';
  codeBlockLanguage.value = 'javascript';
  editor.value?.chain().focus().run();
};

// 处理视频 Modal
const handleVideoOk = () => {
  const url = videoUrl.value.trim();
  if (url && editor.value) {
    (editor.value.chain().focus() as any)
      .setVideo({
        src: url,
        width: videoWidth.value,
      })
      .run();
  }
  videoModalVisible.value = false;
  videoUrl.value = '';
  videoWidth.value = '100%';
};

const handleVideoCancel = () => {
  videoModalVisible.value = false;
  videoUrl.value = '';
  videoWidth.value = '100%';
};

// 处理 iframe Modal
const handleIframeOk = () => {
  const url = iframeUrl.value.trim();
  if (url && editor.value) {
    (editor.value.chain().focus() as any)
      .setIframe({
        src: url,
        width: iframeWidth.value,
        height: iframeHeight.value,
        title: iframeTitle.value || undefined,
        allowfullscreen: iframeAllowFullscreen.value,
      })
      .run();
  }
  iframeModalVisible.value = false;
  iframeUrl.value = '';
  iframeWidth.value = '100%';
  iframeHeight.value = '500px';
  iframeTitle.value = '';
  iframeAllowFullscreen.value = true;
};

const handleIframeCancel = () => {
  iframeModalVisible.value = false;
  iframeUrl.value = '';
  iframeWidth.value = '100%';
  iframeHeight.value = '500px';
  iframeTitle.value = '';
  iframeAllowFullscreen.value = true;
  editor.value?.chain().focus().run();
};

// 销毁清理
onUnmounted(() => {
  editor.value?.destroy();
});
</script>

<template>
  <!-- prettier-ignore -->
  <div
    class="tiptap-editor-wrapper"
    :class="{ 'tiptap-editor-dark': isDark }"
    :data-theme="isDark ? 'dark' : 'light'"
    :style="{ height: typeof props.height === 'number' ? `${props.height}px` : props.height }"
  >
    <!-- Toolbar -->
    <div
      v-if="showToolbar"
      class="tiptap-toolbar"
      :class="{ 'border-b': !isDark }"
    >
      <div class="toolbar-group">
        <button
          type="button"
          class="toolbar-btn"
          :class="{ active: isActive('heading', { level: 1 }) }"
          @click="toolbarActions.toggleHeading(1)"
          :title="$t('ui.editor.h1')"
        >
          H1
        </button>
        <button
          type="button"
          class="toolbar-btn"
          :class="{ active: isActive('heading', { level: 2 }) }"
          @click="toolbarActions.toggleHeading(2)"
          :title="$t('ui.editor.h2')"
        >
          H2
        </button>
        <button
          type="button"
          class="toolbar-btn"
          :class="{ active: isActive('heading', { level: 3 }) }"
          @click="toolbarActions.toggleHeading(3)"
          :title="$t('ui.editor.h3')"
        >
          H3
        </button>
        <button
          type="button"
          class="toolbar-btn"
          :class="{ active: isActive('paragraph') }"
          @click="toolbarActions.setParagraph"
          :title="$t('ui.editor.paragraph')"
        >
          P
        </button>
      </div>

      <div class="toolbar-divider"></div>

      <div class="toolbar-group">
        <button
          type="button"
          class="toolbar-btn"
          :class="{ active: isActive('bold') }"
          @click="toolbarActions.toggleBold"
          :title="$t('ui.editor.bold')"
        >
          <svg
            class="icon"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M6 4h8a4 4 0 014 4 4 4 0 01-4 4H6z"
            />
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M6 12h9a4 4 0 014 4 4 4 0 01-4 4H6z"
            />
          </svg>
        </button>
        <button
          type="button"
          class="toolbar-btn"
          :class="{ active: isActive('italic') }"
          @click="toolbarActions.toggleItalic"
          :title="$t('ui.editor.italic')"
        >
          <svg
            class="icon"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M10 4h4m-2 0v16m-4 0h8"
            />
          </svg>
        </button>
        <button
          type="button"
          class="toolbar-btn"
          :class="{ active: isActive('strike') }"
          @click="toolbarActions.toggleStrike"
          :title="$t('ui.editor.strike')"
        >
          <svg
            class="icon"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M5 12h14M6 6a6 6 0 0112 0M6 18a6 6 0 0012 0"
            />
          </svg>
        </button>
        <button
          type="button"
          class="toolbar-btn"
          :class="{ active: isActive('underline') }"
          @click="toolbarActions.toggleUnderline"
          :title="$t('ui.editor.underline')"
        >
          <svg
            class="icon"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M6 3v8a6 6 0 006 6h0a6 6 0 006-6V3M3 21h18"
            />
          </svg>
        </button>
        <button
          type="button"
          class="toolbar-btn"
          :class="{ active: isActive('code') }"
          @click="toolbarActions.toggleCode"
          :title="$t('ui.editor.code')"
        >
          <svg
            class="icon"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4"
            />
          </svg>
        </button>
      </div>

      <div class="toolbar-divider"></div>

      <div class="toolbar-group">
        <button
          type="button"
          class="toolbar-btn"
          :class="{ active: isActive('subscript') }"
          @click="toolbarActions.toggleSubscript"
          :title="$t('ui.editor.subscript')"
        >
          <svg
            class="icon"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
          >
            <text x="3" y="12" font-size="16" font-weight="bold">X</text>
            <text x="14" y="18" font-size="10">2</text>
          </svg>
        </button>
        <button
          type="button"
          class="toolbar-btn"
          :class="{ active: isActive('superscript') }"
          @click="toolbarActions.toggleSuperscript"
          :title="$t('ui.editor.superscript')"
        >
          <svg
            class="icon"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
          >
            <text x="3" y="16" font-size="16" font-weight="bold">X</text>
            <text x="14" y="8" font-size="10">2</text>
          </svg>
        </button>
        <input
          type="color"
          v-model="textColor"
          @change="toolbarActions.setTextColor(textColor)"
          class="toolbar-color-picker"
          :title="$t('ui.editor.textColor')"
        />
        <input
          type="color"
          v-model="highlightColor"
          @change="toolbarActions.setHighlight(highlightColor)"
          class="toolbar-color-picker"
          :title="$t('ui.editor.highlightColor')"
        />
      </div>

      <div class="toolbar-divider"></div>

      <div class="toolbar-group">
        <button
          type="button"
          class="toolbar-btn"
          :class="{ active: isActive('bulletList') }"
          @click="toolbarActions.toggleBulletList"
          :title="$t('ui.editor.bulletList')"
        >
          <svg
            class="icon"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M4 6h16M4 12h16M4 18h16"
            />
            <circle cx="3" cy="6" r="1" fill="currentColor" />
            <circle cx="3" cy="12" r="1" fill="currentColor" />
            <circle cx="3" cy="18" r="1" fill="currentColor" />
          </svg>
        </button>
        <button
          type="button"
          class="toolbar-btn"
          :class="{ active: isActive('orderedList') }"
          @click="toolbarActions.toggleOrderedList"
          :title="$t('ui.editor.orderedList')"
        >
          <svg
            class="icon"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M4 6h16M4 12h16M4 18h16"
            />
            <text x="2" y="8" font-size="8" fill="currentColor">1.</text>
            <text x="2" y="14" font-size="8" fill="currentColor">2.</text>
            <text x="2" y="20" font-size="8" fill="currentColor">3.</text>
          </svg>
        </button>
        <button
          type="button"
          class="toolbar-btn"
          :class="{ active: isActive('taskList') }"
          @click="toolbarActions.toggleTaskList"
          :title="$t('ui.editor.taskList')"
        >
          <svg
            class="icon"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
          >
            <rect x="3" y="4" width="4" height="4" rx="1" />
            <path d="M10 6h10" stroke-width="2" />
            <rect x="3" y="12" width="4" height="4" rx="1" />
            <path d="M10 14h10" stroke-width="2" />
            <path d="M4 6l1 1 2-2" stroke-width="2" />
          </svg>
        </button>
        <button
          type="button"
          class="toolbar-btn"
          :class="{ active: isActive('blockquote') }"
          @click="toolbarActions.toggleBlockquote"
          :title="$t('ui.editor.blockquote')"
        >
          <svg
            class="icon"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M8 10h8M8 14h6M4 6h16v12H4z"
            />
          </svg>
        </button>
        <button
          type="button"
          class="toolbar-btn"
          :class="{ active: isActive('codeBlock') }"
          @click="toolbarActions.insertCodeBlock"
          :title="$t('ui.editor.insertCodeBlock')"
        >
          <svg
            class="icon"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4"
            />
            <rect
              x="3"
              y="3"
              width="18"
              height="18"
              rx="2"
              stroke="currentColor"
              fill="none"
            />
          </svg>
        </button>
        <button
          @click="toolbarActions.insertTable"
          type="button"
          class="toolbar-btn"
          :title="$t('ui.editor.insertTable')"
        >
          <svg
            class="icon"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
          >
            <rect x="3" y="3" width="18" height="18" rx="2" stroke-width="2" />
            <line x1="12" y1="3" x2="12" y2="21" stroke-width="2" />
            <line x1="3" y1="12" x2="21" y2="12" stroke-width="2" />
          </svg>
        </button>
        <button
          v-if="isActive('table')"
          type="button"
          class="toolbar-btn text-red-500"
          @click="toolbarActions.deleteTable"
          :title="$t('ui.editor.deleteTable')"
        >
          <svg
            class="icon"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
            />
          </svg>
        </button>
        <button
          type="button"
          class="toolbar-btn"
          @click="toolbarActions.insertHorizontalRule"
          :title="$t('ui.editor.insertHorizontalRule')"
        >
          <svg
            class="icon"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
          >
            <line x1="4" y1="12" x2="20" y2="12" stroke-width="2" />
          </svg>
        </button>
      </div>

      <div class="toolbar-divider"></div>

      <div class="toolbar-group">
        <button
          type="button"
          class="toolbar-btn"
          :class="{ active: isActive('textAlign', { textAlign: 'left' }) }"
          @click="toolbarActions.setAlign('left')"
          :title="$t('ui.editor.left')"
        >
          <svg
            class="icon"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M4 6h16M4 12h10M4 18h14"
            />
          </svg>
        </button>
        <button
          type="button"
          class="toolbar-btn"
          :class="{ active: isActive('textAlign', { textAlign: 'center' }) }"
          @click="toolbarActions.setAlign('center')"
          :title="$t('ui.editor.center')"
        >
          <svg
            class="icon"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M4 6h16M7 12h10M5 18h14"
            />
          </svg>
        </button>
        <button
          type="button"
          class="toolbar-btn"
          :class="{ active: isActive('textAlign', { textAlign: 'right' }) }"
          @click="toolbarActions.setAlign('right')"
          :title="$t('ui.editor.right')"
        >
          <svg
            class="icon"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M4 6h16M10 12h10M6 18h14"
            />
          </svg>
        </button>
        <button
          type="button"
          class="toolbar-btn"
          :class="{ active: isActive('textAlign', { textAlign: 'justify' }) }"
          @click="toolbarActions.setAlign('justify')"
          :title="$t('ui.editor.justify')"
        >
          <svg
            class="icon"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M4 6h16M4 12h16M4 18h16"
            />
          </svg>
        </button>
      </div>

      <div class="toolbar-divider"></div>

      <div class="toolbar-group" v-if="isActive('table')">
        <button
          type="button"
          class="toolbar-btn"
          @click="toolbarActions.addRowBefore"
          :title="$t('ui.editor.addRowBefore')"
        >
          R+
        </button>
        <button
          type="button"
          class="toolbar-btn"
          @click="toolbarActions.addRowAfter"
          :title="$t('ui.editor.addRowAfter')"
        >
          +R
        </button>
        <button
          type="button"
          class="toolbar-btn"
          @click="toolbarActions.deleteRow"
          :title="$t('ui.editor.deleteRow')"
        >
          R-
        </button>
        <button
          type="button"
          class="toolbar-btn"
          @click="toolbarActions.addColumnBefore"
          :title="$t('ui.editor.addColumnBefore')"
        >
          C+
        </button>
        <button
          type="button"
          class="toolbar-btn"
          @click="toolbarActions.addColumnAfter"
          :title="$t('ui.editor.addColumnAfter')"
        >
          +C
        </button>
        <button
          type="button"
          class="toolbar-btn"
          @click="toolbarActions.deleteColumn"
          :title="$t('ui.editor.deleteColumn')"
        >
          C-
        </button>
        <button
          type="button"
          class="toolbar-btn"
          @click="toolbarActions.mergeCells"
          :title="$t('ui.editor.mergeCells')"
        >
          M
        </button>
        <button
          type="button"
          class="toolbar-btn"
          @click="toolbarActions.splitCell"
          :title="$t('ui.editor.splitCell')"
        >
          S
        </button>
        <button
          type="button"
          class="toolbar-btn"
          @click="toolbarActions.toggleHeaderRow"
          :title="$t('ui.editor.toggleHeaderRow')"
        >
          HR
        </button>
        <button
          type="button"
          class="toolbar-btn"
          @click="toolbarActions.toggleHeaderColumn"
          :title="$t('ui.editor.toggleHeaderColumn')"
        >
          HC
        </button>
        <button
          type="button"
          class="toolbar-btn"
          @click="toolbarActions.toggleHeaderCell"
          :title="$t('ui.editor.toggleHeaderCell')"
        >
          H
        </button>
      </div>

      <div class="toolbar-divider"></div>

      <div class="toolbar-group">
        <button
          type="button"
          class="toolbar-btn"
          @click="toolbarActions.undo"
          :disabled="!editor?.can().undo()"
          :title="$t('ui.editor.undo')"
        >
          <svg
            class="icon"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M3 10h10a5 5 0 015 5v2M3 10l3-3m-3 3l3 3"
            />
          </svg>
        </button>
        <button
          type="button"
          class="toolbar-btn"
          @click="toolbarActions.redo"
          :disabled="!editor?.can().redo()"
          :title="$t('ui.editor.redo')"
        >
          <svg
            class="icon"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M21 10H11a5 5 0 00-5 5v2M21 10l-3-3m3 3l-3 3"
            />
          </svg>
        </button>
        <button
          type="button"
          class="toolbar-btn"
          @click="toolbarActions.clearFormatting"
          :title="$t('ui.editor.clearFormatting')"
        >
          <svg
            class="icon"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M4 6h16M8 6l8 12M6 18h12"
            />
          </svg>
        </button>
        <button
          type="button"
          class="toolbar-btn text-red-500"
          @click="toolbarActions.clearContent"
          :title="$t('ui.editor.clearContent')"
        >
          <svg
            class="icon"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
            />
          </svg>
        </button>
      </div>

      <div class="toolbar-divider"></div>

      <div class="toolbar-group">
        <button
          type="button"
          class="toolbar-btn"
          :class="{ active: isActive('link') }"
          @click="
            isActive('link')
              ? editor?.chain().focus().unsetLink().run()
              : openLinkModal()
          "
          :title="
            isActive('link')
              ? $t('ui.editor.removeUrl')
              : $t('ui.editor.insertUrl')
          "
        >
          <svg
            class="icon"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1"
            />
          </svg>
        </button>
        <button
          type="button"
          class="toolbar-btn"
          @click="toolbarActions.uploadImage"
          :title="$t('ui.editor.uploadImage')"
        >
          <svg
            class="icon"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"
            />
          </svg>
        </button>
        <button
          type="button"
          class="toolbar-btn"
          @click="toolbarActions.insertVideo"
          :title="$t('ui.editor.insertVideo')"
        >
          <svg
            class="icon"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z"
            />
          </svg>
        </button>
        <button
          type="button"
          class="toolbar-btn"
          @click="toolbarActions.insertIframe"
          :title="$t('ui.editor.insertIframe')"
        >
          <svg
            class="icon"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
          >
            <rect x="3" y="3" width="18" height="18" rx="2" stroke-width="2" />
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M8 12h8M12 8v8"
            />
          </svg>
        </button>
        <button
          type="button"
          class="toolbar-btn"
          @click="toolbarActions.importMarkdown"
          :title="$t('ui.editor.importMarkdown')"
        >
          MD
        </button>
        <input
          ref="fileInputRef"
          type="file"
          accept="image/*"
          class="hidden"
          @change="handleImageUpload"
        />
        <input
          ref="markdownInputRef"
          type="file"
          accept=".md,text/markdown"
          class="hidden"
          @change="handleMarkdownImport"
        />
      </div>
    </div>

    <!-- Editor Content -->
    <div class="tiptap-editor-content-wrapper">
      <EditorContent
        :editor="editor"
        class="tiptap-editor-content"
        :class="{ dark: isDark }"
      />
    </div>

    <!-- Status Bar -->
    <div
      v-if="showStatusBar"
      class="tiptap-statusbar"
      :class="{ 'border-t': !isDark }"
    >
      <div class="status-info">
        <span class="status-item">
          {{ statusInfo.words }} {{ $t('ui.editor.words') }}
        </span>
        <span class="status-divider">|</span>
        <span class="status-item">
          {{ statusInfo.chars }} {{ $t('ui.editor.chars') }}
        </span>
        <span class="status-divider">|</span>
        <span class="status-item">Ln {{ statusInfo.cursor }}</span>
      </div>
      <div class="status-mode">
        <span class="mode-badge" :class="{ 'mode-dark': isDark }">
          {{
            isDark
              ? `🌙 ${$t('preferences.theme.dark')}`
              : `☀️ ${$t('preferences.theme.light')}`
          }}
        </span>
      </div>
    </div>

    <!-- Link Input Modal -->
    <Modal
      v-model:open="linkModalVisible"
      :title="$t('ui.title.insert_url')"
      @ok="handleLinkOk"
      @cancel="handleLinkCancel"
      :ok-text="$t('common.confirm') || 'OK'"
      :cancel-text="$t('common.cancel') || 'Cancel'"
      :mask-closable="false"
    >
      <Input
        v-model:value="linkUrl"
        :placeholder="$t('ui.placeholder.input_url')"
        allow-clear
        @keyup.enter="handleLinkOk"
        @focus="() => {}"
      />
    </Modal>

    <!-- Code Block Insert Modal -->
    <Modal
      v-model:open="codeBlockModalVisible"
      :title="$t('ui.editor.insertCodeBlock')"
      @ok="handleCodeBlockOk"
      @cancel="handleCodeBlockCancel"
      :ok-text="$t('common.confirm') || 'OK'"
      :cancel-text="$t('common.cancel') || 'Cancel'"
      :mask-closable="false"
      width="600px"
    >
      <div class="code-block-modal">
        <div class="modal-field">
          <label class="field-label">
            {{ $t('ui.editor.codeLanguage') }}
          </label>
          <a-select
            v-model:value="codeBlockLanguage"
            :options="languages"
            :placeholder="$t('ui.editor.selectLanguage')"
            show-search
            class="language-select"
          />
        </div>
        <div class="modal-field">
          <label class="field-label">
            {{ $t('ui.editor.codeContent') }}
          </label>
          <textarea
            v-model="codeBlockContent"
            :placeholder="$t('ui.editor.codeContentPlaceholder')"
            class="code-textarea"
            rows="10"
          ></textarea>
        </div>
      </div>
    </Modal>

    <!-- Video Insert Modal -->
    <Modal
      v-model:open="videoModalVisible"
      :title="$t('ui.editor.insertVideo')"
      @ok="handleVideoOk"
      @cancel="handleVideoCancel"
      :ok-text="$t('common.confirm') || 'OK'"
      :cancel-text="$t('common.cancel') || 'Cancel'"
      :mask-closable="false"
      width="500px"
    >
      <div class="video-modal">
        <div class="modal-field">
          <label class="field-label">
            {{ $t('ui.editor.videoUrl') }}
          </label>
          <Input
            v-model:value="videoUrl"
            :placeholder="$t('ui.editor.videoUrlPlaceholder')"
            allow-clear
            @keyup.enter="handleVideoOk"
          />
        </div>
        <div class="modal-field">
          <label class="field-label">
            {{ $t('ui.editor.videoWidth') }}
          </label>
          <a-select
            v-model:value="videoWidth"
            :options="[
              { value: '100%', label: '100%' },
              { value: '75%', label: '75%' },
              { value: '50%', label: '50%' },
              { value: '640px', label: '640px' },
              { value: '800px', label: '800px' },
            ]"
            class="width-select"
          />
        </div>
      </div>
    </Modal>

    <!-- Iframe Insert Modal -->
    <Modal
      v-model:open="iframeModalVisible"
      :title="$t('ui.editor.insertIframe')"
      @ok="handleIframeOk"
      @cancel="handleIframeCancel"
      :ok-text="$t('common.confirm') || 'OK'"
      :cancel-text="$t('common.cancel') || 'Cancel'"
      :mask-closable="false"
      width="500px"
    >
      <div class="iframe-modal">
        <div class="modal-field">
          <label class="field-label">
            {{ $t('ui.editor.iframeUrl') }}
          </label>
          <Input
            v-model:value="iframeUrl"
            :placeholder="$t('ui.editor.iframeUrlPlaceholder')"
            allow-clear
            @keyup.enter="handleIframeOk"
          />
        </div>
        <div class="modal-field">
          <label class="field-label">
            {{ $t('ui.editor.iframeWidth') }}
          </label>
          <a-select
            v-model:value="iframeWidth"
            :options="[
              { value: '100%', label: '100%' },
              { value: '75%', label: '75%' },
              { value: '50%', label: '50%' },
              { value: '640px', label: '640px' },
              { value: '800px', label: '800px' },
            ]"
            class="width-select"
          />
        </div>
        <div class="modal-field">
          <label class="field-label">
            {{ $t('ui.editor.iframeHeight') }}
          </label>
          <a-select
            v-model:value="iframeHeight"
            :options="[
              { value: '500px', label: '500px' },
              { value: '300px', label: '300px' },
              { value: '100%', label: '100%' },
            ]"
            class="height-select"
          />
        </div>
        <div class="modal-field">
          <label class="field-label">
            {{ $t('ui.editor.iframeTitle') }}
          </label>
          <Input
            v-model:value="iframeTitle"
            :placeholder="$t('ui.editor.iframeTitlePlaceholder')"
            allow-clear
          />
        </div>
        <div class="modal-field">
          <label class="field-label">
            {{ $t('ui.editor.allowFullscreen') }}
          </label>
          <div style="display: flex; gap: 12px; align-items: center">
            <a-switch v-model:checked="iframeAllowFullscreen" />
            <span>{{
              iframeAllowFullscreen
                ? $t('ui.editor.allowFullscreenEnabled')
                : $t('ui.editor.allowFullscreenDisabled')
            }}</span>
          </div>
        </div>
      </div>
    </Modal>
  </div>
</template>

<style scoped>
/* ============ 基础容器 ============ */
.tiptap-editor-wrapper {
  margin: 0 !important;
  padding: 0 !important;
  box-sizing: border-box !important;
  display: flex !important;
  flex-direction: column !important;
  width: 100% !important;
  height: 100% !important;
  overflow: hidden !important;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  background-color: #fff;
  border: none !important;
  border-radius: 0 !important;
  transition: all 0.2s ease;
}

/* ============ 暗黑模式变量 ============ */
.tiptap-editor-dark {
  --tte-bg-primary: #0f172a !important;
  --tte-bg-secondary: #1e293b !important;
  --tte-bg-tertiary: #334155 !important;
  --tte-text-primary: #f1f5f9 !important;
  --tte-text-secondary: #94a3b8 !important;
  --tte-text-muted: #64748b !important;
  --tte-border-primary: #334155 !important;
  --tte-border-secondary: #475569 !important;
  --tte-code-bg: #1e293b !important;
  --tte-code-text: #e2e8f0 !important;
  --tte-blockquote-border: #475569 !important;
  --tte-blockquote-text: #94a3b8 !important;
  --tte-link: #60a5fa !important;
  --tte-link-hover: #93c5fd !important;
  --tte-toolbar-bg: #1e293b !important;
  --tte-toolbar-btn: #cbd5e1 !important;
  --tte-toolbar-btn-hover: #334155 !important;
  --tte-toolbar-btn-active: #60a5fa !important;
  --tte-statusbar-bg: #1e293b !important;
}

.tiptap-editor-dark {
  background-color: var(--tte-bg-primary) !important;
  border-color: var(--tte-border-primary) !important;
}

/* ============ Toolbar ============ */
.tiptap-toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  align-items: center;
  padding: 6px 8px;
  user-select: none;
  background-color: #f8fafc;
  border-bottom: 1px solid #e5e7eb;
}

.tiptap-editor-dark .tiptap-toolbar {
  background-color: var(--tte-toolbar-bg) !important;
  border-bottom-color: var(--tte-border-primary) !important;
}

.toolbar-group {
  display: flex;
  gap: 2px;
  align-items: center;
}

.toolbar-divider {
  width: 1px;
  height: 24px;
  margin: 0 4px;
  background-color: #e5e7eb;
}

.tiptap-editor-dark .toolbar-divider {
  background-color: var(--tte-border-secondary) !important;
}

.toolbar-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  padding: 0;
  font-size: 12px;
  font-weight: 500;
  color: #475569;
  cursor: pointer;
  background: transparent;
  border: none;
  border-radius: 4px;
  transition: all 0.15s ease;
}

.toolbar-btn:hover:not(:disabled) {
  color: #1e293b;
  background-color: #e2e8f0;
}

.toolbar-btn.active {
  color: #1e40af;
  background-color: #bfdbfe;
}

.toolbar-btn:disabled {
  cursor: not-allowed;
  opacity: 0.4;
}

.toolbar-color-picker {
  width: 28px;
  height: 28px;
  padding: 2px;
  cursor: pointer;
  border: 1px solid #e5e7eb;
  border-radius: 4px;
  transition: all 0.15s ease;
}

.toolbar-color-picker:hover {
  border-color: #94a3b8;
  transform: scale(1.05);
}

.tiptap-editor-dark .toolbar-color-picker {
  border-color: var(--tte-border-secondary);
}

.tiptap-editor-dark .toolbar-color-picker:hover {
  border-color: var(--tte-text-primary);
}

.tiptap-editor-dark .toolbar-btn {
  color: var(--tte-toolbar-btn) !important;
}

.tiptap-editor-dark .toolbar-btn:hover:not(:disabled) {
  color: var(--tte-text-primary) !important;
  background-color: var(--tte-toolbar-btn-hover) !important;
}

.tiptap-editor-dark .toolbar-btn.active {
  color: #fff !important;
  background-color: var(--tte-toolbar-btn-active) !important;
}

.toolbar-btn .icon {
  width: 16px;
  height: 16px;
}

/* ============ Editor Content ============ */
.tiptap-editor-content-wrapper {
  display: flex !important;
  flex: 1 !important;
  flex-direction: column !important;
  width: 100% !important;
  height: 100% !important;
  min-height: 0 !important;
  overflow: hidden !important;
  background-color: #ffffff !important;
}

.tiptap-editor-dark .tiptap-editor-content-wrapper {
  background-color: var(--tte-bg-primary) !important;
}

.tiptap-editor-content {
  display: flex !important;
  flex: 1 !important;
  flex-direction: column !important;
  width: 100% !important;
  height: 100% !important;
  min-height: 0 !important;
  overflow-y: auto !important;
  overflow-x: hidden !important;
  background-color: #ffffff !important;
}

.tiptap-editor-dark .tiptap-editor-content {
  background-color: var(--tte-bg-primary) !important;
}

.tiptap-editor-wrapper :deep(.ProseMirror) {
  box-sizing: border-box;
  flex: 1 !important;
  width: 100% !important;
  height: 100% !important;
  padding: 16px;
  font-size: 14px;
  line-height: 1.6;
  color: #1f2937;
  word-wrap: break-word;
  white-space: pre-wrap;
  background-color: transparent;
  outline: none;
  transition:
    background-color 0.2s ease,
    color 0.2s ease !important;
}

.tiptap-editor-dark :deep(.ProseMirror),
.tiptap-editor-content.dark :deep(.ProseMirror) {
  color: var(--tte-text-primary) !important;
  background-color: var(--tte-bg-primary) !important;
}

/* Placeholder */
/* prettier-ignore */
.tiptap-editor-wrapper :deep(.ProseMirror p.is-editor-empty:first-child::before) {
  float: left;
  height: 0;
  color: #9ca3af;
  pointer-events: none;
  content: attr(data-placeholder);
}

/* prettier-ignore */
.tiptap-editor-dark :deep(.ProseMirror p.is-editor-empty:first-child::before),
.tiptap-editor-content.dark :deep(.ProseMirror p.is-editor-empty:first-child::before) {
  color: var(--tte-text-secondary) !important;
}

/* Headings */
.tiptap-editor-wrapper :deep(.ProseMirror h1) {
  margin: 16px 0;
  font-size: 28px;
  font-weight: 700;
  color: #111827;
}

.tiptap-editor-wrapper :deep(.ProseMirror h2) {
  margin: 12px 0;
  font-size: 24px;
  font-weight: 700;
  color: #1f2937;
}

.tiptap-editor-wrapper :deep(.ProseMirror h3) {
  margin: 8px 0;
  font-size: 20px;
  font-weight: 700;
  color: #374151;
}

.tiptap-editor-dark :deep(.ProseMirror h1),
.tiptap-editor-dark :deep(.ProseMirror h2),
.tiptap-editor-dark :deep(.ProseMirror h3),
.tiptap-editor-content.dark :deep(.ProseMirror h1),
.tiptap-editor-content.dark :deep(.ProseMirror h2),
.tiptap-editor-content.dark :deep(.ProseMirror h3) {
  color: var(--tte-text-primary) !important;
}

/* Paragraphs & Lists */
.tiptap-editor-wrapper :deep(.ProseMirror p) {
  margin: 8px 0;
}

.tiptap-editor-wrapper :deep(.ProseMirror ul),
.tiptap-editor-wrapper :deep(.ProseMirror ol) {
  padding-left: 24px;
  margin: 8px 0;
}

.tiptap-editor-wrapper :deep(.ProseMirror li) {
  margin: 4px 0;
}

/* Code */
.tiptap-editor-wrapper :deep(.ProseMirror code) {
  padding: 2px 6px;
  font-family: Monaco, Menlo, 'Courier New', monospace;
  font-size: 13px;
  color: #ef4444;
  background-color: #f3f4f6;
  border-radius: 3px;
}

.tiptap-editor-wrapper :deep(.ProseMirror pre) {
  position: relative;
  padding: 12px;
  margin: 8px 0;
  overflow-x: auto;
  color: #f3f4f6;
  background-color: #1f2937;
  border-radius: 6px;
}

.tiptap-editor-wrapper :deep(.ProseMirror pre code) {
  display: block;
  padding: 0;
  font-family: Monaco, Menlo, 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.6;
  color: inherit;
  background: none;
  border-radius: 0;
}

/* Code Block Language Selector */
.tiptap-editor-wrapper :deep(.ProseMirror pre) {
  position: relative;
  padding-top: 36px; /* 为语言选择器留出空间 */
}

.tiptap-editor-wrapper :deep(.ProseMirror pre .code-block-language-selector) {
  position: absolute;
  top: 8px;
  right: 8px;
  z-index: 10;
}

.tiptap-editor-wrapper
  :deep(.ProseMirror pre .code-block-language-selector select) {
  padding: 2px 20px 2px 8px;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  font-size: 11px;
  color: #9ca3af;
  text-transform: uppercase;
  appearance: none;
  cursor: pointer;
  background-color: rgb(0 0 0 / 30%);
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 12 12'%3E%3Cpath fill='%239ca3af' d='M6 9L1 4h10z'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 4px center;
  background-size: 8px;
  border: 1px solid rgb(255 255 255 / 10%);
  border-radius: 3px;
  outline: none;
}

/* prettier-ignore */
.tiptap-editor-wrapper :deep(.ProseMirror pre .code-block-language-selector select:hover) {
  background-color: rgb(0 0 0 / 50%);
  border-color: rgb(255 255 255 / 20%);
}

/* prettier-ignore */
.tiptap-editor-wrapper :deep(.ProseMirror pre .code-block-language-selector select:focus) {
  background-color: rgb(0 0 0 / 60%);
  border-color: #3b82f6;
}

/* prettier-ignore */
.tiptap-editor-dark :deep(.ProseMirror pre .code-block-language-selector select) {
  color: #cbd5e1;
  background-color: rgb(255 255 255 / 10%);
  border-color: rgb(255 255 255 / 20%);
}

/* prettier-ignore */
.tiptap-editor-dark :deep(.ProseMirror pre .code-block-language-selector select:hover) {
  background-color: rgb(255 255 255 / 15%);
}

/* Syntax Highlighting (lowlight) */
.tiptap-editor-wrapper :deep(.ProseMirror .hljs-comment),
.tiptap-editor-wrapper :deep(.ProseMirror .hljs-quote) {
  font-style: italic;
  color: #6b7280;
}

.tiptap-editor-wrapper :deep(.ProseMirror .hljs-keyword),
.tiptap-editor-wrapper :deep(.ProseMirror .hljs-selector-tag),
.tiptap-editor-wrapper :deep(.ProseMirror .hljs-literal),
.tiptap-editor-wrapper :deep(.ProseMirror .hljs-type) {
  color: #c678dd;
}

.tiptap-editor-wrapper :deep(.ProseMirror .hljs-string),
.tiptap-editor-wrapper :deep(.ProseMirror .hljs-number) {
  color: #98c379;
}

.tiptap-editor-wrapper :deep(.ProseMirror .hljs-title),
.tiptap-editor-wrapper :deep(.ProseMirror .hljs-function) {
  color: #61afef;
}

.tiptap-editor-wrapper :deep(.ProseMirror .hljs-params) {
  color: #d19a66;
}

.tiptap-editor-wrapper :deep(.ProseMirror .hljs-built_in),
.tiptap-editor-wrapper :deep(.ProseMirror .hljs-class) {
  color: #e6c07b;
}

.tiptap-editor-wrapper :deep(.ProseMirror .hljs-attr),
.tiptap-editor-wrapper :deep(.ProseMirror .hljs-variable),
.tiptap-editor-wrapper :deep(.ProseMirror .hljs-property) {
  color: #e06c75;
}

.tiptap-editor-wrapper :deep(.ProseMirror .hljs-tag),
.tiptap-editor-wrapper :deep(.ProseMirror .hljs-name) {
  color: #e06c75;
}

.tiptap-editor-wrapper :deep(.ProseMirror .hljs-regexp) {
  color: #56b6c2;
}

.tiptap-editor-wrapper :deep(.ProseMirror .hljs-meta) {
  color: #abb2bf;
}

.tiptap-editor-dark :deep(.ProseMirror code),
.tiptap-editor-content.dark :deep(.ProseMirror code) {
  color: var(--tte-code-text) !important;
  background-color: var(--tte-code-bg) !important;
}

.tiptap-editor-dark :deep(.ProseMirror pre),
.tiptap-editor-content.dark :deep(.ProseMirror pre) {
  color: var(--tte-text-primary) !important;
  background-color: var(--tte-bg-secondary) !important;
  border: 1px solid var(--tte-border-secondary) !important;
}

/* Blockquote */
.tiptap-editor-wrapper :deep(.ProseMirror blockquote) {
  padding-left: 12px;
  margin: 8px 0;
  font-style: italic;
  color: #6b7280;
  border-left: 4px solid #d1d5db;
}

.tiptap-editor-dark :deep(.ProseMirror blockquote),
.tiptap-editor-content.dark :deep(.ProseMirror blockquote) {
  color: var(--tte-blockquote-text) !important;
  border-left-color: var(--tte-blockquote-border) !important;
}

/* Links */
.tiptap-editor-wrapper :deep(.ProseMirror a) {
  color: #3b82f6;
  text-decoration: underline;
  cursor: pointer;
}

.tiptap-editor-wrapper :deep(.ProseMirror a:hover) {
  color: #2563eb;
}

.tiptap-editor-dark :deep(.ProseMirror a),
.tiptap-editor-content.dark :deep(.ProseMirror a) {
  color: var(--tte-link) !important;
}

.tiptap-editor-dark :deep(.ProseMirror a:hover),
.tiptap-editor-content.dark :deep(.ProseMirror a:hover) {
  color: var(--tte-link-hover) !important;
}

/* Selection */
.tiptap-editor-wrapper :deep(.ProseMirror ::selection) {
  background-color: rgb(59 130 246 / 30%);
}

.tiptap-editor-dark :deep(.ProseMirror ::selection),
.tiptap-editor-content.dark :deep(.ProseMirror ::selection) {
  background-color: rgb(96 165 250 / 40%) !important;
}

/* Images */
.tiptap-editor-wrapper :deep(.ProseMirror img) {
  max-width: 100%;
  height: auto;
  margin: 8px 0;
  border-radius: 4px;
}

/* ============ Status Bar ============ */
.tiptap-statusbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px 12px;
  font-size: 12px;
  color: #64748b;
  background-color: #f8fafc;
  border-top: 1px solid #e5e7eb;
}

.tiptap-editor-dark .tiptap-statusbar {
  color: var(--tte-text-secondary) !important;
  background-color: var(--tte-statusbar-bg) !important;
  border-top-color: var(--tte-border-primary) !important;
}

.status-info {
  display: flex;
  gap: 8px;
  align-items: center;
}

.status-item {
  color: inherit;
}

.status-divider {
  color: #cbd5e1;
}

.tiptap-editor-dark .status-divider {
  color: var(--tte-border-secondary) !important;
}

.mode-badge {
  padding: 2px 8px;
  font-size: 11px;
  color: #475569;
  background-color: #e2e8f0;
  border-radius: 4px;
}

.tiptap-editor-dark .mode-badge {
  color: var(--tte-text-secondary) !important;
  background-color: var(--tte-bg-tertiary) !important;
}

/* ============ Scrollbar ============ */
.tiptap-editor-content::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

.tiptap-editor-content::-webkit-scrollbar-track {
  background: #f1f5f9;
}

.tiptap-editor-content::-webkit-scrollbar-thumb {
  background: #cbd5e1;
  border-radius: 4px;
}

.tiptap-editor-content::-webkit-scrollbar-thumb:hover {
  background: #94a3b8;
}

.tiptap-editor-dark .tiptap-editor-content::-webkit-scrollbar-track,
.tiptap-editor-content.dark::-webkit-scrollbar-track {
  background: var(--tte-bg-primary) !important;
}

.tiptap-editor-dark .tiptap-editor-content::-webkit-scrollbar-thumb,
.tiptap-editor-content.dark::-webkit-scrollbar-thumb {
  background: var(--tte-border-secondary) !important;
}

.tiptap-editor-dark .tiptap-editor-content::-webkit-scrollbar-thumb:hover,
.tiptap-editor-content.dark::-webkit-scrollbar-thumb:hover {
  background: #64748b !important;
}

.tiptap-editor-content {
  scrollbar-color: #cbd5e1 #f1f5f9;
  scrollbar-width: thin;
}

.tiptap-editor-dark .tiptap-editor-content,
.tiptap-editor-content.dark {
  scrollbar-color: var(--tte-border-secondary) var(--tte-bg-primary) !important;
}

/* ============ Disabled State ============ */
.tiptap-editor-wrapper:has(.ProseMirror[contenteditable='false']) {
  cursor: not-allowed;
  opacity: 0.7;
}

/* ============ Focus State ============ */
.tiptap-editor-wrapper:focus-within {
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgb(59 130 246 / 10%);
}

.tiptap-editor-dark:focus-within {
  border-color: var(--tte-link) !important;
  box-shadow: 0 0 0 3px rgb(96 165 250 / 20%) !important;
}

/* ============ Transitions ============ */
.tiptap-editor-wrapper,
.tiptap-editor-wrapper :deep(*) {
  transition:
    background-color 0.2s ease,
    color 0.2s ease,
    border-color 0.2s ease !important;
}

/* ============ Utility ============ */
.hidden {
  display: none;
}

/* ============ Code Block Modal ============ */
.code-block-modal {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.modal-field {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.field-label {
  font-size: 14px;
  font-weight: 500;
  color: #374151;
}

.tiptap-editor-dark .field-label {
  color: var(--tte-text-primary) !important;
}

.language-select {
  width: 100%;
}

.code-textarea {
  width: 100%;
  padding: 8px 12px;
  font-family: Monaco, Menlo, 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.5;
  resize: vertical;
  border: 1px solid #d1d5db;
  border-radius: 4px;
  transition: border-color 0.2s ease;
}

.code-textarea:focus {
  border-color: #3b82f6;
  outline: none;
  box-shadow: 0 0 0 3px rgb(59 130 246 / 10%);
}

.tiptap-editor-dark .code-textarea {
  color: var(--tte-text-primary) !important;
  background-color: var(--tte-bg-secondary) !important;
  border-color: var(--tte-border-secondary) !important;
}

.tiptap-editor-dark .code-textarea:focus {
  border-color: var(--tte-link) !important;
  box-shadow: 0 0 0 3px rgb(96 165 250 / 20%) !important;
}

/* ============ Video Modal ============ */
.video-modal {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.width-select {
  width: 100%;
}

/* ============ Video Styles ============ */
.tiptap-editor-wrapper :deep(.ProseMirror video) {
  display: block;
  max-width: 100%;
  height: auto;
  margin: 12px 0;
  cursor: pointer;
  border-radius: 6px;
}

.tiptap-editor-wrapper :deep(.ProseMirror video:hover) {
  box-shadow: 0 4px 12px rgb(0 0 0 / 15%);
}

.tiptap-editor-wrapper :deep(.ProseMirror .ProseMirror-selectednode video) {
  outline: 3px solid #3b82f6;
  outline-offset: 2px;
}

.tiptap-editor-dark :deep(.ProseMirror video) {
  box-shadow: 0 2px 8px rgb(0 0 0 / 30%);
}

.tiptap-editor-dark :deep(.ProseMirror .ProseMirror-selectednode video) {
  outline-color: var(--tte-link);
}

/* ============ Iframe Modal ============ */
.iframe-modal {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.height-select {
  width: 100%;
}
</style>
