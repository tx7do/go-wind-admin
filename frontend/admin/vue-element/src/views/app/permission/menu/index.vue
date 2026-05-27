<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage
      ref="pageRef"
      :config="pageConfig"
      @add="handleAdd"
      @edit="handleEdit"
      @toolbar="handleToolbar"
    >
      <!-- 标题 -->
      <template #title="scope: any">
        <div class="flex w-full items-center gap-1">
          <div class="size-5 flex-shrink-0">
            <IconifyIcon
              v-if="scope.row.type === 'BUTTON'"
              icon="carbon:security"
              class="size-full"
            />
            <IconifyIcon
              v-else-if="scope.row.meta?.icon"
              :icon="scope.row.meta?.icon || 'carbon:circle-dash'"
              class="size-full"
            />
          </div>
          <span class="flex-auto">{{ $t(scope.row.meta?.title) }}</span>
        </div>
      </template>

      <!-- 类型 -->
      <template #type="scope: any">
        <ElTag size="small" effect="dark" round :color="menuTypeToColor(scope.row.type)">
          {{ menuTypeToName(scope.row.type) }}
        </ElTag>
      </template>

      <!-- 状态 -->
      <template #status="scope: any">
        <ElTag size="small" effect="dark" round :color="statusToColor(scope.row.status)">
          {{ statusToName(scope.row.status) }}
        </ElTag>
      </template>

      <!-- 权限标识 -->
      <template #authority="scope: any">
        <ElTag
          v-for="auth in normalizeAuthority(scope.row.meta?.authority)"
          :key="auth"
          class="mb-1 mr-1"
          :style="{
            backgroundColor: getRandomColor(auth),
            color: '#333',
            border: 'none',
          }"
        >
          {{ auth }}
        </ElTag>
      </template>
    </ProPage>

    <!-- 新增/编辑抽屉 -->
    <MenuDrawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import { ElTag } from "element-plus";
import { Icon as IconifyIcon } from "@iconify/vue";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig, ToolsButton } from "@/components/Pro/ProPage/types";
import MenuDrawer from "./menu-drawer.vue";

import {
  buildMenuTree,
  menuTypeToColor,
  menuTypeToName,
  statusList,
  statusToColor,
  statusToName,
  fetchListMenus,
  useDeleteMenu,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { getRandomColor } from "@/utils/color";
import { $t } from "@/i18n";

const { mutateAsync: deleteMenu } = useDeleteMenu();

const pageRef = ref();
const drawerRef = ref();

const pageConfig = computed<ProPageConfig>(() => ({

  search: {
    grid: true,
    fields: [
      {
        type: "input",
        label: $t("pages.menu.name"),
        field: "name",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "select",
        label: $t("common.table.status"),
        field: "status",
        attrs: {
          placeholder: $t("common.placeholder.select"),
          clearable: true,
          filterable: true,
        },
        options: statusList.value,
      },
    ],
  },

  table: {
    listAction: async (query: any) => {
      const { page, pageSize, ...queryParams } = query;
      const result = await fetchListMenus(
        new PaginationQuery({
          paging: { page: page || 1, pageSize: pageSize || 10 },
          formValues: {
            "meta.title": queryParams.name,
            status: queryParams.status,
          },
          orderBy: ["id"],
        })
      );
      return { items: buildMenuTree(result.items || []), total: result.total || 0 };
    },
    deleteAction: async (ids: string) => {
      await deleteMenu({ id: ids as any });
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
    defaultToolbar: ["refresh", "filter"],
    pagination: false,
    tableAttrs: {
      border: true,
      stripe: false,
      "tree-config": {
        parentField: "parentId",
        rowField: "id",
        expandAll: true,
      },
    },
    columns: [
      { type: "index", label: $t("common.table.seq"), width: 60 },
      {
        prop: "meta.title",
        label: $t("pages.menu.name"),
        minWidth: 180,
        fixed: "left",
        treeNode: true,
        slotName: "title",
      },
      {
        prop: "type",
        label: $t("pages.menu.type"),
        width: 95,
        slotName: "type",
      },
      {
        prop: "meta.authority",
        label: $t("pages.menu.authority"),
        minWidth: 150,
        slotName: "authority",
      },
      { prop: "path", label: $t("pages.menu.path"), minWidth: 150 },
      { prop: "component", label: $t("pages.menu.component"), minWidth: 150 },
      {
        prop: "status",
        label: $t("common.table.status"),
        width: 95,
        slotName: "status",
      },
      { prop: "meta.order", label: $t("common.table.sortOrder"), width: 70 },
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
  drawerRef.value?.open();
}

function handleEdit(row: any) {
  drawerRef.value?.open(row);
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

function normalizeAuthority(authority: unknown): string[] {
  if (Array.isArray(authority)) return authority;
  if (typeof authority === "string") {
    return authority
      .split(",")
      .map((s) => s.trim())
      .filter(Boolean);
  }
  return [];
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
