import { defineStore } from 'pinia';

import {
  type identityservicev1_ListPlanQuotaResponse as ListPlanQuotaResponse,
  type identityservicev1_ListPlanResponse as ListPlanResponse,
} from '#/api';
import {
  fetchListPlanQuotas,
  fetchListPlans,
  PaginationQuery,
} from '#/api';

/**
 * 套餐视图状态接口
 */
interface PlanViewState {
  loading: boolean; // 加载状态

  currentPlanId: null | number; // 当前选中的套餐ID
  planList: ListPlanResponse; // 套餐列表
  planQuotaList: ListPlanQuotaResponse; // 套餐配额列表
}

/**
 * 套餐视图状态
 */
export const usePlanViewStore = defineStore('plan-view', {
  state: (): PlanViewState => ({
    currentPlanId: null,
    loading: false,
    planList: { items: [], total: 0 },
    planQuotaList: { items: [], total: 0 },
  }),

  actions: {
    /**
     * 获取套餐列表
     * @param currentPage
     * @param pageSize
     * @param formValues
     */
    async fetchPlanList(
      currentPage: number,
      pageSize: number,
      formValues: any,
    ) {
      this.loading = true;
      try {
        this.planList = await fetchListPlans(
          new PaginationQuery({
            paging: { page: currentPage, pageSize },
            formValues,
          }),
        );

        await this.setCurrentPlanId(null);

        return this.planList;
      } catch (error) {
        console.error('获取套餐列表失败:', error);
        this.resetPlanList();
      } finally {
        this.loading = false;
      }

      return this.planList;
    },

    /**
     * 根据套餐ID获取套餐配额列表
     * @param planId 套餐ID
     * @param currentPage
     * @param pageSize
     * @param formValues
     */
    async fetchPlanQuotaList(
      planId: null | number,
      currentPage: number,
      pageSize: number,
      formValues: any,
    ) {
      if (!planId) {
        this.resetPlanQuotaList(); // 无套餐ID时清空子列表
        return this.planQuotaList;
      }

      this.loading = true;
      try {
        const response = await fetchListPlanQuotas(
          new PaginationQuery({
            paging: { page: currentPage, pageSize },
            formValues: {
              ...formValues,
              plan_id: planId.toString(),
            },
          }),
        );

        // 防竞态：仅当请求的 planId 仍是当前选中的 planId 时才更新
        if (planId === this.currentPlanId) {
          this.planQuotaList = response;
        }
      } catch (error) {
        console.error(`获取套餐[${planId}]的配额列表失败:`, error);
        if (planId === this.currentPlanId) {
          this.resetPlanQuotaList();
        }
      } finally {
        this.loading = false;
      }

      return this.planQuotaList;
    },

    /**
     * 点击套餐时触发：设置当前套餐ID
     * @param planId 套餐ID
     */
    async setCurrentPlanId(planId: null | number) {
      this.currentPlanId = planId; // 更新当前选中的套餐ID
    },

    resetPlanList() {
      this.planList = { items: [], total: 0 };
    },

    resetPlanQuotaList() {
      this.planQuotaList = { items: [], total: 0 };
    },
  },
});
