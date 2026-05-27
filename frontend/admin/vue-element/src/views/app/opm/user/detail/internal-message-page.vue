<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig">
      <!-- 状态 -->
      <template #status="scope: any">
        <ElTag
          size="small"
          effect="dark"
          round
          :color="internalMessageRecipientStatusColor(scope.row.status)"
        >
          {{ internalMessageRecipientStatusLabel(scope.row.status) }}
        </ElTag>
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
  internalMessageRecipientStatusColor,
  internalMessageRecipientStatusLabel,
  fetchListUserInbox,
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
        type: "date-picker",
        label: $t("common.table.createdAt"),
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

      const result = await fetchListUserInbox(
        new PaginationQuery({
          paging: { page: page || 1, pageSize: pageSize || 10 },
          formValues: {
            recipient_user_id: props.userId?.toString(),
            created_at__gte: startTime,
            created_at__lte: endTime,
          },
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
        cellType: "date",
        dateFormat: "YYYY-MM-DD HH:mm:ss",
      },
      {
        prop: "createdAt",
        label: $t("common.table.createdAt"),
        width: 160,
        cellType: "date",
        dateFormat: "YYYY-MM-DD HH:mm:ss",
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
