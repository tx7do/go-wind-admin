import { useCallback, useEffect, useRef, useState } from 'react';
import { Modal } from 'antd';
import { useTranslation } from 'react-i18next';
import { usePreferencesStore } from '@go-wind/admin-react';

/**
 * 热更新检测（部署新版本后提示用户刷新页面）。
 * 行为基准是 vben 端同名部件（framework/effects/layouts/widgets/check-updates）：
 * 每 checkUpdatesInterval 分钟对 BASE_URL（index.html）发一次 HEAD 请求，
 * 以 etag / last-modified 响应头作为版本指纹，指纹变化即弹窗提示刷新。
 * 本地开发（localhost/127.0.0.1）不检测；偏好面板「通用 → 定时检查更新」开关门控。
 */
const LOCAL_HOSTNAMES = new Set(['localhost', '127.0.0.1']);

async function fetchVersionTag(url: string): Promise<string | null> {
  if (LOCAL_HOSTNAMES.has(window.location.hostname)) {
    return null;
  }
  try {
    const response = await fetch(url, { cache: 'no-cache', method: 'HEAD' });
    return response.headers.get('etag') || response.headers.get('last-modified');
  } catch (error) {
    // 必须带出原始错误对象：HEAD 失败（断网/代理异常）不该静默
    console.error('[CheckUpdates] fetch version tag failed:', error);
    return null;
  }
}

export const CheckUpdates = () => {
  const { t } = useTranslation(['widgets', 'common']);
  const enableCheckUpdates = usePreferencesStore(
    (state) => state.preferences.app.enableCheckUpdates,
  );
  const checkUpdatesInterval = usePreferencesStore(
    (state) => state.preferences.app.checkUpdatesInterval,
  );

  const [open, setOpen] = useState(false);
  const lastVersionTagRef = useRef('');
  const currentVersionTagRef = useRef('');
  const timerRef = useRef<ReturnType<typeof setInterval> | undefined>(undefined);
  const checkingRef = useRef(false);

  const stop = useCallback(() => {
    if (timerRef.current !== undefined) {
      clearInterval(timerRef.current);
      timerRef.current = undefined;
    }
  }, []);

  const checkForUpdates = useCallback(async () => {
    const versionTag = await fetchVersionTag(import.meta.env.BASE_URL || '/');
    if (!versionTag) {
      return;
    }
    // 首次运行只记基线，不提示更新
    if (!lastVersionTagRef.current) {
      lastVersionTagRef.current = versionTag;
      return;
    }
    if (lastVersionTagRef.current !== versionTag) {
      stop();
      currentVersionTagRef.current = versionTag;
      setOpen(true);
    }
  }, [stop]);

  const start = useCallback(() => {
    stop();
    // 间隔 <= 0 视为关闭轮询
    if (checkUpdatesInterval <= 0) {
      return;
    }
    timerRef.current = setInterval(() => {
      void checkForUpdates();
    }, checkUpdatesInterval * 60 * 1000);
  }, [checkUpdatesInterval, checkForUpdates, stop]);

  // 开关门控 + 生命周期；页面隐藏时暂停、重新可见时立即补查一次
  // （vben 同款语义：页签在后台时不白发请求，回到前台先查再续轮询）
  useEffect(() => {
    if (!enableCheckUpdates) {
      stop();
      setOpen(false);
      return;
    }
    start();

    const handleVisibilityChange = () => {
      if (document.hidden) {
        stop();
        return;
      }
      if (!checkingRef.current) {
        checkingRef.current = true;
        checkForUpdates()
          .finally(() => {
            checkingRef.current = false;
            start();
          });
      }
    };

    document.addEventListener('visibilitychange', handleVisibilityChange);
    return () => {
      stop();
      document.removeEventListener('visibilitychange', handleVisibilityChange);
    };
  }, [enableCheckUpdates, start, stop, checkForUpdates]);

  return (
    <Modal
      open={open}
      title={t('checkUpdatesTitle')}
      // 与 vben 一致：更新提示不可经 X / ESC / 遮罩绕过，只允许「取消」或「刷新」
      closable={false}
      keyboard={false}
      maskClosable={false}
      cancelText={t('button.cancel', { ns: 'common' })}
      okText={t('refresh', { ns: 'common' })}
      onOk={() => {
        // 记下已确认的指纹再刷新：万一刷新被浏览器策略打断，也不会立即再弹
        lastVersionTagRef.current = currentVersionTagRef.current;
        window.location.reload();
      }}
      onCancel={() => {
        // 取消后轮询保持停止，回到页签（visibilitychange）时会再次检测并重弹
        setOpen(false);
      }}
    >
      {t('checkUpdatesDescription')}
    </Modal>
  );
};
