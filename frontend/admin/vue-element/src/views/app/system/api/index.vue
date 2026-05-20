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
    />

    <!-- 新增/编辑抽屉 -->
    <ApiDrawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ElMessage, ElMessageBox } from "element-plus";

import PageContent from "@/components/CURD/PageContent.vue";
import PageSearch from "@/components/CURD/PageSearch.vue";
import usePage from "@/components/CURD/usePage";
import type { IOperateData, ISearchConfig, IContentConfig } from "@/components/CURD/types";
import ApiDrawer from "./api-drawer.vue";

import { methodList, useApiStore } from "@/stores";
import { $t } from "@/i18n";

const apiStore = useApiStore();

// 使用 CURD hook
const { searchRef, contentRef, handleQueryClick, handleResetClick } = usePage();

// 抽屉引用
const drawerRef = ref();

// 搜索配置
const searchConfig: ISearchConfig = {
  grid: true, // 启用 Grid 布局
  formItems: [
    {
      type: "select",
      label: $t("pages.api.method"),
      prop: "method",
      attrs: {
        placeholder: $t("common.placeholder.select"),
        clearable: true,
        filterable: true,
      },
      options: methodList,
    },
    {
      type: "input",
      label: $t("pages.api.module"),
      prop: "module",
      attrs: {
        placeholder: $t("common.placeholder.input"),
        clearable: true,
      },
    },
    {
      type: "input",
      label: $t("pages.api.path"),
      prop: "path",
      attrs: {
        placeholder: $t("common.placeholder.input"),
        clearable: true,
      },
    },
  ],
};

// 表格配置
const contentConfig: IContentConfig = {
  permPrefix: "sys:api", // API 管理权限前缀
  toolbarRight: [
    {
      name: "sync",
      text: $t("pages.api.button.sync"),
      attrs: {
        type: "danger",
        icon: "refresh",
      },
    },
    "add", // 添加按钮
  ], // 右侧自定义按钮（在defaultToolbar左侧）
  defaultToolbar: ["refresh", "exports", "filter"], // 右侧默认工具栏
  table: {
    border: true,
    stripe: false,
  },
  indexAction: async (query: any) => {
    const { page, pageSize, ...queryParams } = query;
    const result = await apiStore.listApi(
      {
        page: page || 1,
        pageSize: pageSize || 10,
      },
      queryParams,
      null,
      ["path"] // 按 path 字段排序
    );
    // 转换数据格式
    return {
      items: result.items || [],
      total: result.total || 0,
    };
  },
  columns: [
    { type: "index", label: $t("common.table.seq"), width: 60 },
    { prop: "description", label: $t("common.table.description"), minWidth: 150 },
    { prop: "path", label: $t("pages.api.path"), minWidth: 200 },
    { prop: "method", label: $t("pages.api.method"), width: 100 },
    { prop: "module", label: $t("pages.api.module"), minWidth: 120 },
    { prop: "moduleDescription", label: $t("pages.api.moduleDescription"), minWidth: 150 },
    {
      prop: "createdAt",
      label: $t("common.table.createdAt"),
      minWidth: 160,
      template: "date",
      dateFormat: "YYYY-MM-DD HH:mm:ss",
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

// 处理操作点击
const handleOperateClick = (data: IOperateData) => {
  const { name, row } = data;

  if (name === "edit") {
    // 编辑
    drawerRef.value?.open(row);
  } else if (name === "delete") {
    // 删除
    ElMessageBox.confirm(
      $t("common.confirm.do_you_want_delete", { moduleName: $t("pages.api.moduleName") }),
      $t("common.title.confirm"),
      {
        confirmButtonText: $t("common.button.confirm"),
        cancelButtonText: $t("common.button.cancel"),
        type: "warning",
      }
    ).then(async () => {
      try {
        await apiStore.deleteApi(row.id);
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
  if (name === "sync") {
    // 同步权限
    ElMessageBox.confirm(
      $t("common.confirm.do_you_want_sync", { moduleName: $t("pages.api.moduleName") }),
      $t("common.title.confirm"),
      {
        confirmButtonText: $t("common.button.confirm"),
        cancelButtonText: $t("common.button.cancel"),
        type: "warning",
      }
    ).then(async () => {
      try {
        await apiStore.syncApis();
        ElMessage.success($t("pages.api.notification.sync_success"));
        contentRef.value?.fetchPageData({}, true);
      } catch {
        ElMessage.error($t("pages.api.notification.sync_failed"));
      }
    });
  }
};
</script>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
  width: 100%;
  min-width: 0;
  flex-shrink: 0;
}
</style>
