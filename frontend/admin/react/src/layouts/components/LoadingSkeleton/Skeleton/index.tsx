import { useMemo } from 'react';
import { Skeleton as AntdSkeleton } from 'antd';

import { useSkeletonAnimation } from '../hooks/useSkeletonAnimation';
import type { SkeletonBlockProps, SkeletonProps } from './types';
import { useI18n } from '@/core/i18n';
import './styles.css';

/**
 * 基础骨架块组件
 * 支持：自定义尺寸/形状/动画 + Ant Design 主题适配
 */
export const SkeletonBlock = ({
  width = '100%',
  height = '1em',
  shape = 'round',
  animation = 'wave',
  className = '',
  style,
  children,
}: SkeletonProps) => {
  const { visible, animationClass } = useSkeletonAnimation({
    active: true,
    animation,
  });

  // 计算样式
  const blockStyle = useMemo(() => {
    return {
      display: 'inline-block',
      width,
      height,
      borderRadius: shape === 'circle' ? '50%' : shape === 'round' ? '4px' : '0',
      backgroundColor: 'var(--ant-color-fill-tertiary, rgba(0,0,0,0.06))',
      ...style,
    };
  }, [width, height, shape, style]);

  if (!visible) {
    return children ? <>{children}</> : null;
  }

  return (
    <span
      className={`
        skeleton-block
        ${animationClass}
        ${shape === 'circle' ? 'skeleton-circle' : ''}
        ${className}
      `.trim()}
      style={blockStyle}
      aria-hidden="true"
    >
      {children}
    </span>
  );
};

/**
 * 复合骨架组件（支持多块配置 + 预设模板）
 */
export const Skeleton = ({
  active = true,
  delay = 0,
  blocks,
  preset,
  rows = 3,
  rowWidths,
  width,
  height,
  shape = 'round',
  animation = 'wave',
  className = '',
  style,
  children,
}: SkeletonProps) => {
  const { visible, animationClass } = useSkeletonAnimation({
    active,
    animation,
    delay,
  });
  const { t } = useI18n('common');

  // 生成预设块
  const presetBlocks = useMemo((): Array<Partial<SkeletonBlockProps>> => {
    if (!preset) return [];

    switch (preset) {
      case 'paragraph':
        return Array.from({ length: rows }).map((_, i) => ({
          width: rowWidths?.[i] || (i === rows - 1 ? '60%' : '100%'),
          height: '1em',
          shape: 'round' as const,
        }));

      case 'avatar':
        return [{ width: 40, height: 40, shape: 'circle' as const }];

      case 'button':
        return [{ width: 80, height: 32, shape: 'round' as const }];

      case 'input':
        return [{ width: '100%', height: 32, shape: 'round' as const }];

      case 'image':
        return [{ width: '100%', height: 120, shape: 'square' as const }];

      default:
        return [];
    }
  }, [preset, rows, rowWidths]);

  // 合并配置
  const allBlocks = (blocks || presetBlocks || [{ width, height, shape }]) as SkeletonBlockProps[];

  if (!visible) {
    return children ? <>{children}</> : null;
  }

  return (
    <div
      className={`
        skeleton-container
        ${animationClass}
        ${className}
      `.trim()}
      style={style}
      role="status"
      aria-label={t('skeleton.loading')}
    >
      {allBlocks.map((block, index) => (
        <SkeletonBlock
          key={index}
          {...block}
          animation={animation}
          className={block.className}
          style={block.style}
        />
      ))}
      {children}
    </div>
  );
};

// 兼容 Ant Design Skeleton API（可选）
export const SkeletonCompat = (props: any) => {
  if (props.avatar || props.title || props.paragraph) {
    return <AntdSkeleton {...props} />;
  }
  return <Skeleton {...props} />;
};

export default Skeleton;
