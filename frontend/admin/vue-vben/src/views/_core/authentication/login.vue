<script lang="ts" setup>
import type { VbenFormSchema } from '@vben/common-ui';

import { computed, h, onMounted, ref } from 'vue';

import { AuthenticationLogin, z } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { message } from 'ant-design-vue';

import { useAuthStore } from '#/stores';
import { fetchGenerateCaptcha } from '#/api/composables';
import { ssoEnabled, startSsoLogin } from '#/api/composables/sso';
import { fetchTenantBranding } from '#/api/composables/tenant-branding';

defineOptions({ name: 'Login' });

const authStore = useAuthStore();

// 验证码状态
const captchaId = ref('');
const captchaImage = ref('');
const captchaLoading = ref(false);

// SSO 开关（后端未配置 OIDC 时按钮不渲染）
const ssoOn = ref(false);

// 租户白标：租户编号输入失焦后拉取，命中替换欢迎标题
const tenantBranding = ref<{ found: boolean; name: string; logoUrl: string } | null>(null);

async function applyTenantBranding(event: any) {
  const trimmed = (event?.target?.value ?? '').trim();
  if (!trimmed) {
    tenantBranding.value = null;
    return;
  }
  try {
    const branding = await fetchTenantBranding(trimmed);
    tenantBranding.value = branding.found ? branding : null;
  } catch (brandingError) {
    // 白标是非关键路径：失败静默保持默认品牌，原始错误留控制台
    console.warn('fetch tenant branding failed', brandingError);
    tenantBranding.value = null;
  }
}

async function refreshCaptcha() {
  captchaLoading.value = true;
  try {
    const resp = await fetchGenerateCaptcha();
    captchaId.value = resp.captchaId ?? '';
    captchaImage.value = resp.imageBase64 ?? '';
  } catch {
    // 验证码获取失败不阻断页面
  } finally {
    captchaLoading.value = false;
  }
}

onMounted(() => {
  refreshCaptcha();
  // SSO 开关探测：失败/未配置都不显示按钮（非关键路径，静默降级）
  ssoEnabled()
    .then((v) => (ssoOn.value = v))
    .catch(() => (ssoOn.value = false));
});

async function handleSsoLogin() {
  try {
    await startSsoLogin();
  } catch (err: any) {
    // 原始错误必须留在控制台：展示给用户的是归一化文案
    console.error('start sso login failed', err);
    message.error(err?.message || $t('page.sso.failed'));
  }
}

// 验证码图片渲染函数（响应式读取 captchaImage / captchaLoading）
// 作为函数式组件传入 suffix，由 VbenRenderContent 通过 h() 渲染
const renderCaptchaImage = () =>
  h(
    'div',
    {
      title: $t('authentication.captchaRefresh'),
      onClick: () => {
        if (!captchaLoading.value) refreshCaptcha();
      },
      style: {
        height: '36px',
        width: '110px',
        flexShrink: '0',
        cursor: 'pointer',
        borderRadius: '6px',
        overflow: 'hidden',
        border: '1px solid hsl(var(--border))',
        background: 'hsl(var(--input))',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
      },
    },
    captchaImage.value
      ? [
          h('img', {
            src: captchaImage.value,
            alt: 'captcha',
            style: {
              height: '100%',
              width: '100%',
              objectFit: 'cover',
            },
          }),
        ]
      : [
          h(
            'span',
            { style: { color: 'var(--muted-foreground)', fontSize: '12px' } },
            captchaLoading.value ? '...' : $t('authentication.captchaRefresh'),
          ),
        ],
  );

const formSchema = computed((): VbenFormSchema[] => {
  return [
    {
      component: 'VbenInput',
      componentProps: {
        placeholder: $t('authentication.tenantCode'),
        autocomplete: 'off',
        onBlur: applyTenantBranding,
      },
      fieldName: 'tenant_code',
      label: $t('authentication.tenantCode'),
      rules: z.optional(z.string()),
    },
    {
      component: 'VbenInput',
      componentProps: {
        placeholder: $t('authentication.usernameTip'),
        autocomplete: 'username',
      },
      dependencies: {
        trigger(values) {
          if (values.selectAccount) {
          }
        },
        triggerFields: ['selectAccount'],
      },
      fieldName: 'username',
      label: $t('authentication.username'),
      rules: z.string().min(1, { message: $t('authentication.usernameTip') }),
    },
    {
      component: 'VbenInputPassword',
      componentProps: {
        placeholder: $t('authentication.password'),
        autocomplete: 'current-password',
      },
      fieldName: 'password',
      label: $t('authentication.password'),
      rules: z.string().min(1, { message: $t('authentication.passwordTip') }),
    },
    {
      component: 'VbenInput',
      componentProps: {
        placeholder: $t('authentication.captchaTip'),
        autocomplete: 'off',
        class: 'w-auto flex-1 min-w-0',
      },
      fieldName: 'captchaValue',
      label: $t('authentication.captcha'),
      rules: z.string().min(1, { message: $t('authentication.captchaTip') }),
      suffix: renderCaptchaImage,
    },
  ];
});

// 包装 authLogin：提交时把 captchaId 一并传入
async function handleSubmit(values: Record<string, any>) {
  const result = await authStore.authLogin({
    ...values,
    captchaId: captchaId.value,
  });
  // 登录失败时刷新验证码
  if (!result?.userInfo) {
    refreshCaptcha();
  }
}
</script>

<template>
  <!-- 对齐 react：标题/描述在卡片外，登录表单包一张 24px 圆角卡片（边框 + 主色柔影） -->
  <div>
    <div class="mb-7">
      <template v-if="tenantBranding">
        <img
          v-if="tenantBranding.logoUrl"
          :src="tenantBranding.logoUrl"
          :alt="tenantBranding.name"
          class="mb-3 max-h-12 max-w-[200px] object-contain"
        />
        <h2
          class="text-foreground mb-3 text-3xl font-bold leading-9 tracking-tight lg:text-4xl"
        >
          {{ tenantBranding.name }}
        </h2>
        <p class="text-muted-foreground text-sm lg:text-md">
          {{ $t('authentication.welcomeBack') }}
        </p>
      </template>
      <template v-else>
        <h2
          class="text-foreground mb-3 text-3xl font-bold leading-9 tracking-tight lg:text-4xl"
        >
          {{ $t('authentication.welcomeBack') }} 👋🏻
        </h2>
        <p class="text-muted-foreground text-sm lg:text-md">
          {{ $t('authentication.loginSubtitle') }}
        </p>
      </template>
    </div>

    <div
      class="rounded-3xl border border-border bg-card p-8 shadow-[0_12px_40px_-8px_rgba(0,107,230,0.18)]"
    >
      <AuthenticationLogin
        :form-schema="formSchema"
        :loading="authStore.loginLoading"
        :show-code-login="false"
        :show-forget-password="false"
        :show-qrcode-login="false"
        :show-register="false"
        :show-third-party-login="false"
        @submit="handleSubmit"
      >
        <!-- 内置标题已移到卡片外，置空默认标题块 -->
        <template #title><span class="hidden"></span></template>
      </AuthenticationLogin>

      <template v-if="ssoOn">
        <div class="my-4 flex items-center gap-3">
          <div class="h-px flex-1 bg-border"></div>
          <span class="text-muted-foreground text-xs">{{ $t('page.sso.or') }}</span>
          <div class="h-px flex-1 bg-border"></div>
        </div>
        <a-button block size="large" @click="handleSsoLogin">
          {{ $t('page.sso.button') }}
        </a-button>
      </template>
    </div>
  </div>
</template>
