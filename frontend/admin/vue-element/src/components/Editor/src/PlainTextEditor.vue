<template>
  <div class="plain-text-editor-container">
    <textarea
      :value="localValue"
      :disabled="disabled"
      :placeholder="placeholder"
      :style="{ height: editorHeight }"
      class="plain-text-editor-textarea"
      @input="handleInput"
    ></textarea>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";

import { $t } from "@/core/i18n";

interface Props {
  modelValue: string;
  height?: number | string;
  disabled?: boolean;
  placeholder?: string;
}

const props = withDefaults(defineProps<Props>(), {
  disabled: false,
  height: "100%",
  placeholder: $t("common.editor.please_input_content"),
});

const emit = defineEmits<{
  (e: "change", value: string): void;
  (e: "ready"): void;
  (e: "update:modelValue", value: string): void;
}>();

const localValue = ref(props.modelValue);

// 监听外部值变化
watch(
  () => props.modelValue,
  (newVal) => {
    if (newVal !== localValue.value) {
      localValue.value = newVal;
    }
  },
  { immediate: true }
);

// 计算编辑器高度
const editorHeight = computed(() => {
  if (typeof props.height === "number") {
    return `${props.height}px`;
  }
  return props.height;
});

// 处理输入变化
const handleInput = (event: Event) => {
  const target = event.target as HTMLTextAreaElement;
  localValue.value = target.value;
  emit("update:modelValue", target.value);
  emit("change", target.value);
};

// 组件挂载后触发 ready 事件
defineExpose({
  focus: () => {
    const textarea = document.querySelector(".plain-text-editor-textarea") as HTMLTextAreaElement;
    textarea?.focus();
  },
});
</script>

<style scoped>
.plain-text-editor-container {
  position: relative;
  width: 100%;
  height: 100%;
}

.plain-text-editor-textarea {
  width: 100%;
  min-height: 300px;
  padding: 12px;
  font-family: Monaco, Menlo, "Ubuntu Mono", Consolas, source-code-pro, monospace;
  font-size: 14px;
  line-height: 1.6;
  color: var(--el-text-color-primary);
  resize: vertical;
  background-color: var(--el-bg-color);
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  outline: none;
  transition: border-color 0.3s;
}

.plain-text-editor-textarea:focus {
  border-color: var(--el-color-primary);
  box-shadow: 0 0 0 2px var(--el-color-primary-light-9);
}

.plain-text-editor-textarea:disabled {
  color: var(--el-text-color-disabled);
  cursor: not-allowed;
  background-color: var(--el-fill-color-light);
}

.plain-text-editor-textarea::placeholder {
  color: var(--el-text-color-placeholder);
}
</style>
