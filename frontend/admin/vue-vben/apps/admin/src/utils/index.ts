/**
 * 深度克隆一个值，支持对象、数组、Map、Set、Date、RegExp 等类型，且能处理循环引用。
 * @param value
 */
export function deepClone<T>(value: T): T {
  // 尝试使用原生 structuredClone，但发生异常时回退到手写实现
  if (typeof (globalThis as any).structuredClone === 'function') {
    try {
      return (globalThis as any).structuredClone(value);
    } catch {
      // structuredClone 不能克隆某些类型（如 Vue 响应式代理、DOM 节点、window 等）
      // 回退到手写深拷贝
    }
  }

  const seen = new WeakMap<any, any>();

  const _clone = (v: any): any => {
    if (v === null || typeof v !== 'object') return v;
    if (v instanceof Date) return new Date(v.getTime());
    if (v instanceof RegExp) return new RegExp(v.source, v.flags);
    if (v instanceof Map) {
      if (seen.has(v)) return seen.get(v);
      const m = new Map();
      seen.set(v, m);
      for (const [k, val] of v) m.set(_clone(k), _clone(val));
      return m;
    }
    if (v instanceof Set) {
      if (seen.has(v)) return seen.get(v);
      const s = new Set();
      seen.set(v, s);
      for (const item of v) s.add(_clone(item));
      return s;
    }
    if (seen.has(v)) return seen.get(v);

    if (Array.isArray(v)) {
      const arr: any[] = [];
      seen.set(v, arr);
      for (let i = 0; i < v.length; i++) arr[i] = _clone(v[i]);
      return arr;
    }

    const obj: any = Object.create(Object.getPrototypeOf(v));
    seen.set(v, obj);
    for (const key of Reflect.ownKeys(v)) {
      obj[key as any] = _clone((v as any)[key as any]);
    }
    return obj;
  };

  return _clone(value);
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
  if (bytes === 0) return '0 B';

  // 定义单位换算的基数（1024进制）和单位列表
  const k = 1024;
  const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];

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
    throw new TypeError('输入必须是 Array 类型');
  }

  const is_valid_number = (value: unknown): value is number => {
    return (
      typeof value === 'number' && // 必须是原始 number 类型
      Object.prototype.toString.call(value) === '[object Number]' && // 排除数字包装对象（new Number()）
      !Number.isNaN(value) && // 排除 NaN
      Number.isFinite(value) // 排除 Infinity
    );
  };

  return arr.filter((element) => is_valid_number(element));
}
