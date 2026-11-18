<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <!-- 左树右表（issue #5）：左栏机构树导航，右栏组织列表按 parentId 过滤 -->
    <ProSplitter :left-size="25" :left-min-size="20" :left-max-size="40" :right-size="75">
      <template #left>
        <OrgList v-model:selected-keys="selectedOrgKeys" @org-select="handleOrgSelect" />
      </template>
      <template #right>
        <ProPage
          ref="pageRef"
          :config="pageConfig"
          @add="handleAdd"
          @edit="handleEdit"
          @toolbar="handleToolbar"
        >
      <!-- 类型 -->
      <template #type="scope: any">
        <ElTag size="small" round :type="orgUnitTypeToType(scope.row.type)">
          {{ orgUnitTypeToName(scope.row.type) }}
        </ElTag>
      </template>

        <!-- 状态 -->
        <template #status="scope: any">
          <ElTag size="small" effect="plain" round :type="orgUnitStatusToType(scope.row.status)">
            {{ orgUnitStatusToName(scope.row.status) }}
          </ElTag>
        </template>
        </ProPage>
      </template>
    </ProSplitter>

    <!-- 新增/编辑抽屉 -->
    <OrgDrawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import { ElTag } from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig, ToolsButton } from "@/components/Pro/ProPage/types";
import ProSplitter from "@/components/Pro/ProSplitter/index.vue";
import OrgDrawer from "./org-drawer.vue";
import OrgList from "../user/list/org-list.vue";

import {
  orgUnitStatusToType,
  orgUnitStatusToName,
  orgUnitTypeListForQuery,
  orgUnitTypeToType,
  orgUnitTypeToName,
  fetchListOrgUnits,
  useDeleteOrgUnit,
} from "@/api/composables";
import { PaginationQuery } from "@go-wind/admin-core";
import { $t } from "@go-wind/admin-element";

const { mutateAsync: deleteOrgUnit } = useDeleteOrgUnit();

const pageRef = ref();
const drawerRef = ref();

// 左树选中（issue #5）：键与树同构 `${parentId}-${id}`；右表按 parentId
// 精确过滤（_id 后缀键不加 __contains，走 exact）。
const selectedOrgKeys = ref<(number | string)[]>([]);
const selectedOrgNode = ref<any | null>(null);
function handleOrgSelect(node: any) {
  selectedOrgNode.value = node ?? null;
  pageRef.value?.refresh?.();
}

const pageConfig = computed<ProPageConfig>(() => ({
  skeleton: true,
  search: {
    grid: true,
    fields: [
      {
        type: "input",
        label: $t("pages.org_unit.name"),
        field: "name",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "select",
        label: $t("common.table.status"),
        field: "status",
        attrs: { placeholder: $t("common.placeholder.select"), clearable: true },
        options: [
          { value: "ON", label: $t("enum.status.ON") },
          { value: "OFF", label: $t("enum.status.OFF") },
        ],
      },
      {
        type: "select",
        label: $t("pages.org_unit.type"),
        field: "type",
        attrs: { placeholder: $t("common.placeholder.select"), clearable: true },
        options: orgUnitTypeListForQuery.value,
      },
    ],
  },

  table: {
    listAction: async (query: any) => {
      const { page, pageSize, ...queryParams } = query;
      // 左树选中注入 parentId（_id 后缀键走精确匹配）
      if (selectedOrgNode.value?.id) {
        queryParams.parentId = String(selectedOrgNode.value.id);
      }
      const result = await fetchListOrgUnits(
        new PaginationQuery({
          paging: { page: page || 1, pageSize: pageSize || 100 },
          formValues: queryParams,
        })
      );
      return { items: result.items || [], total: result.total || 0 };
    },
    deleteAction: async (ids: string) => {
      await deleteOrgUnit({ id: ids as any });
    },
    toolbar: [
      {
        name: "expandAll",
        label: $t("common.tree.expand_all"),
        attrs: { icon: "SortDown" },
      } as ToolsButton,
      {
        name: "collapseAll",
        label: $t("common.tree.collapse_all"),
        attrs: { icon: "SortUp" },
      } as ToolsButton,
    ],
    toolbarRight: ["add"],
    defaultToolbar: ["exports", "refresh", "zoom", "filter"],
    tableAttrs: {
      border: true,
      stripe: false,
      "tree-config": {
        expandAll: true,
        rowField: "id",
        childrenField: "children",
      },
    },
    columns: [
      { prop: "name", label: $t("pages.org_unit.name"), minWidth: 150, treeNode: true },
      { prop: "code", label: $t("pages.org_unit.code"), minWidth: 120 },
      {
        prop: "type",
        label: $t("pages.org_unit.type"),
        minWidth: 100,
        slotName: "type",
      },
      { prop: "leaderName", label: $t("pages.org_unit.leaderName"), minWidth: 100 },
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

function handleToolbar(name: string) {
  const vxeTable = pageRef.value?.tableRef?.tableRef;
  if (!vxeTable) return;
  if (name === "expandAll") {
    vxeTable.setAllTreeExpand(true);
  } else if (name === "collapseAll") {
    vxeTable.clearTreeExpand();
  }
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
