import type {DeepPartial} from "../types";
import {isPlainObject} from "./merge";

/**
 * 深度比较 initial 与 current，产出「与出厂默认的增量」对象（只含被改过的叶子值）。
 * 与 vben 的 diffPreference 同语义：复制偏好设置时导出的是增量 overrides，
 * 不是全量快照（对齐 framework/effects/layouts/widgets/preferences/preferences-drawer.vue 的 handleCopy）。
 */
export function diffDeep<T extends object>(initial: T, current: DeepPartial<T>): DeepPartial<T> {
    const result: Record<string, any> = {};

    for (const key in current) {
        if (!Object.prototype.hasOwnProperty.call(current, key)) {
            continue;
        }
        const currentValue = (current as Record<string, any>)[key];
        const initialValue = (initial as Record<string, any>)[key];

        if (isPlainObject(currentValue) && isPlainObject(initialValue)) {
            const nested = diffDeep(initialValue, currentValue);
            if (Object.keys(nested).length > 0) {
                result[key] = nested;
            }
        } else if (!isEqual(currentValue, initialValue)) {
            result[key] = currentValue;
        }
    }

    return result as DeepPartial<T>;
}

/**
 * 标量与数组按值比较：数组不等整体替换（与 mergeDeep 的覆盖语义对齐，不做逐项合并）。
 */
function isEqual(a: any, b: any): boolean {
    if (a === b) {
        return true;
    }
    if (Array.isArray(a) && Array.isArray(b)) {
        return a.length === b.length && a.every((item, i) => isEqual(item, b[i]));
    }
    // NaN 与 NaN 视为相等，避免误报差异
    return Number.isNaN(a) && Number.isNaN(b);
}
