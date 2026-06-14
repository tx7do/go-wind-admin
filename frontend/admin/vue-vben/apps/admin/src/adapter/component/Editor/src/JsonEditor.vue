<script setup lang="ts">
import {
  computed,
  getCurrentInstance,
  nextTick,
  onMounted,
  onUnmounted,
  ref,
  watch,
} from 'vue';

import { preferences } from '@vben/preferences';

import VueJsonEditor from 'json-editor-vue';

// vanilla-jsoneditor 暗色主题通过内联 <style> 引入（间接依赖无法直接 import）

// 类型定义
interface Props {
  modelValue: string;
  height?: number | string;
  disabled?: boolean;
  placeholder?: string;
  options?: {
    mode?: any;
    modes?: any[];
    search?: boolean;
  };
}

const props = withDefaults(defineProps<Props>(), {
  disabled: false,
  height: 500,
  placeholder: '{}',
  options: () => ({
    mode: 'text',
    modes: ['tree', 'code', 'form', 'text', 'view'],
    search: true,
  }),
});

const emit = defineEmits<{
  (e: 'change', value: string): void;
  (e: 'error', error: Error): void;
  (e: 'ready'): void;
  (e: 'update:modelValue', value: string): void;
}>();

// 响应式数据
const localValue = ref(props.modelValue);
const jsonData = ref<any[] | null | Record<string, any>>(null);
const parseError = ref<string>('');
const isValidJson = ref(true);
const instance = getCurrentInstance();
let observer: MutationObserver | null = null;
let themeObserver: MutationObserver | null = null;

const isDark = ref(false);

const updateIsDark = () => {
  const prefersDark = preferences.theme.mode === 'dark';
  if (typeof document === 'undefined') {
    isDark.value = prefersDark;
    return;
  }
  const root = document.documentElement;
  isDark.value =
    prefersDark ||
    root.classList.contains('dark') ||
    root.classList.contains('theme-dark') ||
    root.classList.contains('json-editor-dark');
};

// computed
// 验证并格式化 JSON
const validateAndFormat = (value: string) => {
  try {
    if (!value?.trim()) {
      parseError.value = '';
      isValidJson.value = true;
      return { parsed: null, formatted: '' };
    }
    const parsed = JSON.parse(String(value));
    const formatted = JSON.stringify(parsed, null, 2);
    parseError.value = '';
    isValidJson.value = true;
    return { parsed, formatted };
  } catch (error) {
    const err = error as Error;
    parseError.value = `JSON解析错误: ${err.message || '未知错误'}`;
    isValidJson.value = false;
    emit('error', err);
    return { parsed: null, formatted: value };
  }
};

// 初始化数据
const initData = () => {
  const { parsed, formatted } = validateAndFormat(props.modelValue);
  localValue.value = formatted || props.placeholder;

  // 🛡️ 确保 jsonData 是对象类型
  if (parsed !== null && typeof parsed === 'object') {
    jsonData.value = parsed;
  } else if (parsed === null) {
    jsonData.value = {};
  } else {
    // 兜底：非对象值包装处理
    jsonData.value = { value: parsed };
  }
};

// 监听外部值变化
watch(
  () => props.modelValue,
  (newVal) => {
    if (newVal !== localValue.value) {
      const { parsed, formatted } = validateAndFormat(newVal);
      localValue.value = formatted || newVal || props.placeholder;
      console.log('props.modelValue');
      try {
        jsonData.value = parsed || JSON.parse(props.placeholder);
      } catch {
        jsonData.value = {};
      }
    }
  },
  { immediate: true, deep: false },
);

// 监听编辑器内部数据变化
watch(
  () => jsonData.value,
  (newVal) => {
    if (newVal === null) return;

    if (typeof newVal === 'string') {
      if (newVal !== localValue.value) {
        localValue.value = newVal;
        emit('update:modelValue', newVal);
        emit('change', newVal);
      }
      return;
    }

    // 正常对象/数组：序列化为字符串
    try {
      const newValue = JSON.stringify(newVal, null, 2);
      if (newValue !== localValue.value) {
        localValue.value = newValue;
        emit('update:modelValue', newValue);
        emit('change', newValue);
      }
      parseError.value = '';
      isValidJson.value = true;
    } catch (error) {
      const err = error as Error;
      parseError.value = `JSON序列化错误: ${err.message || '未知错误'}`;
      isValidJson.value = false;
      emit('error', err);
    }
  },
  { deep: true },
);

// 高度计算（优化类型安全）
const editorHeight = computed(() => {
  let baseHeight = 500;

  if (typeof props.height === 'number') {
    baseHeight = props.height;
  } else if (typeof props.height === 'string') {
    const numericHeight = Number(props.height);
    if (!Number.isNaN(numericHeight)) {
      baseHeight = numericHeight;
    } else if (props.height.endsWith('px')) {
      const pxValue = Number(props.height.replace('px', ''));
      if (!Number.isNaN(pxValue)) {
        baseHeight = pxValue;
      }
    } else {
      // 百分比等非数值高度直接返回原字符串
      return props.height;
    }
  }

  const finalHeight = Math.max(baseHeight - 40, 200);
  return `${finalHeight}px`;
});

// 刷新编辑器样式
const refreshEditor = () => {
  nextTick(() => {
    const container = instance?.proxy?.$el as HTMLElement | undefined;
    if (!container) return;
    container.dataset.theme = isDark.value ? 'dark' : 'light';
  });
};

// 监听主题变化
watch(
  () => preferences.theme.mode,
  () => {
    updateIsDark();
    refreshEditor();
  },
  { immediate: true },
);

// 监听编辑器模式变化
watch(
  () => props.options?.mode,
  () => {
    refreshEditor();
  },
);

// 编辑器change事件处理
const handleEditorChange = (value: any) => {
  if (typeof value === 'string') {
    const rawValue = value;
    localValue.value = rawValue;
    emit('update:modelValue', rawValue);
    emit('change', rawValue);

    const { parsed } = validateAndFormat(rawValue);
    if (parsed !== null && typeof parsed === 'object') {
      jsonData.value = parsed;
    }
    return;
  }

  if (Array.isArray(value) || (value !== null && typeof value === 'object')) {
    return;
  }

  jsonData.value = { value };
  refreshEditor();
};

// 初始化和销毁逻辑
onMounted(() => {
  updateIsDark();
  initData();
  nextTick(() => {
    emit('ready');
    refreshEditor();

    if (typeof document !== 'undefined') {
      const root = document.documentElement;
      themeObserver = new MutationObserver(() => {
        updateIsDark();
        refreshEditor();
      });
      themeObserver.observe(root, {
        attributes: true,
        attributeFilter: ['class'],
      });
    }

    const container = instance?.proxy?.$el as HTMLElement | undefined;
    if (!container) return;

    const editorEl = container.querySelector('.json-editor-core');

    if (editorEl) {
      observer = new MutationObserver((mutations) => {
        const hasStyleChange = mutations.some(
          (m) =>
            m.type === 'attributes' &&
            ['class', 'style'].includes(m.attributeName || ''),
        );
        if (isDark.value && hasStyleChange) {
          refreshEditor();
        }
      });

      observer.observe(editorEl, {
        childList: true,
        subtree: true,
        attributes: true,
        attributeFilter: ['class', 'style'],
      });
    }
  });
});

onUnmounted(() => {
  if (themeObserver) {
    themeObserver.disconnect();
    themeObserver = null;
  }
  if (observer) {
    observer.disconnect();
    observer = null;
  }
});
</script>

<template>
  <div class="json-editor-container" :class="{ 'jse-theme-dark': isDark }">
    <!-- 错误提示 -->
    <div v-if="parseError" class="error-message">
      {{ parseError }}
    </div>

    <VueJsonEditor
      v-model="jsonData"
      :mode="options?.mode"
      :disabled="disabled"
      :search="options?.search"
      :placeholder="placeholder"
      :style="{ height: editorHeight, width: '100%' }"
      class="json-editor-core"
      @change="handleEditorChange"
    />
  </div>
</template>

<style scoped>
/* ============ 容器 ============ */
.json-editor-container {
  display: flex;
  flex-direction: column;
  width: 100%;
  height: 100%;
  overflow: hidden;
  border: 1px solid #cbd5e1;
  border-radius: 8px;
  background-color: #fff;
  transition:
    border-color 0.2s cubic-bezier(0.08, 0.82, 0.17, 1),
    box-shadow 0.2s cubic-bezier(0.08, 0.82, 0.17, 1);
  /* vanilla-jsoneditor 内置 --jse-* 变量体系 */
  --jse-theme-color: #f8fafc;
  --jse-theme-color-highlight: #f1f5f9;

  /* 模式切换按钮选中态 */
  --jse-button-primary-background: rgb(22 93 255 / 10%);
  --jse-button-primary-background-highlight: rgb(22 93 255 / 15%);
  --jse-button-primary-color: #165dff;
}

/* 聚焦态：蓝色边框 + 微光（对齐 Tiptap） */
.json-editor-container:focus-within {
  border-color: #165dff;
  box-shadow: 0 0 0 3px rgb(22 93 255 / 10%);
}

/* ============ 错误提示 ============ */
.error-message {
  padding: 8px 12px;
  margin: 0;
  font-size: 12px;
  line-height: 1.4;
  color: var(--jse-error-color, #ee5341);
  background-color: #fef2f2;
  border-bottom: 1px solid #fecaca;
}

/* ============ 编辑器核心 ============ */
.json-editor-container :deep(.json-editor-core) {
  flex: 1;
  width: 100%;
  overflow: hidden;
}
</style>

<!-- vanilla-jsoneditor 内置暗色主题（从 jse-theme-dark.css 内联，因 pnpm 间接依赖无法直接 import） -->
<style>
.jse-theme-dark {
  --jse-theme: dark !important;
  --jse-theme-color: #1a1d24 !important;
  --jse-theme-color-highlight: #313540 !important;

  /* ===== 背景色系（对齐 Tiptap --tte-* 变量） ===== */
  --jse-background-color: #14161a;
  --jse-panel-background: #1e2026;
  --jse-panel-background-border: 1px solid #313540;
  --jse-modal-background: #1e2026;
  --jse-modal-code-background: #1e2026;
  --jse-modal-overlay-background: rgb(0 0 0 / 60%);

  /* ===== 文字色系 ===== */
  --jse-text-color: #e3e6eb;
  --jse-text-color-inverse: #626773;
  --jse-menu-color: #e3e6eb;

  /* ===== 边框色系 ===== */
  --jse-main-border: 1px solid #313540;
  --jse-panel-border: 1px solid #313540;
  --jse-panel-color: #e3e6eb;
  --jse-panel-color-readonly: #626773;

  /* ===== 工具栏/菜单按钮 ===== */
  --jse-panel-button-color: #e3e6eb;
  --jse-panel-button-color-highlight: #ffffff;
  --jse-panel-button-background: transparent;
  --jse-panel-button-background-highlight: rgb(255 255 255 / 8%);

  /* 模式切换按钮（text/tree/table）选中态：用半透明蓝色强调 */
  --jse-button-primary-background: rgb(59 130 246 / 20%);
  --jse-button-primary-background-highlight: rgb(59 130 246 / 30%);
  --jse-button-primary-color: #93c5fd;

  /* ===== 导航栏 ===== */
  --jse-navigation-bar-background: #313540;
  --jse-navigation-bar-background-highlight: #475569;
  --jse-navigation-bar-dropdown-color: #e3e6eb;

  /* ===== 右键菜单 ===== */
  --jse-context-menu-background: #1e2026;
  --jse-context-menu-background-highlight: #313540;
  --jse-context-menu-separator-color: #313540;
  --jse-context-menu-color: #e3e6eb;
  --jse-context-menu-pointer-background: #475569;
  --jse-context-menu-pointer-background-highlight: #64748b;
  --jse-context-menu-pointer-color: #e3e6eb;

  /* ===== 语法高亮 ===== */
  --jse-key-color: #60a5fa;
  --jse-value-color: #e3e6eb;
  --jse-value-color-number: #98c379;
  --jse-value-color-boolean: #c678dd;
  --jse-value-color-null: #c678dd;
  --jse-value-color-string: #98c379;
  --jse-value-color-url: #60a5fa;
  --jse-delimiter-color: #626773;

  /* ===== 交互状态 ===== */
  --jse-edit-outline: 2px solid #3b82f6;
  --jse-selection-background-color: #313540;
  --jse-selection-background-inactive-color: #1e2026;
  --jse-hover-background-color: rgb(255 255 255 / 6%);
  --jse-active-line-background-color: rgb(255 255 255 / 4%);
  --jse-search-match-background-color: #313540;

  /* ===== 折叠/搜索 ===== */
  --jse-collapsed-items-background-color: #1e2026;
  --jse-collapsed-items-selected-background-color: #313540;
  --jse-collapsed-items-link-color: #94a3b8;
  --jse-collapsed-items-link-color-highlight: #60a5fa;
  --jse-search-match-color: #475569;
  --jse-search-match-outline: 1px solid #64748b;
  --jse-search-match-active-color: #475569;
  --jse-search-match-active-outline: 1px solid #64748b;

  /* ===== 标签/表格 ===== */
  --jse-tag-background: #313540;
  --jse-tag-color: #94a3b8;
  --jse-table-header-background: #1e2026;
  --jse-table-header-background-highlight: #313540;
  --jse-table-row-odd-background: rgb(255 255 255 / 3%);

  /* ===== 输入/按钮 ===== */
  --jse-input-background: #1e2026;
  --jse-input-border: 1px solid #313540;
  --jse-button-background: #475569;
  --jse-button-background-highlight: #64748b;
  --jse-button-color: #e3e6eb;
  --jse-button-secondary-background: #313540;
  --jse-button-secondary-background-highlight: #475569;
  --jse-button-secondary-background-disabled: #1e2026;
  --jse-button-secondary-color: #e3e6eb;
  --jse-a-color: #60a5fa;
  --jse-a-color-highlight: #93c5fd;

  /* ===== 提示框 ===== */
  --jse-tooltip-color: #e3e6eb;
  --jse-tooltip-background: #1e2026;
  --jse-tooltip-border: 1px solid #313540;
  --jse-tooltip-action-button-color: #e3e6eb;
  --jse-tooltip-action-button-background: #475569;

  /* ===== Svelte Select ===== */
  --jse-svelte-select-background: #1e2026;
  --jse-svelte-select-border: 1px solid #313540;
  --list-background: #1e2026;
  --item-hover-bg: #313540;
  --multi-item-bg: #313540;
  --input-color: #e3e6eb;
  --multi-clear-bg: #475569;
  --multi-item-clear-icon-color: #e3e6eb;
  --multi-item-outline: 1px solid #475569;
  --list-shadow: 0 2px 8px 0 rgb(0 0 0 / 40%);

  /* ===== 其他 ===== */
  --jse-color-picker-background: #313540;
  --jse-color-picker-border-box-shadow: #475569 0 0 0 1px;

  /* 聚焦态边框 */
  border-color: #313540 !important;
}

/* ===== 亮色模式：工具栏模式切换按钮（text/tree/table） ===== */
/* 工具栏背景改为白色而非默认蓝色 */
.json-editor-container:not(.jse-theme-dark) .jse-menu {
  background-color: #f8fafc !important;
}

/* 默认按钮文字：中灰 */
.json-editor-container:not(.jse-theme-dark) .jse-menu button {
  color: #64748b !important;
  background-color: transparent !important;
}

/* hover 按钮 */
.json-editor-container:not(.jse-theme-dark) .jse-menu button:hover {
  color: #1e293b !important;
  background-color: rgb(0 0 0 / 4%) !important;
}

/* 选中的模式按钮：淡蓝背景 + 蓝色文字 */
.json-editor-container:not(.jse-theme-dark) .jse-menu button[class*='selected'],
.json-editor-container:not(.jse-theme-dark) .jse-menu button[class*='active'],
.json-editor-container:not(.jse-theme-dark) .jse-menu button.selected,
.json-editor-container:not(.jse-theme-dark) .jse-menu button.active {
  color: #165dff !important;
  background-color: rgb(22 93 255 / 10%) !important;
}

/* 通过 inline style 设置的选中态 */
.json-editor-container:not(.jse-theme-dark) .jse-menu button[style*='theme-color'] {
  color: #165dff !important;
  background-color: rgb(22 93 255 / 10%) !important;
  background-image: none !important;
}

html.dark .json-editor-container:focus-within {
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgb(59 130 246 / 18%);
}

/* ===== 暗色模式：工具栏模式切换按钮（text/tree/table） ===== */
/* 默认按钮文字：浅灰 */
html.dark .jse-theme-dark .jse-menu button {
  color: #94a3b8 !important;
  background-color: transparent !important;
}

/* hover 按钮 */
html.dark .jse-theme-dark .jse-menu button:hover {
  color: #e3e6eb !important;
  background-color: rgb(255 255 255 / 6%) !important;
}

/* 选中的模式按钮：半透明蓝色背景 + 蓝色文字 */
html.dark .jse-theme-dark .jse-menu button[class*='selected'],
html.dark .jse-theme-dark .jse-menu button[class*='active'],
html.dark .jse-theme-dark .jse-menu button.selected,
html.dark .jse-theme-dark .jse-menu button.active {
  color: #93c5fd !important;
  background-color: rgb(59 130 246 / 18%) !important;
}

/* 通过 inline style 设置的选中态（vanilla-jsoneditor 用 --jse-theme-color 渲染） */
html.dark .jse-theme-dark .jse-menu button[style*='theme-color'] {
  color: #93c5fd !important;
  background-color: rgb(59 130 246 / 18%) !important;
  background-image: none !important;
}
</style>
