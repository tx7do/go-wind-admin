/**
 * Token 定时刷新管理
 * 主动定时刷新 access_token，避免请求时才被动刷新导致延迟
 * 对齐 Vue 版 use-token-refresh.ts
 */
import { globalSSEClient } from '@/core/transport/sse';
import { useAuthStore } from '@/stores/auth';

// ==============================
// 常量
// ==============================

/** Access Token 刷新间隔（1.5 小时） */
const ACCESS_TOKEN_REFRESH_INTERVAL = 90 * 60 * 1000;
/** Refresh Token 刷新间隔（12 小时） */
// const REFRESH_TOKEN_REFRESH_INTERVAL = 12 * 60 * 60 * 1000;

/** 在 access token 到期前多久开始刷新 */
const SAFETY_BUFFER_MS = 5 * 60 * 1000;
/** 最小刷新间隔（避免立即重试风暴） */
const MIN_INTERVAL_MS = 3 * 1000;

// ==============================
// 模块级状态（单例）
// ==============================

let refreshTimer: ReturnType<typeof setTimeout> | null = null;

// ==============================
// 核心：计算下次刷新间隔
// ==============================

function computeNextInterval(): number {
  const { accessTokenExpireAt, refreshTokenExpireAt } = useAuthStore.getState();
  const now = Date.now();

  // refresh token 快过期 → 快速重试
  const refreshRemaining = (refreshTokenExpireAt ?? 0) - now;
  if (refreshTokenExpireAt && refreshRemaining <= SAFETY_BUFFER_MS) {
    return MIN_INTERVAL_MS;
  }

  // access token 已过期或没有 → 快速重试
  const accessRemaining = (accessTokenExpireAt ?? 0) - now;
  if (!accessTokenExpireAt || accessRemaining <= 0) {
    return MIN_INTERVAL_MS;
  }

  // access token 快过期 → 快速重试
  if (accessRemaining <= SAFETY_BUFFER_MS) {
    return MIN_INTERVAL_MS;
  }

  // 正常情况：在过期前提前刷新（80% 剩余时间，不超过默认间隔）
  return Math.floor(
    Math.max(
      MIN_INTERVAL_MS,
      Math.min(ACCESS_TOKEN_REFRESH_INTERVAL, (accessRemaining - SAFETY_BUFFER_MS) * 0.8),
    ),
  );
}

// ==============================
// 核心：定时刷新调度
// ==============================

function scheduleRefresh(): void {
  stopRefreshTimer();

  const tick = async () => {
    try {
      const { refreshTokenValue, refreshTokenExpireAt } = useAuthStore.getState();
      const now = Date.now();

      // 没有 refresh token → 强制登出
      if (!refreshTokenValue) {
        useAuthStore.getState().forceLogout();
        return;
      }

      // refresh token 快过期 → 强制登出
      if (refreshTokenExpireAt && refreshTokenExpireAt - now <= SAFETY_BUFFER_MS) {
        useAuthStore.getState().forceLogout();
        return;
      }

      // 执行刷新
      await useAuthStore.getState().refreshToken();
    } catch (error) {
      console.error('[TokenRefresh] 定时刷新失败:', error);
    } finally {
      // 无论成功失败，只要定时器还在，就安排下一次
      if (refreshTimer !== null) {
        refreshTimer = globalThis.setTimeout(tick, computeNextInterval());
      }
    }
  };

  refreshTimer = globalThis.setTimeout(tick, computeNextInterval());
}

// ==============================
// 公开 API
// ==============================

/**
 * 启动定时刷新
 * 在登录成功后调用
 */
export function startRefreshTimer(): void {
  scheduleRefresh();
}

/**
 * 停止定时刷新
 * 在登出、token 失效时调用
 */
export function stopRefreshTimer(): void {
  if (refreshTimer !== null) {
    globalThis.clearTimeout(refreshTimer);
    refreshTimer = null;
  }
}

/**
 * 连接 SSE 服务器
 * 使用当前 access token 建立 SSE 连接
 */
export function connectSSEServer(): void {
  const { accessToken } = useAuthStore.getState();

  if (!accessToken) {
    console.warn('[TokenRefresh] No access token, skip SSE connection');
    return;
  }

  const sseUrl = `${import.meta.env.VITE_APP_SSE_URL ?? '/api/sse'}?stream=${encodeURIComponent(accessToken)}`;
  globalSSEClient.setHeaders({ Authorization: `Bearer ${accessToken}` });
  globalSSEClient.connect(sseUrl);
}

/**
 * 关闭 SSE 连接
 */
export function disconnectSSEServer(): void {
  globalSSEClient.close();
}
