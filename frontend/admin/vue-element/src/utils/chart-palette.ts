import { hexToRgb } from "./color";

/**
 * 图表数据色板（docs/design-language.md §4：数据色板 = 主色阶梯 + 语义色）。
 * 与 react 基准端 SourceDonutChart 的扩展调色板保持三端一致：
 * 以主色 #006BE6 为核心的低饱和暗色友好序列（蓝→青→靛→绿→紫→黄→粉→青绿→灰）。
 */
export const CHART_PALETTE = [
  "#006BE6",
  "#22D3EE",
  "#818CF8",
  "#34D399",
  "#A78BFA",
  "#FBBF24",
  "#F472B6",
  "#2DD4BF",
  "#94A3B8",
];

/**
 * 读取运行时主题主色（--el-color-primary，由 preferences 注入），
 * 跟随用户自选主题色联动，对齐 react 端 LineChart 的 token.colorPrimary 用法。
 */
export function getChartPrimary(fallback = "#006BE6"): string {
  if (typeof window === "undefined") return fallback;
  return (
    getComputedStyle(document.documentElement).getPropertyValue("--el-color-primary").trim() ||
    fallback
  );
}

/** hex → rgba() 字符串，用于图表面积渐变等 α 衍生（docs/design-language.md §2.3） */
export function hexToRgba(hex: string, alpha: number): string {
  const [r, g, b] = hexToRgb(hex);
  return `rgba(${r}, ${g}, ${b}, ${alpha})`;
}
