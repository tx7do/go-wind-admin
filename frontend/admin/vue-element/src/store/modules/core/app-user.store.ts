import { computed, ref } from "vue";

import { acceptHMRUpdate, defineStore } from "pinia";

export type AccessState = {
  userInfo: BasicUserInfo | null;
  userRoles: string[];
};

/**
 * @zh_CN 用户信息相关
 */
export const useAppUserStore = defineStore("app-user", () => {
  const userInfo = ref<AccessState["userInfo"]>(null);
  const userRoles = ref<AccessState["userRoles"]>([]);

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
    return tenantId.value !== null && tenantId.value !== undefined && tenantId.value > 0;
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
  hot.accept(acceptHMRUpdate(useAppUserStore, hot));
}
