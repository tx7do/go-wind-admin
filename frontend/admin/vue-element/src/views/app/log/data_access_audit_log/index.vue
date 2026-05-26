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
      <!-- 是否成功 -->
      <template #success="{ row }">
        <ElTag size="small" effect="dark" round :color="successToColor(row.success)">
          {{ successToNameWithStatusCode(row.success, row.statusCode) }}
        </ElTag>
      </template>

      <!-- 访问类型 -->
      <template #accessType="{ row }">
        <ElTag
          size="small"
          effect="dark"
          round
          :color="dataAccessAuditLogAccessTypeToColor(row.accessType)"
        >
          {{ dataAccessAuditLogAccessTypeToName(row.accessType) }}
        </ElTag>
      </template>

      <!-- 地理位置 -->
      <template #geoLocation="{ row }">
        {{ row.geoLocation?.province }} {{ row.geoLocation?.city }}
      </template>
    </PageContent>
  </div>
</template>

<script lang="ts" setup>
import { ElTag } from "element-plus";
import dayjs from "dayjs";

import PageContent from "@/components/CURD/PageContent.vue";
import PageSearch from "@/components/CURD/PageSearch.vue";
import usePage from "@/components/CURD/usePage";
import type { ISearchConfig, IContentConfig } from "@/components/CURD/types";

import {
  dataAccessAuditLogAccessTypeList,
  dataAccessAuditLogAccessTypeToColor,
  dataAccessAuditLogAccessTypeToName,
  successStatusList,
  successToColor,
  successToNameWithStatusCode,
  fetchListDataAccessAuditLogs,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/i18n";

// 使用 CURD hook
const { searchRef, contentRef, handleQueryClick, handleResetClick } = usePage();

// 搜索配置
const searchConfig: ISearchConfig = {
  grid: true, // 启用 Grid 布局
  formItems: [
    {
      type: "input",
      label: $t("pages.data_access_audit_log.username"),
      prop: "username",
      attrs: {
        placeholder: $t("common.placeholder.input"),
        clearable: true,
      },
    },
    {
      type: "input",
      label: $t("pages.data_access_audit_log.tableName"),
      prop: "tableName",
      attrs: {
        placeholder: $t("common.placeholder.input"),
        clearable: true,
      },
    },
    {
      type: "select",
      label: $t("pages.data_access_audit_log.accessType"),
      prop: "accessType",
      attrs: {
        placeholder: $t("common.placeholder.select"),
        clearable: true,
        filterable: true,
      },
      options: dataAccessAuditLogAccessTypeList.value,
    },
    {
      type: "input",
      label: $t("pages.data_access_audit_log.ipAddress"),
      prop: "ipAddress",
      attrs: {
        placeholder: $t("common.placeholder.input"),
        clearable: true,
      },
    },
    {
      type: "select",
      label: $t("pages.data_access_audit_log.success"),
      prop: "success",
      attrs: {
        placeholder: $t("common.placeholder.select"),
        clearable: true,
        filterable: true,
      },
      options: successStatusList.value,
    },
    {
      type: "date-picker",
      label: $t("pages.data_access_audit_log.createdAt"),
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
  permPrefix: "sys:data_access_audit_log", // 数据访问审计日志权限前缀
  toolbarRight: [], // 无自定义按钮
  defaultToolbar: ["refresh", "exports", "filter"], // 右侧默认工具栏
  table: {
    border: true,
    stripe: false,
  },
  indexAction: async (query: any) => {
    const { page, pageSize, createdAt, ...queryParams } = query;

    let startTime: string | undefined;
    let endTime: string | undefined;
    if (createdAt && Array.isArray(createdAt) && createdAt.length === 2) {
      startTime = dayjs(createdAt[0]).format("YYYY-MM-DD HH:mm:ss");
      endTime = dayjs(createdAt[1]).format("YYYY-MM-DD HH:mm:ss");
    }

    const result = await fetchListDataAccessAuditLogs(
      new PaginationQuery({
        paging: { page: page || 1, pageSize: pageSize || 10 },
        formValues: {
          username: queryParams.username,
          accessType: queryParams.accessType,
          tableName: queryParams.tableName,
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
  columns: [
    {
      prop: "createdAt",
      label: $t("pages.data_access_audit_log.createdAt"),
      minWidth: 160,
      template: "date",
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
    { prop: "dataCategory", label: $t("pages.data_access_audit_log.dataCategory"), minWidth: 150 },
    { prop: "latencyMs", label: $t("pages.data_access_audit_log.latencyMs"), width: 120 },
    { prop: "username", label: $t("pages.data_access_audit_log.username"), minWidth: 120 },
    {
      prop: "geoLocation",
      label: $t("pages.data_access_audit_log.geoLocation"),
      minWidth: 150,
      slotName: "geoLocation",
    },
    { prop: "ipAddress", label: $t("pages.data_access_audit_log.ipAddress"), width: 140 },
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
