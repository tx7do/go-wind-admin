<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <!-- 搜索 -->
    <PageSearch
      ref="searchRef"
      :search-config="searchConfig"
      @query-click="handleQueryClick"
      @reset-click="handleResetClick"
    />

    <!-- 列表 -->
    <PageContent ref="contentRef" :content-config="contentConfig">
      <!-- 成功状态 -->
      <template #success="{ row }">
        <ElTag size="small" effect="dark" round :color="successToColor(row.success)">
          {{ successToNameWithStatusCode(row.success, row.statusCode) }}
        </ElTag>
      </template>

      <!-- 地理位置 -->
      <template #geoLocation="{ row }">
        {{ row.geoLocation?.province || "" }} {{ row.geoLocation?.city || "" }}
      </template>

      <!-- 平台信息 -->
      <template #platform="{ row }">
        {{ row.deviceInfo?.osName || "" }} {{ row.deviceInfo?.browserName || "" }}
      </template>
    </PageContent>
  </div>
</template>

<script lang="ts" setup>
import { ElTag } from "element-plus";

import PageContent from "@/components/CURD/PageContent.vue";
import PageSearch from "@/components/CURD/PageSearch.vue";
import usePage from "@/components/CURD/usePage";
import type { ISearchConfig, IContentConfig } from "@/components/CURD/types";

import { $t } from "@/i18n";
import {
  methodList,
  successStatusList,
  successToColor,
  successToNameWithStatusCode,
  useApiAuditLogStore,
} from "@/stores";
import dayjs from "dayjs";

const props = defineProps({
  userId: {
    type: Number,
    default: undefined,
  },
});

const apiAuditLogStore = useApiAuditLogStore();

// 使用 CURD hook
const { searchRef, contentRef, handleQueryClick, handleResetClick } = usePage();

// 搜索配置
const searchConfig: ISearchConfig = {
  grid: true,
  formItems: [
    {
      type: "input",
      label: $t("pages.api_audit_log.path"),
      prop: "path",
      attrs: {
        placeholder: $t("common.placeholder.input"),
        clearable: true,
      },
    },
    {
      type: "select",
      label: $t("pages.api_audit_log.httpMethod"),
      prop: "httpMethod",
      attrs: {
        placeholder: $t("common.placeholder.select"),
        clearable: true,
      },
      options: methodList,
    },
    {
      type: "input",
      label: $t("pages.api_audit_log.ipAddress"),
      prop: "ipAddress",
      attrs: {
        placeholder: $t("common.placeholder.input"),
        clearable: true,
      },
    },
    {
      type: "select",
      label: $t("pages.api_audit_log.success"),
      prop: "success",
      attrs: {
        placeholder: $t("common.placeholder.select"),
        clearable: true,
      },
      options: successStatusList.value,
    },
    {
      type: "date-picker",
      label: $t("pages.api_audit_log.createdAt"),
      prop: "createdAt",
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
};

// 表格配置
const contentConfig: IContentConfig = {
  permPrefix: "sys:api_audit_log",
  toolbarRight: [],
  defaultToolbar: ["refresh", "filter"],
  table: {
    border: true,
    stripe: true,
    height: "auto",
  },
  indexAction: async (query: any) => {
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

    const result = await apiAuditLogStore.listApiAuditLog(
      {
        page: page || 1,
        pageSize: pageSize || 10,
      },
      {
        user_id: props.userId?.toString(),
        httpMethod: queryParams.httpMethod,
        path: queryParams.path,
        ipAddress: queryParams.ipAddress,
        success: queryParams.success,
        created_at__gte: startTime,
        created_at__lte: endTime,
      },
      null,
      ["-created_at"]
    );

    return {
      items: result.items || [],
      total: result.total || 0,
    };
  },
  columns: [
    {
      prop: "createdAt",
      label: $t("pages.api_audit_log.createdAt"),
      width: 160,
      template: "date",
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
};
</script>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
  width: 100%;
  min-width: 0;
  flex-shrink: 0;
}
</style>
