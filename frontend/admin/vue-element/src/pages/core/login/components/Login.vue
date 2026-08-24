<template>
  <div class="auth-panel-form">
    <!-- 隐藏“登录”标题，使用主页面的“欢迎回来”作为主标题 -->
    <h3 class="auth-panel-form__title" text-center style="display: none">
      {{ t("core.login.login") }}
    </h3>
    <el-form
      ref="loginFormRef"
      :model="loginFormData"
      :rules="loginRules"
      size="large"
      :validate-on-rule-change="false"
    >
      <!-- 租户编号（可选，留空为平台登录） -->
      <el-form-item prop="tenant_code">
        <el-input v-model.trim="loginFormData.tenant_code" :placeholder="t('core.login.tenantCode')">
          <template #prefix>
            <el-icon><User /></el-icon>
          </template>
        </el-input>
      </el-form-item>

      <!-- 用户名 -->
      <el-form-item prop="username">
        <el-input v-model.trim="loginFormData.username" :placeholder="t('core.login.username')">
          <template #prefix>
            <el-icon><User /></el-icon>
          </template>
        </el-input>
      </el-form-item>

      <!-- 密码 -->
      <el-tooltip :visible="isCapsLock" :content="t('core.login.capsLock')" placement="right">
        <el-form-item prop="password">
          <el-input
            v-model.trim="loginFormData.password"
            :placeholder="t('core.login.password')"
            type="password"
            show-password
            @keyup="checkCapsLock"
            @keyup.enter="handleLoginSubmit"
          >
            <template #prefix>
              <el-icon><Lock /></el-icon>
            </template>
          </el-input>
        </el-form-item>
      </el-tooltip>

      <!-- 验证码 -->
      <el-form-item prop="captchaCode">
        <div class="captcha-row">
          <el-input
            v-model.trim="loginFormData.captchaCode"
            :placeholder="t('core.login.captchaCode')"
            @keyup.enter="handleLoginSubmit"
          >
            <template #prefix>
              <el-icon><Key /></el-icon>
            </template>
          </el-input>
          <div
            class="captcha-image"
            :title="t('core.login.captchaRefresh')"
            @click="refreshCaptcha"
          >
            <img v-if="captchaImage" :src="captchaImage" alt="captcha" />
            <span v-else class="captcha-image__placeholder">{{ t("core.login.captchaRefresh") }}</span>
          </div>
        </div>
      </el-form-item>

      <div class="w-full">
        <el-checkbox v-model="loginFormData.rememberMe">
          {{ t("core.login.rememberMe") }}
        </el-checkbox>
      </div>

      <!-- 登录按钮 -->
      <el-form-item>
        <el-button :loading="loading" type="primary" class="w-full" @click="handleLoginSubmit">
          {{ t("core.login.login") }}
        </el-button>
      </el-form-item>
    </el-form>
  </div>
</template>
<script setup lang="ts">
import type { FormInstance } from "element-plus";
import { useAuth } from "@/composables/use-auth";
import { router } from "@/router";

const { t } = useI18n();
const authStore = useAuth();
const route = useRoute();

const loginFormRef = ref<FormInstance>();
const loading = ref(false);
// 是否大写锁定
const isCapsLock = ref(false);
// 验证码状态
const captchaId = ref("");
const captchaImage = ref("");
// 记住我
const loginFormData = ref<any>({
  tenant_code: "",
  username: "",
  password: "",
  captchaCode: "",
  // 模板中 v-model="loginFormData.rememberMe" 会读写该字段，
  // 显式初始化为 false 以保证响应式稳定（避免运行时动态新增属性）。
  rememberMe: false,
});

const loginRules = computed(() => {
  return {
    username: [
      {
        required: true,
        trigger: "blur",
        message: t("core.login.message.username.required"),
      },
    ],
    password: [
      {
        required: true,
        trigger: "blur",
        message: t("core.login.message.password.required"),
      },
      {
        min: 5,
        message: t("core.login.message.password.min"),
        trigger: "blur",
      },
    ],
    captchaCode: [
      {
        required: true,
        trigger: "blur",
        message: t("core.login.message.captchaCode.required"),
      },
    ],
  };
});

// 获取验证码
async function refreshCaptcha() {
  try {
    const resp = await authStore.getCaptcha();
    captchaId.value = resp?.captchaId ?? "";
    captchaImage.value = resp?.imageBase64 ?? "";
  } catch {
    // 验证码获取失败不阻断页面
  }
}

onMounted(() => {
  refreshCaptcha();
});

/**
 * 登录提交
 */
async function handleLoginSubmit() {
  // 函数入口即同步捕获 redirect：此时当前路由必然是登录页，读取最可靠。
  // 登录请求返回后到 onSuccess 之间存在响应式更新时序竞态（route 与
  // currentRoute.value 可能先行更新、丢掉 query），onSuccess 内再读会拿空，
  // safePath 回退 "/" → 跳首页（即「登录成功不跳目标页、F5 才跳」的根因）。
  const rawRedirect = (route.query.redirect as string) || "/";

  // 1. 表单验证
  const valid = await loginFormRef.value?.validate().then(
    () => true,
    () => false
  );
  if (!valid) return;

  loading.value = true;
  try {
    // 2. 执行登录（把验证码 id 和用户输入一并传入）
    const result = await authStore.login(
      {
        ...loginFormData.value,
        captchaId: captchaId.value,
      },
      async () => {
        // 登录成功，跳转到目标页面
        // 校验 redirect 必须为同源相对路径，防止开放重定向（如 ?redirect=https://evil.com 或 //evil.com）
        // 使用入口捕获的 rawRedirect；decode 对畸形输入（如裸 %）抛 URIError，需兜底
        let decodedPath = "/";
        try {
          decodedPath = decodeURIComponent(rawRedirect);
        } catch {
          decodedPath = "/";
        }
        const safePath =
          typeof decodedPath === "string" &&
          decodedPath.startsWith("/") &&
          !decodedPath.startsWith("//")
            ? decodedPath
            : "/";
        await router.push(safePath);
      }
    );
    // authStore.login 永不抛错：失败（含验证码错误、MFA 闸门）以 userInfo 缺失表达，据此刷新验证码。
    if (!result?.userInfo) {
      refreshCaptcha();
    }
  } finally {
    loading.value = false;
  }
}

// 检查输入大小写
function checkCapsLock(event: KeyboardEvent) {
  // 防止浏览器密码自动填充时报错
  if (event instanceof KeyboardEvent) {
    isCapsLock.value = event.getModifierState("CapsLock");
  }
}
</script>

<style lang="scss" scoped>
.auth-panel-form {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.auth-panel-form__title {
  margin: 0 0 0.5rem;
  font-size: 1.125rem;
  font-weight: 600;
  color: #8b9dc3;

  html:not(.dark) & {
    color: #1a1d28;
  }
}

.captcha-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  width: 100%;
}

.captcha-image {
  flex-shrink: 0;
  width: 120px;
  height: 40px;
  border-radius: 6px;
  overflow: hidden;
  border: 1px solid var(--el-border-color);
  background: var(--el-fill-color-light);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  &__placeholder {
    color: var(--el-text-color-placeholder);
    font-size: 12px;
  }
}
</style>
