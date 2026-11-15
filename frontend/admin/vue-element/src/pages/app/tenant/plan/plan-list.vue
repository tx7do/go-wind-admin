<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage
      ref="pageRef"
      :config="pageConfig"
      @add="handleAdd"
      @edit="handleEdit"
      @row-click="handleRowClick"
    >
      <!-- 套餐版本 -->
      <template #version="scope: any">
        <ElTag size="small" round :type="planVersionToType(scope.row.version)">
          {{ planVersionToName(scope.row.version) }}
        </ElTag>
      </template>

      <!-- 到期处置策略 -->
      <template #expiryPolicy="scope: any">
        <ElTag
          size="small"
          round
          :type="planExpiryPolicyToType(scope.row.expiryPolicy)"
        >
          {{ planExpiryPolicyToName(scope.row.expiryPolicy) }}
        </ElTag>
      </template>
    </ProPage>

    <!-- 弹窗组件自动连接 -->
    <ConnectedDrawer />
  </div>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import { ElTag } from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import { useProModal } from "@/components/Pro";
import PlanDrawer from "./plan-drawer.vue";

import {
  planExpiryPolicyToType,
  planExpiryPolicyToName,
  planVersionToType,
  planVersionToName,
  useDeletePlan,
} from "@/api/composables";
import { $t } from "@go-wind/admin-element";
import { usePlanViewStore } from "@/pages/app/tenant/plan/plan-view.state";

const { mutateAsync: deletePlan } = useDeletePlan();
const planViewStore = usePlanViewStore();

const pageRef = ref();

// useProModal 连接 PlanDrawer 组件
const [ConnectedDrawer, modalApi] = useProModal({
  connectedComponent: PlanDrawer,
  onOpenChange(isOpen) {
    if (!isOpen) pageRef.value?.refresh();
  },
});

const pageConfig = computed<ProPageConfig>(() => ({
  skeleton: true,
  search: {
    grid: true,
    fields: [
      {
        type: "input",
        label: $t("pages.plan.name"),
        field: "name",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
    ],
  },

  table: {
    listAction: async (query: any) => {
      const { page, pageSize, ...queryParams } = query;
      const result = await planViewStore.fetchPlanList(page || 1, pageSize || 10, queryParams);
      return { items: result.items || [], total: result.total || 0 };
    },
    deleteAction: async (ids: string) => {
      await deletePlan({ id: ids as any });
    },
    toolbar: [],
    toolbarRight: ["add"],
    defaultToolbar: ["exports", "refresh", "zoom", "filter"],
    tableAttrs: {
      border: true,
      stripe: true,
      height: "auto",
      "show-overflow": "title",
      "show-header-overflow": "title",
    },
    // 左栏分栏主表：内容列不设宽度（均分贴合容器防横向溢出），仅操作列定宽
    columns: [
      {
        prop: "name",
        label: $t("pages.plan.name"),
        align: "left",
      },
      {
        prop: "version",
        label: $t("pages.plan.version"),
        slotName: "version",
      },
      {
        prop: "expiryPolicy",
        label: $t("pages.plan.expiryPolicy"),
        slotName: "expiryPolicy",
      },
      {
        prop: "dataRetentionDays",
        label: $t("pages.plan.dataRetentionDays"),
        align: "right",
      },
      {
        prop: "description",
        label: $t("common.table.description"),
        align: "left",
      },
      {
        prop: "remark",
        label: $t("common.table.remark"),
        align: "left",
      },
      {
        prop: "createdAt",
        label: $t("common.table.createdAt"),
        cellType: "date",
        dateFormat: "YYYY-MM-DD HH:mm:ss",
      },
      {
        prop: "action",
        label: $t("common.table.action"),
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

// 行点击联动 - 切换套餐时刷新套餐配额列表
function handleRowClick(row: any) {
  if (row?.id) {
    planViewStore.setCurrentPlanId(row.id);
  }
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
