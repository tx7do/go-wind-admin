import { computed } from 'vue';

import { $t } from '@vben/locales';
import { useUserStore } from '@vben/stores';

import { defineStore } from 'pinia';

import {
  createUserServiceClient,
  type userservicev1_User_Gender as User_Gender,
} from '#/generated/api/admin/service/v1';
import { makeOrderBy, makeQueryString, makeUpdateMask } from '#/utils/query';
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

  /**
   * 获取用户
   */
  async function getUser(id: number) {
    return await service.Get({ id });
  }

  /**
   * 创建用户
   */
  async function createUser(values: object) {
    return await service.Create({
      // @ts-ignore proto generated code is error.
      data: {
        ...values,
      },
      // @ts-ignore proto generated code is error.
      password: values.password ?? null,
    });
  }

  /**
   * 更新用户
   */
  async function updateUser(id: number, values: object) {
    const updateMask = makeUpdateMask(Object.keys(values ?? []));
    return await service.Update({
      id,
      // @ts-ignore proto generated code is error.
      data: {
        ...values,
      },
      // @ts-ignore proto generated code is error.
      password: values.password ?? null,
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
    userExists,
  };
});

export const statusList = computed(() => [
  { value: 'ON', label: $t('enum.status.ON') },
  { value: 'OFF', label: $t('enum.status.OFF') },
]);

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
