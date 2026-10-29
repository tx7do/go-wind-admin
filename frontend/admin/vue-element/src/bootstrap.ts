import { createApp } from "vue";

// ===== 样式导入 =====
import "element-plus/dist/index.css";
import "element-plus/theme-chalk/dark/css-vars.css";
import "vxe-table/lib/style.css";
import "@/styles/index.scss";
import "@/styles/tailwind.css";
import "animate.css";

import { setupDirective } from "@/directives";
import { setupI18n } from "@/core/i18n";
import { router, setupRouter } from "@/router";
import { initStores } from "@/stores/setup";
import { registerGlobComp } from "@/registerGlobComp";
import { initPreferences, preferences } from "@/core/preferences";
import { RequestClient } from "@/core/transport/rest";
import {
  connectSSEServer,
  logoutToLoginPage,
  refreshToken,
  startRefreshTimer,
} from "@/composables/use-token-refresh";
import { useAccessStore } from "@/stores";
import { createI18nGetErrorMsg } from "@/composables/use-request-error-msg";
import { i18n } from "@/core/i18n";

import App from "./App.vue";
import { setupVueQuery } from "@/plugins/vue-query";

async function bootstrap(namespace: string) {
  const app = createApp(App);

  // 抑制 Element Plus ElForm labelWidth="auto" ResizeObserver 的无害警告
  // 该警告在布局过渡期（侧边栏宽度变化、布局切换等）时触发，属于正常现象。
  // 可能报告 width 0（容器未准备好）或 width NaN（中间态的无效计算值），两者均需过滤。
  const originalWarn = console.warn;
  console.warn = (...args: unknown[]) => {
    const msg = args[0];
    if (
      msg &&
      typeof msg === "object" &&
      (String(msg).includes("[ElForm] unexpected width 0") ||
        String(msg).includes("[ElForm] unexpected width NaN"))
    ) {
      return;
    }
    originalWarn.apply(console, args);
  };

  // 初始化偏好设置
  await initPreferences({
    namespace,
    overrides: {
      app: {
        name: import.meta.env.VITE_APP_TITLE || "GoWind Admin",
        version: import.meta.env.VITE_APP_VERSION || "0.0.0",
        enableTenant: import.meta.env.VITE_APP_TENANT_ENABLED === "true",
      },
    },
  });

  // 注册全局组件
  registerGlobComp(app);

  setupVueQuery(app);

  // 注册自定义指令
  setupDirective(app);

  // 配置 pinia-store
  await initStores(app, { namespace });

  // 注入 RequestClient 回调（业务层 → 基础设施层）
  // 必须在 initStores 之后，因为 getToken 依赖 accessStore
  const accessStore = useAccessStore();
  RequestClient.init(import.meta.env.VITE_APP_API_URL, {
    getToken: () => accessStore.accessToken,
    getLocale: () => i18n.global.locale.value,
    refreshToken,
    onReAuthenticate: logoutToLoginPage,
    onError: (msg) => console.error("[RequestClient]", msg),
    getErrorMsg: createI18nGetErrorMsg(),
  });

  // 页面刷新后的会话恢复：
  // access token 仅存内存，刷新页面后丢失。refresh token 以 HttpOnly Cookie 传输，
  // 刷新后仍在。若检测到有效的 refresh_exp cookie，静默调 /refresh-token 换回新 access token。
  // 必须在 setupRouter 之前完成——router 安装即触发首次导航，守卫此刻就要读 accessToken；
  // 若恢复晚于守卫执行，守卫看到 null token 会先把用户踢去登录页，恢复成功也为时已晚。
  if (accessStore.accessToken) {
    // 内存中仍有 access token（如 SPA 内导航），仅需恢复定时器
    startRefreshTimer();
  } else {
    // 内存无 access token（页面刷新），尝试静默恢复
    const refreshExpireAt = getRefreshExpireAt();
    const now = Date.now();
    if (refreshExpireAt && refreshExpireAt > now) {
      try {
        // refresh token cookie 由浏览器自动携带，无需前端传参
        const newToken = await refreshToken();
        if (newToken) {
          startRefreshTimer();
          connectSSEServer();
          console.log("[Bootstrap] session silently restored via refresh cookie");
        }
      } catch (e) {
        // 必须打出真实错误：这里若只打固定话术，refresh 端点 500/网络错等
        // 真因会被"cookie 无效"掩盖（secure-cookie 四层回归的排查教训）
        console.warn("[Bootstrap] 静默恢复失败（refresh cookie 可能无效或过期）:", e);
      }
    }
  }

  // 配置路由及路由守卫（首个导航从这里开始，此刻 token 已就绪）
  setupRouter(app);

  // 国际化 i18n 配置
  // defaultLocale 必须取已持久化的偏好 locale：initPreferences 内部的 handleUpdates
  // 会先按偏好 locale 预载语言包，这里若固定 zh-CN 会把已保存的 en-US 等偏好
  // 强行盖回默认值（实测 en-US 偏好启动后 html.lang 仍为 zh-CN）。
  await setupI18n(app, { defaultLocale: preferences.app.locale });

  // 全局错误观测：渲染函数/守卫抛出的异常若无人接管会静默吞掉，
  // 表现为 router-view 塌空白屏且零控制台输出，极难定位。这里强制落到
  // console.error，保证任何渲染期异常都有迹可循（曾用于排查路由白屏）。
  app.config.errorHandler = (err, instance, info) => {
    console.error(
      "[AppErrorHandler]",
      info,
      err,
      instance?.$options?.name || instance?.$options?.__name
    );
  };

  // 懒加载路由组件加载失败（网络抖动/dev 模块失效）同样默认静默，
  // 记录失败路由与原因，便于与白屏现象对账。
  router.onError((error, to) => {
    console.error("[RouterError]", to?.fullPath, error);

    // vite dev 依赖再优化 / 生产换部署版本后，页面持有的旧 chunk 名会失效，
    // 表现为懒加载路由动态导入失败+白屏，手动刷新即可恢复——这里自动化该恢复。
    // 按"目标路由"做一次性防抖（sessionStorage），chunk 真坏掉时不会无限刷新循环。
    const msg = String((error as Error)?.message ?? error);
    if (/Failed to fetch dynamically imported module|Importing a module script failed/.test(msg)) {
      const key = `router:import-failure-reloaded:${to?.fullPath ?? ""}`;
      if (!sessionStorage.getItem(key)) {
        sessionStorage.setItem(key, "1");
        console.warn("[RouterError] 动态导入失败，整页刷新自愈:", to?.fullPath);
        window.location.reload();
      }
    }
  });

  // vite 对动态导入的预取（preload）失败默认只派发事件不处理，同样自愈刷新。
  // 这是 vite 官方推荐姿势，与上面 router.onError 覆盖两个不同失败阶段。
  window.addEventListener("vite:preloadError", (event) => {
    console.warn("[VitePreloadError] 预取失败，整页刷新自愈:", event);
    window.location.reload();
  });

  // 挂载应用
  app.mount("#app");
}

/**
 * 从 refresh_exp cookie 读取 refresh token 的过期时间戳（Unix 秒）。
 * 返回毫秒级时间戳或 null（cookie 不存在/已过期）。
 */
function getRefreshExpireAt(): number | null {
  const match = document.cookie.split("; ").find((row) => row.startsWith("refresh_exp="));
  if (!match) return null;
  const parts = match.split("=");
  if (parts.length < 2) return null;
  const val = parseInt(parts[1], 10);
  if (!Number.isFinite(val) || val <= 0) return null;
  return val * 1000;
}

export { bootstrap };
