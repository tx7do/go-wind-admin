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
      @row-click="handleRowClick"
    >
      <!-- 状态 -->
      <template #status="{ row }">
        <ElTag size="small" effect="dark" round :color="statusToColor(row.status)">
          {{ statusToName(row.status) }}
        </ElTag>
      </template>
    </PageContent>

    <!-- 新增/编辑抽屉 -->
    <PermissionGroupDrawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ElTag, ElMessage, ElMessageBox } from "element-plus";

import PageContent from "@/components/CURD/PageContent.vue";
import PageSearch from "@/components/CURD/PageSearch.vue";
import usePage from "@/components/CURD/usePage";
import type { IOperateData, ISearchConfig, IContentConfig } from "@/components/CURD/types";
import PermissionGroupDrawer from "./permission-group-drawer.vue";

import {
  statusList,
  statusToColor,
  statusToName,
  useDeletePermissionGroup,
} from "@/api/composables";
import { $t } from "@/i18n";
import { usePermissionViewStore } from "@/views/app/permission/permission/permission-view.state";

const { mutateAsync: deletePermissionGroup } = useDeletePermissionGroup();
const permissionViewStore = usePermissionViewStore();

// 使用 CURD hook
const { searchRef, contentRef, handleQueryClick, handleResetClick } = usePage();

// 抽屉引用
const drawerRef = ref();

// 行点击联动 - 切换分组时刷新权限列表
function handleRowClick(row: any) {
  console.log("分组行点击:", row);
  if (row?.id) {
    console.log("设置 currentGroupId:", row.id);
    permissionViewStore.setCurrentGroupId(row.id);
    console.log("needReloadPermissionList:", permissionViewStore.needReloadPermissionList);
  }
}

// 搜索配置
const searchConfig: ISearchConfig = {
  // 移除 grid: true，改用默认 Flex 布局实现基于容器宽度的自动换行
  // Grid 布局依赖视口断点，在左侧窄面板中不会触发换行
  formItems: [
    {
      type: "input",
      label: $t("pages.permission_group.name"),
      prop: "name",
      attrs: {
        placeholder: $t("common.placeholder.input"),
        clearable: true,
        style: { minWidth: "200px" }, // 设置最小宽度，配合 Flex 换行
      },
    },
    {
      type: "input",
      label: $t("pages.permission_group.module"),
      prop: "module",
      attrs: {
        placeholder: $t("common.placeholder.input"),
        clearable: true,
        style: { minWidth: "200px" }, // 设置最小宽度，配合 Flex 换行
      },
    },
    {
      type: "select",
      label: $t("common.table.status"),
      prop: "status",
      attrs: {
        placeholder: $t("common.placeholder.select"),
        clearable: true,
        style: { minWidth: "200px" }, // 设置最小宽度，配合 Flex 换行
      },
      options: statusList.value,
    },
  ],
};

// 表格配置
const contentConfig: IContentConfig = {
  permPrefix: "sys:permission_group",
  toolbarRight: ["add"],
  defaultToolbar: ["refresh", "filter"],
  table: {
    border: true,
    stripe: false,
    height: "auto",
    treeConfig: {
      parentField: "parentId",
      rowField: "id",
      expandAll: true,
    },
  },
  indexAction: async (query: any) => {
    const { page, pageSize, ...queryParams } = query;
    const result = await permissionViewStore.fetchGroupList(page || 1, pageSize || 10, queryParams);
    return {
      items: result.items || [],
      total: result.total || 0,
    };
  },
  columns: [
    {
      prop: "name",
      label: $t("pages.permission_group.name"),
      minWidth: 150,
      treeNode: true,
      align: "left",
    },
    {
      prop: "module",
      label: $t("pages.permission_group.module"),
      minWidth: 120,
      align: "left",
    },
    {
      prop: "status",
      label: $t("common.table.status"),
      width: 95,
      slotName: "status",
    },
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

// 新增按钮点击
function handleAddClick() {
  drawerRef.value?.open({ create: true });
}

// 操作按钮点击
async function handleOperateClick(data: IOperateData) {
  const { name, row } = data;

  if (name === "edit") {
    drawerRef.value?.open({ create: false, row });
  } else if (name === "delete") {
    try {
      await ElMessageBox.confirm(
        $t("common.message.confirmDelete", { moduleName: $t("pages.permission_group.moduleName") }),
        $t("common.title.warning"),
        {
          confirmButtonText: $t("common.button.confirm"),
          cancelButtonText: $t("common.button.cancel"),
          type: "warning",
        }
      );

      await deletePermissionGroup({ id: row.id });
      ElMessage.success($t("common.notification.deleteSuccess"));
      handleSuccess();
    } catch (error) {
      if (error !== "cancel") {
        ElMessage.error($t("common.notification.deleteFailed"));
      }
    }
  }
}

// 工具栏按钮点击
function handleToolbarClick(name: string) {
  console.log("toolbar click:", name);
}

// 成功回调
function handleSuccess() {
  contentRef.value?.fetchPageData();
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
