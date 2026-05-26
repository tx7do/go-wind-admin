/**
 * 用户管理模块通用常量
 */

type TFn = (key: string, options?: Record<string, any>) => string;

// ========== 用户状态 ==========

export const USER_STATUS_COLORS: Record<string, string> = {
  NORMAL: 'success',
  DISABLED: 'error',
  PENDING: 'warning',
  LOCKED: 'error',
  EXPIRED: 'warning',
  CLOSED: 'default',
};

export function getUserStatusMap(t: TFn) {
  const keys = ['NORMAL', 'DISABLED', 'PENDING', 'LOCKED', 'EXPIRED', 'CLOSED'] as const;
  return keys.reduce(
    (acc, key) => {
      acc[key] = { text: t(`statusMap.${key}`), color: USER_STATUS_COLORS[key] };
      return acc;
    },
    {} as Record<string, { text: string; color: string }>,
  );
}

export function getUserStatusOptions(t: TFn) {
  const keys = ['NORMAL', 'DISABLED', 'PENDING', 'LOCKED', 'EXPIRED', 'CLOSED'] as const;
  return keys.map((key) => ({ label: t(`statusMap.${key}`), value: key }));
}

// ========== 性别 ==========

export const GENDER_COLORS: Record<string, string> = {
  MALE: 'blue',
  FEMALE: 'pink',
  SECRET: 'default',
};

export function getGenderMap(t: TFn) {
  const keys = ['MALE', 'FEMALE', 'SECRET'] as const;
  return keys.reduce(
    (acc, key) => {
      acc[key] = { text: t(`genderMap.${key}`), color: GENDER_COLORS[key] };
      return acc;
    },
    {} as Record<string, { text: string; color: string }>,
  );
}

export function getGenderOptions(t: TFn) {
  const keys = ['MALE', 'FEMALE', 'SECRET'] as const;
  return keys.map((key) => ({ label: t(`genderMap.${key}`), value: key }));
}
