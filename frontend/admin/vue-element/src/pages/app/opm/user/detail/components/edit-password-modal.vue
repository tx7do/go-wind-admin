<template>
  <ElDrawer
    v-model="visible"
    :title="$t('pages.user.button.editPassword')"
    width="520px"
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
          :type="newPwdVisible ? 'text' : 'password'"
          :placeholder="$t('common.placeholder.input')"
          clearable
        >
          <template #suffix>
            <!-- 自实现眼睛：EP 2.14 的 show-password 空值不渲染图标，见 PasswordToggle -->
            <PasswordToggle v-model="newPwdVisible" />
          </template>
        </ElInput>
      </ElFormItem>

      <ElFormItem :label="$t('pages.user.form.confirmPassword')" prop="confirmPassword">
        <ElInput
          v-model="formData.confirmPassword"
          :type="confirmPwdVisible ? 'text' : 'password'"
          :placeholder="$t('common.placeholder.input')"
          clearable
        >
          <template #suffix>
            <PasswordToggle v-model="confirmPwdVisible" />
          </template>
        </ElInput>
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
  </ElDrawer>
</template>

<script lang="ts" setup>
import { computed, reactive, ref } from "vue";
import { ElMessage } from "element-plus";

import { useEditUserPassword } from "@/api/composables";
import { $t } from "@go-wind/admin-element";

const { mutateAsync: editUserPassword } = useEditUserPassword();

const emit = defineEmits<{
  success: [];
}>();

const visible = ref(false);
const submitLoading = ref(false);
const formRef = ref();
const userId = ref<string>();

// 表单数据
const formData = reactive({
  newPassword: "",
  confirmPassword: "",
});

// 各密码字段明文开关（PasswordToggle 眼睛）
const newPwdVisible = ref(false);
const confirmPwdVisible = ref(false);

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
  newPwdVisible.value = false;
  confirmPwdVisible.value = false;
  formRef.value?.clearValidate();
}

// 打开弹窗
function open(data: { userId: string }) {
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
    emit("success");
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
