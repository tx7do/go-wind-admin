import { useUserStore } from '@vben/stores';

import { defineStore } from 'pinia';

import {
  type userservicev1_ListOrgUnitResponse as ListOrgUnitResponse,
  type userservicev1_ListTenantResponse as ListTenantResponse,
  type userservicev1_ListUserResponse as ListUserResponse,
} from '#/generated/api/admin/service/v1';
import { useOrgUnitStore, useTenantStore, useUserListStore } from '#/stores';

const userStore = useUserStore();
const userListStore = useUserListStore();
const orgUnitStore = useOrgUnitStore();
const tenantStore = useTenantStore();

interface UserViewState {
  loading: boolean; // 加载状态

  currentTenantId: number | undefined; // 当前选中的租户ID
  currentOrgUnitId: number | undefined; // 当前选中的组织ID

  userList: ListUserResponse; // 用户列表
  tenantList: ListTenantResponse; // 租户列表
  orgUnitList: ListOrgUnitResponse; // 组织列表
}

/**
 * 用户视图状态管理
 */
export const useUserViewStore = defineStore('user-view', {
  state: (): UserViewState => ({
    currentTenantId: undefined,
    currentOrgUnitId: undefined,
    loading: false,
    userList: { items: [], total: 0 },
    tenantList: { items: [], total: 0 },
    orgUnitList: { items: [], total: 0 },
  }),

  actions: {
    /**
     * 获取租户列表
     */
    async fetchTenantList(formValues?: any): Promise<ListTenantResponse> {
      this.loading = true;
      try {
        this.tenantList = await tenantStore.listTenant(undefined, {
          ...formValues,
          status: 'ON',
        });
        return this.tenantList;
      } catch (error) {
        console.error('获取租户列表失败:', error);
        this.resetTenantList();
      } finally {
        this.loading = false;
      }

      return this.tenantList;
    },

    /**
     * 根据租户获取组织列表
     */
    async fetchOrgUnitList(formValues?: any): Promise<ListOrgUnitResponse> {
      this.loading = true;
      try {
        this.orgUnitList = await orgUnitStore.listOrgUnit(undefined, {
          ...formValues,
          tenant_id: this.currentTenantId ?? 0,
          status: 'ON',
        });
      } catch (error) {
        console.error(`获取租户[${this.currentTenantId}]的组织失败:`, error);
        this.resetOrgUnitList();
      } finally {
        this.loading = false;
      }

      return this.orgUnitList;
    },

    /**
     * 根据租户以及组织获取用户列表
     */
    async fetchUserList(
      currentPage: number,
      pageSize: number,
      formValues: any,
    ): Promise<ListUserResponse> {
      this.loading = true;
      try {
        this.userList = await userListStore.listUser(
          {
            page: currentPage,
            pageSize,
          },
          {
            ...formValues,
            tenant_id: this.currentTenantId ?? 0,
            org_unit_id: this.currentOrgUnitId,
          },
        );
      } catch (error) {
        console.error(
          `获取租户[${this.currentTenantId}]组织[${this.currentOrgUnitId}]的用户失败:`,
          error,
        );
        this.resetUserList();
      } finally {
        this.loading = false;
      }

      return this.userList;
    },

    resetTenantList() {
      this.tenantList = { items: [], total: 0 };
    },

    resetOrgUnitList() {
      this.orgUnitList = { items: [], total: 0 };
    },

    resetUserList() {
      this.userList = { items: [], total: 0 };
    },

    setCurrentTenantId(tenantId: number | undefined) {
      this.currentTenantId = tenantId;
      this.currentOrgUnitId = undefined;
    },

    setCurrentOrgUnitId(orgUnitId: number | undefined) {
      this.currentOrgUnitId = orgUnitId;
    },

    getCurrentTenantId(): number {
      if (userStore.isTenantUser()) {
        return userStore.tenantId as number;
      }
      return this.currentTenantId ?? 0;
    },

    /**
     * 当前用户是否为租户用户
     */
    isTenantUser(): boolean {
      return userStore.isTenantUser();
    },
  },
});
