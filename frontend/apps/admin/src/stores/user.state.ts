import { computed } from 'vue';

import { $t } from '@vben/locales';
import { useUserStore } from '@vben/stores';

import { defineStore } from 'pinia';

import {
  createUserServiceClient,
  type identityservicev1_User_Gender as User_Gender,
  type identityservicev1_User_Status as User_Status,
} from '#/generated/api/admin/service/v1';
import {
  makeOrderBy,
  makeQueryString,
  makeUpdateMask,
  omit,
} from '#/utils/query';
import { type Paging, requestClientRequestHandler } from '#/utils/request';

export const useUserListStore = defineStore('user-list', () => {
  const service = createUserServiceClient(requestClientRequestHandler);
  const userStore = useUserStore();

  /**
   * 查询用户列表
   */
  async function listUser(
    paging?: Paging,
    formValues?: null | object,
    fieldMask?: null | string,
    orderBy?: null | string[],
  ) {
    const noPaging =
      paging?.page === undefined && paging?.pageSize === undefined;
    return await service.List({
      // @ts-ignore proto generated code is error.
      fieldMask,
      orderBy: makeOrderBy(orderBy),
      query: makeQueryString(formValues, userStore.isTenantUser()),
      page: paging?.page,
      pageSize: paging?.pageSize,
      noPaging,
    });
  }

  async function countUser(formValues?: null | object) {
    // @ts-ignore proto generated code is error.
    return await service.Count({
      query: makeQueryString(formValues, userStore.isTenantUser()),
    });
  }

  /**
   * 获取用户
   */
  async function getUser(id: number) {
    return await service.Get({ id });
  }

  /**
   * 创建用户
   */
  async function createUser(values: Record<string, any> = {}) {
    const password = values.password ?? null;
    const cleaned = omit(values, 'password');
    return await service.Create({
      // @ts-ignore proto generated code is error.
      data: {
        ...cleaned,
      },
      password,
    });
  }

  /**
   * 更新用户
   */
  async function updateUser(id: number, values: Record<string, any> = {}) {
    const password = values.password ?? null;
    const cleaned = omit(values, 'password');
    const updateMask = makeUpdateMask(Object.keys(cleaned ?? []));
    return await service.Update({
      id,
      // @ts-ignore proto generated code is error.
      data: {
        ...cleaned,
      },
      password,
      // @ts-ignore proto generated code is error.
      updateMask,
    });
  }

  /**
   * 删除用户
   */
  async function deleteUser(id: number) {
    return await service.Delete({ id });
  }

  /**
   * 用户是否存在
   * @param username 用户名
   */
  async function userExists(username: string) {
    return await service.UserExists({ username });
  }

  /**
   * 修改用户密码
   * @param id 用户ID
   * @param password 用户新密码
   */
  async function editUserPassword(id: number, password: string) {
    return await service.EditUserPassword({
      userId: id,
      newPassword: password,
    });
  }

  function $reset() {}

  return {
    $reset,
    listUser,
    getUser,
    createUser,
    updateUser,
    deleteUser,
    editUserPassword,
    countUser,
    userExists,
  };
});

export const userStatusList = computed(() => [
  { value: 'NORMAL', label: $t('enum.user.status.NORMAL') },
  { value: 'DISABLED', label: $t('enum.user.status.DISABLED') },
  { value: 'PENDING', label: $t('enum.user.status.PENDING') },
  { value: 'LOCKED', label: $t('enum.user.status.LOCKED') },
  { value: 'EXPIRED', label: $t('enum.user.status.EXPIRED') },
  { value: 'CLOSED', label: $t('enum.user.status.CLOSED') },
]);

const USER_STATUS_COLOR_MAP = {
  // 正常态：蓝色（活跃/正常使用）
  NORMAL: '#4096FF',
  // 禁用态：中性灰（非活跃但未删除，区别于锁定/终止）
  DISABLED: '#909399',
  // 待审核/待激活：警告橙（需处理但非风险）
  PENDING: '#FF9A2E',
  // 锁定态：警示红（临时锁定，可解锁）
  LOCKED: '#F56C6C',
  // 终止/离职：危险红（永久失效，高风险）
  TERMINATED: '#F53F3F',
  // 过期态：浅灰（权限/账号过期，非核心风险）
  EXPIRED: '#C9CDD4',
  // 关闭态：深灰（已注销，完全失效）
  CLOSED: '#86909C',
  // 默认值：兜底色（未知状态）
  DEFAULT: '#86909C',
} as const;

export function userStatusToColor(status: User_Status) {
  return (
    USER_STATUS_COLOR_MAP[status as keyof typeof USER_STATUS_COLOR_MAP] ||
    USER_STATUS_COLOR_MAP.DEFAULT
  );
}

export function userStatusToName(status?: User_Status) {
  const values = userStatusList.value;
  const matchedItem = values.find((item) => item.value === status);
  return matchedItem ? matchedItem.label : '';
}

export const genderList = computed(() => [
  { value: 'SECRET', label: $t('enum.gender.SECRET') },
  { value: 'MALE', label: $t('enum.gender.MALE') },
  { value: 'FEMALE', label: $t('enum.gender.FEMALE') },
]);

/**
 * 性别转名称
 * @param gender 性别值
 */
export function genderToName(gender?: User_Gender) {
  const values = genderList.value;
  const matchedItem = values.find((item) => item.value === gender);
  return matchedItem ? matchedItem.label : '';
}

/**
 * 性别转颜色值
 * @param gender 性别值
 */
export function genderToColor(gender?: User_Gender) {
  switch (gender) {
    case 'FEMALE': {
      // 女性：温和粉色，符合大众视觉认知
      return '#F77272';
    } // 柔和粉色
    case 'MALE': {
      // 男性：专业蓝色，体现沉稳感
      return '#4096FF';
    } // 浅蓝色
    case 'SECRET': {
      // 保密：中性灰色，代表未知
      return '#86909C';
    } // 浅灰色
    default: {
      // 异常情况：默认中性色
      return '#C9CDD4';
    } // 极浅灰色
  }
}
