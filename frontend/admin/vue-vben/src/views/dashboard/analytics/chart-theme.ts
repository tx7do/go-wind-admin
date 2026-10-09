/**
 * ECharts 系列色辅助：从 <html> 上的主题 CSS 变量读取色值并包裹为 hsl() 串。
 * 这些变量（--primary / --success / --warning / --destructive）由 vben 偏好系统
 * 在主题切换时更新，因此返回值随亮/暗模式自动变化，避免硬编码 hex 在暗黑下不协调。
 */
const THEME_VARS = [
  '--primary',
  '--success',
  '--warning',
  '--destructive',
] as const;

const readHsl = (varName: string): string => {
  const root = document.documentElement;
  const raw = getComputedStyle(root).getPropertyValue(varName).trim();
  if (!raw) {
    return 'transparent';
  }
  // 变量值为 "H S% L%" 形式（无 hsl() 包裹），需手动包裹为合法 CSS 颜色。
  // 这里必须输出逗号分隔形式 hsl(H, S%, L%)：浏览器 canvas 虽然两种分隔都解析，
  // 但 ECharts emphasis 态会对颜色调用 zrender 的 parse/liftColor，该解析器
  // 只认逗号分隔的 hsl()，遇到空格分隔会误判参数不足而回退成黑色，导致
  // 饼图/玫瑰图 hover 时被 hover 的扇区变黑（视觉上像消失）。
  // 将空格分隔转换为逗号分隔，使静态态与 emphasis 态颜色一致。
  const parts = raw.split(/\s+/);
  if (parts.length !== 3) {
    return 'transparent';
  }
  return `hsl(${parts[0]}, ${parts[1]}, ${parts[2]})`;
};

/** 返回主题感知的系列色数组，用于 ECharts 多系列区分。 */
export const getSeriesColors = (): string[] =>
  THEME_VARS.map(readHsl);

/** 返回主题感知的单一强调色（用于单系列 / axisPointer 等）。 */
export const getAccentColor = (index: number): string => {
  const colors = getSeriesColors();
  return colors[index % colors.length] ?? 'transparent';
};
