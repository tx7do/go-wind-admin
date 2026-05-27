<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig" @add="handleAdd" @edit="handleEdit">
      <!-- 策略类型 -->
      <template #type="scope: any">
        <ElTag size="small" effect="dark" round :color="loginPolicyTypeToColor(scope.row.type)">
          {{ loginPolicyTypeToName(scope.row.type) }}
        </ElTag>
      </template>

      <!-- 策略方法 -->
      <template #method="scope: any">
        <ElTag size="small" effect="dark" round :color="loginPolicyMethodToColor(scope.row.method)">
          {{ loginPolicyMethodToName(scope.row.method) }}
        </ElTag>
      </template>
    </ProPage>

    <!-- 新增/编辑抽屉 -->
    <LoginPolicyDrawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import { ElTag } from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
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

const pageRef = ref();
const drawerRef = ref();

const pageConfig = computed<ProPageConfig>(() => ({

  search: {
    grid: true,
    fields: [
      {
        type: "select",
        label: $t("pages.login_policy.type"),
        field: "type",
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
        field: "method",
        attrs: {
          placeholder: $t("common.placeholder.select"),
          clearable: true,
          filterable: true,
        },
        options: loginPolicyMethodList.value,
      },
    ],
  },

  table: {
    listAction: async (query: any) => {
      const { page, pageSize, ...queryParams } = query;
      const result = await fetchListLoginPolicies(
        new PaginationQuery({
          paging: { page: page || 1, pageSize: pageSize || 10 },
          formValues: queryParams,
        })
      );
      return { items: result.items || [], total: result.total || 0 };
    },
    deleteAction: async (ids: string) => {
      await deleteLoginPolicy({ id: ids as any });
    },
    toolbar: [],
    toolbarRight: ["add"],
    defaultToolbar: ["refresh", "exports", "filter"],
    tableAttrs: { border: true, stripe: false },
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
        cellType: "date",
        dateFormat: "YYYY-MM-DD HH:mm:ss",
      },
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
</script>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
  width: 100%;
  min-width: 0;
  flex-shrink: 0;
}
</style>
