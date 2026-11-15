<template>
  <component
    :is="containerComponent"
    v-model="visible"
    v-bind="containerProps"
    :before-close="handleContainerBeforeClose"
    @close="handleClose"
  >
    <!-- 内容区域（支持 v-loading） -->
    <div v-loading="loading">
      <!-- 自定义内容模式：default slot 替代 ProForm -->
      <slot v-if="$slots.default" />

      <!-- ProForm 配置化模式 -->
      <ProForm
        v-else
        ref="formRef"
        :model-value="formData as any"
        :fields="fields as any"
        :disabled="mode === 'view'"
        :colon="config.colon"
        v-bind="formProps"
      >
        <template v-for="(_, name) in $slots" :key="name" #[name]="slotProps">
          <slot :name="name" v-bind="slotProps" />
        </template>
      </ProForm>
    </div>

    <template #footer>
      <slot name="footer">
        <ElButton @click="handleClose">{{ t("common.button.cancel") }}</ElButton>
        <ElButton v-if="mode !== 'view'" type="primary" :loading="submitting" @click="handleSubmit">
          {{ t("common.button.confirm") }}
        </ElButton>
      </slot>
    </template>
  </component>
</template>

<script setup lang="ts" generic="T extends Record<string, any>">
import { computed, ref } from "vue";
import { ElButton, ElDrawer, ElDialog, ElMessage } from "element-plus";
import { useI18n } from "@go-wind/admin-element";
import ProForm from "../ProForm/index.vue";
import type { ProModalConfig, ModalMode } from "./types";

defineOptions({ inheritAttrs: false });

const props = withDefaults(
  defineProps<{
    visible: boolean;
    mode?: ModalMode;
    config: ProModalConfig<T>;
    formData?: T;
    /** 内容区域 loading 状态（异步加载数据时使用） */
    loading?: boolean;
    /** 弹窗标题（优先级高于 config.drawer/dialog 中的 title） */
    title?: string;
    /** 关闭守卫（docs/form_drawer_close_guard.md）：返回 true 才允许关闭；
     *  遮罩/ESC/X/底部取消四条路径统一走它；提交成功路径不经守卫 */
    closeGuard?: () => boolean | Promise<boolean>;
  }>(),
  { mode: "add", loading: false }
);
const emit = defineEmits<{
  "update:visible": [boolean];
  submit: [];
}>();

const { t } = useI18n();

const visible = computed({
  get: () => props.visible,
  set: (v) => emit("update:visible", v),
});
const formRef = ref<any>(null);
const containerProps = computed(() => {
  const base =
    props.config.component === "drawer"
      ? { destroyOnClose: true, appendToBody: true, lockScroll: false, ...props.config.drawer }
      : {
          destroyOnClose: true,
          alignCenter: true,
          appendToBody: true,
          lockScroll: false,
          ...props.config.dialog,
        };
  // title prop 优先级最高
  if (props.title) {
    return { ...base, title: props.title };
  }
  return base;
});
const formProps = computed(() => ({
  labelWidth: "auto",
  ...props.config.form,
}));
const fields = computed(() => props.config.fields ?? []);

const containerComponent = computed(() =>
  props.config.component === "drawer" ? ElDrawer : ElDialog
);
const submitting = ref(false);

function doClose() {
  visible.value = false;
  formRef.value?.resetFields();
}

/** 底部取消按钮的关闭尝试：走守卫。visible 已为 false 时直接放行——
 *  EP 的 close 事件在关闭开始后触发，会再进到这里，去重防二次弹确认。 */
async function attemptClose() {
  if (!props.visible) return;
  const allow = props.closeGuard ? await props.closeGuard() : true;
  if (allow) doClose();
}

/** 遮罩 / ESC / 标题栏 X 的统一入口（EP before-close）：守卫放行后才真正关 */
async function handleContainerBeforeClose(done: () => void) {
  if (!props.visible) {
    done();
    return;
  }
  const allow = props.closeGuard ? await props.closeGuard() : true;
  if (allow) {
    doClose();
    done();
  }
}

function handleClose() {
  void attemptClose();
}

async function handleSubmit() {
  if (props.mode === "view") return;
  // 二段式校验：validate 失败时 reject 的是字段错误对象（非 false），
  // 若直接放 try/catch 用 `error !== false` 判断会恒为 true，导致校验失败
  // （如必填项为空）也误弹「操作失败」。这里先单独校验，失败即返回。
  const valid = await formRef.value?.validate().then(
    () => true,
    () => false
  );
  if (!valid) return;

  try {
    submitting.value = true;
    if (typeof props.config.beforeSubmit === "function") {
      props.config.beforeSubmit(props.formData!);
    }
    await props.config.submitAction?.(props.formData!);
    emit("submit");
    visible.value = false;
  } catch {
    // 仅提交动作失败会进入此分支（校验失败已在上方提前 return）
    ElMessage.error(t("common.operationFailed"));
  } finally {
    submitting.value = false;
  }
}

defineExpose({ formRef });
</script>
