import { type ComputedRef, type MaybeRef } from 'vue';

/**
 * 深层递归所有属性为可选。
 * 带深度上限（6 层，足够覆盖 vxe/表单等任意真实配置的嵌套深度）：
 * 无界版本对 vxe-table 这类巨型递归类型做结构比较时会让 TS 实例化超深直接报 TS2589。
 */
type DeepPartial<T, D extends readonly unknown[] = []> = D['length'] extends 6
  ? T
  : T extends object
    ? {
        [P in keyof T]?: DeepPartial<T[P], [...D, unknown]>;
      }
    : T;

/**
 * 深层递归所有属性为只读
 */
type DeepReadonly<T> = {
  readonly [P in keyof T]: T[P] extends object ? DeepReadonly<T[P]> : T[P];
};

/**
 * 任意类型的异步函数
 */

type AnyPromiseFunction<T extends any[] = any[], R = void> = (
  ...arg: T
) => PromiseLike<R>;

/**
 * 任意类型的普通函数
 */
type AnyNormalFunction<T extends any[] = any[], R = void> = (...arg: T) => R;

/**
 * 任意类型的函数
 */
type AnyFunction<T extends any[] = any[], R = void> =
  | AnyNormalFunction<T, R>
  | AnyPromiseFunction<T, R>;

/**
 *  T | null 包装
 */
type Nullable<T> = null | T;

/**
 * T | Not null 包装
 */
type NonNullable<T> = T extends null | undefined ? never : T;

/**
 * 字符串类型对象
 */
type Recordable<T> = Record<string, T>;

/**
 * 字符串类型对象（只读）
 */
interface ReadonlyRecordable<T = any> {
  readonly [key: string]: T;
}

/**
 * setTimeout 返回值类型
 */
type TimeoutHandle = ReturnType<typeof setTimeout>;

/**
 * setInterval 返回值类型
 */
type IntervalHandle = ReturnType<typeof setInterval>;

/**
 * 也许它是一个计算的 ref，或者一个 getter 函数
 *
 */
type MaybeReadonlyRef<T> = (() => T) | ComputedRef<T>;

/**
 * 也许它是一个 ref，或者一个普通值，或者一个 getter 函数
 *
 */
type MaybeComputedRef<T> = MaybeReadonlyRef<T> | MaybeRef<T>;

type Merge<O extends object, T extends object> = {
  [K in keyof O | keyof T]: K extends keyof T
    ? T[K]
    : K extends keyof O
      ? O[K]
      : never;
};

/**
 * T = [
 *  { name: string; age: number; },
 *  { sex: 'male' | 'female'; age: string }
 * ]
 * =>
 * MergeAll<T> = {
 *  name: string;
 *  sex: 'male' | 'female';
 *  age: string
 * }
 */
type MergeAll<
  T extends object[],
  R extends object = Record<string, any>,
> = T extends [infer F extends object, ...infer Rest extends object[]]
  ? MergeAll<Rest, Merge<R, F>>
  : R;

type EmitType = (name: Name, ...args: any[]) => void;

export type {
  AnyFunction,
  AnyNormalFunction,
  AnyPromiseFunction,
  DeepPartial,
  DeepReadonly,
  EmitType,
  IntervalHandle,
  MaybeComputedRef,
  MaybeReadonlyRef,
  Merge,
  MergeAll,
  NonNullable,
  Nullable,
  ReadonlyRecordable,
  Recordable,
  TimeoutHandle,
};
