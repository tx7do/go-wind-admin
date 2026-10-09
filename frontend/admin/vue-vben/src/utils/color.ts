/**
 * 基于字符串稳定分配一个 antd 预设调色板名。
 * 返回的预设名交给 <a-tag :color> 渲染，由 antd 主题 token 驱动，
 * 亮/暗模式自动切换，避免自制 hsl 浅色在暗黑下产生刺眼色块。
 * 同一字符串始终映射到同一颜色，保证区分度稳定。
 */
const PRESET_COLORS = [
  'blue',
  'purple',
  'cyan',
  'green',
  'magenta',
  'pink',
  'red',
  'orange',
  'volcano',
  'geekblue',
  'lime',
  'gold',
] as const;

export type PresetColorName = (typeof PRESET_COLORS)[number];

const hashString = (str: string): number => {
  let hash = 0;
  for (let i = 0; i < str.length; i++) {
    hash = str.charCodeAt(i) + ((hash << 5) - hash);
  }
  return Math.abs(hash);
};

export const getRandomColor = (str: string): PresetColorName => {
  const hue = hashString(str) % PRESET_COLORS.length;
  return PRESET_COLORS[hue]!;
};

/**
 * 根据首字符稳定分配一个 antd 预设调色板名（用于头像占位）。
 * 同样由 antd 主题 token 驱动，亮/暗模式自动切换。
 */
export const getCharColor = (char: string): PresetColorName => {
  const hue = hashString(char) % PRESET_COLORS.length;
  return PRESET_COLORS[hue]!;
};
