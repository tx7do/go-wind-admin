<script lang="ts" setup>
import type { VbenFormSchema } from '@vben/common-ui';

import { computed, h, onMounted, ref } from 'vue';
import { useRoute } from 'vue-router';

import { AuthenticationLogin, z } from '@vben/common-ui';
import { $t } from '@vben/locales';
import { useAccessStore } from '@vben/stores';

import { router } from '#/router';
import { useAuthStore } from '#/stores';

defineOptions({ name: 'MfaChallenge' });

const authStore = useAuthStore();
const accessStore = useAccessStore();
const route = useRoute();

// 顶部标题与说明（AuthenticationLogin 自带 header 插槽能力有限，用副标题提示）
const subtitle = ref($t('authentication.mfaDesc'));

const formSchema = computed((): VbenFormSchema[] => {
  return [
    {
      component: 'VbenInput',
      componentProps: {
        placeholder: $t('authentication.mfaCodePlaceholder'),
        autocomplete: 'one-time-code',
        inputmode: 'numeric',
        maxlength: 6,
      },
      fieldName: 'totpCode',
      label: $t('authentication.mfaCodeLabel'),
      rules: z
        .string()
        .min(1, { message: $t('authentication.mfaCodeRequired') })
        .regex(/^\d{6}$/, { message: $t('authentication.mfaCodeFormatError') }),
    },
  ];
});

async function handleSubmit(values: Record<string, any>) {
  if (!accessStore.mfaOperationId) {
    await router.push({ name: 'Login' });
    return;
  }
  // 验证成功后回到进入登录流程前的原目标页（由 store 登录分支透传）
  const redirect = (route.query.redirect as string) || '';
  const safeRedirect = redirect.startsWith('/') && !redirect.startsWith('//') ? redirect : '';
  // 有 redirect 才传 onSuccess（覆盖默认 homePath 跳转）；空时走默认跳转
  const result = await authStore.completeMfaChallenge(
    values.totpCode,
    safeRedirect ? async () => { await router.replace(safeRedirect); } : undefined,
  );
  if (!result?.userInfo) {
    // 验证失败：completeMfaChallenge 内部已通知错误并清理状态，回登录页重新走流程
    await router.push({ name: 'Login' });
  }
}

async function handleCancel() {
  accessStore.mfaOperationId = null;
  await router.push({ name: 'Login' });
}

// 取消按钮：通过 footer 插槽注入（AuthenticationLogin 不原生提供取消）
const renderCancel = () =>
  h(
    'button',
    {
      type: 'button',
      onClick: handleCancel,
      style: {
        marginTop: '8px',
        width: '100%',
        padding: '8px 0',
        borderRadius: '6px',
        border: '1px solid hsl(var(--border))',
        background: 'transparent',
        color: 'hsl(var(--muted-foreground))',
        cursor: 'pointer',
      },
    },
    $t('authentication.mfaCancel'),
  );

onMounted(() => {
  // 直接访问挑战页但无待验证状态时，回登录页
  if (!accessStore.mfaOperationId) {
    router.push({ name: 'Login' });
  }
});
</script>

<template>
  <AuthenticationLogin
    :form-schema="formSchema"
    :loading="authStore.loginLoading"
    :show-code-login="false"
    :show-forget-password="false"
    :show-qrcode-login="false"
    :show-register="false"
    :show-remember-me="false"
    :show-third-party-login="false"
    :submit-button-text="$t('authentication.mfaVerify')"
    :sub-title="subtitle"
    :title="$t('authentication.mfaTitle')"
    @submit="handleSubmit"
  >
    <template #to-register>
      <component :is="renderCancel" />
    </template>
  </AuthenticationLogin>
</template>
