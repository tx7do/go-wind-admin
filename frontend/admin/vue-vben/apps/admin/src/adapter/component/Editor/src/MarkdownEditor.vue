<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue';

import { preferences } from '@vben/preferences';

import { type EditorProps, MdEditor } from 'md-editor-v3';

import { $t } from '#/locales';

import { isDarkMode } from './utils';

import 'md-editor-v3/lib/style.css';

interface UseEditorConfigProps {
  modelValue: string;
  height?: number | string;
  disabled?: boolean;
  placeholder?: string;
  options?: Partial<EditorProps>;
  enableExport?: boolean;
  uploadImage?: (file: File) => Promise<string>;
}

const props = withDefaults(defineProps<UseEditorConfigProps>(), {
  disabled: false,
  height: '100%', // 默认撑满父容器
  placeholder: $t('common.editor.please_input_content'),
  options: () => ({}),
  uploadImage: undefined,
});

const emit = defineEmits<{
  (e: 'change', value: string): void;
  (e: 'imageUpload', file: File): void;
  (e: 'ready'): void;
  (e: 'update:modelValue', value: string): void;
}>();

const localValue = ref(props.modelValue);
const isDark = ref(isDarkMode());
const wrapperRef = ref<HTMLDivElement>(); // 修正类型定义
let resizeObserver: null | ResizeObserver = null; // 监听容器尺寸变化

const toolbars = computed(() => {
  const base = [
    // === 文本格式 ===
    'bold',
    'underline',
    'italic',
    'strikeThrough',
    '-',

    // === 文本样式 ===
    'title',
    'sub',
    'sup',
    'alignLeft',
    'alignCenter',
    'alignRight',
    'alignJustify',
    '-',

    // === 列表与引用 ===
    'quote',
    'unorderedList',
    'orderedList',
    'task',
    'indent',
    'outdent',
    '-',

    // === 代码与插入 ===
    'codeRow',
    'code',
    'link',
    'image',
    'table',
    'horizontalRule',
    'emoji',
    'footnote',
    '-',

    // === 图表与公式 ===
    'mermaid',
    'katex',
    '-',

    // === 编辑操作 ===
    'revoke',
    'next',
    'clear',
    'save',
    '=',

    // === 视图与导出 ===
    'pageFullscreen',
    'fullscreen',
    'preview',
    'htmlPreview',
    'catalog',
    'help',
  ];

  // 导出功能（按需）
  if (props.enableExport) {
    base.splice(base.indexOf('save') + 1, 0, 'exportPdf', 'exportHtml', '-');
  }

  return base;
});

// 编辑器配置
const editorProps = computed(
  () =>
    ({
      preview: true,
      showCodeRowNumber: true,
      noMermaid: false,
      noKatex: false,
      toolbars: toolbars.value,
      // 合并用户自定义配置
      ...props.options,
    }) as EditorProps,
);

// 响应式高度（传给编辑器）
const editorHeight = ref<number>(600); // 初始兜底高度

// 计算编辑器主题
const theme = computed(() => (isDark.value ? 'dark' : 'light'));

// 核心：计算编辑器应有的高度（撑满可视区域）
const updateEditorHeight = () => {
  if (!wrapperRef.value) return;

  // 获取父容器的实际可用高度（撑满父容器）
  const containerRect = wrapperRef.value.getBoundingClientRect();
  const containerHeight = containerRect.height;

  // 如果传入height是100%，则使用容器高度；否则使用传入值
  if (props.height === '100%' || props.height === '100vh') {
    // 特殊处理：如果是100vh则使用视口高度
    const finalHeight =
      props.height === '100vh' ? window.innerHeight : containerHeight;
    // 避免高度为0的异常情况
    editorHeight.value = finalHeight > 0 ? finalHeight : 600; // 兜底高度600px
  } else if (typeof props.height === 'number') {
    editorHeight.value = props.height;
  } else if (typeof props.height === 'string') {
    // 处理带px的字符串（如"800px"）
    const pxMatch = props.height.match(/^(\d+)px$/);
    if (pxMatch) {
      editorHeight.value = Number(pxMatch[1]);
    } else {
      editorHeight.value = 600; // 兜底
    }
  }
};

const handleChange = (value: string) => {
  localValue.value = value;
  emit('update:modelValue', value);
  emit('change', value);
};

async function doUploadImage(file: File): Promise<string> {
  console.log('Uploading image:', file);

  if (!file || !props.uploadImage) {
    emit('imageUpload', file!);
    return '';
  }

  try {
    return await props.uploadImage(file);
  } catch (error) {
    console.error('Image upload failed:', error);
    return '';
  }
}

const handleUploadImages = async (
  files: File[],
  callback: (urls: string[]) => void,
) => {
  const uploadPromises = files.map((file) => doUploadImage(file));
  const urls = await Promise.all(uploadPromises);
  callback(urls);
};

const handleSave = (val: string) => {
  // 创建 Blob 对象（Markdown 格式）
  const blob = new Blob([val], { type: 'text/markdown;charset=utf-8' });

  // 生成文件名（带时间戳）
  const timestamp = new Date().toISOString().slice(0, 19).replaceAll(':', '-');
  const filename = `document-${timestamp}.md`;

  // 创建下载链接
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;

  // 触发下载
  document.body.append(link);
  link.click();

  // 清理
  link.remove();
  URL.revokeObjectURL(url);
};

// 监听外部值变化
watch(
  () => props.modelValue,
  (newVal) => {
    if (newVal !== localValue.value) {
      localValue.value = newVal;
    }
  },
);

// 监听主题变化
watch(
  () => preferences.theme.mode,
  (mode) => {
    isDark.value = mode === 'dark';
    nextTick(() => updateEditorHeight()); // 主题切换后重新计算高度
  },
);

// 监听props.height变化
watch(
  () => props.height,
  () => {
    nextTick(updateEditorHeight);
  },
);

onMounted(() => {
  emit('ready');
  nextTick(() => {
    updateEditorHeight(); // 初始化高度

    // 监听容器尺寸变化（窗口缩放/父容器变化时自动调整）
    if (wrapperRef.value) {
      resizeObserver = new ResizeObserver(() => {
        updateEditorHeight();
      });
      resizeObserver.observe(wrapperRef.value);

      // 监听窗口大小变化（兜底）
      window.addEventListener('resize', updateEditorHeight);
    }
  });
});

// 组件卸载时清理监听
onUnmounted(() => {
  if (resizeObserver) {
    resizeObserver.disconnect();
    resizeObserver = null;
  }
  window.removeEventListener('resize', updateEditorHeight);
});
</script>

<template>
  <div ref="wrapperRef" class="markdown-editor-wrapper">
    <MdEditor
      v-model="localValue"
      :theme="theme"
      :height="editorHeight"
      :placeholder="placeholder"
      :disabled="disabled"
      :preview="editorProps.preview"
      :toolbars="editorProps.toolbars"
      :show-code-row-number="editorProps.showCodeRowNumber"
      :no-mermaid="editorProps.noMermaid"
      :no-katex="editorProps.noKatex"
      @on-upload-img="handleUploadImages"
      @on-save="handleSave"
      @update:model-value="handleChange"
    />
  </div>
</template>

<style scoped>
.markdown-editor-wrapper {
  box-sizing: border-box;
  width: 100%;
  min-height: 1px;
  overflow: hidden;
  border: 1px solid hsl(var(--border));
  border-radius: 8px;
  transition:
    border-color 0.2s cubic-bezier(0.08, 0.82, 0.17, 1),
    box-shadow 0.2s cubic-bezier(0.08, 0.82, 0.17, 1);
}

.markdown-editor-wrapper :deep(.md-editor) {
  width: 100% !important;
  border: none !important;
  border-radius: 8px !important;
}

/* 聚焦态 */
.markdown-editor-wrapper:focus-within {
  border-color: hsl(var(--primary));
  box-shadow: 0 0 0 3px hsl(var(--primary) / 10%);
}
</style>

<!--
  暗色模式配色覆盖必须用非 scoped 块，
  scoped 的 :deep() 对 CSS 自定义属性的覆盖在 Vite 构建中不稳定。
  使用 html.dark 前缀 + .markdown-editor-wrapper 限定作用范围。
  颜色统一对接 Vben 框架 CSS 变量。
-->
<style>
/* CSS 变量覆盖 */
html.dark .markdown-editor-wrapper {
  border-color: hsl(var(--accent-dark));
}

html.dark .markdown-editor-wrapper:focus-within {
  border-color: hsl(var(--primary));
  box-shadow: 0 0 0 3px hsl(var(--primary) / 18%);
}

html.dark .markdown-editor-wrapper .md-editor.md-editor-dark {
  --md-color: hsl(var(--foreground));
  --md-hover-color: hsl(var(--foreground));
  --md-bk-color: hsl(var(--background));
  --md-bk-color-outstand: hsl(var(--accent-dark));
  --md-bk-hover-color: hsl(var(--accent-lighter));
  --md-border-color: hsl(var(--accent-dark));
  --md-border-hover-color: hsl(var(--accent-darker));
  --md-border-active-color: hsl(var(--primary));
}

/* 工具栏背景 */
html.dark
  .markdown-editor-wrapper
  .md-editor.md-editor-dark
  .md-editor-toolbar-wrapper {
  background-color: hsl(var(--accent-lighter)) !important;
  border-bottom-color: hsl(0deg 0% 100% / 8%) !important;
}

/* 工具栏按钮颜色 */
html.dark
  .markdown-editor-wrapper
  .md-editor.md-editor-dark
  .md-editor-toolbar
  .md-editor-toolbar-item {
  color: hsl(var(--muted-foreground)) !important;
}

/* 工具栏按钮 hover */
html.dark
  .markdown-editor-wrapper
  .md-editor.md-editor-dark
  .md-editor-toolbar
  .md-editor-toolbar-item:not([disabled]):hover {
  color: hsl(var(--foreground)) !important;
  background-color: hsl(0deg 0% 100% / 8%) !important;
}

/* CodeMirror 编辑区 */
html.dark .markdown-editor-wrapper .md-editor.md-editor-dark .cm-content {
  color: hsl(var(--foreground)) !important;
  caret-color: hsl(var(--foreground)) !important;
}

html.dark .markdown-editor-wrapper .md-editor.md-editor-dark .cm-editor {
  background-color: hsl(var(--background)) !important;
}

/* 编辑器外层背景 */
html.dark .markdown-editor-wrapper .md-editor.md-editor-dark {
  background-color: hsl(var(--background)) !important;
}

/* 输入区域背景 */
html.dark
  .markdown-editor-wrapper
  .md-editor.md-editor-dark
  .md-editor-input-wrapper {
  background-color: hsl(var(--background)) !important;
}

/* placeholder 颜色 */
html.dark
  .markdown-editor-wrapper
  .md-editor.md-editor-dark
  .md-editor-input-wrapper
  textarea::placeholder {
  color: hsl(var(--muted-foreground)) !important;
}
</style>
