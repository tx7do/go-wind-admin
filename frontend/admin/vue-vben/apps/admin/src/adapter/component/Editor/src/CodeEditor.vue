<script setup lang="ts">
import {
  computed,
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
  watch,
} from 'vue';

import { preferences } from '@vben/preferences';

import hljs from 'highlight.js';
import * as monaco from 'monaco-editor';

import { $t } from '#/locales';

import { initMonacoWorkers } from './monaco-loader';
import { isDarkMode } from './utils';

// 扩展monaco主题类型，增强类型安全
type MonacoTheme = 'dark' | 'hc-black' | 'light' | 'vs' | 'vs-dark';
type EditorLanguage =
  | 'c'
  | 'cpp'
  | 'csharp'
  | 'css'
  | 'go'
  | 'html'
  | 'java'
  | 'javascript'
  | 'json'
  | 'plaintext'
  | 'python'
  | 'typescript'
  | string;

interface Props {
  modelValue: string;
  height?: number | string;
  disabled?: boolean;
  placeholder?: string;
  autoDetectLanguage?: boolean;
  options?: {
    fontSize?: number;
    language?: EditorLanguage;
    lineNumbers?: boolean;
    // 扩展支持更多monaco原生配置
    minimap?: boolean;
    tabSize?: number;
    theme?: MonacoTheme;
    wordWrap?: 'bounded' | 'off' | 'on' | 'wordWrapColumn';
  };
}

// Props定义和默认值（必须在所有非导入代码之前）
const props = withDefaults(defineProps<Props>(), {
  autoDetectLanguage: true, // 默认启用自动侦测
  disabled: false,
  height: '100%',
  placeholder: $t('common.editor.please_input_content'),
  options: () => ({
    language: 'javascript',
    theme: 'light',
    lineNumbers: true,
    tabSize: 2,
    minimap: false,
    fontSize: 14,
    wordWrap: 'on',
  }),
});

const emit = defineEmits<{
  (e: 'change', value: string): void;
  (e: 'error', error: Error): void;
  (e: 'ready'): void;
  (e: 'update:modelValue', value: string): void;
}>();

const languageMap: Record<string, EditorLanguage> = {
  javascript: 'javascript',
  js: 'javascript',
  typescript: 'typescript',
  ts: 'typescript',
  json: 'json',
  html: 'html',
  css: 'css',
  python: 'python',
  java: 'java',
  sql: 'sql',
  markdown: 'markdown',
  shell: 'shell',
  php: 'php',
  go: 'go',
  golang: 'go',
  ruby: 'ruby',
  c: 'c',
  'c++': 'cpp',
  cplusplus: 'cpp',
  cpp: 'cpp',
  'c#': 'csharp',
  csharp: 'csharp',
};

/**
 * 根据代码内容自动检测编程语言
 */
const detectLanguage = (content: string): EditorLanguage => {
  try {
    if (!content || content.trim() === '') {
      return props.options?.language || 'plaintext';
    }

    // 1. 优先识别JSON（特殊处理：JSON识别准确率更高）
    if (content.trim().startsWith('{') || content.trim().startsWith('[')) {
      try {
        JSON.parse(content);
        return 'json';
      } catch {
        // JSON 解析失败，继续其他检测
      }
    }

    const detectedLanguage = hljs.highlightAuto(content);

    // 2. 使用language-detect识别
    const detected = detectedLanguage.language;
    const detectedKey =
      typeof detected === 'string' ? detected.toLowerCase() : '';
    // 3. 映射为monaco支持的语言ID，未匹配则返回纯文本
    return languageMap[detectedKey] || 'plaintext';
  } catch (error) {
    emit(
      'error',
      new Error(`Language detection failed: ${(error as Error).message}`),
    );
    return props.options?.language || 'plaintext';
  }
};

// 初始化monaco workers（确保只执行一次）
if (typeof window !== 'undefined') {
  initMonacoWorkers();
}

// 响应式数据
const editorContainer = ref<HTMLDivElement | null>(null);
let editor: monaco.editor.IStandaloneCodeEditor | null = null;
let editorModel: monaco.editor.ITextModel | null = null; // 单独管理model，避免内存泄漏
const isUpdatingFromProp = ref(false); // 改为响应式，避免闭包问题

// 计算编辑器高度（增加容错和最小高度）
const editorHeight = computed(() => {
  if (typeof props.height === 'number') {
    return `${Math.max(props.height, 200)}px`; // 最小高度200px
  }
  // 处理百分比/px字符串，默认100%
  return props.height?.toString() || '100%';
});

// 计算主题名称（兼容原生monaco主题值，支持暗黑模式自动切换）
const themeName = computed<MonacoTheme>(() => {
  const propsTheme = props.options?.theme;
  // 如果明确指定了主题，使用指定的主题
  if (propsTheme && propsTheme !== 'light' && propsTheme !== 'dark') {
    return propsTheme;
  }
  // 否则根据 isDarkMode 自动判断
  return isDarkMode() ? 'vs-dark' : 'vs';
});

// 监听外部值变化（优化防抖和空值处理）
watch(
  () => props.modelValue,
  async (newVal) => {
    if (!editor || isUpdatingFromProp.value) return;

    // 避免频繁更新，等待DOM就绪
    await nextTick();

    // 检查编辑器中的值是否与新值相同，如果相同则跳过更新
    const currentValue = editor.getValue();
    if (currentValue === newVal) {
      return;
    }

    isUpdatingFromProp.value = true;

    // 保存当前光标位置
    const currentPosition = editor.getPosition();

    // 空值处理：显示placeholder（monaco无原生placeholder，手动模拟）
    const valueToSet = newVal || '';
    editor.setValue(valueToSet);

    // 自动检测语言并更新
    if (props.autoDetectLanguage && editorModel && valueToSet.trim()) {
      const detectedLanguage = detectLanguage(valueToSet);
      monaco.editor.setModelLanguage(editorModel, detectedLanguage);
    }

    // 恢复光标位置，避免光标跳到第一位
    if (currentPosition && valueToSet.length >= currentPosition.column) {
      editor.setPosition(currentPosition);
    }

    isUpdatingFromProp.value = false;
  },
  { immediate: true, flush: 'post' }, // 确保DOM更新后执行
);

// 监听主题变化（优化生效逻辑）
watch(
  () => themeName.value,
  (newTheme) => {
    if (editor) {
      // 先设置全局主题，再强制重绘
      monaco.editor.setTheme(newTheme);
      editor.layout();
    }
  },
  { immediate: true },
);

// 监听系统暗黑模式变化（preferences）
watch(
  () => preferences.theme.mode,
  () => {
    // themeName computed 会自动重新计算，从而触发上面的 watch
    if (editor) {
      const newTheme = isDarkMode() ? 'vs-dark' : 'vs';
      monaco.editor.setTheme(newTheme);
      editor.layout();
    }
  },
);

watch(
  () => props.disabled,
  (disabled) => {
    if (editor) {
      editor.updateOptions({ readOnly: disabled });
    }
  },
);

// 初始化编辑器（优化错误处理和配置）
onMounted(async () => {
  try {
    if (!editorContainer.value) {
      const error = new Error('Editor container element not found');
      emit('error', error);
      console.error('Monaco editor initialization failed:', error);
      return;
    }

    await nextTick(); // 确保容器已挂载

    // 创建独立的model，便于后续管理
    const initialLanguage = props.autoDetectLanguage
      ? detectLanguage(props.modelValue || '')
      : props.options?.language || 'javascript';

    editorModel = monaco.editor.createModel(
      props.modelValue || '',
      initialLanguage,
    );

    // 创建编辑器实例（完善配置）
    editor = monaco.editor.create(editorContainer.value, {
      model: editorModel,
      theme: themeName.value,
      automaticLayout: true, // 自动适应容器大小
      minimap: {
        enabled: props.options?.minimap !== false,
      },
      lineNumbers: props.options?.lineNumbers === false ? 'off' : 'on',
      tabSize: props.options?.tabSize || 2,
      insertSpaces: true,
      readOnly: props.disabled,
      scrollBeyondLastLine: false,
      wordWrap: props.options?.wordWrap || 'on',
      fontSize: props.options?.fontSize || 14,
      fontFamily: "'Monaco', 'Menlo', 'Ubuntu Mono', 'Consolas', monospace",
      lineHeight: 1.6,
      // 增强用户体验的配置
      quickSuggestions: !props.disabled, // 禁用时关闭智能提示
      codeLens: !props.disabled,
      folding: true, // 启用代码折叠
      colorDecorators: true,
      // 性能优化
      renderLineHighlight: 'gutter',
      scrollbar: {
        vertical: 'visible',
        horizontal: 'auto',
      },
    });

    // 监听编辑器内容变化（防抖处理）
    let changeTimeout: null | number = null;
    editor.onDidChangeModelContent(() => {
      if (isUpdatingFromProp.value || !editor) return;

      // 防抖：避免频繁触发事件
      if (changeTimeout) clearTimeout(changeTimeout);
      changeTimeout = window.setTimeout(() => {
        const newValue = editor!.getValue() || '';
        emit('update:modelValue', newValue);
        emit('change', newValue);
      }, 100);
    });

    // 模拟placeholder（monaco无原生支持）
    const updatePlaceholder = () => {
      if (!editor || !props.placeholder) return;
      const value = editor.getValue().trim();
      const domNode = editor.getDomNode();
      if (domNode) {
        const placeholderEl = domNode.querySelector('.monaco-placeholder');
        if (!value && !placeholderEl) {
          const placeholder = document.createElement('div');
          placeholder.className = 'monaco-placeholder';
          placeholder.style.position = 'absolute';
          placeholder.style.top = '10px';
          placeholder.style.left = '10px';
          placeholder.style.color = 'hsl(var(--input-placeholder))';
          placeholder.style.pointerEvents = 'none';
          placeholder.style.fontSize = `${props.options?.fontSize || 14}px`;
          placeholder.textContent = props.placeholder;
          domNode.append(placeholder);
        } else if (value && placeholderEl) {
          placeholderEl.remove();
        }
      }
    };

    // 初始化placeholder并监听变化
    updatePlaceholder();
    editor.onDidChangeModelContent(updatePlaceholder);

    // 触发ready事件
    emit('ready');
  } catch (error) {
    emit('error', error as Error);
    console.error('Monaco editor initialization failed:', error);
  }
});

// 清理编辑器实例（彻底避免内存泄漏）
onBeforeUnmount(() => {
  if (editor) {
    editor.dispose();
    editor = null;
  }
  if (editorModel) {
    editorModel.dispose();
    editorModel = null;
  }
  // 清空monaco缓存（可选，根据实际情况）
  monaco.editor.getModels().forEach((model) => model.dispose());
});

// 暴露编辑器方法（增强类型安全）
defineExpose({
  focus: () => {
    editor?.focus();
    editor?.revealLine(1);
  },
  setLanguage: (language: EditorLanguage) => {
    if (editor && editorModel) {
      monaco.editor.setModelLanguage(editorModel, language);
    }
  },
  getValue: (): string => {
    return editor?.getValue() || '';
  },
  setValue: (value: string) => {
    if (editor) {
      isUpdatingFromProp.value = true;
      editor.setValue(value);
      isUpdatingFromProp.value = false;
    }
  },
  getEditorInstance: (): monaco.editor.IStandaloneCodeEditor | null => {
    return editor;
  },
  formatCode: () => {
    if (editor && editorModel && !props.disabled) {
      // 注：格式化需要集成外部格式化工具（如prettier）
      // 这里仅保留接口，实现需在外部完成
      console.info('Format code - requires external formatter integration');
    }
  },
});
</script>

<template>
  <div class="code-editor-wrapper">
    <div
      ref="editorContainer"
      class="code-editor-container"
      :style="{ height: editorHeight }"
      :class="{ 'code-editor-disabled': disabled }"
    ></div>
  </div>
</template>

<style scoped>
.code-editor-wrapper {
  position: relative;
  box-sizing: border-box;
  width: 100%;
  height: 100%;
}

.code-editor-container {
  box-sizing: border-box;
  width: 100%;
  min-height: 200px; /* 最小高度 */
  overflow: hidden;
  border: 1px solid hsl(var(--border));
  border-radius: 6px; /* 优化圆角 */
  transition: border-color 0.2s ease;
}

/* 禁用状态样式 */
.code-editor-disabled {
  cursor: not-allowed;
  opacity: 0.85;
}

/* Monaco编辑器样式穿透（优化默认样式） */
.code-editor-container :deep(.monaco-editor) {
  font-size: 14px !important;
}

.code-editor-container :deep(.monaco-editor .monaco-scrollable-element) {
  scrollbar-width: thin;
}

/* 滚动条样式 */
.code-editor-container,
:deep(.monaco-editor .monaco-scrollable-element::-webkit-scrollbar-thumb) {
  background-color: hsl(var(--accent-darker)) !important;
}

.code-editor-container,
:deep(.monaco-editor .monaco-scrollable-element::-webkit-scrollbar-track) {
  background-color: hsl(var(--accent)) !important;
}
</style>
