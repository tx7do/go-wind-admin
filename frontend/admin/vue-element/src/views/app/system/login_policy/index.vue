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
      <!-- 策略类型 -->
      <template #type="{ row }">
        <ElTag size="small" effect="dark" round :color="loginPolicyTypeToColor(row.type)">
          {{ loginPolicyTypeToName(row.type) }}
        </ElTag>
      </template>

      <!-- 策略方法 -->
      <template #method="{ row }">
        <ElTag size="small" effect="dark" round :color="loginPolicyMethodToColor(row.method)">
          {{ loginPolicyMethodToName(row.method) }}
        </ElTag>
      </template>
    </PageContent>

    <!-- 新增/编辑抽屉 -->
    <LoginPolicyDrawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ElMessage, ElMessageBox, ElTag } from "element-plus";

import PageContent from "@/components/CURD/PageContent.vue";
import PageSearch from "@/components/CURD/PageSearch.vue";
import usePage from "@/components/CURD/usePage";
import type { IOperateData, ISearchConfig, IContentConfig } from "@/components/CURD/types";
import LoginPolicyDrawer from "./login-policy-drawer.vue";

import {
  loginPolicyMethodList,
  loginPolicyMethodToColor,
  loginPolicyMethodToName,
  loginPolicyTypeList,
  loginPolicyTypeToColor,
  loginPolicyTypeToName,
  fetchListLoginPolicies,
  useDeleteLoginPolicy,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/i18n";

const { mutateAsync: deleteLoginPolicy } = useDeleteLoginPolicy();

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
      label: $t("pages.login_policy.type"),
      prop: "type",
      attrs: {
        placeholder: $t("common.placeholder.select"),
        clearable: true,
        filterable: true,
      },
      options: loginPolicyTypeList.value,
    },
    {
      type: "select",
      label: $t("pages.login_policy.method"),
      prop: "method",
      attrs: {
        placeholder: $t("common.placeholder.select"),
        clearable: true,
        filterable: true,
      },
      options: loginPolicyMethodList.value,
    },
  ],
};

// 表格配置
const contentConfig: IContentConfig = {
  permPrefix: "sys:login_policy", // 登录策略管理权限前缀
  toolbarRight: ["add"], // 右侧自定义按钮（在defaultToolbar左侧）
  defaultToolbar: ["refresh", "exports", "filter"], // 右侧默认工具栏
  table: {
    border: true,
    stripe: false,
  },
  indexAction: async (query: any) => {
    const { page, pageSize, ...queryParams } = query;
    const result = await fetchListLoginPolicies(
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
    { prop: "targetId", label: $t("pages.login_policy.targetId"), minWidth: 120 },
    {
      prop: "type",
      label: $t("pages.login_policy.type"),
      width: 100,
      slotName: "type",
    },
    {
      prop: "method",
      label: $t("pages.login_policy.method"),
      width: 100,
      slotName: "method",
    },
    { prop: "value", label: $t("pages.login_policy.value"), minWidth: 150 },
    { prop: "reason", label: $t("pages.login_policy.reason"), minWidth: 150 },
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
      $t("common.confirm.do_you_want_delete", { moduleName: $t("pages.login_policy.moduleName") }),
      $t("common.title.confirm"),
      {
        confirmButtonText: $t("common.button.confirm"),
        cancelButtonText: $t("common.button.cancel"),
        type: "warning",
      }
    ).then(async () => {
      try {
        await deleteLoginPolicy({ id: row.id });
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
