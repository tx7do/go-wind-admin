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
        <ElTag size="small" effect="dark" round :color="positionTypeToColor(row.type)">
          {{ positionTypeToName(row.type) }}
        </ElTag>
      </template>

      <!-- 状态 -->
      <template #status="{ row }">
        <ElTag size="small" effect="dark" round :color="statusToColor(row.status)">
          {{ statusToName(row.status) }}
        </ElTag>
      </template>
    </PageContent>

    <!-- 新增/编辑抽屉 -->
    <PositionDrawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ElTag, ElMessage, ElMessageBox } from "element-plus";

import PageContent from "@/components/CURD/PageContent.vue";
import PageSearch from "@/components/CURD/PageSearch.vue";
import usePage from "@/components/CURD/usePage";
import type { IOperateData, ISearchConfig, IContentConfig } from "@/components/CURD/types";
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
      label: $t("pages.position.name"),
      prop: "name",
      attrs: {
        placeholder: $t("common.placeholder.input"),
        clearable: true,
      },
    },
    {
      type: "input",
      label: $t("pages.position.code"),
      prop: "code",
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
      options: statusList.value,
    },
    {
      type: "select",
      label: $t("pages.position.type"),
      prop: "type",
      attrs: {
        placeholder: $t("common.placeholder.select"),
        clearable: true,
      },
      options: positionTypeList.value,
    },
    {
      type: "tree-select",
      label: $t("pages.position.orgUnit"),
      prop: "orgUnitId",
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
};

// 表格配置
const contentConfig: IContentConfig = {
  permPrefix: "sys:platform_admin", // 职位管理权限前缀
  toolbarRight: ["add"], // 右侧自定义按钮（在defaultToolbar左侧）
  defaultToolbar: ["refresh", "filter"], // 右侧默认工具栏
  table: {
    border: true,
    stripe: true,
  },
  indexAction: async (query: any) => {
    const { page, pageSize, ...queryParams } = query;
    const result = await fetchListPositions(
      new PaginationQuery({
        paging: { page: page || 1, pageSize: pageSize || 10 },
        formValues: queryParams,
      })
    );
    // 转换数据格式：将 items 转换为 list
    return {
      items: result.items || [],
      total: result.total || 0,
    };
  },
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
        $t("common.message.confirmDelete", { moduleName: $t("pages.position.moduleName") }),
        $t("common.title.warning"),
        {
          confirmButtonText: $t("common.button.confirm"),
          cancelButtonText: $t("common.button.cancel"),
          type: "warning",
        }
      );

      await deletePosition({ id: row.id });
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
