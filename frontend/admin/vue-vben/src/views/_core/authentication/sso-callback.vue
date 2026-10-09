<script lang="ts" setup>
import { onMounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';

import { Page } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { DEFAULT_HOME_PATH, LOGIN_PATH } from '@vben/constants';

import { Spin } from 'ant-design-vue';

import { useAuthStore } from '#/stores';

/**
 * OIDC SSO 回调页：IdP 授权后重定向回 /auth/sso/callback?code=...&state=...，
 * 本页把 code+state 提交后端换本系统 JWT，成功后进首页；失败展示原因并给返回入口。
 */
const route = useRoute();
const router = useRouter();
const authStore = useAuthStore();

const error = ref('');

onMounted(async () => {
  const code = (route.query.code as string) || '';
  const state = (route.query.state as string) || '';

  if (!code || !state) {
    error.value = $t('page.sso.missingParams');
    return;
  }

  try {
    await authStore.completeSsoLogin(code, state, async () => {
      await router.replace(DEFAULT_HOME_PATH);
    });
  } catch (err: any) {
    // 原始错误必须留在控制台：展示给用户的是归一化文案
    console.error('sso callback login failed', err);
    error.value = err?.message || $t('page.sso.failed');
  }
});

</script>

<template>
  <Page>
    <div v-if="error" class="sso-callback-error">
      <p class="error-title">{{ $t('page.sso.failed') }}</p>
      <p class="error-detail">{{ error }}</p>
      <a-button type="primary" @click="router.replace(LOGIN_PATH)">
        {{ $t('page.sso.backToLogin') }}
      </a-button>
    </div>
    <div v-else class="sso-callback-loading">
      <Spin size="large" />
      <p>{{ $t('page.sso.redirecting') }}</p>
    </div>
  </Page>
</template>

<style lang="scss" scoped>
.sso-callback-loading,
.sso-callback-error {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  padding: 80px 0;
}

.error-title {
  font-size: 18px;
  font-weight: 600;
}

.error-detail {
  color: hsl(var(--destructive));
}
</style>
