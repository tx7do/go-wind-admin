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
    >
      <!-- 是否启用 -->
      <template #isEnabled="{ row }">
        <ElTag size="small" effect="dark" round :color="enableBoolToColor(row.isEnabled)">
          {{ enableBoolToName(row.isEnabled) }}
        </ElTag>
      </template>

      <!-- 是否默认 -->
      <template #isDefault="{ row }">
        <ElTag size="small" effect="dark" round :color="enableBoolToColor(row.isDefault)">
          {{ enableBoolToName(row.isDefault) }}
        </ElTag>
      </template>
    </PageContent>

    <!-- 新增/编辑抽屉 -->
    <LanguageDrawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ElMessage, ElMessageBox, ElTag } from "element-plus";

import PageContent from "@/components/CURD/PageContent.vue";
import PageSearch from "@/components/CURD/PageSearch.vue";
import usePage from "@/components/CURD/usePage";
import type { IOperateData, ISearchConfig, IContentConfig } from "@/components/CURD/types";
import LanguageDrawer from "./language-drawer.vue";

import {
  enableBoolToColor,
  enableBoolToName,
  fetchListLanguages,
  useDeleteLanguage,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/i18n";

const { mutateAsync: deleteLanguage } = useDeleteLanguage();

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
      label: $t("pages.language.languageName"),
      prop: "languageName",
      attrs: {
        placeholder: $t("common.placeholder.input"),
        clearable: true,
      },
    },
    {
      type: "input",
      label: $t("pages.language.languageCode"),
      prop: "languageCode",
      attrs: {
        placeholder: $t("common.placeholder.input"),
        clearable: true,
      },
    },
  ],
};

// 表格配置
const contentConfig: IContentConfig = {
  permPrefix: "sys:language", // 语言管理权限前缀
  toolbarRight: ["add"], // 右侧自定义按钮（在defaultToolbar左侧）
  defaultToolbar: ["refresh", "exports", "filter"], // 右侧默认工具栏
  table: {
    border: true,
    stripe: false,
  },
  indexAction: async (query: any) => {
    const { page, pageSize, ...queryParams } = query;
    const result = await fetchListLanguages(
      new PaginationQuery({
        paging: { page: page || 1, pageSize: pageSize || 10 },
        formValues: queryParams,
      })
    );
    // 转换数据格式
    return {
      items: result.items || [],
      total: result.total || 0,
    };
  },
  columns: [
    { type: "index", label: $t("common.table.seq"), width: 60 },
    { prop: "nativeName", label: $t("pages.language.nativeName"), minWidth: 120, fixed: "left" },
    { prop: "languageName", label: $t("pages.language.languageName"), minWidth: 120 },
    { prop: "languageCode", label: $t("pages.language.languageCode"), minWidth: 120 },
    {
      prop: "isEnabled",
      label: $t("pages.language.isEnabled"),
      width: 100,
      slotName: "isEnabled",
    },
    {
      prop: "isDefault",
      label: $t("pages.language.isDefault"),
      width: 100,
      slotName: "isDefault",
    },
    { prop: "sortOrder", label: $t("common.table.sortOrder"), width: 100 },
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
      $t("common.confirm.do_you_want_delete", { moduleName: $t("pages.language.moduleName") }),
      $t("common.title.confirm"),
      {
        confirmButtonText: $t("common.button.confirm"),
        cancelButtonText: $t("common.button.cancel"),
        type: "warning",
      }
    ).then(async () => {
      try {
        await deleteLanguage({ id: row.id });
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
</script>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
  width: 100%;
  min-width: 0;
  flex-shrink: 0;
}
</style>
