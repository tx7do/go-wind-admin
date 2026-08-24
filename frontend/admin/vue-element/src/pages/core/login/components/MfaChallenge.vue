<template>
  <div class="mfa-panel-form">
    <h3 class="mfa-panel-form__title">{{ t("core.login.mfaTitle") }}</h3>
    <p class="mfa-panel-form__desc">{{ t("core.login.mfaDesc") }}</p>
    <el-form ref="mfaFormRef" :model="mfaForm" :rules="mfaRules" size="large" @submit.prevent>
      <el-form-item prop="code">
        <el-input
          v-model.trim="mfaForm.code"
          :placeholder="t('core.login.mfaCodePlaceholder')"
          inputmode="numeric"
          maxlength="6"
          @keyup.enter="handleSubmit"
        >
          <template #prefix>
            <el-icon><Key /></el-icon>
          </template>
        </el-input>
      </el-form-item>
      <el-form-item>
        <el-button type="primary" class="w-full" :loading="loginLoading" @click="handleSubmit">
          {{ t("core.login.mfaVerify") }}
        </el-button>
      </el-form-item>
      <el-form-item>
        <el-button class="w-full" @click="cancelMfa">
          {{ t("core.login.mfaCancel") }}
        </el-button>
      </el-form-item>
    </el-form>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { ElNotification } from "element-plus";
import { useRoute, useRouter } from "vue-router";
import { useAuth } from "@/composables/use-auth";
import { i18n } from "@/core/i18n";
import { useAccessStore } from "@/stores";

const t = i18n.global.t;
const router = useRouter();
const route = useRoute();
const { completeMfaChallenge, loginLoading } = useAuth();
const accessStore = useAccessStore();

const mfaFormRef = ref();
const mfaForm = ref({ code: "" });
const mfaRules = {
  code: [
    { required: true, message: t("core.login.mfaCodeRequired"), trigger: "blur" },
    {
      pattern: /^\d{6}$/,
      message: t("core.login.mfaCodeFormatError"),
      trigger: "blur",
    },
  ],
};

const handleSubmit = async () => {
  if (!accessStore.mfaOperationId) {
    ElNotification({
      title: t("core.authentication.loginFailed"),
      message: t("core.login.mfaNotRequired"),
      type: "error",
    });
    await router.push({ name: "Login" });
    return;
  }
  // 在发起 MFA 验证请求前，当前路由必然是 MFA 挑战页，此时读取 redirect 最可靠。
  // 验证请求返回后到 onSuccess 之间，路由可能因 token 写入等副作用先行更新，
  // 导致 onSuccess 内再读 route.query.redirect 时拿不到值，回退默认首页跳转。
  // 故在此处同步捕获并解码（含同源校验与畸形输入兜底），闭包传入 onSuccess。
  const rawRedirect = (route.query.redirect as string) || "";
  let decodedRedirect = "";
  try {
    decodedRedirect = decodeURIComponent(rawRedirect);
  } catch {
    decodedRedirect = "";
  }
  const safeRedirect =
    typeof decodedRedirect === "string" &&
    decodedRedirect.startsWith("/") &&
    !decodedRedirect.startsWith("//")
      ? decodedRedirect
      : "";
  await mfaFormRef.value?.validate(async (valid: boolean) => {
    if (!valid) return;
    // 有 redirect 才传 onSuccess（覆盖默认 homePath 跳转）；空时走默认跳转
    // 使用上面捕获的 safeRedirect，避免响应式更新时序导致读不到。
    const result = await completeMfaChallenge(
      mfaForm.value.code,
      safeRedirect
        ? async () => {
            await router.replace(safeRedirect);
          }
        : undefined
    );
    if (!result?.userInfo) {
      ElNotification({
        title: t("core.authentication.loginFailed"),
        message: t("core.login.mfaVerifyFailed"),
        type: "error",
      });
    }
  });
};

const cancelMfa = async () => {
  accessStore.mfaOperationId = null;
  await router.push({ name: "Login" });
};
</script>

<style scoped>
.mfa-panel-form {
  width: 100%;
}
.mfa-panel-form__title {
  text-align: center;
  font-size: 20px;
  font-weight: 600;
  margin-bottom: 8px;
}
.mfa-panel-form__desc {
  text-align: center;
  font-size: 13px;
  color: var(--el-text-color-secondary);
  margin-bottom: 24px;
}
.w-full {
  width: 100%;
}
</style>
