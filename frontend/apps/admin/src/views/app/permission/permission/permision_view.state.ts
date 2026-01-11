import { defineStore } from 'pinia';

import {
  type permissionservicev1_ListPermissionGroupResponse as ListPermissionGroupResponse,
  type permissionservicev1_ListPermissionResponse as ListPermissionResponse,
} from '#/generated/api/admin/service/v1';
import { usePermissionGroupStore, usePermissionStore } from '#/stores';

const permissionStore = usePermissionStore();
const permissionGroupStore = usePermissionGroupStore();

interface PermissionViewState {
  currentGroupId: null | number; // 当前选中的分组ID
  loading: boolean; // 加载状态
  needReloadGroupList: boolean; // 是否需要重新加载分组列表
  needReloadPermissionList: boolean; // 是否需要重新加载权限列表

  groupList: ListPermissionGroupResponse; // 权限分组列表
  permList: ListPermissionResponse; // 权限列表
}

/**
 * 权限视图状态
 */
export const usePermissionViewStore = defineStore('permission-view', {
  state: (): PermissionViewState => ({
    currentGroupId: null,
    loading: false,
    needReloadGroupList: false,
    needReloadPermissionList: false,
    groupList: { items: [], total: 0 },
    permList: { items: [], total: 0 },
  }),

  actions: {
    /**
     * 获取分组列表
     */
    async fetchGroupList(
      currentPage: number,
      pageSize: number,
      formValues: any,
    ) {
      this.loading = true;
      try {
        this.groupList = await permissionGroupStore.listPermissionGroup(
          {
            page: currentPage,
            pageSize,
          },
          formValues,
        );
        return this.groupList;
      } catch (error) {
        console.error('获取权限分组失败:', error);
        this.resetGroupList();
      } finally {
        this.loading = false;
      }

      return this.groupList;
    },

    /**
     * 根据分组ID获取权限列表
     * @param groupId 分组ID
     * @param currentPage
     * @param pageSize
     * @param formValues
     */
    async fetchPermissionList(
      groupId: null | number,
      currentPage: number,
      pageSize: number,
      formValues: any,
    ) {
      if (!groupId) {
        this.resetPermissionList();
        return this.permList;
      }

      this.loading = true;
      try {
        this.permList = await permissionStore.listPermission(
          {
            page: currentPage,
            pageSize,
          },
          {
            ...formValues,
            group_id: groupId.toString(),
          },
        );
      } catch (error) {
        console.error(`获取分组[${groupId}]的权限点失败:`, error);
        this.resetPermissionList();
      } finally {
        this.loading = false;
      }

      return this.permList;
    },

    /**
     * 设置当前选中的分组ID，并联动刷新权限列表
     * @param groupId 分组ID
     */
    async setCurrentGroupId(groupId: number) {
      this.currentGroupId = groupId; // 更新当前选中的分组ID
      this.needReloadPermissionList = true;
    },

    resetGroupList() {
      this.groupList = { items: [], total: 0 };
    },

    resetPermissionList() {
      this.permList = { items: [], total: 0 };
    },

    reloadGroupList() {
      this.currentGroupId = null;
      this.needReloadGroupList = true;
    },
  },
});
