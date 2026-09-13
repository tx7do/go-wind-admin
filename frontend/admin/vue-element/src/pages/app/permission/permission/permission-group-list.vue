<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage
      ref="pageRef"
      :config="pageConfig"
      @add="handleAdd"
      @edit="handleEdit"
      @row-click="handleRowClick"
      @toolbar="handleToolbar"
    >
      <!-- 状态 -->
      <template #status="scope: any">
        <ElTag size="small" effect="plain" round :type="statusToType(scope.row.status)">
          {{ statusToName(scope.row.status) }}
        </ElTag>
      </template>
    </ProPage>

    <!-- 新增/编辑抽屉 -->
    <PermissionGroupDrawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import { ElTag } from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig, ToolsButton } from "@/components/Pro/ProPage/types";
import PermissionGroupDrawer from "./permission-group-drawer.vue";

import {
  statusList,
  statusToType,
  statusToName,
  useDeletePermissionGroup,
  createPagedExportAction,
} from "@/api/composables";
import { $t } from "@/core/i18n";
import { usePermissionViewStore } from "@/pages/app/permission/permission/permission-view.state";

const { mutateAsync: deletePermissionGroup } = useDeletePermissionGroup();
const permissionViewStore = usePermissionViewStore();

const pageRef = ref();
const drawerRef = ref();

const pageConfig = computed<ProPageConfig>(() => ({
  skeleton: true,
  search: {
    // 左栏窄容器：grid 列数按容器宽度自适应（inline+minWidth 在窄栏会溢出换行）
    grid: true,
    fields: [
      {
        type: "input",
        label: $t("pages.permission_group.name"),
        field: "name",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "input",
        label: $t("pages.permission_group.module"),
        field: "module",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "select",
        label: $t("common.table.status"),
        field: "status",
        attrs: { placeholder: $t("common.placeholder.select"), clearable: true },
        options: statusList.value,
      },
    ],
  },

  table: {
    listAction: async (query: any) => {
      const { page, pageSize, ...queryParams } = query;
      const result = await permissionViewStore.fetchGroupList(
        page || 1,
        pageSize || 10,
        queryParams
      );
      return { items: result.items || [], total: result.total || 0 };
    },
    deleteAction: async (ids: string) => {
      await deletePermissionGroup({ id: ids as any });
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
    exportsAction: createPagedExportAction(async (query: any) =>
      permissionViewStore.fetchGroupList(
        Number(query.paging?.page) || 1,
        Number(query.paging?.pageSize) || 1000,
        query.formValues ?? {},
      )),
    defaultToolbar: ["refresh", "filter", "exports"],
    tableAttrs: {
      border: true,
      stripe: false,
      height: "auto",
      "show-overflow": "title",
      "show-header-overflow": "title",
      "tree-config": {
        parentField: "parentId",
        rowField: "id",
        expandAll: true,
      },
    },
    // 左栏分栏主表：内容列不设宽度（均分贴合容器防横向溢出），仅操作列定宽
    columns: [
      {
        prop: "name",
        label: $t("pages.permission_group.name"),
        treeNode: true,
        align: "left",
      },
      {
        prop: "module",
        label: $t("pages.permission_group.module"),
        align: "left",
      },
      {
        prop: "status",
        label: $t("common.table.status"),
        slotName: "status",
      },
      {
        prop: "action",
        label: $t("common.table.action"),
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

// 行点击联动 - 切换分组时刷新权限列表
function handleRowClick(row: any) {
  if (row?.id) {
    permissionViewStore.setCurrentGroupId(row.id);
  }
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
  // ProSplitter 面板内必须允许收缩：flex-shrink:0 会让内容超宽时整体
  // 溢出面板（搜索按钮顶进右栏、整页出现横向滚动条）
  flex-shrink: 1;
}
</style>
