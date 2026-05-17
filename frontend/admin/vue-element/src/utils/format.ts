/**
 * 数据格式化相关工具函数
 */

/**
 * 格式化增长率
 * 保留两位小数，去掉末尾的 0，取绝对值
 *
 * @param growthRate 增长率（小数形式，如 0.15 表示 15%）
 * @returns 格式化后的增长率字符串
 *
 * @example
 * ```ts
 * formatGrowthRate(0.1234);  // "12.34%"
 * formatGrowthRate(0.1000);  // "10%"
 * formatGrowthRate(0);       // "-"
 * formatGrowthRate(-0.05);   // "5%"（取绝对值）
 * ```
 */
export function formatGrowthRate(growthRate: number): string {
  if (growthRate === 0) {
    return "-";
  }

  const formattedRate = Math.abs(growthRate * 100)
    .toFixed(2)
    .replace(/\.?0+$/, "");

  return formattedRate + "%";
}

/**
 * 格式化文件大小
 * @param bytes 字节数
 * @param decimals 保留小数位数，默认 2
 * @returns 格式化后的文件大小字符串
 *
 * @example
 * ```ts
 * formatFileSize(1024);      // "1 KB"
 * formatFileSize(1048576);   // "1 MB"
 * formatFileSize(1234567);   // "1.18 MB"
 * ```
 */
export function formatFileSize(bytes: number, decimals: number = 2): string {
  if (bytes === 0) return "0 Bytes";

  const k = 1024;
  const sizes = ["Bytes", "KB", "MB", "GB", "TB", "PB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));

  return parseFloat((bytes / Math.pow(k, i)).toFixed(decimals)) + " " + sizes[i];
}

/**
 * 格式化数字，添加千分位分隔符
 * @param num 数字
 * @returns 格式化后的字符串
 *
 * @example
 * ```ts
 * formatNumber(1234567);     // "1,234,567"
 * formatNumber(1234567.89);  // "1,234,567.89"
 * ```
 */
export function formatNumber(num: number): string {
  return num.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ",");
}

/**
 * 格式化金额（人民币）
 * @param amount 金额
 * @param decimals 保留小数位数，默认 2
 * @returns 格式化后的金额字符串
 *
 * @example
 * ```ts
 * formatCurrency(1234567);      // "¥1,234,567.00"
 * formatCurrency(1234567.8);    // "¥1,234,567.80"
 * formatCurrency(1234567, 0);   // "¥1,234,567"
 * ```
 */
export function formatCurrency(amount: number, decimals: number = 2): string {
  const formatted = amount.toFixed(decimals).replace(/\B(?=(\d{3})+(?!\d))/g, ",");
  return "¥" + formatted;
}

/**
 * 将以分为单位的金额转换为以美元为单位的字符串，保留两位小数。
 * @param cents - 以分为单位的金额
 */
export function centToDollar(cents: number): string {
  return (cents / 100).toFixed(2);
}

/**
 * 将以字节为单位的大小转换为以 GB 为单位的字符串，保留两位小数。
 * @param bytes
 */
export function bytesToGB(bytes: number): string {
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
}

/**
 * 将字节数格式化为最接近的可读单位（B/KB/MB/GB/TB/PB）
 * @param bytes 字节数
 * @param decimals 保留的小数位数，默认2位
 * @returns 格式化后的字符串（如 "2.50 MB", "1.80 GB"）
 */
export function formatBytes(bytes: number, decimals: number = 2): string {
  // 处理0的特殊情况
  if (bytes === 0) return "0 B";

  // 定义单位换算的基数（1024进制）和单位列表
  const k = 1024;
  const units = ["B", "KB", "MB", "GB", "TB", "PB"];

  // 计算最合适的单位索引
  const i = Math.floor(Math.log(bytes) / Math.log(k));

  // 处理超出最大单位（PB）的情况
  const unitIndex = Math.min(i, units.length - 1);

  // 计算对应单位的数值并保留指定小数位
  const value = (bytes / k ** unitIndex).toFixed(decimals);

  // 返回格式化后的字符串
  return `${value} ${units[unitIndex]}`;
}

/**
 * 从数组或对象中提取所有有效数字（原始 number 类型，排除 NaN/Infinity）
 * @returns 仅包含有效数字的数组
 */
export function filterNumbers(arr: unknown[]): number[] {
  if (!Array.isArray(arr)) {
    throw new TypeError("输入必须是 Array 类型");
  }

  const is_valid_number = (value: unknown): value is number => {
    return (
      typeof value === "number" && // 必须是原始 number 类型
      Object.prototype.toString.call(value) === "[object Number]" && // 排除数字包装对象（new Number()）
      !Number.isNaN(value) && // 排除 NaN
      Number.isFinite(value) // 排除 Infinity
    );
  };

  return arr.filter((element) => is_valid_number(element));
}
