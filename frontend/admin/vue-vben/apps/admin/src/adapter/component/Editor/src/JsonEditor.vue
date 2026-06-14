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
  border: 1px solid hsl(var(--border));
  border-radius: 8px;
  background-color: hsl(var(--background));
  transition:
    border-color 0.2s cubic-bezier(0.08, 0.82, 0.17, 1),
    box-shadow 0.2s cubic-bezier(0.08, 0.82, 0.17, 1);
  /* vanilla-jsoneditor 内置 --jse-* 变量体系 */
  --jse-theme-color: hsl(var(--accent-lighter));
  --jse-theme-color-highlight: hsl(var(--accent));

  /* 模式切换按钮选中态 */
  --jse-button-primary-background: hsl(var(--primary) / 10%);
  --jse-button-primary-background-highlight: hsl(var(--primary) / 15%);
  --jse-button-primary-color: hsl(var(--primary));
}

/* 聚焦态 */
.json-editor-container:focus-within {
  border-color: hsl(var(--primary));
  box-shadow: 0 0 0 3px hsl(var(--primary) / 10%);
}

/* ============ 错误提示 ============ */
.error-message {
  padding: 8px 12px;
  margin: 0;
  font-size: 12px;
  line-height: 1.4;
  color: hsl(var(--destructive));
  background-color: hsl(var(--destructive) / 5%);
  border-bottom: 1px solid hsl(var(--destructive) / 20%);
}

/* ============ 编辑器核心 ============ */
.json-editor-container :deep(.json-editor-core) {
  flex: 1;
  width: 100%;
  overflow: hidden;
}
</style>

<!-- vanilla-jsoneditor 暗色主题（颜色对接 Vben 框架 CSS 变量） -->
<style>
.jse-theme-dark {
  --jse-theme: dark !important;
  --jse-theme-color: hsl(var(--accent-lighter)) !important;
  --jse-theme-color-highlight: hsl(var(--accent-dark)) !important;

  /* ===== 背景色系 ===== */
  --jse-background-color: hsl(var(--background));
  --jse-panel-background: hsl(var(--accent));
  --jse-panel-background-border: 1px solid hsl(var(--accent-dark));
  --jse-modal-background: hsl(var(--accent));
  --jse-modal-code-background: hsl(var(--accent));
  --jse-modal-overlay-background: hsl(0deg 0% 0% / 60%);

  /* ===== 文字色系 ===== */
  --jse-text-color: hsl(var(--foreground));
  --jse-text-color-inverse: hsl(var(--muted-foreground));
  --jse-menu-color: hsl(var(--foreground));

  /* ===== 边框色系 ===== */
  --jse-main-border: 1px solid hsl(var(--accent-dark));
  --jse-panel-border: 1px solid hsl(var(--accent-dark));
  --jse-panel-color: hsl(var(--foreground));
  --jse-panel-color-readonly: hsl(var(--muted-foreground));

  /* ===== 工具栏/菜单按钮 ===== */
  --jse-panel-button-color: hsl(var(--foreground));
  --jse-panel-button-color-highlight: hsl(var(--foreground));
  --jse-panel-button-background: transparent;
  --jse-panel-button-background-highlight: hsl(0deg 0% 100% / 8%);

  /* 模式切换按钮选中态 */
  --jse-button-primary-background: hsl(var(--primary) / 20%);
  --jse-button-primary-background-highlight: hsl(var(--primary) / 30%);
  --jse-button-primary-color: hsl(var(--primary));

  /* ===== 导航栏 ===== */
  --jse-navigation-bar-background: hsl(var(--accent-dark));
  --jse-navigation-bar-background-highlight: hsl(var(--accent-darker));
  --jse-navigation-bar-dropdown-color: hsl(var(--foreground));

  /* ===== 右键菜单 ===== */
  --jse-context-menu-background: hsl(var(--accent));
  --jse-context-menu-background-highlight: hsl(var(--accent-dark));
  --jse-context-menu-separator-color: hsl(var(--accent-dark));
  --jse-context-menu-color: hsl(var(--foreground));
  --jse-context-menu-pointer-background: hsl(var(--accent-darker));
  --jse-context-menu-pointer-background-highlight: hsl(var(--muted-foreground));
  --jse-context-menu-pointer-color: hsl(var(--foreground));

  /* ===== 语法高亮 ===== */
  --jse-key-color: hsl(var(--primary));
  --jse-value-color: hsl(var(--foreground));
  --jse-value-color-number: hsl(var(--success));
  --jse-value-color-boolean: hsl(291deg 61% 64%);
  --jse-value-color-null: hsl(291deg 61% 64%);
  --jse-value-color-string: hsl(var(--success));
  --jse-value-color-url: hsl(var(--primary));
  --jse-delimiter-color: hsl(var(--muted-foreground));

  /* ===== 交互状态 ===== */
  --jse-edit-outline: 2px solid hsl(var(--primary));
  --jse-selection-background-color: hsl(var(--accent-dark));
  --jse-selection-background-inactive-color: hsl(var(--accent));
  --jse-hover-background-color: hsl(0deg 0% 100% / 6%);
  --jse-active-line-background-color: hsl(0deg 0% 100% / 4%);
  --jse-search-match-background-color: hsl(var(--accent-dark));

  /* ===== 折叠/搜索 ===== */
  --jse-collapsed-items-background-color: hsl(var(--accent));
  --jse-collapsed-items-selected-background-color: hsl(var(--accent-dark));
  --jse-collapsed-items-link-color: hsl(var(--muted-foreground));
  --jse-collapsed-items-link-color-highlight: hsl(var(--primary));
  --jse-search-match-color: hsl(var(--accent-darker));
  --jse-search-match-outline: 1px solid hsl(var(--muted-foreground));
  --jse-search-match-active-color: hsl(var(--accent-darker));
  --jse-search-match-active-outline: 1px solid hsl(var(--muted-foreground));

  /* ===== 标签/表格 ===== */
  --jse-tag-background: hsl(var(--accent-dark));
  --jse-tag-color: hsl(var(--muted-foreground));
  --jse-table-header-background: hsl(var(--accent));
  --jse-table-header-background-highlight: hsl(var(--accent-dark));
  --jse-table-row-odd-background: hsl(0deg 0% 100% / 3%);

  /* ===== 输入/按钮 ===== */
  --jse-input-background: hsl(var(--accent));
  --jse-input-border: 1px solid hsl(var(--accent-dark));
  --jse-button-background: hsl(var(--accent-darker));
  --jse-button-background-highlight: hsl(var(--muted-foreground));
  --jse-button-color: hsl(var(--foreground));
  --jse-button-secondary-background: hsl(var(--accent-dark));
  --jse-button-secondary-background-highlight: hsl(var(--accent-darker));
  --jse-button-secondary-background-disabled: hsl(var(--accent));
  --jse-button-secondary-color: hsl(var(--foreground));
  --jse-a-color: hsl(var(--primary));
  --jse-a-color-highlight: hsl(var(--primary));

  /* ===== 提示框 ===== */
  --jse-tooltip-color: hsl(var(--foreground));
  --jse-tooltip-background: hsl(var(--accent));
  --jse-tooltip-border: 1px solid hsl(var(--accent-dark));
  --jse-tooltip-action-button-color: hsl(var(--foreground));
  --jse-tooltip-action-button-background: hsl(var(--accent-darker));

  /* ===== Svelte Select ===== */
  --jse-svelte-select-background: hsl(var(--accent));
  --jse-svelte-select-border: 1px solid hsl(var(--accent-dark));
  --list-background: hsl(var(--accent));
  --item-hover-bg: hsl(var(--accent-dark));
  --multi-item-bg: hsl(var(--accent-dark));
  --input-color: hsl(var(--foreground));
  --multi-clear-bg: hsl(var(--accent-darker));
  --multi-item-clear-icon-color: hsl(var(--foreground));
  --multi-item-outline: 1px solid hsl(var(--accent-darker));
  --list-shadow: 0 2px 8px 0 hsl(0deg 0% 0% / 40%);

  /* ===== 其他 ===== */
  --jse-color-picker-background: hsl(var(--accent-dark));
  --jse-color-picker-border-box-shadow: hsl(var(--accent-darker)) 0 0 0 1px;

  /* 聚焦态边框 */
  border-color: hsl(var(--accent-dark)) !important;
}

/* ===== 亮色模式：工具栏模式切换按钮 ===== */
.json-editor-container:not(.jse-theme-dark) .jse-menu {
  background-color: hsl(var(--accent-lighter)) !important;
}

.json-editor-container:not(.jse-theme-dark) .jse-menu button {
  color: hsl(var(--muted-foreground)) !important;
  background-color: transparent !important;
}

.json-editor-container:not(.jse-theme-dark) .jse-menu button:hover {
  color: hsl(var(--foreground)) !important;
  background-color: hsl(0deg 0% 0% / 4%) !important;
}

.json-editor-container:not(.jse-theme-dark) .jse-menu button[class*='selected'],
.json-editor-container:not(.jse-theme-dark) .jse-menu button[class*='active'],
.json-editor-container:not(.jse-theme-dark) .jse-menu button.selected,
.json-editor-container:not(.jse-theme-dark) .jse-menu button.active {
  color: hsl(var(--primary)) !important;
  background-color: hsl(var(--primary) / 10%) !important;
}

.json-editor-container:not(.jse-theme-dark)
  .jse-menu
  button[style*='theme-color'] {
  color: hsl(var(--primary)) !important;
  background-color: hsl(var(--primary) / 10%) !important;
  background-image: none !important;
}

html.dark .json-editor-container:focus-within {
  border-color: hsl(var(--primary));
  box-shadow: 0 0 0 3px hsl(var(--primary) / 18%);
}

/* ===== 暗色模式：工具栏模式切换按钮 ===== */
html.dark .jse-theme-dark .jse-menu button {
  color: hsl(var(--muted-foreground)) !important;
  background-color: transparent !important;
}

html.dark .jse-theme-dark .jse-menu button:hover {
  color: hsl(var(--foreground)) !important;
  background-color: hsl(0deg 0% 100% / 6%) !important;
}

html.dark .jse-theme-dark .jse-menu button[class*='selected'],
html.dark .jse-theme-dark .jse-menu button[class*='active'],
html.dark .jse-theme-dark .jse-menu button.selected,
html.dark .jse-theme-dark .jse-menu button.active {
  color: hsl(var(--primary)) !important;
  background-color: hsl(var(--primary) / 18%) !important;
}

html.dark .jse-theme-dark .jse-menu button[style*='theme-color'] {
  color: hsl(var(--primary)) !important;
  background-color: hsl(var(--primary) / 18%) !important;
  background-image: none !important;
}
</style>
