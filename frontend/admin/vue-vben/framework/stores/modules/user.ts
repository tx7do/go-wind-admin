import { computed, ref } from 'vue';

import { acceptHMRUpdate, defineStore } from 'pinia';

interface BasicUserInfo {
  [key: string]: any;

  /**
   * 头像
   */
  avatar: string;
  /**
   * 用户id
   */
  id: number;

  /**
   * 用户昵称
   */
  nickname: string;
  /**
   * 用户实名
   */
  realname: string;
  /**
   * 用户角色
   */
  roles?: string[];
  /**
   * 租户id
   */
  tenantId: number;
  /**
   * 用户名
   */
  username: string;
}

export type AccessState = {
  userInfo: BasicUserInfo | null;
  userRoles: string[];
};

/**
 * @zh_CN 用户信息相关
 */
export const useUserStore = defineStore('core-user', () => {
  const userInfo = ref<AccessState['userInfo']>(null);
  const userRoles = ref<AccessState['userRoles']>([]);

  function setUserInfo(info: BasicUserInfo | null) {
    userInfo.value = info;
    setUserRoles(info?.roles ?? []);
  }

  function setUserRoles(roles: string[]) {
    userRoles.value = roles;
  }

  const tenantId = computed(() => userInfo.value?.tenantId ?? null);
  const isLoggedIn = computed(() => userInfo.value !== null);

  /**
   * @zh_CN 判断当前用户是否为租户用户
   */
  function isTenantUser(): boolean {
    return (
      tenantId.value !== null &&
      tenantId.value !== undefined &&
      tenantId.value > 0
    );
  }

  function $reset() {}

  return {
    $reset,
    isLoggedIn,
    isTenantUser,
    setUserInfo,
    setUserRoles,
    tenantId,
    userInfo,
    userRoles,
  };
});

// 解决热更新问题
const hot = import.meta.hot;
if (hot) {
  hot.accept(acceptHMRUpdate(useUserStore, hot));
}
