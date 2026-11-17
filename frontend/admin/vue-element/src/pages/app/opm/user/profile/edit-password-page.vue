<template>
  <div class="page-container">
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
          :type="oldPwdVisible ? 'text' : 'password'"
          :placeholder="$t('common.placeholder.input')"
          clearable
        >
          <template #suffix>
            <!-- 自实现眼睛：EP 2.14 的 show-password 空值不渲染图标，见 PasswordToggle -->
            <PasswordToggle v-model="oldPwdVisible" />
          </template>
        </ElInput>
      </ElFormItem>

      <ElFormItem :label="$t('pages.user.form.newPassword')" prop="newPassword">
        <ElInput
          v-model="formData.newPassword"
          :type="newPwdVisible ? 'text' : 'password'"
          :placeholder="$t('common.placeholder.input')"
          clearable
        >
          <template #suffix>
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

    <div class="form-actions">
      <ElButton type="primary" :loading="submitLoading" @click="handleSubmit">
        {{ $t("pages.user.button.updatePassword") }}
      </ElButton>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { reactive, ref } from "vue";
import { ElMessage } from "element-plus";

import { useChangePassword } from "@/api/composables";
import { useAuth } from "@go-wind/admin-element";
import { $t } from "@go-wind/admin-element";

const { mutateAsync: changePassword } = useChangePassword();
const { forceLogout } = useAuth();

const submitLoading = ref(false);
const formRef = ref();

// 表单数据
const formData = reactive({
  oldPassword: "",
  newPassword: "",
  confirmPassword: "",
});

// 各密码字段明文开关（PasswordToggle 眼睛）
const oldPwdVisible = ref(false);
const newPwdVisible = ref(false);
const confirmPwdVisible = ref(false);

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
    // 校验失败由 ElForm 就地标红，字段级错误对象无需再记日志
    return;
  }

  // 检查两次输入的密码是否一致
  if (formData.newPassword !== formData.confirmPassword) {
    ElMessage.error($t("common.notification.password_mismatch"));
    return;
  }

  submitLoading.value = true;

  try {
    await changePassword({ oldPassword: formData.oldPassword, newPassword: formData.newPassword });

    // 后端改密成功即吊销本人全部会话（含当前会话），前端同步 forceLogout
    // （不调已失效的 logout API），由路由守卫重定向到登录页。
    ElMessage.success($t("common.notification.passwordChanged"));
    forceLogout();
    return;
  } catch (error) {
    console.error("Failed to change password:", error);
    ElMessage.error((error as any)?.message || $t("common.notification.updateFailed"));
  } finally {
    submitLoading.value = false;
  }
}
</script>

<style lang="scss" scoped>
.page-container {
  width: 100%;
  max-width: 800px;
}

.profile-form {
  padding-top: 20px;
}

.form-actions {
  padding-top: 20px;
}
</style>
