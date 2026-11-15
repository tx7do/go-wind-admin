<template>
  <div class="sso-callback-page">
    <ElResult v-if="error" status="error" :title="t('core.sso.failed')" :sub-title="error">
      <template #extra>
        <ElButton type="primary" @click="router.replace('/login')">
          {{ t("core.sso.backToLogin") }}
        </ElButton>
      </template>
    </ElResult>
    <div v-else class="sso-callback-loading">
      <ElIcon class="is-loading" :size="36"><Loading /></ElIcon>
      <p>{{ t("core.sso.redirecting") }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { ElButton, ElIcon, ElMessage, ElResult } from "element-plus";
import { Loading } from "@element-plus/icons-vue";
import { useAuth } from "@go-wind/admin-element";
import { useI18n } from "@go-wind/admin-element";
import { DEFAULT_HOME_PATH } from "@/constants";

/**
 * OIDC SSO 回调页：IdP 授权后重定向回 /auth/sso/callback?code=...&state=...，
 * 本页把 code+state 提交后端换本系统 JWT，成功后进首页；失败展示原因并给返回入口。
 */
const route = useRoute();
const router = useRouter();
const { completeSsoLogin } = useAuth();
const { t } = useI18n();

const error = ref("");

onMounted(async () => {
  const code = (route.query.code as string) || "";
  const state = (route.query.state as string) || "";

  if (!code || !state) {
    error.value = t("core.sso.missingParams");
    return;
  }

  try {
    await completeSsoLogin(code, state, async () => {
      await router.replace(DEFAULT_HOME_PATH);
    });
  } catch (err: any) {
    // 原始错误必须留在控制台：展示给用户的是归一化文案
    console.error("sso callback login failed", err);
    error.value = err?.message || t("core.sso.failed");
    ElMessage.error(error.value);
  }
});
</script>

<style lang="scss" scoped>
.sso-callback-page {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  padding: 24px;
}

.sso-callback-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  color: var(--el-text-color-secondary);
}
</style>
