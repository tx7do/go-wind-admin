<template>
  <div>
    <h3 text-center m-0 mb-20px>{{ t("core.resetPwd.resetPassword") }}</h3>
    <el-form ref="formRef" :model="model" :rules="rules" size="large">
      <!-- 用户名"-->
      <el-form-item prop="username">
        <el-input v-model.trim="model.username" :placeholder="t('core.login.username')">
          <template #prefix>
            <el-icon><User /></el-icon>
          </template>
        </el-input>
      </el-form-item>
      <el-form-item>
        <el-button type="warning" class="w-full" @click="submit">
          {{ t("core.resetPwd.resetPassword") }}
        </el-button>
      </el-form-item>
    </el-form>

    <div flex-center gap-10px>
      <el-text size="default">{{ t("core.resetPwd.thinkOfPasswd") }}</el-text>
      <el-link type="primary" underline="never" @click="toLogin">
        {{ t("core.login.login") }}
      </el-link>
    </div>
  </div>
</template>
<script setup lang="ts">
import { useI18n } from "vue-i18n";
import type { FormInstance } from "element-plus";

const { t } = useI18n();

const emit = defineEmits(["update:modelValue"]);
const toLogin = () => emit("update:modelValue", "login");

const model = ref({
  username: "",
});

const rules = computed(() => {
  return {
    username: [
      {
        required: true,
        trigger: "blur",
        message: t("core.login.message.username.required"),
      },
    ],
  };
});

const formRef = ref<FormInstance>();

const submit = async () => {
  await formRef.value?.validate();
  ElMessage.warning(t("common.message.developing"));
};
</script>
