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
      <!-- 类型 -->
      <template #type="{ row }">
        <ElTag size="small" effect="dark" round :color="orgUnitTypeToColor(row.type)">
          {{ orgUnitTypeToName(row.type) }}
        </ElTag>
      </template>

      <!-- 状态 -->
      <template #status="{ row }">
        <ElTag size="small" effect="dark" round :color="orgUnitStatusToColor(row.status)">
          {{ orgUnitStatusToName(row.status) }}
        </ElTag>
      </template>
    </PageContent>

    <!-- 新增/编辑抽屉 -->
    <OrgDrawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ElTag, ElMessage, ElMessageBox } from "element-plus";

import PageContent from "@/components/CURD/PageContent.vue";
import PageSearch from "@/components/CURD/PageSearch.vue";
import usePage from "@/components/CURD/usePage";
import type { IOperateData, ISearchConfig, IContentConfig } from "@/components/CURD/types";
import OrgDrawer from "./org-drawer.vue";

import {
  orgUnitStatusToColor,
  orgUnitStatusToName,
  orgUnitTypeListForQuery,
  orgUnitTypeToColor,
  orgUnitTypeToName,
  fetchListOrgUnits,
  useDeleteOrgUnit,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/i18n";

const { mutateAsync: deleteOrgUnit } = useDeleteOrgUnit();

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
      label: $t("pages.org_unit.name"),
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
      },
      options: [
        { value: "ON", label: $t("enum.status.ON") },
        { value: "OFF", label: $t("enum.status.OFF") },
      ],
    },
    {
      type: "select",
      label: $t("pages.org_unit.type"),
      prop: "type",
      attrs: {
        placeholder: $t("common.placeholder.select"),
        clearable: true,
      },
      options: orgUnitTypeListForQuery.value,
    },
  ],
};

// 表格配置
const contentConfig: IContentConfig = {
  permPrefix: "sys:platform_admin", // 组织单位管理权限前缀
  toolbarRight: ["add"], // 右侧自定义按钮（在defaultToolbar左侧）
  defaultToolbar: ["refresh", "filter"], // 右侧默认工具栏
  table: {
    border: true,
    stripe: false,
    // 树形表格配置
    treeConfig: {
      expandAll: true,
      rowField: "id",
      childrenField: "children",
    },
  },
  indexAction: async (query: any) => {
    const { page, pageSize, ...queryParams } = query;
    const result = await fetchListOrgUnits(
      new PaginationQuery({
        paging: { page: page || 1, pageSize: pageSize || 100 },
        formValues: queryParams,
      })
    );
    console.log("org unit data:", result);
    // 转换数据格式：将 items 转换为 list
    return {
      items: result.items || [],
      total: result.total || 0,
    };
  },
  columns: [
    { type: "index", label: $t("common.table.seq"), width: 60 },
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
    { prop: "sortOrder", label: $t("common.table.sortOrder"), width: 80 },
    {
      prop: "createdAt",
      label: $t("common.table.createdAt"),
      minWidth: 160,
      template: "date",
      dateFormat: "YYYY-MM-DD HH:mm:ss",
    },
    { prop: "remark", label: $t("common.table.remark"), minWidth: 150 },
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
        $t("common.message.confirmDelete", { moduleName: $t("pages.org_unit.moduleName") }),
        $t("common.title.warning"),
        {
          confirmButtonText: $t("common.button.confirm"),
          cancelButtonText: $t("common.button.cancel"),
          type: "warning",
        }
      );

      await deleteOrgUnit({ id: row.id });
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
