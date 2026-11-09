import { defineStore } from "pinia";

import type {
  identityservicev1_ListPlanQuotaResponse as ListPlanQuotaResponse,
  identityservicev1_ListPlanResponse,
} from "@/api/generated/admin/service/v1";
import { fetchListPlans, fetchListPlanQuotas } from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";

/**
 * 套餐视图状态接口
 */
interface PlanViewState {
  loading: boolean; // 加载状态
  needReloadQuotaList: boolean; // 是否需要重新加载套餐配额列表

  currentPlanId: null | string; // 当前选中的套餐ID
  planList: identityservicev1_ListPlanResponse; // 套餐列表
  planQuotaList: ListPlanQuotaResponse; // 套餐配额列表
}

/**
 * 套餐视图状态
 */
export const usePlanViewStore = defineStore("plan-view", {
  state: (): PlanViewState => ({
    currentPlanId: null,
    loading: false,
    needReloadQuotaList: false,
    planList: { items: [], total: "0" },
    planQuotaList: { items: [], total: "0" },
  }),

  actions: {
    /**
     * 获取套餐列表
     */
    async fetchPlanList(currentPage: number, pageSize: number, formValues: any) {
      this.loading = true;
      try {
        const res = await fetchListPlans(
          new PaginationQuery({
            paging: { page: currentPage, pageSize },
            formValues,
          })
        );
        this.planList = res;
        return res;
      } catch (error) {
        console.error("获取套餐列表失败:", error);
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
      planId: null | string,
      currentPage: number,
      pageSize: number,
      formValues: any
    ) {
      if (!planId) {
        this.resetPlanQuotaList(); // 无套餐ID时清空子列表
        return this.planQuotaList;
      }

      this.loading = true;
      try {
        const res = await fetchListPlanQuotas(
          new PaginationQuery({
            paging: { page: currentPage, pageSize },
            formValues: {
              ...formValues,
              plan_id: planId,
            },
          })
        );
        // 防竞态：快速切换套餐（A→B）时两次请求并发，
        // 若 A 后返回，这里校验 planId 是否仍为当前选中套餐，
        // 避免过期的 A 结果覆盖当前 B 的列表。
        if (planId === this.currentPlanId) {
          this.planQuotaList = res;
        }
        return res;
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
     * 点击套餐时触发：设置当前套餐ID + 刷新套餐配额列表
     * @param planId 套餐ID
     */
    async setCurrentPlanId(planId: null | string) {
      this.currentPlanId = planId; // 更新当前选中的套餐ID
      this.needReloadQuotaList = true; // 触发套餐配额列表刷新
    },

    resetPlanList() {
      this.planList = { items: [], total: "0" };
    },

    resetPlanQuotaList() {
      this.planQuotaList = { items: [], total: "0" };
    },
  },
});
