import React, { useMemo } from 'react';
import { usePreferences } from '@go-wind/admin-react';

import type { AuthPageLayoutType } from '@go-wind/admin-react';

import './background.less';

interface BackgroundProps {
  layout: AuthPageLayoutType;
}

/**
 * 认证页面背景装饰组件
 * 根据不同的 authPageLayout 模式展示不同风格的背景
 *
 * - panel-center: 全屏背景 + 多层光斑 + 中心面板
 * - panel-left: 左侧品牌区渐变 + 右侧面板
 * - panel-right: 右侧品牌区渐变 + 左侧面板
 */
const Background: React.FC<BackgroundProps> = ({ layout }) => {
  const { isDark } = usePreferences();

  const modeClass = isDark ? 'dark' : 'light';

  // 根据布局模式决定装饰元素的位置和可见性
  const layoutConfig = useMemo(() => {
    switch (layout) {
      case 'panel-center':
        return {
          showBrandArea: false,
          showFullscreenDecor: true,
          brandPosition: null,
        };
      case 'panel-left':
        return {
          showBrandArea: true,
          showFullscreenDecor: false,
          brandPosition: 'left',
        };
      case 'panel-right':
        return {
          showBrandArea: true,
          showFullscreenDecor: false,
          brandPosition: 'right',
        };
      default:
        return {
          showBrandArea: false,
          showFullscreenDecor: true,
          brandPosition: null,
        };
    }
  }, [layout]);

  return (
    <div className={`auth-background ${modeClass}`}>
      {/* ===== 全屏背景装饰（panel-center 模式） ===== */}
      {layoutConfig.showFullscreenDecor && (
        <>
          {/* 主背景渐变层 */}
          <div className={`auth-bg-gradient ${modeClass}`} />

          {/* 大光斑 - 右上 */}
          <div className={`auth-bg-orb auth-bg-orb-1 ${modeClass}`} />

          {/* 中光斑 - 左下 */}
          <div className={`auth-bg-orb auth-bg-orb-2 ${modeClass}`} />

          {/* 小光斑 - 中心偏左 */}
          <div className={`auth-bg-orb auth-bg-orb-3 ${modeClass}`} />

          {/* 网格纹理 */}
          <div className={`auth-bg-grid ${modeClass}`} />
        </>
      )}

      {/* ===== 品牌区背景装饰（panel-left / panel-right 模式） ===== */}
      {layoutConfig.showBrandArea && (
        <div className={`auth-bg-brand-area ${layoutConfig.brandPosition} ${modeClass}`}>
          {/* 品牌区渐变层 */}
          <div className={`auth-bg-brand-gradient ${modeClass}`} />

          {/* 装饰光斑 */}
          <div className={`auth-bg-orb auth-bg-orb-brand-1 ${modeClass}`} />
          <div className={`auth-bg-orb auth-bg-orb-brand-2 ${modeClass}`} />

          {/* 品牌区网格纹理 */}
          <div className={`auth-bg-grid ${modeClass}`} />
        </div>
      )}
    </div>
  );
};

export default Background;
