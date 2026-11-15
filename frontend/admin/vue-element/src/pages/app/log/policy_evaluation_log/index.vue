<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig" @operate="handleOperate">
      <!-- 评估结果 -->
      <template #result="scope: any">
        <ElTag size="small" round :type="successToType(scope.row.result)">
          {{ successToName(scope.row.result) }}
        </ElTag>
      </template>
    </ProPage>

    <!-- 详情抽屉 -->
    <PolicyEvaluationLogDetailDrawer ref="drawerRef" />
  </div>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import { ElTag } from "element-plus";
import dayjs from "dayjs";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import PolicyEvaluationLogDetailDrawer from "./detail-drawer.vue";

import {
  methodList,
  successStatusList,
  successToType,
  successToName,
  fetchListPolicyEvaluationLogs,
} from "@/api/composables";
import { PaginationQuery } from "@go-wind/admin-core";
import { exportAuditLogsServer } from "@/api/composables";
import { $t } from "@go-wind/admin-element";

const pageRef = ref();
const drawerRef = ref();
const latestFormValuesRef = ref<Record<string, unknown>>({});

function handleOperate(data: { name: string; row: any }) {
  if (data.name === "detail") {
    drawerRef.value?.open({ row: data.row });
  }
}

// 服务端全量导出（XLSX，上限 50 万行）：当前搜索条件经 PaginationQuery
// 同源序列化透传，导出的即当前搜索看到的。
const exporting = ref(false);

async function handleServerExport() {
  if (exporting.value) return;
  exporting.value = true;
  try {
    await exportAuditLogsServer(
      "policy_evaluation",
      new PaginationQuery({ formValues: latestFormValuesRef.value }),
    );
    ElMessage.success($t("pages.policy_evaluation_log.exportServerSuccess"));
  } catch (error: any) {
    // 原始错误必须留在控制台：用户可见的那句翻译不包含服务端的原因
    console.error("server-side audit export failed", error);
    ElMessage.error(error?.message || $t("pages.policy_evaluation_log.exportServerFailed"));
  } finally {
    exporting.value = false;
  }
}

const pageConfig = computed<ProPageConfig>(() => ({
  skeleton: true,
  exportFilename: "policy-evaluation-logs",
  search: {
    grid: true,
    fields: [
      {
        type: "select",
        label: $t("pages.policy_evaluation_log.requestMethod"),
        field: "requestMethod",
        attrs: { placeholder: $t("common.placeholder.select"), clearable: true, filterable: true },
        options: methodList,
      },
      {
        type: "input",
        label: $t("pages.policy_evaluation_log.requestPath"),
        field: "requestPath",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "select",
        label: $t("pages.policy_evaluation_log.result"),
        field: "result",
        attrs: { placeholder: $t("common.placeholder.select"), clearable: true, filterable: true },
        options: successStatusList.value,
      },
      {
        type: "input",
        label: $t("pages.policy_evaluation_log.userId"),
        field: "userId",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "input",
        label: $t("pages.policy_evaluation_log.ipAddress"),
        field: "ipAddress",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "date-picker",
        label: $t("pages.policy_evaluation_log.createdAt"),
        field: "createdAt",
        attrs: {
          type: "datetimerange",
          startPlaceholder: $t("common.placeholder.date"),
          endPlaceholder: $t("common.placeholder.date"),
          clearable: true,
          shortcuts: [
            {
              text: $t("common.dateRange.today"),
              value: () => [dayjs().startOf("day").toDate(), dayjs().endOf("day").toDate()],
            },
            {
              text: $t("common.dateRange.yesterday"),
              value: () => [
                dayjs().subtract(1, "day").startOf("day").toDate(),
                dayjs().subtract(1, "day").endOf("day").toDate(),
              ],
            },
            {
              text: $t("common.dateRange.thisWeek"),
              value: () => [dayjs().startOf("week").toDate(), dayjs().endOf("week").toDate()],
            },
            {
              text: $t("common.dateRange.lastWeek"),
              value: () => [
                dayjs().subtract(1, "week").startOf("week").toDate(),
                dayjs().subtract(1, "week").endOf("week").toDate(),
              ],
            },
            {
              text: $t("common.dateRange.thisMonth"),
              value: () => [dayjs().startOf("month").toDate(), dayjs().endOf("month").toDate()],
            },
            {
              text: $t("common.dateRange.lastMonth"),
              value: () => [
                dayjs().subtract(1, "month").startOf("month").toDate(),
                dayjs().subtract(1, "month").endOf("month").toDate(),
              ],
            },
          ],
        },
      },
    ],
  },

  table: {
    // 服务端全量导出：导出全部下拉的第三项，错误处理在本页 handleServerExport
    exportServer: handleServerExport,

    listAction: async (query: any) => {
      const { page, pageSize, createdAt, ...queryParams } = query;

      let startTime: string | undefined;
      let endTime: string | undefined;
      if (createdAt && Array.isArray(createdAt) && createdAt.length === 2) {
        startTime = dayjs(createdAt[0]).format("YYYY-MM-DD HH:mm:ss");
        endTime = dayjs(createdAt[1]).format("YYYY-MM-DD HH:mm:ss");
      }

      // 列表与导出共用同一份搜索条件：服务端导出透传的就是这里看到的
      const formValues = {
        requestMethod: queryParams.requestMethod,
        requestPath: queryParams.requestPath,
        result: queryParams.result,
        userId: queryParams.userId,
        ipAddress: queryParams.ipAddress,
        created_at__gte: startTime,
        created_at__lte: endTime,
      };
      latestFormValuesRef.value = formValues;

      const result = await fetchListPolicyEvaluationLogs(
        new PaginationQuery({
          paging: { page: page || 1, pageSize: pageSize || 10 },
          formValues,
          orderBy: ["-created_at"],
        })
      );
      return { items: result.items || [], total: result.total || 0 };
    },
    toolbar: [],
    toolbarRight: [],
    defaultToolbar: ["exports", "refresh", "zoom", "filter"],
    tableAttrs: { border: true, stripe: false },
    columns: [
      {
        prop: "createdAt",
        label: $t("pages.policy_evaluation_log.createdAt"),
        minWidth: 160,
        cellType: "date",
        dateFormat: "YYYY-MM-DD HH:mm:ss",
      },
      {
        prop: "result",
        label: $t("pages.policy_evaluation_log.result"),
        width: 110,
        slotName: "result",
      },
      { prop: "requestMethod", label: $t("pages.policy_evaluation_log.requestMethod"), width: 120 },
      {
        prop: "requestPath",
        label: $t("pages.policy_evaluation_log.requestPath"),
        minWidth: 200,
      },
      {
        prop: "userId",
        label: $t("pages.policy_evaluation_log.userId"),
        width: 120,
        align: "right",
      },
      {
        prop: "permissionId",
        label: $t("pages.policy_evaluation_log.permissionId"),
        width: 120,
        align: "right",
      },
      {
        prop: "policyId",
        label: $t("pages.policy_evaluation_log.policyId"),
        width: 120,
        align: "right",
      },
      {
        prop: "ipAddress",
        label: $t("pages.policy_evaluation_log.ipAddress"),
        width: 140,
        align: "right",
      },
      {
        prop: "action",
        label: $t("common.table.action"),
        fixed: "right",
        width: 90,
        cellType: "tool",
        buttons: [{ name: "detail", label: $t("common.button.detail"), icon: "lucide:eye" }],
      },
    ],
  },
}));
</script>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
  width: 100%;
  min-width: 0;
  flex-shrink: 0;
}
</style>
