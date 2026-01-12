import { computed } from 'vue';

import { $t } from '@vben/locales';

import { defineStore } from 'pinia';

import {
  createLoginPolicyServiceClient,
  type LoginPolicy_Type,
} from '#/generated/api/admin/service/v1';
import { makeQueryString, makeUpdateMask } from '#/utils/query';
import { type Paging, requestClientRequestHandler } from '#/utils/request';

export const useLoginPolicyStore = defineStore('login-policy', () => {
  const service = createLoginPolicyServiceClient(requestClientRequestHandler);

  /**
   * 查询登录策略列表
   */
  async function listLoginPolicy(
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
      orderBy: orderBy ?? [],
      query: makeQueryString(formValues ?? null),
      page: paging?.page,
      pageSize: paging?.pageSize,
      noPaging,
    });
  }

  /**
   * 获取登录策略
   */
  async function getLoginPolicy(id: number) {
    return await service.Get({ id });
  }

  /**
   * 创建登录策略
   */
  async function createLoginPolicy(values: object) {
    return await service.Create({
      data: {
        ...values,
      },
    });
  }

  /**
   * 更新登录策略
   */
  async function updateLoginPolicy(id: number, values: object) {
    return await service.Update({
      id,
      data: {
        ...values,
      },
      // @ts-ignore proto generated code is error.
      updateMask: makeUpdateMask(Object.keys(values ?? [])),
    });
  }

  /**
   * 删除登录策略
   */
  async function deleteLoginPolicy(id: number) {
    return await service.Delete({ id });
  }

  function $reset() {}

  return {
    $reset,
    listLoginPolicy,
    getLoginPolicy,
    createLoginPolicy,
    updateLoginPolicy,
    deleteLoginPolicy,
  };
});

export const loginPolicyTypeList = computed(() => [
  { value: 'BLACKLIST', label: $t('enum.loginPolicy.type.BLACKLIST') },
  { value: 'WHITELIST', label: $t('enum.loginPolicy.type.WHITELIST') },
]);

export const loginPolicyMethodList = computed(() => [
  { value: 'IP', label: $t('enum.loginPolicy.method.IP') },
  { value: 'MAC', label: $t('enum.loginPolicy.method.MAC') },
  { value: 'REGION', label: $t('enum.loginPolicy.method.REGION') },
  { value: 'TIME', label: $t('enum.loginPolicy.method.TIME') },
  { value: 'DEVICE', label: $t('enum.loginPolicy.method.DEVICE') },
]);

export function loginPolicyTypeToName(typeName: any) {
  const values = loginPolicyTypeList.value;
  const matchedItem = values.find((item) => item.value === typeName);
  return matchedItem ? matchedItem.label : '';
}

export function loginPolicyTypeToColor(typeName: LoginPolicy_Type) {
  switch (typeName) {
    case 'BLACKLIST': {
      return 'red'; // 黑名单用红色（表示限制/禁止）
    }
    case 'WHITELIST': {
      return 'green'; // 白名单用绿色（表示允许/信任）
    }
    default: {
      // 新增默认分支，处理未知类型，避免返回undefined
      return 'gray'; // 未知类型用灰色（中性默认值）
    }
  }
}

export function loginPolicyMethodToName(methodName: any) {
  const values = loginPolicyMethodList.value;
  const matchedItem = values.find((item) => item.value === methodName);
  return matchedItem ? matchedItem.label : '';
}
