<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig" @operate="handleOperate">
      <!-- 是否成功 -->
      <template #success="scope: any">
        <ElTag size="small" round :type="successToType(scope.row.success)">
          {{ successToNameWithStatusCode(scope.row.success, scope.row.statusCode) }}
        </ElTag>
      </template>

      <!-- 访问类型 -->
      <template #accessType="scope: any">
        <ElTag
          size="small"
          round
          :type="dataAccessAuditLogAccessTypeToType(scope.row.accessType)"
        >
          {{ dataAccessAuditLogAccessTypeToName(scope.row.accessType) }}
        </ElTag>
      </template>

      <!-- 数据分类 -->
      <template #dataCategory="scope: any">
        <ElTag
          size="small"
          round
          :type="dataAccessAuditLogCategoryToType(scope.row.dataCategory)"
        >
          {{ dataAccessAuditLogCategoryToName(scope.row.dataCategory) }}
        </ElTag>
      </template>

      <!-- 地理位置 -->
      <template #geoLocation="scope: any">
        {{ scope.row.geoLocation?.province }} {{ scope.row.geoLocation?.city }}
      </template>
    </ProPage>

    <!-- 详情抽屉 -->
    <DataAccessAuditLogDetailDrawer ref="drawerRef" />
  </div>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import { ElTag } from "element-plus";
import dayjs from "dayjs";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import DataAccessAuditLogDetailDrawer from "./detail-drawer.vue";

import {
  dataAccessAuditLogAccessTypeList,
  dataAccessAuditLogAccessTypeToType,
  dataAccessAuditLogAccessTypeToName,
  dataAccessAuditLogCategoryToType,
  dataAccessAuditLogCategoryToName,
  successStatusList,
  successToType,
  successToNameWithStatusCode,
  fetchListDataAccessAuditLogs,
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
      'data_access',
      new PaginationQuery({ formValues: latestFormValuesRef.value }),
    );
    ElMessage.success($t("pages.data_access_audit_log.exportServerSuccess"));
  } catch (error: any) {
    // 原始错误必须留在控制台：用户可见的那句翻译不包含服务端的原因
    console.error("server-side audit export failed", error);
    ElMessage.error(error?.message || $t("pages.data_access_audit_log.exportServerFailed"));
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
  exportFilename: "data-access-audit-logs",
  search: {
    grid: true,
    fields: [
      {
        type: "input",
        label: $t("pages.data_access_audit_log.username"),
        field: "username",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "input",
        label: $t("pages.data_access_audit_log.tableName"),
        field: "tableName",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "select",
        label: $t("pages.data_access_audit_log.accessType"),
        field: "accessType",
        attrs: { placeholder: $t("common.placeholder.select"), clearable: true, filterable: true },
        options: dataAccessAuditLogAccessTypeList.value,
      },
      {
        type: "input",
        label: $t("pages.data_access_audit_log.ipAddress"),
        field: "ipAddress",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "select",
        label: $t("pages.data_access_audit_log.success"),
        field: "success",
        attrs: { placeholder: $t("common.placeholder.select"), clearable: true, filterable: true },
        options: successStatusList.value,
      },
      {
        type: "date-picker",
        label: $t("pages.data_access_audit_log.createdAt"),
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
        username: queryParams.username,
        accessType: queryParams.accessType,
        tableName: queryParams.tableName,
        ipAddress: queryParams.ipAddress,
        success: queryParams.success,
        created_at__gte: startTime,
        created_at__lte: endTime,
      };
      latestFormValuesRef.value = formValues;

      const result = await fetchListDataAccessAuditLogs(
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
        label: $t("pages.data_access_audit_log.createdAt"),
        minWidth: 160,
        cellType: "date",
        dateFormat: "YYYY-MM-DD HH:mm:ss",
      },
      {
        prop: "success",
        label: $t("pages.data_access_audit_log.success"),
        width: 120,
        slotName: "success",
      },
      {
        prop: "accessType",
        label: $t("pages.data_access_audit_log.accessType"),
        width: 140,
        slotName: "accessType",
      },
      { prop: "tableName", label: $t("pages.data_access_audit_log.tableName"), minWidth: 150 },
      {
        prop: "dataCategory",
        label: $t("pages.data_access_audit_log.dataCategory"),
        minWidth: 150,
        slotName: "dataCategory",
      },
      {
        prop: "latencyMs",
        label: $t("pages.data_access_audit_log.latencyMs"),
        width: 120,
        align: "right",
      },
      { prop: "username", label: $t("pages.data_access_audit_log.username"), minWidth: 120 },
      {
        prop: "geoLocation",
        label: $t("pages.data_access_audit_log.geoLocation"),
        minWidth: 150,
        slotName: "geoLocation",
      },
      {
        prop: "ipAddress",
        label: $t("pages.data_access_audit_log.ipAddress"),
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
