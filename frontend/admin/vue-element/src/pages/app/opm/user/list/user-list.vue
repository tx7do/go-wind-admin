<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage
      ref="pageRef"
      :config="pageConfig"
      @add="handleAdd"
      @edit="handleEdit"
      @operate="handleOperate"
    >
      <!-- 状态 -->
      <template #status="scope: any">
        <ElTag size="small" effect="plain" round :type="userStatusToType(scope.row.status)">
          {{ userStatusToName(scope.row.status) }}
        </ElTag>
      </template>
      <!-- 角色 -->
      <template #roleNames="scope: any">
        <div>
          <ElTag
            v-for="role in scope.row.roleNames"
            :key="role"
            class="mb-1 mr-1"
            :style="{
              backgroundColor: getRandomColor(role),
              color: '#333',
              border: 'none',
            }"
          >
            {{ role }}
          </ElTag>
        </div>
      </template>
      <!-- 组织 -->
      <template #orgUnitNames="scope: any">
        <div>
          <ElTag
            v-for="orgUnit in scope.row.orgUnitNames"
            :key="orgUnit"
            class="mb-1 mr-1"
            :style="{
              backgroundColor: getRandomColor(orgUnit),
              color: '#333',
              border: 'none',
            }"
          >
            {{ orgUnit }}
          </ElTag>
        </div>
      </template>
      <!-- 职位 -->
      <template #positionNames="scope: any">
        <div>
          <ElTag
            v-for="position in scope.row.positionNames"
            :key="position"
            class="mb-1 mr-1"
            :style="{
              backgroundColor: getRandomColor(position),
              color: '#333',
              border: 'none',
            }"
          >
            {{ position }}
          </ElTag>
        </div>
      </template>
    </ProPage>

    <!-- 新增/编辑抽屉 -->
    <UserDrawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { computed, ref, watch } from "vue";
import { ElMessageBox, ElNotification, ElTag } from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import UserDrawer from "./user-drawer.vue";

import {
  disableMfa,
  fetchListPositions,
  fetchListRoles,
  userStatusList,
  userStatusToType,
  userStatusToName,
  useDeleteUser,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/core/i18n";
import { router } from "@/router";
import { getRandomColor } from "@/utils/color";
import { useUserViewStore } from "./user-view.state";
import { useAppUserStore } from "@/stores";

const { mutateAsync: deleteUser } = useDeleteUser();
const userViewStore = useUserViewStore();

const pageRef = ref();
const drawerRef = ref();

// 监听组织和租户变化，重新加载数据
watch(
  () => [userViewStore.currentOrgUnitId, userViewStore.currentTenantId],
  () => {
    pageRef.value?.refresh();
    // 角色/职位下拉选项依赖租户/组织，切换后必须重跑 initFn 刷新，
    // 否则下拉里仍是旧租户/旧组织下的选项，按过期选项搜索会查不到人。
    pageRef.value?.reloadFieldOptions(["roleId", "positionId"]);
  }
);

const pageConfig = computed<ProPageConfig>(() => ({
  skeleton: true,
  search: {
    grid: true,
    fields: [
      {
        type: "input",
        label: $t("pages.user.form.username"),
        field: "username",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "input",
        label: $t("pages.user.form.realname"),
        field: "realname",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "input",
        label: $t("pages.user.form.mobile"),
        field: "mobile",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "select",
        label: $t("pages.user.form.status"),
        field: "status",
        attrs: { placeholder: $t("common.placeholder.select"), clearable: true },
        options: userStatusList.value,
      },
      {
        type: "tree-select",
        label: $t("pages.user.form.role"),
        field: "roleId",
        attrs: {
          placeholder: $t("common.placeholder.select"),
          clearable: true,
          filterable: true,
          nodeKey: "id",
          props: { label: "name", value: "id", children: "children" },
        },
        initFn: async (item: any) => {
          try {
            const result = await fetchListRoles(
              new PaginationQuery({
                formValues: {
                  status: "ON",
                  type__not: "TEMPLATE",
                  tenant_id: userViewStore.currentTenantId ?? 0,
                },
              })
            );
            item.attrs.data = result.items || [];
          } catch (error) {
            console.error("Failed to load roles:", error);
          }
        },
      },
      {
        type: "tree-select",
        label: $t("pages.user.form.position"),
        field: "positionId",
        attrs: {
          placeholder: $t("common.placeholder.select"),
          clearable: true,
          filterable: true,
          nodeKey: "id",
          props: { label: "name", value: "id", children: "children" },
        },
        initFn: async (item: any) => {
          try {
            const result = await fetchListPositions(
              new PaginationQuery({
                formValues: {
                  status: "ON",
                  org_unit_id: userViewStore.currentOrgUnitId,
                  tenant_id: userViewStore.currentTenantId ?? 0,
                },
              })
            );
            item.attrs.data = result.items || [];
          } catch (error) {
            console.error("Failed to load positions:", error);
          }
        },
      },
    ],
  },

  table: {
    listAction: async (query: any) => {
      const { page, pageSize, ...queryParams } = query;
      const result = await userViewStore.fetchUserList(page || 1, pageSize || 10, queryParams);
      return { items: result.items || [], total: result.total || 0 };
    },
    deleteAction: async (ids: string) => {
      await deleteUser(ids as any);
    },
    toolbar: [],
    toolbarRight: ["add"],
    defaultToolbar: ["refresh", "filter"],
    tableAttrs: { border: true, stripe: true, height: "auto" },
    columns: [
      { type: "index", label: $t("common.table.seq"), width: 60 },
      { prop: "username", label: $t("pages.user.table.username"), width: 120 },
      { prop: "realname", label: $t("pages.user.table.realname"), width: 100 },
      { prop: "nickname", label: $t("pages.user.table.nickname"), width: 100 },
      { prop: "email", label: $t("pages.user.table.email"), width: 160 },
      { prop: "mobile", label: $t("pages.user.table.mobile"), width: 130 },
      {
        prop: "orgUnitNames",
        label: $t("pages.user.table.orgUnitId"),
        width: 130,
        slotName: "orgUnitNames",
      },
      {
        prop: "positionNames",
        label: $t("pages.user.table.positionId"),
        width: 130,
        slotName: "positionNames",
      },
      {
        prop: "roleNames",
        label: $t("pages.user.table.roleId"),
        width: 100,
        slotName: "roleNames",
      },
      {
        prop: "status",
        label: $t("pages.user.table.status"),
        width: 95,
        slotName: "status",
      },
      {
        prop: "lastLoginAt",
        label: $t("pages.user.table.lastLoginAt"),
        width: 160,
        cellType: "date",
        dateFormat: "YYYY-MM-DD HH:mm:ss",
      },
      {
        prop: "createdAt",
        label: $t("common.table.createdAt"),
        width: 160,
        cellType: "date",
        dateFormat: "YYYY-MM-DD HH:mm:ss",
      },
      { prop: "remark", label: $t("common.table.remark"), width: 250 },
      {
        prop: "action",
        label: $t("common.table.action"),
        fixed: "right",
        width: 150,
        cellType: "tool",
        buttons: [
          { name: "detail", label: $t("common.button.detail"), icon: "lucide:eye" },
          { name: "edit", label: $t("common.button.edit"), icon: "lucide:pen-line" },
          ...(isPlatformSide.value
            ? [{ name: "resetMfa", label: $t("pages.user.resetMfa"), icon: "lucide:shield-off" }]
            : []),
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

// 仅平台侧操作者（tenantId=0）显示救援重置；非平台用户由后端 403 兜底
const isPlatformSide = computed(() => Number(useAppUserStore().userInfo?.tenantId ?? 0) === 0);

function handleAdd() {
  drawerRef.value?.open({ create: true });
}

function handleEdit(row: any) {
  drawerRef.value?.open({ create: false, row });
}

function handleOperate(data: { name: string; row: any }) {
  if (data.name === "detail") {
    router.push(`/opm/users/detail/${data.row.id}`);
  } else if (data.name === "resetMfa") {
    handleResetMfa(data.row);
  }
}

// 管理端救援重置：清空目标用户 TOTP 因子（认证器丢失解锁用，仅平台管理员生效）
async function handleResetMfa(row: any) {
  try {
    await ElMessageBox.confirm(
      $t("pages.user.resetMfaConfirmDesc"),
      $t("pages.user.resetMfaConfirmTitle"),
      {
        confirmButtonText: $t("pages.user.resetMfa"),
        cancelButtonText: $t("core.login.mfaCancel"),
        type: "warning",
      }
    );
  } catch {
    return;
  }
  try {
    await disableMfa({ userId: row.id, method: "TOTP" } as any);
    ElNotification.success($t("pages.user.resetMfaSuccess"));
  } catch (err: any) {
    ElNotification.error(err?.message || $t("pages.user.resetMfaFailed"));
  }
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
