<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig">
      <!-- 是否成功 -->
      <template #success="scope: any">
        <ElTag size="small" effect="dark" round :color="successToColor(scope.row.success)">
          {{ successToNameWithStatusCode(scope.row.success, scope.row.statusCode) }}
        </ElTag>
      </template>

      <!-- 操作类型 -->
      <template #action="scope: any">
        <ElTag
          size="small"
          effect="dark"
          round
          :color="operationAuditLogActionToColor(scope.row.action)"
        >
          {{ operationAuditLogActionToName(scope.row.action) }}
        </ElTag>
      </template>

      <!-- 地理位置 -->
      <template #geoLocation="scope: any">
        {{ scope.row.geoLocation?.province }} {{ scope.row.geoLocation?.city }}
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
  operationAuditLogActionList,
  operationAuditLogActionToColor,
  operationAuditLogActionToName,
  successStatusList,
  successToColor,
  successToNameWithStatusCode,
  fetchListOperationAuditLogs,
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
        label: $t("pages.operation_audit_log.username"),
        field: "username",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "input",
        label: $t("pages.operation_audit_log.resourceType"),
        field: "resourceType",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "select",
        label: $t("pages.operation_audit_log.action"),
        field: "action",
        attrs: { placeholder: $t("common.placeholder.select"), clearable: true, filterable: true },
        options: operationAuditLogActionList.value,
      },
      {
        type: "input",
        label: $t("pages.operation_audit_log.ipAddress"),
        field: "ipAddress",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "select",
        label: $t("pages.operation_audit_log.success"),
        field: "success",
        attrs: { placeholder: $t("common.placeholder.select"), clearable: true, filterable: true },
        options: successStatusList.value,
      },
      {
        type: "date-picker",
        label: $t("pages.operation_audit_log.createdAt"),
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

      const result = await fetchListOperationAuditLogs(
        new PaginationQuery({
          paging: { page: page || 1, pageSize: pageSize || 10 },
          formValues: {
            username: queryParams.username,
            resourceType: queryParams.resourceType,
            action: queryParams.action,
            ipAddress: queryParams.ipAddress,
            success: queryParams.success,
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
        label: $t("pages.operation_audit_log.createdAt"),
        minWidth: 160,
        cellType: "date",
        dateFormat: "YYYY-MM-DD HH:mm:ss",
      },
      {
        prop: "success",
        label: $t("pages.operation_audit_log.success"),
        width: 120,
        slotName: "success",
      },
      {
        prop: "action",
        label: $t("pages.operation_audit_log.action"),
        width: 120,
        slotName: "action",
      },
      { prop: "resourceType", label: $t("pages.operation_audit_log.resourceType"), minWidth: 150 },
      { prop: "resourceId", label: $t("pages.operation_audit_log.resourceId"), minWidth: 150 },
      { prop: "requestId", label: $t("pages.operation_audit_log.requestId"), minWidth: 180 },
      { prop: "username", label: $t("pages.operation_audit_log.username"), minWidth: 120 },
      {
        prop: "geoLocation",
        label: $t("pages.operation_audit_log.geoLocation"),
        minWidth: 150,
        slotName: "geoLocation",
      },
      { prop: "ipAddress", label: $t("pages.operation_audit_log.ipAddress"), width: 140 },
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
