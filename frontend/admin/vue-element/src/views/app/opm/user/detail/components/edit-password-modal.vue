<template>
  <ElDialog
    v-model="visible"
    :title="$t('pages.user.button.editPassword')"
    width="500px"
    :close-on-click-modal="false"
    :destroy-on-close="true"
    @close="handleClose"
  >
    <ElForm
      ref="formRef"
      :model="formData"
      :rules="formRules"
      label-width="120px"
      class="modal-form"
    >
      <ElFormItem :label="$t('pages.user.form.newPassword')" prop="newPassword">
        <ElInput
          v-model="formData.newPassword"
          type="password"
          :placeholder="$t('common.placeholder.input')"
          show-password
          strength="strong"
          clearable
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.user.form.confirmPassword')" prop="confirmPassword">
        <ElInput
          v-model="formData.confirmPassword"
          type="password"
          :placeholder="$t('common.placeholder.input')"
          show-password
          clearable
        />
      </ElFormItem>
    </ElForm>

    <template #footer>
      <div class="dialog-footer">
        <ElButton @click="handleClose">{{ $t("common.button.cancel") }}</ElButton>
        <ElButton type="primary" :loading="submitLoading" @click="handleSubmit">
          {{ $t("common.button.confirm") }}
        </ElButton>
      </div>
    </template>
  </ElDialog>
</template>

<script lang="ts" setup>
import { computed, reactive, ref } from "vue";
import { ElMessage } from "element-plus";

import { useEditUserPassword } from "@/api/composables";
import { $t } from "@/i18n";

const { mutateAsync: editUserPassword } = useEditUserPassword();

const visible = ref(false);
const submitLoading = ref(false);
const formRef = ref();
const userId = ref<number>();

// 表单数据
const formData = reactive({
  newPassword: "",
  confirmPassword: "",
});

// 表单验证规则
const formRules = computed(() => ({
  newPassword: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  confirmPassword: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
}));

// 重置表单
function resetForm() {
  Object.assign(formData, {
    newPassword: "",
    confirmPassword: "",
  });
  formRef.value?.clearValidate();
}

// 打开弹窗
function open(data: { userId: number }) {
  visible.value = true;
  userId.value = data.userId;

  // 重置表单
  resetForm();
}

// 关闭弹窗
function handleClose() {
  visible.value = false;
  resetForm();
}

// 提交表单
async function handleSubmit() {
  if (!formRef.value) return;

  try {
    await formRef.value.validate();

    // 检查两次输入是否一致
    if (formData.newPassword !== formData.confirmPassword) {
      ElMessage.error($t("common.notification.password_mismatch"));
      return;
    }

    submitLoading.value = true;

    await editUserPassword({ userId: userId.value!, newPassword: formData.newPassword });

    ElMessage.success($t("common.notification.updateSuccess"));
    handleClose();
  } catch (error) {
    if (error !== false) {
      ElMessage.error($t("common.notification.updateFailed"));
    }
  } finally {
    submitLoading.value = false;
  }
}

// 暴露方法
defineExpose({
  open,
});
</script>

<style lang="scss" scoped>
.modal-form {
  padding-right: 12px;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>
