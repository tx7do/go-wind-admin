import { useCallback, useEffect, useRef, useState } from 'react';

/**
 * 页面级全屏：元素 fullscreen + 进入时补页面底色（fullscreen 元素默认透出 UA 黑底）。
 * 退出（含 ESC）由 fullscreenchange 事件驱动清理内联底色。
 */
export function usePageFullscreen<T extends HTMLElement>() {
  const rootRef = useRef<T | null>(null);
  const [isFullscreen, setIsFullscreen] = useState(false);

  useEffect(() => {
    const sync = () => {
      const active = document.fullscreenElement === rootRef.current;
      setIsFullscreen(active);
      if (!active && rootRef.current) {
        rootRef.current.style.background = '';
      }
    };
    document.addEventListener('fullscreenchange', sync);
    return () => document.removeEventListener('fullscreenchange', sync);
  }, []);

  const toggle = useCallback(() => {
    const el = rootRef.current;
    if (!el) return;
    if (document.fullscreenElement) {
      void document.exitFullscreen?.().catch((error: unknown) =>
        console.error('[fullscreen] exit failed', error),
      );
    } else {
      // 补底色后再进（换主题后取值即当前主题底色）
      el.style.background = getComputedStyle(document.documentElement).backgroundColor;
      void el.requestFullscreen?.().catch((error: unknown) =>
        console.error('[fullscreen] enter failed', error),
      );
    }
  }, []);

  return { rootRef, isFullscreen, toggle };
}
