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
      <!-- 操作类型 -->
      <template #action="{ row }">
        <ElTag
          size="small"
          effect="dark"
          round
          :color="permissionAuditLogActionToColor(row.action)"
        >
          {{ permissionAuditLogActionToName(row.action) }}
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
  permissionAuditLogActionList,
  permissionAuditLogActionToColor,
  permissionAuditLogActionToName,
  fetchListPermissionAuditLogs,
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
      label: $t("pages.permission_audit_log.targetType"),
      prop: "targetType",
      attrs: {
        placeholder: $t("common.placeholder.input"),
        clearable: true,
      },
    },
    {
      type: "input",
      label: $t("pages.permission_audit_log.operatorName"),
      prop: "operatorName",
      attrs: {
        placeholder: $t("common.placeholder.input"),
        clearable: true,
      },
    },
    {
      type: "select",
      label: $t("pages.permission_audit_log.action"),
      prop: "action",
      attrs: {
        placeholder: $t("common.placeholder.select"),
        clearable: true,
        filterable: true,
      },
      options: permissionAuditLogActionList.value,
    },
    {
      type: "input",
      label: $t("pages.permission_audit_log.ipAddress"),
      prop: "ipAddress",
      attrs: {
        placeholder: $t("common.placeholder.input"),
        clearable: true,
      },
    },
    {
      type: "date-picker",
      label: $t("pages.permission_audit_log.createdAt"),
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
  permPrefix: "sys:permission_audit_log", // 权限审计日志权限前缀
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

    const result = await fetchListPermissionAuditLogs(
      new PaginationQuery({
        paging: { page: page || 1, pageSize: pageSize || 10 },
        formValues: {
          targetType: queryParams.targetType,
          operatorName: queryParams.operatorName,
          action: queryParams.action,
          ipAddress: queryParams.ipAddress,
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
      label: $t("pages.permission_audit_log.createdAt"),
      minWidth: 160,
      template: "date",
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
    { prop: "ipAddress", label: $t("pages.permission_audit_log.ipAddress"), width: 140 },
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
