import type { ReactNode } from 'react';

interface StatsCardProps {
  title: string;
  value: number;
  icon: ReactNode;
  /** 图标底/文字同系语义色（tailwind 色名），保证色块与图标一体 */
  tone?: 'blue' | 'cyan' | 'violet' | 'emerald';
}

const TONE_STYLES: Record<
  NonNullable<StatsCardProps['tone']>,
  { chip: string; glow: string }
> = {
  blue: { chip: 'bg-blue-500/12 text-blue-400', glow: 'from-blue-500/10' },
  cyan: { chip: 'bg-cyan-500/12 text-cyan-400', glow: 'from-cyan-500/10' },
  violet: { chip: 'bg-violet-500/12 text-violet-400', glow: 'from-violet-500/10' },
  emerald: { chip: 'bg-emerald-500/12 text-emerald-400', glow: 'from-emerald-500/10' },
};

/**
 * 统计卡片组件：标题 + 大数值 + 语义色图标块，
 * hover 时边框提亮 + 轻微上浮。
 */
export const StatsCard = ({ title, value, icon, tone = 'blue' }: StatsCardProps) => {
  const toneStyle = TONE_STYLES[tone];
  return (
    <div
      className={
        'group relative h-full overflow-hidden rounded-xl border border-white/8 ' +
        'bg-[color:var(--ant-color-bg-container)] p-5 transition-all duration-200 ' +
        'hover:-translate-y-0.5 hover:border-white/15 hover:shadow-[0_8px_24px_rgba(0,0,0,0.35)]'
      }
    >
      {/* 左上角主色微光，给卡片一点生气 */}
      <div
        className={
          'pointer-events-none absolute -top-10 -left-10 h-28 w-28 rounded-full ' +
          'bg-gradient-to-br to-transparent opacity-60 blur-2xl ' +
          toneStyle.glow
        }
      />
      <div className="relative flex items-start justify-between">
        <div className="flex-1 min-w-0">
          <div className="mb-3 text-sm text-[color:var(--ant-color-text-secondary)]">
            {title}
          </div>
          <div className="text-[28px] leading-none font-semibold tracking-tight text-[color:var(--ant-color-text)] tabular-nums">
            {value.toLocaleString()}
          </div>
        </div>
        <div
          className={
            'flex shrink-0 items-center justify-center w-11 h-11 rounded-lg ml-3 ' +
            toneStyle.chip
          }
        >
          {icon}
        </div>
      </div>
    </div>
  );
};
