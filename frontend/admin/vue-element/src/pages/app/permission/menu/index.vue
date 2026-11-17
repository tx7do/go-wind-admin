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
          <!-- meta.title 可为空（BUTTON 型菜单常无 meta）：$t(undefined) 会抛
               SyntaxError: Invalid arguments 打崩 VxeTableBody 渲染（整页表格
               空白 + tableBodyElem.querySelector 连锁报错）。空标题回退路由名，
               与 vben 基准端同款守卫。 -->
          <span class="flex-auto">
            {{ scope.row.meta?.title ? $t(scope.row.meta.title) : (scope.row.name ?? "-") }}
          </span>
        </div>
      </template>

      <!-- 类型 -->
      <template #type="scope: any">
        <ElTag size="small" round :type="menuTypeToType(scope.row.type)">
          {{ menuTypeToName(scope.row.type) }}
        </ElTag>
      </template>

      <!-- 状态 -->
      <template #status="scope: any">
        <ElTag size="small" effect="plain" round :type="statusToType(scope.row.status)">
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
import { ElMessage, ElMessageBox, ElTag } from "element-plus";
import { Icon as IconifyIcon } from "@iconify/vue";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig, ToolsButton } from "@/components/Pro/ProPage/types";
import MenuDrawer from "./menu-drawer.vue";

import {
  buildMenuTree,
  buildSyncMenusRequest,
  menuTypeToType,
  menuTypeToName,
  statusList,
  statusToType,
  statusToName,
  fetchListMenus,
  useDeleteMenu,
  useSyncMenus,
} from "@/api/composables";
import { PaginationQuery } from "@go-wind/admin-core";
import { getRandomColor } from "@go-wind/admin-core";
import { $t } from "@go-wind/admin-element";
import { accessRoutes } from "@/router/routes";

const { mutateAsync: deleteMenu } = useDeleteMenu();
const { mutateAsync: syncMenus, isPending: isSyncing } = useSyncMenus();

/**
 * 菜单标题关键词的前端过滤。
 * 标题没法在服务端搜：库里 meta.title 存的是 i18n key 或源语言文本，用户看到的
 * 译文只有前端知道；且后端 go-crud 对 meta.* JSONB 路径的过滤条件值绑定有缺陷
 * （值被丢成空串，CONTAINS 恒真/EQ 恒假）。因此按「所见即所搜」在前端匹配：
 * 译文 + 库里原文 + 路由名三路 contains；命中行连同祖先链一起保留，
 * 才能还原完整树形。
 */
function filterMenusByTitle(items: any[], keyword?: string): any[] {
  const kw = (keyword ?? "").trim().toLowerCase();
  if (!kw) return items;
  const byId = new Map<number, any>(items.map((item) => [item.id as number, item]));
  const keep = new Set<number>();
  for (const item of items) {
    const title = item.meta?.title ?? "";
    const haystack = [$t(title), title, item.name ?? ""].join("\n").toLowerCase();
    if (!haystack.includes(kw)) continue;
    keep.add(item.id as number);
    let parentId = item.parentId;
    while (parentId != null && !keep.has(parentId)) {
      keep.add(parentId);
      parentId = byId.get(parentId)?.parentId;
    }
  }
  return items.filter((item) => keep.has(item.id as number));
}

const pageRef = ref();
const drawerRef = ref();

const pageConfig = computed<ProPageConfig>(() => ({
  skeleton: true,
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
    exportFilename: "menus",
    listAction: async (query: any) => {
      const { page, pageSize, ...queryParams } = query;
      // 标题走前端过滤（见 filterMenusByTitle），服务端只收 status
      const result = await fetchListMenus(
        new PaginationQuery({
          paging: { page: page || 1, pageSize: pageSize || 1000 },
          formValues: {
            status: queryParams.status,
          },
          orderBy: ["id"],
        })
      );
      const items = filterMenusByTitle(result.items || [], queryParams.name);
      return { items: buildMenuTree(items), total: items.length };
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
      {
        name: "syncMenus",
        label: $t("pages.menu.sync"),
        attrs: { type: "warning", plain: true, icon: "Refresh", loading: isSyncing.value },
      } as ToolsButton,
    ],
    toolbarRight: ["add"],
    defaultToolbar: ["exports", "refresh", "zoom", "filter"],
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
  drawerRef.value?.open();
}

function handleEdit(row: any) {
  drawerRef.value?.open(row);
}

function handleSuccess() {
  pageRef.value?.refresh();
}

function handleToolbar(name: string) {
  if (name === "syncMenus") {
    handleSyncMenus();
    return;
  }
  const vxeTable = pageRef.value?.tableRef?.tableRef;
  if (!vxeTable) return;
  if (name === "expandAll") {
    vxeTable.setAllTreeExpand(true);
  } else if (name === "collapseAll") {
    vxeTable.clearTreeExpand();
  }
}

/* 同步菜单：将本地静态路由全量推送到后端（后端会清空重建，角色授权需重做） */
async function handleSyncMenus() {
  try {
    await ElMessageBox.confirm($t("pages.menu.syncConfirm"), $t("pages.menu.sync"), {
      type: "warning",
      confirmButtonText: $t("common.button.confirm"),
      cancelButtonText: $t("common.button.cancel"),
    });
  } catch {
    return; // 用户取消确认框，非错误
  }

  try {
    await syncMenus(buildSyncMenusRequest(accessRoutes));
    ElMessage.success($t("common.notification.syncSuccess"));
    pageRef.value?.refresh();
  } catch (error) {
    console.error("sync menus failed:", error);
    ElMessage.error($t("common.notification.syncFailed"));
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
