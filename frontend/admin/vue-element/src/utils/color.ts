/**
 * 生成基于字符串的固定随机色（HSL模式，保证饱和度和明度适中）
 * @param str
 */
export const getRandomColor = (str: string) => {
  let hash = 0;
  for (let i = 0; i < str.length; i++) {
    hash = str.charCodeAt(i) + ((hash << 5) - hash);
  }
  const hue = Math.abs(hash % 360);
  return `hsl(${hue}, 50%, 85%)`;
};

/**
 * 根据首字母生成固定随机色
 * @param char
 */
export const getCharColor = (char: string) => {
  let hash = 0;
  for (let i = 0; i < char.length; i++) {
    hash = char.charCodeAt(i) + ((hash << 5) - hash);
  }
  const hue = Math.abs(hash % 360);
  const saturation = 60;
  const lightness = 45;
  return `hsl(${hue}, ${saturation}%, ${lightness}%)`;
};

// 辅助函数：将十六进制颜色转换为 RGB
export function hexToRgb(hex: string): [number, number, number] {
  const bigint = parseInt(hex.slice(1), 16);
  return [(bigint >> 16) & 255, (bigint >> 8) & 255, bigint & 255];
}

// 辅助函数：将 RGB 转换为十六进制颜色
export function rgbToHex(r: number, g: number, b: number): string {
  return `#${((1 << 24) + (r << 16) + (g << 8) + b).toString(16).slice(1)}`;
}
