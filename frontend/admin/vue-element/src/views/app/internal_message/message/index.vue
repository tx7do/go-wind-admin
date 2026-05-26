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
      @operate-click="handleOperateClick"
    >
      <!-- 状态 -->
      <template #status="{ row }">
        <ElTag size="small" effect="dark" round :color="internalMessageStatusColor(row.status)">
          {{ internalMessageStatusLabel(row.status) }}
        </ElTag>
      </template>

      <!-- 类型 -->
      <template #type="{ row }">
        <ElTag size="small" effect="dark" round :color="internalMessageTypeColor(row.type)">
          {{ internalMessageTypeLabel(row.type) }}
        </ElTag>
      </template>
    </PageContent>

    <!-- 抽屉 -->
    <InternalMessageDrawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ElMessageBox, ElMessage, ElTag } from "element-plus";

import PageContent from "@/components/CURD/PageContent.vue";
import PageSearch from "@/components/CURD/PageSearch.vue";
import usePage from "@/components/CURD/usePage";
import type { ISearchConfig, IContentConfig } from "@/components/CURD/types";

import {
  internalMessageStatusColor,
  internalMessageStatusLabel,
  internalMessageStatusList,
  internalMessageTypeColor,
  internalMessageTypeLabel,
  internalMessageTypeList,
  fetchListInternalMessages,
  fetchListMessageCategories,
  useDeleteInternalMessage,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/i18n";

import InternalMessageDrawer from "./internal-message-drawer.vue";

const { mutateAsync: deleteMessage } = useDeleteInternalMessage();

// 使用 CURD hook
const { searchRef, contentRef, handleQueryClick, handleResetClick } = usePage();

// 抽屉引用
const drawerRef = ref<InstanceType<typeof InternalMessageDrawer>>();

// 搜索配置
const searchConfig: ISearchConfig = {
  grid: true, // 启用 Grid 布局
  formItems: [
    {
      type: "input",
      label: $t("pages.internal_message.title"),
      prop: "title",
      attrs: {
        placeholder: $t("common.placeholder.input"),
        clearable: true,
      },
    },
    {
      type: "select",
      label: $t("pages.internal_message.status"),
      prop: "status",
      attrs: {
        placeholder: $t("common.placeholder.select"),
        clearable: true,
        filterable: true,
      },
      options: internalMessageStatusList.value,
    },
    {
      type: "select",
      label: $t("pages.internal_message.type"),
      prop: "type",
      attrs: {
        placeholder: $t("common.placeholder.select"),
        clearable: true,
        filterable: true,
      },
      options: internalMessageTypeList.value,
    },
    {
      type: "api-tree-select",
      label: $t("pages.internal_message.categoryId"),
      prop: "category_id",
      attrs: {
        placeholder: $t("common.placeholder.select"),
        clearable: true,
        filterable: true,
        nodeKey: "id",
        props: {
          label: "name",
          children: "children",
        },
        defaultExpandAll: true,
      },
      api: async () => {
        const result = await fetchListMessageCategories(
          new PaginationQuery({ formValues: { is_enabled: "true" } })
        );
        return result.items || [];
      },
    },
  ],
};

// 表格配置
const contentConfig: IContentConfig = {
  permPrefix: "sys:internal_message", // 内部消息权限前缀
  toolbarRight: ["add"], // 右侧自定义按钮
  defaultToolbar: ["refresh", "exports", "filter"], // 右侧默认工具栏
  table: {
    border: true,
    stripe: false,
  },
  pagination: false, // 禁用分页
  indexAction: async (query: any) => {
    const { page, pageSize, ...queryParams } = query;
    const result = await fetchListInternalMessages(
      new PaginationQuery({
        paging: { page: page || 1, pageSize: pageSize || 10 },
        formValues: queryParams,
      })
    );
    return {
      items: result.items || [],
      total: result.total || 0,
    };
  },
  columns: [
    { prop: "title", label: $t("pages.internal_message.title"), minWidth: 200 },
    { prop: "categoryName", label: $t("pages.internal_message.categoryName"), minWidth: 150 },
    {
      prop: "status",
      label: $t("pages.internal_message.status"),
      width: 120,
      slotName: "status",
    },
    {
      prop: "type",
      label: $t("pages.internal_message.type"),
      width: 120,
      slotName: "type",
    },
    { prop: "senderName", label: $t("pages.internal_message.senderName"), minWidth: 120 },
    {
      prop: "createdAt",
      label: $t("common.table.createdAt"),
      width: 160,
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
        { name: "edit", text: $t("common.button.edit") },
        { name: "delete", text: $t("common.button.delete"), attrs: { type: "danger" } },
      ],
    },
  ],
};

// 处理操作列点击
const handleOperateClick = async (data: any) => {
  const { name, row } = data;

  if (name === "edit") {
    drawerRef.value?.open(row);
  } else if (name === "delete") {
    try {
      await ElMessageBox.confirm(
        $t("common.confirm.do_you_want_delete", {
          moduleName: $t("pages.internal_message.moduleName"),
        }),
        $t("common.title.confirm"),
        {
          confirmButtonText: $t("common.button.confirm"),
          cancelButtonText: $t("common.button.cancel"),
          type: "warning",
        }
      );
      await deleteMessage({ id: row.id });
      ElMessage.success($t("common.notification.delete_success"));
      contentRef.value?.fetchPageData({}, true);
    } catch {
      // 用户取消
    }
  }
};

// 处理成功回调
const handleSuccess = () => {
  contentRef.value?.fetchPageData({}, true);
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
