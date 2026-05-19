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
      <!-- 状态 -->
      <template #status="{ row }">
        <ElTag
          size="small"
          effect="dark"
          round
          :color="internalMessageRecipientStatusColor(row.status)"
        >
          {{ internalMessageRecipientStatusLabel(row.status) }}
        </ElTag>
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
  internalMessageRecipientStatusColor,
  internalMessageRecipientStatusLabel,
  useInternalMessageStore,
} from "@/stores";
import dayjs from "dayjs";

const props = defineProps({
  userId: {
    type: Number,
    default: undefined,
  },
});

const internalMessageStore = useInternalMessageStore();

// 使用 CURD hook
const { searchRef, contentRef, handleQueryClick, handleResetClick } = usePage();

// 搜索配置
const searchConfig: ISearchConfig = {
  grid: true,
  formItems: [
    {
      type: "date-picker",
      label: $t("common.table.createdAt"),
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
  permPrefix: "sys:internal_message",
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

    const result = await internalMessageStore.listUserInbox(
      {
        page: page || 1,
        pageSize: pageSize || 10,
      },
      {
        recipient_user_id: props.userId?.toString(),
        created_at__gte: startTime,
        created_at__lte: endTime,
      }
    );

    return {
      items: result.items || [],
      total: result.total || 0,
    };
  },
  columns: [
    {
      prop: "title",
      label: $t("pages.internal_message.title"),
      minWidth: 200,
      align: "left",
    },
    {
      prop: "status",
      label: $t("pages.internal_message.status"),
      width: 100,
      slotName: "status",
    },
    {
      prop: "readAt",
      label: $t("pages.internal_message.readAt"),
      width: 160,
      formatter: "formatDateTime",
    },
    {
      prop: "createdAt",
      label: $t("common.table.createdAt"),
      width: 160,
      formatter: "formatDateTime",
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
