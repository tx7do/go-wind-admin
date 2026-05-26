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
    <PageContent
      ref="contentRef"
      :content-config="contentConfig"
      @add-click="handleAddClick"
      @operate-click="handleOperateClick"
      @toolbar-click="handleToolbarClick"
    >
      <!-- 标题 -->
      <template #title="{ row }">
        <div class="flex w-full items-center gap-1">
          <div class="size-5 flex-shrink-0">
            <IconifyIcon v-if="row.type === 'BUTTON'" icon="carbon:security" class="size-full" />
            <IconifyIcon
              v-else-if="row.meta?.icon"
              :icon="row.meta?.icon || 'carbon:circle-dash'"
              class="size-full"
            />
          </div>
          <span class="flex-auto">{{ $t(row.meta?.title) }}</span>
        </div>
      </template>

      <!-- 类型 -->
      <template #type="{ row }">
        <ElTag size="small" effect="dark" round :color="menuTypeToColor(row.type)">
          {{ menuTypeToName(row.type) }}
        </ElTag>
      </template>

      <!-- 状态 -->
      <template #status="{ row }">
        <ElTag size="small" effect="dark" round :color="statusToColor(row.status)">
          {{ statusToName(row.status) }}
        </ElTag>
      </template>

      <!-- 权限标识 -->
      <template #authority="{ row }">
        <ElTag
          v-for="auth in normalizeAuthority(row.meta?.authority)"
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
    </PageContent>

    <!-- 新增/编辑抽屉 -->
    <MenuDrawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ElMessage, ElMessageBox, ElTag } from "element-plus";
import { Icon as IconifyIcon } from "@iconify/vue";

import PageContent from "@/components/CURD/PageContent.vue";
import PageSearch from "@/components/CURD/PageSearch.vue";
import usePage from "@/components/CURD/usePage";
import type { IOperateData, ISearchConfig, IContentConfig } from "@/components/CURD/types";
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

// 使用 CURD hook
const { searchRef, contentRef, handleQueryClick, handleResetClick } = usePage();

// 抽屉引用
const drawerRef = ref();

// 搜索配置
const searchConfig: ISearchConfig = {
  grid: true, // 启用 Grid 布局
  formItems: [
    {
      type: "input",
      label: $t("pages.menu.name"),
      prop: "name",
      attrs: {
        placeholder: $t("common.placeholder.input"),
        clearable: true,
      },
    },
    {
      type: "select",
      label: $t("common.table.status"),
      prop: "status",
      attrs: {
        placeholder: $t("common.placeholder.select"),
        clearable: true,
        filterable: true,
      },
      options: statusList.value,
    },
  ],
};

// 表格配置
const contentConfig: IContentConfig = {
  permPrefix: "sys:menu", // 菜单管理权限前缀
  toolbarRight: ["add"], // 右侧自定义按钮（在defaultToolbar左侧）
  defaultToolbar: ["refresh", "filter"], // 右侧默认工具栏
  table: {
    border: true,
    stripe: false,
    treeConfig: {
      parentField: "parentId",
      rowField: "id",
      expandAll: true,
    },
  },
  pagination: false, // 禁用分页（树形表格不需要分页）
  indexAction: async (query: any) => {
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
    // 转换数据格式为树形结构
    return {
      items: buildMenuTree(result.items || []),
      total: result.total || 0,
    };
  },
  columns: [
    {
      type: "index",
      label: $t("common.table.seq"),
      width: 60,
    },
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
      template: "tool",
      action: [
        {
          name: "edit",
          text: $t("common.button.edit"),
        },
        {
          name: "delete",
          text: $t("common.button.delete"),
          attrs: {
            type: "danger",
          },
        },
      ],
    },
  ],
};

// 处理操作点击
const handleOperateClick = (data: IOperateData) => {
  const { name, row } = data;

  if (name === "edit") {
    // 编辑
    drawerRef.value?.open(row);
  } else if (name === "delete") {
    // 删除
    ElMessageBox.confirm(
      $t("common.confirm.do_you_want_delete", { moduleName: $t("pages.menu.moduleName") }),
      $t("common.title.confirm"),
      {
        confirmButtonText: $t("common.button.confirm"),
        cancelButtonText: $t("common.button.cancel"),
        type: "warning",
      }
    ).then(async () => {
      try {
        await deleteMenu({ id: row.id });
        ElMessage.success($t("common.notification.delete_success"));
        contentRef.value?.fetchPageData({}, true);
      } catch {
        ElMessage.error($t("common.notification.delete_failed"));
      }
    });
  }
};

// 处理新增点击
const handleAddClick = () => {
  drawerRef.value?.open();
};

// 处理成功回调
const handleSuccess = () => {
  contentRef.value?.fetchPageData({}, true);
};

// 处理工具栏点击
const handleToolbarClick = (name: string) => {
  console.log("Toolbar clicked:", name);
};

// 展开所有节点
const expandAll = () => {
  const tableRef = contentRef.value?.tableRef;
  if (tableRef) {
    tableRef.setAllTreeExpand(true);
  }
};

// 折叠所有节点
const collapseAll = () => {
  const tableRef = contentRef.value?.tableRef;
  if (tableRef) {
    tableRef.clearTreeExpand();
  }
};

// 规范化权限标识
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

// 暴露方法给父组件（用于工具栏按钮）
defineExpose({
  expandAll,
  collapseAll,
});
</script>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
  width: 100%;
  min-width: 0;
  flex-shrink: 0;
}
</style>
