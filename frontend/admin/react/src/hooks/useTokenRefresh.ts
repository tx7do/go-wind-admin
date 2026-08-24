/**
 * Token 定时刷新管理
 * 主动定时刷新 access_token，避免请求时才被动刷新导致延迟
 * 对齐 Vue 版 use-token-refresh.ts
 */
import { fetchUserProfile } from '@/api/hooks/user-profile';
import { globalSSEClient } from '@/core/transport/sse';
import { useAuthStore } from '@/stores/auth';

// ==============================
// 常量
// ==============================

/** Access Token 刷新间隔（1.5 小时） */
const ACCESS_TOKEN_REFRESH_INTERVAL = 90 * 60 * 1000;

/** 在 access token 到期前多久开始刷新 */
const SAFETY_BUFFER_MS = 5 * 60 * 1000;
/** 最小刷新间隔（避免立即重试风暴） */
const MIN_INTERVAL_MS = 3 * 1000;

// ==============================
// 模块级状态（单例）
// ==============================

let refreshTimer: ReturnType<typeof setTimeout> | null = null;

// ==============================
// Cookie 读取辅助
// ==============================

/**
 * 从 refresh_exp cookie 读取 refresh token 的过期时间戳（Unix 秒）。
 * refresh_exp 为非 HttpOnly cookie，仅含过期时间戳，无敏感信息。
 * 由后端 setRefreshCookies 在登录/刷新时写入。
 * 返回毫秒级时间戳或 null（cookie 不存在/已过期）。
 */
function getRefreshExpireAt(): number | null {
  const match = document.cookie
    .split('; ')
    .find((row) => row.startsWith('refresh_exp='));
  if (!match) return null;
  const parts = match.split('=');
  if (parts.length < 2) return null;
  const val = parseInt(parts[1], 10);
  if (!Number.isFinite(val) || val <= 0) return null;
  return val * 1000;
}

// ==============================
// 核心：计算下次刷新间隔
// ==============================

function computeNextInterval(): number {
  const { accessTokenExpireAt } = useAuthStore.getState();
  const now = Date.now();

  // refresh token 快过期 → 快速重试（使其在过期前尽快刷新）
  const refreshExpireAt = getRefreshExpireAt();
  const refreshRemaining = (refreshExpireAt ?? 0) - now;
  if (refreshExpireAt && refreshRemaining <= SAFETY_BUFFER_MS) {
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
      const now = Date.now();

      // refresh token 不存在或已过期 → 强制登出
      // refresh token 过期时间从 refresh_exp cookie 读取
      const refreshExpireAt = getRefreshExpireAt();
      if (!refreshExpireAt) {
        useAuthStore.getState().forceLogout();
        return;
      }

      // refresh token 快过期 → 强制登出
      if (refreshExpireAt - now <= SAFETY_BUFFER_MS) {
        useAuthStore.getState().forceLogout();
        return;
      }

      // 执行刷新
      const newToken = await useAuthStore.getState().refreshToken();

      // token 刷新成功后，使用新 token 重连 SSE
      if (newToken) {
        // react 无刷新后 userInfo 拉取路径（AuthGuard 只查 token），刷新页面后 userInfo 为 null。
        // SSE streamID 已改为 userId，无 userInfo 则无法重连，故在首次 tick 时惰性补拉一次。
        // 运行于此处可保证 queryClient 已就绪（mount 后），与 login 内 fetchUserProfile 同源。
        if (useAuthStore.getState().userInfo === null) {
          try {
            const profile = (await fetchUserProfile()) as unknown as Parameters<
              ReturnType<typeof useAuthStore.getState>['setUserInfo']
            >[0];
            if (profile) {
              useAuthStore.getState().setUserInfo(profile);
            }
          } catch (e) {
            console.error('[TokenRefresh] lazy fetch user profile failed:', e);
          }
        }
        reconnectSSEServer();
      }
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
 * 重连 SSE 服务器（使用最新的 access token）
 * 在 token 刷新成功后调用，确保 SSE 连接携带新凭证
 */
export function reconnectSSEServer(): void {
  const { accessToken, userInfo } = useAuthStore.getState();

  // streamID 已改为 userId，鉴权仍走 Authorization 头。两者缺一不可。
  if (!accessToken || userInfo?.id == null) {
    console.warn('[TokenRefresh] No access token or userId, skip SSE reconnect');
    return;
  }

  // 仅在当前已有 SSE 连接时才重连，避免无连接时意外建立连接
  const currentStatus = globalSSEClient.getStatus();
  if (currentStatus === 'disconnected') {
    console.log('[TokenRefresh] SSE is disconnected, skip reconnect');
    return;
  }

  const sseUrl = `${import.meta.env.VITE_SSE_URL ?? '/api/sse'}?stream=${userInfo.id}`;
  globalSSEClient.setHeaders({ Authorization: `Bearer ${accessToken}` });
  globalSSEClient.reconnect(sseUrl);
  console.log('[TokenRefresh] SSE reconnected with new token');
}

/**
 * 连接 SSE 服务器
 * 使用当前 access token 建立 SSE 连接
 */
export function connectSSEServer(): void {
  const { accessToken, userInfo } = useAuthStore.getState();

  // streamID 已改为 userId，鉴权仍走 Authorization 头。两者缺一不可。
  if (!accessToken || userInfo?.id == null) {
    console.warn('[TokenRefresh] No access token or userId, skip SSE connection');
    return;
  }

  const sseUrl = `${import.meta.env.VITE_SSE_URL ?? '/api/sse'}?stream=${userInfo.id}`;
  globalSSEClient.setHeaders({ Authorization: `Bearer ${accessToken}` });
  globalSSEClient.connect(sseUrl);
}

/**
 * 关闭 SSE 连接
 */
export function disconnectSSEServer(): void {
  globalSSEClient.close();
}
