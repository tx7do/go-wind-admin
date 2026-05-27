<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig">
      <!-- 成功状态 -->
      <template #success="scope: any">
        <ElTag size="small" effect="dark" round :color="successToColor(scope.row.success)">
          {{ successToNameWithStatusCode(scope.row.success, scope.row.statusCode) }}
        </ElTag>
      </template>

      <!-- 地理位置 -->
      <template #geoLocation="scope: any">
        {{ scope.row.geoLocation?.province || "" }} {{ scope.row.geoLocation?.city || "" }}
      </template>

      <!-- 平台信息 -->
      <template #platform="scope: any">
        {{ scope.row.deviceInfo?.osName || "" }} {{ scope.row.deviceInfo?.browserName || "" }}
      </template>
    </ProPage>
  </div>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import { ElTag } from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";

import { $t } from "@/i18n";
import {
  methodList,
  successStatusList,
  successToColor,
  successToNameWithStatusCode,
  fetchListApiAuditLogs,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import dayjs from "dayjs";

const props = defineProps({
  userId: {
    type: Number,
    default: undefined,
  },
});

const pageRef = ref();

const pageConfig = computed<ProPageConfig>(() => ({

  search: {
    grid: true,
    fields: [
      {
        type: "input",
        label: $t("pages.api_audit_log.path"),
        field: "path",
        attrs: {
          placeholder: $t("common.placeholder.input"),
          clearable: true,
        },
      },
      {
        type: "select",
        label: $t("pages.api_audit_log.httpMethod"),
        field: "httpMethod",
        attrs: {
          placeholder: $t("common.placeholder.select"),
          clearable: true,
        },
        options: methodList,
      },
      {
        type: "input",
        label: $t("pages.api_audit_log.ipAddress"),
        field: "ipAddress",
        attrs: {
          placeholder: $t("common.placeholder.input"),
          clearable: true,
        },
      },
      {
        type: "select",
        label: $t("pages.api_audit_log.success"),
        field: "success",
        attrs: {
          placeholder: $t("common.placeholder.select"),
          clearable: true,
        },
        options: successStatusList.value,
      },
      {
        type: "date-picker",
        label: $t("pages.api_audit_log.createdAt"),
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
      const { page, pageSize, ...queryParams } = query;

      let startTime: string | undefined;
      let endTime: string | undefined;

      if (
        queryParams.createdAt !== undefined &&
        Array.isArray(queryParams.createdAt) &&
        queryParams.createdAt.length === 2
      ) {
        startTime = dayjs(queryParams.createdAt[0]).format("YYYY-MM-DD HH:mm:ss");
        endTime = dayjs(queryParams.createdAt[1]).format("YYYY-MM-DD HH:mm:ss");
      }

      const result = await fetchListApiAuditLogs(
        new PaginationQuery({
          paging: { page: page || 1, pageSize: pageSize || 10 },
          formValues: {
            user_id: props.userId?.toString(),
            httpMethod: queryParams.httpMethod,
            path: queryParams.path,
            ipAddress: queryParams.ipAddress,
            success: queryParams.success,
            created_at__gte: startTime,
            created_at__lte: endTime,
          },
          orderBy: ["-created_at"],
        })
      );

      return {
        items: result.items || [],
        total: result.total || 0,
      };
    },
    toolbarRight: [],
    defaultToolbar: ["refresh", "filter"],
    tableAttrs: { border: true, stripe: true, height: "auto" },
    columns: [
      {
        prop: "createdAt",
        label: $t("pages.api_audit_log.createdAt"),
        width: 160,
        cellType: "date",
        dateFormat: "YYYY-MM-DD HH:mm:ss",
      },
      {
        prop: "success",
        label: $t("pages.api_audit_log.success"),
        width: 100,
        slotName: "success",
      },
      {
        prop: "username",
        label: $t("pages.api_audit_log.username"),
        minWidth: 120,
      },
      {
        prop: "httpMethod",
        label: $t("pages.api_audit_log.httpMethod"),
        width: 90,
      },
      {
        prop: "path",
        label: $t("pages.api_audit_log.path"),
        minWidth: 200,
        align: "left",
      },
      {
        prop: "latencyMs",
        label: $t("pages.api_audit_log.latencyMs"),
        width: 120,
      },
      {
        prop: "platform",
        label: $t("pages.api_audit_log.platform"),
        minWidth: 150,
        slotName: "platform",
      },
      {
        prop: "geoLocation",
        label: $t("pages.api_audit_log.geoLocation"),
        minWidth: 150,
        slotName: "geoLocation",
      },
      {
        prop: "ipAddress",
        label: $t("pages.api_audit_log.ipAddress"),
        width: 140,
      },
    ],
  },
}));
</script>

<style lang="scss" scoped>
.app-container {
  height: 100%;
  padding: 0;
  width: 100%;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
}
</style>
