import type { HttpResponse } from './types';

/**
 * 创建更新掩码字符串
 * @param keys - 字段键名数组
 * @returns 逗号分隔的字符串
 */
export function makeUpdateMask(keys: string[]): string {
  return [...keys, 'id'].join(',');
}

/**
 * 默认的请求 ID 生成器
 */
export function defaultIdGenerator(): string {
  try {
    // 优先使用标准 API
    const rnd = (
      globalThis as unknown as { crypto?: { randomUUID?: () => string } }
    )?.crypto?.randomUUID?.();
    if (typeof rnd === 'string' && rnd.length > 0) return rnd;
  } catch {
    // ignore
  }
  // 降级方案
  return Math.random().toString(36).slice(2) + Date.now().toString(36);
}

/**
 * 默认的错误消息提取（不依赖 i18n，纯逻辑 fallback）
 * 仅基于 reason；不再回退到后端 message 字段（该字段后续将被移除）。
 * 如需 i18n 翻译，通过 RequestClientCallbacks.getErrorMsg 注入。
 */
export function getDefaultErrorMsg(error: unknown): string {
  // 网络错误
  const errStr = String(error ?? '');
  if (errStr.includes('Network Error')) {
    return 'Network Error';
  }

  // 超时
  if (
    error &&
    typeof error === 'object' &&
    'message' in error &&
    String(error.message).includes('timeout')
  ) {
    return 'Request Timeout';
  }

  // 获取后端返回数据
  const resData =
    error &&
    typeof error === 'object' &&
    'response' in error &&
    error.response &&
    typeof error.response === 'object' &&
    'data' in error.response
      ? (error.response.data as HttpResponse)
      : undefined;

  if (!resData) {
    return 'Unknown Error';
  }

  const { reason } = resData;

  // 1. 优先使用 reason（原始值，i18n 翻译由注入的 getErrorMsg 负责）
  if (reason) {
    return reason;
  }

  // 2. 兜底：不再回退到后端 message 字段
  return 'Unknown Error';
}

export function bindMethods<T extends object>(instance: T): void {
  const prototype = Object.getPrototypeOf(instance);
  const propertyNames = Object.getOwnPropertyNames(prototype);

  propertyNames.forEach((propertyName) => {
    const descriptor = Object.getOwnPropertyDescriptor(prototype, propertyName);
    const propertyValue = instance[propertyName as keyof T];

    if (
      typeof propertyValue === 'function' &&
      propertyName !== 'constructor' &&
      descriptor &&
      !descriptor.get &&
      !descriptor.set
    ) {
      instance[propertyName as keyof T] = propertyValue.bind(instance);
    }
  });
}
