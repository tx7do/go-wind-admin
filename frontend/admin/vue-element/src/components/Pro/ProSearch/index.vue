<template>
  <div v-show="visible" class="pro-search">
    <ElForm
      ref="formRef"
      :model="queryParams"
      :inline="inline"
      v-bind="formAttrs"
      :class="formClass"
    >
      <template v-for="(field, index) in fields" :key="field.field">
        <ElFormItem v-show="!isFieldHidden(index)" :label="field.label" :prop="String(field.field)">
          <template #label>
            <span class="flex items-center gap-1">
              {{ field.label }}
              <ElTooltip
                v-if="field.tips"
                :content="typeof field.tips === 'string' ? field.tips : ''"
                placement="top"
              >
                <ElIcon style="--color: var(--el-text-color-secondary)"><QuestionFilled /></ElIcon>
              </ElTooltip>
              <span v-if="colon" class="ml-0.5">:</span>
            </span>
          </template>

          <!-- 自定义插槽 -->
          <slot
            v-if="field.slotName || field.type === 'custom'"
            :name="field.slotName ?? field.field"
            :model="queryParams"
            :field="field.field"
            :attrs="{ style: { width: '100%' }, ...field.attrs }"
          />

          <!-- api-tree-select -->
          <ElTreeSelect
            v-else-if="field.type === 'api-tree-select'"
            v-model="queryParams[field.field]"
            v-bind="{ style: { width: '100%' }, clearable: true, ...field.attrs }"
          />

          <!-- 动态组件 -->
          <component
            :is="getComponent(field.type)"
            v-else
            v-model="queryParams[field.field]"
            v-bind="{ style: { width: '100%' }, clearable: true, ...field.attrs }"
            @keyup.enter="handleSearch"
          >
            <template v-if="['select', 'radio', 'checkbox'].includes(field.type ?? '')">
              <component
                :is="
                  field.type === 'select' ? ElOption : field.type === 'radio' ? ElRadio : ElCheckbox
                "
                v-for="opt in field.options"
                :key="opt.value"
                :label="opt.label"
                :value="opt.value"
                :disabled="opt.disabled"
              />
            </template>
          </component>
        </ElFormItem>
      </template>

      <!-- 操作按钮区域 -->
      <ElFormItem class="pro-search__actions-wrapper">
        <div :class="actionClass">
          <ElButton
            v-if="showSearchButton"
            type="primary"
            :icon="Search"
            :loading="searching"
            @click="handleSearch"
          >
            {{ searchButtonText || t("common.button.search") }}
          </ElButton>

          <ElButton v-if="showResetButton" text :icon="Refresh" @click="handleReset">
            {{ resetButtonText || t("common.button.reset") }}
          </ElButton>

          <!-- 展开/收起 -->
          <span
            v-if="isExpandable && hasHiddenFields"
            class="pro-search__collapse-btn"
            @click="toggleExpand"
          >
            {{ expanded ? t("common.button.collapse") : t("common.button.expand") }}
            <ElIcon class="ml-1">
              <component :is="expanded ? ArrowUp : ArrowDown" />
            </ElIcon>
          </span>
        </div>
      </ElFormItem>
    </ElForm>
  </div>
</template>

<script setup lang="ts" generic="T extends Record<string, any>">
import { computed, ref, reactive, onMounted, nextTick, markRaw, h } from "vue";
import {
  ElForm,
  ElFormItem,
  ElInput,
  ElInputNumber,
  ElSelect,
  ElOption,
  ElRadio,
  ElCheckbox,
  ElCascader,
  ElTreeSelect,
  ElDatePicker,
  ElTimePicker,
  ElTimeSelect,
  ElButton,
  ElIcon,
  ElTooltip,
} from "element-plus";
import { Search, Refresh, ArrowUp, ArrowDown, QuestionFilled } from "@element-plus/icons-vue";
import InputTag from "@/components/InputTag/index.vue";
import { useI18n } from "@go-wind/admin-element";
import type { ProSearchConfig, ProSearchEmits } from "./types";

defineOptions({ inheritAttrs: false });

const props = withDefaults(defineProps<ProSearchConfig<T>>(), {
  colon: false,
  inline: true,
  isExpandable: true,
  showNumber: 3,
  showSearchButton: true,
  showResetButton: true,
  searchButtonText: "",
  resetButtonText: "",
});

const emit = defineEmits<ProSearchEmits<T>>();
const { t } = useI18n();

const formRef = ref<InstanceType<typeof ElForm>>();
const queryParams = reactive<Record<string, any>>({});
const expanded = ref(false);
const searching = ref(false);
const visible = ref(true);

// 判断字段是否应隐藏（收起模式下，超出 showNumber 的字段用 CSS hidden 隐藏）
function isFieldHidden(index: number): boolean {
  if (!props.isExpandable || expanded.value) return false;
  return index >= (props.showNumber ?? 3);
}

// 是否有可展开的隐藏字段
const hasHiddenFields = computed(() => props.fields.length > (props.showNumber ?? 3));

// 表单属性
const formAttrs = computed<Record<string, any>>(() => ({
  labelPosition: "right" as const,
  labelWidth: "auto",
  size: "default",
  ...props.form,
}));

// 按钮区域 class
const actionClass = computed(() => {
  const cls = ["pro-search__actions"];
  if (props.grid) {
    cls.push("pro-search__actions--grid");
  }
  return cls;
});

// 表单 class
const formClass = computed(() => {
  if (props.grid) {
    return "pro-search--grid";
  }
  return "pro-search--inline";
});

// 动态解析组件
const getComponent = (type?: string) => {
  const map: Record<string, any> = {
    input: markRaw(ElInput),
    select: markRaw(ElSelect),
    "input-number": markRaw(ElInputNumber),
    "date-picker": markRaw(ElDatePicker),
    "time-picker": markRaw(ElTimePicker),
    "time-select": markRaw(ElTimeSelect),
    cascader: markRaw(ElCascader),
    "tree-select": markRaw(ElTreeSelect),
    "input-tag": markRaw(InputTag),
    "custom-tag": markRaw(InputTag),
    date: markRaw(ElDatePicker),
    datetime: () => h(ElDatePicker, { type: "datetime" }),
    daterange: () => h(ElDatePicker, { type: "daterange" }),
    number: markRaw(ElInputNumber),
  };
  return map[type ?? "input"] || ElInput;
};
// 搜索
async function handleSearch() {
  try {
    searching.value = true;
    // 过滤空值
    const params = {} as Record<string, any>;
    Object.keys(queryParams).forEach((key) => {
      const val = queryParams[key];
      if (val !== "" && val !== null && val !== undefined) {
        params[key] = val;
      }
    });
    emit("search", params as T);
  } finally {
    searching.value = false;
  }
}

// 重置
function handleReset() {
  formRef.value?.resetFields();
  nextTick(() => formRef.value?.clearValidate());
  // 恢复初始值
  props.fields.forEach((field) => {
    if (field.initialValue !== undefined) {
      (queryParams as any)[field.field] = field.initialValue;
    }
  });
  emit("reset", { ...queryParams } as T);
}

// 展开/收起
function toggleExpand() {
  expanded.value = !expanded.value;
  emit("expand", expanded.value);
}

// 初始化
onMounted(() => {
  props.fields.forEach((field) => {
    if (field.initFn) field.initFn(field as any);
    // api-tree-select: 异步加载数据
    if (field.type === "api-tree-select" && typeof field.api === "function") {
      field.api().then((data) => {
        if (!field.attrs) field.attrs = {};
        field.attrs.data = data;
      });
    }
    // 初始值
    if (["input-tag", "custom-tag", "cascader"].includes(field.type ?? "")) {
      (queryParams as any)[field.field] = Array.isArray(field.initialValue)
        ? field.initialValue
        : [];
    } else if (field.type === "input-number" || field.type === "number") {
      (queryParams as any)[field.field] = field.initialValue ?? null;
    } else {
      (queryParams as any)[field.field] = field.initialValue ?? "";
    }
  });
});

// 暴露方法
defineExpose({
  queryParams,
  formRef,
  visible,
  expanded,
  toggleVisible: () => {
    visible.value = !visible.value;
  },
  toggleExpand,
  reset: handleReset,
  search: handleSearch,
  setQueryParams: (params: Partial<T>) => {
    Object.assign(queryParams, params);
  },
  // 按 field 名重跑该字段的 initFn（重新拉取下拉选项）。
  // initFn 默认只在 onMounted 执行一次；当字段选项依赖外部响应式数据
  // （如租户/组织切换）时，调用方可通过此方法触发重新加载，避免选项过期。
  reloadFieldOptions: (fieldNames: string | string[]) => {
    const names = Array.isArray(fieldNames) ? fieldNames : [fieldNames];
    props.fields.forEach((field) => {
      if (field.initFn && names.includes(field.field as string)) {
        field.initFn(field as any);
      }
    });
  },
});
</script>

<style lang="scss" scoped>
.pro-search {
  /* 四边 20px（§2.2 p-5），与列表卡同一档内缩：EP 的 ElForm 自身不带 padding，
     所以这里只有一个内缩来源（react 端要额外归零 ProComponents 的 24px，本端无此问题）。
     原 `16px 20px` 上下比左右浅 4px，字段行贴卡片上下边、却与列表内容左右对齐，读起来不一致。 */
  padding: 20px;
  background: var(--el-bg-color);
  /* 描边与投影走 §2.2/§2.3 的卡片档（`--gowind-card-*`，定义见 _element-plus.scss 的
   * :root 与 html.dark）：不走 `--el-box-shadow-light`，那是浮层用的 12px 全向晕，
   * 比本端悬浮档还重；静止卡片只需要 1px/2px 的浅落影。骨架卡 .ts-search 是这里的镜像。 */
  border: 1px solid var(--gowind-card-border);
  /* 卡片圆角 12（§2.4 大容器用 LG）：原先 8 与 .el-input__wrapper 的 8 同档，
     卡片不比它装的内容更圆；react/vben 的列表卡实测都是 12/控件 8，本端对齐到同一档。
     骨架卡的 .ts-search 是这里的镜像，两处必须一起改。 */
  border-radius: 12px;
  box-shadow: var(--gowind-card-shadow);
}

// === Grid 布局模式（参考 Vben） ===
// === 流式布局（grid 与 inline 两模式共用，issue #2）===
// grid 模式曾用 auto-fit 等分轨道（列宽下限/隐式列兜底等一整段防溢出调优），
// 但等分轨道对长短不一的查询条件必然出现「宽条件挤不下、窄条件两侧留白」——
// issue #2 建议改自左向右自动分布。flex 流式（原 inline 形态）正是该形态：
// 各项按内容宽落位（min-width 240px 下限）、空间不足自动换行，且窄容器
// （Splitter 栏位）与窄屏（768px/640px 断点）行为已验证。grid 旗标保留为
// 兼容面（38 个页面显式传 grid: true），布局与 inline 归并，折叠逻辑
// （showNumber 索引隐藏）与布局无关，不受影响。
.pro-search--grid,
.pro-search--inline {
  display: flex;
  flex-wrap: wrap;
  gap: 16px 24px;

  // 表单项 - 允许自动换行和拉伸
  :deep(.el-form-item) {
    flex: 1 1 auto;
    min-width: 240px;
    max-width: 100%;
  }

  // 表单项内容区域
  :deep(.el-form-item__content) {
    min-width: 0;
    width: 100%;
  }

  // 输入框宽度自适应
  :deep(.el-input),
  :deep(.el-select),
  :deep(.el-cascader),
  :deep(.el-tree-select),
  :deep(.el-date-editor) {
    width: 100%;
  }

  // 按钮区域包装器 - 保持固定宽度靠右
  :deep(.pro-search__actions-wrapper) {
    flex: 0 0 auto !important;
    margin-left: auto;
    min-width: fit-content;
    width: auto;

    // 当空间不足时，按钮区域换到下一行并占满整行
    @media (max-width: 768px) {
      margin-left: 0;
      width: 100%;
      margin-top: 8px;
    }
  }
}

// 窄屏适配：当容器宽度小于 640px 时，表单项占满整行
@media (max-width: 640px) {
  .pro-search--inline {
    :deep(.el-form-item) {
      width: 100%;
      min-width: unset;
    }
  }
}

// === 展开/收起按钮 ===
.pro-search__collapse-btn {
  display: inline-flex;
  align-items: center;
  cursor: pointer;
  color: var(--gowind-primary-text);
  font-size: 13px;
  user-select: none;
  white-space: nowrap;
  margin-left: 12px;

  &:hover {
    opacity: 0.85;
  }
}

// === 通用重置 ===
:deep(.el-form-item) {
  margin-right: 0;
  margin-bottom: 0;
}

// === 按钮统一规范 ===
:deep(.pro-search__actions) {
  .el-button {
    height: 32px;
    border-radius: 6px;
    padding: 0 16px;
    font-size: 14px;
    line-height: normal;
    transition: all 0.2s ease;
    cursor: pointer;

    // 搜索按钮（primary，非 plain）—— 标准 Element Plus 主色实心主操作。
    // plain+primary 由 Element Plus 内置镂空样式接管，此处用 :not(.is-plain) 排除。
    &.el-button--primary:not(.is-plain) {
      background-color: var(--el-color-primary);
      border-color: var(--el-color-primary);
      color: var(--el-color-white);

      &:hover,
      &:focus {
        background-color: var(--el-color-primary-light-3);
        border-color: var(--el-color-primary-light-3);
        color: var(--el-color-white);
      }

      &:active {
        background-color: var(--el-color-primary-dark-2);
        border-color: var(--el-color-primary-dark-2);
        color: var(--el-color-white);
      }
    }

    // 重置按钮（text）—— 由 Element Plus text 按钮样式接管（透明背景、无边框），
    // 此处仅保留尺寸规范，不覆盖其颜色/背景，确保 text 语义在亮/暗模式下都正确。
    &.is-text {
      height: 32px;
      border-radius: 6px;
      padding: 0 16px;
      font-size: 14px;
      line-height: normal;
    }
  }
}

// 表单项标签样式优化
:deep(.el-form-item__label) {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 120px;
}

:deep(.el-input-number .el-input__inner) {
  text-align: left;
}
</style>
