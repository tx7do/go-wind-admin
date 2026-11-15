<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig" @add="handleAdd" @edit="handleEdit">
      <!-- 类型 -->
      <template #type="scope: any">
        <ElTag size="small" round :type="positionTypeToType(scope.row.type)">
          {{ positionTypeToName(scope.row.type) }}
        </ElTag>
      </template>

      <!-- 状态 -->
      <template #status="scope: any">
        <ElTag size="small" effect="plain" round :type="statusToType(scope.row.status)">
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
  positionTypeToType,
  positionTypeToName,
  statusList,
  statusToType,
  statusToName,
  fetchListOrgUnits,
  fetchListPositions,
  useCreatePosition,
  useDeletePosition,
  createImportsAction,
  generateImportTemplate,
} from "@/api/composables";
import { PaginationQuery } from "@go-wind/admin-core";
import { $t } from "@go-wind/admin-element";

const { mutateAsync: deletePosition } = useDeletePosition();
const { mutateAsync: createPosition } = useCreatePosition();

const pageRef = ref();
const drawerRef = ref();

const pageConfig = computed<ProPageConfig>(() => {
  // Excel 导入字段清单：与抽屉表单一致；orgUnit 以"组织名称"列承载，
  // 导入期解析为 orgUnitId（见下方 createRow 包装）。
  const orgUnitList = ref<any[]>([]);
  const importFields = [
    { label: $t("pages.position.name"), prop: "name" },
    { label: $t("pages.position.code"), prop: "code" },
    { label: $t("pages.position.type"), prop: "type" },
    { label: $t("common.table.status"), prop: "status" },
    { label: $t("pages.position.headcount"), prop: "headcount" },
    { label: $t("common.table.sortOrder"), prop: "sortOrder" },
    { label: $t("pages.position.description"), prop: "description" },
    { label: $t("pages.position.orgUnit"), prop: "__orgUnitName" },
    { label: $t("common.table.remark"), prop: "remark" },
  ];
  return {
  skeleton: true,
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
    importsAction: createImportsAction(importFields, async (values) => {
      // 解析组织名称 → orgUnitId：按名称过滤拉取后精确匹配，未命中视为行级错误
      const orgName = values.__orgUnitName;
      delete values.__orgUnitName;
      if (orgName && String(orgName).trim() !== "") {
        if (orgUnitList.value.length === 0) {
          const result = await fetchListOrgUnits(
            new PaginationQuery({
              paging: { page: 1, pageSize: 500 },
              formValues: { status: "ON" },
            }),
          );
          orgUnitList.value = result.items || [];
        }
        const hit = orgUnitList.value.find(
          (u: any) => u.name === String(orgName).trim(),
        );
        if (!hit) {
          throw new Error(`${$t("pages.position.orgUnit")}: ${orgName}`);
        }
        values.orgUnitId = hit.id;
      }
      return createPosition(values);
    }),
    importTemplate: () => generateImportTemplate(importFields),
    toolbar: [],
    toolbarRight: ["add"],
    defaultToolbar: ["exports", "imports", "refresh", "zoom", "filter"],
    tableAttrs: { border: true, stripe: true },
    emptyActionText: "common.button.add",
    columns: [
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
      { prop: "headcount", label: $t("pages.position.headcount"), width: 80, align: "right" },
      {
        prop: "status",
        label: $t("common.table.status"),
        minWidth: 100,
        slotName: "status",
      },
      { prop: "sortOrder", label: $t("common.table.sortOrder"), width: 80, align: "right" },
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
          { name: "edit", label: $t("common.button.edit"), icon: "lucide:pen-line" },
          {
            name: "delete",
            label: $t("common.button.delete"),
            icon: "lucide:trash-2",
            attrs: { type: "danger" },
          },
        ],
      },
    ],
  },
  };
});

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
