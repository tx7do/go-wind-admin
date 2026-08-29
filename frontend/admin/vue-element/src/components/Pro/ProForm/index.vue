<template>
  <ElForm v-bind="$attrs" ref="formRef" :size="size" :model="modelValue" :disabled="disabled">
    <ElRow :gutter="20">
      <ElCol
        v-for="field in resolvedFields"
        v-show="!field.hidden && (!field.displayIf || field.displayIf(modelValue))"
        :key="field.field"
        :span="field.span"
        v-bind="field.col"
      >
        <ElFormItem :label="field.label" :prop="field.field" :rules="field.rules">
          <!-- 标签 + 提示 -->
          <template #label>
            <span class="flex items-center gap-1">
              {{ field.label }}
              <ElTooltip v-if="field.tips" v-bind="getTooltipProps(field.tips)">
                <ElIcon class="text-gray-400"><QuestionFilled /></ElIcon>
              </ElTooltip>
              <span v-if="colon" class="ml-0.5">:</span>
            </span>
          </template>

          <!-- 自定义插槽 -->
          <slot
            v-if="field.slotName || field.type === 'custom'"
            :name="field.slotName ?? field.field"
            :model="modelValue"
            :field="field"
          />

          <!-- 异步树选择 -->
          <ElTreeSelect
            v-else-if="field.type === 'api-tree-select'"
            v-model="modelValue[field.field]"
            style="width: 100%"
            v-bind="field.attrs || {}"
            v-on="field.events || {}"
          />

          <!-- 动态渲染组件 -->
          <component
            :is="resolveComponent(field.type)"
            v-else
            v-model="modelValue[field.field]"
            style="width: 100%"
            v-bind="getBindAttrs(field)"
            v-on="field.events || {}"
          >
            <!-- 选项渲染：select/radio/checkbox -->
            <component
              :is="getChildrenComponent(field.type!)"
              v-for="opt in field.options || []"
              :key="opt.value"
              :label="opt.label"
              :value="opt.value"
              :disabled="opt.disabled"
            />
          </component>
        </ElFormItem>
      </ElCol>
    </ElRow>
  </ElForm>
</template>

<script setup lang="ts" generic="T extends Record<string, any>">
import { ref, computed, onMounted, markRaw } from "vue";
import { ElMessage } from "element-plus";
import { i18n } from "@/core/i18n";
import {
  ElForm,
  ElFormItem,
  ElInput,
  ElSelect,
  ElOption,
  ElSwitch,
  ElDatePicker,
  ElTimePicker,
  ElTimeSelect,
  ElInputNumber,
  ElCascader,
  ElTreeSelect,
  ElRadioGroup,
  ElRadio,
  ElCheckboxGroup,
  ElCheckbox,
  ElRow,
  ElCol,
  ElTooltip,
  ElIcon,
} from "element-plus";
import { QuestionFilled } from "@element-plus/icons-vue";
import InputTag from "@/components/InputTag/index.vue";
import IconSelect from "@/components/IconSelect/index.vue";
import type { ProFormField, FormValueType } from "./types";

const t = i18n.global.t;

// 关闭属性透传
defineOptions({ inheritAttrs: false });

// 组件 Props
const props = withDefaults(
  defineProps<{
    modelValue: T;
    fields: ProFormField<T>[];
    colon?: boolean;
    disabled?: boolean;
    size?: "" | "default" | "small" | "large";
  }>(),
  {
    colon: false,
    disabled: false,
    size: "default",
  }
);

// 表单实例
const formRef = ref<InstanceType<typeof ElForm>>();

// =============== 响应式字段解析 ===============
const resolvedFields = computed(() => {
  return props.fields.map((item) => ({
    ...item,
    span: item.span ?? 24,
  }));
});

// =============== 组件映射 ===============
const componentMap: Record<FormValueType, any> = {
  input: markRaw(ElInput),
  textarea: markRaw(ElInput),
  select: markRaw(ElSelect),
  radio: markRaw(ElRadioGroup),
  checkbox: markRaw(ElCheckboxGroup),
  switch: markRaw(ElSwitch),
  "date-picker": markRaw(ElDatePicker),
  "time-picker": markRaw(ElTimePicker),
  "time-select": markRaw(ElTimeSelect),
  "input-number": markRaw(ElInputNumber),
  cascader: markRaw(ElCascader),
  "tree-select": markRaw(ElTreeSelect),
  "api-tree-select": markRaw(ElTreeSelect),
  "input-tag": markRaw(InputTag),
  "custom-tag": markRaw(InputTag),
  "icon-select": markRaw(IconSelect),
  number: markRaw(ElInputNumber),
  date: markRaw(ElDatePicker),
  custom: null,
};

// 子选项组件映射
const childrenMap: Record<string, any> = {
  select: markRaw(ElOption),
  radio: markRaw(ElRadio),
  checkbox: markRaw(ElCheckbox),
};

// 解析主组件
const resolveComponent = (type?: FormValueType) => {
  if (!type) return ElInput;
  return componentMap[type] || ElInput;
};

// 解析子选项组件
const getChildrenComponent = (type: string) => {
  return childrenMap[type] || ElOption;
};

// =============== 工具函数 ===============
// 处理 textarea 自动注入 type 属性
const getBindAttrs = (field: ProFormField<T>) => {
  const attrs = { ...(field.attrs || {}) };
  if (field.type === "textarea") attrs.type = "textarea";
  return attrs;
};

// 提示框配置
const getTooltipProps = (tips: string | Record<string, any>) => {
  return typeof tips === "string" ? { content: tips, placement: "top" } : tips;
};

// =============== 初始化逻辑 ===============
onMounted(() => {
  resolvedFields.value.forEach((field) => {
    // 自动注入初始值
    if (field.initialValue !== undefined && props.modelValue[field.field] === undefined) {
      props.modelValue[field.field] = field.initialValue;
    }

    // 执行初始化函数
    field.initFn?.(field);

    // 异步树数据加载
    if (field.type === "api-tree-select" && field.api) {
      if (!field.attrs) field.attrs = {};
      field.attrs.loading = true;

      field
        .api()
        .then((data) => {
          field.attrs!.data = data;
        })
        .catch(() => ElMessage.error(t("common.message.loadDataFailed")))
        .finally(() => {
          field.attrs!.loading = false;
        });
    }
  });
});

// =============== 暴露方法 ===============
defineExpose({
  /** 表单实例 */
  formRef,
  /** 表单校验 */
  validate: () => formRef.value?.validate(),
  /** 重置表单 */
  resetFields: () => formRef.value?.resetFields(),
});
</script>
