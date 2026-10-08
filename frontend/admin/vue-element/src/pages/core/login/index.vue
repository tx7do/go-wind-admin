<template>
  <div class="login-layout">
    <!-- 顶部工具栏（对齐 react：仅右上角主题/语言，不放品牌信息） -->
    <div class="login-header">
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
            <template v-if="tenantBranding">
              <img
                v-if="tenantBranding.logoUrl"
                :src="tenantBranding.logoUrl"
                :alt="tenantBranding.name"
                class="tenant-logo"
              />
              <h2 class="form-title">{{ tenantBranding.name }}</h2>
              <p class="form-subtitle">{{ t("core.login.welcomeTitle") }}</p>
            </template>
            <template v-else>
              <h2 class="form-title">
                {{ t("core.login.welcomeTitle") }}
                <span class="wave">👋</span>
              </h2>
              <p class="form-subtitle">{{ t("core.login.welcomeSubtitle") }}</p>
            </template>
          </div>

          <!-- 登录表单卡片（对齐 react：24px 圆角 + 边框 + 主色柔影） -->
          <div class="form-card">
            <transition name="fade-slide" mode="out-in">
              <component
                :is="formComponents[component]"
                class="auth-panel__form"
                @tenant-branding="tenantBranding = $event"
              />
            </transition>
          </div>
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
import ThemeSwitch from "@/components/ThemeSwitch/index.vue";
import SloganIcon from "./icons/slogan.vue";

const { t } = useI18n();

type LayoutMap = "login";

const component = ref<LayoutMap>("login");

// 租户白标：由 Login 子组件在租户编号失焦时经 provide/inject 上报。
// 更简单的做法：白标状态放这里，经 props 传给子组件由其触发回调。
const tenantBranding = ref<{ found: boolean; name: string; logoUrl: string } | null>(null);

const formComponents = {
  login: defineAsyncComponent(() => import("./components/Login.vue")),
};
</script>

<style lang="scss" scoped>
.login-layout {
  display: flex;
  flex-direction: column;
  width: 100%;
  min-height: 100vh;
  background-color: #0b0f19;

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
  justify-content: flex-end;
  padding: 20px 40px;

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
  background: radial-gradient(ellipse at center, #111827 0%, #0b0f19 70%);
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
    background: radial-gradient(circle, rgba(0, 107, 230, 0.08) 0%, transparent 70%);
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
    background: radial-gradient(circle, rgba(0, 107, 230, 0.06) 0%, transparent 70%);
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
    margin-bottom: 36px;

    .slogan-icon {
      width: 360px;
      height: 360px;
      filter: drop-shadow(0 8px 32px rgba(0, 107, 230, 0.3));

      /* 外层不再整体浮动：vben 遗留的 5s 上下 20px 跳动与 SVG 内部
         24s~70s 慢速旋转体系不搭；生命感由 SVG 内光晕呼吸承担。 */

      html:not(.dark) & {
        filter: drop-shadow(0 8px 32px rgba(0, 107, 230, 0.2));
      }
    }
  }

  .brand-info {
    .brand-title {
      font-size: 26px;
      font-weight: 700;
      letter-spacing: 0.5px;
      color: #f8fafc;
      margin: 0 0 14px 0;

      html:not(.dark) & {
        color: #1a1d28;
      }
    }

    .brand-desc {
      font-size: 15px;
      line-height: 1.6;
      color: #9ca3af;
      margin: 0;

      html:not(.dark) & {
        color: #6b7280;
      }
    }
  }
}

// 右侧登录表单（对齐 react UserLayout：面板占 48% 宽、表单内容居中，避免大屏下贴右缘）
.login-form-wrapper {
  width: 48%;
  min-width: 500px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 64px 56px;
  background-color: #111827;
  position: relative;

  html:not(.dark) & {
    background-color: #ffffff;
  }

  .login-form-container {
    width: 100%;
    max-width: 420px;
    flex: 1;
    display: flex;
    flex-direction: column;
    justify-content: center;

    .form-header {
      margin-bottom: 24px;

      .tenant-logo {
  max-height: 48px;
  max-width: 200px;
  object-fit: contain;
  margin-bottom: 12px;
}

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

  // 登录表单卡片（对齐 react：实底表面色 + 24px 大圆角 + 主色柔影）
  .form-card {
    padding: 32px;
    border-radius: 24px;
    background-color: #111827;
    border: 1px solid rgba(255, 255, 255, 0.1);
    box-shadow: 0 12px 40px -8px rgba(0, 107, 230, 0.18);

    html:not(.dark) & {
      background-color: #ffffff;
      border-color: rgba(0, 0, 0, 0.08);
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

  // 暗色模式下的表单样式（对齐设计语言：输入框与表面同层 + 白 α 边框）
  :deep(.el-input__wrapper) {
    background-color: #111827 !important;
    border: 1px solid rgba(255, 255, 255, 0.1) !important;
    box-shadow: none !important;
    transition: all 0.2s ease;

    &:hover {
      border-color: rgba(255, 255, 255, 0.2) !important;
    }

    &.is-focus {
      border-color: #006be6 !important;
      box-shadow:
        0 0 0 1px #006be6 inset,
        0 0 12px rgba(0, 107, 230, 0.15) !important;
    }

    .el-input__inner {
      color: #f8fafc !important;
      font-weight: 400;

      &::placeholder {
        color: #9ca3af !important;
      }
    }
  }

  // 输入框前缀图标颜色
  :deep(.el-input__prefix) {
    .el-icon {
      color: #9ca3af !important;
    }
  }

  // 输入框后缀图标颜色
  :deep(.el-input__suffix) {
    .el-icon {
      color: #9ca3af !important;
    }
  }

  :deep(.el-checkbox__label) {
    color: #f8fafc !important;
    font-weight: 400;
  }

  :deep(.el-checkbox__inner) {
    border-color: rgba(255, 255, 255, 0.25) !important;
    background-color: #111827 !important;
  }

  :deep(.el-checkbox.is-checked .el-checkbox__inner) {
    background-color: #006be6 !important;
    border-color: #006be6 !important;
  }

  :deep(.el-link) {
    color: var(--el-color-primary-light-3) !important;
    font-weight: 500;
    text-decoration: underline;
    text-decoration-color: rgba(0, 107, 230, 0.3);
    text-underline-offset: 2px;

    &:hover {
      color: var(--el-color-primary-light-5) !important;
      text-decoration-color: var(--el-color-primary-light-5);
    }
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
        border-color: var(--el-color-primary) !important;
      }

      &.is-focus {
        border-color: var(--el-color-primary) !important;
        box-shadow: 0 0 0 1px var(--el-color-primary) inset !important;
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
      color: var(--gowind-primary-text) !important;
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
    color: #5a6a80;
    font-weight: 300;

    html:not(.dark) & {
      color: #9ca3af;
    }
  }
}
</style>
