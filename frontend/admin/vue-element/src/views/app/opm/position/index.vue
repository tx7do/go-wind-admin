<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig" @add="handleAdd" @edit="handleEdit">
      <!-- 类型 -->
      <template #type="scope: any">
        <ElTag size="small" effect="dark" round :color="positionTypeToColor(scope.row.type)">
          {{ positionTypeToName(scope.row.type) }}
        </ElTag>
      </template>

      <!-- 状态 -->
      <template #status="scope: any">
        <ElTag size="small" effect="dark" round :color="statusToColor(scope.row.status)">
          {{ statusToName(scope.row.status) }}
        </ElTag>
      </template>
    </ProPage>

    <!-- 新增/编辑抽屉 -->
    <PositionDrawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import { ElTag } from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import PositionDrawer from "./position-drawer.vue";

import {
  positionTypeList,
  positionTypeToColor,
  positionTypeToName,
  statusList,
  statusToColor,
  statusToName,
  fetchListOrgUnits,
  fetchListPositions,
  useDeletePosition,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/i18n";

const { mutateAsync: deletePosition } = useDeletePosition();

const pageRef = ref();
const drawerRef = ref();

const pageConfig = computed<ProPageConfig>(() => ({

  search: {
    grid: true,
    fields: [
      {
        type: "input",
        label: $t("pages.position.name"),
        field: "name",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "input",
        label: $t("pages.position.code"),
        field: "code",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "select",
        label: $t("common.table.status"),
        field: "status",
        attrs: { placeholder: $t("common.placeholder.select"), clearable: true },
        options: statusList.value,
      },
      {
        type: "select",
        label: $t("pages.position.type"),
        field: "type",
        attrs: { placeholder: $t("common.placeholder.select"), clearable: true },
        options: positionTypeList.value,
      },
      {
        type: "tree-select",
        label: $t("pages.position.orgUnit"),
        field: "orgUnitId",
        attrs: {
          placeholder: $t("common.placeholder.select"),
          clearable: true,
          filterable: true,
          "default-expand-all": true,
          nodeKey: "id",
          props: {
            label: "name",
            value: "id",
            children: "children",
          },
        },
        initFn: async (item: any) => {
          try {
            const result = await fetchListOrgUnits(
              new PaginationQuery({ formValues: { status: "ON" } })
            );
            item.attrs.data = result.items || [];
          } catch (error) {
            console.error("Failed to load org unit tree:", error);
          }
        },
      },
    ],
  },

  table: {
    listAction: async (query: any) => {
      const { page, pageSize, ...queryParams } = query;
      const result = await fetchListPositions(
        new PaginationQuery({
          paging: { page: page || 1, pageSize: pageSize || 10 },
          formValues: queryParams,
        })
      );
      return { items: result.items || [], total: result.total || 0 };
    },
    deleteAction: async (ids: string) => {
      await deletePosition({ id: ids as any });
    },
    toolbar: [],
    toolbarRight: ["add"],
    defaultToolbar: ["refresh", "filter"],
    tableAttrs: { border: true, stripe: true },
    columns: [
      { type: "index", label: $t("common.table.seq"), width: 60 },
      { prop: "name", label: $t("pages.position.name"), minWidth: 120 },
      { prop: "code", label: $t("pages.position.code"), minWidth: 120 },
      {
        prop: "type",
        label: $t("pages.position.type"),
        minWidth: 100,
        slotName: "type",
      },
      { prop: "description", label: $t("pages.position.description"), minWidth: 150 },
      { prop: "orgUnitName", label: $t("pages.position.orgUnitName"), minWidth: 120 },
      { prop: "headcount", label: $t("pages.position.headcount"), width: 80 },
      {
        prop: "status",
        label: $t("common.table.status"),
        minWidth: 100,
        slotName: "status",
      },
      { prop: "sortOrder", label: $t("common.table.sortOrder"), width: 80 },
      {
        prop: "createdAt",
        label: $t("common.table.createdAt"),
        minWidth: 160,
        cellType: "date",
        dateFormat: "YYYY-MM-DD HH:mm:ss",
      },
      { prop: "remark", label: $t("common.table.remark"), minWidth: 150 },
      {
        prop: "action",
        label: $t("common.table.action"),
        fixed: "right",
        width: 150,
        cellType: "tool",
        buttons: [
          { name: "edit", label: $t("common.button.edit") },
          { name: "delete", label: $t("common.button.delete"), attrs: { type: "danger" } },
        ],
      },
    ],
  },
}));

function handleAdd() {
  drawerRef.value?.open({ create: true });
}

function handleEdit(row: any) {
  drawerRef.value?.open({ create: false, row });
}

function handleSuccess() {
  pageRef.value?.refresh();
}
</script>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
  width: 100%;
  min-width: 0;
  flex-shrink: 0;
}
</style>
