<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig" @operate="handleOperate">
      <!-- 操作类型 -->
      <template #action="scope: any">
        <ElTag
          size="small"
          round
          :type="permissionAuditLogActionToType(scope.row.action)"
        >
          {{ permissionAuditLogActionToName(scope.row.action) }}
        </ElTag>
      </template>

      <!-- 地理位置 -->
      <template #geoLocation="scope: any">
        {{ scope.row.geoLocation?.province }} {{ scope.row.geoLocation?.city }}
      </template>
    </ProPage>

    <!-- 详情抽屉 -->
    <PermissionAuditLogDetailDrawer ref="drawerRef" />
  </div>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import { ElTag } from "element-plus";
import dayjs from "dayjs";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import PermissionAuditLogDetailDrawer from "./detail-drawer.vue";

import {
  permissionAuditLogActionList,
  permissionAuditLogActionToType,
  permissionAuditLogActionToName,
  fetchListPermissionAuditLogs,
} from "@/api/composables";
import { PaginationQuery } from "@go-wind/admin-core";
import { exportAuditLogsServer } from "@/api/composables";
import { $t } from "@go-wind/admin-element";

const pageRef = ref();
const drawerRef = ref();
const latestFormValuesRef = ref<Record<string, unknown>>({});

const exporting = ref(false);

async function handleServerExport() {
  if (exporting.value) return;
  exporting.value = true;
  try {
    await exportAuditLogsServer(
      'permission',
      new PaginationQuery({ formValues: latestFormValuesRef.value }),
    );
    ElMessage.success($t("pages.permission_audit_log.exportServerSuccess"));
  } catch (error: any) {
    // 原始错误必须留在控制台：用户可见的那句翻译不包含服务端的原因
    console.error("server-side audit export failed", error);
    ElMessage.error(error?.message || $t("pages.permission_audit_log.exportServerFailed"));
  } finally {
    exporting.value = false;
  }
}

function handleOperate(data: { name: string; row: any }) {
  // 服务端全量导出（XLSX，上限 50 万行）：不带搜索条件，突破导出弹窗
  // 客户端聚合的 1 万行上限；带条件的导出走右侧既有的导出弹窗
  if (data.name === "detail") {
    drawerRef.value?.open({ row: data.row });
  }
}

const pageConfig = computed<ProPageConfig>(() => ({
  skeleton: true,
  exportFilename: "permission-audit-logs",
  search: {
    grid: true,
    fields: [
      {
        type: "input",
        label: $t("pages.permission_audit_log.targetType"),
        field: "targetType",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "input",
        label: $t("pages.permission_audit_log.operatorName"),
        field: "operatorName",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "select",
        label: $t("pages.permission_audit_log.action"),
        field: "action",
        attrs: { placeholder: $t("common.placeholder.select"), clearable: true, filterable: true },
        options: permissionAuditLogActionList.value,
      },
      {
        type: "input",
        label: $t("pages.permission_audit_log.ipAddress"),
        field: "ipAddress",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "date-picker",
        label: $t("pages.permission_audit_log.createdAt"),
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
        targetType: queryParams.targetType,
        operatorName: queryParams.operatorName,
        action: queryParams.action,
        ipAddress: queryParams.ipAddress,
        created_at__gte: startTime,
        created_at__lte: endTime,
      };
      latestFormValuesRef.value = formValues;

      const result = await fetchListPermissionAuditLogs(
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
        label: $t("pages.permission_audit_log.createdAt"),
        minWidth: 160,
        cellType: "date",
        dateFormat: "YYYY-MM-DD HH:mm:ss",
      },
      {
        prop: "action",
        label: $t("pages.permission_audit_log.action"),
        width: 140,
        slotName: "action",
      },
      { prop: "targetType", label: $t("pages.permission_audit_log.targetType"), minWidth: 150 },
      { prop: "targetName", label: $t("pages.permission_audit_log.targetName"), minWidth: 150 },
      { prop: "reason", label: $t("pages.permission_audit_log.reason"), minWidth: 200 },
      { prop: "operatorName", label: $t("pages.permission_audit_log.operatorName"), minWidth: 120 },
      {
        prop: "geoLocation",
        label: $t("pages.permission_audit_log.geoLocation"),
        minWidth: 150,
        slotName: "geoLocation",
      },
      {
        prop: "ipAddress",
        label: $t("pages.permission_audit_log.ipAddress"),
        width: 140,
        align: "right",
      },
      {
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
