<template>
  <div class="login-layout">
    <!-- 顶部 Logo 区域 -->
    <div class="login-header">
      <div class="header-left">
        <el-image :src="logo" class="header-logo" fit="contain" />
        <span class="header-title">{{ t("core.login.headerTitle") }}</span>
      </div>
      <div class="header-right">
        <ThemeSwitch class="header-icon" />
        <LangSelect class="header-icon" size="text-20px" />
      </div>
    </div>

    <!-- 主内容区 -->
    <div class="login-content">
      <!-- 左侧品牌展示 -->
      <div class="login-brand">
        <div class="brand-content">
          <div class="brand-illustration">
            <SloganIcon class="slogan-icon" />
          </div>
          <div class="brand-info">
            <h2 class="brand-title">{{ t("core.login.brandTitle") }}</h2>
            <p class="brand-desc">{{ t("core.login.brandDesc") }}</p>
          </div>
        </div>
      </div>

      <!-- 右侧登录表单 -->
      <div class="login-form-wrapper">
        <div class="login-form-container">
          <div class="form-header">
            <h2 class="form-title">
              {{ t("core.login.welcomeTitle") }}
              <span class="wave">👋</span>
            </h2>
            <p class="form-subtitle">{{ t("core.login.welcomeSubtitle") }}</p>
          </div>

          <transition name="fade-slide" mode="out-in">
            <component
              :is="formComponents[component]"
              v-model="component"
              class="auth-panel__form"
            />
          </transition>
        </div>

        <!-- 版权信息 - 放在右侧面板最底部 -->
        <div class="form-copyright">
          <el-text size="small">{{ t("core.login.copyright") }}</el-text>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import logo from "@/assets/images/logo.png";
import ThemeSwitch from "@/components/ThemeSwitch/index.vue";
import SloganIcon from "./icons/slogan.vue";

const { t } = useI18n();

type LayoutMap = "login" | "register" | "resetPwd";

const component = ref<LayoutMap>("login");

const formComponents = {
  login: defineAsyncComponent(() => import("./components/Login.vue")),
  register: defineAsyncComponent(() => import("./components/Register.vue")),
  resetPwd: defineAsyncComponent(() => import("./components/ResetPwd.vue")),
};
</script>

<style lang="scss" scoped>
.login-layout {
  display: flex;
  flex-direction: column;
  width: 100%;
  min-height: 100vh;
  background-color: #f5f7ff;

  html:not(.dark) & {
    background-color: #f5f7ff;
  }
}

// 顶部 Header
.login-header {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  z-index: 10;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 20px 40px;

  .header-left {
    display: flex;
    align-items: center;
    gap: 10px;

    .header-logo {
      width: 24px;
      height: 24px;
    }

    .header-title {
      font-size: 14px;
      font-weight: 600;
      color: #1a1d28;

      html:not(.dark) & {
        color: #1a1d28;
      }
    }
  }

  .header-right {
    display: flex;
    align-items: center;
    gap: 12px;

    .header-icon {
      cursor: pointer;
      transition: opacity 0.3s ease;

      &:hover {
        opacity: 0.7;
      }
    }
  }
}

// 主内容区
.login-content {
  display: flex;
  flex: 1;
  min-height: 0;
}

// 左侧品牌展示
.login-brand {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px;
  background: radial-gradient(ellipse at center, #e8f0ff 0%, #f5f7ff 70%);
  position: relative;
  overflow: hidden;

  html:not(.dark) & {
    background: radial-gradient(ellipse at center, #e8f0ff 0%, #f5f7ff 70%);
  }

  &::before {
    content: "";
    position: absolute;
    top: 30%;
    left: 20%;
    width: 250px;
    height: 250px;
    background: radial-gradient(circle, rgba(64, 158, 255, 0.08) 0%, transparent 70%);
    border-radius: 50%;
    filter: blur(60px);
  }

  &::after {
    content: "";
    position: absolute;
    bottom: 25%;
    right: 15%;
    width: 180px;
    height: 180px;
    background: radial-gradient(circle, rgba(64, 158, 255, 0.06) 0%, transparent 70%);
    border-radius: 50%;
    filter: blur(50px);
  }

  .brand-content {
    display: flex;
    flex-direction: column;
    align-items: center;
    text-align: center;
    position: relative;
    z-index: 1;
  }

  .brand-illustration {
    margin-bottom: 32px;

    .slogan-icon {
      width: 280px;
      height: 280px;
      filter: drop-shadow(0 8px 32px rgba(64, 158, 255, 0.3));

      html:not(.dark) & {
        filter: drop-shadow(0 8px 32px rgba(64, 158, 255, 0.2));
      }
    }
  }

  .brand-info {
    .brand-title {
      font-size: 20px;
      font-weight: 600;
      color: #1a1d28;
      margin: 0 0 8px 0;
    }

    .brand-desc {
      font-size: 13px;
      color: #6b7280;
      margin: 0;
    }
  }
}

// 右侧登录表单
.login-form-wrapper {
  width: 520px;
  min-width: 480px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 40px;
  background-color: #141927;
  position: relative;

  html:not(.dark) & {
    background-color: #ffffff;
  }

  .login-form-container {
    width: 100%;
    max-width: 380px;
    flex: 1;
    display: flex;
    flex-direction: column;
    justify-content: center;

    .form-header {
      margin-bottom: 24px;

      .form-title {
        font-size: 22px;
        font-weight: 600;
        color: #e5eaf3;
        margin: 0 0 8px 0;

        html:not(.dark) & {
          color: #1a1d28;
        }

        .wave {
          display: inline-block;
          animation: wave 2.5s infinite;
          transform-origin: 70% 70%;
        }
      }

      .form-subtitle {
        font-size: 13px;
        color: #6b7a8d;
        margin: 0;

        html:not(.dark) & {
          color: #6b7280;
        }
      }
    }

    .form-section-title {
      font-size: 15px;
      font-weight: 600;
      color: #e5eaf3;
      text-align: center;
      margin: 24px 0 20px 0;

      html:not(.dark) & {
        color: #1a1d28;
      }
    }

    .form-footer {
      margin-top: 24px;
      padding-top: 16px;
      border-top: 1px solid rgba(0, 0, 0, 0.08);
      text-align: center;

      html:not(.dark) & {
        border-top: 1px solid rgba(0, 0, 0, 0.08);
      }

      > .el-text {
        display: block;
        color: #6b7a8d;

        html:not(.dark) & {
          color: #6b7280;
        }
      }
    }
  }
}

@keyframes wave {
  0% {
    transform: rotate(0deg);
  }
  10% {
    transform: rotate(14deg);
  }
  20% {
    transform: rotate(-8deg);
  }
  30% {
    transform: rotate(14deg);
  }
  40% {
    transform: rotate(-4deg);
  }
  50% {
    transform: rotate(10deg);
  }
  60% {
    transform: rotate(0deg);
  }
  100% {
    transform: rotate(0deg);
  }
}

// 响应式
@media (max-width: 768px) {
  .login-content {
    flex-direction: column;
  }

  .login-brand {
    display: none;
  }

  .login-form-wrapper {
    width: 100%;
    min-width: auto;
    padding: 40px 20px;
  }
}

// 覆盖表单样式
.auth-panel__form {
  :deep(.el-form-item) {
    margin-bottom: 16px;
  }

  // 暗色模式下的表单样式
  :deep(.el-input__wrapper) {
    background-color: #1e2435 !important;
    border: 1px solid #3a4560 !important;
    box-shadow: none !important;

    &:hover {
      border-color: #409eff !important;
    }

    &.is-focus {
      border-color: #409eff !important;
      box-shadow: 0 0 0 1px #409eff inset !important;
    }

    .el-input__inner {
      color: #e5eaf3 !important;

      &::placeholder {
        color: #6b7a8d !important;
      }
    }
  }

  // 输入框前缀图标颜色
  :deep(.el-input__prefix) {
    .el-icon {
      color: #4a5568 !important;
    }
  }

  // 输入框后缀图标颜色
  :deep(.el-input__suffix) {
    .el-icon {
      color: #4a5568 !important;
    }
  }

  :deep(.el-checkbox__label) {
    color: #a0aec0 !important;
  }

  :deep(.el-checkbox__inner) {
    border-color: #3a4560 !important;
    background-color: #1e2435 !important;
  }

  :deep(.el-link) {
    color: #409eff !important;
  }
}

// 亮色模式下的表单样式
html:not(.dark) {
  .auth-panel__form {
    :deep(.el-input__wrapper) {
      background-color: #ffffff !important;
      border: 1px solid #c0c4cc !important;
      box-shadow: none !important;

      &:hover {
        border-color: #409eff !important;
      }

      &.is-focus {
        border-color: #409eff !important;
        box-shadow: 0 0 0 1px #409eff inset !important;
      }

      .el-input__inner {
        color: #1a1d28 !important;

        &::placeholder {
          color: #a8abb2 !important;
        }
      }
    }

    // 输入框前缀图标颜色
    :deep(.el-input__prefix) {
      .el-icon {
        color: #c0c4cc !important;
      }
    }

    // 输入框后缀图标颜色
    :deep(.el-input__suffix) {
      .el-icon {
        color: #c0c4cc !important;
      }
    }

    :deep(.el-checkbox__label) {
      color: #6b7280 !important;
    }

    :deep(.el-checkbox__inner) {
      border-color: #c0c4cc !important;
      background-color: #ffffff !important;
    }

    :deep(.el-link) {
      color: #409eff !important;
    }
  }
}

// 版权信息 - 固定在右侧面板底部
.form-copyright {
  position: absolute;
  bottom: 20px;
  left: 0;
  right: 0;
  text-align: center;

  :deep(.el-text) {
    color: #6b7a8d;

    html:not(.dark) & {
      color: #9ca3af;
    }
  }
}
</style>
