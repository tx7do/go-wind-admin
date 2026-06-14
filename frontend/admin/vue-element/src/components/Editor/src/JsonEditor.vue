<template>
  <div class="json-editor-container" :class="{ 'json-editor-dark': isDark }">
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

<script setup lang="ts">
import { computed, getCurrentInstance, nextTick, onMounted, onUnmounted, ref, watch } from "vue";

import { preferences } from "@/core/preferences";

import VueJsonEditor from "json-editor-vue";

import "jsoneditor/dist/jsoneditor.min.css";

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
  placeholder: "{}",
  options: () => ({
    mode: "text",
    modes: ["tree", "code", "form", "text", "view"],
    search: true,
  }),
});

const emit = defineEmits<{
  (e: "change", value: string): void;
  (e: "error", error: Error): void;
  (e: "ready"): void;
  (e: "update:modelValue", value: string): void;
}>();

// 响应式数据
const localValue = ref(props.modelValue);
const jsonData = ref<any[] | null | Record<string, any>>(null);
const parseError = ref<string>("");
const isValidJson = ref(true);
const instance = getCurrentInstance();
let observer: MutationObserver | null = null;
let themeObserver: MutationObserver | null = null;

const isDark = ref(false);

const updateIsDark = () => {
  const prefersDark = preferences.theme.mode === "dark";
  if (typeof document === "undefined") {
    isDark.value = prefersDark;
    return;
  }
  const root = document.documentElement;
  isDark.value =
    prefersDark ||
    root.classList.contains("dark") ||
    root.classList.contains("theme-dark") ||
    root.classList.contains("json-editor-dark");
};

// computed
// 验证并格式化 JSON
const validateAndFormat = (value: string) => {
  try {
    if (!value?.trim()) {
      parseError.value = "";
      isValidJson.value = true;
      return { parsed: null, formatted: "" };
    }
    const parsed = JSON.parse(String(value));
    const formatted = JSON.stringify(parsed, null, 2);
    parseError.value = "";
    isValidJson.value = true;
    return { parsed, formatted };
  } catch (error) {
    const err = error as Error;
    parseError.value = `JSON解析错误: ${err.message || "未知错误"}`;
    isValidJson.value = false;
    emit("error", err);
    return { parsed: null, formatted: value };
  }
};

// 初始化数据
const initData = () => {
  const { parsed, formatted } = validateAndFormat(props.modelValue);
  localValue.value = formatted || props.placeholder;

  // 🛡️ 确保 jsonData 是对象类型
  if (parsed !== null && typeof parsed === "object") {
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
      console.log("props.modelValue");
      try {
        jsonData.value = parsed || JSON.parse(props.placeholder);
      } catch {
        jsonData.value = {};
      }
    }
  },
  { immediate: true, deep: false }
);

// 监听编辑器内部数据变化
watch(
  () => jsonData.value,
  (newVal) => {
    if (newVal === null) return;

    if (typeof newVal === "string") {
      if (newVal !== localValue.value) {
        localValue.value = newVal;
        emit("update:modelValue", newVal);
        emit("change", newVal);
      }
      return;
    }

    // 正常对象/数组：序列化为字符串
    try {
      const newValue = JSON.stringify(newVal, null, 2);
      if (newValue !== localValue.value) {
        localValue.value = newValue;
        emit("update:modelValue", newValue);
        emit("change", newValue);
      }
      parseError.value = "";
      isValidJson.value = true;
    } catch (error) {
      const err = error as Error;
      parseError.value = `JSON序列化错误: ${err.message || "未知错误"}`;
      isValidJson.value = false;
      emit("error", err);
    }
  },
  { deep: true }
);

// 高度计算（优化类型安全）
const editorHeight = computed(() => {
  let baseHeight = 500;

  if (typeof props.height === "number") {
    baseHeight = props.height;
  } else if (typeof props.height === "string") {
    const numericHeight = Number(props.height);
    if (!Number.isNaN(numericHeight)) {
      baseHeight = numericHeight;
    } else if (props.height.endsWith("px")) {
      const pxValue = Number(props.height.replace("px", ""));
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
    container.dataset.theme = isDark.value ? "dark" : "light";
  });
};

// 监听主题变化
watch(
  () => preferences.theme.mode,
  () => {
    updateIsDark();
    refreshEditor();
  },
  { immediate: true }
);

// 监听编辑器模式变化
watch(
  () => props.options?.mode,
  () => {
    refreshEditor();
  }
);

// 编辑器change事件处理
const handleEditorChange = (value: any) => {
  if (typeof value === "string") {
    const rawValue = value;
    localValue.value = rawValue;
    emit("update:modelValue", rawValue);
    emit("change", rawValue);

    const { parsed } = validateAndFormat(rawValue);
    if (parsed !== null && typeof parsed === "object") {
      jsonData.value = parsed;
    }
    return;
  }

  if (Array.isArray(value) || (value !== null && typeof value === "object")) {
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
    emit("ready");
    refreshEditor();

    if (typeof document !== "undefined") {
      const root = document.documentElement;
      themeObserver = new MutationObserver(() => {
        updateIsDark();
        refreshEditor();
      });
      themeObserver.observe(root, {
        attributes: true,
        attributeFilter: ["class"],
      });
    }

    const container = instance?.proxy?.$el as HTMLElement | undefined;
    if (!container) return;

    const editorEl = container.querySelector(".json-editor-core");

    if (editorEl) {
      observer = new MutationObserver((mutations) => {
        const hasStyleChange = mutations.some(
          (m) => m.type === "attributes" && ["class", "style"].includes(m.attributeName || "")
        );
        if (isDark.value && hasStyleChange) {
          refreshEditor();
        }
      });

      observer.observe(editorEl, {
        childList: true,
        subtree: true,
        attributes: true,
        attributeFilter: ["class", "style"],
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

<style scoped>
.json-editor-container {
  display: flex;
  flex-direction: column;
  width: 100%;
  height: 100%;
  overflow: hidden;
  border: 1px solid var(--el-border-color-light);
  border-radius: 6px;
  transition: all 0.2s ease;
}

.json-editor-dark {
  --je-bg-primary: var(--dark-bg-table);
  --je-bg-secondary: var(--dark-bg-hover);
  --je-text-primary: var(--dark-text-primary);
  --je-text-secondary: var(--dark-text-secondary);
  --je-border-primary: var(--dark-border);
  --je-border-secondary: var(--dark-drawer-input-border);
  --je-error-bg: rgba(248, 113, 113, 0.1);
  --je-error-text: var(--el-color-danger);
  --je-error-border: var(--dark-border);

  background-color: var(--je-bg-primary) !important;
  border-color: var(--je-border-primary) !important;
}

.error-message {
  padding: 8px 12px;
  margin: 0;
  font-size: 12px;
  line-height: 1.4;
  color: var(--el-color-danger);
  background-color: var(--el-color-danger-light-9);
  border-bottom: 1px solid var(--el-color-danger-light-7);
}

.json-editor-dark .error-message {
  color: var(--je-error-text) !important;
  background-color: var(--je-error-bg) !important;
  border-bottom-color: var(--je-error-border) !important;
}

.json-editor-container :deep(.json-editor-core) {
  flex: 1;
  width: 100%;
  overflow: hidden;
}

/* 暗黑模式 - 基础样式 */
.json-editor-dark :deep(.jsoneditor) {
  font-family: Monaco, Consolas, "Courier New", monospace !important;
  font-size: 14px !important;
  color: var(--je-text-primary) !important;
  background-color: var(--je-bg-primary) !important;
  border: none !important;
}

.json-editor-dark :deep(.jsoneditor > *) {
  background-color: var(--je-bg-primary) !important;
}

/* 暗黑模式 - 菜单样式 */
.json-editor-dark :deep(.jsoneditor-menu) {
  padding: 4px !important;
  background-color: var(--je-bg-secondary) !important;
  border-bottom: 1px solid var(--je-border-primary) !important;
}

.json-editor-dark :deep(.jsoneditor-menu button) {
  padding: 4px 8px !important;
  margin: 0 2px !important;
  color: var(--je-text-primary) !important;
  border: none !important;
  border-radius: 4px !important;
  transition: background-color 0.2s ease !important;
}

.json-editor-dark :deep(.jsoneditor-menu button:hover) {
  background-color: var(--je-border-secondary) !important;
}

/* 暗黑模式 - 树状视图 */
.json-editor-dark :deep(.jsoneditor-tree) {
  padding: 8px !important;
  color: var(--je-text-primary) !important;
  background-color: var(--je-bg-primary) !important;
}

.json-editor-dark :deep(.jsoneditor-tree .jsoneditor-field) {
  margin-right: 4px !important;
  font-weight: 500 !important;
  color: var(--el-color-primary-light-5, #93c5fd) !important;
}

.json-editor-dark :deep(.jsoneditor-tree .jsoneditor-string) {
  color: #98c379 !important;
}

.json-editor-dark :deep(.jsoneditor-tree .jsoneditor-number) {
  color: #d19a66 !important;
}

.json-editor-dark :deep(.jsoneditor-tree .jsoneditor-boolean) {
  color: var(--el-color-primary-light-5, #60a5fa) !important;
}

.json-editor-dark :deep(.jsoneditor-tree .jsoneditor-null) {
  color: var(--je-text-secondary) !important;
}

/* 暗黑模式 - 代码/文本模式 */
.json-editor-dark :deep(.jsoneditor-code) {
  color: var(--je-text-primary) !important;
  background-color: var(--je-bg-primary) !important;
}

.json-editor-dark :deep(.jsoneditor-code textarea) {
  padding: 8px !important;
  font-family: Monaco, Consolas, "Courier New", monospace !important;
  color: var(--je-text-primary) !important;
  resize: none !important;
  background-color: var(--je-bg-primary) !important;
  border: none !important;
}

.json-editor-dark :deep(.jsoneditor-code textarea:focus) {
  border: 1px solid var(--dark-primary) !important;
  border-radius: 2px !important;
  outline: none !important;
}

/* 暗黑模式 - 表单模式 */
.json-editor-dark :deep(.jsoneditor-form) {
  color: var(--je-text-primary) !important;
  background-color: var(--je-bg-primary) !important;
}

.json-editor-dark :deep(.jsoneditor-form input),
.json-editor-dark :deep(.jsoneditor-form textarea),
.json-editor-dark :deep(.jsoneditor-form select) {
  color: var(--je-text-primary) !important;
  background-color: var(--je-bg-secondary) !important;
  border: 1px solid var(--je-border-secondary) !important;
  border-radius: 4px !important;
}

/* 暗黑模式 - 搜索框 */
.json-editor-dark :deep(.jsoneditor-search) {
  padding: 4px 8px !important;
  margin: 0 4px !important;
  color: var(--je-text-primary) !important;
  background-color: var(--je-bg-secondary) !important;
  border: 1px solid var(--je-border-primary) !important;
  border-radius: 4px !important;
}

.json-editor-dark :deep(.jsoneditor-search::placeholder) {
  color: var(--je-text-secondary) !important;
  opacity: 1 !important;
}

/* 暗黑模式 - 输入框 */
.json-editor-dark :deep(.jsoneditor-text-input) {
  padding: 2px 4px !important;
  color: var(--je-text-primary) !important;
  background-color: var(--je-bg-secondary) !important;
  border: 1px solid var(--je-border-secondary) !important;
  border-radius: 2px !important;
}

.json-editor-dark :deep(.jsoneditor-text-input:focus) {
  background-color: var(--je-bg-secondary) !important;
  border-color: var(--dark-primary) !important;
  outline: none !important;
}

/* 禁用状态 */
.json-editor-container :deep(.jsoneditor-disabled) {
  cursor: not-allowed !important;
  background-color: var(--el-fill-color-light) !important;
  opacity: 0.8 !important;
}

/* 暗黑模式 - 滚动条 */
.json-editor-dark :deep(.jsoneditor-tree::-webkit-scrollbar),
.json-editor-dark :deep(.jsoneditor-code::-webkit-scrollbar) {
  width: 8px !important;
  height: 8px !important;
}

.json-editor-dark :deep(.jsoneditor-tree::-webkit-scrollbar-track),
.json-editor-dark :deep(.jsoneditor-code::-webkit-scrollbar-track) {
  background: var(--je-bg-primary) !important;
}

.json-editor-dark :deep(.jsoneditor-tree::-webkit-scrollbar-thumb),
.json-editor-dark :deep(.jsoneditor-code::-webkit-scrollbar-thumb) {
  background: var(--je-border-secondary) !important;
  border-radius: 4px !important;
}

.json-editor-dark :deep(.jsoneditor-tree::-webkit-scrollbar-thumb:hover),
.json-editor-dark :deep(.jsoneditor-code::-webkit-scrollbar-thumb:hover) {
  background: var(--je-text-secondary) !important;
}

/* 亮色模式基础样式 */
.json-editor-container :deep(.jsoneditor) {
  font-family: Monaco, Consolas, "Courier New", monospace !important;
  font-size: 14px !important;
}

.json-editor-container :deep(.jsoneditor-menu) {
  padding: 4px !important;
}

.json-editor-container :deep(.jsoneditor-menu button) {
  padding: 4px 8px !important;
  border-radius: 4px !important;
}
</style>
