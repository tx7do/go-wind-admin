<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig">
      <!-- 状态 -->
      <template #status="scope: any">
        <ElTag
          size="small"
          effect="dark"
          round
          :color="getLoginAuditLogStatusColor(scope.row.status)"
        >
          {{ loginAuditLogStatusToName(scope.row.status) }}
        </ElTag>
      </template>

      <!-- 操作类型 -->
      <template #actionType="scope: any">
        <ElTag
          size="small"
          effect="dark"
          round
          :color="getLoginAuditLogActionTypeColor(scope.row.actionType)"
        >
          {{ loginAuditLogActionTypeToName(scope.row.actionType) }}
        </ElTag>
      </template>

      <!-- 风险等级 -->
      <template #riskLevel="scope: any">
        <ElTag
          size="small"
          effect="dark"
          round
          :color="getLoginAuditLogRiskLevelColor(scope.row.riskLevel)"
        >
          {{ loginAuditLogRiskLevelToName(scope.row.riskLevel) }}
        </ElTag>
      </template>

      <!-- 地理位置 -->
      <template #geoLocation="scope: any">
        {{ scope.row.geoLocation?.province }} {{ scope.row.geoLocation?.city }}
      </template>

      <!-- 平台信息 -->
      <template #platform="scope: any">
        {{ scope.row.deviceInfo?.osName }} {{ scope.row.deviceInfo?.browserName }}
      </template>
    </ProPage>
  </div>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import { ElTag } from "element-plus";
import dayjs from "dayjs";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";

import {
  getLoginAuditLogActionTypeColor,
  getLoginAuditLogRiskLevelColor,
  getLoginAuditLogStatusColor,
  loginAuditLogActionTypeList,
  loginAuditLogActionTypeToName,
  loginAuditLogRiskLevelList,
  loginAuditLogRiskLevelToName,
  loginAuditLogStatusList,
  loginAuditLogStatusToName,
  fetchListLoginAuditLogs,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/i18n";

const pageRef = ref();

const pageConfig = computed<ProPageConfig>(() => ({

  search: {
    grid: true,
    fields: [
      {
        type: "input",
        label: $t("pages.login_audit_log.username"),
        field: "username",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "input",
        label: $t("pages.login_audit_log.ipAddress"),
        field: "ipAddress",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "select",
        label: $t("pages.login_audit_log.actionType"),
        field: "actionType",
        attrs: { placeholder: $t("common.placeholder.select"), clearable: true, filterable: true },
        options: loginAuditLogActionTypeList.value,
      },
      {
        type: "select",
        label: $t("pages.login_audit_log.riskLevel"),
        field: "riskLevel",
        attrs: { placeholder: $t("common.placeholder.select"), clearable: true, filterable: true },
        options: loginAuditLogRiskLevelList.value,
      },
      {
        type: "select",
        label: $t("pages.login_audit_log.status"),
        field: "status",
        attrs: { placeholder: $t("common.placeholder.select"), clearable: true, filterable: true },
        options: loginAuditLogStatusList.value,
      },
      {
        type: "date-picker",
        label: $t("pages.login_audit_log.createdAt"),
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
    listAction: async (query: any) => {
      const { page, pageSize, createdAt, ...queryParams } = query;

      let startTime: string | undefined;
      let endTime: string | undefined;
      if (createdAt && Array.isArray(createdAt) && createdAt.length === 2) {
        startTime = dayjs(createdAt[0]).format("YYYY-MM-DD HH:mm:ss");
        endTime = dayjs(createdAt[1]).format("YYYY-MM-DD HH:mm:ss");
      }

      const result = await fetchListLoginAuditLogs(
        new PaginationQuery({
          paging: { page: page || 1, pageSize: pageSize || 10 },
          formValues: {
            username: queryParams.username,
            ipAddress: queryParams.ipAddress,
            status: queryParams.status,
            actionType: queryParams.actionType,
            riskType: queryParams.riskLevel,
            created_at__gte: startTime,
            created_at__lte: endTime,
          },
          orderBy: ["-created_at"],
        })
      );
      return { items: result.items || [], total: result.total || 0 };
    },
    toolbar: [],
    toolbarRight: [],
    defaultToolbar: ["refresh", "exports", "filter"],
    tableAttrs: { border: true, stripe: false },
    columns: [
      {
        prop: "createdAt",
        label: $t("pages.login_audit_log.createdAt"),
        minWidth: 160,
        cellType: "date",
        dateFormat: "YYYY-MM-DD HH:mm:ss",
      },
      { prop: "status", label: $t("pages.login_audit_log.status"), width: 120, slotName: "status" },
      { prop: "username", label: $t("pages.login_audit_log.username"), minWidth: 120 },
      {
        prop: "actionType",
        label: $t("pages.login_audit_log.actionType"),
        width: 140,
        slotName: "actionType",
      },
      {
        prop: "riskLevel",
        label: $t("pages.login_audit_log.riskLevel"),
        width: 120,
        slotName: "riskLevel",
      },
      {
        prop: "deviceInfo.platform",
        label: $t("pages.login_audit_log.platform"),
        minWidth: 150,
        slotName: "platform",
      },
      {
        prop: "geoLocation",
        label: $t("pages.login_audit_log.geoLocation"),
        minWidth: 150,
        slotName: "geoLocation",
      },
      { prop: "ipAddress", label: $t("pages.login_audit_log.ipAddress"), width: 140 },
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
