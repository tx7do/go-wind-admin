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

      <div class="flex-x-between w-full">
        <el-checkbox v-model="loginFormData.rememberMe">
          {{ t("core.login.rememberMe") }}
        </el-checkbox>
        <el-link type="primary" underline="never" @click="toOtherForm('resetPwd')">
          {{ t("core.login.forgetPassword") }}
        </el-link>
      </div>

      <!-- 登录按钮 -->
      <el-form-item>
        <el-button :loading="loading" type="primary" class="w-full" @click="handleLoginSubmit">
          {{ t("core.login.login") }}
        </el-button>
      </el-form-item>
    </el-form>

    <div flex-center gap-10px>
      <el-text size="default">{{ t("core.login.noAccount") }}</el-text>
      <el-link type="primary" underline="never" @click="toOtherForm('register')">
        {{ t("core.login.register") }}
      </el-link>
    </div>
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
// 记住我
const loginFormData = ref<any>({
  username: "admin",
  password: "123456",
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
  };
});

/**
 * 登录提交
 */
async function handleLoginSubmit() {
  // 1. 表单验证
  const valid = await loginFormRef.value?.validate().then(
    () => true,
    () => false
  );
  if (!valid) return;

  loading.value = true;
  try {
    // 2. 执行登录
    await authStore.login(loginFormData.value, async () => {
      // 登录成功，跳转到目标页面
      const redirectPath = (route.query.redirect as string) || "/";
      await router.push(decodeURIComponent(redirectPath));
    });
  } catch {
    // 登录失败处理
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

const emit = defineEmits(["update:modelValue"]);
function toOtherForm(type: "register" | "resetPwd") {
  emit("update:modelValue", type);
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
</style>
