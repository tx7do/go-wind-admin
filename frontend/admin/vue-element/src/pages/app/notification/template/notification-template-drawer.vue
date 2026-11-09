<template>
  <ProModal
    v-model:visible="visible"
    :title="title"
    :config="{ component: 'drawer', drawer: { size: DRAWER_WIDTH } }"
    :close-guard="guard.allowClose"
  >
    <ElForm
      ref="formRef"
      :model="formData"
      :rules="formRules"
      label-width="120px"
      class="drawer-form"
    >
      <ElFormItem :label="t('pages.notification_template.name')" prop="name">
        <ElInput
          v-model="formData.name"
          :placeholder="t('common.placeholder.input')"
          maxlength="100"
          clearable
        />
      </ElFormItem>

      <ElFormItem :label="t('pages.notification_template.code')" prop="code">
        <!-- 编辑态 code 不可改：它是发送方的引用锚，改码等于让所有引用悬空 -->
        <ElInput
          v-model="formData.code"
          :disabled="!isCreate"
          :placeholder="t('pages.notification_template.codePattern')"
          maxlength="64"
          clearable
        />
        <div class="field-tip">{{ t("pages.notification_template.codeHint") }}</div>
      </ElFormItem>

      <ElFormItem :label="t('pages.notification_template.titleTemplate')" prop="titleTemplate">
        <ElInput
          v-model="formData.titleTemplate"
          :placeholder="t('pages.notification_template.titleTemplatePlaceholder')"
          maxlength="500"
          clearable
        />
      </ElFormItem>

      <ElFormItem :label="t('pages.notification_template.contentTemplate')" prop="contentTemplate">
        <ElInput
          v-model="formData.contentTemplate"
          type="textarea"
          :rows="6"
          maxlength="20000"
          show-word-limit
          :placeholder="t('pages.notification_template.contentTemplatePlaceholder')"
        />
      </ElFormItem>

      <ElFormItem :label="t('pages.notification_template.isEnabled')">
        <ElSwitch v-model="formData.isEnabled" />
      </ElFormItem>

      <ElFormItem :label="t('pages.notification_template.remark')" prop="remark">
        <ElInput
          v-model="formData.remark"
          type="textarea"
          :rows="2"
          maxlength="500"
          :placeholder="t('common.placeholder.input')"
        />
      </ElFormItem>
    </ElForm>
    <template #footer>
      <ElButton @click="guard.requestClose(handleClose)">{{ $t("common.button.cancel") }}</ElButton>
      <ElButton type="primary" :loading="submitLoading" @click="handleSubmit">
        {{ $t("common.button.confirm") }}
      </ElButton>
    </template>
  </ProModal>
</template>

<script lang="ts" setup>
import { computed, reactive, ref } from "vue";
import {
  ElButton,
  ElForm,
  ElFormItem,
  ElInput,
  ElMessage,
  ElSwitch,
} from "element-plus";

import ProModal from "@/components/Pro/ProModal/index.vue";
import { useDrawerCloseGuard } from "@/components/Pro/composables/use-drawer-close-guard";
import type {
  notificationservicev1_NotificationTemplate,
} from "@/api/generated/admin/service/v1";
import { useI18n } from "@/core/i18n";
import { DRAWER_WIDTH } from "@/constants";
import {
  NOTIFICATION_TEMPLATE_UPDATE_MASK,
  useCreateNotificationTemplate,
  useUpdateNotificationTemplate,
} from "@/api/composables";

const emit = defineEmits<{
  success: [];
}>();

const { t } = useI18n();

const { mutateAsync: createTemplate } = useCreateNotificationTemplate();
const { mutateAsync: updateTemplate } = useUpdateNotificationTemplate();

const visible = ref(false);
const submitLoading = ref(false);
const isCreate = ref(true);
const currentId = ref<string>();
const formRef = ref();

const formData = reactive({
  name: "",
  code: "",
  titleTemplate: "",
  contentTemplate: "",
  isEnabled: true,
  remark: "",
});

const formRules = {
  name: [{ required: true, message: t("pages.notification_template.requiredName"), trigger: "blur" }],
  code: [
    { required: true, message: t("pages.notification_template.requiredCode"), trigger: "blur" },
    {
      pattern: /^[A-Za-z0-9_-]{1,64}$/,
      message: t("pages.notification_template.codePattern"),
      trigger: "blur",
    },
  ],
  titleTemplate: [
    { required: true, message: t("pages.notification_template.requiredTitleTemplate"), trigger: "blur" },
  ],
  contentTemplate: [
    { required: true, message: t("pages.notification_template.requiredContentTemplate"), trigger: "blur" },
  ],
};

const title = computed(() =>
  isCreate.value ? t("pages.notification_template.create") : t("pages.notification_template.edit")
);

function resetForm() {
  formData.name = "";
  formData.code = "";
  formData.titleTemplate = "";
  formData.contentTemplate = "";
  formData.isEnabled = true;
  formData.remark = "";
  formRef.value?.clearValidate();
}

// 关闭守卫：遮罩/ESC/X/取消统一，脏数据先确认（docs/form_drawer_close_guard.md）
const guard = useDrawerCloseGuard(() => ({ ...formData }));

function open(options: { create: boolean; row?: notificationservicev1_NotificationTemplate }) {
  visible.value = true;
  isCreate.value = options.create;
  currentId.value = options.row?.id;
  resetForm();

  if (!options.create && options.row) {
    formData.name = options.row.name ?? "";
    formData.code = options.row.code ?? "";
    formData.titleTemplate = options.row.titleTemplate ?? "";
    formData.contentTemplate = options.row.contentTemplate ?? "";
    formData.isEnabled = !!options.row.isEnabled;
    formData.remark = options.row.remark ?? "";
  }

  // 填充完成后记录基线快照（回填不算脏）
  guard.syncBaseline();
}

function handleClose() {
  visible.value = false;
  resetForm();
}

async function handleSubmit() {
  if (!formRef.value) return;

  const valid = await formRef.value.validate().then(
    () => true,
    () => false
  );
  if (!valid) return;

  const data: notificationservicev1_NotificationTemplate = {
    name: formData.name,
    code: formData.code || undefined,
    titleTemplate: formData.titleTemplate,
    contentTemplate: formData.contentTemplate,
    isEnabled: !!formData.isEnabled,
    remark: formData.remark,
  };

  try {
    submitLoading.value = true;
    if (isCreate.value) {
      await createTemplate({ data });
      ElMessage.success(t("pages.notification_template.createSuccess"));
    } else if (currentId.value !== undefined) {
      // code 不进掩码：引用锚不可改，服务端同样按掩码裁剪
      await updateTemplate({ id: currentId.value, data, updateMask: NOTIFICATION_TEMPLATE_UPDATE_MASK });
      ElMessage.success(t("pages.notification_template.updateSuccess"));
    }
    emit("success");
    handleClose();
  } catch (error: any) {
    // 原始错误必须留在控制台：用户可见的那句翻译不包含服务端的原因
    console.error("save notification template failed", error);
    ElMessage.error(error?.message || t("pages.notification_template.saveFailed"));
  } finally {
    submitLoading.value = false;
  }
}

defineExpose({ open });
</script>

<style lang="scss" scoped>
.drawer-form {
  padding-right: 10px;
}

.field-tip {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
  margin-top: 4px;
}
</style>
