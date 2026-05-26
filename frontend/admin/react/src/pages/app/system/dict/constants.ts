/**
 * 字典模块通用常量
 */

type TFn = (key: string, options?: Record<string, any>) => string;

// ========== 启用/禁用 ==========

export function getEnableColor(isEnabled: boolean): string {
  return isEnabled ? 'success' : 'error';
}

export function getEnableLabel(t: TFn, isEnabled: boolean): string {
  return isEnabled ? t('statusMap.true') : t('statusMap.false');
}

export function enableBoolOptions(t: TFn) {
  return [
    { label: t('statusMap.true'), value: 'true' },
    { label: t('statusMap.false'), value: 'false' },
  ];
}

export function enableBoolRadioOptions(t: TFn) {
  return [
    { label: t('statusMap.true'), value: true },
    { label: t('statusMap.false'), value: false },
  ];
}

/**
 * 获取条目标签（根据当前语言环境从 i18n 中提取）
 */
export function getEntryLabel(row: any, locale: string): string {
  const currentI18n = row.i18n?.[locale];
  if (!currentI18n) return '';
  return currentI18n.entryLabel ?? '';
}
