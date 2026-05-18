/**
 * 查询操作符类型
 */
export type QueryOperator =
  | "contains"
  | "endswith"
  | "eq"
  | "exact"
  | "gt"
  | "gte"
  | "icontains"
  | "iendswith"
  | "iexact"
  | "in"
  | "iregex"
  | "isnull"
  | "istartswith"
  | "lt"
  | "lte"
  | "ne"
  | "nin"
  | "not"
  | "not_in"
  | "not_isnull"
  | "range"
  | "regex"
  | "search"
  | "startswith";

/**
 * 日期时间操作符类型
 */
export type DateOperator =
  | "date"
  | "day"
  | "iso_week_day"
  | "iso_year"
  | "minute"
  | "month"
  | "quarter"
  | "second"
  | "time"
  | "week"
  | "week_day"
  | "year";

/**
 * 基础条件对象：键为「字段」或「字段__操作符」，值为任意类型
 */
export type BaseQueryCondition = Record<string, (number | string)[] | boolean | number | string>;

/**
 * 逻辑组合节点：支持$and/$or嵌套
 */
export interface LogicalQueryNode {
  $and?: (BaseQueryCondition | LogicalQueryNode)[];
  $or?: (BaseQueryCondition | LogicalQueryNode)[];
}

/**
 * 完整Query类型：兼容简单数组/复杂嵌套对象
 *
 * @example 用法示例
 * const complexQuery: QueryRule = {
 *   $and: [
 *     { deptId: 1 },
 *     {
 *       $or: [
 *         { entryTime__gte: "2024-01-01" },
 *         { userName__icontains: "张" }
 *       ]
 *     },
 *     { status: "active" }
 *   ]
 * };
 */
export type QueryRule = (BaseQueryCondition | LogicalQueryNode)[] | LogicalQueryNode;

/**
 * 判断是否为纯对象
 * @param v
 */
function isPlainObject(v: any): v is Record<string, any> {
  return Object.prototype.toString.call(v) === "[object Object]";
}

/**
 * 清理查询规则对象，移除 null、undefined 和空字符串值
 * @param obj 查询规则对象
 */
export function cleanQueryRule(obj: any): any {
  if (obj === null || obj === undefined || obj === "") {
    return undefined;
  }

  const t = typeof obj;
  if (t === "number" || t === "boolean" || t === "string") {
    return obj;
  }

  if (Array.isArray(obj)) {
    const arr = obj.map((v) => cleanQueryRule(v)).filter((v) => v !== undefined);
    return arr.length === 0 ? undefined : arr;
  }

  // 仅对纯对象递归；Date/RegExp/类实例直接保留
  if (isPlainObject(obj)) {
    const entries = Object.entries(obj)
      .map(([k, v]) => [k, cleanQueryRule(v)] as [string, any])
      .filter(([_, v]) => v !== undefined);
    const result = Object.fromEntries(entries);
    return Object.keys(result).length === 0 ? undefined : result;
  }

  // 非纯对象（Date/RegExp/类实例等）直接返回
  return obj;
}

/**
 * 移除对象中的 null 和 undefined 值
 * @param obj
 */
export const removeNullUndefined = (obj: any) =>
  Object.fromEntries(
    Object.entries(obj).filter(([_, v]) => v !== null && v !== undefined && v !== "")
  );

/**
 * 创建列表查询JSON过滤字符串
 * @param formValues 查询表单值
 * @param needCleanTenant 是否需要清理租户字段
 */
export function makeQueryString(
  formValues?: null | object,
  needCleanTenant: boolean = false
): string | undefined {
  if (formValues === null || formValues === undefined) {
    return undefined;
  }

  // 去除掉空值
  const cleaned: any = removeNullUndefined(formValues);

  if (cleaned === undefined) return undefined;

  // 若是数组，直接按数组处理
  if (Array.isArray(cleaned)) {
    return cleaned.length === 0 ? undefined : JSON.stringify(cleaned);
  }

  // 过滤掉空对象
  if (Object.keys(cleaned).length === 0) {
    return undefined;
  }

  if (needCleanTenant) {
    // 删除租户相关字段 tenant_id 和 tenantId
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const { tenant_id, tenantId, ...rest } = cleaned as Record<string, any>;

    // 过滤掉空对象
    if (Object.keys(rest).length === 0) {
      return undefined;
    }

    return JSON.stringify(rest);
  }

  // 默认返回整个 cleaned 对象的 JSON 字符串
  return JSON.stringify(cleaned);
}

/**
 * 创建列表查询Google AIP-160规范过滤字符串
 *
 * Google AIP-160 过滤语法示例：
 * - 简单相等: name="John"
 * - 比较操作: age > 18
 * - 包含操作: description : "keyword"
 * - 逻辑组合: status="active" AND (age > 18 OR name="John")
 * - 列表成员: roles has "admin"
 *
 * @param filterValues 过滤条件对象
 * @returns Google AIP-160 格式的过滤字符串
 * @example
 * makeFilterString({ name: "John", age: 25 })
 * // 返回: 'name="John" AND age=25'
 *
 * makeFilterString({ status: "active", minAge: 18 })
 * // 返回: 'status="active" AND age>=18'
 */
export function makeFilterString(filterValues?: null | object): string | undefined {
  if (filterValues === null || filterValues === undefined) {
    return undefined;
  }

  // 去除掉空值
  const cleaned = removeNullUndefined(filterValues);

  // 如果是空对象，返回 undefined
  if (Object.keys(cleaned).length === 0) {
    return undefined;
  }

  // 将对象转换为 AIP-160 格式的过滤条件数组
  const conditions: string[] = [];

  for (const [key, value] of Object.entries(cleaned)) {
    if (value === null || value === undefined || value === "") {
      continue;
    }

    // 处理特殊字段名（带操作符后缀）
    // 例如: age__gte -> age >=, name__contains -> name :
    let fieldName = key;
    let operator = "=";

    // 检查是否包含操作符后缀
    if (key.includes("__")) {
      const parts = key.split("__");
      fieldName = parts[0];
      const opSuffix = parts[1];

      // 映射操作符后缀到 AIP-160 操作符
      switch (opSuffix) {
        case "gt":
          operator = ">";
          break;
        case "gte":
          operator = ">=";
          break;
        case "lt":
          operator = "<";
          break;
        case "lte":
          operator = "<=";
          break;
        case "ne":
        case "not_eq":
          operator = "!=";
          break;
        case "contains":
        case "icontains":
          operator = ":";
          break;
        case "startswith":
        case "istartswith":
          // AIP-160 不直接支持 startswith，使用 : 近似
          operator = ":";
          break;
        case "endswith":
        case "iendswith":
          // AIP-160 不直接支持 endswith，使用 : 近似
          operator = ":";
          break;
        case "in":
          // in 操作符需要特殊处理
          if (Array.isArray(value)) {
            const values = value.map((v) => formatAipValue(v)).join(", ");
            conditions.push(`${fieldName} in [${values}]`);
            continue;
          }
          break;
        case "nin":
        case "not_in":
          // not in 操作符需要特殊处理
          if (Array.isArray(value)) {
            const values = value.map((v) => formatAipValue(v)).join(", ");
            conditions.push(`${fieldName} not in [${values}]`);
            continue;
          }
          break;
        default:
          // 默认使用相等操作符
          operator = "=";
          break;
      }
    }

    // 格式化值
    const formattedValue = formatAipValue(value);

    // 构建条件字符串
    if (operator === ":") {
      // 包含操作使用特殊语法
      conditions.push(`${fieldName} : ${formattedValue}`);
    } else {
      conditions.push(`${fieldName}${operator}${formattedValue}`);
    }
  }

  // 如果没有有效条件，返回 undefined
  if (conditions.length === 0) {
    return undefined;
  }

  // 用 AND 连接所有条件
  return conditions.join(" AND ");
}

/**
 * 格式化 AIP-160 值
 * @param value 原始值
 * @returns 格式化后的值字符串
 */
function formatAipValue(value: any): string {
  if (value === null || value === undefined) {
    return "null";
  }

  // 布尔值
  if (typeof value === "boolean") {
    return value ? "true" : "false";
  }

  // 数字
  if (typeof value === "number") {
    return String(value);
  }

  // 数组（用于 IN 操作）
  if (Array.isArray(value)) {
    return `[${value.map((v) => formatAipValue(v)).join(", ")}]`;
  }

  // 字符串需要用双引号包裹
  if (typeof value === "string") {
    // 转义特殊字符
    const escaped = value.replace(/\\/g, "\\\\").replace(/"/g, '\\"');
    return `"${escaped}"`;
  }

  // 其他类型转为字符串
  return `"${String(value)}"`;
}

/**
 * 创建排序字符串
 * @param orderBy
 */
export function makeOrderBy(orderBy?: null | string[]): string | undefined {
  if (orderBy === undefined) {
    orderBy = ["-created_at"];
  }
  if (orderBy === null) {
    orderBy = ["-created_at"];
  }
  return JSON.stringify(orderBy) ?? undefined;
}

/**
 * 创建更新字段掩码
 * @param keys
 */
export function makeUpdateMask(keys: string[]): string {
  if (keys === undefined || keys.length === 0) {
    return "";
  }
  return keys.join(",");
}

/**
 * 从对象中省略指定键，返回新对象
 * @example 用法示例
 * const original = { a: 1, b: 2, c: 3 };
 * const result = omit(original, ['b', 'c']);
 * // result 的值为 { a: 1 }
 * @param obj 原始对象
 * @param keys 需要省略的键或键数组
 */
export function omit<T extends Record<string, any>, K extends string>(
  obj: null | T | undefined,
  keys: K | K[]
): Omit<T, K> {
  if (obj === null || typeof obj !== "object") return obj as any;
  const result = { ...obj } as Record<string, any>;
  const keysArr = Array.isArray(keys) ? keys : [keys];
  for (const key of keysArr) {
    if (Object.prototype.hasOwnProperty.call(result, key)) {
      delete result[key];
    }
  }
  return result as Omit<T, K>;
}
