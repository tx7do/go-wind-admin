<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig" @add="handleAdd" @edit="handleEdit">
      <!-- 配额类型 -->
      <template #quotaType="scope: any">
        <ElTag size="small" round :type="planQuotaTypeToType(scope.row.quotaType)">
          {{ planQuotaTypeToName(scope.row.quotaType) }}
        </ElTag>
      </template>
    </ProPage>

    <!-- 弹窗组件自动连接 -->
    <ConnectedDrawer />
  </div>
</template>

<script lang="ts" setup>
import { ref, watch, onMounted } from "vue";
import { ElTag } from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import { useProModal } from "@/components/Pro";
import PlanQuotaDrawer from "./plan-quota-drawer.vue";

import {
  planQuotaTypeToType,
  planQuotaTypeToName,
  useDeletePlanQuota,
} from "@/api/composables";
import { $t } from "@go-wind/admin-element";
import { usePlanViewStore } from "@/pages/app/tenant/plan/plan-view.state";

const { mutateAsync: deletePlanQuota } = useDeletePlanQuota();
const planViewStore = usePlanViewStore();

const pageRef = ref();

// useProModal 连接 PlanQuotaDrawer 组件
const [ConnectedDrawer, modalApi] = useProModal({
  connectedComponent: PlanQuotaDrawer,
  onOpenChange(isOpen) {
    if (!isOpen) pageRef.value?.refresh();
  },
});

// 监听套餐切换,自动刷新套餐配额列表
watch(
  () => planViewStore.needReloadQuotaList,
  (needReload) => {
    if (needReload && pageRef.value) {
      pageRef.value.refresh();
      planViewStore.needReloadQuotaList = false;
    }
  }
);

// 初始化时加载一次
onMounted(() => {
  if (planViewStore.currentPlanId) {
    planViewStore.needReloadQuotaList = true;
  }
});

const pageConfig = computed<ProPageConfig>(() => ({
  skeleton: true,
  search: {
    grid: true,
    fields: [],
  },

  table: {
    listAction: async (query: any) => {
      const { page, pageSize, ...queryParams } = query;
      const result = await planViewStore.fetchPlanQuotaList(
        planViewStore.currentPlanId,
        page || 1,
        pageSize || 10,
        queryParams
      );
      return { items: result.items || [], total: result.total || 0 };
    },
    deleteAction: async (ids: string) => {
      await deletePlanQuota({ id: ids as any });
    },
    toolbar: [],
    toolbarRight: ["add"],
    defaultToolbar: ["exports", "refresh", "zoom", "filter"],
    tableAttrs: { border: true, stripe: true, height: "auto" },
    columns: [
      {
        prop: "quotaType",
        label: $t("pages.plan.quotaType"),
        width: 120,
        slotName: "quotaType",
      },
      {
        prop: "quotaValue",
        label: $t("pages.plan.quotaValue"),
        width: 120,
        align: "right",
      },
      {
        prop: "createdAt",
        label: $t("common.table.createdAt"),
        minWidth: 160,
        cellType: "date",
        dateFormat: "YYYY-MM-DD HH:mm:ss",
      },
      {
        prop: "action",
        label: $t("common.table.action"),
        fixed: "right",
        width: 150,
        cellType: "tool",
        buttons: [
          { name: "edit", label: $t("common.button.edit"), icon: "lucide:pen-line" },
          {
            name: "delete",
            label: $t("common.button.delete"),
            icon: "lucide:trash-2",
            attrs: { type: "danger" },
          },
        ],
      },
    ],
  },
}));

function handleAdd() {
  modalApi.open({ create: true });
}

function handleEdit(row: any) {
  modalApi.open({ create: false, row });
}
</script>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
  width: 100%;
  min-width: 0;
  flex-shrink: 0;
}
</style>
