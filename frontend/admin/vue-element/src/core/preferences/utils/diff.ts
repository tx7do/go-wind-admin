import type { DeepPartial, Preferences } from "../types";

/**
 * 深度比较 initial 与 current，产出「与出厂默认的增量」对象（只含被改过的叶子值）。
 * 与 react 端 core/preferences/utils/diff.ts、vben 的 diffPreference 同语义：
 * 复制偏好设置时导出的是增量 overrides，不是全量快照。
 */
export function diffDeep(
  initial: Preferences,
  current: DeepPartial<Preferences>,
): DeepPartial<Preferences> {
  const result: Record<string, unknown> = {};

  for (const key in current) {
    if (!Object.prototype.hasOwnProperty.call(current, key)) {
      continue;
    }
    const currentValue = (current as unknown as Record<string, unknown>)[key];
    const initialValue = (initial as unknown as Record<string, unknown>)[key];

    if (isPlainObject(currentValue) && isPlainObject(initialValue)) {
      const nested = diffDeep(
        initialValue as unknown as Preferences,
        currentValue as unknown as DeepPartial<Preferences>,
      );
      if (Object.keys(nested).length > 0) {
        result[key] = nested;
      }
    } else if (!isEqual(currentValue, initialValue)) {
      result[key] = currentValue;
    }
  }

  return result as unknown as DeepPartial<Preferences>;
}

/**
 * 标量与数组按值比较：数组不等整体替换，不做逐项合并。
 */
function isEqual(a: unknown, b: unknown): boolean {
  if (a === b) {
    return true;
  }
  if (Array.isArray(a) && Array.isArray(b)) {
    return a.length === b.length && a.every((item, i) => isEqual(item, b[i]));
  }
  // NaN 与 NaN 视为相等，避免误报差异
  return Number.isNaN(a) && Number.isNaN(b);
}

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return (
    value !== null &&
    typeof value === "object" &&
    Object.prototype.toString.call(value) === "[object Object]" &&
    !Array.isArray(value)
  );
}
