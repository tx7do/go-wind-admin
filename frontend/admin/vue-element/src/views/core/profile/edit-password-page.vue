<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ElCard :bordered="false" class="profile-card">
      <template #header>
        <div class="card-header">
          <span>{{ $t("pages.user.profile.tab.editPassword") }}</span>
        </div>
      </template>

      <ElForm
        ref="formRef"
        :model="formData"
        :rules="formRules"
        label-width="120px"
        class="profile-form"
      >
        <ElFormItem :label="$t('pages.user.form.oldPassword')" prop="oldPassword">
          <ElInput
            v-model="formData.oldPassword"
            type="password"
            :placeholder="$t('common.placeholder.input')"
            show-password
            clearable
          />
        </ElFormItem>

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

      <div class="form-actions">
        <ElButton type="primary" :loading="submitLoading" @click="handleSubmit">
          {{ $t("pages.user.button.updatePassword") }}
        </ElButton>
      </div>
    </ElCard>
  </div>
</template>

<script lang="ts" setup>
import { reactive, ref } from "vue";
import { ElMessage } from "element-plus";

import { useUserProfileStore } from "@/stores";
import { $t } from "@/i18n";

const userProfileStore = useUserProfileStore();

const submitLoading = ref(false);
const formRef = ref();

// 表单数据
const formData = reactive({
  oldPassword: "",
  newPassword: "",
  confirmPassword: "",
});

// 表单验证规则
const formRules = {
  oldPassword: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  newPassword: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  confirmPassword: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
};

// 提交表单
async function handleSubmit() {
  if (!formRef.value) return;

  try {
    await formRef.value.validate();
  } catch {
    return;
  }

  // 检查两次输入的密码是否一致
  if (formData.newPassword !== formData.confirmPassword) {
    ElMessage.error($t("common.notification.password_mismatch"));
    return;
  }

  submitLoading.value = true;

  try {
    await userProfileStore.changePassword(formData.oldPassword, formData.newPassword);

    ElMessage.success($t("common.notification.updateSuccess"));

    // 清空表单
    Object.assign(formData, {
      oldPassword: "",
      newPassword: "",
      confirmPassword: "",
    });
    formRef.value?.clearValidate();
  } catch (_error) {
    ElMessage.error($t("common.notification.updateFailed"));
  } finally {
    submitLoading.value = false;
  }
}
</script>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
  width: 100%;
  min-width: 0;
  flex-shrink: 0;
}

.profile-card {
  max-width: 800px;
}

.card-header {
  font-size: 16px;
  font-weight: 500;
}

.profile-form {
  padding-top: 20px;
}

.form-actions {
  padding-top: 20px;
}
</style>
